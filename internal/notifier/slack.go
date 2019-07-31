package notifier

import (
	"bytes"
	"encoding/json"
	"net/http"
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
func (s Slack) Send(secret string, i *Information) error {
	err := i.Error
	var errMessage string

	if err != nil {
		errMessage = err.Error()
	} else {
		errMessage = noErrorMessage
	}

	slackMessage, err := json.Marshal(SlackMessage{
		Attachments: []SlackAttachment{
			{
				Fallback: "",
				Color:    getStatusColor(i.Status, s),
				Text:     titleText,
				Fields: []SlackField{
					{
						Title: statusMessageFieldName,
						Value: errMessage,
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
				},
				Footer: footerContent,
			},
		},
	})

	if err != nil {
		return err
	}

	request, err := http.NewRequest("POST", secret, bytes.NewBuffer(slackMessage))
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
