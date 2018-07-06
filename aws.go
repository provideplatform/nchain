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

// GetTaskDefinition retrieves ECS task definition containing one or more docker containers
func GetTaskDefinition(accessKeyID, secretAccessKey, region, taskDefinition string) (response *ecs.DescribeTaskDefinitionOutput, err error) {
	client, err := NewECS(accessKeyID, secretAccessKey, region)

	response, err = client.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: stringOrNil(taskDefinition),
	})

	if response != nil {
		Log.Debugf("ECS task definition retrieved for %s: %s", taskDefinition, response)
	}

	return response, err
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

// AuthorizeSecurityGroupEgress authorizes egress for a given lists of tcp and udp ports on a given security group
func AuthorizeSecurityGroupEgress(accessKeyID, secretAccessKey, region, securityGroupID, ipv4Cidr string, tcpPorts, udpPorts []int64) (response *ec2.AuthorizeSecurityGroupEgressOutput, err error) {
	client, err := NewEC2(accessKeyID, secretAccessKey, region)

	ranges := make([]*ec2.IpRange, 0)
	ranges = append(ranges, &ec2.IpRange{
		CidrIp: stringOrNil(ipv4Cidr),
	})

	permissions := make([]*ec2.IpPermission, 0)
	for i := range tcpPorts {
		port := tcpPorts[i]
		permissions = append(permissions, &ec2.IpPermission{
			FromPort:   &port,
			ToPort:     &port,
			IpProtocol: stringOrNil("tcp"),
			IpRanges:   ranges,
		})
	}
	for i := range udpPorts {
		port := udpPorts[i]
		permissions = append(permissions, &ec2.IpPermission{
			FromPort:   &port,
			ToPort:     &port,
			IpProtocol: stringOrNil("udp"),
			IpRanges:   ranges,
		})
	}

	response, err = client.AuthorizeSecurityGroupEgress(&ec2.AuthorizeSecurityGroupEgressInput{
		GroupId:       stringOrNil(securityGroupID),
		IpPermissions: permissions,
	})

	if response != nil {
		Log.Debugf("EC2 security group egress authorization successful for %s: %s", region, response)
	}

	return response, err
}

// AuthorizeSecurityGroupEgressAllPortsAllProtocols authorizes egress for all ports and protocols on a given security group
func AuthorizeSecurityGroupEgressAllPortsAllProtocols(accessKeyID, secretAccessKey, region, securityGroupID string) (response *ec2.AuthorizeSecurityGroupEgressOutput, err error) {
	client, err := NewEC2(accessKeyID, secretAccessKey, region)

	ranges := make([]*ec2.IpRange, 0)
	ranges = append(ranges, &ec2.IpRange{
		CidrIp: stringOrNil("0.0.0.0/0"),
	})

	permissions := make([]*ec2.IpPermission, 0)
	permissions = append(permissions, &ec2.IpPermission{
		IpProtocol: stringOrNil("-1"),
		IpRanges:   ranges,
	})

	response, err = client.AuthorizeSecurityGroupEgress(&ec2.AuthorizeSecurityGroupEgressInput{
		GroupId:       stringOrNil(securityGroupID),
		IpPermissions: permissions,
	})

	if response != nil {
		Log.Debugf("EC2 security group egress authorization successful for %s: %s", region, response)
	}

	return response, err
}

// AuthorizeSecurityGroupIngressAllPortsAllProtocols authorizes egress for all ports and protocols on a given security group
func AuthorizeSecurityGroupIngressAllPortsAllProtocols(accessKeyID, secretAccessKey, region, securityGroupID string) (response *ec2.AuthorizeSecurityGroupIngressOutput, err error) {
	client, err := NewEC2(accessKeyID, secretAccessKey, region)

	ranges := make([]*ec2.IpRange, 0)
	ranges = append(ranges, &ec2.IpRange{
		CidrIp: stringOrNil("0.0.0.0/0"),
	})

	permissions := make([]*ec2.IpPermission, 0)
	permissions = append(permissions, &ec2.IpPermission{
		IpProtocol: stringOrNil("-1"),
		IpRanges:   ranges,
	})

	response, err = client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		GroupId:       stringOrNil(securityGroupID),
		IpPermissions: permissions,
	})

	if response != nil {
		Log.Debugf("EC2 security group egress authorization successful for %s: %s", region, response)
	}

	return response, err
}

// AuthorizeSecurityGroupIngress authorizes ingress for a given lists of tcp and udp ports on a given security group
func AuthorizeSecurityGroupIngress(accessKeyID, secretAccessKey, region, securityGroupID, ipv4Cidr string, tcpPorts, udpPorts []int64) (response *ec2.AuthorizeSecurityGroupIngressOutput, err error) {
	client, err := NewEC2(accessKeyID, secretAccessKey, region)

	ranges := make([]*ec2.IpRange, 0)
	ranges = append(ranges, &ec2.IpRange{
		CidrIp: stringOrNil(ipv4Cidr),
	})

	permissions := make([]*ec2.IpPermission, 0)
	for i := range tcpPorts {
		port := tcpPorts[i]
		permissions = append(permissions, &ec2.IpPermission{
			FromPort:   &port,
			ToPort:     &port,
			IpProtocol: stringOrNil("tcp"),
			IpRanges:   ranges,
		})
	}
	for i := range udpPorts {
		port := udpPorts[i]
		permissions = append(permissions, &ec2.IpPermission{
			FromPort:   &port,
			ToPort:     &port,
			IpProtocol: stringOrNil("udp"),
			IpRanges:   ranges,
		})
	}

	response, err = client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		GroupId:       stringOrNil(securityGroupID),
		IpPermissions: permissions,
	})

	if response != nil {
		Log.Debugf("EC2 security group egress authorization successful for %s: %s", region, response)
	}

	return response, err
}

// CreateSecurityGroup creates a new EC2 security group in the given region using the given rules
func CreateSecurityGroup(accessKeyID, secretAccessKey, region, name, description string, vpcID *string) (response *ec2.CreateSecurityGroupOutput, err error) {
	client, err := NewEC2(accessKeyID, secretAccessKey, region)

	response, err = client.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
		Description: stringOrNil(description),
		GroupName:   stringOrNil(name),
		VpcId:       vpcID,
	})

	if response != nil {
		Log.Debugf("EC2 security group created for %s: %s", region, response)
	}

	return response, err
}

// DeleteSecurityGroup removes an EC2 security group in the given region given the security group id
func DeleteSecurityGroup(accessKeyID, secretAccessKey, region, securityGroupID string) (response *ec2.DeleteSecurityGroupOutput, err error) {
	client, err := NewEC2(accessKeyID, secretAccessKey, region)

	response, err = client.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
		GroupId: stringOrNil(securityGroupID),
	})

	if response != nil {
		Log.Debugf("EC2 security group deleted for %s: %s; security group id: %s", region, response, securityGroupID)
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

	taskDefinitionResp, err := GetTaskDefinition(accessKeyID, secretAccessKey, region, taskDefinition)
	if err != nil {
		return taskIds, fmt.Errorf("Failed to start container in region: %s; %s", region, err.Error())
	}

	containerOverrides := make([]*ecs.ContainerOverride, 0)
	for i := range taskDefinitionResp.TaskDefinition.ContainerDefinitions {
		containerDefinition := taskDefinitionResp.TaskDefinition.ContainerDefinitions[i]

		env := make([]*ecs.KeyValuePair, 0)
		if envOverrides, envOverridesOk := overrides["environment"].(map[string]string); envOverridesOk {
			for envVar := range envOverrides {
				env = append(env, &ecs.KeyValuePair{
					Name:  stringOrNil(envVar),
					Value: stringOrNil(envOverrides[envVar]),
				})
			}
		}

		containerOverrides = append(containerOverrides, &ecs.ContainerOverride{
			Name:        containerDefinition.Name,
			Environment: env,
		})
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

// StopContainer destroys an ECS docker container task given its task id
func StopContainer(accessKeyID, secretAccessKey, region, taskID string, cluster *string) (response *ecs.StopTaskOutput, err error) {
	client, err := NewECS(accessKeyID, secretAccessKey, region)

	if cluster == nil {
		clusters, err := GetClusters(accessKeyID, secretAccessKey, region)
		if err == nil {
			if len(clusters.ClusterArns) > 0 {
				cluster = clusters.ClusterArns[0]
			}
		}
	}

	response, err = client.StopTask(&ecs.StopTaskInput{
		Cluster: cluster,
		Task:    stringOrNil(taskID),
	})

	if err != nil {
		Log.Warningf("Container instance not stopped for %s; %s", taskID, err.Error())
	}

	return response, err
}

// GetContainerDetails retrieves an ECS docker container task given its task id
func GetContainerDetails(accessKeyID, secretAccessKey, region, taskID string, cluster *string) (response *ecs.DescribeTasksOutput, err error) {
	client, err := NewECS(accessKeyID, secretAccessKey, region)

	if cluster == nil {
		clusters, err := GetClusters(accessKeyID, secretAccessKey, region)
		if err == nil {
			if len(clusters.ClusterArns) > 0 {
				cluster = clusters.ClusterArns[0]
			}
		}
	}

	response, err = client.DescribeTasks(&ecs.DescribeTasksInput{
		Cluster: cluster,
		Tasks:   []*string{stringOrNil(taskID)},
	})

	if response != nil {
		Log.Debugf("ECS container details retrieved for %s: %s", taskID, response)
	}

	return response, err
}

// GetNetworkInterfaceDetails retrieves elastic network interface details for a given network interface id
func GetNetworkInterfaceDetails(accessKeyID, secretAccessKey, region, networkInterfaceID string) (response *ec2.DescribeNetworkInterfacesOutput, err error) {
	client, err := NewEC2(accessKeyID, secretAccessKey, region)

	response, err = client.DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{
		NetworkInterfaceIds: []*string{stringOrNil(networkInterfaceID)},
	})

	if response != nil {
		Log.Debugf("EC2 network interface details retrieved for %s: %s", networkInterfaceID, response)
	}

	return response, err
}
