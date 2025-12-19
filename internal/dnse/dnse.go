package dnse

import (
	"crypto/tls"
	"fmt"
	"log"
	"math/rand"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	brokerHost   = "datafeed-lts-krx.dnse.com.vn"
	brokerPort   = 443
	clientPrefix = "dnse-user-client-mkv-"
)

func Run(investorID, token string) error {
	rand.Seed(time.Now().UnixNano())

	clientID := fmt.Sprintf("%s%d", clientPrefix, rand.Intn(1000)+1000)

	brokerURL := fmt.Sprintf("wss://%s:%d/wss", brokerHost, brokerPort)

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
			InsecureSkipVerify: false,
		})

	opts.OnConnect = func(c mqtt.Client) {
		log.Println("Connected to MQTT Broker")

		symbol := "FPT"

		topics := []string{
			"plaintext/quotes/krx/mdds/stockinfo/v1/roundlot/symbol/" + symbol,
			"plaintext/quotes/krx/mdds/topprice/v1/roundlot/symbol/" + symbol,
			"plaintext/quotes/krx/mdds/v2/ohlc/stock/1/" + symbol,
			"plaintext/quotes/krx/mdds/tick/v1/roundlot/symbol/" + symbol,
		}

		for _, topic := range topics {
			if token := c.Subscribe(topic, 1, onMessage); token.Wait() && token.Error() != nil {
				log.Printf("❌ Subscribe error for %s: %v", topic, token.Error())
			} else {
				log.Printf("✅ Subscribed to %s", topic)
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

func onMessage(client mqtt.Client, msg mqtt.Message) {
	log.Printf("📨 Message received on topic: %s", msg.Topic())
	log.Printf("📦 Payload length: %d bytes", len(msg.Payload()))
	log.Printf("📄 Raw payload: %s", string(msg.Payload()))
}
