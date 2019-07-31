package notifier

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTeams_Send(t *testing.T) {
	teams := Teams{}

	i := &Information{
		ConfigurationType: testConfigurationType,
		CrName:            testCrName,
		Status:            testStatus,
		Error:             testError,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var message TeamsMessage
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&message)

		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, message.Title, titleText)
		assert.Equal(t, message.ThemeColor, getStatusColor(i.Status, teams))

		mainSection := message.Sections[0]

		assert.Equal(t, mainSection.Text, noErrorMessage)

		for _, fact := range mainSection.Facts {
			switch fact.Name {
			case configurationTypeFieldName:
				if fact.Value != i.ConfigurationType {
					t.Fatalf("%s is not equal! Must be %s", configurationTypeFieldName, i.ConfigurationType)
				}
			case crNameFieldName:
				if fact.Value != i.CrName {
					t.Fatalf("%s is not equal! Must be %s", crNameFieldName, i.CrName)
				}
			case statusFieldName:
				if fact.Value != getStatusName(i.Status) {
					t.Fatalf("%s is not equal! Must be %s", statusFieldName, getStatusName(i.Status))
				}
			}
		}
	}))

	defer server.Close()
	if err := teams.Send(server.URL, i); err != nil {
		t.Fatal(err)
	}
}
