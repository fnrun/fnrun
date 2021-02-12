package kafka

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/Shopify/sarama"
	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/fnrun/fnrun/pkg/run"
	"github.com/mitchellh/mapstructure"
)

type kafkaSourceConfig struct {
	Brokers  string `mapstructure:"brokers"`
	Group    string `mapstructure:"group"`
	Version  string `mapstructure:"version,omitempty"`
	Topics   string `mapstructure:"topics"`
	Assignor string `mapstructure:"assignor,omitempty"`
	Oldest   bool   `mapstructure:"oldest,omitempty"`
}

type kafkaSource struct {
	config *kafkaSourceConfig
}

func (k *kafkaSource) Serve(ctx context.Context, f fn.Fn) error {
	version, err := sarama.ParseKafkaVersion(k.config.Version)
	if err != nil {
		return err
	}

	config := sarama.NewConfig()
	config.Version = version

	switch k.config.Assignor {
	case "sticky":
		config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategySticky
	case "roundrobin":
		config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	case "range":
		config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRange
	default:
		return fmt.Errorf("Unrecognized consumer group partition assignor: %s", k.config.Assignor)
	}

	if k.config.Oldest {
		config.Consumer.Offsets.Initial = sarama.OffsetOldest
	}

	consumer := consumer{
		ready: make(chan bool),
		ctx:   ctx,
		f:     f,
	}

	client, err := sarama.NewConsumerGroup(strings.Split(k.config.Brokers, ","), k.config.Group, config)
	if err != nil {
		return fmt.Errorf("Error creating consumer group client: %v", err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			if err := client.Consume(ctx, strings.Split(k.config.Topics, ","), &consumer); err != nil {
				log.Fatalf("Error from consumer: %v", err)
			}

			if ctx.Err() != nil {
				return
			}
			consumer.ready = make(chan bool)
		}
	}()

	<-consumer.ready
	<-ctx.Done()

	wg.Wait()

	return client.Close()
}

func (k *kafkaSource) RequiresConfig() bool {
	return true
}

func (k *kafkaSource) ConfigureMap(configMap map[string]interface{}) error {
	return mapstructure.Decode(configMap, k.config)
}

func New() run.Source {
	return &kafkaSource{
		config: &kafkaSourceConfig{
			Brokers:  "",
			Group:    "",
			Version:  "2.1.1",
			Topics:   "",
			Assignor: "range",
			Oldest:   false,
		},
	}
}
