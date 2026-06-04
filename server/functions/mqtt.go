package main

import (
	"crypto/tls"
	"log"
	"os"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-redis/redis"
)

type mqttTopicCfg struct {
	pattern string
	ttl     time.Duration
}

var mqttTopics = []mqttTopicCfg{
	{"v2/Bus/RealTimeNearStop/City/#", 60 * time.Second},
	{"v2/Bus/News/City/+", 5 * time.Minute},
	{"v2/Bus/News/InterCity", 5 * time.Minute},
	{"v2/Rail/Metro/Alert/#", 5 * time.Minute},
	{"v3/Rail/TRA/Alert", 5 * time.Minute},
	{"v2/Rail/THSR/AlertInfo", 5 * time.Minute},
}

func startMQTT(rc *redis.Client) mqtt.Client {
	clientID := os.Getenv("MQTT_CLIENT_ID")
	username := os.Getenv("MQTT_USERNAME")
	password := os.Getenv("MQTT_PASSWORD")
	if clientID == "" || username == "" || password == "" {
		log.Println("[MQTT] credentials not set — skipping MQTT subscriber")
		return nil
	}
	opts := mqtt.NewClientOptions().
		AddBroker("mqtts://mqtt.transportdata.tw:8883").
		SetClientID(clientID).
		SetUsername(username).
		SetPassword(password).
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(10 * time.Second).
		SetTLSConfig(&tls.Config{}).
		SetOnConnectHandler(func(c mqtt.Client) {
			log.Println("[MQTT] connected")
			mqttsubscribeall(c, rc)
		}).
		SetConnectionLostHandler(func(_ mqtt.Client, err error) {
			log.Printf("[MQTT] connection lost: %v", err)
		})
	c := mqtt.NewClient(opts)
	tok := c.Connect()
	tok.Wait()
	if err := tok.Error(); err != nil {
		log.Printf("[MQTT] initial connect failed: %v — will auto-retry", err)
	}
	return c
}
func mqttsubscribeall(c mqtt.Client, rc *redis.Client) {
	for _, t := range mqttTopics {
		pattern, ttl := t.pattern, t.ttl
		tok := c.Subscribe(pattern, 1, func(_ mqtt.Client, msg mqtt.Message) {
			mqtthandle(rc, msg, ttl)
		})
		tok.Wait()
		if err := tok.Error(); err != nil {
			log.Printf("[MQTT] subscribe failed topic=%s err=%v", pattern, err)
		} else {
			log.Printf("[MQTT] subscribed topic=%s", pattern)
		}
	}
}
func mqtthandle(rc *redis.Client, msg mqtt.Message, ttl time.Duration) {
	key := "mqtt:" + strings.ReplaceAll(msg.Topic(), "/", ":")
	if err := rc.Set(key, msg.Payload(), ttl).Err(); err != nil {
		log.Printf("[MQTT] redis set failed key=%s err=%v", key, err)
		return
	}
	if err := rc.Publish(key, msg.Payload()).Err(); err != nil {
		log.Printf("[MQTT] redis publish failed key=%s err=%v", key, err)
	}
}
