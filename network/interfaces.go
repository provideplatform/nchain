package network

import (
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/route53"
)

const awsOrchestrationProvider = "aws"
const azureOrchestrationProvider = "azure"
const googleOrchestrationProvider = "gcp"

const bcoinP2PProvider = "bcoin"
const parityP2PProvider = "parity"
const quorumP2PProvider = "quorum"

// OrchestrationAPI defines an interface for implementations to orchestrate cloud or on-premise infrastructure
type OrchestrationAPI interface {
	CreateLoadBalancer(vpcID *string, name *string, securityGroupIds []string, listeners []*elb.Listener) (response *elb.CreateLoadBalancerOutput, err error)
	DeleteLoadBalancer(name *string) (response *elb.DeleteLoadBalancerOutput, err error)
	GetLoadBalancers(loadBalancerName *string) (response *elb.DescribeLoadBalancersOutput, err error)

	CreateLoadBalancerV2(vpcID, name, balancerType *string, securityGroupIds []string) (response *elbv2.CreateLoadBalancerOutput, err error)
	CreateListenerV2(loadBalancerARN, targetGroupARN, protocol *string, port *int64, certificate interface{}) (*elbv2.CreateListenerOutput, error)
	DeleteLoadBalancerV2(loadBalancerARN *string) (response *elbv2.DeleteLoadBalancerOutput, err error)
	GetLoadBalancersV2(loadBalancerArn *string, loadBalancerName *string, nextMarker *string) (response *elbv2.DescribeLoadBalancersOutput, err error)

	GetTargetGroup(targetGroupName string) (response *elbv2.DescribeTargetGroupsOutput, err error)
	CreateTargetGroup(vpcID *string, name, protocol *string, port int64, healthCheckPort, healthCheckStatusCode *int64, healthCheckPath *string) (response *elbv2.CreateTargetGroupOutput, err error)
	DeleteTargetGroup(targetGroupARN *string) (response *elbv2.DeleteTargetGroupOutput, err error)
	RegisterTarget(targetGroupARN, ipAddress *string, port *int64) (response *elbv2.RegisterTargetsOutput, err error)
	DeregisterTarget(targetGroupARN, ipAddress *string, port *int64) (response *elbv2.DeregisterTargetsOutput, err error)

	CreateDNSRecord(hostedZoneID, name, recordType string, value []string, ttl int64) (response *route53.ChangeResourceRecordSetsOutput, err error)
	DeleteDNSRecord(hostedZoneID, name, recordType string, value []string, ttl int64) (response *route53.ChangeResourceRecordSetsOutput, err error)

	ImportSelfSignedCertificate(dnsNames []string, certificateARN *string) (*acm.ImportCertificateOutput, error)
	DeleteCertificate(certificateARN *string) (response *acm.DeleteCertificateOutput, err error)

	CreateDefaultSubnets(vpcID string) ([]*ec2.CreateDefaultSubnetOutput, error)
	GetVPCs(vpcID *string) (response *ec2.DescribeVpcsOutput, err error)
	GetSubnets(vpcID *string) (response *ec2.DescribeSubnetsOutput, err error)
	GetClusters() (response *ecs.ListClustersOutput, err error)

	AuthorizeSecurityGroupEgress(securityGroupID, ipv4Cidr string, tcpPorts, udpPorts []int64) (response *ec2.AuthorizeSecurityGroupEgressOutput, err error)
	AuthorizeSecurityGroupEgressAllPortsAllProtocols(securityGroupID string) (response *ec2.AuthorizeSecurityGroupEgressOutput, err error)
	AuthorizeSecurityGroupIngressAllPortsAllProtocols(securityGroupID string) (response *ec2.AuthorizeSecurityGroupIngressOutput, err error)
	AuthorizeSecurityGroupIngress(securityGroupID, ipv4Cidr string, tcpPorts, udpPorts []int64) (response *ec2.AuthorizeSecurityGroupIngressOutput, err error)
	CreateSecurityGroup(name, description string, vpcID *string) (response *ec2.CreateSecurityGroupOutput, err error)
	DeleteSecurityGroup(securityGroupID string) (response *ec2.DeleteSecurityGroupOutput, err error)
	GetSecurityGroups() (response *ec2.DescribeSecurityGroupsOutput, err error)

	StartContainer(image, taskDefinition *string, launchType, cluster, vpcName *string, securityGroupIds []string, subnetIds []string, overrides, security map[string]interface{}) (taskIds []string, err error)
	StopContainer(taskID string, cluster *string) (response *ecs.StopTaskOutput, err error)
	GetContainerDetails(taskID string, cluster *string) (response *ecs.DescribeTasksOutput, err error)
	GetContainerLogEvents(taskID string, cluster *string, startFromHead bool, startTime, endTime, limit *int64, nextToken *string) (response *cloudwatchlogs.GetLogEventsOutput, err error)
	GetLogEvents(logGroupID string, logStreamID string, startFromHead bool, startTime, endTime, limit *int64, nextToken *string) (response *cloudwatchlogs.GetLogEventsOutput, err error)
	GetNetworkInterfaceDetails(networkInterfaceID string) (response *ec2.DescribeNetworkInterfacesOutput, err error)

	// AuthorizeSecurityGroupIngress(*securityGroupID, cidr, tcp, udp) (interface{}, error)
	// AuthorizeSecurityGroupEgressAllPortsAllProtocols(*securityGroup.GroupId) (interface{}, error)
	// AuthorizeSecurityGroupEgress(*securityGroup.GroupId, cidr, tcp, udp) (interface{}, error)
	// CreateLoadBalancerV2(common.StringOrNil(vpcID), l.Name, common.StringOrNil("application"), securityGroupIds) (interface{}, error)
	// CreateListenerV2(common.StringOrNil(targetBalancerArn), common.StringOrNil(targetGroupArn), common.StringOrNil("HTTP"), &port) (interface{}, error)
	// CreateSecurityGroup(securityGroupDesc, securityGroupDesc, common.StringOrNil(vpcID)) (interface{}, error)
	// CreateTargetGroup(common.StringOrNil(vpcID), common.StringOrNil(targetGroupName), common.StringOrNil("HTTP"), tcpPort) (interface{}, error)
	// DeleteLoadBalancerV2(common.StringOrNil(targetBalancerArn)) (interface{}, error)
	// DeleteTargetGroup(common.StringOrNil(targetGroupArn.(string))) (interface{}, error)
	// DeleteSecurityGroup(securityGroupID.(string)) (interface{}, error)
	// DeregisterTarget(common.StringOrNil(targetGroupArn.(string)), node.PrivateIPv4, &port) (interface{}, error)
	// GetContainerDetails(id, nil) (interface{}, error)
	// GetContainerLogEvents(id.(string), nil, startFromHead, nil, nil, limit, nextToken) (interface{}, error)
	// GetNetworkInterfaceDetails(*kvp.Value) (interface{}, error)
	// GetSecurityGroups(accessKeyID, secretAccessKey, region) (interface{}, error)
	// RegisterTarget(common.StringOrNil(targetGroupArn), node.PrivateIPv4, &port) (interface{}, error)
	// StartContainer(*resolvedContainer, nil, nil, common.StringOrNil(vpc), securityGroupIds, []string{}, overrides) (interface{}, error)
	// StopContainer(taskID, nil) (interface{}, error)
}

// P2PAPI defines an interface for p2p network implementations
type P2PAPI interface {
	AcceptNonReservedPeers() error
	DropNonReservedPeers() error
	AddPeer(string) error
	RemovePeer(string) error
	// TODO: ResolvePeerURL() error
	Upgrade() error
}
