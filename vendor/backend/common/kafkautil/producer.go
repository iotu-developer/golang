// by liudan 2017.02
package kafkautil

import (
	"encoding/json"
	"errors"
	"log"
	"time"

	"third/sarama"
)

const (
	NeverTimeout = -1 * time.Second
)

type AsyncProducer struct {
	p sarama.AsyncProducer
}

func NewAsyncProducer(addrs []string, ackType sarama.RequiredAcks, flushInterval time.Duration) (*AsyncProducer, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V0_10_0_1
	config.Producer.RequiredAcks = ackType
	config.Producer.Flush.Frequency = flushInterval
	config.Producer.Return.Errors = true

	producer, err := sarama.NewAsyncProducer(addrs, config)
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			select {
			case err, ok := <-producer.Errors():
				if !ok {
					goto exit
				}
				log.Printf("kafka async producer error:%s, msg:%+v", err.Error(), err.Msg)
				time.Sleep(1 * time.Second)
			}
		}
	exit:
		log.Printf("kafka async producer error watcher gone")
	}()

	return &AsyncProducer{p: producer}, nil
}

func (producer *AsyncProducer) Send(msg *sarama.ProducerMessage, timeout time.Duration) error {
	if msg == nil {
		return nil
	}

	if timeout == NeverTimeout {
		producer.p.Input() <- msg
	} else {
		select {
		case producer.p.Input() <- msg: //pass
		case <-time.After(timeout):
			log.Printf("kafka async producer send msg:%+v timeout", msg)
			return errors.New("send msg timeout")
		}
	}
	return nil
}

type JSONEncoder struct {
	Value interface{}
}

func (je *JSONEncoder) Encode() ([]byte, error) {
	return json.Marshal(je.Value)
}

func (je *JSONEncoder) Length() int {
	b, _ := je.Encode()
	return len(b)
}

func (producer *AsyncProducer) SendJSON(topic, key string, value interface{}, timeout time.Duration) error {
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(key),
		Value: &JSONEncoder{value},
	}

	return producer.Send(msg, timeout)
}

func (producer *AsyncProducer) Close() error {
	return producer.p.Close()
}

type SyncProducer struct {
	p sarama.SyncProducer
}

// NewSyncProducer publish each message in sync mode which returns until message is acked by kafka broker.
// The throughout capacity may be limited.
// Using AsyncProducer if you are looking for high throughput producer.
func NewSyncProducer(addrs []string, ackType sarama.RequiredAcks) (*SyncProducer, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V0_10_0_1
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true

	config.Producer.RequiredAcks = ackType
	config.Producer.Flush.MaxMessages = 1

	producer, err := sarama.NewSyncProducer(addrs, config)
	if err != nil {
		return nil, err
	}
	return &SyncProducer{p: producer}, nil
}

func (producer *SyncProducer) Send(msg *sarama.ProducerMessage) (partition int32, offset int64, err error) {
	return producer.p.SendMessage(msg)
}

func (producer *SyncProducer) SendJSON(topic, key string, value interface{}) (partition int32, offset int64, err error) {
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(key),
		Value: &JSONEncoder{value},
	}
	return producer.p.SendMessage(msg)
}

func (producer *SyncProducer) Close() error {
	return producer.p.Close()
}
