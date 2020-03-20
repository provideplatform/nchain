package common

import (
	"os"
	"strings"
	"time"

	awsconf "github.com/kthomas/go-aws-config"
	"github.com/kthomas/go-logger"
)

const defaultDockerhubOrganization = "provide"
const reachabilityTimeout = time.Millisecond * 2500

var (
	// Log is the default package logger
	Log *logger.Logger

	// DefaultAWSConfig is the default Amazon Web Services API config
	DefaultAWSConfig *awsconf.Config

	// ListenAddr is the http server listen address
	ListenAddr string

	// CertificatePath is the SSL certificate path used by HTTPS listener
	CertificatePath string
	// PrivateKeyPath is the private key used by HTTPS listener
	PrivateKeyPath string

	requireTLS bool

	// DefaultHTTPPort is the default http port, i.e., for json-rpc or rest api listeners
	DefaultHTTPPort = 8545

	// DefaultPeerDiscoveryPort is the default port for p2p discovery
	DefaultPeerDiscoveryPort = 30303

	// DefaultWebsocketPort is the default websocket port
	DefaultWebsocketPort = 8546

	// TxFilters contains in-memory Filter instances used for real-time stream processing
	TxFilters = map[string][]interface{}{}

	// ConsumeNATSStreamingSubscriptions is a flag the indicates if the goldmine instance is running in API or consumer mode
	ConsumeNATSStreamingSubscriptions bool

	// DefaultDockerhubOrganization is the default public Dockerhub organization to leverage when resolving repository names
	DefaultDockerhubOrganization *string

	// DefaultInfrastructureDomain is the DNS name which managed subdomains are created for various infrastructure (i.e., load balancers)
	DefaultInfrastructureDomain string

	// DefaultInfrastructureRoute53HostedZoneID is the Route53 hosted zone which is used to create/destroy managed subdomains for various infrastructure (i.e., load balancers)
	DefaultInfrastructureRoute53HostedZoneID string

	// DefaultInfrastructureAWSConfig is the AWS configuration which is used to support various managed infrastructure (i.e., load balancers)
	DefaultInfrastructureAWSConfig *awsconf.Config

	// DefaultInfrastructureAzureRegion is the default Azure region configuration which is used to support various managed infrastructure (i.e., load balancers)
	DefaultInfrastructureAzureRegion *string // FIXME-- rename like awsconf

	// DefaultInfrastructureUsesSelfSignedCertificate is a flag that indicates if various managed infrastructure (i.e., load balancers) should use a self-signed cert
	DefaultInfrastructureUsesSelfSignedCertificate bool
)

func init() {
	ListenAddr = os.Getenv("LISTEN_ADDR")
	if ListenAddr == "" {
		ListenAddr = buildListenAddr()
	}

	requireTLS = os.Getenv("REQUIRE_TLS") == "true"

	lvl := os.Getenv("LOG_LEVEL")
	if lvl == "" {
		lvl = "INFO"
	}
	Log = logger.NewLogger("goldmine", lvl, true)

	DefaultAWSConfig = awsconf.GetConfig()

	ConsumeNATSStreamingSubscriptions = strings.ToLower(os.Getenv("CONSUME_NATS_STREAMING_SUBSCRIPTIONS")) == "true"

	requireInfrastructureSupport()
}

func requireInfrastructureSupport() {
	if os.Getenv("DEFAULT_DOCKERHUB_ORGANIZATION") != "" {
		DefaultDockerhubOrganization = StringOrNil(os.Getenv("DEFAULT_DOCKERHUB_ORGANIZATION"))
	} else {
		DefaultDockerhubOrganization = StringOrNil(defaultDockerhubOrganization)
	}

	if os.Getenv("INFRASTRUCTURE_DOMAIN") != "" {
		DefaultInfrastructureDomain = os.Getenv("INFRASTRUCTURE_DOMAIN")
	}

	if os.Getenv("INFRASTRUCTURE_ROUTE53_HOSTED_ZONE_ID") != "" {
		DefaultInfrastructureRoute53HostedZoneID = os.Getenv("INFRASTRUCTURE_ROUTE53_HOSTED_ZONE_ID")
	}

	var awsAccessKeyID string
	if os.Getenv("INFRASTRUCTURE_AWS_ACCESS_KEY_ID") != "" {
		awsAccessKeyID = os.Getenv("INFRASTRUCTURE_AWS_ACCESS_KEY_ID")
	}

	var awsSecretAccessKey string
	if os.Getenv("INFRASTRUCTURE_AWS_SECRET_ACCESS_KEY") != "" {
		awsSecretAccessKey = os.Getenv("INFRASTRUCTURE_AWS_SECRET_ACCESS_KEY")
	}

	var awsDefaultRegion string
	if os.Getenv("INFRASTRUCTURE_AWS_DEFAULT_REGION") != "" {
		awsDefaultRegion = os.Getenv("INFRASTRUCTURE_AWS_DEFAULT_REGION")
	}

	var awsDefaultCertificateArn string
	if os.Getenv("INFRASTRUCTURE_AWS_DEFAULT_CERTIFICATE_ARN") != "" {
		awsDefaultCertificateArn = os.Getenv("INFRASTRUCTURE_AWS_DEFAULT_CERTIFICATE_ARN")
	}

	if awsAccessKeyID != "" && awsSecretAccessKey != "" && awsDefaultRegion != "" && awsDefaultCertificateArn != "" {
		DefaultInfrastructureAWSConfig = &awsconf.Config{
			AccessKeyId:           &awsAccessKeyID,
			SecretAccessKey:       &awsSecretAccessKey,
			DefaultRegion:         &awsDefaultRegion,
			DefaultCertificateArn: &awsDefaultCertificateArn,
		}
	}

	DefaultInfrastructureUsesSelfSignedCertificate = !(DefaultInfrastructureDomain != "" && DefaultInfrastructureRoute53HostedZoneID != "" && DefaultInfrastructureAWSConfig != nil && DefaultInfrastructureAWSConfig.DefaultCertificateArn != nil && *DefaultInfrastructureAWSConfig.DefaultCertificateArn != "")
}
