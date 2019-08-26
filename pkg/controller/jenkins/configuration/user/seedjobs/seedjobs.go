package seedjobs

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/constants"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/groovy"
	"reflect"

	"github.com/go-logr/logr"
	stackerr "github.com/pkg/errors"
	apps "k8s.io/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// UsernameSecretKey is username data key in Kubernetes secret used to create Jenkins username/password credential
	UsernameSecretKey = "username"
	// PasswordSecretKey is password data key in Kubernetes secret used to create Jenkins username/password credential
	PasswordSecretKey = "password"
	// PrivateKeySecretKey is private key data key in Kubernetes secret used to create Jenkins SSH credential
	PrivateKeySecretKey = "privateKey"

	// JenkinsCredentialTypeLabelName is label for kubernetes-credentials-provider-plugin which determine Jenkins
	// credential type
	JenkinsCredentialTypeLabelName = "jenkins.io/credentials-type"

	// AgentName is the name of seed job agent
	AgentName = "seed-job-agent"
)

// SeedJobs defines API for configuring and ensuring Jenkins Seed Jobs and Deploy Keys
type SeedJobs struct {
	jenkinsClient jenkinsclient.Jenkins
	k8sClient     k8s.Client
	logger        logr.Logger
}

// New creates SeedJobs object
func New(jenkinsClient jenkinsclient.Jenkins, k8sClient k8s.Client, logger logr.Logger) *SeedJobs {
	return &SeedJobs{
		jenkinsClient: jenkinsClient,
		k8sClient:     k8sClient,
		logger:        logger,
	}
}

// EnsureSeedJobs configures seed job and runs it for every entry from Jenkins.Spec.SeedJobs
func (s *SeedJobs) EnsureSeedJobs(jenkins *v1alpha2.Jenkins) (done bool, err error) {
	if s.isRecreatePodNeeded(*jenkins) {
		s.logger.Info("Some seed job has been deleted, recreating pod")
		return false, s.restartJenkinsMasterPod(*jenkins)
	}

	if len(jenkins.Spec.SeedJobs) > 0 {
		err := s.createAgent(s.jenkinsClient, s.k8sClient, jenkins, jenkins.Namespace, AgentName)
		if err != nil {
			return false, err
		}
	} else if len(jenkins.Spec.SeedJobs) == 0 {
		err := s.k8sClient.Delete(context.TODO(), &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: jenkins.Namespace,
				Name:      fmt.Sprintf("%s-deployment", AgentName),
			},
		})

		if err != nil {
			return false, err
		}
	}

	if err = s.ensureLabelsForSecrets(*jenkins); err != nil {
		return false, err
	}

	requeue, err := s.createJobs(jenkins)
	if err != nil {
		return false, err
	}
	if requeue {
		return false, nil
	}

	seedJobIDs := s.getAllSeedJobIDs(*jenkins)
	if done && !reflect.DeepEqual(seedJobIDs, jenkins.Status.CreatedSeedJobs) {
		jenkins.Status.CreatedSeedJobs = seedJobIDs
		return false, stackerr.WithStack(s.k8sClient.Update(context.TODO(), jenkins))
	}

	return true, nil
}

// createJob is responsible for creating jenkins job which configures jenkins seed jobs and deploy keys
func (s *SeedJobs) createJobs(jenkins *v1alpha2.Jenkins) (bool, error) {
	groovyClient := groovy.New(s.jenkinsClient, s.k8sClient, s.logger, jenkins, "user-groovy", jenkins.Spec.GroovyScripts.Customization)
	for _, seedJob := range jenkins.Spec.SeedJobs {
		credentialValue, err := s.credentialValue(jenkins.Namespace, seedJob)
		if err != nil {
			return true, err
		}

		groovyScript := seedJobCreatingGroovyScript(seedJob)

		hash := sha256.New()
		hash.Write([]byte(groovyScript))
		hash.Write([]byte(credentialValue))
		requeue, err := groovyClient.EnsureSingle(seedJob.ID, fmt.Sprintf("%s.groovy", seedJob.ID), base64.URLEncoding.EncodeToString(hash.Sum(nil)), groovyScript)

		if err != nil {
			return true, err
		}

		if requeue {
			return true, nil
		}
	}

	return false, nil
}

// ensureLabelsForSecrets adds labels to Kubernetes secrets where are Jenkins credentials used for seed jobs,
// thanks to them kubernetes-credentials-provider-plugin will create Jenkins credentials in Jenkins and
// Operator will able to watch any changes made to them
func (s *SeedJobs) ensureLabelsForSecrets(jenkins v1alpha2.Jenkins) error {
	for _, seedJob := range jenkins.Spec.SeedJobs {
		if seedJob.JenkinsCredentialType == v1alpha2.BasicSSHCredentialType || seedJob.JenkinsCredentialType == v1alpha2.UsernamePasswordCredentialType {
			requiredLabels := resources.BuildLabelsForWatchedResources(jenkins)
			requiredLabels[JenkinsCredentialTypeLabelName] = string(seedJob.JenkinsCredentialType)

			secret := &corev1.Secret{}
			namespaceName := types.NamespacedName{Namespace: jenkins.ObjectMeta.Namespace, Name: seedJob.CredentialID}
			err := s.k8sClient.Get(context.TODO(), namespaceName, secret)
			if err != nil {
				return stackerr.WithStack(err)
			}

			if !resources.VerifyIfLabelsAreSet(secret, requiredLabels) {
				secret.ObjectMeta.Labels = requiredLabels
				err = stackerr.WithStack(s.k8sClient.Update(context.TODO(), secret))
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (s *SeedJobs) credentialValue(namespace string, seedJob v1alpha2.SeedJob) (string, error) {
	if seedJob.JenkinsCredentialType == v1alpha2.BasicSSHCredentialType || seedJob.JenkinsCredentialType == v1alpha2.UsernamePasswordCredentialType {
		secret := &corev1.Secret{}
		namespaceName := types.NamespacedName{Namespace: namespace, Name: seedJob.CredentialID}
		err := s.k8sClient.Get(context.TODO(), namespaceName, secret)
		if err != nil {
			return "", err
		}

		if seedJob.JenkinsCredentialType == v1alpha2.BasicSSHCredentialType {
			return string(secret.Data[PrivateKeySecretKey]), nil
		}
		return string(secret.Data[UsernameSecretKey]) + string(secret.Data[PasswordSecretKey]), nil
	}
	return "", nil
}

func (s *SeedJobs) getAllSeedJobIDs(jenkins v1alpha2.Jenkins) []string {
	var ids []string
	for _, seedJob := range jenkins.Spec.SeedJobs {
		ids = append(ids, seedJob.ID)
	}
	return ids
}

//TODO move to k8sClient
func (s *SeedJobs) getJenkinsMasterPod(jenkins v1alpha2.Jenkins) (*corev1.Pod, error) {
	jenkinsMasterPodName := resources.GetJenkinsMasterPodName(jenkins)
	currentJenkinsMasterPod := &corev1.Pod{}
	err := s.k8sClient.Get(context.TODO(), types.NamespacedName{Name: jenkinsMasterPodName, Namespace: jenkins.Namespace}, currentJenkinsMasterPod)
	if err != nil {
		return nil, err // don't wrap error
	}
	return currentJenkinsMasterPod, nil
}

//TODO move to k8sClient
func (s *SeedJobs) restartJenkinsMasterPod(jenkins v1alpha2.Jenkins) error {
	currentJenkinsMasterPod, err := s.getJenkinsMasterPod(jenkins)
	if err != nil {
		return err
	}
	s.logger.Info(fmt.Sprintf("Terminating Jenkins Master Pod %s/%s", currentJenkinsMasterPod.Namespace, currentJenkinsMasterPod.Name))
	return stackerr.WithStack(s.k8sClient.Delete(context.TODO(), currentJenkinsMasterPod))
}

func (s *SeedJobs) isRecreatePodNeeded(jenkins v1alpha2.Jenkins) bool {
	for _, createdSeedJob := range jenkins.Status.CreatedSeedJobs {
		found := false
		for _, seedJob := range jenkins.Spec.SeedJobs {
			if createdSeedJob == seedJob.ID {
				found = true
				break
			}
		}
		if !found {
			return true
		}
	}
	return false
}

// CreateAgent deploys Jenkins agent to Kubernetes cluster
func (s SeedJobs) createAgent(jenkinsClient jenkinsclient.Jenkins, k8sClient client.Client, jenkinsManifest *v1alpha2.Jenkins, namespace string, agentName string) error {
	var exists bool

	nodes, err := jenkinsClient.GetAllNodes()
	if err != nil {
		return err
	}

	if len(nodes) != 0 {
		for _, node := range nodes {
			if node.GetName() == agentName {
				exists = true
			}
		}
	}

	// Create node if not exists
	if !exists {
		_, err = jenkinsClient.CreateNode(agentName, 1, "The jenkins-operator generated agent", "/home/jenkins", agentName)
		if err != nil {
			return err
		}
	}

	secret, err := jenkinsClient.GetNodeSecret(agentName)
	if err != nil {
		return err
	}

	deployment := agentDeployment(jenkinsManifest, namespace, agentName, secret)

	err = k8sClient.Create(context.TODO(), deployment)
	if err != nil {
		err := k8sClient.Update(context.TODO(), deployment)
		if err != nil {
			return err
		}
	}

	return nil
}

func agentDeployment(jenkinsManifest *v1alpha2.Jenkins, namespace string, agentName string, secret string) *apps.Deployment {
	return &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jnlp",
			Namespace: namespace,
		},
		Spec: apps.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  fmt.Sprintf("%s-container", agentName),
							Image: "jenkins/jnlp-slave:alpine",
							Env: []corev1.EnvVar{
								{
									Name: "JENKINS_TUNNEL",
									Value: fmt.Sprintf("%s.%s:%d",
										resources.GetJenkinsSlavesServiceName(jenkinsManifest),
										jenkinsManifest.ObjectMeta.Namespace,
										jenkinsManifest.Spec.SlaveService.Port),
								},
								{
									Name:  "JENKINS_SECRET",
									Value: secret,
								},
								{
									Name:  "JENKINS_AGENT_NAME",
									Value: agentName,
								},
								{
									Name: "JENKINS_URL",
									Value: fmt.Sprintf("http://%s.%s:%d",
										resources.GetJenkinsHTTPServiceName(jenkinsManifest),
										jenkinsManifest.ObjectMeta.Namespace,
										jenkinsManifest.Spec.Service.Port,
									),
								},
								{
									Name:  "JENKINS_AGENT_WORKDIR",
									Value: "/home/jenkins/agent",
								},
							},
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": fmt.Sprintf("%s-selector", agentName),
					},
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": fmt.Sprintf("%s-selector", agentName),
				},
			},
		},
	}
}

func seedJobCreatingGroovyScript(s v1alpha2.SeedJob) string {
	return `
import hudson.model.FreeStyleProject;
import hudson.plugins.git.GitSCM;
import hudson.plugins.git.BranchSpec;
import hudson.triggers.SCMTrigger;
import hudson.util.Secret;
import javaposse.jobdsl.plugin.*;
import jenkins.model.Jenkins;
import jenkins.model.JenkinsLocationConfiguration;
import com.cloudbees.plugins.credentials.CredentialsScope;
import com.cloudbees.plugins.credentials.domains.Domain;
import com.cloudbees.plugins.credentials.SystemCredentialsProvider;
import jenkins.model.JenkinsLocationConfiguration;
import org.jenkinsci.plugins.workflow.job.WorkflowJob;
import org.jenkinsci.plugins.workflow.cps.CpsScmFlowDefinition;

import hudson.model.FreeStyleProject
import hudson.model.labels.LabelAtom
import hudson.plugins.git.BranchSpec
import hudson.plugins.git.GitSCM
import hudson.plugins.git.SubmoduleConfig
import hudson.plugins.git.extensions.impl.CloneOption
import javaposse.jobdsl.plugin.ExecuteDslScripts
import javaposse.jobdsl.plugin.LookupStrategy
import javaposse.jobdsl.plugin.RemovedJobAction
import javaposse.jobdsl.plugin.RemovedViewAction

import static com.google.common.collect.Lists.newArrayList

Jenkins jenkins = Jenkins.instance

def jobDslSeedName = "` + s.ID + `-` + constants.SeedJobSuffix + `";
def jobRef = jenkins.getItem(jobDslSeedName)

def repoList = GitSCM.createRepoList("` + s.RepositoryURL + `", "` + s.CredentialID + `")
def gitExtensions = [new CloneOption(true, true, ";", 10)]
def scm = new GitSCM(
        repoList,
        newArrayList(new BranchSpec("` + s.RepositoryBranch + `")),
        false,
        Collections.<SubmoduleConfig>emptyList(),
        null,
        null,
        gitExtensions
)

def executeDslScripts = new ExecuteDslScripts()
executeDslScripts.setTargets("` + s.Targets + `")
executeDslScripts.setSandbox(false)
executeDslScripts.setRemovedJobAction(RemovedJobAction.DELETE)
executeDslScripts.setRemovedViewAction(RemovedViewAction.DELETE)
executeDslScripts.setLookupStrategy(LookupStrategy.SEED_JOB)

if (jobRef == null) {
        jobRef = jenkins.createProject(FreeStyleProject, jobDslSeedName)
}

jobRef.getBuildersList().clear()
jobRef.getBuildersList().add(executeDslScripts)
jobRef.setDisplayName("` + fmt.Sprintf("Seed Job from %s", s.ID) + `")
jobRef.setScm(scm)

jobRef.setAssignedLabel(new LabelAtom("` + AgentName + `"))

jenkins.getQueue().schedule(jobRef)

`
}
