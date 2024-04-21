package rmq

import (
	"context"
	"log"
	"strings"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/snusEbjoer/todo-utils/utils"
)

type Rmq struct {
	ch       *amqp.Channel
	conn     *amqp.Connection
	queue    amqp.Queue
	exchange string
	handlers map[string]func(msg amqp.Delivery) []byte
}

func New(url string, queue string) (*Rmq, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	err = ch.ExchangeDeclare(
		queue+"_topic",
		"topic",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}
	q, err := ch.QueueDeclare(
		queue,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}
	return &Rmq{ch, conn, q, queue + "_topic", map[string]func(msg amqp.Delivery) []byte{}}, nil
}

func GetTopic(routingKey string) string {
	arr := strings.Split(routingKey, ".")
	return arr[0] + "_topic"
}

func (r *Rmq) Send(ctx context.Context, sendTo string, body []byte) ([]byte, error) {
	ch, err := r.conn.Channel()
	if err != nil {
		return nil, err
	}

	corrId := utils.RandomString(32)
	err = ch.PublishWithContext(
		context.Background(),
		GetTopic(sendTo),
		sendTo,
		false,
		false,
		amqp.Publishing{
			Body:          body,
			ContentType:   "application/json",
			CorrelationId: corrId,
			ReplyTo:       r.exchange,
		},
	)
	if err != nil {
		return nil, err
	}
	err = ch.QueueBind(
		r.queue.Name,
		sendTo,
		r.exchange,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}
	msgs, err := ch.Consume(
		r.queue.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	defer ch.Close()
	if err != nil {
		return nil, err
	}
	for d := range msgs {
		if d.CorrelationId == corrId && d.RoutingKey == sendTo {
			d.Ack(true)
			return d.Body, nil
		}
	}
	return nil, err
}

func (r *Rmq) HandleMessage(routingKey string, handler func(msg amqp.Delivery) []byte) {
	r.handlers[routingKey] = handler
}

func (r *Rmq) Reply(d amqp.Delivery, handler func(msg amqp.Delivery) []byte) {
	ch, err := r.conn.Channel()
	if err != nil {
		log.Fatal(err)
	}

	err = ch.PublishWithContext(
		context.Background(),
		d.ReplyTo,
		d.RoutingKey,
		false,
		false,
		amqp.Publishing{
			Body:          handler(d),
			ContentType:   d.ContentType,
			CorrelationId: d.CorrelationId,
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	d.Ack(true)
	ch.Close()
}

func (r *Rmq) Listen() {
	ch, err := r.conn.Channel()
	if err != nil {
		log.Fatal(err)
	}

	defer ch.Close()
	for k := range r.handlers {
		err := ch.QueueBind(
			r.queue.Name,
			k,
			r.exchange,
			false,
			nil,
		)
		if err != nil {
			log.Fatal(err)
		}
	}

	msgs, err := ch.Consume(
		r.queue.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}

	for d := range msgs {
		if handler, ok := r.handlers[d.RoutingKey]; ok {
			go r.Reply(d, handler)
		}
	}

}
