package notifier

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	testConfigurationType        = "test-configuration"
	testCrName                   = "test-cr"
	testStatus            Status = 1
	testError             error

	client = http.Client{}
)

const (
	// StatusSuccess contains value for success state
	StatusSuccess = 0

	// StatusError contains value for error state
	StatusError = 1

	noErrorMessage = "No errors has found."

	titleText                  = "Operator reconciled."
	statusMessageFieldName     = "Status message"
	statusFieldName            = "Status"
	crNameFieldName            = "CR Name"
	configurationTypeFieldName = "Configuration Type"
	footerContent              = "Powered by Jenkins Operator <3"
)

// Status represents the state of operator
type Status int

// StatusColor is useful for better UX
type StatusColor string

// Information represents details about operator status
type Information struct {
	ConfigurationType string
	CrName            string
	Status            Status
	Error             error
}

// Notification contains message which will be sent
type Notification struct {
	Jenkins   *v1alpha2.Jenkins
	K8sClient k8sclient.Client
	Logger    logr.Logger

	// Recipient is mobile number or email address
	// It's not used in Slack or Microsoft Teams
	Recipient string

	Information *Information
}

// Service is skeleton for additional services
type Service interface {
	Send(secret string, i *Information) error
}

// Listen is goroutine that listens for incoming messages and sends it
func Listen(notification chan *Notification) {
	n := <-notification
	if len(n.Jenkins.Spec.Notification) > 0 {
		for _, endpoint := range n.Jenkins.Spec.Notification {
			var err error
			var service Service
			var selector v1alpha2.SecretKeySelector
			secret := &corev1.Secret{}

			if endpoint.Slack != (v1alpha2.Slack{}) {
				n.Logger.V(log.VDebug).Info("Slack detected")
				service = Slack{}
				selector = endpoint.Slack.URLSecretKeySelector
			} else if endpoint.Teams != (v1alpha2.Teams{}) {
				n.Logger.V(log.VDebug).Info("Microsoft Teams detected")
				service = Teams{}
				selector = endpoint.Teams.URLSecretKeySelector
			} else if endpoint.Mailgun != (v1alpha2.Mailgun{}) {
				n.Logger.V(log.VDebug).Info("Mailgun detected")
				service = Mailgun{
					Domain:    endpoint.Mailgun.Domain,
					Recipient: endpoint.Mailgun.Recipient,
					From:      endpoint.Mailgun.From,
				}
				selector = endpoint.Mailgun.APIKeySecretKeySelector
			} else {
				n.Logger.Info("Notification service not found or not defined")
			}

			err = n.K8sClient.Get(context.TODO(), types.NamespacedName{Name: selector.Name, Namespace: n.Jenkins.Namespace}, secret)
			if err != nil {
				n.Logger.Info(fmt.Sprintf("Failed to get secret with name `%s`. %+v", selector.Name, err))
			}

			n.Logger.V(log.VDebug).Info(fmt.Sprintf("Endpoint URL: %s", string(secret.Data[selector.Key])))
			err = notify(service, string(secret.Data[selector.Key]), n.Information)

			if err != nil {
				n.Logger.Info(fmt.Sprintf("Failed to send notifications. %+v", err))
			} else {
				n.Logger.Info("Sent notification")
			}
		}
	}
}

func getStatusName(status Status) string {
	switch status {
	case StatusSuccess:
		return "Success"
	case StatusError:
		return "Error"
	default:
		return "Undefined"
	}
}

func getStatusColor(status Status, service Service) StatusColor {
	switch service.(type) {
	case Slack:
		switch status {
		case StatusSuccess:
			return "good"
		case StatusError:
			return "danger"
		default:
			return "#c8c8c8"
		}
	case Teams:
		switch status {
		case StatusSuccess:
			return "54A254"
		case StatusError:
			return "E81123"
		default:
			return "C8C8C8"
		}
	case Mailgun:
		switch status {
		case StatusSuccess:
			return "green"
		case StatusError:
			return "red"
		default:
			return "gray"
		}
	default:
		return "#c8c8c8"
	}
}

func notify(service Service, secret string, i *Information) error {
	var err error
	switch svc := service.(type) {
	case Slack:
		err = svc.Send(secret, i)
	case Teams:
		err = svc.Send(secret, i)
	case Mailgun:
		err = svc.Send(secret, i)
	}

	return err
}
