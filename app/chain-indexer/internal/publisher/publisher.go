// Package publisher 定义事件发布接口和 Kafka 实现。
package publisher

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-zero/core/logx"
)

// EventPublisher 事件发布接口
// 抽象消息队列实现，方便后续切换 Kafka / Redis Stream / Channel
type EventPublisher interface {
	// Publish 发布一条消息到指定主题
	Publish(ctx context.Context, topic string, key string, value []byte) error
	// Close 关闭发布者
	Close() error
}

// ============= Kafka 实现 =============

// KafkaPublisher Kafka 事件发布者
type KafkaPublisher struct {
	writers map[string]*kafka.Writer
	brokers []string
}

// NewKafkaPublisher 创建 Kafka 发布者
func NewKafkaPublisher(brokers []string) *KafkaPublisher {
	return &KafkaPublisher{
		writers: make(map[string]*kafka.Writer),
		brokers: brokers,
	}
}

// getWriter 获取或创建指定 topic 的 writer
func (p *KafkaPublisher) getWriter(topic string) *kafka.Writer {
	if w, ok := p.writers[topic]; ok {
		return w
	}

	w := &kafka.Writer{
		Addr:         kafka.TCP(p.brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
		WriteTimeout: 10 * time.Second,
		RequiredAcks: kafka.RequireOne,
		Async:        false, // 同步写入，确保消息不丢
	}

	p.writers[topic] = w
	logx.Infof("kafka writer created for topic: %s", topic)
	return w
}

// Publish 发布消息到 Kafka
func (p *KafkaPublisher) Publish(ctx context.Context, topic string, key string, value []byte) error {
	writer := p.getWriter(topic)

	msg := kafka.Message{
		Key:   []byte(key),
		Value: value,
		Time:  time.Now(),
	}

	if err := writer.WriteMessages(ctx, msg); err != nil {
		logx.Errorf("kafka publish to %s failed: %v", topic, err)
		return err
	}

	return nil
}

// Close 关闭所有 Kafka writer
func (p *KafkaPublisher) Close() error {
	for topic, w := range p.writers {
		if err := w.Close(); err != nil {
			logx.Errorf("close kafka writer for %s: %v", topic, err)
		}
	}
	return nil
}
