package notifier

import (
	"context"
	"fmt"
	"github.com/mailgun/mailgun-go/v3"
	"time"
)

// Mailgun is service for sending emails
type Mailgun struct {
	Domain    string
	Recipient string
	From      string
}

// Send is function for sending directly to API
func (m Mailgun) Send(secret string, i *Information) error {
	mg := mailgun.NewMailgun(m.Domain, secret)

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

	content = fmt.Sprintf(content, getStatusColor(i.Status, m), i.CrName, i.ConfigurationType, getStatusColor(i.Status, m), getStatusName(i.Status))

	msg := mg.NewMessage(fmt.Sprintf("Jenkins Operator Notifier <%s>", m.From), "Jenkins Operator Status", "", m.Recipient)
	msg.SetHtml(content)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	_, _, err := mg.Send(ctx, msg)

	if err != nil {
		return err
	}

	return nil
}
