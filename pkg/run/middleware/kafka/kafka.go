package kafka

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/Shopify/sarama"
	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/fnrun/fnrun/pkg/run"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

type kafkaMiddleware struct {
	Brokers      []string
	SuccessTopic string
	ErrorTopic   string

	CertFile  string
	KeyFile   string
	CAFile    string `mapstructure:"caFile,omitempty"`
	VerifySSL bool   `mapstructure:"verifySSL,omitempty"`

	producer            sarama.SyncProducer
	initializationError error
	once                sync.Once
}

func sendMessage(producer sarama.SyncProducer, topic string, message interface{}) error {
	_, _, err := producer.SendMessage(&sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(fmt.Sprint(message)),
	})

	return err
}

func (m *kafkaMiddleware) newTLSConfiguration() (*tls.Config, error) {
	if m.CertFile != "" && m.KeyFile != "" && m.CAFile != "" {
		cert, err := tls.LoadX509KeyPair(m.CertFile, m.KeyFile)
		if err != nil {
			return nil, err
		}

		caCert, err := ioutil.ReadFile(m.CAFile)
		if err != nil {
			return nil, err
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		return &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            caCertPool,
			InsecureSkipVerify: m.VerifySSL,
		}, nil
	}

	return nil, nil
}

func (m *kafkaMiddleware) initializeProducer() error {
	if m.producer != nil {
		return nil
	}

	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll // Wait for all in-sync replicas to ack the message
	config.Producer.Retry.Max = 10                   // Retry up to 10 times to produce the message
	config.Producer.Return.Successes = true

	tlsConfig, err := m.newTLSConfiguration()
	if err != nil {
		return err
	}

	if tlsConfig != nil {
		config.Net.TLS.Config = tlsConfig
		config.Net.TLS.Enable = true
	}

	producer, err := sarama.NewSyncProducer(m.Brokers, config)
	if err != nil {
		return err
	}

	m.producer = producer
	return nil
}

func (m *kafkaMiddleware) RequiresConfig() bool {
	return true
}

func (m *kafkaMiddleware) ConfigureMap(configMap map[string]interface{}) error {
	decoderConfig := &mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   m,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToSliceHookFunc(","),
		),
	}
	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return err
	}

	return decoder.Decode(configMap)
}

func (m *kafkaMiddleware) Invoke(ctx context.Context, input interface{}, f fn.Fn) (interface{}, error) {
	m.once.Do(func() {
		if nil == m.producer {
			m.initializationError = m.initializeProducer()
		}
	})

	if m.initializationError != nil {
		return nil, m.initializationError
	}

	output, err := f.Invoke(ctx, input)

	if err != nil && m.ErrorTopic != "" {
		if newErr := sendMessage(m.producer, m.ErrorTopic, err); newErr != nil {
			err = errors.Wrap(err, newErr.Error())
		}
	}

	if err == nil && m.SuccessTopic != "" {
		err = sendMessage(m.producer, m.SuccessTopic, output)
	}

	return output, err
}

func New() run.Middleware {
	return &kafkaMiddleware{}
}
