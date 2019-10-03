package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"

	"github.com/mailgun/mailgun-go/v3"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const content = `
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN"
        "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html>
<head></head>
<body>
		<h1 style="background-color: %s; color: white; padding: 3px 10px;">%s</h1>
		<h3>%s</h3>
		<table>
			<tr>
				<td><b>CR name:</b></td>
				<td>%s</td>
			</tr>
			<tr>
				<td><b>Configuration type:</b></td>
				<td>%s</td>
			</tr>
		</table>
		<h6 style="font-size: 11px; color: grey; margin-top: 15px;">Powered by Jenkins Operator <3</h6>
</body>
</html>`

// MailGun is a sending emails notification service
type MailGun struct {
	k8sClient k8sclient.Client
}

func (m MailGun) getStatusColor(logLevel v1alpha2.NotificationLogLevel) StatusColor {
	switch logLevel {
	case v1alpha2.NotificationLogLevelInfo:
		return "blue"
	case v1alpha2.NotificationLogLevelWarning:
		return "red"
	default:
		return "gray"
	}
}

// Send is function for sending directly to API
func (m MailGun) Send(event Event, config v1alpha2.Notification) error {
	secret := &corev1.Secret{}

	selector := config.Mailgun.APIKeySecretKeySelector

	err := m.k8sClient.Get(context.TODO(), types.NamespacedName{Name: selector.Name, Namespace: event.Jenkins.Namespace}, secret)
	if err != nil {
		return errors.WithStack(err)
	}

	secretValue := string(secret.Data[selector.Key])
	if secretValue == "" {
		return errors.Errorf("Mailgun API is empty in secret '%s/%s[%s]", event.Jenkins.Namespace, selector.Name, selector.Key)
	}

	mg := mailgun.NewMailgun(config.Mailgun.Domain, secretValue)

	var statusMessage string

	if config.Verbose {
		message := event.Message + "<ul>"
		for _, msg := range event.MessagesVerbose {
			message = message + "<li>" + msg + "</li>"
		}
		message = message + "</ul>"
		statusMessage = message
	} else {
		statusMessage = event.Message
	}

	htmlMessage := fmt.Sprintf(content, m.getStatusColor(event.LogLevel), notificationTitle(event), statusMessage, event.Jenkins.Name, event.ConfigurationType)

	msg := mg.NewMessage(fmt.Sprintf("Jenkins Operator Notifier <%s>", config.Mailgun.From), notificationTitle(event), "", config.Mailgun.Recipient)
	msg.SetHtml(htmlMessage)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	_, _, err = mg.Send(ctx, msg)

	return err
}
