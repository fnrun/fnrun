// Package kafka provides an fnrun source that receives messages from Kafka.
// The kafka source will invoke a function with a message and will mark the
// message as received unless the function returns an error.
package kafka

import (
	"context"

	"github.com/Shopify/sarama"
	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/fnrun/fnrun/pkg/run"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

type kafkaSource struct {
	Group        string
	Brokers      []string
	Topics       []string
	Oldest       bool
	Assignor     sarama.BalanceStrategy
	Version      sarama.KafkaVersion `mapstructure:",omitempty"`
	IgnoreErrors bool

	client sarama.ConsumerGroup
}

func (k *kafkaSource) setUpConsumerGroup() error {
	if k.client != nil {
		return nil
	}

	config := sarama.NewConfig()
	config.Version = k.Version
	config.Consumer.Group.Rebalance.Strategy = k.Assignor

	if k.Oldest {
		config.Consumer.Offsets.Initial = sarama.OffsetOldest
	}

	client, err := sarama.NewConsumerGroup(k.Brokers, k.Group, config)
	if err != nil {
		return errors.Wrap(err, "error creating consumer group client")
	}
	k.client = client

	return nil
}

func (k *kafkaSource) Serve(ctx context.Context, f fn.Fn) error {
	if err := k.setUpConsumerGroup(); err != nil {
		return err
	}

	consumer := &consumer{
		ctx:          ctx,
		f:            f,
		ignoreErrors: k.IgnoreErrors,
	}

	errorCh := make(chan error, 1)

	go func() {
		defer close(errorCh)
		defer k.client.Close()

		for {
			if ctx.Err() != nil {
				return
			}

			if err := k.client.Consume(ctx, k.Topics, consumer); err != nil {
				errorCh <- err
				return
			}
		}
	}()

	return <-errorCh
}

func (k *kafkaSource) RequiresConfig() bool {
	return true
}

func (k *kafkaSource) ConfigureMap(configMap map[string]interface{}) error {
	// Honor default value of sticky. There is an issue with trying to overwrite
	// a sarama.BalanceStrategy when one already exists. It is not apparent to me
	// exactly what the issue is, so the following code is in place to honor the
	// existing default without actually setting the default value in the source
	// upon instantiation.
	if _, exists := configMap["assignor"]; !exists {
		configMap["assignor"] = "sticky"
	}

	decoderConfig := &mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   k,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToSliceHookFunc(","),
			stringToKafkaVersionHookFunc(),
			stringToBalanceStrategyHookFunc(),
		),
	}
	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return err
	}

	return decoder.Decode(configMap)
}

// New returns a kafka source with default values. The resulting value must be
// configured with at least broker and topic information before calling Serve.
func New() run.Source {
	version, _ := sarama.ParseKafkaVersion("2.1.1")

	return &kafkaSource{
		Version: version,
		Oldest:  false,
	}
}
