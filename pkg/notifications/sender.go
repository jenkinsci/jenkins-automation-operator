package notifications

import (
	"reflect"
	"strings"

	"github.com/jenkinsci/jenkins-automation-operator/api/v1alpha2"
	k8sevent "github.com/jenkinsci/jenkins-automation-operator/pkg/event"
	"github.com/jenkinsci/jenkins-automation-operator/pkg/log"
	"github.com/jenkinsci/jenkins-automation-operator/pkg/notifications/event"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Provider is the communication service handler.
type Provider interface {
	Send(event event.Event) error
}

// Listen listens for incoming events and send it as notifications.
func Listen(events chan event.Event, k8sEvent k8sevent.Recorder, k8sClient k8sclient.Client) {
	for e := range events {
		logger := log.Log.WithValues("cr", e.Jenkins.Name)

		if !e.Reason.HasMessages() {
			logger.V(log.VWarn).Info("Reason has no messages, this should not happen")

			continue // skip empty messages
		}

		k8sEvent.Emit(&e.Jenkins,
			eventLevelToKubernetesEventType(e.Level),
			k8sevent.Reason(reflect.TypeOf(e.Reason).Name()),
			strings.Join(e.Reason.Short(), "; "),
		)
	}
}

func eventLevelToKubernetesEventType(level v1alpha2.NotificationLevel) k8sevent.Type {
	switch level {
	case v1alpha2.NotificationLevelWarning:
		return k8sevent.TypeWarning
	case v1alpha2.NotificationLevelInfo:
		return k8sevent.TypeNormal
	default:
		return k8sevent.TypeNormal
	}
}
