package orchestration

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/route53"
	awswrapper "github.com/kthomas/go-aws-wrapper"
	"github.com/provideapp/nchain/common"
	provide "github.com/provideservices/provide-go/api/c2"
)

const awsTaskStatusRunning = "running"

const awsAttachmentElasticNetworkInterface = "ElasticNetworkInterface"
const awsElasticNetworkInterfaceId = "networkInterfaceId"

// AWSOrchestrationProvider is a network.orchestration.API implementing the AWS API
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
func (p *AWSOrchestrationProvider) CreateLoadBalancer(vpcID *string, name *string, securityGroupIDs []string, listeners []*elb.Listener) (response *elb.CreateLoadBalancerOutput, err error) {
	return awswrapper.CreateLoadBalancer(p.accessKeyID, p.secretAccessKey, p.region, vpcID, name, securityGroupIDs, listeners)

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
func (p *AWSOrchestrationProvider) CreateLoadBalancerV2(vpcID, name, balancerType *string, securityGroupIDs []string) (response *elbv2.CreateLoadBalancerOutput, err error) {
	return awswrapper.CreateLoadBalancerV2(p.accessKeyID, p.secretAccessKey, p.region, vpcID, name, balancerType, securityGroupIDs)
}

// CreateDefaultSubnets needs docs
func (p *AWSOrchestrationProvider) CreateDefaultSubnets(vpcID string) ([]*ec2.CreateDefaultSubnetOutput, error) {
	return awswrapper.CreateDefaultSubnets(p.accessKeyID, p.secretAccessKey, p.region, vpcID)

}

// CreateListenerV2 needs docs
func (p *AWSOrchestrationProvider) CreateListenerV2(loadBalancerARN, targetGroupARN, protocol *string, port *int64, certificate interface{}) (*elbv2.CreateListenerOutput, error) {
	return awswrapper.CreateListenerV2(p.accessKeyID, p.secretAccessKey, p.region, loadBalancerARN, targetGroupARN, protocol, port, certificate)

}

// DeleteLoadBalancerV2 needs docs
func (p *AWSOrchestrationProvider) DeleteLoadBalancerV2(loadBalancerARN *string) (response *elbv2.DeleteLoadBalancerOutput, err error) {
	return awswrapper.DeleteLoadBalancerV2(p.accessKeyID, p.secretAccessKey, p.region, loadBalancerARN)

}

// GetLoadBalancersV2 needs docs
func (p *AWSOrchestrationProvider) GetLoadBalancersV2(loadBalancerArn *string, loadBalancerName *string, nextMarker *string) (response *elbv2.DescribeLoadBalancersOutput, err error) {
	return awswrapper.GetLoadBalancersV2(p.accessKeyID, p.secretAccessKey, p.region, loadBalancerArn, loadBalancerName, nextMarker)

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

// CreateDNSRecord needs docs
func (p *AWSOrchestrationProvider) CreateDNSRecord(hostedZoneID, name, recordType string, value []string, ttl int64) (response *route53.ChangeResourceRecordSetsOutput, err error) {
	return awswrapper.CreateDNSRecord(p.accessKeyID, p.secretAccessKey, p.region, hostedZoneID, name, recordType, value, ttl)
}

// DeleteDNSRecord needs docs
func (p *AWSOrchestrationProvider) DeleteDNSRecord(hostedZoneID, name, recordType string, value []string, ttl int64) (response *route53.ChangeResourceRecordSetsOutput, err error) {
	return awswrapper.DeleteDNSRecord(p.accessKeyID, p.secretAccessKey, p.region, hostedZoneID, name, recordType, value, ttl)
}

// ImportSelfSignedCertificate needs docs
func (p *AWSOrchestrationProvider) ImportSelfSignedCertificate(dnsNames []string, certificateARN *string) (*acm.ImportCertificateOutput, error) {
	return awswrapper.ImportSelfSignedCertificate(p.accessKeyID, p.secretAccessKey, p.region, dnsNames, certificateARN)
}

// DeleteCertificate needs docs
func (p *AWSOrchestrationProvider) DeleteCertificate(certificateARN *string) (response *acm.DeleteCertificateOutput, err error) {
	return awswrapper.DeleteCertificate(p.accessKeyID, p.secretAccessKey, p.region, certificateARN)
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
func (p *AWSOrchestrationProvider) CreateSecurityGroup(name, description string, vpcID *string, cfg map[string]interface{}) ([]string, error) {
	securityGroupIDs := make([]string, 0)
	securityGroup, err := awswrapper.CreateSecurityGroup(p.accessKeyID, p.secretAccessKey, p.region, name, description, vpcID)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "InvalidGroup.Duplicate":
				common.Log.Debugf("Security group %s already exists in EC2 region %s", description, p.region)
				securityGroups, gerr := p.GetSecurityGroups()
				if gerr == nil {
					for _, secGroup := range securityGroups.SecurityGroups {
						if secGroup.GroupName != nil && *secGroup.GroupName == description && secGroup.GroupId != nil {
							securityGroupIDs = append(securityGroupIDs, *secGroup.GroupId)
							break
						}
					}
				}
			default:
				desc := fmt.Sprintf("Failed to create security group in EC2 region %s; %s", p.region, err.Error())
				common.Log.Warning(desc)
				return nil, errors.New(desc)
			}
		}
	} else {
		if securityGroup != nil {
			securityGroupIDs = append(securityGroupIDs, *securityGroup.GroupId)
		}

		if egress, egressOk := cfg["egress"]; egressOk {
			switch egress.(type) {
			case string:
				if egress.(string) == "*" {
					_, err := p.AuthorizeSecurityGroupEgressAllPortsAllProtocols(*securityGroup.GroupId)
					if err != nil {
						if aerr, ok := err.(awserr.Error); ok {
							switch aerr.Code() {
							case "InvalidPermission.Duplicate":
								common.Log.Debugf("Attempted to authorize duplicate security group egress across all ports and protocols in EC2 %s region; security group id: %s", p.region, *securityGroup.GroupId)
							default:
								common.Log.Warningf("Failed to authorize security group egress across all ports and protocols in EC2 %s region; security group id: %s; %s", p.region, *securityGroup.GroupId, err.Error())
							}
						}
					}
				}
			case map[string]interface{}:
				egressCfg := egress.(map[string]interface{})
				for cidr := range egressCfg {
					tcp := make([]int64, 0)
					if _tcp, tcpOk := egressCfg[cidr].(map[string]interface{})["tcp"].([]interface{}); tcpOk {
						for i := range _tcp {
							tcp = append(tcp, int64(_tcp[i].(float64)))
						}
					}

					udp := make([]int64, 0)
					if _udp, udpOk := egressCfg[cidr].(map[string]interface{})["udp"].([]interface{}); udpOk {
						for i := range _udp {
							udp = append(udp, int64(_udp[i].(float64)))
						}
					}

					_, err := p.AuthorizeSecurityGroupEgress(*securityGroup.GroupId, cidr, tcp, udp)
					if err != nil {
						if aerr, ok := err.(awserr.Error); ok {
							switch aerr.Code() {
							case "InvalidPermission.Duplicate":
								common.Log.Debugf("Attempted to authorize duplicate security group egress in EC2 %s region; security group id: %s; tcp ports: %d; udp ports: %d", p.region, *securityGroup.GroupId, tcp, udp)
							default:
								common.Log.Warningf("Failed to authorize security group egress in EC2 %s region; security group id: %s; tcp ports: %d; udp ports: %d; %s", p.region, *securityGroup.GroupId, tcp, udp, err.Error())
							}
						}
					}
				}
			}
		}

		if ingress, ingressOk := cfg["ingress"]; ingressOk {
			switch ingress.(type) {
			case string:
				if ingress.(string) == "*" {
					_, err := p.AuthorizeSecurityGroupIngressAllPortsAllProtocols(*securityGroup.GroupId)
					if err != nil {
						if aerr, ok := err.(awserr.Error); ok {
							switch aerr.Code() {
							case "InvalidPermission.Duplicate":
								common.Log.Debugf("Attempted to authorize duplicate security group ingress across all ports and protocols in EC2 %s region; security group id: %s", p.region, *securityGroup.GroupId)
							default:
								common.Log.Warningf("Failed to authorize security group ingress across all ports and protocols in EC2 %s region; security group id: %s; %s", p.region, *securityGroup.GroupId, err.Error())
							}
						}
					}
				}
			case map[string]interface{}:
				ingressCfg := ingress.(map[string]interface{})
				for cidr := range ingressCfg {
					tcp := make([]int64, 0)
					if _tcp, tcpOk := ingressCfg[cidr].(map[string]interface{})["tcp"].([]interface{}); tcpOk {
						for i := range _tcp {
							tcp = append(tcp, int64(_tcp[i].(float64)))
						}
					}

					udp := make([]int64, 0)
					if _udp, udpOk := ingressCfg[cidr].(map[string]interface{})["udp"].([]interface{}); udpOk {
						for i := range _udp {
							udp = append(udp, int64(_udp[i].(float64)))
						}
					}

					_, err := p.AuthorizeSecurityGroupIngress(*securityGroup.GroupId, cidr, tcp, udp)
					if err != nil {
						if aerr, ok := err.(awserr.Error); ok {
							switch aerr.Code() {
							case "InvalidPermission.Duplicate":
								common.Log.Debugf("Attempted to authorize duplicate security group ingress in EC2 %s region; security group id: %s; tcp ports: %d; udp ports: %d", p.region, *securityGroup.GroupId, tcp, udp)
							default:
								common.Log.Warningf("Failed to authorize security group ingress in EC2 %s region; security group id: %s; tcp ports: %d; udp ports: %d; %s", p.region, *securityGroup.GroupId, tcp, udp, err.Error())
							}
						}
					}
				}
			}
		}
	}

	return securityGroupIDs, nil
}

// DeleteSecurityGroup needs docs
func (p *AWSOrchestrationProvider) DeleteSecurityGroup(securityGroupID string) (interface{}, error) {
	resp, err := awswrapper.DeleteSecurityGroup(p.accessKeyID, p.secretAccessKey, p.region, securityGroupID)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "InvalidGroup.NotFound":
				common.Log.Debugf("Attempted to unregister security group which does not exist; security group id: %s", securityGroupID)
			default:
				return resp, err
			}
		}
	}

	return resp, nil
}

// SetInstanceSecurityGroups needs docs
func (p *AWSOrchestrationProvider) SetInstanceSecurityGroups(instanceID string, securityGroupIDs []string) (response *ec2.ModifyInstanceAttributeOutput, err error) {
	return awswrapper.SetInstanceSecurityGroups(p.accessKeyID, p.secretAccessKey, p.region, instanceID, securityGroupIDs)

}

// TerminateInstance needs docs
func (p *AWSOrchestrationProvider) TerminateInstance(instanceID string) (response *ec2.TerminateInstancesOutput, err error) {
	return awswrapper.TerminateInstance(p.accessKeyID, p.secretAccessKey, p.region, instanceID)

}

// StartContainer needs docs
func (p *AWSOrchestrationProvider) StartContainer(image, taskDefinition *string, taskRole, launchType, cluster, vpcName *string, cpu, memory *int64, entrypoint []*string, securityGroupIDs []string, subnetIds []string, overrides map[string]interface{}, security map[string]interface{}) (taskIds []string, networkInterfaces []*provide.NetworkInterface, err error) {
	taskIds, err = awswrapper.StartContainer(p.accessKeyID, p.secretAccessKey, p.region, image, taskDefinition, taskRole, taskRole, launchType, cluster, vpcName, cpu, memory, entrypoint, securityGroupIDs, subnetIds, overrides, security)
	return taskIds, networkInterfaces, err
}

// StopContainer needs docs
func (p *AWSOrchestrationProvider) StopContainer(taskID string, cluster *string) (response *ecs.StopTaskOutput, err error) {
	return awswrapper.StopContainer(p.accessKeyID, p.secretAccessKey, p.region, taskID, cluster)
}

// GetContainerDetails needs docs
func (p *AWSOrchestrationProvider) GetContainerDetails(taskID string, cluster *string) (response *ecs.DescribeTasksOutput, err error) {
	return awswrapper.GetContainerDetails(p.accessKeyID, p.secretAccessKey, p.region, taskID, cluster)

}

// GetContainerInterfaces retrieves the container interfaces
func (p *AWSOrchestrationProvider) GetContainerInterfaces(taskID string, cluster *string) ([]*provide.NetworkInterface, error) {
	interfaces := make([]*provide.NetworkInterface, 0)

	containerDetails, err := p.GetContainerDetails(taskID, nil)
	if err != nil {
		return nil, err
	}

	if len(containerDetails.Tasks) > 0 {
		task := containerDetails.Tasks[0] // FIXME-- should this support exposing all tasks?
		taskStatus := ""
		if task.LastStatus != nil {
			taskStatus = strings.ToLower(*task.LastStatus)
		}
		if taskStatus != awsTaskStatusRunning && task.StoppedAt != nil {
			return nil, fmt.Errorf("Unable to resolve network interfaces for container status: %s; task id: %s stopped at %s", taskStatus, taskID, *task.StoppedAt)
		}

		if len(task.Attachments) > 0 {
			attachment := task.Attachments[0]
			if attachment.Type != nil && *attachment.Type == awsAttachmentElasticNetworkInterface {
				for i := range attachment.Details {
					kvp := attachment.Details[i]
					if kvp.Name != nil && *kvp.Name == awsElasticNetworkInterfaceId && kvp.Value != nil {
						interfaceDetails, err := p.GetNetworkInterfaceDetails(*kvp.Value)
						if err == nil {
							for _, netInterface := range interfaceDetails.NetworkInterfaces {
								networkInterface := &provide.NetworkInterface{
									PrivateIPv4: netInterface.PrivateIpAddress,
								}

								if netInterface.Association != nil {
									networkInterface.Host = netInterface.Association.PublicDnsName
									networkInterface.IPv4 = netInterface.Association.PublicIp
								}

								interfaces = append(interfaces, networkInterface)
							}
						} else {
							return nil, err
						}
					}
				}
			}
		}
	}

	common.Log.Debugf("Resolved %d network interfaces for container with task id: %s", len(interfaces), taskID)
	return interfaces, nil
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
