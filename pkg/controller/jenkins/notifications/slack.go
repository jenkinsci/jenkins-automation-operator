package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Slack is a Slack notification service
type Slack struct {
	k8sClient k8sclient.Client
}

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

func (s Slack) getStatusColor(logLevel v1alpha2.NotificationLogLevel) StatusColor {
	switch logLevel {
	case v1alpha2.NotificationLogLevelInfo:
		return "#439FE0"
	case v1alpha2.NotificationLogLevelWarning:
		return "danger"
	default:
		return "#c8c8c8"
	}
}

// Send is function for sending directly to API
func (s Slack) Send(event Event, config v1alpha2.Notification) error {
	secret := &corev1.Secret{}
	selector := config.Slack.WebHookURLSecretKeySelector

	err := s.k8sClient.Get(context.TODO(), types.NamespacedName{Name: selector.Name, Namespace: event.Jenkins.Namespace}, secret)
	if err != nil {
		return err
	}

	sm := &SlackMessage{
		Attachments: []SlackAttachment{
			{
				Fallback: "",
				Color:    s.getStatusColor(event.LogLevel),
				Fields: []SlackField{
					{
						Title: "",
						Value: event.Message,
						Short: false,
					},
					{
						Title: namespaceFieldName,
						Value: event.Jenkins.Namespace,
						Short: true,
					},
					{
						Title: crNameFieldName,
						Value: event.Jenkins.Name,
						Short: true,
					},
				},
			},
		},
	}

	mainAttachment := &sm.Attachments[0]

	mainAttachment.Title = notificationTitle(event)

	if config.Verbose {
		// TODO: or for title == message
		message := event.Message
		for _, msg := range event.MessagesVerbose {
			message = message + "\n - " + msg
		}
		mainAttachment.Fields[0].Value = message
	}

	if event.Phase != PhaseUnknown {
		mainAttachment.Fields = append(mainAttachment.Fields, SlackField{
			Title: phaseFieldName,
			Value: string(event.Phase),
			Short: true,
		})
	}

	slackMessage, err := json.Marshal(sm)
	if err != nil {
		return err
	}

	secretValue := string(secret.Data[selector.Key])
	if secretValue == "" {
		return errors.Errorf("Slack WebHook URL is empty in secret '%s/%s[%s]", event.Jenkins.Namespace, selector.Name, selector.Key)
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
