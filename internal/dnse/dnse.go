package dnse

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	auth "github.com/MKVAppDev/go-ingestion/internal/dnseauth"
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

	mqttClient mqtt.Client
	tickersMu  sync.Mutex
	tickers    map[string]struct{}

	username   string
	password   string
	investorID string
	token      string
	tokenMu    sync.RWMutex
}

func NewClient(pub *redispub.Publisher, env string, workers int, buffer int) *Client {
	return &Client{
		publisher: pub,
		env:       env,
		source:    "dnse",
		market:    "krx",
		msgCh:     make(chan Msg, buffer),
		workers:   workers,
		tickers:   make(map[string]struct{}),
	}
}

func (c *Client) Run(username, password, investorID, token string) error {
	c.username = username
	c.password = password
	c.investorID = investorID
	c.setToken(token)

	rand.Seed(time.Now().UnixNano())
	clientID := fmt.Sprintf("%s%d", clientPrefix, rand.Intn(1000)+1000)
	brokerURL := fmt.Sprintf("wss://%s:%d/wss", brokerHost, brokerPort)

	for i := 0; i < c.workers; i++ {
		go c.worker(i)
	}

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
		log.Println("✅ Connected to MQTT Broker")

		c.tickersMu.Lock()
		defer c.tickersMu.Unlock()

		for symbol := range c.tickers {
			c.subscribeSymbol(m, symbol)
		}
	}

	opts.SetDefaultPublishHandler(func(c mqtt.Client, m mqtt.Message) {
		log.Printf("🔔 Default handler - Message on topic: %s, payload: %s", m.Topic(), string(m.Payload()))
	})

	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		log.Printf("❌ Connection lost: %v - Will attempt to reconnect...", err)
	}

	opts.OnReconnecting = func(mqttClient mqtt.Client, opts *mqtt.ClientOptions) {
		log.Println("🔄 Attempting to reconnect to MQTT Broker...")

		reconnectTimeout := time.After(10 * time.Second)
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-reconnectTimeout:
				log.Println("⏱️ Reconnect timeout - attempting to refresh token...")
				if err := c.refreshToken(); err != nil {
					log.Printf("❌ Failed to refresh token: %v", err)
					return
				}

				currentToken := c.getToken()
				opts.SetUsername(c.investorID)
				opts.SetPassword(currentToken)
				log.Println("🔑 Token refreshed successfully, reconnecting with new credentials...")
				return

			case <-ticker.C:
				if mqttClient.IsConnected() {
					log.Println("✅ Reconnected successfully!")
					return
				}
			}
		}
	}

	c.mqttClient = mqtt.NewClient(opts)
	if token := c.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("connect error: %v", token.Error())
	}

	c.listenTickerEvents()

	log.Println("🟢 MQTT Client is running. Press Ctrl+C to exit.")
	select {}
}

func (c *Client) setToken(token string) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	c.token = token
}

func (c *Client) getToken() string {
	c.tokenMu.RLock()
	defer c.tokenMu.RUnlock()
	return c.token
}

func (c *Client) refreshToken() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Printf("🔐 Authenticating with username: %s", c.username)
	newToken, err := auth.Authentication(ctx, c.username, c.password)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	info, err := auth.GetInvestorInfo(ctx, newToken)
	if err != nil {
		return fmt.Errorf("get investor info failed: %w", err)
	}

	if info.InvestorID != c.investorID {
		return fmt.Errorf("investor ID mismatch: expected %s, got %s", c.investorID, info.InvestorID)
	}

	c.setToken(newToken)
	log.Println("✅ Token refreshed and verified successfully")
	return nil
}

func (c *Client) subscribeSymbol(m mqtt.Client, symbol string) {

	topics := []string{
		"plaintext/quotes/krx/mdds/stockinfo/v1/roundlot/symbol/" + symbol,
		"plaintext/quotes/krx/mdds/topprice/v1/roundlot/symbol/" + symbol,
		"plaintext/quotes/krx/mdds/v2/ohlc/stock/1/" + symbol,
		"plaintext/quotes/krx/mdds/tick/v1/roundlot/symbol/" + symbol,
	}

	for _, topic := range topics {
		if token := m.Subscribe(topic, 1, c.onMessage); token.Wait() && token.Error() != nil {
			log.Printf("❌ Subscribe error for %s: %v", topic, token.Error())
		}
	}

	log.Printf("✅ Subscribed to %s", symbol)
}

func (c *Client) unsubscribeSymbol(m mqtt.Client, symbol string) {
	topics := []string{
		"plaintext/quotes/krx/mdds/stockinfo/v1/roundlot/symbol/" + symbol,
		"plaintext/quotes/krx/mdds/topprice/v1/roundlot/symbol/" + symbol,
		"plaintext/quotes/krx/mdds/v2/ohlc/stock/1/" + symbol,
		"plaintext/quotes/krx/mdds/tick/v1/roundlot/symbol/" + symbol,
	}

	if token := m.Unsubscribe(topics...); token.Wait() && token.Error() != nil {
		log.Printf("❌ Unsubscribe error for %v: %v", topics, token.Error())
	}

	log.Printf("⛔ Unsubscribed %s", symbol)
}

type tickerEvent struct {
	Symbol string `json:"symbol"`
	Active bool   `json:"active"`
}

func (c *Client) listenTickerEvents() {
	channel := fmt.Sprintf("%s.dnse.krx.tickers.events", c.env)

	go func() {
		ctx := context.Background()
		pubsub := c.publisher.Subscribe(ctx, channel)
		defer pubsub.Close()

		ch := pubsub.Channel()
		log.Printf("listening ticker events on %s", channel)

		for msg := range ch {
			var evt tickerEvent
			if err := json.Unmarshal([]byte(msg.Payload), &evt); err != nil {
				log.Printf("ticker event invalid: %v", err)
				continue
			}

			sym := strings.ToUpper(strings.TrimSpace(evt.Symbol))
			if sym == "" {
				continue
			}

			if evt.Active == true {
				c.tickersMu.Lock()
				_, existed := c.tickers[sym]
				if !existed {
					c.tickers[sym] = struct{}{}
				}
				c.tickersMu.Unlock()

				if !existed && c.mqttClient != nil && c.mqttClient.IsConnected() {
					c.subscribeSymbol(c.mqttClient, sym)
				}
			} else {
				c.tickersMu.Lock()
				_, existed := c.tickers[sym]
				if existed {
					delete(c.tickers, sym)
				}
				c.tickersMu.Unlock()

				if existed && c.mqttClient != nil && c.mqttClient.IsConnected() {
					c.unsubscribeSymbol(c.mqttClient, sym)
				}
			}
		}
	}()
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
