package seedjobs

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"

	"github.com/go-logr/logr"
	stackerr "github.com/pkg/errors"
	"github.com/robfig/cron"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

// ValidateSeedJobs verify seed jobs configuration
func (s *SeedJobs) ValidateSeedJobs(jenkins v1alpha2.Jenkins) ([]string, error) {
	var messages []string

	if msg := s.validateIfIDIsUnique(jenkins.Spec.SeedJobs); msg != nil {
		messages = append(messages, msg...)
	}

	for _, seedJob := range jenkins.Spec.SeedJobs {
		logger := s.logger.WithValues("seedJob", seedJob.ID).V(log.VWarn)

		if len(seedJob.ID) == 0 {
			messages = append(messages, "id can't be empty")
		}

		if len(seedJob.RepositoryBranch) == 0 {
			messages = append(messages, "repository branch can't be empty")
		}

		if len(seedJob.RepositoryURL) == 0 {
			messages = append(messages, "repository URL branch can't be empty")
		}

		if len(seedJob.Targets) == 0 {
			messages = append(messages, "targets can't be empty")
		}

		if _, ok := v1alpha2.AllowedJenkinsCredentialMap[string(seedJob.JenkinsCredentialType)]; !ok {
			messages = append(messages, "unknown credential type")
		}

		if (seedJob.JenkinsCredentialType == v1alpha2.BasicSSHCredentialType ||
			seedJob.JenkinsCredentialType == v1alpha2.UsernamePasswordCredentialType) && len(seedJob.CredentialID) == 0 {
			messages = append(messages, "credential ID can't be empty")
		}

		// validate repository url match private key
		if strings.Contains(seedJob.RepositoryURL, "git@") && seedJob.JenkinsCredentialType == v1alpha2.NoJenkinsCredentialCredentialType {
			messages = append(messages, "Jenkins credential must be set while using ssh repository url")
		}

		if seedJob.JenkinsCredentialType == v1alpha2.BasicSSHCredentialType || seedJob.JenkinsCredentialType == v1alpha2.UsernamePasswordCredentialType {
			secret := &v1.Secret{}
			namespaceName := types.NamespacedName{Namespace: jenkins.Namespace, Name: seedJob.CredentialID}
			err := s.k8sClient.Get(context.TODO(), namespaceName, secret)
			if err != nil && apierrors.IsNotFound(err) {
				messages = append(messages, fmt.Sprintf("required secret '%s' with Jenkins credential not found", seedJob.CredentialID))
			} else if err != nil {
				return nil, stackerr.WithStack(err)
			}

			if seedJob.JenkinsCredentialType == v1alpha2.BasicSSHCredentialType {
				if msg := validateBasicSSHSecret(logger, *secret); msg != nil {
					messages = append(messages, msg...)
				}
			}
			if seedJob.JenkinsCredentialType == v1alpha2.UsernamePasswordCredentialType {
				if msg := validateUsernamePasswordSecret(logger, *secret); msg != nil {
					messages = append(messages, msg...)
				}
			}
		}

		if len(seedJob.BuildPeriodically) > 0 {
			if msg := s.validateSchedule(seedJob, seedJob.BuildPeriodically, "buildPeriodically"); msg != nil {
				messages = append(messages, msg...)
			}
		}

		if len(seedJob.PollSCM) > 0 {
			if msg := s.validateSchedule(seedJob, seedJob.PollSCM, "pollSCM"); msg != nil {
				messages = append(messages, msg...)
			}
		}

		if seedJob.GitHubPushTrigger {
			if msg := s.validateGitHubPushTrigger(jenkins); msg != nil {
				messages = append(messages, msg...)
			}
		}
	}

	return messages, nil
}

func (s *SeedJobs) validateSchedule(job v1alpha2.SeedJob, str string, key string) []string {
	var messages []string
	_, err := cron.Parse(str)
	if err != nil {
		messages = append(messages, fmt.Sprintf("`%s` schedule '%s' is invalid cron spec in `%s`", key, str, job.ID))
	}
	return messages
}

func (s *SeedJobs) validateGitHubPushTrigger(jenkins v1alpha2.Jenkins) []string {
	var messages []string
	exists := false
	for _, plugin := range jenkins.Spec.Master.BasePlugins {
		if plugin.Name == "github" {
			exists = true
		}
	}

	userExists := false
	for _, plugin := range jenkins.Spec.Master.Plugins {
		if plugin.Name == "github" {
			userExists = true
		}
	}

	if !exists && !userExists {
		messages = append(messages, "githubPushTrigger is set. This function requires `github` plugin installed in .Spec.Master.Plugins because seed jobs Push Trigger function needs it")
	}
	return messages
}

func (s *SeedJobs) validateIfIDIsUnique(seedJobs []v1alpha2.SeedJob) []string {
	var messages []string
	ids := map[string]bool{}
	for _, seedJob := range seedJobs {
		if _, found := ids[seedJob.ID]; found {
			messages = append(messages, fmt.Sprintf("'%s' seed job ID is not unique", seedJob.ID))
		}
		ids[seedJob.ID] = true
	}
	return messages
}

func validateBasicSSHSecret(logger logr.InfoLogger, secret v1.Secret) []string {
	var messages []string
	username, exists := secret.Data[UsernameSecretKey]
	if !exists {
		messages = append(messages, fmt.Sprintf("required data '%s' not found in secret '%s'", UsernameSecretKey, secret.ObjectMeta.Name))
	}
	if len(username) == 0 {
		messages = append(messages, fmt.Sprintf("required data '%s' is empty in secret '%s'", UsernameSecretKey, secret.ObjectMeta.Name))
	}

	privateKey, exists := secret.Data[PrivateKeySecretKey]
	if !exists {
		messages = append(messages, fmt.Sprintf("required data '%s' not found in secret '%s'", PrivateKeySecretKey, secret.ObjectMeta.Name))
	}
	if len(string(privateKey)) == 0 {
		messages = append(messages, fmt.Sprintf("required data '%s' not found in secret '%s'", PrivateKeySecretKey, secret.ObjectMeta.Name))
	}
	if err := validatePrivateKey(string(privateKey)); err != nil {
		messages = append(messages, fmt.Sprintf("private key '%s' invalid in secret '%s': %s", PrivateKeySecretKey, secret.ObjectMeta.Name, err))
	}

	return messages
}

func validateUsernamePasswordSecret(logger logr.InfoLogger, secret v1.Secret) []string {
	var messages []string
	username, exists := secret.Data[UsernameSecretKey]
	if !exists {
		messages = append(messages, fmt.Sprintf("required data '%s' not found in secret '%s'", UsernameSecretKey, secret.ObjectMeta.Name))
	}
	if len(username) == 0 {
		messages = append(messages, fmt.Sprintf("required data '%s' is empty in secret '%s'", UsernameSecretKey, secret.ObjectMeta.Name))
	}
	password, exists := secret.Data[PasswordSecretKey]
	if !exists {
		messages = append(messages, fmt.Sprintf("required data '%s' not found in secret '%s'", PasswordSecretKey, secret.ObjectMeta.Name))
	}
	if len(password) == 0 {
		messages = append(messages, fmt.Sprintf("required data '%s' is empty in secret '%s'", PasswordSecretKey, secret.ObjectMeta.Name))
	}

	return messages
}

func validatePrivateKey(privateKey string) error {
	block, _ := pem.Decode([]byte(privateKey))
	if block == nil {
		return stackerr.New("failed to decode PEM block")
	}

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return stackerr.WithStack(err)
	}

	err = priv.Validate()
	if err != nil {
		return stackerr.WithStack(err)
	}

	return nil
}
