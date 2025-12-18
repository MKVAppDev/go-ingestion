package dnse

import (
	"crypto/tls"
	"encoding/json"
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
		SetTLSConfig(&tls.Config{
			InsecureSkipVerify: true,
		})

	opts.OnConnect = func(c mqtt.Client) {
		log.Println("Connected to MQTT Broker")

		topic := "plaintext/quotes/krx/mdds/stockinfo/v1/roundlot/symbol/fpt"

		if token := c.Subscribe(topic, 1, onMessage); token.Wait() && token.Error() != nil {
			log.Printf("Subscribe error: %v", token.Error())
		} else {
			log.Printf("Subscribed to %s", topic)
		}
	}

	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		log.Printf("Connection lost: %v", err)
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("connect error: %v", token.Error())
	}

	select {}
}

func onMessage(client mqtt.Client, msg mqtt.Message) {
	var payload map[string]any
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		return
	}

	symbol, _ := payload["symbol"].(string)
	matchPrice := payload["matchPrice"]
	matchQtty := payload["matchQtty"]
	side, _ := payload["side"].(string)
	sendingTime, _ := payload["sendingTime"].(string)

	fmt.Printf("%v: %v - Match Quantity: %v - Side: %s - Time: %s\n",
		symbol, matchPrice, matchQtty, side, sendingTime)
}
