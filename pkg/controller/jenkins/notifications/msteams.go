package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Teams is a Microsoft MicrosoftTeams notification service
type Teams struct {
	k8sClient k8sclient.Client
}

// TeamsMessage is representation of json message structure
type TeamsMessage struct {
	Type       string         `json:"@type"`
	Context    string         `json:"@context"`
	ThemeColor StatusColor    `json:"themeColor"`
	Title      string         `json:"title"`
	Sections   []TeamsSection `json:"sections"`
	Summary    string         `json:"summary"`
}

// TeamsSection is MS Teams message section
type TeamsSection struct {
	Facts []TeamsFact `json:"facts"`
	Text  string      `json:"text"`
}

// TeamsFact is field where we can put content
type TeamsFact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (t Teams) getStatusColor(logLevel v1alpha2.NotificationLogLevel) StatusColor {
	switch logLevel {
	case v1alpha2.NotificationLogLevelInfo:
		return "439FE0"
	case v1alpha2.NotificationLogLevelWarning:
		return "E81123"
	default:
		return "C8C8C8"
	}
}

// Send is function for sending directly to API
func (t Teams) Send(event Event, config v1alpha2.Notification) error {
	secret := &corev1.Secret{}

	selector := config.Teams.WebHookURLSecretKeySelector

	err := t.k8sClient.Get(context.TODO(), types.NamespacedName{Name: selector.Name, Namespace: event.Jenkins.Namespace}, secret)
	if err != nil {
		return errors.WithStack(err)
	}

	secretValue := string(secret.Data[selector.Key])
	if secretValue == "" {
		return errors.Errorf("Microsoft Teams WebHook URL is empty in secret '%s/%s[%s]", event.Jenkins.Namespace, selector.Name, selector.Key)
	}

	tm := &TeamsMessage{
		Type:       "MessageCard",
		Context:    "https://schema.org/extensions",
		ThemeColor: t.getStatusColor(event.LogLevel),
		Sections: []TeamsSection{
			{
				Facts: []TeamsFact{
					{
						Name:  crNameFieldName,
						Value: event.Jenkins.Name,
					},
					{
						Name:  namespaceFieldName,
						Value: event.Jenkins.Namespace,
					},
				},
				Text: event.Message,
			},
		},
		Summary: event.Message,
	}

	tm.Title = notificationTitle(event)

	if config.Verbose {
		message := event.Message
		for _, msg := range event.MessagesVerbose {
			message = message + "\n\n - " + msg
		}
		tm.Sections[0].Text += message
		tm.Summary = message
	}

	if event.Phase != PhaseUnknown {
		tm.Sections[0].Facts = append(tm.Sections[0].Facts, TeamsFact{
			Name:  phaseFieldName,
			Value: string(event.Phase),
		})
	}

	msg, err := json.Marshal(tm)
	if err != nil {
		return errors.WithStack(err)
	}

	request, err := http.NewRequest("POST", secretValue, bytes.NewBuffer(msg))
	if err != nil {
		return errors.WithStack(err)
	}

	resp, err := client.Do(request)
	if err != nil {
		return errors.WithStack(err)
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("Invalid response from server: %s", resp.Status))
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}
