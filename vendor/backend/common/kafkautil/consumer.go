// by liudan 2017.02
package kafkautil

import (
	"encoding/json"
	"errors"
	"log"
	"third/sarama"
	cluster "third/sarama-cluster"
	"time"
)

type Consumer struct {
	c              *cluster.Consumer
	autoMarkOffset bool
}

func NewConsumer(addrs []string, groupID string, topics []string, initialOffset int64, autoMarkOffset bool) (*Consumer, error) {
	config := cluster.NewConfig()
	config.Version = sarama.V0_10_0_1
	config.Consumer.Offsets.Initial = initialOffset
	config.Consumer.Return.Errors = true
	config.Group.Return.Notifications = true

	// init consumer
	consumer, err := cluster.NewConsumer(addrs, groupID, topics, config)
	if err != nil {
		return nil, err
	}

	// watch errors
	go func() {
		for {
			select {
			case err, ok := <-consumer.Errors():
				if !ok {
					goto exit
				}
				log.Printf("kafka consumer Error: %s", err.Error())

			}
		}
	exit:
		log.Printf("kafka consumer error watch gone")
	}()

	// watch notifications
	go func() {
		for {
			select {
			case note, ok := <-consumer.Notifications():
				if !ok {
					goto exit
				}
				log.Printf("kafka consumer Rebalanced: %+v", note)

			}
		}
	exit:
		log.Printf("kafka consumer notification watcher gone")
	}()

	return &Consumer{
		c:              consumer,
		autoMarkOffset: autoMarkOffset,
	}, nil
}

func (consumer *Consumer) Receive(timeout time.Duration) (*sarama.ConsumerMessage, error) {
	var msg *sarama.ConsumerMessage
	if timeout == NeverTimeout {
		msg = <-consumer.c.Messages()
	} else {
		select {
		case msg = <-consumer.c.Messages(): // pass
		case <-time.After(timeout):
			//log.Printf("kafka consumer receive msg timeout")
			return nil, errors.New("receive msg timeout")
		}
	}

	if consumer.autoMarkOffset {
		consumer.c.MarkOffset(msg, "")
	}

	return msg, nil
}

func (consumer *Consumer) ReceiveJSON(timeout time.Duration, v interface{}) (*sarama.ConsumerMessage, error) {
	msg, err := consumer.Receive(timeout)
	if err != nil {
		return msg, err
	}

	return msg, json.Unmarshal(msg.Value, v)
}

func (consumer *Consumer) MarkOffset(msg *sarama.ConsumerMessage) {
	consumer.c.MarkOffset(msg, "")
}

func (consumer *Consumer) Close() error {
	return consumer.c.Close()
}
