package notifications

import (
	"context"
	"errors"
	"fmt"
	"github.com/emersion/go-smtp"
	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testSMTPUsername = "username"
	testSMTPPassword = "password"

	testSMTPPort = 1025
)

type testServer struct{}

// Login handles a login command with username and password.
func (bkd *testServer) Login(state *smtp.ConnectionState, username, password string) (smtp.Session, error) {
	if username != testSMTPUsername || password != testSMTPPassword {
		return nil, errors.New("invalid username or password")
	}
	return &testSession{}, nil
}

// AnonymousLogin requires clients to authenticate using SMTP AUTH before sending emails
func (bkd *testServer) AnonymousLogin(state *smtp.ConnectionState) (smtp.Session, error) {
	return nil, smtp.ErrAuthRequired
}

// A Session is returned after successful login.
type testSession struct{}

func (s *testSession) Mail(from string) error {
	log.Println("Mail from:", from)
	return nil
}

func (s *testSession) Rcpt(to string) error {
	log.Println("Rcpt to:", to)
	return nil
}

func (s *testSession) Data(r io.Reader) error {
	if b, err := ioutil.ReadAll(r); err != nil {
		return err
	} else {
		log.Println("Data:", string(b))
	}
	return nil
}

func (s *testSession) Reset() {}

func (s *testSession) Logout() error {
	return nil
}


func TestSMTP_Send(t *testing.T) {
	fakeClient := fake.NewFakeClient()
	testUsernameSelectorKeyName := "test-username-selector"
	testPasswordSelectorKeyName := "test-password-selector"
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

	smtpClient := SMTP{k8sClient: fakeClient}

	ts := &testServer{}

	// Create fake SMTP server

	s := smtp.NewServer(ts)

	s.Addr = fmt.Sprintf(":%d", testSMTPPort)
	s.Domain = "localhost"
	s.ReadTimeout = 10 * time.Second
	s.WriteTimeout = 10 * time.Second
	s.MaxMessageBytes = 1024 * 1024
	s.MaxRecipients = 50
	s.AllowInsecureAuth = true

	// Create secrets
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: testNamespace,
		},

		Data: map[string][]byte{
			testUsernameSelectorKeyName: []byte(testSMTPUsername),
			testPasswordSelectorKeyName: []byte(testSMTPPassword),
		},
	}

	err := fakeClient.Create(context.TODO(), secret)
	assert.NoError(t, err)

	go func() {
		if err := s.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	err = smtpClient.Send(event, v1alpha2.Notification{
		SMTP: &v1alpha2.SMTP{
			Server: "localhost",
			From: "test@localhost",
			To: "test@localhost",
			TLSInsecureSkipVerify: true,
			Port: testSMTPPort,
			UsernameSecretKeySelector: v1alpha2.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: testSecretName,
				},
				Key: testUsernameSelectorKeyName,
			},
			PasswordSecretKeySelector: v1alpha2.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: testSecretName,
				},
				Key: testPasswordSelectorKeyName,
			},
		},
	})

	assert.NoError(t, err)
}
