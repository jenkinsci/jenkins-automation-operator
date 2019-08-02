package notifier

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"time"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/mailgun/mailgun-go/v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const content = `
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN"
        "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html>
<head></head>
<body>
		<h1 style="background-color: %s; color: white; padding: 3px 10px;">Jenkins Operator Reconciled</h1>
		<h3>Failed to do something</h3>
		<table>
			<tr>
				<td><b>CR name:</b></td>
				<td>%s</td>
			</tr>
			<tr>
				<td><b>Configuration type:</b></td>
				<td>%s</td>
			</tr>
			<tr>
				<td><b>Status:</b></td>
				<td><b style="color: %s;">%s</b></td>
			</tr>
		</table>
		<h6 style="font-size: 11px; color: grey; margin-top: 15px;">Powered by Jenkins Operator <3</h6>
</body>
</html>`

// Mailgun is service for sending emails
type Mailgun struct{}

// Send is function for sending directly to API
func (m Mailgun) Send(n *Notification, config v1alpha2.Notification) error {
	secret := &corev1.Secret{}
	i := n.Information

	selector := config.Mailgun.APIKeySecretKeySelector

	err := n.K8sClient.Get(context.TODO(), types.NamespacedName{Name: selector.Name, Namespace: n.Jenkins.Namespace}, secret)
	if err != nil {
		return err
	}

	secretValue := string(secret.Data[selector.Name])
	if secretValue == "" {
		return errors.Errorf("SecretValue %s is empty", selector.Name)
	}

	mg := mailgun.NewMailgun(config.Mailgun.Domain, secretValue)

	htmlMessage := fmt.Sprintf(content, getStatusColor(i.LogLevel, m), i.CrName, i.ConfigurationType, getStatusColor(i.LogLevel, m), string(i.LogLevel))

	msg := mg.NewMessage(fmt.Sprintf("Jenkins Operator Notifier <%s>", config.Mailgun.From), "Jenkins Operator Status", "", config.Mailgun.Recipient)
	msg.SetHtml(htmlMessage)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	_, _, err = mg.Send(ctx, msg)

	return err
}
