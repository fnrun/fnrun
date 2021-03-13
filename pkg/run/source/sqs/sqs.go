// Package sqs provides an fnrun source that polls for messages from SQS and
// deletes them unless the fn it serves returns an error.
//
// Because the sqs source only deletes a message if it is handled successfully,
// it is compatible with any redrive policies set on the queue.
//
// The sqs source may be configured with either a string or
// map[string]interface{} value. A string config value should contain the name
// of the queue the source will poll for messages. A map value should contain
// a `queue` containing the name of the target queue, a `timeout` containing
// an integer value representing the number of seconds of the message visibility
// timeout, and a `batchSize` contain an integer describing the maximum number
// of messages that can be received with each polling request to the queue.
package sqs

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/fnrun/fnrun/pkg/run"
	"github.com/mitchellh/mapstructure"
)

type sqsSource struct {
	config *sqsSourceConfig
}

type sqsSourceConfig struct {
	QueueName string `mapstructure:"queue"`
	Timeout   int64  `mapstructure:"timeout,omitempty"`
	BatchSize int64  `mapstructure:"batchSize,omitempty"`
}

func (*sqsSource) RequiresConfig() bool {
	return true
}

func (s *sqsSource) ConfigureString(queueName string) error {
	s.config.QueueName = queueName
	return nil
}

func (s *sqsSource) ConfigureMap(configMap map[string]interface{}) error {
	return mapstructure.Decode(configMap, s.config)
}

func (s *sqsSource) Serve(ctx context.Context, f fn.Fn) error {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := sqs.New(sess)

	urlResult, err := svc.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: &s.config.QueueName,
	})

	if err != nil {
		return err
	}

	queueURL := urlResult.QueueUrl

	for {
		msgResult, err := svc.ReceiveMessageWithContext(ctx, &sqs.ReceiveMessageInput{
			AttributeNames: []*string{
				aws.String(sqs.MessageSystemAttributeNameSentTimestamp),
			},
			MessageAttributeNames: []*string{
				aws.String(sqs.QueueAttributeNameAll),
			},
			QueueUrl:            queueURL,
			MaxNumberOfMessages: &s.config.BatchSize,
			VisibilityTimeout:   &s.config.Timeout,
		})

		if err != nil {
			return err
		}

		for _, message := range msgResult.Messages {
			_, err = f.Invoke(ctx, createInput(message))
			if err != nil {
				continue
			}

			_, err = svc.DeleteMessageWithContext(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      queueURL,
				ReceiptHandle: message.ReceiptHandle,
			})
			if err != nil {
				return err
			}
		}
	}
}

func createInput(message *sqs.Message) map[string]interface{} {
	input := make(map[string]interface{})

	input["id"] = *message.MessageId
	input["body"] = *message.Body

	return input
}

// New creates a new instance of the sqs source with default values. The
// resulting object must be configured with a queue name. If a queue name is not
// configured, Serve will return an error.
func New() run.Source {
	return &sqsSource{
		config: &sqsSourceConfig{
			Timeout:   30,
			BatchSize: 1,
		},
	}
}
