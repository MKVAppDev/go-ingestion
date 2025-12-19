package dnse

import (
	"crypto/tls"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/MKVAppDev/go-ingestion/internal/redispub"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	brokerHost   = "datafeed-lts-krx.dnse.com.vn"
	brokerPort   = 443
	clientPrefix = "dnse-user-client-mkv-"
)

type Msg struct {
	Topic   string
	Payload []byte
}

type Client struct {
	publisher *redispub.Publisher
	env       string
	source    string
	market    string
	msgCh     chan Msg
	workers   int
}

func NewClient(pub *redispub.Publisher, env string, workers int, buffer int) *Client {
	return &Client{
		publisher: pub,
		env:       env,
		source:    "dnse",
		market:    "krx",
		msgCh:     make(chan Msg, buffer),
		workers:   workers,
	}
}

func (c *Client) Run(investorID, token string, tickers []string) error {
	rand.Seed(time.Now().UnixNano())
	clientID := fmt.Sprintf("%s%d", clientPrefix, rand.Intn(1000)+1000)
	brokerURL := fmt.Sprintf("wss://%s:%d/wss", brokerHost, brokerPort)

	for i := 0; i < c.workers; i++ {
		go c.worker(i)
	}

	channels := c.allChannels(tickers)

	done := make(chan struct{})
	go c.monitorSubscribers(done, channels)

	opts := mqtt.NewClientOptions().
		AddBroker(brokerURL).
		SetClientID(clientID).
		SetUsername(investorID).
		SetPassword(token).
		SetProtocolVersion(5).
		SetKeepAlive(60 * time.Second).
		SetPingTimeout(10 * time.Second).
		SetAutoReconnect(true).
		SetTLSConfig(&tls.Config{
			InsecureSkipVerify: true,
		})

	opts.OnConnect = func(m mqtt.Client) {
		log.Println("Connected to MQTT Broker")

		for _, symbol := range tickers {
			topics := []string{
				"plaintext/quotes/krx/mdds/stockinfo/v1/roundlot/symbol/" + symbol,
				"plaintext/quotes/krx/mdds/topprice/v1/roundlot/symbol/" + symbol,
				"plaintext/quotes/krx/mdds/v2/ohlc/stock/1/" + symbol,
				"plaintext/quotes/krx/mdds/tick/v1/roundlot/symbol/" + symbol,
			}

			for _, topic := range topics {
				if token := m.Subscribe(topic, 1, c.onMessage); token.Wait() && token.Error() != nil {
					log.Printf("❌ Subscribe error for %s: %v", topic, token.Error())
				} else {
					log.Printf("✅ Subscribed to %s", topic)
				}
			}
		}
	}

	opts.SetDefaultPublishHandler(func(c mqtt.Client, m mqtt.Message) {
		log.Printf("🔔 Default handler - Message on topic: %s, payload: %s", m.Topic(), string(m.Payload()))
	})

	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		log.Printf("❌ Connection lost: %v - Will attempt to reconnect...", err)
	}

	opts.OnReconnecting = func(c mqtt.Client, opts *mqtt.ClientOptions) {
		log.Println("🔄 Attempting to reconnect to MQTT Broker...")
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("connect error: %v", token.Error())
	}

	log.Println("🟢 MQTT Client is running. Press Ctrl+C to exit.")
	select {}
}

func (c *Client) onMessage(client mqtt.Client, msg mqtt.Message) {

	m := Msg{
		Topic:   msg.Topic(),
		Payload: append([]byte(nil), msg.Payload()...),
	}

	select {
	case c.msgCh <- m:
	default:
		log.Println("⚠️ msgCh full, dropping message")
	}
}

func (c *Client) worker(id int) {

	for m := range c.msgCh {
		datatype := mapDatatype(m.Topic)
		symbol := extractSymbol(m.Topic)

		if symbol == "" || datatype == "" {
			continue
		}

		channel := buildRedisChannel(c.env, c.source, c.market, datatype, symbol)

		err := c.publisher.Publish(channel, m.Payload)

		if err != nil {
			log.Printf("[worker %d] redis publish error: %v", id, err)
		}
	}
}

func (c *Client) allChannels(tickers []string) []string {
	var channels []string
	datatypes := []string{"tick", "ohlc", "topprice", "stockinfo"}

	for _, sym := range tickers {
		for _, dt := range datatypes {
			ch := buildRedisChannel(c.env, c.source, c.market, dt, sym)
			channels = append(channels, ch)
		}
	}

	return channels
}

func mapDatatype(topic string) string {
	switch {
	case strings.Contains(topic, "/tick/"):
		return "tick"
	case strings.Contains(topic, "/ohlc/"):
		return "ohlc"
	case strings.Contains(topic, "/topprice/"):
		return "topprice"
	case strings.Contains(topic, "/stockinfo/"):
		return "stockinfo"
	default:
		return ""
	}
}

func extractSymbol(topic string) string {
	parts := strings.Split(topic, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func buildRedisChannel(env, source, market, datatype, symbol string) string {
	// channel format as <env>.<source>.<market>.<datatype>.<symbol>
	// example: prod.dnse.krx.tick.FPT
	return fmt.Sprintf("%s.%s.%s.%s.%s", env, source, market, datatype, symbol)
}

func (c *Client) monitorSubscribers(done chan<- struct{}, channels []string) {

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	idleStart := time.Now()

	for range ticker.C {
		m, err := c.publisher.NumSubMany(channels...)
		if err != nil {
			log.Printf("monitor error PubSubNumSub %v", err)
			continue
		}

		var total int64
		for _, n := range m {
			total += n
		}

		if total == 0 {
			if time.Since(idleStart) >= 5*time.Minute {
				close(done)
				return
			}
		} else {
			idleStart = time.Now()
		}
	}

}
