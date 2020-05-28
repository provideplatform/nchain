package orchestration

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/route53"

	azurewrapper "github.com/kthomas/go-azure-wrapper"
	provide "github.com/provideservices/provide-go"

	"github.com/provideapp/goldmine/common"
)

// AzureOrchestrationProvider is a network.orchestration.API implementing the Azure API
type AzureOrchestrationProvider struct {
	region         string
	tenantID       string
	subscriptionID string
	clientID       string
	clientSecret   string
}

var networkInterfaces map[string]*provide.NetworkInterface

// InitAzureOrchestrationProvider initializes and returns the Microsoft Azure infrastructure orchestration provider
func InitAzureOrchestrationProvider(credentials map[string]interface{}, region string) *AzureOrchestrationProvider {
	networkInterfaces = make(map[string]*provide.NetworkInterface)

	tenantID, tenantIDOk := credentials["azure_tenant_id"].(string)
	subscriptionID, subscriptionIDOk := credentials["azure_subscription_id"].(string)
	clientID, clientIDOk := credentials["azure_client_id"].(string)
	clientSecret, clientSecretOk := credentials["azure_client_secret"].(string)

	if !tenantIDOk || !subscriptionIDOk || !clientIDOk || !clientSecretOk {
		common.Log.Warning("Failed to initialize Azure orchestration API provider; tenant_id, subscription_id, client_id and client_secret are all required credentials")
		return nil
	}

	return &AzureOrchestrationProvider{
		region:         region,
		tenantID:       tenantID,
		subscriptionID: subscriptionID,
		clientID:       clientID,
		clientSecret:   clientSecret,
	}
}

func (p *AzureOrchestrationProvider) targetCredentials() *provide.TargetCredentials {
	return &provide.TargetCredentials{
		AzureTenantID:       common.StringOrNil(p.tenantID),
		AzureSubscriptionID: common.StringOrNil(p.subscriptionID),
		AzureClientID:       common.StringOrNil(p.clientID),
		AzureClientSecret:   common.StringOrNil(p.clientSecret),
	}
}

func (p *AzureOrchestrationProvider) CreateLoadBalancer(vpcID *string, name *string, securityGroupIds []string, listeners []*elb.Listener) (response *elb.CreateLoadBalancerOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) DeleteLoadBalancer(name *string) (response *elb.DeleteLoadBalancerOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) GetLoadBalancers(loadBalancerName *string) (response *elb.DescribeLoadBalancersOutput, err error) {
	return nil, nil
}

//CreateLoadBalancerV2 creates load balancer on Azure
func (p *AzureOrchestrationProvider) CreateLoadBalancerV2(vpcID, name, balancerType *string, securityGroupIds []string) (response *elbv2.CreateLoadBalancerOutput, err error) {
	// returns empty array for azure showcase
	return &elbv2.CreateLoadBalancerOutput{
		LoadBalancers: []*elbv2.LoadBalancer{}}, nil
	// elbv2.LoadBalancer{}

}

func (p *AzureOrchestrationProvider) CreateListenerV2(loadBalancerARN, targetGroupARN, protocol *string, port *int64, certificate interface{}) (*elbv2.CreateListenerOutput, error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) DeleteLoadBalancerV2(loadBalancerARN *string) (response *elbv2.DeleteLoadBalancerOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) GetLoadBalancersV2(loadBalancerArn *string, loadBalancerName *string, nextMarker *string) (response *elbv2.DescribeLoadBalancersOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) GetTargetGroup(targetGroupName string) (response *elbv2.DescribeTargetGroupsOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) CreateTargetGroup(vpcID *string, name, protocol *string, port int64, healthCheckPort, healthCheckStatusCode *int64, healthCheckPath *string) (response *elbv2.CreateTargetGroupOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) DeleteTargetGroup(targetGroupARN *string) (response *elbv2.DeleteTargetGroupOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) RegisterTarget(targetGroupARN, ipAddress *string, port *int64) (response *elbv2.RegisterTargetsOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) DeregisterTarget(targetGroupARN, ipAddress *string, port *int64) (response *elbv2.DeregisterTargetsOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) CreateDNSRecord(hostedZoneID, name, recordType string, value []string, ttl int64) (response *route53.ChangeResourceRecordSetsOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) DeleteDNSRecord(hostedZoneID, name, recordType string, value []string, ttl int64) (response *route53.ChangeResourceRecordSetsOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) ImportSelfSignedCertificate(dnsNames []string, certificateARN *string) (*acm.ImportCertificateOutput, error) {
	return nil, nil
}
func (p *AzureOrchestrationProvider) DeleteCertificate(certificateARN *string) (response *acm.DeleteCertificateOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) CreateDefaultSubnets(vpcID string) ([]*ec2.CreateDefaultSubnetOutput, error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) GetVPCs(vpcID *string) (response *ec2.DescribeVpcsOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) GetSubnets(vpcID *string) (response *ec2.DescribeSubnetsOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) GetClusters() (response *ecs.ListClustersOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) AuthorizeSecurityGroupEgress(securityGroupID, ipv4Cidr string, tcpPorts, udpPorts []int64) (response *ec2.AuthorizeSecurityGroupEgressOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) AuthorizeSecurityGroupEgressAllPortsAllProtocols(securityGroupID string) (response *ec2.AuthorizeSecurityGroupEgressOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) AuthorizeSecurityGroupIngressAllPortsAllProtocols(securityGroupID string) (response *ec2.AuthorizeSecurityGroupIngressOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) AuthorizeSecurityGroupIngress(securityGroupID, ipv4Cidr string, tcpPorts, udpPorts []int64) (response *ec2.AuthorizeSecurityGroupIngressOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) CreateSecurityGroup(name, description string, vpcID *string, cfg map[string]interface{}) ([]string, error) {
	return []string{}, nil
}

func (p *AzureOrchestrationProvider) DeleteSecurityGroup(securityGroupID string) (interface{}, error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) GetSecurityGroups() (response *ec2.DescribeSecurityGroupsOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) StartContainer(
	image, taskDefinition *string,
	taskRole, launchType, resourceGroupName, virtualNetworkID *string,
	cpu, memory *int64,
	entrypoint []*string,
	securityGroupIds []string,
	subnetIds []string,
	overrides, security map[string]interface{},
) (taskIds []string, err error) {
	if resourceGroupName == nil {
		resourceGroupName = common.StringOrNil(fmt.Sprintf("prvd-0"))
	}

	_, err = azurewrapper.UpsertResourceGroup(context.TODO(), p.targetCredentials(), p.region, *resourceGroupName)
	if err != nil {
		common.Log.Warning(fmt.Sprintf("Failed to create Azure security group: %s", err.Error()))
		return []string{}, err
	}

	containerCPU := cpu
	if containerCPU == nil {
		ccpu := int64(2)
		containerCPU = &ccpu
	}

	containerMemory := memory
	if containerMemory == nil {
		cmem := int64(4)
		containerMemory = &cmem
	}

	params := &provide.ContainerParams{
		Region:            p.region,
		ResourceGroupName: *resourceGroupName,
		Image:             image,
		VirtualNetworkID:  virtualNetworkID,
		CPU:               containerCPU,
		Memory:            containerMemory,
		Entrypoint:        entrypoint,
		SecurityGroupIds:  securityGroupIds,
		SubnetIds:         subnetIds,
		Environment:       overrides,
		Security:          security,
	}

	result, err := azurewrapper.StartContainer(params, p.targetCredentials())
	if err != nil {
		return taskIds, err
	}

	id := result.ContainerIds[0]

	networkInterfaces[id] = result.ContainerInterfaces[0]
	common.Log.Debugf("StartContainer: Receiving network interface with values; %+v", networkInterfaces)

	return result.ContainerIds, err
}

// StopContainer
func (p *AzureOrchestrationProvider) StopContainer(taskID string, cluster *string) (response *ecs.StopTaskOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) GetContainerDetails(taskID string, cluster *string) (response *ecs.DescribeTasksOutput, err error) {
	// todo
	return nil, nil
}

func (p *AzureOrchestrationProvider) GetContainerInterfaces(taskID string, cluster *string) ([]*provide.NetworkInterface, error) {
	// todo
	common.Log.Debugf("GetContainerInterfaces: Receiving network interface request for id: %s", taskID)

	ints := make([]*provide.NetworkInterface, 1)
	i := networkInterfaces[taskID]
	common.Log.Debugf("GetContainerInterfaces: Receiving network interface: %+v", i)
	common.Log.Debugf("GetContainerInterfaces: Current network interfaces: %+v", networkInterfaces)
	if i == nil {
		common.Log.Debugf("GetContainerInterfaces: sending empty response")
		return []*provide.NetworkInterface{}, nil
	}
	ints[0] = i
	common.Log.Debugf("GetContainerInterfaces: sending : %+v", ints)
	return ints, nil
}

func (p *AzureOrchestrationProvider) GetContainerLogEvents(taskID string, cluster *string, startFromHead bool, startTime, endTime, limit *int64, nextToken *string) (response *cloudwatchlogs.GetLogEventsOutput, err error) {
	// todo
	return nil, nil
}

func (p *AzureOrchestrationProvider) GetLogEvents(logGroupID string, logStreamID string, startFromHead bool, startTime, endTime, limit *int64, nextToken *string) (response *cloudwatchlogs.GetLogEventsOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) GetNetworkInterfaceDetails(networkInterfaceID string) (response *ec2.DescribeNetworkInterfacesOutput, err error) {
	// todo
	return nil, nil
}
