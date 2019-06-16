package backuprestore

import (
	"bytes"
	"context"
	"fmt"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

// BackupAndRestore represents Jenkins backup and restore client
type BackupAndRestore struct {
	config    rest.Config
	k8sClient k8s.Client
	clientSet kubernetes.Clientset

	jenkinsClient jenkinsclient.Jenkins
	logger        logr.Logger
	jenkins       *v1alpha2.Jenkins
}

// New returns Jenkins backup and restore client
func New(k8sClient k8s.Client, clientSet kubernetes.Clientset, jenkinsClient jenkinsclient.Jenkins,
	logger logr.Logger, jenkins *v1alpha2.Jenkins, config rest.Config) *BackupAndRestore {
	return &BackupAndRestore{k8sClient: k8sClient, clientSet: clientSet, jenkinsClient: jenkinsClient, logger: logger, jenkins: jenkins, config: config}
}

// Validate validates backup and restore configuration
func (bar *BackupAndRestore) Validate() bool {
	valid := true
	allContainers := map[string]v1alpha2.Container{}
	for _, container := range bar.jenkins.Spec.Master.Containers {
		allContainers[container.Name] = container
	}

	restore := bar.jenkins.Spec.Restore
	if len(restore.ContainerName) > 0 {
		_, found := allContainers[restore.ContainerName]
		if !found {
			valid = false
			bar.logger.V(log.VWarn).Info(fmt.Sprintf("restore container '%s' not found in CR spec.master.containers", restore.ContainerName))
		}
		if restore.Action.Exec == nil {
			valid = false
			bar.logger.V(log.VWarn).Info(fmt.Sprintf("spec.restore.action.exec is not configured"))
		}
	}

	backup := bar.jenkins.Spec.Backup
	if len(backup.ContainerName) > 0 {
		_, found := allContainers[backup.ContainerName]
		if !found {
			valid = false
			bar.logger.V(log.VWarn).Info(fmt.Sprintf("backup container '%s' not found in CR spec.master.containers", backup.ContainerName))
		}
		if backup.Action.Exec == nil {
			valid = false
			bar.logger.V(log.VWarn).Info(fmt.Sprintf("spec.backup.action.exec is not configured"))
		}
		if backup.Interval == 0 {
			valid = false
			bar.logger.V(log.VWarn).Info(fmt.Sprintf("spec.backup.interval is not configured"))
		}
	}

	if len(restore.ContainerName) > 0 && len(backup.ContainerName) == 0 {
		valid = false
		bar.logger.V(log.VWarn).Info("spec.backup.containerName is not configured")
	}
	if len(backup.ContainerName) > 0 && len(restore.ContainerName) == 0 {
		valid = false
		bar.logger.V(log.VWarn).Info("spec.restore.containerName is not configured")
	}

	return valid
}

// Restore performs Jenkins restore backup operation
func (bar *BackupAndRestore) Restore() error {
	jenkins := bar.jenkins
	if jenkins.Status.RestoredBackup != 0 {
		bar.logger.V(log.VDebug).Info("Skipping restore backup, backup already restored")
		return nil
	}
	if jenkins.Status.LastBackup == 0 {
		bar.logger.Info("Skipping restore backup")
		if jenkins.Status.PendingBackup == 0 {
			jenkins.Status.PendingBackup = 1
			return bar.k8sClient.Update(context.TODO(), jenkins)
		}
		return nil
	}

	var backupNumber uint64
	if jenkins.Spec.Restore.RecoveryOnce != 0 {
		backupNumber = jenkins.Spec.Restore.RecoveryOnce
	} else {
		backupNumber = jenkins.Status.LastBackup
	}
	bar.logger.Info(fmt.Sprintf("Restoring backup '%d'", backupNumber))
	podName := resources.GetJenkinsMasterPodName(*jenkins)
	command := jenkins.Spec.Restore.Action.Exec.Command
	command = append(command, fmt.Sprintf("%d", backupNumber))
	_, _, err := bar.exec(podName, jenkins.Spec.Restore.ContainerName, command)

	if err == nil {
		jenkins.Spec.Restore.RecoveryOnce = 0
		jenkins.Status.RestoredBackup = backupNumber
		jenkins.Status.PendingBackup = backupNumber + 1
		return bar.k8sClient.Update(context.TODO(), jenkins)
	}

	//TODO reload?
	//TODO after 3 fails stop

	return err
}

// Backup performs Jenkins backup operation
func (bar *BackupAndRestore) Backup() error {
	jenkins := bar.jenkins
	if jenkins.Status.PendingBackup == jenkins.Status.LastBackup {
		bar.logger.V(log.VDebug).Info("Skipping backup")
		return nil
	}
	backupNumber := jenkins.Status.PendingBackup
	bar.logger.Info(fmt.Sprintf("Performing backup '%d'", backupNumber))
	podName := resources.GetJenkinsMasterPodName(*jenkins)
	command := jenkins.Spec.Backup.Action.Exec.Command
	command = append(command, fmt.Sprintf("%d", backupNumber))
	_, _, err := bar.exec(podName, jenkins.Spec.Backup.ContainerName, command)

	if err == nil {
		if jenkins.Status.RestoredBackup == 0 {
			jenkins.Status.RestoredBackup = backupNumber
		}
		jenkins.Status.LastBackup = backupNumber
		jenkins.Status.PendingBackup = backupNumber
		return bar.k8sClient.Update(context.TODO(), jenkins)
	}

	//TODO after 3 fails stop

	return err
}

func (bar *BackupAndRestore) exec(podName, containerName string, command []string) (stdout, stderr bytes.Buffer, err error) {
	req := bar.clientSet.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(bar.jenkins.Namespace).
		SubResource("exec")
	req.VersionedParams(&corev1.PodExecOptions{
		Command:   command,
		Container: containerName,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(&bar.config, "POST", req.URL())
	if err != nil {
		return stdout, stderr, errors.Wrap(err, "pod exec error while creating Executor")
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	bar.logger.V(log.VDebug).Info(fmt.Sprintf("pod exec: stdout '%s' stderr '%s'", stdout.String(), stderr.String()))
	if err != nil {
		return stdout, stderr, errors.Wrapf(err, "pod exec error operation on stream: stdout '%s' stderr '%s'", stdout.String(), stderr.String())
	}

	return
}
