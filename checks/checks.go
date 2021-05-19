package checks

import "github.com/aws/aws-sdk-go/aws/session"

const (
	NoticeLevelInfo    = "INFO"
	NoticeLevelWarning = "WARNING"
	NoticeLevelError   = "ERROR"
)

type Check interface {
	Do() ([]Notice, error)
	Name() string
}

type Notice struct {
	Title                string
	Description          string
	Level                string
	OffendingInstanceIDs []string
}

func All(sess *session.Session) []Check {
	return []Check{
		NewInstancesInOtherRegionsCheck(sess),
		NewAMIsAreValidCheck(sess),
		NewInstanceStatusChecksAreOKCheck(sess),
	}
}
