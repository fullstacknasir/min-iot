package mqtt

import (
	"crypto/tls"
	"fmt"
	"log"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"miniot/apps/ingestor/internal/config"
	"miniot/apps/ingestor/internal/processor"
)

const (
	MQTTKeepAlive      = 60 * time.Second
	MQTTPingTimeout    = 10 * time.Second
	MQTTConnectTimeout = 10 * time.Second
	DefaultQoS         = 1
)

type Subscriber struct {
	client    paho.Client
	processor *processor.TelemetryProcessor
	topic     string
}

func NewSubscriber(cfg *config.Config, p *processor.TelemetryProcessor) *Subscriber {
	scheme := "tcp"
	if cfg.EMQXUseTLS {
		scheme = "ssl"
	}
	brokerURI := fmt.Sprintf("%s://%s:%s", scheme, cfg.EMQXHost, cfg.EMQXPort)

	opts := paho.NewClientOptions().
		AddBroker(brokerURI).
		SetClientID(cfg.MQTTClientID).
		SetUsername(cfg.MQTTClientID).
		SetPassword("system-secret").
		SetKeepAlive(MQTTKeepAlive).
		SetPingTimeout(MQTTPingTimeout).
		SetConnectTimeout(MQTTConnectTimeout).
		SetAutoReconnect(true).
		SetMaxReconnectInterval(5 * time.Second)

	if cfg.EMQXUseTLS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: cfg.EMQXTLSSkipVerify,
		}
		opts.SetTLSConfig(tlsConfig)
	}

	s := &Subscriber{
		processor: p,
		topic:     cfg.MQTTTopic,
	}

	opts.SetOnConnectHandler(func(c paho.Client) {
		log.Printf("[MQTT] Connected to EMQX broker at %s", brokerURI)
		s.subscribe(c)
	})

	opts.SetConnectionLostHandler(func(c paho.Client, err error) {
		log.Printf("[MQTT WARN] Connection lost to EMQX: %v. Reconnecting...", err)
	})

	s.client = paho.NewClient(opts)
	return s
}

// Start initiates connection to EMQX broker.
func (s *Subscriber) Start() error {
	token := s.client.Connect()
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}
	return nil
}

// Stop cleanly disconnects from MQTT broker.
func (s *Subscriber) Stop() {
	if s.client != nil && s.client.IsConnected() {
		s.client.Disconnect(250)
		log.Println("[MQTT] Disconnected from EMQX broker")
	}
}

// IsConnected returns whether the subscriber has an active MQTT connection.
func (s *Subscriber) IsConnected() bool {
	return s.client != nil && s.client.IsConnected()
}


func (s *Subscriber) subscribe(c paho.Client) {
	token := c.Subscribe(s.topic, DefaultQoS, func(_ paho.Client, msg paho.Message) {
		if err := s.processor.ProcessMessage(msg.Topic(), msg.Payload()); err != nil {
			log.Printf("[Ingest Error] Failed to process message on %s: %v", msg.Topic(), err)
			return
		}
		log.Printf("[Ingested] Successfully processed telemetry on %s", msg.Topic())
	})
	token.Wait()
	log.Printf("[MQTT] Subscribed to topic: %s", s.topic)
}
