package notifier

import (
	"context"
	"fmt"
	"time"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"

	"github.com/mailgun/mailgun-go/v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Mailgun is service for sending emails
type Mailgun struct {
	Domain    string
	Recipient string
	From      string
}

// Send is function for sending directly to API
func (m Mailgun) Send(n *Notification) error {
	var selector v1alpha2.SecretKeySelector
	secret := &corev1.Secret{}

	i := n.Information

	err := n.K8sClient.Get(context.TODO(), types.NamespacedName{Name: selector.Name, Namespace: n.Jenkins.Namespace}, secret)
	if err != nil {
		n.Logger.V(log.VWarn).Info(fmt.Sprintf("Failed to get secret with name `%s`. %+v", selector.Name, err))
		return err
	}

	mg := mailgun.NewMailgun(m.Domain, secret.StringData[selector.Name])

	content := `
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
</html>
	`

	content = fmt.Sprintf(content, getStatusColor(i.LogLevel, m), i.CrName, i.ConfigurationType, getStatusColor(i.LogLevel, m), string(i.LogLevel))

	msg := mg.NewMessage(fmt.Sprintf("Jenkins Operator Notifier <%s>", m.From), "Jenkins Operator Status", "", m.Recipient)
	msg.SetHtml(content)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	_, _, err = mg.Send(ctx, msg)

	if err != nil {
		return err
	}

	return nil
}
