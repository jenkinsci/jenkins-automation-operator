package notifier

import (
	"bytes"
	"encoding/json"
	"net/http"
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
func (t Teams) Send(secret string, i *Information) error {
	err := i.Error
	var errMessage string

	if err != nil {
		errMessage = err.Error()
	} else {
		errMessage = noErrorMessage
	}

	msg, err := json.Marshal(TeamsMessage{
		Type:       "MessageCard",
		Context:    "https://schema.org/extensions",
		ThemeColor: getStatusColor(i.Status, t),
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
						Name:  statusFieldName,
						Value: getStatusName(i.Status),
					},
				},
				Text: errMessage,
			},
		},
	})

	if err != nil {
		return err
	}

	request, err := http.NewRequest("POST", secret, bytes.NewBuffer(msg))
	if err != nil {
		return err
	}

	resp, err := client.Do(request)
	if err != nil {
		return err
	}

	err = resp.Body.Close()
	if err != nil {
		return err
	}

	return nil
}
