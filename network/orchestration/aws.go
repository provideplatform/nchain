package orchestration

import (
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elbv2"
	awswrapper "github.com/kthomas/go-aws-wrapper"
	"github.com/provideapp/goldmine/common"
)

// AWSOrchestrationProvider is a network.OrchestrationAPI implementing the AWS API
type AWSOrchestrationProvider struct {
	accessKeyID     string
	secretAccessKey string
	region          string
}

// InitAWSOrchestrationProvider initializes and returns the Amazon Web Services infrastructure orchestration provider
func InitAWSOrchestrationProvider(credentials map[string]interface{}, region string) *AWSOrchestrationProvider {
	accessKeyID, accessKeyIDOk := credentials["aws_access_key_id"].(string)
	secretAccessKey, secretAccessKeyOk := credentials["aws_secret_access_key"].(string)
	if !accessKeyIDOk || !secretAccessKeyOk {
		common.Log.Warning("Failed to initialize AWS orchestration API provider; both aws_access_key_id and aws_secret_access_key are required credentials")
		return nil
	}

	return &AWSOrchestrationProvider{
		accessKeyID:     accessKeyID,
		secretAccessKey: secretAccessKey,
		region:          region,
	}
}

// LaunchAMI needs docs
func (p *AWSOrchestrationProvider) LaunchAMI(imageID, userData string, minCount, maxCount int64) (instanceIds []string, err error) {
	return awswrapper.LaunchAMI(p.accessKeyID, p.secretAccessKey, p.region, imageID, userData, minCount, maxCount)
}

// GetTaskDefinition needs docs
func (p *AWSOrchestrationProvider) GetTaskDefinition(taskDefinition string) (response *ecs.DescribeTaskDefinitionOutput, err error) {
	return awswrapper.GetTaskDefinition(p.accessKeyID, p.secretAccessKey, p.region, taskDefinition)
}

// GetInstanceDetails needs docs
func (p *AWSOrchestrationProvider) GetInstanceDetails(instanceID string) (response *ec2.DescribeInstancesOutput, err error) {
	return awswrapper.GetInstanceDetails(p.accessKeyID, p.secretAccessKey, p.region, instanceID)

}

// CreateLoadBalancer needs docs
func (p *AWSOrchestrationProvider) CreateLoadBalancer(vpcID *string, name *string, securityGroupIds []string, listeners []*elb.Listener) (response *elb.CreateLoadBalancerOutput, err error) {
	return awswrapper.CreateLoadBalancer(p.accessKeyID, p.secretAccessKey, p.region, vpcID, name, securityGroupIds, listeners)

}

// DeleteLoadBalancer needs docs
func (p *AWSOrchestrationProvider) DeleteLoadBalancer(name *string) (response *elb.DeleteLoadBalancerOutput, err error) {
	return awswrapper.DeleteLoadBalancer(p.accessKeyID, p.secretAccessKey, p.region, name)

}

// GetLoadBalancers needs docs
func (p *AWSOrchestrationProvider) GetLoadBalancers(loadBalancerName *string) (response *elb.DescribeLoadBalancersOutput, err error) {
	return awswrapper.GetLoadBalancers(p.accessKeyID, p.secretAccessKey, p.region, loadBalancerName)

}

// CreateLoadBalancerV2 needs docs
func (p *AWSOrchestrationProvider) CreateLoadBalancerV2(vpcID, name, balancerType *string, securityGroupIds []string) (response *elbv2.CreateLoadBalancerOutput, err error) {
	return awswrapper.CreateLoadBalancerV2(p.accessKeyID, p.secretAccessKey, p.region, vpcID, name, balancerType, securityGroupIds)

}

// CreateDefaultSubnets needs docs
func (p *AWSOrchestrationProvider) CreateDefaultSubnets(vpcID string) ([]*ec2.CreateDefaultSubnetOutput, error) {
	return awswrapper.CreateDefaultSubnets(p.accessKeyID, p.secretAccessKey, p.region, vpcID)

}

// CreateListenerV2 needs docs
func (p *AWSOrchestrationProvider) CreateListenerV2(loadBalancerARN, targetGroupARN, protocol *string, port *int64) (*elbv2.CreateListenerOutput, error) {
	return awswrapper.CreateListenerV2(p.accessKeyID, p.secretAccessKey, p.region, loadBalancerARN, targetGroupARN, protocol, port)

}

// DeleteLoadBalancerV2 needs docs
func (p *AWSOrchestrationProvider) DeleteLoadBalancerV2(loadBalancerARN *string) (response *elbv2.DeleteLoadBalancerOutput, err error) {
	return awswrapper.DeleteLoadBalancerV2(p.accessKeyID, p.secretAccessKey, p.region, loadBalancerARN)

}

// GetLoadBalancersV2 needs docs
func (p *AWSOrchestrationProvider) GetLoadBalancersV2(loadBalancerArn *string, loadBalancerName *string) (response *elbv2.DescribeLoadBalancersOutput, err error) {
	return awswrapper.GetLoadBalancersV2(p.accessKeyID, p.secretAccessKey, p.region, loadBalancerArn, loadBalancerName)

}

// GetTargetGroup needs docs
func (p *AWSOrchestrationProvider) GetTargetGroup(targetGroupName string) (response *elbv2.DescribeTargetGroupsOutput, err error) {
	return awswrapper.GetTargetGroup(p.accessKeyID, p.secretAccessKey, p.region, targetGroupName)

}

// CreateTargetGroup needs docs
func (p *AWSOrchestrationProvider) CreateTargetGroup(vpcID *string, name, protocol *string, port int64, healthCheckPort, healthCheckStatusCode *int64, healthCheckPath *string) (response *elbv2.CreateTargetGroupOutput, err error) {
	return awswrapper.CreateTargetGroup(p.accessKeyID, p.secretAccessKey, p.region, vpcID, name, protocol, port, healthCheckPort, healthCheckStatusCode, healthCheckPath)

}

// DeleteTargetGroup needs docs
func (p *AWSOrchestrationProvider) DeleteTargetGroup(targetGroupARN *string) (response *elbv2.DeleteTargetGroupOutput, err error) {
	return awswrapper.DeleteTargetGroup(p.accessKeyID, p.secretAccessKey, p.region, targetGroupARN)

}

// RegisterTarget needs docs
func (p *AWSOrchestrationProvider) RegisterTarget(targetGroupARN, ipAddress *string, port *int64) (response *elbv2.RegisterTargetsOutput, err error) {
	return awswrapper.RegisterTarget(p.accessKeyID, p.secretAccessKey, p.region, targetGroupARN, ipAddress, port)

}

// DeregisterTarget needs docs
func (p *AWSOrchestrationProvider) DeregisterTarget(targetGroupARN, ipAddress *string, port *int64) (response *elbv2.DeregisterTargetsOutput, err error) {
	return awswrapper.DeregisterTarget(p.accessKeyID, p.secretAccessKey, p.region, targetGroupARN, ipAddress, port)

}

// RegisterInstanceWithLoadBalancer needs docs
func (p *AWSOrchestrationProvider) RegisterInstanceWithLoadBalancer(loadBalancerName, instanceID *string) (response *elb.RegisterInstancesWithLoadBalancerOutput, err error) {
	return awswrapper.RegisterInstanceWithLoadBalancer(p.accessKeyID, p.secretAccessKey, p.region, loadBalancerName, instanceID)

}

// DeregisterInstanceFromLoadBalancer needs docs
func (p *AWSOrchestrationProvider) DeregisterInstanceFromLoadBalancer(loadBalancerName, instanceID *string) (response *elb.DeregisterInstancesFromLoadBalancerOutput, err error) {
	return awswrapper.DeregisterInstanceFromLoadBalancer(p.accessKeyID, p.secretAccessKey, p.region, loadBalancerName, instanceID)
}

// GetSecurityGroups needs docs
func (p *AWSOrchestrationProvider) GetSecurityGroups() (response *ec2.DescribeSecurityGroupsOutput, err error) {
	return awswrapper.GetSecurityGroups(p.accessKeyID, p.secretAccessKey, p.region)

}

// GetVPCs needs docs
func (p *AWSOrchestrationProvider) GetVPCs(vpcID *string) (response *ec2.DescribeVpcsOutput, err error) {
	return awswrapper.GetVPCs(p.accessKeyID, p.secretAccessKey, p.region, vpcID)

}

// GetSubnets needs docs
func (p *AWSOrchestrationProvider) GetSubnets(vpcID *string) (response *ec2.DescribeSubnetsOutput, err error) {
	return awswrapper.GetSubnets(p.accessKeyID, p.secretAccessKey, p.region, vpcID)

}

// GetClusters needs docs
func (p *AWSOrchestrationProvider) GetClusters() (response *ecs.ListClustersOutput, err error) {
	return awswrapper.GetClusters(p.accessKeyID, p.secretAccessKey, p.region)

}

// AuthorizeSecurityGroupEgress needs docs
func (p *AWSOrchestrationProvider) AuthorizeSecurityGroupEgress(securityGroupID, ipv4Cidr string, tcpPorts, udpPorts []int64) (response *ec2.AuthorizeSecurityGroupEgressOutput, err error) {
	return awswrapper.AuthorizeSecurityGroupEgress(p.accessKeyID, p.secretAccessKey, p.region, securityGroupID, ipv4Cidr, tcpPorts, udpPorts)

}

// AuthorizeSecurityGroupEgressAllPortsAllProtocols needs docs
func (p *AWSOrchestrationProvider) AuthorizeSecurityGroupEgressAllPortsAllProtocols(securityGroupID string) (response *ec2.AuthorizeSecurityGroupEgressOutput, err error) {
	return awswrapper.AuthorizeSecurityGroupEgressAllPortsAllProtocols(p.accessKeyID, p.secretAccessKey, p.region, securityGroupID)

}

// AuthorizeSecurityGroupIngressAllPortsAllProtocols needs docs
func (p *AWSOrchestrationProvider) AuthorizeSecurityGroupIngressAllPortsAllProtocols(securityGroupID string) (response *ec2.AuthorizeSecurityGroupIngressOutput, err error) {
	return awswrapper.AuthorizeSecurityGroupIngressAllPortsAllProtocols(p.accessKeyID, p.secretAccessKey, p.region, securityGroupID)

}

// AuthorizeSecurityGroupIngress needs docs
func (p *AWSOrchestrationProvider) AuthorizeSecurityGroupIngress(securityGroupID, ipv4Cidr string, tcpPorts, udpPorts []int64) (response *ec2.AuthorizeSecurityGroupIngressOutput, err error) {
	return awswrapper.AuthorizeSecurityGroupIngress(p.accessKeyID, p.secretAccessKey, p.region, securityGroupID, ipv4Cidr, tcpPorts, udpPorts)

}

// CreateSecurityGroup needs docs
func (p *AWSOrchestrationProvider) CreateSecurityGroup(name, description string, vpcID *string) (response *ec2.CreateSecurityGroupOutput, err error) {
	return awswrapper.CreateSecurityGroup(p.accessKeyID, p.secretAccessKey, p.region, name, description, vpcID)

}

// DeleteSecurityGroup needs docs
func (p *AWSOrchestrationProvider) DeleteSecurityGroup(securityGroupID string) (response *ec2.DeleteSecurityGroupOutput, err error) {
	return awswrapper.DeleteSecurityGroup(p.accessKeyID, p.secretAccessKey, p.region, securityGroupID)

}

// SetInstanceSecurityGroups needs docs
func (p *AWSOrchestrationProvider) SetInstanceSecurityGroups(instanceID string, securityGroupIds []string) (response *ec2.ModifyInstanceAttributeOutput, err error) {
	return awswrapper.SetInstanceSecurityGroups(p.accessKeyID, p.secretAccessKey, p.region, instanceID, securityGroupIds)

}

// TerminateInstance needs docs
func (p *AWSOrchestrationProvider) TerminateInstance(instanceID string) (response *ec2.TerminateInstancesOutput, err error) {
	return awswrapper.TerminateInstance(p.accessKeyID, p.secretAccessKey, p.region, instanceID)

}

// StartContainer needs docs
func (p *AWSOrchestrationProvider) StartContainer(taskDefinition string, launchType, cluster, vpcName *string, securityGroupIds []string, subnetIds []string, overrides map[string]interface{}) (taskIds []string, err error) {
	return awswrapper.StartContainer(p.accessKeyID, p.secretAccessKey, p.region, taskDefinition, launchType, cluster, vpcName, securityGroupIds, subnetIds, overrides)

}

// StopContainer needs docs
func (p *AWSOrchestrationProvider) StopContainer(taskID string, cluster *string) (response *ecs.StopTaskOutput, err error) {
	return awswrapper.StopContainer(p.accessKeyID, p.secretAccessKey, p.region, taskID, cluster)

}

// GetContainerDetails needs docs
func (p *AWSOrchestrationProvider) GetContainerDetails(taskID string, cluster *string) (response *ecs.DescribeTasksOutput, err error) {
	return awswrapper.GetContainerDetails(p.accessKeyID, p.secretAccessKey, p.region, taskID, cluster)

}

// GetContainerLogEvents needs docs
func (p *AWSOrchestrationProvider) GetContainerLogEvents(taskID string, cluster *string, startFromHead bool, startTime, endTime, limit *int64, nextToken *string) (response *cloudwatchlogs.GetLogEventsOutput, err error) {
	return awswrapper.GetContainerLogEvents(p.accessKeyID, p.secretAccessKey, p.region, taskID, cluster, startFromHead, startTime, endTime, limit, nextToken)

}

// GetLogEvents needs docs
func (p *AWSOrchestrationProvider) GetLogEvents(logGroupID string, logStreamID string, startFromHead bool, startTime, endTime, limit *int64, nextToken *string) (response *cloudwatchlogs.GetLogEventsOutput, err error) {
	return awswrapper.GetLogEvents(p.accessKeyID, p.secretAccessKey, p.region, logGroupID, logStreamID, startFromHead, startTime, endTime, limit, nextToken)

}

// GetNetworkInterfaceDetails needs docs
func (p *AWSOrchestrationProvider) GetNetworkInterfaceDetails(networkInterfaceID string) (response *ec2.DescribeNetworkInterfacesOutput, err error) {
	return awswrapper.GetNetworkInterfaceDetails(p.accessKeyID, p.secretAccessKey, p.region, networkInterfaceID)
}
