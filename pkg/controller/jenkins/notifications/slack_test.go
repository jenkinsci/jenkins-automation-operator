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

func TestSlack_Send(t *testing.T) {
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
	slack := Slack{k8sClient: fakeClient}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var message SlackMessage
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&message)

		if err != nil {
			t.Fatal(err)
		}

		mainAttachment := message.Attachments[0]

		assert.Equal(t, mainAttachment.Title, notificationTitle(event))
		for _, field := range mainAttachment.Fields {
			switch field.Title {
			case phaseFieldName:
				assert.Equal(t, field.Value, string(event.Phase))
			case crNameFieldName:
				assert.Equal(t, field.Value, event.Jenkins.Name)
			case "":
				assert.Equal(t, field.Value, event.Message)
			case loggingLevelFieldName:
				assert.Equal(t, field.Value, string(event.LogLevel))
			case namespaceFieldName:
				assert.Equal(t, field.Value, event.Jenkins.Namespace)
			default:
				t.Errorf("Unexpected field %+v", field)
			}
		}

		assert.Equal(t, mainAttachment.Footer, "")
		assert.Equal(t, mainAttachment.Color, slack.getStatusColor(event.LogLevel))
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

	err = slack.Send(event, v1alpha2.Notification{
		Slack: &v1alpha2.Slack{
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
