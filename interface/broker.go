package pubsub

// Broker -
type Broker interface {
	Topic

	CreatePublisher(opts ...PublisherOptionFunc) Publisher
	CreateSubscriber(opts ...SubscriberOptionFunc) Subscriber
}

// Topic -
type Topic interface {
	Topics() []string
	NumTopics() int
	RegisterTopic(topic string) (conn interface{}, err error)
	NumSubcribers(topic string) int
	Close(topics ...string)
}

// Status -
type Status interface {
	ConnStatus() *ConnStatus

	// Channel Status

	// Server Status

	// Publisher Status

	// Subscription Status
}

// ConnStatus -
type ConnStatus struct {
	DisConnected bool
	Connected    bool
	Closed       bool
	Reconnecting bool
	Connecting   bool

	DrainingSubs bool
	DrainingPubs bool
}