package notifier

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

	i := Information{
		ConfigurationType: testConfigurationType,
		CrName:            testCrName,
		Message:           testMessage,
		MessageVerbose:    testMessageVerbose,
		Namespace:         testNamespace,
		LogLevel:          testLoggingLevel,
	}

	notification := &Notification{
		K8sClient:   fakeClient,
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
			case namespaceFieldName:
				assert.Equal(t, field.Value, i.Namespace)
			default:
				t.Fail()
			}
		}

		assert.Equal(t, mainAttachment.Footer, footerContent)
		assert.Equal(t, mainAttachment.Color, getStatusColor(i.LogLevel, Slack{}))
	}))

	defer server.Close()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: testSecretName,
		},

		Data: map[string][]byte{
			testURLSelectorKeyName: []byte(server.URL),
		},
	}

	err := notification.K8sClient.Create(context.TODO(), secret)
	assert.NoError(t, err)

	slack := Slack{}

	assert.NoError(t, slack.Send(notification, v1alpha2.Notification{
		Slack: v1alpha2.Slack{
			URLSecretKeySelector: v1alpha2.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: testSecretName,
				},
				Key: testURLSelectorKeyName,
			},
		},
	}))
}
