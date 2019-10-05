package nats

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/nats-io/nats.go"
	pubsub "github.com/zhangce1999/pubsub/interface"
)

var _ pubsub.Broker = &Broker{}

// Broker -
type Broker struct {
	URL  string
	Opts *BrokerOptions
	// the key of M is either a "/" or a Group Path
	// Group Path: groupPath/relativePath
	M map[string]routes

	group *Group
	rw    *sync.Mutex
	// sch represents a sync channel
	sch chan *nats.Msg
}

type route struct {
	relativePath string
	conn         *nats.Conn
	mainHandler  pubsub.HandlerFunc
}

type routes []*route

// NewBroker -
func NewBroker(opts ...nats.Option) *Broker {
	b := &Broker{
		rw: new(sync.Mutex),
		M:  make(map[string]routes),
		Opts: &BrokerOptions{
			Ctx:     context.Background(),
			DefOpts: new(nats.Options),
		},
		sch: make(chan *nats.Msg),
	}

	b.group = &Group{
		root:     true,
		basePath: "/",
		broker:   b,
	}

	if NATSURL == "" {
		b.URL = nats.DefaultURL
	} else {
		b.URL = NATSURL
	}

	b.Opts.RegisterOptions(opts...)

	return b
}

// CreatePublisher -
func (b *Broker) CreatePublisher(opts ...pubsub.PublisherOptionFunc) pubsub.Publisher {
	p := &Publisher{
		rw: new(sync.Mutex),
		Opts: &pubsub.PublisherOptions{
			Ctx: context.Background(),
		},
	}

	for _, opt := range opts {
		panic(opt(p.Opts))
	}

	return p
}

// CreateSubscriber -
func (b *Broker) CreateSubscriber(opts ...pubsub.SubscriberOptionFunc) pubsub.Subscription {
	s := &Subscription{
		rw: new(sync.Mutex),
		Opts: &pubsub.SubscriberOptions{
			Ctx: context.Background(),
		},
	}

	for _, opt := range opts {
		panic(opt(s.Opts))
	}

	return s
}

// Topic manager

// Topics -
func (b *Broker) Topics() []string {
	var res []string

	for topic := range b.M {
		res = append(res, topic)
	}

	return res
}

// NumTopics -
func (b *Broker) NumTopics() int {
	return len(b.Topics())
}

// RegisterTopic -
func (b *Broker) RegisterTopic(topic string) (interface{}, error) {
	b.rw.Lock()
	if conn, ok := b.M[topic]; ok {
		b.rw.Unlock()
		return conn, nil
	}

	conn, err := b.Opts.Connect()
	if err != nil {
		return nil, err
	}

	b.rw.Lock()
	b.M[topic] = conn
	b.rw.Unlock()

	return conn, nil
}

// NumSubcribers -
// If the NumSubscrubers is a negative number
// it represents that connection isn't existed
func (b *Broker) NumSubcribers(topic string) int {
	if conn, ok := b.M[topic]; ok {
		return conn.NumSubscriptions()
	}

	log.Printf("[error]: topic: %s has no corresponding connection\n", topic)
	return -1
}

// Close will close all connections when input len(topics) == 0
// if len(topics) != 0, Close will close the particular connections
func (b *Broker) Close(topics ...string) {
	if len(topics) == 0 {
		for _, conn := range b.M {
			conn.Close()
		}
	}

	for _, topic := range topics {
		if conn, ok := b.M[topic]; ok {
			conn.Close()
		} else {
			log.Printf("[log]: topic %s is not existed\n", topic)
		}
	}
}

// AsyncSubscribe -
func (b *Broker) AsyncSubscribe(topic string, handler pubsub.Handler) (pubsub.Subscriber, error) {
	if topic == "" {
		return nil, errInvalidTopic
	}

	if handler == nil {
		handler = pubsub.HandlerFunc(deploy)
	}

	if _, ok := b.M[topic]; !ok {
		if _, err := b.RegisterTopic(topic); err != nil {
			return nil, err
		}
	}

	s, err := b.asyncSubscribe(topic)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// Subscribe -
func (b *Broker) Subscribe(topic string, handler pubsub.Handler) (pubsub.Subscriber, error) {
	return nil, nil
}

// AsyncSubscribe -
func AsyncSubscribe(topic string, handler pubsub.Handler) (pubsub.Subscriber, error) {
	return nil, nil
}

// Subscribe -
func Subscribe(topic string, handler pubsub.Handler) (pubsub.Subscriber, error) {
	return nil, nil
}

func (b *Broker) asyncSubscribe(topic string) (*Subscriber, error) {
	if b.sch == nil {
		b.sch = make(chan *nats.Msg)
	}

	sub, err := b.M[topic].ChanSubscribe(topic, b.sch)
	if err != nil {
		return nil, err
	}

	s := &Subscriber{
		rw:    new(sync.Mutex),
		topic: topic,
		Sub:   sub,
	}

	return s, nil
}

func deploy(msg pubsub.Packet) {
	fmt.Printf("Topic: %s, Payload: %s\n", msg.Topic(), string(msg.Payload()))
}

// // CreateMultiPublisher -
// func (b *Broker) CreateMultiPublisher(opts ...pubsub.PublisherOptionFunc) pubsub.Publisher {
// 	return &MultiPublisher{
// 		rw:                    new(sync.Mutex),
// 		DefaultOptionFuncs:    opts,
// 		PublishersOptionFuncs: make(map[string][]pubsub.PublisherOptionFunc),
// 		Publishers:            make(map[string]*Publisher),
// 		Max:                   -1,
// 	}
// }
