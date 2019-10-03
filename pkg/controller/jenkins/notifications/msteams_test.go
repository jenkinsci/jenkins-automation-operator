package notifications

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestTeams_Send(t *testing.T) {
	fakeClient := fake.NewFakeClient()
	testURLSelectorKeyName := "test-url-selector"
	testSecretName := "test-secret"

	event := Event{
		Jenkins: v1alpha2.Jenkins{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testCrName,
				Namespace: testNamespace,
			},
		},
		Phase:           testPhase,
		Message:         testMessage,
		MessagesVerbose: testMessageVerbose,
		LogLevel:        testLoggingLevel,
	}
	teams := Teams{k8sClient: fakeClient}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var message TeamsMessage
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&message)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, message.Title, notificationTitle(event))
		assert.Equal(t, message.ThemeColor, teams.getStatusColor(event.LogLevel))

		mainSection := message.Sections[0]

		assert.Equal(t, mainSection.Text, event.Message)

		for _, fact := range mainSection.Facts {
			switch fact.Name {
			case phaseFieldName:
				assert.Equal(t, fact.Value, string(event.Phase))
			case crNameFieldName:
				assert.Equal(t, fact.Value, event.Jenkins.Name)
			case messageFieldName:
				assert.Equal(t, fact.Value, event.Message)
			case loggingLevelFieldName:
				assert.Equal(t, fact.Value, string(event.LogLevel))
			case namespaceFieldName:
				assert.Equal(t, fact.Value, event.Jenkins.Namespace)
			default:
				t.Errorf("Found unexpected '%+v' fact", fact)
			}
		}
	}))

	defer server.Close()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: testNamespace,
		},

		Data: map[string][]byte{
			testURLSelectorKeyName: []byte(server.URL),
		},
	}

	err := fakeClient.Create(context.TODO(), secret)
	assert.NoError(t, err)

	err = teams.Send(event, v1alpha2.Notification{
		Teams: &v1alpha2.MicrosoftTeams{
			WebHookURLSecretKeySelector: v1alpha2.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: testSecretName,
				},
				Key: testURLSelectorKeyName,
			},
		},
	})
	assert.NoError(t, err)
}
