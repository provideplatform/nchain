package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
)

// NewEC2 initializes and returns an instance of the EC2 API client
func NewEC2(accessKeyID, secretAccessKey, region string) (*ec2.EC2, error) {
	var err error
	cfg := aws.NewConfig().WithMaxRetries(10).WithRegion(region).WithCredentials(credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""))
	sess := session.New(cfg)
	ec2 := ec2.New(sess)
	return ec2, err
}

// NewECS initializes and returns an instance of the ECS API client
func NewECS(accessKeyID, secretAccessKey, region string) (*ecs.ECS, error) {
	var err error
	cfg := aws.NewConfig().WithMaxRetries(10).WithRegion(region).WithCredentials(credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""))
	sess := session.New(cfg)
	ecs := ecs.New(sess)
	return ecs, err
}

// LaunchAMI launches an EC2 instance for a given AMI id
func LaunchAMI(accessKeyID, secretAccessKey, region, imageID, userData string, minCount, maxCount int64) (instanceIds []string, err error) {
	client, err := NewEC2(accessKeyID, secretAccessKey, region)

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
	client, err := NewEC2(accessKeyID, secretAccessKey, region)

	response, err = client.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{stringOrNil(instanceID)},
	})

	if response != nil {
		Log.Debugf("EC2 instance details retrieved for %s: %s", instanceID, response)
	}

	return response, err
}

// GetSecurityGroups retrieves EC2 security group details for the given region
func GetSecurityGroups(accessKeyID, secretAccessKey, region string) (response *ec2.DescribeSecurityGroupsOutput, err error) {
	client, err := NewEC2(accessKeyID, secretAccessKey, region)

	response, err = client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{})

	if response != nil {
		Log.Debugf("EC2 security group details retrieved for %s: %s", region, response)
	}

	return response, err
}

// GetSubnets retrieves EC2 subnet details for the given region
func GetSubnets(accessKeyID, secretAccessKey, region string) (response *ec2.DescribeSubnetsOutput, err error) {
	client, err := NewEC2(accessKeyID, secretAccessKey, region)

	response, err = client.DescribeSubnets(&ec2.DescribeSubnetsInput{})

	if response != nil {
		Log.Debugf("EC2 subnet details retrieved for %s: %s", region, response)
	}

	return response, err
}

// GetClusters retrieves ECS cluster details for the given region
func GetClusters(accessKeyID, secretAccessKey, region string) (response *ecs.ListClustersOutput, err error) {
	client, err := NewECS(accessKeyID, secretAccessKey, region)

	response, err = client.ListClusters(&ecs.ListClustersInput{})

	if response != nil {
		Log.Debugf("ECS cluster details retrieved for %s: %s", region, response)
	}

	return response, err
}

// SetInstanceSecurityGroups sets the security groups for a given instance id and array of security group ids
func SetInstanceSecurityGroups(accessKeyID, secretAccessKey, region, instanceID string, securityGroupIds []string) (response *ec2.ModifyInstanceAttributeOutput, err error) {
	client, err := NewEC2(accessKeyID, secretAccessKey, region)

	groupIds := make([]*string, 0)
	for i := range securityGroupIds {
		groupIds = append(groupIds, stringOrNil(securityGroupIds[i]))
	}

	response, err = client.ModifyInstanceAttribute(&ec2.ModifyInstanceAttributeInput{
		InstanceId: stringOrNil(instanceID),
		Groups:     groupIds,
	})

	if response != nil {
		Log.Debugf("EC2 instance attribute modified for %s: %s", instanceID, response)
	}

	return response, err
}

// TerminateInstance destroys an EC2 instance given its instance id
func TerminateInstance(accessKeyID, secretAccessKey, region, instanceID string) (response *ec2.TerminateInstancesOutput, err error) {
	client, err := NewEC2(accessKeyID, secretAccessKey, region)

	response, err = client.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: []*string{stringOrNil(instanceID)},
	})

	if err != nil {
		Log.Warningf("EC2 instance not terminated for %s; %s", instanceID, err.Error())
	}

	return response, err
}

// StartContainer starts a new ECS task for the given task definition
func StartContainer(accessKeyID, secretAccessKey, region, taskDefinition string, launchType, cluster *string, securityGroupIds []string, subnetIds []string, overrides map[string]interface{}) (taskIds []string, err error) {
	client, err := NewECS(accessKeyID, secretAccessKey, region)

	if launchType == nil {
		launchType = stringOrNil("FARGATE")
	}

	if cluster == nil {
		clusters, err := GetClusters(accessKeyID, secretAccessKey, region)
		if err == nil {
			if len(clusters.ClusterArns) > 0 {
				cluster = clusters.ClusterArns[0]
			}
		}
	}

	securityGroups := make([]*string, 0)
	for i := range securityGroupIds {
		securityGroups = append(securityGroups, stringOrNil(securityGroupIds[i]))
	}

	subnets := make([]*string, 0)
	if len(subnetIds) > 0 {
		for i := range subnetIds {
			subnets = append(subnets, stringOrNil(subnetIds[i]))
		}
	} else {
		availableSubnets, err := GetSubnets(accessKeyID, secretAccessKey, region)
		if err == nil {
			for i := range availableSubnets.Subnets {
				subnets = append(subnets, availableSubnets.Subnets[i].SubnetId)
			}
		}
	}

	containerOverrides := make([]*ecs.ContainerOverride, 0)
	for container := range overrides {
		env := make([]*ecs.KeyValuePair, 0)
		envOverrides, envOverridesOk := overrides[container].(map[string]interface{})["environment"].(map[string]string)
		if envOverridesOk {
			for envVar := range envOverrides {
				env = append(env, &ecs.KeyValuePair{
					Name:  stringOrNil(envVar),
					Value: stringOrNil(envOverrides[envVar]),
				})
			}
		}
		override := &ecs.ContainerOverride{
			Name:        stringOrNil(container),
			Environment: env,
		}
		containerOverrides = append(containerOverrides, override)
	}

	response, err := client.RunTask(&ecs.RunTaskInput{
		Cluster:        cluster,
		TaskDefinition: stringOrNil(taskDefinition),
		LaunchType:     launchType,
		NetworkConfiguration: &ecs.NetworkConfiguration{
			AwsvpcConfiguration: &ecs.AwsVpcConfiguration{
				AssignPublicIp: stringOrNil("ENABLED"),
				SecurityGroups: securityGroups,
				Subnets:        subnets,
			},
		},
		Overrides: &ecs.TaskOverride{
			ContainerOverrides: containerOverrides,
		},
	})

	if err != nil {
		return taskIds, fmt.Errorf("Failed to start container in region: %s; %s", region, err.Error())
	}

	if response != nil {
		Log.Debugf("ECS run task response received: %s", response)
		for i := range response.Tasks {
			taskIds = append(taskIds, *response.Tasks[i].TaskArn)
		}
	}

	return taskIds, err
}

// StopContainer destroys an EC2 instance given its instance id
func StopContainer(accessKeyID, secretAccessKey, region, cluster, taskID string) (response *ecs.StopTaskOutput, err error) {
	client, err := NewECS(accessKeyID, secretAccessKey, region)

	response, err = client.StopTask(&ecs.StopTaskInput{
		Cluster: stringOrNil(cluster),
		Task:    stringOrNil(taskID),
	})

	if err != nil {
		Log.Warningf("Container instance not stopped for %s; %s", taskID, err.Error())
	}

	return response, err
}
