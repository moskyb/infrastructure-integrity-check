package checks

import (
	"fmt"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/go-multierror"
	"github.com/moskyb/infrastructure-integrity-check/ec2_helper"
	"github.com/pkg/errors"
)

type InstancesInOtherRegions struct {
	awsSession        *session.Session
	otherRegionCounts map[string]int
	errors            *multierror.Error
}

func NewInstancesInOtherRegionsCheck(sess *session.Session) InstancesInOtherRegions {
	return InstancesInOtherRegions{
		awsSession:        sess,
		otherRegionCounts: map[string]int{},
	}
}

func (i InstancesInOtherRegions) Name() string {
	return "instances-run-in-ap-southeast-2"
}

func (i InstancesInOtherRegions) Do() ([]Notice, error) {
	ec2Svc := ec2.New(i.awsSession)
	regionsOutput, err := ec2Svc.DescribeRegions(&ec2.DescribeRegionsInput{AllRegions: aws.Bool(true)})
	if err != nil {
		return nil, errors.Wrap(err, "describing aws regions")
	}

	wg := sync.WaitGroup{}
	for _, region := range regionsOutput.Regions {
		if *region.RegionName == "ap-southeast-2" {
			continue
		}
		wg.Add(1)

		go func(region string) {
			defer wg.Done()
			ec2ForThisRegion := ec2.New(i.awsSession, aws.NewConfig().WithRegion(region))
			reservations, err := ec2_helper.DescribeAllInstances(ec2ForThisRegion, &ec2.DescribeInstancesInput{})
			if err != nil {
				multierror.Append(i.errors, errors.Wrapf(err, "describing instances for region %s", region))
				return
			}

			instances := ec2_helper.InstancesForReservations(reservations)
			if len(instances) > 0 {
				i.otherRegionCounts[region] += len(instances)
			}
		}(*region.RegionName)
	}
	wg.Wait()

	if len(i.otherRegionCounts) == 0 {
		return nil, i.errors.ErrorOrNil()
	}

	return []Notice{{
		Title:       i.NoticeTitle(),
		Description: i.NoticeDescription(),
		Level:       NoticeLevelError,
	}}, i.errors.ErrorOrNil()
}

func (i InstancesInOtherRegions) NoticeTitle() string {
	totalInstancesInOtherRegions := 0
	for _, count := range i.otherRegionCounts {
		totalInstancesInOtherRegions += count
	}
	return fmt.Sprintf("Found %d instances in regions outside ap-southeast-2", totalInstancesInOtherRegions)
}

func (i InstancesInOtherRegions) NoticeDescription() string {
	var b strings.Builder
	b.WriteString("I found instances in the following regions:\n")
	for region, count := range i.otherRegionCounts {
		b.WriteString(fmt.Sprintf("%d instances in the %s region\n", count, region))
	}
	return b.String()
}
