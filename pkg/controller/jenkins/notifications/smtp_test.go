package notifications

import (
	"context"
	"errors"
	"fmt"
	"github.com/emersion/go-smtp"
	"io"
	"io/ioutil"
	"log"
	"mime/quotedprintable"
	"net"
	"regexp"
	"testing"
	"time"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testSMTPUsername = "username"
	testSMTPPassword = "password"

	testSMTPPort = 1025
)

var smtpEvent = Event{
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
	re := regexp.MustCompile(`\t+<tr>\n\t+<td><b>(.*):</b></td>\n\t+<td>(.*)</td>\n\t+</tr>`)

	b, err := ioutil.ReadAll(quotedprintable.NewReader(r))
	if err != nil {
		return err
	}

	fmt.Println(string(b))

	res := re.FindAllStringSubmatch(string(b), -1)

	if smtpEvent.Jenkins.Name == res[0][1] {
		return fmt.Errorf("jenkins CR not identical: %s, expected: %s", res[0][1], smtpEvent.Jenkins.Name)
	} else if string(smtpEvent.Phase) == res[1][1] {
		return fmt.Errorf("phase not identical: %s, expected: %s", res[1][1], smtpEvent.Phase)
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

	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", testSMTPPort))
	assert.NoError(t, err)

	go func() {
		err := s.Serve(l)
		assert.NoError(t, err)
	}()

	err = smtpClient.Send(smtpEvent, v1alpha2.Notification{
		SMTP: &v1alpha2.SMTP{
			Server:                "localhost",
			From:                  "test@localhost",
			To:                    "test@localhost",
			TLSInsecureSkipVerify: true,
			Port:                  testSMTPPort,
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
