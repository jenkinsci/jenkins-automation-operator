package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Teams is Microsoft Teams Service
type Teams struct{}

// TeamsMessage is representation of json message structure
type TeamsMessage struct {
	Type       string         `json:"@type"`
	Context    string         `json:"@context"`
	ThemeColor StatusColor    `json:"themeColor"`
	Title      string         `json:"title"`
	Sections   []TeamsSection `json:"sections"`
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

// Send is function for sending directly to API
func (t Teams) Send(n *Notification, config v1alpha2.Notification) error {
	var selector v1alpha2.SecretKeySelector
	secret := &corev1.Secret{}
	i := n.Information

	selector = config.Teams.URLSecretKeySelector

	err := n.K8sClient.Get(context.TODO(), types.NamespacedName{Name: selector.Name, Namespace: n.Jenkins.Namespace}, secret)
	if err != nil {
		n.Logger.V(log.VWarn).Info(fmt.Sprintf("Failed to get secret with name `%s`. %+v", selector.Name, err))
	}

	msg, err := json.Marshal(TeamsMessage{
		Type:       "MessageCard",
		Context:    "https://schema.org/extensions",
		ThemeColor: getStatusColor(i.LogLevel, t),
		Title:      titleText,
		Sections: []TeamsSection{
			{
				Facts: []TeamsFact{
					{
						Name:  crNameFieldName,
						Value: i.CrName,
					},
					{
						Name:  configurationTypeFieldName,
						Value: i.ConfigurationType,
					},
					{
						Name:  loggingLevelFieldName,
						Value: string(i.LogLevel),
					},
					{
						Name:  namespaceFieldName,
						Value: i.Namespace,
					},
				},
				Text: i.Message,
			},
		},
	})

	secretValue := string(secret.Data[selector.Key])
	if secretValue == "" {
		return fmt.Errorf("SecretValue %s is empty", selector.Name)
	}

	if err != nil {
		return err
	}

	request, err := http.NewRequest("POST", secretValue, bytes.NewBuffer(msg))
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
