package resources

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	logx "github.com/jenkinsci/kubernetes-operator/pkg/log"
	"github.com/jenkinsci/kubernetes-operator/pkg/util"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var logger = logx.Log

//nolint:gocritic
func GetSecretData(k8sClient client.Client, secretName, namespace string) (data []byte, requeue bool, err error) {
	if len(secretName) == 0 {
		return data, false, nil
	}

	secret := &corev1.Secret{}
	err = k8sClient.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: namespace}, secret)
	if errors.IsNotFound(err) {
		logger.V(logx.VDebug).Info(fmt.Sprintf("Secret %s in namespace %s not found\n", secretName, namespace))
		return data, true, err
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		logger.V(logx.VDebug).Info(fmt.Sprintf("Error getting secret %s in namespace %s: %v\n", secret, namespace, statusError.ErrStatus.Message))
		return data, true, err
	} else if err != nil {
		return data, true, err
	} else {
		logger.V(logx.VDebug).Info(fmt.Sprintf("Found secret %s in namespace %s\n", secretName, namespace))
		for _, v := range secret.Data {
			data = append(data, v...)
		}
	}

	return data, false, nil
}

func WriteDataToTempFile(data []byte) (filename string, requeue bool, err error) {
	tmpFile, err := ioutil.TempFile(os.TempDir(), "operator-sec-")
	if err != nil {
		logger.V(logx.VDebug).Info("Cannot create temporary file")
		return filename, true, err
	}

	logger.V(logx.VDebug).Info(fmt.Sprintf("Created File: %s", tmpFile.Name()))

	if _, err := tmpFile.Write(data); err != nil {
		logger.V(logx.VDebug).Info("Failed to write to temporary file")
		return filename, true, err
	}

	if err := tmpFile.Close(); err != nil {
		logger.V(logx.VDebug).Info("Failed to close temporary file")
		return filename, true, err
	}

	return tmpFile.Name(), false, nil
}

func CopySecret(k8sClient client.Client, k8sClientSet kubernetes.Clientset, restConfig *rest.Config, podName, secretName, namespace string) (requeue bool, err error) {
	data, requeue, err := GetSecretData(k8sClient, secretName, namespace)
	if err != nil {
		return requeue, err
	}
	if len(data) > 0 {
		fn, requeue, err := WriteDataToTempFile(data)
		if err != nil {
			return requeue, err
		}

		defer os.Remove(fn)

		//wait for jenkins pods running
		if err = WaitForPodRunning(k8sClient, podName, namespace, time.Duration(30)*time.Second); err != nil {
			logger.Error(err, "")
			return true, err
		}

		co := util.NewCopyOptions(restConfig, k8sClientSet, genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
		err = co.Run([]string{fn, fmt.Sprintf("%s/%s:%s", namespace, podName, ConfigurationAsCodeSecretVolumePath)})
		if err != nil {
			return true, err
		}
	}

	return false, nil
}
