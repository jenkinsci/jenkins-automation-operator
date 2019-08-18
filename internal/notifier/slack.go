package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Slack is messaging service
type Slack struct{}

// SlackMessage is representation of json message
type SlackMessage struct {
	Text        string            `json:"text"`
	Attachments []SlackAttachment `json:"attachments"`
}

// SlackAttachment is representation of json attachment
type SlackAttachment struct {
	Fallback string       `json:"fallback"`
	Color    StatusColor  `json:"color"`
	Pretext  string       `json:"pretext"`
	Title    string       `json:"title"`
	Text     string       `json:"text"`
	Fields   []SlackField `json:"fields"`
	Footer   string       `json:"footer"`
}

// SlackField is representation of json field.
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// Send is function for sending directly to API
func (s Slack) Send(n *Notification, config v1alpha2.Notification) error {
	secret := &corev1.Secret{}
	i := n.Information

	selector := config.Slack.URLSecretKeySelector

	err := n.K8sClient.Get(context.TODO(), types.NamespacedName{Name: selector.Name, Namespace: n.Jenkins.Namespace}, secret)
	if err != nil {
		return err
	}

	slackMessage, err := json.Marshal(SlackMessage{
		Attachments: []SlackAttachment{
			{
				Fallback: "",
				Color:    getStatusColor(i.LogLevel, s),
				Text:     titleText,
				Fields: []SlackField{
					{
						Title: messageFieldName,
						Value: i.Message,
						Short: false,
					},
					{
						Title: crNameFieldName,
						Value: i.CrName,
						Short: true,
					},
					{
						Title: configurationTypeFieldName,
						Value: i.ConfigurationType,
						Short: true,
					},
					{
						Title: loggingLevelFieldName,
						Value: string(i.LogLevel),
						Short: true,
					},
					{
						Title: namespaceFieldName,
						Value: i.Namespace,
						Short: true,
					},
				},
				Footer: footerContent,
			},
		},
	})

	secretValue := string(secret.Data[selector.Key])
	if secretValue == "" {
		return errors.Errorf("SecretValue %s is empty", selector.Name)
	}

	if err != nil {
		return err
	}

	request, err := http.NewRequest("POST", secretValue, bytes.NewBuffer(slackMessage))
	if err != nil {
		return err
	}

	resp, err := client.Do(request)
	if err != nil {
		return err
	}

	defer func() { _ = resp.Body.Close() }()
	return nil
}
