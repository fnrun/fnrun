package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/mocks"
	"github.com/fnrun/fnrun/fn"
	"github.com/fnrun/fnrun/run/config"
)

func stringIs(t *testing.T, got, want, message string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: want %q, got %q", message, want, got)
	}
}

func TestConfigureMap(t *testing.T) {
	m := New().(*kafkaMiddleware)

	brokers := []string{"127.0.0.1", "127.0.0.2"}
	successTopic := "mySuccessTopic"
	errorTopic := "myErrorTopic"

	certFile := "/path/to/cert/file"
	keyFile := "/path/to/key/file"
	caFile := "/path/to/ca/file"
	verifySSL := true

	var configMap map[string]interface{}
	jsonString := fmt.Sprintf(
		`{
		"brokers": "%s",
		"successTopic": "%s",
		"errorTopic": "%s",
		"certFile": "%s",
		"keyFile": "%s",
		"caFile": "%s",
		"verifySSL": %v
		}`,
		strings.Join(brokers, ","),
		successTopic,
		errorTopic,
		certFile,
		keyFile,
		caFile,
		verifySSL,
	)
	if err := json.Unmarshal([]byte(jsonString), &configMap); err != nil {
		t.Fatalf("unmarshaling json returned error: %+v", err)
	}

	if err := config.Configure(m, configMap); err != nil {
		t.Fatalf("configuring the middleware returned an error: %+v", err)
	}

	if !reflect.DeepEqual(m.Brokers, brokers) {
		t.Errorf("incorrect brokers config: want %v, got %v", brokers, m.Brokers)
	}

	stringIs(t, m.SuccessTopic, successTopic, "success topic")
	stringIs(t, m.ErrorTopic, errorTopic, "error topic")
	stringIs(t, m.CertFile, certFile, "cert file")
	stringIs(t, m.KeyFile, keyFile, "key file")
	stringIs(t, m.CAFile, caFile, "CA file")

	if m.VerifySSL != verifySSL {
		t.Errorf("verifySSL: want %v, got %v", verifySSL, m.VerifySSL)
	}
}

func makeChecker(t *testing.T, want string) func([]byte) error {
	t.Helper()

	return func(val []byte) error {
		got := string(val)

		if got != want {
			return fmt.Errorf("unexpected message value: want %q, got %q", want, got)
		}

		return nil
	}
}

func echoJsonInvokeFunc(ctx context.Context, input interface{}) (interface{}, error) {
	m := map[string]interface{}{
		"input": input,
	}

	bytes, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	return string(bytes), nil
}

func TestInvoke(t *testing.T) {
	producer := mocks.NewSyncProducer(t, nil)
	producer.ExpectSendMessageWithCheckerFunctionAndSucceed(
		makeChecker(t, `{"input":"some value"}`),
	)

	m := kafkaMiddleware{
		producer:     producer,
		SuccessTopic: "successTopic",
		ErrorTopic:   "errorTopic",
	}

	_, err := m.Invoke(
		context.Background(),
		"some value",
		fn.NewFnFromInvokeFunc(echoJsonInvokeFunc),
	)
	if err != nil {
		t.Fatalf("Invoke returned error: %+v", err)
	}
}

func TestInvoke_ignoreSuccessOnNilValue(t *testing.T) {
	producer := &failOnSendSyncProducer{t: t}

	m := kafkaMiddleware{
		producer:     producer,
		SuccessTopic: "successTopic",
		ErrorTopic:   "errorTopic",
	}

	_, err := m.Invoke(
		context.Background(),
		"some value",
		fn.NewFnFromInvokeFunc(func(_ctx context.Context, _input interface{}) (interface{}, error) {
			return nil, nil
		}),
	)
	if err != nil {
		t.Fatalf("Invoke returned error: %+v", err)
	}
}

func errorInvokeFunc(ctx context.Context, input interface{}) (interface{}, error) {
	return nil, errors.New("some error message")
}

func TestInvoke_withError(t *testing.T) {
	producer := mocks.NewSyncProducer(t, nil)
	producer.ExpectSendMessageWithCheckerFunctionAndSucceed(
		makeChecker(t, "some error message"),
	)

	m := kafkaMiddleware{
		producer:     producer,
		SuccessTopic: "successTopic",
		ErrorTopic:   "errorTopic",
	}

	_, _ = m.Invoke(
		context.Background(),
		"some value",
		fn.NewFnFromInvokeFunc(errorInvokeFunc),
	)
}

// -----------------------------------------------------------------------------
// Mock SyncProducer that fails on send

type failOnSendSyncProducer struct {
	t *testing.T
}

func (sp *failOnSendSyncProducer) SendMessage(msg *sarama.ProducerMessage) (partition int32, offset int64, err error) {
	sp.t.Fatalf("should not send message")
	return 0, 0, fmt.Errorf("should not send message")
}

func (sp *failOnSendSyncProducer) SendMessages(msg []*sarama.ProducerMessage) error {
	sp.t.Fatalf("should not send messages")
	return fmt.Errorf("should not send message")
}

func (sp *failOnSendSyncProducer) Close() error {
	return nil
}

func (sp *failOnSendSyncProducer) AbortTxn() error {
	return nil
}

func (sp *failOnSendSyncProducer) AddMessageToTxn(msg *sarama.ConsumerMessage, groupId string, metadata *string) error {
	return nil
}

func (sp *failOnSendSyncProducer) AddOffsetsToTxn(offsets map[string][]*sarama.PartitionOffsetMetadata, groupId string) error {
	return nil
}

func (sp *failOnSendSyncProducer) BeginTxn() error {
	return nil
}

func (sp *failOnSendSyncProducer) CommitTxn() error {
	return nil
}

func (sp *failOnSendSyncProducer) IsTransactional() bool {
	return false
}

func (sp *failOnSendSyncProducer) TxnStatus() sarama.ProducerTxnStatusFlag {
	return sarama.ProducerTxnFlagReady
}
