/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/go-logr/logr"
	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/event"
	"github.com/jenkinsci/kubernetes-operator/pkg/notifications/reason"
	"github.com/jenkinsci/kubernetes-operator/pkg/plugins"

	// routev1 "github.com/openshift/api/route/v1"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	//	"math/rand"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	DefaultJenkinsImageEnvVar = "DEFAULT_JENKINS_IMAGE"

	APIVersion             = "core/v1"
	containerProbeURI      = "login"
	containerProbePortName = "http"

	reconcileInit             = "Init"
	reconcileInitMessage      = "Initializing Jenkins operator"
	reconcileFailed           = "ReconciliationFailed"
	reconcileCompleted        = "ReconciliationCompleted"
	reconcileCompletedMessage = "Reconciliation completed successfully"

	ConditionReconcileComplete conditionsv1.ConditionType = "ReconciliationComplete"

	DefaultBackupConfigName = "default"
)

// JenkinsReconciler reconciles a Jenkins object
type JenkinsReconciler struct {
	client.Client
	Log                          logr.Logger
	Scheme                       *runtime.Scheme
	jenkinsAPIConnectionSettings jenkinsclient.JenkinsAPIConnectionSettings
	NotificationEvents           chan event.Event
}

type reconcileError struct {
	err     error
	counter uint64
}

var reconcileErrors = map[string]reconcileError{}

func (r *JenkinsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha2.Jenkins{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
	// Owns(&routev1.Route{}).Complete(r)
}

// +kubebuilder:rbac:groups=jenkins.io,resources=jenkins,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jenkins.io,resources=jenkins/finalizers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jenkins.io,resources=jenkins/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=jenkins.io,resources=backupconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jenkins.io,resources=backupconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=replicasets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets/finalizers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps/finalizers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/log,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/exec,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/portforward,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=*,verbs=get;list;watch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=route.openshift.io,resources=routes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=build.openshift.io,resources=builds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=build.openshift.io,resources=buildconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=image.openshift.io,resources=imagestreams,verbs=get;list;watch;create;update;patch;delete

func (r *JenkinsReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	jenkinsName := request.NamespacedName
	logger := r.Log.WithValues("jenkins", jenkinsName)
	reconcileFailLimit := uint64(10)
	logger.V(log.VDebug).Info(fmt.Sprintf("Got a reconcialition request at: %+v for Jenkins: %s in namespace: %s", time.Now(), request.Name, request.Namespace))

	// Fetch the Jenkins jenkins
	jenkins := &v1alpha2.Jenkins{}
	err := r.Client.Get(ctx, jenkinsName, jenkins)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			logger.Info("API returned not found error: Deletion has been performed: " + request.String())
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Info(fmt.Sprintf("Error while trying to get jenkins named:  %s : %s", jenkinsName, err))
		return ctrl.Result{}, err
	}
	logger.Info(fmt.Sprintf("Jenkins instance correctly found: %s", jenkins.UID))
	if jenkins.Status == nil {
		jenkins.Status = &v1alpha2.JenkinsStatus{}
	}
	// setInitialConditions
	if jenkins.Status.Conditions == nil {
		setInitialConditions(jenkins)
		err = r.updateJenkinsStatus(jenkins, jenkinsName)
		if err != nil {
			logger.Info(fmt.Sprintf("Failed to set initial conditions to status: %s", err))
			return ctrl.Result{Requeue: true}, err
		}
	}

	result, err := r.reconcile(ctx, request, jenkins)
	if err != nil {
		reconciliationFailed := conditionsv1.Condition{
			Type:    conditionsv1.ConditionDegraded,
			Status:  corev1.ConditionTrue,
			Reason:  reconcileFailed,
			Message: fmt.Sprintf("Failed reconciliation %v", err),
		}
		conditionsv1.SetStatusCondition(&jenkins.Status.Conditions, reconciliationFailed)
		err = r.Status().Update(ctx, jenkins)
		if err != nil {
			logger.V(log.VWarn).Info(fmt.Sprintf("Failed to add conditions to status: %s", err))
		}
		return result, err
	}

	if err != nil && apierrors.IsConflict(err) {
		logger.V(log.VWarn).Info(fmt.Sprintf("Reconcile loop failed 1#: %+v", err))

		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		logger.V(log.VWarn).Info(fmt.Sprintf("Reconcile loop failed 2#: %+v", err))
		lastErrors, found := reconcileErrors[request.Name]
		if found {
			if err.Error() == lastErrors.err.Error() {
				lastErrors.counter++
			} else {
				lastErrors.counter = 1
				lastErrors.err = err
			}
		} else {
			lastErrors = reconcileError{
				err:     err,
				counter: 1,
			}
		}
		reconcileErrors[request.Name] = lastErrors
		if lastErrors.counter >= reconcileFailLimit {
			logger.V(log.VWarn).Info(fmt.Sprintf("Reconcile loop failed %d times with the same error, giving up: %+v", reconcileFailLimit, err))
			r.sendNewReconcileLoopFailedNotification(jenkins, reconcileFailLimit, err)
			return ctrl.Result{}, nil
		}

		if log.Debug {
			logger.V(log.VWarn).Info(fmt.Sprintf("Reconcile loop failed: %+v", err))
		} else if err.Error() != fmt.Sprintf("Operation cannot be fulfilled on jenkins.io \"%s\": the object has been modified; please apply your changes to the latest version and try again", request.Name) {
			logger.V(log.VWarn).Info(fmt.Sprintf("Reconcile loop failed: %s", err))
		}

		if groovyErr, ok := err.(*jenkinsclient.GroovyScriptExecutionFailed); ok {
			r.sendNewGroovyScriptExecutionFailedNotification(jenkins, groovyErr)
			return ctrl.Result{}, nil
		}
		logger.V(log.VWarn).Info(fmt.Sprintf("Requeing: !!! Reconcile loop failed: %+v", err))
		return ctrl.Result{Requeue: false}, nil
	}

	r.setStatusConditions(jenkins)
	err = r.updateJenkinsStatus(jenkins, jenkinsName)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}
	logger.Info("Reconcile loop success !!!")
	return ctrl.Result{}, nil
}

func (r *JenkinsReconciler) updateJenkinsStatus(jenkins *v1alpha2.Jenkins, jenkinsName types.NamespacedName) error {
	ctx := context.Background()
	err := r.Status().Update(ctx, jenkins)
	if err != nil {
		r.Log.Info("Failed to update Jenkins status...reloading")
		err = r.Client.Get(ctx, jenkinsName, jenkins)
		if err != nil {
			r.Log.Info("Failed to get Jenkins twice...")
			return err
		}
		if jenkins.Status == nil {
			jenkins.Status = &v1alpha2.JenkinsStatus{}
		}
		r.setStatusConditions(jenkins)
		err = r.Status().Update(ctx, jenkins)
		return err
	}
	return nil
}

func (r *JenkinsReconciler) setStatusConditions(jenkins *v1alpha2.Jenkins) {
	conditionsv1.SetStatusCondition(&jenkins.Status.Conditions, conditionsv1.Condition{
		Type:    ConditionReconcileComplete,
		Status:  corev1.ConditionTrue,
		Reason:  reconcileCompleted,
		Message: reconcileCompletedMessage,
	})
	conditionsv1.SetStatusCondition(&jenkins.Status.Conditions, conditionsv1.Condition{
		Type:    conditionsv1.ConditionAvailable,
		Status:  corev1.ConditionTrue,
		Reason:  reconcileCompleted,
		Message: reconcileCompletedMessage,
	})
	conditionsv1.SetStatusCondition(&jenkins.Status.Conditions, conditionsv1.Condition{
		Type:    conditionsv1.ConditionProgressing,
		Status:  corev1.ConditionFalse,
		Reason:  reconcileCompleted,
		Message: reconcileCompletedMessage,
	})
	conditionsv1.SetStatusCondition(&jenkins.Status.Conditions, conditionsv1.Condition{
		Type:    conditionsv1.ConditionDegraded,
		Status:  corev1.ConditionFalse,
		Reason:  reconcileCompleted,
		Message: reconcileCompletedMessage,
	})
	conditionsv1.SetStatusCondition(&jenkins.Status.Conditions, conditionsv1.Condition{
		Type:    conditionsv1.ConditionUpgradeable,
		Status:  corev1.ConditionTrue,
		Reason:  reconcileCompleted,
		Message: reconcileCompletedMessage,
	})
}

func (r *JenkinsReconciler) reconcile(ctx context.Context, request ctrl.Request, jenkins *v1alpha2.Jenkins) (ctrl.Result, error) {
	logger := r.Log.WithValues("cr", request.Name)
	var err error
	var requeue bool
	_, err = r.setDefaults(ctx, jenkins)
	if err != nil {
		logger.V(log.VDebug).Info(fmt.Sprintf("setDefaults returned an error %s", err))
		return reconcile.Result{}, err
	}

	if jenkins.Status.Spec.BackupEnabled {
		// Create PVC for Backup VolumeMount
		pvc := &corev1.PersistentVolumeClaim{}
		jenkinsBackupPVCName := request.Name + "-jenkins-backup"
		pvcNamespacedName := types.NamespacedName{
			Namespace: request.Namespace,
			Name:      jenkinsBackupPVCName,
		}
		err = r.Client.Get(context.TODO(), pvcNamespacedName, pvc)
		if err != nil {
			if apierrors.IsNotFound(err) {
				logger.Info(fmt.Sprintf("Creating PVC %s in Namespace %s for Jenkins instance %s",
					pvcNamespacedName.Name,
					pvcNamespacedName.Namespace,
					jenkinsBackupPVCName))
				pvc.Name = jenkinsBackupPVCName
				pvc.Namespace = request.Namespace
				pvc.Spec = corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("2Gi"),
						},
					},
				}
				err = r.Client.Create(context.TODO(), pvc)
				if err != nil {
					return ctrl.Result{}, err
				}
			}
		}

		// Create default BackupConfig
		defaultBackupConfig := &v1alpha2.BackupConfig{}
		backupConfigNamespacedName := types.NamespacedName{
			Namespace: request.Namespace,
			Name:      DefaultBackupConfigName,
		}
		err = r.Client.Get(context.TODO(), backupConfigNamespacedName, defaultBackupConfig)
		if err != nil {
			if apierrors.IsNotFound(err) {
				logger.Info(fmt.Sprintf("Creating BackupConfig %s in Namespace %s for Jenkins instance %s",
					pvcNamespacedName.Name,
					pvcNamespacedName.Namespace,
					DefaultBackupConfigName))
				defaultBackupConfig.Name = DefaultBackupConfigName
				defaultBackupConfig.Namespace = request.Namespace
				defaultBackupConfig.Spec.JenkinsRef = jenkins.Name
				defaultBackupConfig.Spec.Options = v1alpha2.BackupOptions{
					Jobs:    true,
					Plugins: true,
					Config:  true,
				}
				defaultBackupConfig.Spec.RestartAfterRestore = v1alpha2.RestartConfig{
					Enabled: true,
					Safe:    false,
				}
				err = r.Client.Create(context.TODO(), defaultBackupConfig)
				if err != nil {
					return ctrl.Result{}, err
				}
			}
		}
	}

	logger.V(log.VDebug).Info(fmt.Sprintf("setDefaults reported a change: %v", requeue))
	config := r.newReconcilerConfiguration(jenkins)
	// Reconcile base configuration
	logger.V(log.VDebug).Info("Starting base configuration reconciliation for validation")
	baseConfiguration := base.New(config, r.jenkinsAPIConnectionSettings)
	var baseConfigurationValidationMessages []string
	baseConfigurationValidationMessages, err = baseConfiguration.Validate(jenkins)
	if err != nil {
		logger.V(log.VDebug).Info(fmt.Sprintf("Error while trying to validate base configuration %s", err))
		return ctrl.Result{}, err
	}
	if len(baseConfigurationValidationMessages) > 0 {
		message := "Validation of base configuration failed, please correct Jenkins CR."
		r.sendNewBaseConfigurationFailedNotification(jenkins, message, baseConfigurationValidationMessages)
		logger.V(log.VWarn).Info(message)
		for _, msg := range baseConfigurationValidationMessages {
			logger.V(log.VWarn).Info(msg)
		}
		return ctrl.Result{}, nil // don't requeue
	}
	logger.V(log.VDebug).Info("Base configuration validation finished: No errors on validation messages")
	logger.V(log.VDebug).Info("Starting base configuration reconciliation...")
	_, _, err = baseConfiguration.Reconcile(request)
	if err != nil {
		if r.isJenkinsPodTerminating(err) {
			logger.Info(fmt.Sprintf("Jenkins Pod in Terminating state with DeletionTimestamp set detected. Changing Jenkins Phase to %s", constants.JenkinsStatusReinitializing))
			logger.Info("Base configuration reinitialized, jenkins pod restarted")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		logger.V(log.VDebug).Info(fmt.Sprintf("Base configuration reconciliation failed with error, requeuing:  %s ", err))
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}
	logger.V(log.VDebug).Info("Base configuration reconciliation successful.")
	// Re-reading Jenkins
	jenkins = &v1alpha2.Jenkins{}
	err = r.Client.Get(context.TODO(), request.NamespacedName, jenkins)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.Log.V(log.VWarn).Info(fmt.Sprintf("Object not found: %s: %+v", request, jenkins))
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		r.Log.V(log.VWarn).Info(fmt.Sprintf("Error reading object not found: %s: %+v", request, jenkins))
		return ctrl.Result{}, errors.WithStack(err)
	}

	return ctrl.Result{}, nil
}

func (r *JenkinsReconciler) sendNewBaseConfigurationFailedNotification(jenkins *v1alpha2.Jenkins, message string, baseMessages []string) {
	r.NotificationEvents <- event.Event{
		Jenkins: *jenkins,
		Phase:   event.PhaseBase,
		Level:   v1alpha2.NotificationLevelWarning,
		Reason:  reason.NewBaseConfigurationFailed(reason.HumanSource, []string{message}, append([]string{message}, baseMessages...)...),
	}
}

func (r *JenkinsReconciler) newReconcilerConfiguration(jenkins *v1alpha2.Jenkins) configuration.Configuration {
	config := configuration.Configuration{
		Client:                       r.Client,
		JenkinsAPIConnectionSettings: r.jenkinsAPIConnectionSettings,
		Notifications:                &r.NotificationEvents,
		Jenkins:                      jenkins,
		Scheme:                       r.Scheme,
	}
	return config
}

func (r *JenkinsReconciler) sendNewReconcileLoopFailedNotification(jenkins *v1alpha2.Jenkins, reconcileFailLimit uint64, err error) {
	r.NotificationEvents <- event.Event{
		Jenkins: *jenkins,
		Phase:   event.PhaseBase,
		Level:   v1alpha2.NotificationLevelWarning,
		Reason: reason.NewReconcileLoopFailed(
			reason.OperatorSource,
			[]string{fmt.Sprintf("Reconcile loop failed %d times with the same error, giving up: %s", reconcileFailLimit, err)},
		),
	}
}

func (r *JenkinsReconciler) setDefaults(ctx context.Context, jenkins *v1alpha2.Jenkins) (requeue bool, err error) {
	logger := r.Log.WithValues("cr", jenkins.Name)
	calculatedSpec, err := r.getCalculatedSpec(ctx, jenkins)
	if err != nil {
		logger.Info(fmt.Sprintf("Calculating defaulted spec returned an error:  %s", err))
		return false, err
	}
	logger.Info("Comparing current status.Spec")
	if !reflect.DeepEqual(jenkins.Status.Spec, calculatedSpec) {
		logger.Info("Current calculated spec is different from the newly calculated: resetting it")
		jenkins.Status.Spec = calculatedSpec
		logger.Info("Updating Jenkins status with requested and default values")
		err = r.Update(ctx, jenkins)
		return true, errors.WithStack(err)
	}
	return false, nil
}

func (r *JenkinsReconciler) getCalculatedSpec(ctx context.Context, jenkins *v1alpha2.Jenkins) (*v1alpha2.JenkinsSpec, error) {
	// getCalculatedSpec returns the calculated spec from the requested spec. It returns a JenkinsSpec containing
	// the requested specs and the defaulted values.
	jenkinsName := jenkins.Name
	logger := r.Log.WithValues("cr", jenkinsName)
	requestedSpec := jenkins.Spec

	// We make a copy of the requested spec, and we will build the actual one then
	calculatedSpec := requestedSpec.DeepCopy()
	jenkins.Status.Spec = calculatedSpec
	jenkinsContainer, err := r.getJenkinsMasterContainer(calculatedSpec)
	if err != nil {
		return nil, err
	}

	jenkinsMaster := r.getCalculatedJenkinsMaster(calculatedSpec, jenkinsContainer)
	calculatedSpec.Master = jenkinsMaster

	jenkinsMasterImage := r.getDefaultJenkinsImage()
	imageRef := requestedSpec.JenkinsImageRef
	if len(imageRef) != 0 {
		logger.Info(fmt.Sprintf("Jenkins %s has a jenkinsImageRef defined: %s", jenkinsName, imageRef))
		jenkinsImage := &v1alpha2.JenkinsImage{}
		err := r.Client.Get(ctx, types.NamespacedName{Namespace: jenkins.Namespace, Name: imageRef}, jenkinsImage)
		if err != nil {
			return nil, errors.Errorf("JenkinsImage '%s' referenced by Jenkins instance '%s' not found", imageRef, jenkinsName)
		}
		logger.Info(fmt.Sprintf("JenkinsImage found with Status:  %+v", jenkinsImage.Status))
		if jenkinsImage.Status.Phase == v1alpha2.ImageBuildSuccessful {
			// Get the latest build otherwise
			builds := jenkinsImage.Status.Builds
			imageSHA256 := builds[len(builds)-1].Image
			jenkinsMasterImage = fmt.Sprintf("%s/%s@%s", jenkinsImage.Spec.To.Registry, jenkinsImage.Spec.To.Name, imageSHA256)
			logger.Info(fmt.Sprintf("JenkinsImage found with latest build mapping to:  %s", jenkinsMasterImage))
		}
	}
	logger.Info("Setting default Jenkins master image: " + jenkinsMasterImage)
	jenkinsContainer.Image = jenkinsMasterImage
	jenkinsContainer.ImagePullPolicy = corev1.PullAlways

	if len(jenkinsContainer.ImagePullPolicy) == 0 {
		logger.Info(fmt.Sprintf("Setting default Jenkins master image pull policy: %s", corev1.PullAlways))
		jenkinsContainer.ImagePullPolicy = corev1.PullAlways
	}

	if jenkinsContainer.ReadinessProbe == nil {
		logger.Info("Setting default Jenkins readinessProbe")
		jenkinsContainer.ReadinessProbe = resources.NewSimpleProbe(containerProbeURI, containerProbePortName, corev1.URISchemeHTTP, 30)
	}
	if jenkinsContainer.LivenessProbe == nil {
		logger.Info("Setting default Jenkins livenessProbe")
		jenkinsContainer.LivenessProbe = resources.NewProbe(containerProbeURI, containerProbePortName, corev1.URISchemeHTTP, 80, 5, 12)
	}
	if isJavaOpsVariableNotSet(jenkinsContainer) {
		logger.Info("Setting default Jenkins container JAVA_OPTS environment variable")
		jenkinsContainer.Env = append(jenkinsContainer.Env, corev1.EnvVar{
			Name:  constants.JavaOpsVariableName,
			Value: "-XX:+UnlockExperimentalVMOptions -XX:MaxRAMFraction=1 -Djenkins.install.runSetupWizard=false -Djava.awt.headless=true -Dhudson.security.csrf.DefaultCrumbIssuer.EXCLUDE_SESSION_ID=true -Dcasc.reload.token=$(POD_NAME)",
		})
	}
	if calculatedSpec.Master.BasePlugins == nil {
		calculatedSpec.Master.BasePlugins = []v1alpha2.Plugin{}
	}
	if len(calculatedSpec.Master.BasePlugins) == 0 {
		logger.Info("Setting default operator plugins")
		calculatedSpec.Master.BasePlugins = basePlugins()
	}
	if isResourceRequirementsNotSet(jenkinsContainer.Resources) {
		logger.Info("Setting default Jenkins master container resource requirements")
		jenkinsContainer.Resources = resources.NewResourceRequirements("1", "500Mi", "1500m", "3Gi")
	}
	if reflect.DeepEqual(requestedSpec.Service, v1alpha2.Service{}) {
		logger.Info("Setting default Jenkins master service")
		serviceType := corev1.ServiceTypeClusterIP
		if r.jenkinsAPIConnectionSettings.UseNodePort {
			serviceType = corev1.ServiceTypeNodePort
		}
		calculatedSpec.Service = v1alpha2.Service{
			Type: serviceType,
			Port: constants.DefaultHTTPPortInt32,
		}
	}
	if reflect.DeepEqual(calculatedSpec.JNLPService, v1alpha2.Service{}) {
		logger.Info("Setting default Jenkins JNLP service")
		calculatedSpec.JNLPService = v1alpha2.Service{
			Type: corev1.ServiceTypeClusterIP,
			Port: constants.DefaultJNLPPortInt32,
		}
	}
	if len(calculatedSpec.Master.Containers) > 1 {
		for i, container := range calculatedSpec.Master.Containers[1:] {
			r.setDefaultsForContainer(jenkins, container.Name, i+1)
		}
	}

	if len(calculatedSpec.Master.Containers) == 0 || len(calculatedSpec.Master.Containers) == 1 {
		calculatedSpec.Master.Containers = []v1alpha2.Container{jenkinsContainer}
	} else {
		noJenkinsContainers := calculatedSpec.Master.Containers[1:]
		containers := []v1alpha2.Container{jenkinsContainer}
		containers = append(containers, noJenkinsContainers...)
		calculatedSpec.Master.Containers = containers
	}

	configurationAsCode := calculatedSpec.ConfigurationAsCode
	if configurationAsCode == nil {
		configurationAsCode = &v1alpha2.Configuration{
			Enabled:          true,
			DefaultConfig:    true,
			EnableAutoReload: true,
		}
	}
	calculatedSpec.ConfigurationAsCode = configurationAsCode
	return calculatedSpec, nil
}

func (r *JenkinsReconciler) getCalculatedJenkinsMaster(calculatedSpec *v1alpha2.JenkinsSpec, jenkinsContainer v1alpha2.Container) *v1alpha2.JenkinsMaster {
	var master *v1alpha2.JenkinsMaster
	if calculatedSpec.Master == nil {
		master = &v1alpha2.JenkinsMaster{
			Containers: []v1alpha2.Container{jenkinsContainer},
		}
	} else {
		master = calculatedSpec.Master.DeepCopy()
		calculatedSpec.Master.Containers[0] = jenkinsContainer
	}
	return master
}

func (r *JenkinsReconciler) getJenkinsMasterContainer(spec *v1alpha2.JenkinsSpec) (v1alpha2.Container, error) {
	var jenkinsContainer v1alpha2.Container
	if spec.Master == nil || len(spec.Master.Containers) == 0 {
		image := r.getDefaultJenkinsImage()
		jenkinsContainer = v1alpha2.Container{
			Name:  resources.JenkinsMasterContainerName,
			Image: image,
		}
	} else {
		if spec.Master.Containers[0].Name != resources.JenkinsMasterContainerName {
			error := errors.Errorf("first container in spec.master.containers must be Jenkins container with name '%s', please correct CR", resources.JenkinsMasterContainerName)
			return v1alpha2.Container{}, error
		}
		jenkinsContainer = spec.Master.Containers[0]
	}
	return jenkinsContainer, nil
}

func (r *JenkinsReconciler) sendNewGroovyScriptExecutionFailedNotification(jenkins *v1alpha2.Jenkins, groovyErr *jenkinsclient.GroovyScriptExecutionFailed) {
	r.NotificationEvents <- event.Event{
		Jenkins: *jenkins,
		Phase:   event.PhaseBase,
		Level:   v1alpha2.NotificationLevelWarning,
		Reason: reason.NewGroovyScriptExecutionFailed(
			reason.OperatorSource,
			[]string{fmt.Sprintf("%s Source '%s' Name '%s' groovy script execution failed", groovyErr.ConfigurationType, groovyErr.Source, groovyErr.Name)},
			[]string{fmt.Sprintf("%s Source '%s' Name '%s' groovy script execution failed, logs: %+v", groovyErr.ConfigurationType, groovyErr.Source, groovyErr.Name, groovyErr.Logs)}...,
		),
	}
}

func isJavaOpsVariableNotSet(container v1alpha2.Container) bool {
	for _, env := range container.Env {
		if env.Name == constants.JavaOpsVariableName {
			return false
		}
	}
	return true
}

func (r *JenkinsReconciler) setDefaultsForContainer(jenkins *v1alpha2.Jenkins, containerName string, containerIndex int) bool {
	changed := false
	logger := r.Log.WithValues("cr", jenkins.Name, "container", containerName)

	actualSpec := jenkins.Status.Spec
	if len(actualSpec.Master.Containers[containerIndex].ImagePullPolicy) == 0 {
		logger.Info(fmt.Sprintf("Setting default container image pull policy: %s", corev1.PullAlways))
		changed = true
		actualSpec.Master.Containers[containerIndex].ImagePullPolicy = corev1.PullAlways
	}

	if len(actualSpec.Master.Containers[containerIndex].ImagePullPolicy) == 0 {
		logger.Info(fmt.Sprintf("Setting default container image pull policy: %s", corev1.PullAlways))
		changed = true
		actualSpec.Master.Containers[containerIndex].ImagePullPolicy = corev1.PullAlways
	}
	if isResourceRequirementsNotSet(actualSpec.Master.Containers[containerIndex].Resources) {
		logger.Info("Setting default container resource requirements")
		changed = true
		actualSpec.Master.Containers[containerIndex].Resources = resources.NewResourceRequirements("50m", "50Mi", "100m", "100Mi")
	}
	return changed
}

func isResourceRequirementsNotSet(requirements corev1.ResourceRequirements) bool {
	return reflect.DeepEqual(requirements, corev1.ResourceRequirements{})
}

func basePlugins() (result []v1alpha2.Plugin) {
	for _, value := range plugins.BasePlugins() {
		result = append(result, v1alpha2.Plugin{Name: value.Name, Version: value.Version})
	}
	return
}

func (r *JenkinsReconciler) isJenkinsPodTerminating(err error) bool {
	return strings.Contains(err.Error(), "Terminating state with DeletionTimestamp")
}

func setInitialConditions(jenkins *v1alpha2.Jenkins) {
	conditionsv1.SetStatusCondition(&jenkins.Status.Conditions, conditionsv1.Condition{
		Type:    ConditionReconcileComplete,
		Status:  corev1.ConditionUnknown, // we just started trying to reconcile
		Reason:  reconcileInit,
		Message: reconcileInitMessage,
	})
	conditionsv1.SetStatusCondition(&jenkins.Status.Conditions, conditionsv1.Condition{
		Type:    conditionsv1.ConditionAvailable,
		Status:  corev1.ConditionFalse,
		Reason:  reconcileInit,
		Message: reconcileInitMessage,
	})
	conditionsv1.SetStatusCondition(&jenkins.Status.Conditions, conditionsv1.Condition{
		Type:    conditionsv1.ConditionProgressing,
		Status:  corev1.ConditionTrue,
		Reason:  reconcileInit,
		Message: reconcileInitMessage,
	})
	conditionsv1.SetStatusCondition(&jenkins.Status.Conditions, conditionsv1.Condition{
		Type:    conditionsv1.ConditionDegraded,
		Status:  corev1.ConditionFalse,
		Reason:  reconcileInit,
		Message: reconcileInitMessage,
	})
	conditionsv1.SetStatusCondition(&jenkins.Status.Conditions, conditionsv1.Condition{
		Type:    conditionsv1.ConditionUpgradeable,
		Status:  corev1.ConditionUnknown,
		Reason:  reconcileInit,
		Message: reconcileInitMessage,
	})
}

// getDefaultJenkinsImage returns the default jenkins image the operator should be using
func (r *JenkinsReconciler) getDefaultJenkinsImage() string {
	jenkinsImage, _ := os.LookupEnv(DefaultJenkinsImageEnvVar)
	if len(jenkinsImage) == 0 {
		jenkinsImage = constants.DefaultJenkinsMasterImage
		if resources.RouteAPIAvailable {
			jenkinsImage = constants.DefaultOpenShiftJenkinsMasterImage
		}
	}
	return jenkinsImage
}
