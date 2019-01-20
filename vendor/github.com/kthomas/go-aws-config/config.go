package awsconf

import (
	"os"
	"sync"
)

type Config struct {
	AccessKeyId		*string
	SecretAccessKey		*string
	DefaultS3Bucket		*string
	DefaultRegion		*string
	DefaultSqsQueueUrl	*string
	DefaultVpcID *string
}

var configInstance *Config
var configOnce sync.Once

func GetConfig() (*Config) {
	configOnce.Do(func() {
		var awsAccessKeyId string
		if os.Getenv("AWS_ACCESS_KEY_ID") != "" {
			awsAccessKeyId = os.Getenv("AWS_ACCESS_KEY_ID")
		}

		var awsSecretAccessKey string
		if os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
			awsSecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
		}

		var awsDefaultRegion string
		if os.Getenv("AWS_DEFAULT_REGION") != "" {
			awsDefaultRegion = os.Getenv("AWS_DEFAULT_REGION")
		}

		var awsDefaultS3Bucket string
		if os.Getenv("AWS_DEFAULT_S3_BUCKET") != "" {
			awsDefaultS3Bucket = os.Getenv("AWS_DEFAULT_S3_BUCKET")
		}

		var awsDefaultSqsQueueUrl string
		awsDefaultSqsQueueUrl = os.Getenv("AWS_DEFAULT_SQS_QUEUE_URL")
		if os.Getenv("AWS_DEFAULT_SQS_QUEUE_URL") != "" {
			awsDefaultSqsQueueUrl = os.Getenv("AWS_DEFAULT_SQS_QUEUE_URL")
		}

        var awsDefaultVpcId string
        if os.Getenv("AWS_DEFAULT_VPC_ID") != "" {
			awsDefaultVpcId = os.Getenv("AWS_DEFAULT_VPC_ID")
		}

		configInstance = &Config{
			AccessKeyId: &awsAccessKeyId,
			SecretAccessKey: &awsSecretAccessKey,
			DefaultRegion: &awsDefaultRegion,
			DefaultS3Bucket: &awsDefaultS3Bucket,
			DefaultSqsQueueUrl: &awsDefaultSqsQueueUrl,
            DefaultVpcID: &awsDefaultVpcId,
		}
	})
	return configInstance
}
