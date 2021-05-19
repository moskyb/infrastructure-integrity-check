package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/moskyb/infrastructure-integrity-check/checker"
	"github.com/moskyb/infrastructure-integrity-check/checks"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.WithField("service", "infrastructure-integrity-lambda")
	sess, err := session.NewSession()
	if err != nil {
		logger.Fatal(errors.Wrap(err, "creating AWS session"))
	}

	checker := checker.NewChecker(logger, checks.All(sess))
	notices, err := checker.CheckAll()
	if err != nil {
		logger.Fatal(err)
	}

	for _, notice := range notices {
		fmt.Printf("NOTICE: %s\n\n", notice.Title)
		fmt.Println(notice.Description)
	}
}
