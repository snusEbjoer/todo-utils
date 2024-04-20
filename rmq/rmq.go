package rmq

import (
	"context"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/snusEbjoer/todo-utils/utils"
)

type Rmq struct {
	ch    *amqp.Channel
	Queue amqp.Queue
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
	return &Rmq{ch, q}, nil
}

func (r *Rmq) Send(ctx context.Context, sendTo string, body []byte) ([]byte, error) {
	corrId := utils.RandomString(32)
	err := r.ch.PublishWithContext(ctx,
		"",
		sendTo,
		false,
		false,
		amqp.Publishing{
			ContentType:   "application/json",
			Body:          body,
			ReplyTo:       r.Queue.Name,
			CorrelationId: corrId,
		},
	)
	if err != nil {
		return nil, err
	}
	msgs, err := r.ch.Consume(
		r.Queue.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}
	defer r.ch.Cancel("", false)
	var data []byte
	for d := range msgs {
		if corrId == d.CorrelationId {
			data = d.Body
			d.Ack(true)
			break
		}
	}
	return data, err
}

func (r *Rmq) HandleMessage(routingKey string, handler func(msg amqp.Delivery) []byte) {
	err := r.ch.QueueBind(
		r.Queue.Name,
		routingKey,
		"",
		false,
		nil,
	)
	utils.FailOnError(err, "handle later")
	msgs, err := r.ch.Consume(
		r.Queue.Name,
		r.Queue.Name,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer r.ch.Cancel(r.Queue.Name, false)
	for d := range msgs {
		err := r.ch.PublishWithContext(context.Background(),
			"",
			d.ReplyTo,
			false,
			false,
			amqp.Publishing{
				Body:          handler(d),
				ContentType:   "application/json",
				CorrelationId: d.CorrelationId,
			})
		if err != nil {
			log.Fatal(err)
		}
		break
	}
}
