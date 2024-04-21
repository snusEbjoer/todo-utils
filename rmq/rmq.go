package rmq

import (
	"context"
	"fmt"
	"log"
	"strings"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/snusEbjoer/todo-utils/utils"
)

type Handler struct {
	routingKey string
	handler    func(msg amqp.Delivery) []byte
}

type Rmq struct {
	ch       *amqp.Channel
	conn     *amqp.Connection
	queue    amqp.Queue
	exchange string
	handlers []Handler
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
	if err != nil {
		return nil, err
	}
	return &Rmq{ch, conn, q, queue + "_topic", []Handler{}}, nil
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
		fmt.Printf("get msg from %s with routing key : %s \n", d.Exchange, d.RoutingKey)
		if d.CorrelationId == corrId && d.RoutingKey == sendTo {
			d.Ack(true)
			return d.Body, nil
		}
	}
	return nil, err
}

func (r *Rmq) HandleMessage(routingKey string, handler func(msg amqp.Delivery) []byte) {
	r.handlers = append(r.handlers, Handler{routingKey: routingKey, handler: handler})
}

func (r *Rmq) Reply(d amqp.Delivery, h Handler) {
	ch, err := r.conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(d.RoutingKey)
	ch.PublishWithContext(
		context.Background(),
		d.ReplyTo,
		d.RoutingKey,
		false,
		false,
		amqp.Publishing{
			Body:          h.handler(d),
			ContentType:   d.ContentType,
			CorrelationId: d.CorrelationId,
		},
	)
	d.Ack(true)
	ch.Close()
}

func (r *Rmq) Listen() {
	ch, err := r.conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()
	if err != nil {
		log.Fatal(err)
	}
	for _, h := range r.handlers {
		err := ch.QueueBind(
			r.queue.Name,
			h.routingKey,
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
	var forever chan struct{}
	go func() {
		for d := range msgs {
			for _, h := range r.handlers {
				if h.routingKey == d.RoutingKey {
					go r.Reply(d, h)
				}

			}
		}
	}()
	fmt.Println("Waiting for messages")
	<-forever
}
