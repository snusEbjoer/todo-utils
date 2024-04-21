package rmq

import (
	"context"
	"fmt"
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
	return &Rmq{ch, conn, q, queue + "_topic"}, nil
}
func GetTopic(routingKey string) string {
	arr := strings.Split(routingKey, ".")
	return arr[0] + "_topic"
}
func (r *Rmq) Send(ctx context.Context, sendTo string, body []byte) ([]byte, error) {
	corrId := utils.RandomString(32)
	err := r.ch.PublishWithContext(
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
	err = r.ch.QueueBind(
		r.queue.Name,
		sendTo,
		r.exchange,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}
	msgs, err := r.ch.Consume(
		r.queue.Name,
		r.queue.Name,
		false,
		false,
		false,
		false,
		nil,
	)
	defer r.ch.Cancel(r.queue.Name, false)
	if err != nil {
		return nil, err
	}
	for d := range msgs {
		if d.CorrelationId == corrId {
			return d.Body, nil
		}
	}
	return nil, err
}

func (r *Rmq) HandleMessage(routingKey string, handler func(msg amqp.Delivery) []byte) {
	ch, err := r.conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	err = ch.QueueBind(
		r.queue.Name,
		routingKey,
		r.exchange,
		false,
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}
	msgs, err := ch.Consume(
		r.queue.Name,
		r.queue.Name,
		false,
		false,
		false,
		false,
		nil,
	)
	defer ch.Close()
	if err != nil {
		log.Fatal(err)
	}
	for d := range msgs {
		fmt.Println(d.RoutingKey)
		if routingKey == d.RoutingKey {
			r.ch.PublishWithContext(
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
			d.Ack(true)
		}
	}
}

func (r *Rmq) Listen() {
	ch, err := r.conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	msgs, err := ch.Consume(
		r.queue.Name,
		r.queue.Name,
		false,
		false,
		false,
		false,
		nil,
	)
	defer ch.Close()
	if err != nil {
		log.Fatal(err)
	}
	var forever chan struct{}
	go func() {
		for d := range msgs {
			fmt.Printf("got message with routong key: %s", d.RoutingKey)
		}
	}()
	fmt.Println("Waiting for messages")
	<-forever
}
