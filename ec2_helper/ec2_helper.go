package ec2_helper

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func DescribeAllInstances(ec2Svc *ec2.EC2, input *ec2.DescribeInstancesInput) ([]*ec2.Reservation, error) {
	input.MaxResults = aws.Int64(1000)
	output, err := ec2Svc.DescribeInstances(input)
	if err != nil {
		return nil, err
	}

	if output.NextToken == nil {
		return output.Reservations, nil
	}

	reservations := output.Reservations

	input.NextToken = output.NextToken
	nextReservations, err := DescribeAllInstances(ec2Svc, input)
	if err != nil {
		return nil, err
	}

	reservations = append(reservations, nextReservations...)
	return reservations, nil
}

func InstancesForReservations(reservations []*ec2.Reservation) []*ec2.Instance {
	instances := make([]*ec2.Instance, 0, len(reservations))

	for _, reservation := range reservations {
		instances = append(instances, reservation.Instances...)
	}

	return instances
}
