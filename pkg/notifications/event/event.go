package event

import (
	"github.com/jenkinsci/jenkins-automation-operator/api/v1alpha2"
	"github.com/jenkinsci/jenkins-automation-operator/pkg/notifications/reason"
)

// Phase defines the context where notification has been generated: base or user.
type Controller string

// Event contains event details which will be sent as a notification.
type Event struct {
	Jenkins    v1alpha2.Jenkins
	Controller Controller
	Level      v1alpha2.NotificationLevel
	Reason     reason.Reason
}

const (
	// JenkinsController points to the events sent by the Jenkins Controller
	JenkinsController Controller = "jenkins"
	BackupController  Controller = "backup"
)
