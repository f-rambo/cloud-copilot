package lib

import (
	"context"
	"sync"

	"github.com/IBM/sarama"
	"github.com/pkg/errors"
)

type KafkaConsumer struct {
	consumer sarama.Consumer
	group    sarama.ConsumerGroup
	config   *sarama.Config
	brokers  []string
	topics   []string
	groupID  string
	mu       sync.Mutex
}

func NewKafkaConsumer(brokers []string, topics []string, groupID string) (*KafkaConsumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	consumer, err := sarama.NewConsumer(brokers, config)
	if err != nil {
		return nil, err
	}

	group, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		consumer.Close()
		return nil, err
	}

	return &KafkaConsumer{
		consumer: consumer,
		group:    group,
		config:   config,
		brokers:  brokers,
		topics:   topics,
		groupID:  groupID,
	}, nil
}

func (k *KafkaConsumer) Close() error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if err := k.consumer.Close(); err != nil {
		return err
	}

	if err := k.group.Close(); err != nil {
		return err
	}

	return nil
}

func (k *KafkaConsumer) ConsumeMessages(ctx context.Context, handler MessageHandler) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			err := k.group.Consume(ctx, k.topics, handler)
			if err != nil {
				return errors.Wrap(err, "ConsumeMessages")
			}
		}
	}
}

type MessageHandler interface {
	Setup(sarama.ConsumerGroupSession) error
	Cleanup(sarama.ConsumerGroupSession) error
	ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error
}

type DefaultMessageHandler struct {
	handleMessage func(ctx context.Context, key, val []byte) error
}

func NewDefaultMessageHandler(handleMessage func(ctx context.Context, key, val []byte) error) MessageHandler {
	return &DefaultMessageHandler{handleMessage: handleMessage}
}

// Before
func (h *DefaultMessageHandler) Setup(sarama.ConsumerGroupSession) error { return nil }

// After
func (h *DefaultMessageHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *DefaultMessageHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case <-session.Context().Done():
			return nil
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			if err := h.handleMessage(session.Context(), msg.Key, msg.Value); err != nil {
				return err
			}
			session.MarkMessage(msg, "")
		}
	}
}
