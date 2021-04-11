package kafka

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/fnrun/fnrun/pkg/fn"
	"github.com/fnrun/fnrun/pkg/run/config"
)

func equals(t *testing.T, got, want interface{}) {
	t.Helper()

	if got != want {
		t.Errorf("want %+v, got %+v", want, got)
	}
}

func deepEquals(t *testing.T, got, want interface{}) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("want %#v, got %#v", want, got)
	}
}

func TestConfigure_invalidConfig(t *testing.T) {
	k := New()
	err := config.Configure(k, nil)

	if err == nil {
		t.Error("expected config.Configure to return an error but it did not")
	}
}

func TestConfigureMap(t *testing.T) {
	k := New().(*kafkaSource)
	err := config.Configure(k, map[string]interface{}{
		"group":    "myGroupName",
		"brokers":  "1.2.3.4,2.3.4.5",
		"topics":   "topicA,topicB",
		"oldest":   true,
		"version":  "2.1.0",
		"assignor": "roundrobin",
	})

	if err != nil {
		t.Fatalf("config.Configure returned an error: %+v", err)
	}

	equals(t, k.Group, "myGroupName")
	deepEquals(t, k.Brokers, []string{"1.2.3.4", "2.3.4.5"})
	deepEquals(t, k.Topics, []string{"topicA", "topicB"})
	equals(t, k.Oldest, true)
	equals(t, k.Version.String(), "2.1.0")
	equals(t, k.Assignor.Name(), sarama.BalanceStrategyRoundRobin.Name())
}

func TestConfigureMap_invalidMapValue(t *testing.T) {
	k := New().(*kafkaSource)
	err := config.Configure(k, map[string]interface{}{
		"oldest": 3,
	})

	if err == nil {
		t.Fatal("expected config.Configure to return an error but it did not")
	}
}

func TestServe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	client := newTestConsumerGroupHandler()
	k := &kafkaSource{
		client: client,
	}

	capturedInputCh := make(chan interface{}, 1)

	f := fn.NewFnFromInvokeFunc(func(ctx context.Context, input interface{}) (interface{}, error) {
		cancel()
		capturedInputCh <- input
		return input, nil
	})

	client.InputCh <- newConsumerMessage("my-topic", []byte("some key"), []byte("some value"))
	if err := k.Serve(ctx, f); err != nil {
		t.Fatal(err)
	}

	capturedInput := (<-capturedInputCh).(map[string]interface{})

	gotInput := map[string]interface{}{
		"key":   capturedInput["key"],
		"value": capturedInput["value"],
		"topic": capturedInput["topic"],
	}
	wantInput := map[string]interface{}{
		"key":   "some key",
		"value": "some value",
		"topic": "my-topic",
	}

	if !reflect.DeepEqual(gotInput, wantInput) {
		t.Errorf("unexpected input value: want %#v, got %#v", wantInput, gotInput)
	}
}

func TestServe_withFnError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	expectedErr := errors.New("expected error")

	client := newTestConsumerGroupHandler()
	k := &kafkaSource{
		client: client,
	}

	f := fn.NewFnFromInvokeFunc(func(ctx context.Context, input interface{}) (interface{}, error) {
		cancel()
		return nil, expectedErr
	})

	client.InputCh <- newConsumerMessage("my-topic", []byte("some key"), []byte("some value"))
	err := k.Serve(ctx, f)

	if err != expectedErr {
		t.Errorf("Serve did not returned expected error: want %+v, got %+v", expectedErr, err)
	}
}

func TestServe_withFnError_ignoreErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	client := newTestConsumerGroupHandler()
	k := &kafkaSource{
		IgnoreErrors: true,
		client:       client,
	}

	f := fn.NewFnFromInvokeFunc(func(ctx context.Context, input interface{}) (interface{}, error) {
		cancel()
		return nil, errors.New("unexpected error")
	})

	client.InputCh <- newConsumerMessage("my-topic", []byte("some key"), []byte("some value"))
	err := k.Serve(ctx, f)

	if err != nil {
		t.Errorf("Serve returned error: %+v", err)
	}
}

func newConsumerMessage(topic string, key, value []byte) *sarama.ConsumerMessage {
	return &sarama.ConsumerMessage{
		Headers:        []*sarama.RecordHeader{},
		Timestamp:      time.Now(),
		BlockTimestamp: time.Now(),
		Topic:          topic,
		Key:            key,
		Value:          value,
	}
}

// -----------------------------------------------------------------------------
// Mock consumer group handler

type testConsumerGroupHandler struct {
	InputCh chan *sarama.ConsumerMessage
	errorCh chan error
	closed  bool
}

func (cg *testConsumerGroupHandler) Consume(ctx context.Context, topics []string, handler sarama.ConsumerGroupHandler) error {
	session := &testConsumerGroupSession{Ctx: ctx}
	claim := newTestConsumerGroupClaim(session, cg.InputCh)

	go func() {
		<-ctx.Done()
		cg.Close()
	}()

	return handler.ConsumeClaim(session, claim)
}

func (cg *testConsumerGroupHandler) Errors() <-chan error {
	return cg.errorCh
}

func (cg *testConsumerGroupHandler) Close() error {
	if !cg.closed {
		close(cg.InputCh)
		close(cg.errorCh)
		cg.closed = true
	}

	return nil
}

func newTestConsumerGroupHandler() *testConsumerGroupHandler {
	return &testConsumerGroupHandler{
		InputCh: make(chan *sarama.ConsumerMessage, 10),
		errorCh: make(chan error),
	}
}

// -----------------------------------------------------------------------------
// Mock consumer group session

type testConsumerGroupSession struct {
	Ctx context.Context
}

var _ sarama.ConsumerGroupSession = (*testConsumerGroupSession)(nil)

func (sess *testConsumerGroupSession) Claims() map[string][]int32 {
	return map[string][]int32{}
}

func (sess *testConsumerGroupSession) MemberID() string {
	return ""
}

func (sess *testConsumerGroupSession) GenerationID() int32 {
	return 0
}

func (sess *testConsumerGroupSession) MarkOffset(topic string, partition int32, offset int64, metadata string) {
}

func (sess *testConsumerGroupSession) Commit() {
}

func (sess *testConsumerGroupSession) ResetOffset(topic string, partition int32, offset int64, metadata string) {
}

func (sess *testConsumerGroupSession) MarkMessage(msg *sarama.ConsumerMessage, metadata string) {
}

func (sess *testConsumerGroupSession) Context() context.Context {
	return sess.Ctx
}

// -----------------------------------------------------------------------------
// Mock consumer group claim

type testConsumerGroupClaim struct {
	messageChan <-chan *sarama.ConsumerMessage
	session     *testConsumerGroupSession
}

func (gc *testConsumerGroupClaim) Topic() string {
	return ""
}

func (gc *testConsumerGroupClaim) Partition() int32 {
	return 0
}

func (gc *testConsumerGroupClaim) InitialOffset() int64 {
	return 0
}

func (gc *testConsumerGroupClaim) HighWaterMarkOffset() int64 {
	return 0
}

func (gc *testConsumerGroupClaim) Messages() <-chan *sarama.ConsumerMessage {
	return gc.messageChan
}

func newTestConsumerGroupClaim(session *testConsumerGroupSession, ch <-chan *sarama.ConsumerMessage) *testConsumerGroupClaim {
	claim := testConsumerGroupClaim{
		messageChan: ch,
		session:     session,
	}

	return &claim
}
