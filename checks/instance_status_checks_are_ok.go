package checks

import (
	"fmt"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/go-multierror"
	"github.com/moskyb/infrastructure-integrity-check/ec2_helper"
	"github.com/pkg/errors"
)

type InstanceStatusChecksAreOK struct {
	ec2Svc                     *ec2.EC2
	instanceFailedStatusChecks map[string][]*ec2.InstanceStatusDetails
	errors                     *multierror.Error

	errorsMut sync.Mutex
}

func NewInstanceStatusChecksAreOKCheck(awsSession *session.Session) *InstanceStatusChecksAreOK {
	return &InstanceStatusChecksAreOK{
		ec2Svc:                     ec2.New(awsSession),
		instanceFailedStatusChecks: map[string][]*ec2.InstanceStatusDetails{},
	}
}

func (a *InstanceStatusChecksAreOK) Name() string {
	return "instances-are-passing-status-checks"
}

func (a *InstanceStatusChecksAreOK) Do() ([]Notice, error) {
	reservations, err := ec2_helper.DescribeAllInstances(a.ec2Svc, &ec2.DescribeInstancesInput{})
	if err != nil {
		return nil, errors.Wrap(err, "describing all ec2 instances")
	}

	instances := ec2_helper.InstancesForReservations(reservations)

	wg := sync.WaitGroup{}
	for _, instance := range instances {
		wg.Add(1)
		go func(instance *ec2.Instance) {
			defer wg.Done()
			a.checkInstanceStatus(instance)
		}(instance)

	}
	wg.Wait()

	if a.errors != nil {
		return nil, a.errors
	}

	if len(a.instanceFailedStatusChecks) == 0 {
		return nil, nil
	}

	return []Notice{{
		Title:       a.NoticeTitle(),
		Description: a.NoticeDescription(),
		Level:       NoticeLevelWarning,
	}}, nil
}

func (a *InstanceStatusChecksAreOK) checkInstanceStatus(instance *ec2.Instance) {
	instanceStatusOut, err := a.ec2Svc.DescribeInstanceStatus(&ec2.DescribeInstanceStatusInput{
		InstanceIds: []*string{instance.InstanceId},
	})
	if err != nil {
		a.errorsMut.Lock()
		defer a.errorsMut.Unlock()

		multierror.Append(a.errors, errors.Wrapf(err, "describing AMI image for instance %s", *instance.InstanceId))
		return
	}

	for _, status := range instanceStatusOut.InstanceStatuses {
		failedStatuses := []*ec2.InstanceStatusDetails{}
		if *status.InstanceStatus.Status != "ok" {
			failedStatuses = append(failedStatuses, status.InstanceStatus.Details...)
		}

		if *status.SystemStatus.Status != "ok" {
			failedStatuses = append(failedStatuses, status.SystemStatus.Details...)
		}

		if len(failedStatuses) > 0 {
			a.instanceFailedStatusChecks[*instance.InstanceId] = failedStatuses
		}
	}
}

func (a *InstanceStatusChecksAreOK) NoticeTitle() string {
	return fmt.Sprintf("Found %d instances with failing status checks", len(a.instanceFailedStatusChecks))
}

func (a *InstanceStatusChecksAreOK) NoticeDescription() string {
	var b strings.Builder
	for instanceID, failedStatusChecks := range a.instanceFailedStatusChecks {
		b.WriteString(fmt.Sprintf("Instance `%s` is failing %d status checks:\n", instanceID, len(failedStatusChecks)))
		for _, statusCheck := range failedStatusChecks {
			b.WriteString(fmt.Sprintf(" - %s is in state: %s\n", *statusCheck.Name, *statusCheck.Status))
		}
	}

	return b.String()
}
