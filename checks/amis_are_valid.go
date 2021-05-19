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

type AMIsAreValid struct {
	ec2Svc                   *ec2.EC2
	instancesWithInvalidAMIs map[string]string
	errors                   *multierror.Error

	errorsMut sync.Mutex
}

func NewAMIsAreValidCheck(awsSession *session.Session) *AMIsAreValid {
	return &AMIsAreValid{
		ec2Svc:                   ec2.New(awsSession),
		instancesWithInvalidAMIs: map[string]string{},
	}
}

func (a *AMIsAreValid) Name() string {
	return "instances-use-valid-amis"
}

func (a *AMIsAreValid) Do() ([]Notice, error) {
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
			a.checkInstanceAMI(instance)
		}(instance)

	}
	wg.Wait()

	if a.errors != nil {
		return nil, a.errors
	}

	if len(a.instancesWithInvalidAMIs) == 0 {
		return nil, nil
	}

	return []Notice{{
		Title:       a.NoticeTitle(),
		Description: a.NoticeDescription(),
		Level:       NoticeLevelWarning,
	}}, nil
}

func (a *AMIsAreValid) checkInstanceAMI(instance *ec2.Instance) {
	imageOutput, err := a.ec2Svc.DescribeImages(&ec2.DescribeImagesInput{ImageIds: []*string{instance.ImageId}})
	if err != nil {
		a.errorsMut.Lock()
		defer a.errorsMut.Unlock()

		multierror.Append(a.errors, errors.Wrapf(err, "describing AMI image for instance %s", *instance.InstanceId))
		return
	}

	for _, image := range imageOutput.Images {
		if !isValidAMI(*image.Name) {
			a.instancesWithInvalidAMIs[*instance.InstanceId] = *image.Name
		}
	}
}

func (a *AMIsAreValid) NoticeTitle() string {
	return fmt.Sprintf("Found %d instances with invalid AMIs", len(a.instancesWithInvalidAMIs))
}

func (a *AMIsAreValid) NoticeDescription() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("We allow AMIs in the %v families. I found the following instances using invalid AMIs, not part of those families\n", validAMIFlavours))
	for instanceID, amiName := range a.instancesWithInvalidAMIs {
		b.WriteString(fmt.Sprintf("Instance %s was using AMI: %s\n", instanceID, amiName))
	}

	return b.String()
}

var validAMIFlavours = []string{"amzn2-ami-ecs-hvm-2.0", "ubuntu-bionic-18.04", "ubuntu-focal-20.04"}

func isValidAMI(amiName string) bool {
	for _, flavour := range validAMIFlavours {
		if strings.Contains(strings.ToLower(amiName), flavour) {
			return true
		}
	}
	return false
}
