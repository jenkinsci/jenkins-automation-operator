package notifier

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTeams_Send(t *testing.T) {
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
		var message TeamsMessage
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&message)

		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, message.Title, titleText)
		assert.Equal(t, message.ThemeColor, getStatusColor(i.LogLevel, Teams{}))

		mainSection := message.Sections[0]

		assert.Equal(t, mainSection.Text, i.Message)

		for _, fact := range mainSection.Facts {
			switch fact.Name {
			case configurationTypeFieldName:
				assert.Equal(t, fact.Value, i.ConfigurationType)
			case crNameFieldName:
				assert.Equal(t, fact.Value, i.CrName)
			case messageFieldName:
				assert.Equal(t, fact.Value, i.Message)
			case loggingLevelFieldName:
				assert.Equal(t, fact.Value, string(i.LogLevel))
			}
		}
	}))

	teams := Teams{apiURL: server.URL}

	defer server.Close()
	assert.NoError(t, teams.Send(notification))
}
