package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// NewEC2
func NewEC2(accessKeyID, secretAccessKey string, region *string) (*ec2.EC2, error) {
	var err error
	cfg := aws.NewConfig().WithMaxRetries(10).WithCredentials(credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""))
	if region != nil {
		cfg = cfg.WithRegion(*region)
	}
	sess := session.New(cfg)
	ec2 := ec2.New(sess)
	return ec2, err
}

// LaunchAMI  launches an EC2 instance for a given AMI id
func LaunchAMI(accessKeyID, secretAccessKey, region, imageID, userData string, minCount, maxCount int64) (instanceIds []string, err error) {
	client, err := NewEC2(accessKeyID, secretAccessKey, stringOrNil(region))

	reservation, err := client.RunInstances(&ec2.RunInstancesInput{
		ImageId:  stringOrNil(imageID),
		MinCount: &minCount,
		MaxCount: &maxCount,
		UserData: stringOrNil(userData),
	})

	if err != nil {
		return instanceIds, fmt.Errorf("Failed to launch AMI in region: %s; %s", region, err.Error())
	}

	if reservation != nil {
		Log.Debugf("EC2 run instance reservation created: %s", reservation)
		for i := range reservation.Instances {
			instanceIds = append(instanceIds, *reservation.Instances[i].InstanceId)
		}
	}

	return instanceIds, err
}

// GetInstanceDetails retrieves EC2 instance details for a given instance id
func GetInstanceDetails(accessKeyID, secretAccessKey, region, instanceID string) (response *ec2.DescribeInstancesOutput, err error) {
	client, err := NewEC2(accessKeyID, secretAccessKey, stringOrNil(region))

	response, err = client.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{stringOrNil(instanceID)},
	})

	if response != nil {
		Log.Debugf("EC2 instance details retrieved for %s: %s", instanceID, response)
	}

	return response, err
}

// TerminateInstance destroys an EC2 instance given its instance id
func TerminateInstance(accessKeyID, secretAccessKey, instanceID string) (response *ec2.TerminateInstancesOutput, err error) {
	client, err := NewEC2(accessKeyID, secretAccessKey, nil)

	response, err = client.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: []*string{stringOrNil(instanceID)},
	})

	if err != nil {
		Log.Warningf("EC2 instance not terminated for %s: %s", instanceID, response)
	}

	return response, err
}
