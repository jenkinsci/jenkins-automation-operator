package notifier

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSlack_Send(t *testing.T) {
	slack := Slack{}

	i := &Information{
		ConfigurationType: testConfigurationType,
		CrName:            testCrName,
		Status:            testStatus,
		Error:             testError,
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
				if field.Value != i.ConfigurationType {
					t.Fatalf("%s is not equal! Must be %s", configurationTypeFieldName, i.ConfigurationType)
				}
			case crNameFieldName:
				if field.Value != i.CrName {
					t.Fatalf("%s is not equal! Must be %s", crNameFieldName, i.CrName)
				}
			case statusMessageFieldName:
				if field.Value != noErrorMessage {
					t.Fatalf("Error thrown but not expected")
				}
			}
		}

		assert.Equal(t, mainAttachment.Footer, footerContent)
		assert.Equal(t, mainAttachment.Color, getStatusColor(i.Status, slack))
	}))

	defer server.Close()

	if err := slack.Send(server.URL, i); err != nil {
		t.Fatal(err)
	}
}
