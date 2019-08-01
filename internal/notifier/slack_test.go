package notifier

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlack_Send(t *testing.T) {
	i := &Information{
		ConfigurationType: testConfigurationType,
		CrName:            testCrName,
		Message:           testMessage,
		MessageVerbose:    testMessageVerbose,
		Namespace:         testNamespace,
		LogLevel:          testLoggingLevel,
	}

	notification := &Notification{
		Information: i,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var message SlackMessage
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&message)

		if err != nil {
			t.Fatal(err)
		}

		mainAttachment := message.Attachments[0]

		assert.Equal(t, mainAttachment.Text, titleText)

		for _, field := range mainAttachment.Fields {
			switch field.Title {
			case configurationTypeFieldName:
				assert.Equal(t, field.Value, i.ConfigurationType)
			case crNameFieldName:
				assert.Equal(t, field.Value, i.CrName)
			case messageFieldName:
				assert.Equal(t, field.Value, i.Message)
			case loggingLevelFieldName:
				assert.Equal(t, field.Value, string(i.LogLevel))
			}
		}

		assert.Equal(t, mainAttachment.Footer, footerContent)
		assert.Equal(t, mainAttachment.Color, getStatusColor(i.LogLevel, Slack{}))
	}))

	defer server.Close()

	slack := Slack{apiURL: server.URL}

	assert.NoError(t, slack.Send(notification))
}
