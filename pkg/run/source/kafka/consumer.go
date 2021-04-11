package kafka

import (
	"context"

	"github.com/Shopify/sarama"
	"github.com/fnrun/fnrun/pkg/fn"
)

type consumer struct {
	ctx context.Context
	f   fn.Fn
}

func (consumer *consumer) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (consumer *consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (consumer *consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		input := createInput(message)

		_, err := consumer.f.Invoke(consumer.ctx, input)
		if err != nil {
			return err // TODO This MIGHT be wrong!
		}
		session.MarkMessage(message, "")
	}

	return nil
}

func createInput(message *sarama.ConsumerMessage) map[string]interface{} {
	input := make(map[string]interface{})

	input["key"] = string(message.Key)
	input["value"] = string(message.Value)
	input["offset"] = message.Offset
	input["partition"] = message.Partition
	input["topic"] = message.Topic
	input["timestamp"] = message.Timestamp

	return input
}
