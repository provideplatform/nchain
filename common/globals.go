/*
 * Copyright 2017-2022 Provide Technologies Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	awsconf "github.com/kthomas/go-aws-config"
	"github.com/kthomas/go-logger"

	ident "github.com/provideplatform/provide-go/api/ident"
	vault "github.com/provideplatform/provide-go/api/vault"
	util "github.com/provideplatform/provide-go/common/util"
)

const defaultDockerhubOrganization = "provide"
const reachabilityTimeout = time.Millisecond * 2500

const refreshTokenTickInterval = 60000 * 45 * time.Millisecond  // 45 minutes
const refreshTokenSleepInterval = 60000 * 10 * time.Millisecond // 10 minutes

var (
	// Log is the default package logger
	Log *logger.Logger

	// DefaultAWSConfig is the default Amazon Web Services API config
	DefaultAWSConfig *awsconf.Config

	// DefaultHTTPPort is the default http port, i.e., for json-rpc or rest api listeners
	DefaultHTTPPort = 8545

	// DefaultPeerDiscoveryPort is the default port for p2p discovery
	DefaultPeerDiscoveryPort = 30303

	// DefaultWebsocketPort is the default websocket port
	DefaultWebsocketPort = 8546

	// defaultPaymentsAccessJWT for the default payments instance
	defaultPaymentsAccessJWT string

	// defaultPaymentsRefreshJWT for the default payments instance
	defaultPaymentsRefreshJWT string

	// DefaultVault for this instance of nchain
	DefaultVault *vault.Vault

	// DefaultKey for this instance of nchain
	DefaultKey *vault.Key

	// TxFilters contains in-memory Filter instances used for real-time stream processing
	TxFilters = map[string][]interface{}{}

	// ConsumeNATSStreamingSubscriptions is a flag the indicates if the nchain instance is running in API or consumer mode
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
	lvl := os.Getenv("LOG_LEVEL")
	if lvl == "" {
		lvl = "INFO"
	}
	var endpoint *string
	if os.Getenv("SYSLOG_ENDPOINT") != "" {
		endpt := os.Getenv("SYSLOG_ENDPOINT")
		endpoint = &endpt
	}
	Log = logger.NewLogger("nchain", lvl, endpoint)

	DefaultAWSConfig = awsconf.GetConfig()
	ConsumeNATSStreamingSubscriptions = strings.ToLower(os.Getenv("CONSUME_NATS_STREAMING_SUBSCRIPTIONS")) == "true"
}

func RequireInfrastructureSupport() {
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

func RequirePayments() {
	paymentsAccessJWT := os.Getenv("PAYMENTS_ACCESS_TOKEN")
	if paymentsAccessJWT != "" {
		defaultPaymentsAccessJWT = paymentsAccessJWT
	}

	if defaultPaymentsAccessJWT == "" {
		defaultPaymentsRefreshJWT = os.Getenv("PAYMENTS_REFRESH_TOKEN")
		if defaultPaymentsRefreshJWT == "" {
			Log.Panicf("failed to parse PAYMENTS_REFRESH_TOKEN from nchain environent")
		}

		err := refreshPaymentsAccessToken()
		if err != nil {
			Log.Panicf(err.Error())
		}

		go func() {
			timer := time.NewTicker(refreshTokenTickInterval)
			for {
				select {
				case <-timer.C:
					err = refreshPaymentsAccessToken()
					if err != nil {
						Log.Debugf("failed to refresh payments access token; %s", err.Error())
					}
				default:
					time.Sleep(refreshTokenSleepInterval)
				}
			}
		}()
	}
}

func refreshPaymentsAccessToken() error {
	if defaultPaymentsRefreshJWT == "" {
		return errors.New("failed to refresh payments access token")
	}

	token, err := refreshAccessToken(defaultPaymentsRefreshJWT)
	if err != nil {
		return fmt.Errorf("failed to authorize access token for given payments refresh token; %s", err.Error())
	}

	if token.AccessToken == nil {
		return fmt.Errorf("failed to authorize access token for given payments refresh token: %s", token.ID.String())
	}

	defaultPaymentsAccessJWT = *token.AccessToken
	return nil
}

func RequireVault() {
	util.RequireVault()

	vaults, err := vault.ListVaults(util.DefaultVaultAccessJWT, map[string]interface{}{})
	if err != nil {
		Log.Panicf("failed to fetch vaults for given nchain vault token; %s", err.Error())
	}

	if len(vaults) > 0 {
		// HACK
		DefaultVault = vaults[0]
		Log.Debugf("resolved default nchain vault instance: %s", DefaultVault.ID.String())
	} else {
		DefaultVault, err = vault.CreateVault(util.DefaultVaultAccessJWT, map[string]interface{}{
			"name":        fmt.Sprintf("nchain vault %d", time.Now().Unix()),
			"description": "default organizational keystore",
		})
		if err != nil {
			Log.Panicf("failed to create default vaults for nchain instance; %s", err.Error())
		}
		Log.Debugf("created default nchain vault instance: %s", DefaultVault.ID.String())
	}

	if DefaultVault != nil {
		keys, err := vault.ListKeys(util.DefaultVaultAccessJWT, DefaultVault.ID.String(), map[string]interface{}{
			"spec": "secp256k1",
		})
		if err != nil {
			Log.Panicf("failed to fetch keys for given nchain vault token; %s", err.Error())
		}
		if len(keys) > 0 {
			DefaultKey = keys[0]
		} else {
			DefaultKey, err = vault.CreateKey(util.DefaultVaultAccessJWT, DefaultVault.ID.String(), map[string]interface{}{
				"name":        "providepayments mtx vault",
				"description": "default providepayments managed transactions (mtx) vault for secp256k1 signing",
				"spec":        "secp256k1",
				"type":        "asymmetric",
				"usage":       "sign/verify",
			})
			if err != nil {
				Log.Panicf("failed to create default nchain mtx key instance; %s", err.Error())
			}
			Log.Debugf("created default nchain mtx key instance: %s", DefaultKey.ID.String())
		}
	}
}

func refreshAccessToken(token string) (*ident.Token, error) {
	status, resp, err := ident.InitDefaultIdentService(StringOrNil(token)).Post("tokens", map[string]interface{}{
		"grant_type": "refresh_token",
	})
	if err != nil {
		return nil, err
	}

	if status != 201 {
		return nil, fmt.Errorf("failed to refresh access token; status: %v; %s", status, err.Error())
	}

	// FIXME...
	tkn := &ident.Token{}
	tknraw, _ := json.Marshal(resp)
	err = json.Unmarshal(tknraw, &tkn)

	if err != nil {
		return nil, fmt.Errorf("failed to authorize token; status: %v; %s", status, err.Error())
	}

	return tkn, nil
}
