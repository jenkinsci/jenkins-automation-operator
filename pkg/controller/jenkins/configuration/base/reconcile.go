package base

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkinsio/v1alpha1"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/constants"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/groovy"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/plugins"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"
	"github.com/jenkinsci/kubernetes-operator/version"

	"github.com/bndr/gojenkins"
	"github.com/go-logr/logr"
	stackerr "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	fetchAllPlugins = 1
)

// ReconcileJenkinsBaseConfiguration defines values required for Jenkins base configuration
type ReconcileJenkinsBaseConfiguration struct {
	k8sClient       client.Client
	scheme          *runtime.Scheme
	logger          logr.Logger
	jenkins         *v1alpha1.Jenkins
	local, minikube bool
}

// New create structure which takes care of base configuration
func New(client client.Client, scheme *runtime.Scheme, logger logr.Logger,
	jenkins *v1alpha1.Jenkins, local, minikube bool) *ReconcileJenkinsBaseConfiguration {
	return &ReconcileJenkinsBaseConfiguration{
		k8sClient: client,
		scheme:    scheme,
		logger:    logger,
		jenkins:   jenkins,
		local:     local,
		minikube:  minikube,
	}
}

// Reconcile takes care of base configuration
func (r *ReconcileJenkinsBaseConfiguration) Reconcile() (reconcile.Result, jenkinsclient.Jenkins, error) {
	metaObject := resources.NewResourceObjectMeta(r.jenkins)

	err := r.ensureResourcesRequiredForJenkinsPod(metaObject)
	if err != nil {
		return reconcile.Result{}, nil, err
	}

	result, err := r.ensureJenkinsMasterPod(metaObject)
	if err != nil {
		return reconcile.Result{}, nil, err
	}
	if result.Requeue {
		return result, nil, nil
	}
	r.logger.V(log.VDebug).Info("Jenkins master pod is present")

	result, err = r.waitForJenkins(metaObject)
	if err != nil {
		return reconcile.Result{}, nil, err
	}
	if result.Requeue {
		return result, nil, nil
	}
	r.logger.V(log.VDebug).Info("Jenkins master pod is ready")

	jenkinsClient, err := r.ensureJenkinsClient(metaObject)
	if err != nil {
		return reconcile.Result{}, nil, err
	}
	r.logger.V(log.VDebug).Info("Jenkins API client set")

	ok, err := r.verifyPlugins(jenkinsClient)
	if err != nil {
		return reconcile.Result{}, nil, err
	}
	if !ok {
		r.logger.V(log.VWarn).Info("Please correct Jenkins CR(spec.master.OperatorPlugins or spec.master.plugins)")
		// TODO inform user via Admin Monitor and don't restart Jenkins
		return reconcile.Result{Requeue: true}, nil, r.restartJenkinsMasterPod(metaObject)
	}

	result, err = r.ensureBaseConfiguration(jenkinsClient)
	return result, jenkinsClient, err
}

func (r *ReconcileJenkinsBaseConfiguration) ensureResourcesRequiredForJenkinsPod(metaObject metav1.ObjectMeta) error {
	if err := r.createOperatorCredentialsSecret(metaObject); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("Operator credentials secret is present")

	if err := r.createScriptsConfigMap(metaObject); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("Scripts config map is present")

	if err := r.createInitConfigurationConfigMap(metaObject); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("Init configuration config map is present")

	if err := r.createBaseConfigurationConfigMap(metaObject); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("Base configuration config map is present")

	if err := r.createUserConfigurationConfigMap(metaObject); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("User configuration config map is present")

	if err := r.createUserConfigurationSecret(metaObject); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("User configuration secret is present")

	if err := r.createRBAC(metaObject); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("Service account, role and role binding are present")

	if err := r.createService(metaObject, resources.GetJenkinsHTTPServiceName(r.jenkins), r.jenkins.Spec.Service); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("Jenkins HTTP Service is present")
	if err := r.createService(metaObject, resources.GetJenkinsSlavesServiceName(r.jenkins), r.jenkins.Spec.SlaveService); err != nil {
		return err
	}
	r.logger.V(log.VDebug).Info("Jenkins slave Service is present")

	return nil
}

func (r *ReconcileJenkinsBaseConfiguration) verifyPlugins(jenkinsClient jenkinsclient.Jenkins) (bool, error) {
	allPluginsInJenkins, err := jenkinsClient.GetPlugins(fetchAllPlugins)
	if err != nil {
		return false, stackerr.WithStack(err)
	}

	var installedPlugins []string
	for _, jenkinsPlugin := range allPluginsInJenkins.Raw.Plugins {
		if !jenkinsPlugin.Deleted {
			installedPlugins = append(installedPlugins, plugins.Plugin{Name: jenkinsPlugin.ShortName, Version: jenkinsPlugin.Version}.String())
		}
	}
	r.logger.V(log.VDebug).Info(fmt.Sprintf("Installed plugins '%+v'", installedPlugins))

	userPlugins := map[string][]plugins.Plugin{}
	for rootPlugin, dependentPluginNames := range r.jenkins.Spec.Master.Plugins {
		var dependentPlugins []plugins.Plugin
		for _, pluginNameWithVersion := range dependentPluginNames {
			plugin, err := plugins.New(pluginNameWithVersion)
			if err != nil {
				return false, err
			}
			dependentPlugins = append(dependentPlugins, *plugin)
		}
		userPlugins[rootPlugin] = dependentPlugins
	}

	status := true
	allRequiredPlugins := []map[string][]plugins.Plugin{plugins.BasePluginsMap, userPlugins}
	for _, requiredPlugins := range allRequiredPlugins {
		for rootPluginName, p := range requiredPlugins {
			rootPlugin, _ := plugins.New(rootPluginName)
			if found, ok := isPluginInstalled(allPluginsInJenkins, *rootPlugin); !ok {
				r.logger.V(log.VWarn).Info(fmt.Sprintf("Missing plugin '%s', actual '%+v'", rootPlugin, found))
				status = false
			}
			for _, requiredPlugin := range p {
				if found, ok := isPluginInstalled(allPluginsInJenkins, requiredPlugin); !ok {
					r.logger.V(log.VWarn).Info(fmt.Sprintf("Missing plugin '%s', actual '%+v'", requiredPlugin, found))
					status = false
				}
			}
		}
	}

	return status, nil
}

func isPluginInstalled(plugins *gojenkins.Plugins, requiredPlugin plugins.Plugin) (gojenkins.Plugin, bool) {
	p := plugins.Contains(requiredPlugin.Name)
	if p == nil {
		return gojenkins.Plugin{}, false
	}

	return *p, p.Active && p.Enabled && !p.Deleted
}

func (r *ReconcileJenkinsBaseConfiguration) createOperatorCredentialsSecret(meta metav1.ObjectMeta) error {
	found := &corev1.Secret{}
	err := r.k8sClient.Get(context.TODO(), types.NamespacedName{Name: resources.GetOperatorCredentialsSecretName(r.jenkins), Namespace: r.jenkins.ObjectMeta.Namespace}, found)

	if err != nil && apierrors.IsNotFound(err) {
		return stackerr.WithStack(r.createResource(resources.NewOperatorCredentialsSecret(meta, r.jenkins)))
	} else if err != nil && !apierrors.IsNotFound(err) {
		return stackerr.WithStack(err)
	}

	if found.Data[resources.OperatorCredentialsSecretUserNameKey] != nil &&
		found.Data[resources.OperatorCredentialsSecretPasswordKey] != nil {
		return nil
	}

	return stackerr.WithStack(r.updateResource(resources.NewOperatorCredentialsSecret(meta, r.jenkins)))
}

func (r *ReconcileJenkinsBaseConfiguration) createScriptsConfigMap(meta metav1.ObjectMeta) error {
	configMap, err := resources.NewScriptsConfigMap(meta, r.jenkins)
	if err != nil {
		return err
	}
	return stackerr.WithStack(r.createOrUpdateResource(configMap))
}

func (r *ReconcileJenkinsBaseConfiguration) createInitConfigurationConfigMap(meta metav1.ObjectMeta) error {
	configMap, err := resources.NewInitConfigurationConfigMap(meta, r.jenkins)
	if err != nil {
		return err
	}
	return stackerr.WithStack(r.createOrUpdateResource(configMap))
}

func (r *ReconcileJenkinsBaseConfiguration) createBaseConfigurationConfigMap(meta metav1.ObjectMeta) error {
	configMap := resources.NewBaseConfigurationConfigMap(meta, r.jenkins)
	return stackerr.WithStack(r.createOrUpdateResource(configMap))
}

func (r *ReconcileJenkinsBaseConfiguration) createUserConfigurationConfigMap(meta metav1.ObjectMeta) error {
	currentConfigMap := &corev1.ConfigMap{}
	err := r.k8sClient.Get(context.TODO(), types.NamespacedName{Name: resources.GetUserConfigurationConfigMapNameFromJenkins(r.jenkins), Namespace: r.jenkins.Namespace}, currentConfigMap)
	if err != nil && errors.IsNotFound(err) {
		return stackerr.WithStack(r.k8sClient.Create(context.TODO(), resources.NewUserConfigurationConfigMap(r.jenkins)))
	} else if err != nil {
		return stackerr.WithStack(err)
	}
	if !resources.VerifyIfLabelsAreSet(currentConfigMap, resources.BuildLabelsForWatchedResources(*r.jenkins)) {
		currentConfigMap.ObjectMeta.Labels = resources.BuildLabelsForWatchedResources(*r.jenkins)
		return stackerr.WithStack(r.k8sClient.Update(context.TODO(), currentConfigMap))
	}

	return nil
}

func (r *ReconcileJenkinsBaseConfiguration) createUserConfigurationSecret(meta metav1.ObjectMeta) error {
	currentSecret := &corev1.Secret{}
	err := r.k8sClient.Get(context.TODO(), types.NamespacedName{Name: resources.GetUserConfigurationSecretNameFromJenkins(r.jenkins), Namespace: r.jenkins.Namespace}, currentSecret)
	if err != nil && errors.IsNotFound(err) {
		return stackerr.WithStack(r.k8sClient.Create(context.TODO(), resources.NewUserConfigurationSecret(r.jenkins)))
	} else if err != nil {
		return stackerr.WithStack(err)
	}
	if !resources.VerifyIfLabelsAreSet(currentSecret, resources.BuildLabelsForWatchedResources(*r.jenkins)) {
		currentSecret.ObjectMeta.Labels = resources.BuildLabelsForWatchedResources(*r.jenkins)
		return stackerr.WithStack(r.k8sClient.Update(context.TODO(), currentSecret))
	}

	return nil
}

func (r *ReconcileJenkinsBaseConfiguration) createRBAC(meta metav1.ObjectMeta) error {
	serviceAccount := resources.NewServiceAccount(meta)
	err := r.createResource(serviceAccount)
	if err != nil && !errors.IsAlreadyExists(err) {
		return stackerr.WithStack(err)
	}

	role := resources.NewRole(meta)
	err = r.createOrUpdateResource(role)
	if err != nil {
		return stackerr.WithStack(err)
	}

	roleBinding := resources.NewRoleBinding(meta)
	err = r.createOrUpdateResource(roleBinding)
	if err != nil {
		return stackerr.WithStack(err)
	}

	return nil
}

func (r *ReconcileJenkinsBaseConfiguration) createService(meta metav1.ObjectMeta, name string, config v1alpha1.Service) error {
	service := corev1.Service{}
	err := r.k8sClient.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: meta.Namespace}, &service)
	if err != nil && errors.IsNotFound(err) {
		service = resources.UpdateService(corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: meta.Namespace,
				Labels:    meta.Labels,
			},
			Spec: corev1.ServiceSpec{
				Selector: meta.Labels,
			},
		}, config)
		if err = r.createResource(&service); err != nil {
			return stackerr.WithStack(err)
		}
	} else if err != nil {
		return stackerr.WithStack(err)
	}

	service = resources.UpdateService(service, config)
	return stackerr.WithStack(r.updateResource(&service))
}

func (r *ReconcileJenkinsBaseConfiguration) getJenkinsMasterPod(meta metav1.ObjectMeta) (*corev1.Pod, error) {
	jenkinsMasterPod := resources.NewJenkinsMasterPod(meta, r.jenkins)
	currentJenkinsMasterPod := &corev1.Pod{}
	err := r.k8sClient.Get(context.TODO(), types.NamespacedName{Name: jenkinsMasterPod.Name, Namespace: jenkinsMasterPod.Namespace}, currentJenkinsMasterPod)
	if err != nil {
		return nil, err // don't wrap error
	}
	return currentJenkinsMasterPod, nil
}

func (r *ReconcileJenkinsBaseConfiguration) ensureJenkinsMasterPod(meta metav1.ObjectMeta) (reconcile.Result, error) {
	// Check if this Pod already exists
	currentJenkinsMasterPod, err := r.getJenkinsMasterPod(meta)
	if err != nil && errors.IsNotFound(err) {
		jenkinsMasterPod := resources.NewJenkinsMasterPod(meta, r.jenkins)
		r.logger.Info(fmt.Sprintf("Creating a new Jenkins Master Pod %s/%s", jenkinsMasterPod.Namespace, jenkinsMasterPod.Name))
		err = r.createResource(jenkinsMasterPod)
		if err != nil {
			return reconcile.Result{}, stackerr.WithStack(err)
		}
		now := metav1.Now()
		r.jenkins.Status = v1alpha1.JenkinsStatus{
			ProvisionStartTime: &now,
		}
		err = r.updateResource(r.jenkins)
		if err != nil {
			return reconcile.Result{}, err // don't wrap error
		}
		return reconcile.Result{}, nil
	} else if err != nil && !errors.IsNotFound(err) {
		return reconcile.Result{}, stackerr.WithStack(err)
	}

	if currentJenkinsMasterPod != nil && isPodTerminating(*currentJenkinsMasterPod) {
		return reconcile.Result{Requeue: true}, nil
	}
	if currentJenkinsMasterPod != nil && r.isRecreatePodNeeded(*currentJenkinsMasterPod) {
		return reconcile.Result{Requeue: true}, r.restartJenkinsMasterPod(meta)
	}

	return reconcile.Result{}, nil
}

func isPodTerminating(pod corev1.Pod) bool {
	return pod.ObjectMeta.DeletionTimestamp != nil
}

func (r *ReconcileJenkinsBaseConfiguration) isRecreatePodNeeded(currentJenkinsMasterPod corev1.Pod) bool {
	if version.Version != r.jenkins.Status.OperatorVersion {
		r.logger.Info(fmt.Sprintf("Jenkins Operator version has changed, actual '%+v' new '%+v' - recreating pod",
			r.jenkins.Status.OperatorVersion, version.Version))
		return true
	}

	if currentJenkinsMasterPod.Status.Phase == corev1.PodFailed ||
		currentJenkinsMasterPod.Status.Phase == corev1.PodSucceeded ||
		currentJenkinsMasterPod.Status.Phase == corev1.PodUnknown {
		r.logger.Info(fmt.Sprintf("Invalid Jenkins pod phase '%+v', recreating pod", currentJenkinsMasterPod.Status))
		return true
	}

	if r.jenkins.Spec.Master.Image != currentJenkinsMasterPod.Spec.Containers[0].Image {
		r.logger.Info(fmt.Sprintf("Jenkins image has changed to '%+v', recreating pod", r.jenkins.Spec.Master.Image))
		return true
	}

	if r.jenkins.Spec.Master.ImagePullPolicy != currentJenkinsMasterPod.Spec.Containers[0].ImagePullPolicy {
		r.logger.Info(fmt.Sprintf("Jenkins image pull policy has changed to '%+v', recreating pod", r.jenkins.Spec.Master.ImagePullPolicy))
		return true
	}

	if len(r.jenkins.Spec.Master.Annotations) > 0 &&
		!reflect.DeepEqual(r.jenkins.Spec.Master.Annotations, currentJenkinsMasterPod.ObjectMeta.Annotations) {
		r.logger.Info(fmt.Sprintf("Jenkins pod annotations have changed to '%+v', recreating pod", r.jenkins.Spec.Master.Annotations))
		return true
	}

	if !reflect.DeepEqual(r.jenkins.Spec.Master.Resources, currentJenkinsMasterPod.Spec.Containers[0].Resources) {
		r.logger.Info(fmt.Sprintf("Jenkins pod resources have changed, actual '%+v' required '%+v' - recreating pod",
			currentJenkinsMasterPod.Spec.Containers[0].Resources, r.jenkins.Spec.Master.Resources))
		return true
	}

	if !reflect.DeepEqual(r.jenkins.Spec.Master.NodeSelector, currentJenkinsMasterPod.Spec.NodeSelector) {
		r.logger.Info(fmt.Sprintf("Jenkins pod node selector has changed, actual '%+v' required '%+v' - recreating pod",
			currentJenkinsMasterPod.Spec.NodeSelector, r.jenkins.Spec.Master.NodeSelector))
		return true
	}

	requiredEnvs := resources.GetJenkinsMasterPodBaseEnvs()
	requiredEnvs = append(requiredEnvs, r.jenkins.Spec.Master.Env...)
	if !reflect.DeepEqual(requiredEnvs, currentJenkinsMasterPod.Spec.Containers[0].Env) {
		r.logger.Info(fmt.Sprintf("Jenkins env have changed, actual '%+v' required '%+v' - recreating pod",
			currentJenkinsMasterPod.Spec.Containers[0].Env, requiredEnvs))
		return true
	}

	return false
}

func (r *ReconcileJenkinsBaseConfiguration) restartJenkinsMasterPod(meta metav1.ObjectMeta) error {
	currentJenkinsMasterPod, err := r.getJenkinsMasterPod(meta)
	r.logger.Info(fmt.Sprintf("Terminating Jenkins Master Pod %s/%s", currentJenkinsMasterPod.Namespace, currentJenkinsMasterPod.Name))
	if err != nil {
		return err
	}
	return stackerr.WithStack(r.k8sClient.Delete(context.TODO(), currentJenkinsMasterPod))
}

func (r *ReconcileJenkinsBaseConfiguration) waitForJenkins(meta metav1.ObjectMeta) (reconcile.Result, error) {
	jenkinsMasterPodStatus, err := r.getJenkinsMasterPod(meta)
	if err != nil {
		return reconcile.Result{}, err
	}

	if jenkinsMasterPodStatus.ObjectMeta.DeletionTimestamp != nil {
		r.logger.V(log.VDebug).Info("Jenkins master pod is terminating")
		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 5}, nil
	}

	if jenkinsMasterPodStatus.Status.Phase != corev1.PodRunning {
		r.logger.V(log.VDebug).Info("Jenkins master pod not ready")
		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 5}, nil
	}

	for _, containerStatus := range jenkinsMasterPodStatus.Status.ContainerStatuses {
		if !containerStatus.Ready {
			r.logger.V(log.VDebug).Info("Jenkins master pod not ready, readiness probe failed")
			return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 5}, nil
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileJenkinsBaseConfiguration) ensureJenkinsClient(meta metav1.ObjectMeta) (jenkinsclient.Jenkins, error) {
	jenkinsURL, err := jenkinsclient.BuildJenkinsAPIUrl(
		r.jenkins.ObjectMeta.Namespace, resources.GetJenkinsHTTPServiceName(r.jenkins), r.jenkins.Spec.Service.Port, r.local, r.minikube)
	if err != nil {
		return nil, err
	}
	r.logger.V(log.VDebug).Info(fmt.Sprintf("Jenkins API URL %s", jenkinsURL))

	credentialsSecret := &corev1.Secret{}
	err = r.k8sClient.Get(context.TODO(), types.NamespacedName{Name: resources.GetOperatorCredentialsSecretName(r.jenkins), Namespace: r.jenkins.ObjectMeta.Namespace}, credentialsSecret)
	if err != nil {
		return nil, stackerr.WithStack(err)
	}
	currentJenkinsMasterPod, err := r.getJenkinsMasterPod(meta)
	if err != nil {
		return nil, err
	}

	var tokenCreationTime *time.Time
	tokenCreationTimeBytes := credentialsSecret.Data[resources.OperatorCredentialsSecretTokenCreationKey]
	if tokenCreationTimeBytes != nil {
		tokenCreationTime = &time.Time{}
		err = tokenCreationTime.UnmarshalText(tokenCreationTimeBytes)
		if err != nil {
			tokenCreationTime = nil
		}

	}
	if credentialsSecret.Data[resources.OperatorCredentialsSecretTokenKey] == nil ||
		tokenCreationTimeBytes == nil || tokenCreationTime == nil ||
		currentJenkinsMasterPod.ObjectMeta.CreationTimestamp.Time.UTC().After(tokenCreationTime.UTC()) {
		r.logger.Info("Generating Jenkins API token for operator")
		userName := string(credentialsSecret.Data[resources.OperatorCredentialsSecretUserNameKey])
		jenkinsClient, err := jenkinsclient.New(
			jenkinsURL,
			userName,
			string(credentialsSecret.Data[resources.OperatorCredentialsSecretPasswordKey]))
		if err != nil {
			return nil, err
		}

		token, err := jenkinsClient.GenerateToken(userName, "token")
		if err != nil {
			return nil, err
		}

		credentialsSecret.Data[resources.OperatorCredentialsSecretTokenKey] = []byte(token.GetToken())
		now, _ := time.Now().UTC().MarshalText()
		credentialsSecret.Data[resources.OperatorCredentialsSecretTokenCreationKey] = now
		err = r.updateResource(credentialsSecret)
		if err != nil {
			return nil, stackerr.WithStack(err)
		}
	}

	return jenkinsclient.New(
		jenkinsURL,
		string(credentialsSecret.Data[resources.OperatorCredentialsSecretUserNameKey]),
		string(credentialsSecret.Data[resources.OperatorCredentialsSecretTokenKey]))
}

func (r *ReconcileJenkinsBaseConfiguration) ensureBaseConfiguration(jenkinsClient jenkinsclient.Jenkins) (reconcile.Result, error) {
	groovyClient := groovy.New(jenkinsClient, r.k8sClient, r.logger, fmt.Sprintf("%s-base-configuration", constants.OperatorName), resources.JenkinsBaseConfigurationVolumePath)

	err := groovyClient.ConfigureJob()
	if err != nil {
		return reconcile.Result{}, err
	}

	configuration := &corev1.ConfigMap{}
	namespaceName := types.NamespacedName{Namespace: r.jenkins.Namespace, Name: resources.GetBaseConfigurationConfigMapName(r.jenkins)}
	err = r.k8sClient.Get(context.TODO(), namespaceName, configuration)
	if err != nil {
		return reconcile.Result{}, stackerr.WithStack(err)
	}

	done, err := groovyClient.Ensure(configuration.Data, r.jenkins)
	if err != nil {
		return reconcile.Result{}, err
	}
	if !done {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Second * 10}, nil
	}

	return reconcile.Result{}, nil
}
