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
	queueName string `mapstructure:"queue"`
	timeout   int64  `mapstructure:"timeout,omitempty"`
	batchSize int64  `mapstructure:"batchSize,omitempty"`
}

func (*sqsSource) RequiresConfig() bool {
	return true
}

func (s *sqsSource) ConfigureString(queueName string) error {
	s.config.queueName = queueName
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
		QueueName: &s.config.queueName,
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
			MaxNumberOfMessages: &s.config.batchSize,
			VisibilityTimeout:   &s.config.timeout,
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

func New() run.Source {
	return &sqsSource{
		config: &sqsSourceConfig{
			timeout:   30,
			batchSize: 1,
		},
	}
}
