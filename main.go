package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gitlab.com/sharesies/infrastructure-integrity-lambda/checker"
	"gitlab.com/sharesies/infrastructure-integrity-lambda/checks"
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
