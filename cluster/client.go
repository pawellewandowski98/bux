package cluster

import (
	"context"

	"github.com/go-redis/redis/v8"
	zLogger "github.com/mrz1836/go-logger"
)

type (

	// Client is the client (configuration)
	Client struct {
		pubSubService
		options *clientOptions
	}

	// clientOptions holds all the configuration for the client
	clientOptions struct {
		coordinator     Coordinator                 // which coordinator to use, either 'memory' or 'redis'
		debug           bool                        // For extra logs and additional debug information
		logger          zLogger.GormLoggerInterface // Internal logger interface
		newRelicEnabled bool                        // Whether to use New Relic
		prefix          string                      // the cluster key prefix to use before all keys
		redisOptions    *redis.Options
	}
)

// NewClient create new cluster client
func NewClient(ctx context.Context, opts ...ClientOps) (*Client, error) {
	// Create a new client with defaults
	client := &Client{options: defaultClientOptions()}

	// Overwrite defaults with any set by user
	for _, opt := range opts {
		opt(client.options)
	}

	// Use NewRelic if it's enabled (use existing txn if found on ctx)
	ctx = client.options.getTxnCtx(ctx)

	// Set logger if not set
	if client.options.logger == nil {
		client.options.logger = zLogger.NewGormLogger(client.IsDebug(), 4)
	}

	if client.options.coordinator == CoordinatorRedis {
		pubSubClient, err := NewRedisPubSub(ctx, client.options.redisOptions)
		if err != nil {
			return nil, err
		}
		pubSubClient.debug = client.IsDebug()
		pubSubClient.logger = client.options.logger
		pubSubClient.prefix = client.GetClusterPrefix()
		client.pubSubService = pubSubClient
	} else {
		pubSubClient, err := NewMemoryPubSub(ctx)
		if err != nil {
			return nil, err
		}

		pubSubClient.debug = client.IsDebug()
		pubSubClient.logger = client.options.logger
		pubSubClient.prefix = client.GetClusterPrefix()
		client.pubSubService = pubSubClient
	}

	// Return the client
	return client, nil
}

// IsDebug returns whether debugging is on or off
func (c *Client) IsDebug() bool {
	return c.options.debug
}

// GetClusterPrefix returns the cluster key prefix that can be used in things like Redis
func (c *Client) GetClusterPrefix() string {
	return c.options.prefix
}
