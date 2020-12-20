package dynamodb

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/cenkalti/backoff/v4"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/job"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/provider"
	"github.com/traefik/traefik/v2/pkg/safe"
)

var _ provider.Provider = (*Provider)(nil)

// Provider holds configuration for provider.
type Provider struct {
	AccessKeyID     string `description:"The AWS credentials access key to use for making requests"`
	RefreshSeconds  int    `description:"Polling interval (in seconds)" export:"true"`
	Region          string `description:"The AWS region to use for requests" export:"true"`
	SecretAccessKey string `description:"The AWS credentials secret key to use for making requests"`
	TableName       string `description:"The AWS dynamodb table that stores configuration for traefik" export:"true"`
	Endpoint        string `description:"The endpoint of a dynamodb. Used for testing with a local dynamodb"`
}

type dynamoDBClient struct {
	db dynamodbiface.DynamoDBAPI
}

// Init the provider
func (p *Provider) Init() error {
	return nil
}

// SetDefaults sets the default values.
func (p *Provider) SetDefaults() {
	p.RefreshSeconds = 15
}

func (p *Provider) createClient(logger log.Logger) (*dynamoDBClient, error) {
	log.Info("Creating Provider client...")
	sess, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		return nil, err
	}
	ec2meta := ec2metadata.New(sess)
	if p.Region == "" && ec2meta.Available() {
		logger.Infoln("No region provided, querying instance metadata endpoint...")
		identity, err := ec2meta.GetInstanceIdentityDocument()
		if err != nil {
			return nil, err
		}
		p.Region = identity.Region
	}
	cfg := &aws.Config{
		Credentials: credentials.NewChainCredentials(
			[]credentials.Provider{
				&credentials.StaticProvider{
					Value: credentials.Value{
						AccessKeyID:     p.AccessKeyID,
						SecretAccessKey: p.SecretAccessKey,
					},
				},
				&credentials.EnvProvider{},
				&credentials.SharedCredentialsProvider{},
				defaults.RemoteCredProvider(*(defaults.Config()), defaults.Handlers()),
			}),
	}

	// Set the region if it is defined by the user or resolved from the EC2 metadata.
	if p.Region != "" {
		cfg.Region = &p.Region
	}

	if p.Endpoint != "" {
		cfg.Endpoint = aws.String(p.Endpoint)
	}

	return &dynamoDBClient{
		dynamodb.New(sess, cfg),
	}, nil
}

func (p *Provider) loadConfiguration(ctx context.Context, client *dynamoDBClient, configurationChan chan<- dynamic.Message) error {
	fmt.Printf("ggggg dynamodb loadConfiguration")
	configurationChan <- dynamic.Message{
		ProviderName: "dynamodb",
		// NOTE: TODO
		Configuration: nil,
	}
	return nil
}

// Provide configuration to traefik from DynamoDB.
func (p Provider) Provide(configurationChan chan<- dynamic.Message, pool *safe.Pool) error {
	pool.GoCtx(func(routineCtx context.Context) {
		ctxLog := log.With(routineCtx, log.Str(log.ProviderName, "dynamodb"))
		logger := log.FromContext(ctxLog)
		fmt.Printf("hhhhh dynamodb\n")

		operation := func() error {
			awsClient, err := p.createClient(logger)
			if err != nil {
				return fmt.Errorf("unable to create AWS client: %w", err)
			}

			err = p.loadConfiguration(ctxLog, awsClient, configurationChan)
			if err != nil {
				return fmt.Errorf("failed to get dynamodb configuration: %w", err)
			}

			ticker := time.NewTicker(time.Second * time.Duration(p.RefreshSeconds))
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					err = p.loadConfiguration(ctxLog, awsClient, configurationChan)
					if err != nil {
						return fmt.Errorf("failed to refresh dynamodb configuration: %w", err)
					}

				case <-routineCtx.Done():
					return nil
				}
			}
		}

		notify := func(err error, time time.Duration) {
			logger.Errorf("Provider connection error %+v, retrying in %s", err, time)
		}
		err := backoff.RetryNotify(safe.OperationWithRecover(operation), backoff.WithContext(job.NewBackOff(backoff.NewExponentialBackOff()), routineCtx), notify)
		if err != nil {
			logger.Errorf("Cannot connect to Provider api %+v", err)
		}
	})

	return nil
}
