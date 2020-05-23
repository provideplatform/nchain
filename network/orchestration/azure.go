package orchestration

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/route53"

	azurewrapper "github.com/kthomas/go-azure-wrapper"
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

// InitAzureOrchestrationProvider initializes and returns the Microsoft Azure infrastructure orchestration provider
func InitAzureOrchestrationProvider(credentials map[string]interface{}, region string) *AzureOrchestrationProvider {
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

func (p *AzureOrchestrationProvider) CreateLoadBalancer(vpcID *string, name *string, securityGroupIds []string, listeners []*elb.Listener) (response *elb.CreateLoadBalancerOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) DeleteLoadBalancer(name *string) (response *elb.DeleteLoadBalancerOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) GetLoadBalancers(loadBalancerName *string) (response *elb.DescribeLoadBalancersOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) CreateLoadBalancerV2(vpcID, name, balancerType *string, securityGroupIds []string) (response *elbv2.CreateLoadBalancerOutput, err error) {
	return nil, nil
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
	vpc := ""
	if (vpcID == nil) {
		vpc = "goldmine-vpc"
	}
	else {
		vpc = *vpcID
	}
	id, err := azurewrapper.UpsertResourceGroup(ctx.TODO(), "goldmine", vpc, region)
	return [*id], err
}

func (p *AzureOrchestrationProvider) DeleteSecurityGroup(securityGroupID string) (interface{}, error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) GetSecurityGroups() (response *ec2.DescribeSecurityGroupsOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) StartContainer(image, taskDefinition *string, taskRole, launchType, resourceGroupName, virtualNetworkID *string, cpu, memory *int64, entrypoint []*string, securityGroupIds []string, subnetIds []string, overrides, security map[string]interface{}) (taskIds []string, err error) {
	if resourceGroupName == nil {
		resourceGroupName = common.StringOrNil(fmt.Sprintf("prvd-%d", time.Now().Unix()))
	}
	
	params := &provide.ContainerParams{
		Region:            region,
		ResourceGroupName: resourceGroupName,
		Image:             image,
		VirtualNetworkID:  virtualNetworkID,
		Cpu:               cpu,
		Memory:            memory,
		Entrypoint:        entrypoint,
		SecurityGroupIds:  securityGroupIds,
		SubnetIds:         subnetIds,
		Environment:       overrides,
		Security:          security,
	}

	creds := &provide.TargetCredentials{
		AzureSubscriptionID: p.subscriptionID,
		// AzureTenantID
		// AzureClientID
		// AzureClientSecret
	}

	result := azurewrapper.StartContainer(params, creds)
	if result.Err != nil {
		panic(fmt.Sprintf("%s", result.Err.Error()))
	}
	println(fmt.Sprintf("container: %+v", result.ContainerIds))
	return result.ContainerIds, result.Err
}

func (p *AzureOrchestrationProvider) StopContainer(taskID string, cluster *string) (response *ecs.StopTaskOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) GetContainerDetails(taskID string, cluster *string) (response *ecs.DescribeTasksOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) GetContainerInterfaces(taskID string, cluster *string) ([]*NetworkInterface, error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) GetContainerLogEvents(taskID string, cluster *string, startFromHead bool, startTime, endTime, limit *int64, nextToken *string) (response *cloudwatchlogs.GetLogEventsOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) GetLogEvents(logGroupID string, logStreamID string, startFromHead bool, startTime, endTime, limit *int64, nextToken *string) (response *cloudwatchlogs.GetLogEventsOutput, err error) {
	return nil, nil
}

func (p *AzureOrchestrationProvider) GetNetworkInterfaceDetails(networkInterfaceID string) (response *ec2.DescribeNetworkInterfacesOutput, err error) {
	return nil, nil
}
