package awswrapper

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elbv2"
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

// NewELB initializes and returns an instance of the ELB API client
func NewELB(accessKeyID, secretAccessKey, region string) (*elb.ELB, error) {
	var err error
	cfg := aws.NewConfig().WithMaxRetries(10).WithRegion(region).WithCredentials(credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""))
	sess := session.New(cfg)
	elb := elb.New(sess)
	return elb, err
}

// NewELBv2 initializes and returns an instance of the ELB v2 API client
func NewELBv2(accessKeyID, secretAccessKey, region string) (*elbv2.ELBV2, error) {
	var err error
	cfg := aws.NewConfig().WithMaxRetries(10).WithRegion(region).WithCredentials(credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""))
	sess := session.New(cfg)
	elb := elbv2.New(sess)
	return elb, err
}

// NewCloudwatchLogs initializes and returns an instance of the Cloudwatch logs API client
func NewCloudwatchLogs(accessKeyID, secretAccessKey, region string) (*cloudwatchlogs.CloudWatchLogs, error) {
	var err error
	cfg := aws.NewConfig().WithMaxRetries(10).WithRegion(region).WithCredentials(credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""))
	sess := session.New(cfg)
	logs := cloudwatchlogs.New(sess)
	return logs, err
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
		log.Debugf("EC2 run instance reservation created: %s", reservation)
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

	if err != nil {
		log.Warningf("ECS task definition retrieval failed for task definition: %;: %s", taskDefinition, err.Error())
		return nil, err
	}

	return response, err
}

// GetInstanceDetails retrieves EC2 instance details for a given instance id
func GetInstanceDetails(accessKeyID, secretAccessKey, region, instanceID string) (response *ec2.DescribeInstancesOutput, err error) {
	client, err := NewEC2(accessKeyID, secretAccessKey, region)

	response, err = client.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{stringOrNil(instanceID)},
	})

	if err != nil {
		log.Warningf("ECS instance details retrieval failed for instance: %s; %s", instanceID, err.Error())
		return nil, err
	}

	return response, err
}

// CreateLoadBalancer creates a load balancer in EC2 for the given region and parameters
func CreateLoadBalancer(accessKeyID, secretAccessKey, region string, vpcID *string, name *string, securityGroupIds []string, listeners []*elb.Listener) (response *elb.CreateLoadBalancerOutput, err error) {
	client, err := NewELB(accessKeyID, secretAccessKey, region)

	groupIds := make([]*string, 0)
	for i := range securityGroupIds {
		groupIds = append(groupIds, stringOrNil(securityGroupIds[i]))
	}

	zones := make([]*string, 0)
	subnets := make([]*string, 0)

	if vpcID != nil && *vpcID == "" {
		vpcsResp, err := GetVPCs(accessKeyID, secretAccessKey, region, nil)
		if err != nil {
			return nil, fmt.Errorf("Failed to provision load balancer in region: %s; %s", region, err.Error())
		}
		if len(vpcsResp.Vpcs) > 0 {
			log.Warningf("No default AWS VPC id provided; attempt to provision load balancer in %s region will use arbitrary VPC", region)
			vpcID = vpcsResp.Vpcs[0].VpcId
		}
	}
	availableSubnets, err := GetSubnets(accessKeyID, secretAccessKey, region, vpcID)
	if err == nil {
		for i := range availableSubnets.Subnets {
			subnets = append(subnets, availableSubnets.Subnets[i].SubnetId)
			zones = append(zones, availableSubnets.Subnets[i].AvailabilityZone)
		}
	}

	response, err = client.CreateLoadBalancer(&elb.CreateLoadBalancerInput{
		AvailabilityZones: zones,
		Listeners:         listeners,
		SecurityGroups:    groupIds,
		LoadBalancerName:  name,
		Subnets:           subnets,
	})

	if err != nil {
		log.Warningf("Failed to provision load balancer in region: %s; %s", region, err.Error())
		return nil, err
	}

	return response, nil
}

// DeleteLoadBalancer deprovisions and removes a load balancer in EC2 for the given region and parameters
func DeleteLoadBalancer(accessKeyID, secretAccessKey, region string, name *string) (response *elb.DeleteLoadBalancerOutput, err error) {
	client, err := NewELB(accessKeyID, secretAccessKey, region)

	response, err = client.DeleteLoadBalancer(&elb.DeleteLoadBalancerInput{
		LoadBalancerName: name,
	})

	if err != nil {
		log.Warningf("Failed to deprovision load balancer in region: %s; %s", region, err.Error())
		return nil, err
	}

	return response, nil
}

// GetLoadBalancers retrieves EC2 load balancers for the given region
func GetLoadBalancers(accessKeyID, secretAccessKey, region string, loadBalancerName *string) (response *elb.DescribeLoadBalancersOutput, err error) {
	client, err := NewELB(accessKeyID, secretAccessKey, region)

	response, err = client.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{
		LoadBalancerNames: []*string{loadBalancerName},
	})

	if err != nil {
		log.Warningf("Load balancer details retrieval failed for region: %s; %s", region, err.Error())
		return nil, err
	}

	return response, err
}

// CreateLoadBalancerV2 creates a load balancer in EC2 for the given region and parameters
func CreateLoadBalancerV2(accessKeyID, secretAccessKey, region string, vpcID, name, balancerType *string, securityGroupIds []string) (response *elbv2.CreateLoadBalancerOutput, err error) {
	client, err := NewELBv2(accessKeyID, secretAccessKey, region)

	groupIds := make([]*string, 0)
	for i := range securityGroupIds {
		groupIds = append(groupIds, stringOrNil(securityGroupIds[i]))
	}

	zones := make([]*string, 0)
	subnets := make([]*string, 0)

	if vpcID != nil && *vpcID == "" {
		vpcsResp, err := GetVPCs(accessKeyID, secretAccessKey, region, nil)
		if err != nil {
			return nil, fmt.Errorf("Failed to provision load balancer in region: %s; %s", region, err.Error())
		}
		if len(vpcsResp.Vpcs) > 0 {
			log.Warningf("No default AWS VPC id provided; attempt to provision load balancer in %s region will use arbitrary VPC", region)
			vpcID = vpcsResp.Vpcs[0].VpcId
		}
	}
	availableSubnets, err := GetSubnets(accessKeyID, secretAccessKey, region, vpcID)
	if err == nil {
		for i := range availableSubnets.Subnets {
			subnets = append(subnets, availableSubnets.Subnets[i].SubnetId)
			zones = append(zones, availableSubnets.Subnets[i].AvailabilityZone)
		}
	}

	response, err = client.CreateLoadBalancer(&elbv2.CreateLoadBalancerInput{
		SecurityGroups: groupIds,
		Name:           name,
		Subnets:        subnets,
		Type:           balancerType,
	})

	if err != nil {
		log.Warningf("Failed to provision load balancer in region: %s; %s", region, err.Error())
		return nil, err
	}

	return response, nil
}

// CreateListenerV2 creates a load balanced listener in EC2 for the given region and parameters
func CreateListenerV2(accessKeyID, secretAccessKey, region string, loadBalancerARN, targetGroupARN, protocol *string, port *int64) (*elbv2.CreateListenerOutput, error) {
	client, err := NewELBv2(accessKeyID, secretAccessKey, region)

	order := int64(1)
	response, err := client.CreateListener(&elbv2.CreateListenerInput{
		// Certificates []*Certificate `type:"list"`
		DefaultActions: []*elbv2.Action{&elbv2.Action{
			Order:          &order,
			TargetGroupArn: targetGroupARN,
			Type:           stringOrNil("forward"),
		}},
		LoadBalancerArn: loadBalancerARN,
		Port:            port,
		Protocol:        protocol,
		// SslPolicy *string `type:"string"`
	})

	if err != nil {
		log.Warningf("Failed to create load balanced listener in region: %s; %s", region, err.Error())
		return nil, err
	}

	return response, nil
}

// DeleteLoadBalancerV2 deprovisions and removes a load balancer in EC2 for the given region and parameters
func DeleteLoadBalancerV2(accessKeyID, secretAccessKey, region string, loadBalancerARN *string) (response *elbv2.DeleteLoadBalancerOutput, err error) {
	client, err := NewELBv2(accessKeyID, secretAccessKey, region)

	response, err = client.DeleteLoadBalancer(&elbv2.DeleteLoadBalancerInput{
		LoadBalancerArn: loadBalancerARN,
	})

	if err != nil {
		log.Warningf("Failed to deprovision load balancer in region: %s; %s", region, err.Error())
		return nil, err
	}

	return response, nil
}

// GetLoadBalancersV2 retrieves EC2 load balancers for the given region
func GetLoadBalancersV2(accessKeyID, secretAccessKey, region string, loadBalancerArn *string, loadBalancerName *string) (response *elbv2.DescribeLoadBalancersOutput, err error) {
	client, err := NewELBv2(accessKeyID, secretAccessKey, region)

	response, err = client.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{
		LoadBalancerArns: []*string{loadBalancerArn},
		Names:            []*string{loadBalancerName},
	})

	if err != nil {
		log.Warningf("Load balancer details retrieval failed for region: %s; %s", region, err.Error())
		return nil, err
	}

	return response, err
}

// CreateTargetGroup creates a target group for load balancing
func CreateTargetGroup(accessKeyID, secretAccessKey, region string, vpcID *string, name, protocol *string, port int64) (response *elbv2.CreateTargetGroupOutput, err error) {
	client, err := NewELBv2(accessKeyID, secretAccessKey, region)

	if vpcID != nil && *vpcID == "" {
		vpcsResp, err := GetVPCs(accessKeyID, secretAccessKey, region, nil)
		if err != nil {
			return nil, fmt.Errorf("Failed to create target group in region: %s; %s", region, err.Error())
		}
		if len(vpcsResp.Vpcs) > 0 {
			log.Warningf("No default AWS VPC id provided; attempt to create target group in %s region will use arbitrary VPC", region)
			vpcID = vpcsResp.Vpcs[0].VpcId
		}
	}

	response, err = client.CreateTargetGroup(&elbv2.CreateTargetGroupInput{
		Name:       name,
		Port:       &port,
		Protocol:   protocol,
		TargetType: stringOrNil("ip"),
		VpcId:      vpcID,
	})

	if err != nil {
		log.Warningf("Failed to register target group in region: %s; %s", region, err.Error())
		return nil, err
	}

	return response, nil
}

// DeleteTargetGroup creates a target group for load balancing
func DeleteTargetGroup(accessKeyID, secretAccessKey, region string, targetGroupARN *string) (response *elbv2.DeleteTargetGroupOutput, err error) {
	client, err := NewELBv2(accessKeyID, secretAccessKey, region)

	response, err = client.DeleteTargetGroup(&elbv2.DeleteTargetGroupInput{
		TargetGroupArn: targetGroupARN,
	})

	if err != nil {
		log.Warningf("Failed to delete target group in region: %s; %s", region, err.Error())
		return nil, err
	}

	return response, nil
}

// RegisterTarget registers a docker container with a load-balanced target group
func RegisterTarget(accessKeyID, secretAccessKey, region string, targetGroupARN, ipAddress *string, port *int64) (response *elbv2.RegisterTargetsOutput, err error) {
	client, err := NewELBv2(accessKeyID, secretAccessKey, region)

	response, err = client.RegisterTargets(&elbv2.RegisterTargetsInput{
		TargetGroupArn: targetGroupARN,
		Targets: []*elbv2.TargetDescription{
			&elbv2.TargetDescription{
				Id:   ipAddress,
				Port: port,
			},
		},
	})

	if err != nil {
		log.Warningf("Failed to register target from load balancer in region: %s; %s", region, err.Error())
		return nil, err
	}

	return response, nil
}

// DeregisterTarget deregisters a docker container with a load-balanced target group
func DeregisterTarget(accessKeyID, secretAccessKey, region string, targetGroupARN, ipAddress *string, port *int64) (response *elbv2.DeregisterTargetsOutput, err error) {
	client, err := NewELBv2(accessKeyID, secretAccessKey, region)

	response, err = client.DeregisterTargets(&elbv2.DeregisterTargetsInput{
		TargetGroupArn: targetGroupARN,
		Targets: []*elbv2.TargetDescription{
			&elbv2.TargetDescription{
				Id:   ipAddress,
				Port: port,
			},
		},
	})

	if err != nil {
		log.Warningf("Failed to deregister target from load balancer in region: %s; %s", region, err.Error())
		return nil, err
	}

	return response, nil
}

// RegisterInstanceWithLoadBalancer creates a load balancer in EC2 for the given region and parameters
func RegisterInstanceWithLoadBalancer(accessKeyID, secretAccessKey, region string, loadBalancerName, instanceID *string) (response *elb.RegisterInstancesWithLoadBalancerOutput, err error) {
	client, err := NewELB(accessKeyID, secretAccessKey, region)

	response, err = client.RegisterInstancesWithLoadBalancer(&elb.RegisterInstancesWithLoadBalancerInput{
		Instances: []*elb.Instance{
			&elb.Instance{InstanceId: instanceID},
		},
		LoadBalancerName: loadBalancerName,
	})

	if err != nil {
		log.Warningf("Failed to register instance with load balancer in region: %s; %s", region, err.Error())
		return nil, err
	}

	return response, nil
}

// DeregisterInstanceFromLoadBalancer creates a load balancer in EC2 for the given region and parameters
func DeregisterInstanceFromLoadBalancer(accessKeyID, secretAccessKey, region string, loadBalancerName, instanceID *string) (response *elb.DeregisterInstancesFromLoadBalancerOutput, err error) {
	client, err := NewELB(accessKeyID, secretAccessKey, region)

	response, err = client.DeregisterInstancesFromLoadBalancer(&elb.DeregisterInstancesFromLoadBalancerInput{
		Instances: []*elb.Instance{
			&elb.Instance{InstanceId: instanceID},
		},
		LoadBalancerName: loadBalancerName,
	})

	if err != nil {
		log.Warningf("Failed to deregister instance from load balancer in region: %s; %s", region, err.Error())
		return nil, err
	}

	return response, nil
}

// GetSecurityGroups retrieves EC2 security group details for the given region
func GetSecurityGroups(accessKeyID, secretAccessKey, region string) (response *ec2.DescribeSecurityGroupsOutput, err error) {
	client, err := NewEC2(accessKeyID, secretAccessKey, region)

	response, err = client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{})

	if err != nil {
		log.Warningf("EC2 security group details retrieval failed for region: %s; %s", region, err.Error())
		return nil, err
	}

	return response, err
}

// GetVPCs retrieves EC2 VPC details for the given region
func GetVPCs(accessKeyID, secretAccessKey, region string, vpcID *string) (response *ec2.DescribeVpcsOutput, err error) {
	client, err := NewEC2(accessKeyID, secretAccessKey, region)

	response, err = client.DescribeVpcs(&ec2.DescribeVpcsInput{})

	if err != nil {
		log.Warningf("EC2 VPC details retrieval failed for region: %s; %s", region, err.Error())
		return nil, err
	}

	return response, err
}

// GetSubnets retrieves EC2 subnet details for the given region
func GetSubnets(accessKeyID, secretAccessKey, region string, vpcID *string) (response *ec2.DescribeSubnetsOutput, err error) {
	client, err := NewEC2(accessKeyID, secretAccessKey, region)

	describeSubnetsInput := &ec2.DescribeSubnetsInput{}
	subnetFilters := make([]*ec2.Filter, 0)
	if vpcID != nil {
		subnetFilters = append(subnetFilters, &ec2.Filter{
			Name:   stringOrNil("vpc-id"),
			Values: []*string{vpcID},
		})
	}
	describeSubnetsInput.SetFilters(subnetFilters)

	response, err = client.DescribeSubnets(describeSubnetsInput)

	if err != nil {
		log.Warningf("EC2 subnet details retrieval failed for region: %s; %s", region, err.Error())
		return nil, err
	}

	return response, err
}

// GetClusters retrieves ECS cluster details for the given region
func GetClusters(accessKeyID, secretAccessKey, region string) (response *ecs.ListClustersOutput, err error) {
	client, err := NewECS(accessKeyID, secretAccessKey, region)

	response, err = client.ListClusters(&ecs.ListClustersInput{})

	if err != nil {
		log.Warningf("ECS cluster details retrieval failed for region: %s; %s", region, err.Error())
		return nil, err
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

	if err != nil {
		log.Warningf("EC2 security group egress authorization failed for %s; %s", region, err.Error())
		return nil, err
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

	if err != nil {
		log.Warningf("EC2 security group egress authorization failed for %s; %s", region, err.Error())
		return nil, err
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

	if err != nil {
		log.Warningf("EC2 security group ingress authorization failed for %s; %s", region, err.Error())
		return nil, err
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

	if err != nil {
		log.Warningf("EC2 security group ingress authorization failed for %s; %s", region, err.Error())
		return nil, err
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

	if err != nil {
		log.Warningf("EC2 security group creation failed failed for %s; %s", region, err.Error())
		return nil, err
	}

	return response, err
}

// DeleteSecurityGroup removes an EC2 security group in the given region given the security group id
func DeleteSecurityGroup(accessKeyID, secretAccessKey, region, securityGroupID string) (response *ec2.DeleteSecurityGroupOutput, err error) {
	client, err := NewEC2(accessKeyID, secretAccessKey, region)

	response, err = client.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
		GroupId: stringOrNil(securityGroupID),
	})

	if err != nil {
		log.Warningf("EC2 security group deletion failed for %s; %s", region, err.Error())
		return nil, err
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

	if err != nil {
		log.Warningf("EC2 instance attribute modification failed for %s; %s", region, err.Error())
		return nil, err
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
		log.Warningf("EC2 instance termination request failed for instance id: %s; %s", instanceID, err.Error())
		return nil, err
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
		vpcID := awsDefaultVpcID
		if vpcID == "" {
			vpcsResp, err := GetVPCs(accessKeyID, secretAccessKey, region, nil)
			if err != nil {
				return taskIds, fmt.Errorf("Failed to start container in region: %s; %s", region, err.Error())
			}
			if len(vpcsResp.Vpcs) > 0 {
				log.Warningf("No default AWS VPC id provided; attempt to start container in %s region will use arbitrary VPC", region)
				vpcID = *vpcsResp.Vpcs[0].VpcId
			}
		}
		availableSubnets, err := GetSubnets(accessKeyID, secretAccessKey, region, stringOrNil(vpcID))
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
		if envOverrides, envOverridesOk := overrides["environment"].(map[string]interface{}); envOverridesOk {
			for envVar := range envOverrides {
				if val, valOk := envOverrides[envVar].(string); valOk {
					env = append(env, &ecs.KeyValuePair{
						Name:  stringOrNil(envVar),
						Value: stringOrNil(val),
					})
				} else if val, valOk := envOverrides[envVar].(*string); valOk {
					env = append(env, &ecs.KeyValuePair{
						Name:  stringOrNil(envVar),
						Value: val,
					})
				}
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
		log.Warningf("ECS docker container start request failed for task definition: %s; %s", taskDefinition, err.Error())
		return taskIds, err
	}

	if response != nil {
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
		log.Warningf("ECS docker container not stopped for task id: %s; %s", taskID, err.Error())
		return nil, err
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

	if err != nil {
		log.Warningf("ECS container details retreival failed for task id: %s; %s", taskID, err.Error())
		return nil, err
	}

	return response, err
}

// GetContainerLogEvents returns cloudwatch log events for the given ECS docker container task given its task id=
func GetContainerLogEvents(accessKeyID, secretAccessKey, region, taskID string, cluster *string) (response *cloudwatchlogs.GetLogEventsOutput, err error) {
	containerDetails, err := GetContainerDetails(accessKeyID, secretAccessKey, region, taskID, cluster)
	if err == nil {
		if len(containerDetails.Tasks) > 0 {
			task := containerDetails.Tasks[0]
			taskDefinition, err := GetTaskDefinition(accessKeyID, secretAccessKey, region, *task.TaskDefinitionArn)
			if err == nil {
				if len(taskDefinition.TaskDefinition.ContainerDefinitions) > 0 {
					containerDefinition := taskDefinition.TaskDefinition.ContainerDefinitions[0]
					containerLogConfig := containerDefinition.LogConfiguration
					if containerLogConfig != nil && containerLogConfig.LogDriver != nil && *containerLogConfig.LogDriver == "awslogs" {
						logGroup := containerLogConfig.Options["awslogs-group"]
						logRegion := containerLogConfig.Options["awslogs-region"]
						logStreamPrefix := containerLogConfig.Options["awslogs-stream-prefix"]
						if logGroup != nil && logRegion != nil && logStreamPrefix != nil {
							taskArnSplitIdx := strings.LastIndex(*task.TaskArn, "/")
							taskArn := string(*task.TaskArn)
							logStream := fmt.Sprintf("%s/%s/%s", *logStreamPrefix, *containerDefinition.Name, taskArn[taskArnSplitIdx+1:])
							return GetLogEvents(accessKeyID, secretAccessKey, *logRegion, *logGroup, logStream, true)
						}
					}
				}
			}
		}
	}
	return nil, err
}

// GetLogEvents retrieves cloudwatch log events for a given log stream id
func GetLogEvents(accessKeyID, secretAccessKey, region, logGroupID string, logStreamID string, startFromHead bool) (response *cloudwatchlogs.GetLogEventsOutput, err error) {
	client, err := NewCloudwatchLogs(accessKeyID, secretAccessKey, region)

	response, err = client.GetLogEvents(&cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  stringOrNil(logGroupID),
		LogStreamName: stringOrNil(logStreamID),
		StartFromHead: &startFromHead,
	})

	if err != nil {
		log.Warningf("Cloudwatch log retreival failed for log stream: %s; %s", logStreamID, err.Error())
		return nil, err
	}

	return response, err
}

// GetNetworkInterfaceDetails retrieves elastic network interface details for a given network interface id
func GetNetworkInterfaceDetails(accessKeyID, secretAccessKey, region, networkInterfaceID string) (response *ec2.DescribeNetworkInterfacesOutput, err error) {
	client, err := NewEC2(accessKeyID, secretAccessKey, region)

	response, err = client.DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{
		NetworkInterfaceIds: []*string{stringOrNil(networkInterfaceID)},
	})

	if err != nil {
		log.Warningf("EC2 network interface details retreival failed for network interface id: %s; %s", networkInterfaceID, err.Error())
		return nil, err
	}

	return response, err
}

func stringOrNil(str string) *string {
	if str == "" {
		return nil
	}
	return &str
}
