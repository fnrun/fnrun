// Package servicebus provides an fnrun source that reads messages from an Azure
// servicebus queue or DLQ. It will complete or abandon messages based on
// whether the fn runs successfully.
package servicebus

import (
	"context"
	"errors"
	"log"
	"time"

	servicebus "github.com/Azure/azure-service-bus-go"
	"github.com/fnrun/fnrun/fn"
	"github.com/fnrun/fnrun/run"
	"github.com/mitchellh/mapstructure"
)

func newInputFromMessage(msg *servicebus.Message) map[string]interface{} {
	return map[string]interface{}{
		"ContentType": msg.ContentType,
		"Data":        string(msg.Data),
	}
}

type queueSource struct {
	ServiceBusConnStr     string        `mapstructure:"connectionString"`
	QueueName             string        `mapstructure:"queueName"`
	IsDeadLetterReceiver  bool          `mapstructure:"isDeadLetterReceiver"`
	AutoRenewLockInterval time.Duration `mapstructure:"autoRenewLockInterval"`
}

func (q *queueSource) RequiresConfig() bool {
	return true
}

func (q *queueSource) ConfigureMap(configMap map[string]interface{}) error {
	decoderConfig := &mapstructure.DecoderConfig{
		Metadata:   nil,
		Result:     q,
		DecodeHook: mapstructure.StringToTimeDurationHookFunc(),
	}
	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return err
	}

	err = decoder.Decode(configMap)
	if err != nil {
		return err
	}

	if q.ServiceBusConnStr == "" {
		return errors.New("expected connection string to have a value")
	}
	if q.QueueName == "" {
		return errors.New("expected queue name to be set")
	}

	return nil
}

func (q *queueSource) serveQueue(ctx context.Context, queue *servicebus.Queue, f fn.Fn) error {
	for {
		err := queue.ReceiveOne(ctx, servicebus.HandlerFunc(func(ctx context.Context, msg *servicebus.Message) error {
			newCtx, cancel := context.WithCancel(ctx)
			running := true

			go func() {
				if q.AutoRenewLockInterval.String() != "0s" {
					for running {
						time.Sleep(q.AutoRenewLockInterval)

						err := queue.RenewLocks(newCtx, msg)

						if err != nil && running {
							log.Printf("error renewing lock: %+v\n", err)
							cancel()
						}
					}
				}
			}()

			_, err := f.Invoke(newCtx, newInputFromMessage(msg))
			running = false

			if err != nil {
				log.Printf("Abandoning due to error: %+v\n", err)
				if err = msg.Abandon(newCtx); err != nil {
					return err
				}
			} else {
				if err = msg.Complete(newCtx); err != nil {
					return err
				}
			}

			return nil
		}))

		if err != nil {
			return err
		}
	}
}

func (q *queueSource) serveDLQ(ctx context.Context, queue *servicebus.Queue, f fn.Fn) error {
	dlq := queue.NewDeadLetter()
	defer func() {
		dlq.Close(ctx)
	}()

	for {
		err := dlq.ReceiveOne(ctx, servicebus.HandlerFunc(func(ctx context.Context, msg *servicebus.Message) error {
			_, err := f.Invoke(ctx, newInputFromMessage(msg))

			if err != nil {
				log.Printf("Abandoning due to error: %+v\n", err)
				if err = msg.Abandon(ctx); err != nil {
					log.Printf("Error abandoning message: %+v\n", err)
				}
			} else {
				if err = msg.Complete(ctx); err != nil {
					log.Printf("Error completing message: %+v\n", err)
				}
			}

			return nil
		}))

		if err != nil {
			return err
		}
	}
}

func (q *queueSource) Serve(ctx context.Context, f fn.Fn) error {
	ns, err := servicebus.NewNamespace(servicebus.NamespaceWithConnectionString(q.ServiceBusConnStr))
	if err != nil {
		return err
	}

	queue, err := ns.NewQueue(q.QueueName)
	if err != nil {
		return err
	}
	defer func() {
		_ = queue.Close(ctx)
	}()

	if q.IsDeadLetterReceiver {
		return q.serveDLQ(ctx, queue, f)
	}

	return q.serveQueue(ctx, queue, f)
}

// New returns as servicebus source with default values. The resulting value
// must be configured with at least a connection string and queue name before
// calling Serve.
func New() run.Source {
	return &queueSource{}
}
