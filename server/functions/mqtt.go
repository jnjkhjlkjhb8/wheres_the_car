package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"
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

func startMQTT(rc *redis.Client, dispatcher *notificationDispatcher) mqtt.Client {
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
			mqttsubscribeall(c, rc, dispatcher)
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
func mqttsubscribeall(c mqtt.Client, rc *redis.Client, dispatcher *notificationDispatcher) {
	for _, t := range mqttTopics {
		pattern, ttl := t.pattern, t.ttl
		tok := c.Subscribe(pattern, 1, func(_ mqtt.Client, msg mqtt.Message) {
			mqtthandle(rc, msg, ttl, dispatcher)
		})
		tok.Wait()
		if err := tok.Error(); err != nil {
			log.Printf("[MQTT] subscribe failed topic=%s err=%v", pattern, err)
		} else {
			log.Printf("[MQTT] subscribed topic=%s", pattern)
		}
	}
}
func mqtthandle(rc *redis.Client, msg mqtt.Message, ttl time.Duration, dispatcher *notificationDispatcher) {
	key := "mqtt:" + strings.ReplaceAll(msg.Topic(), "/", ":")
	if err := rc.Set(key, msg.Payload(), ttl).Err(); err != nil {
		log.Printf("[MQTT] redis set failed key=%s err=%v", key, err)
		return
	}
	if err := rc.Publish(key, msg.Payload()).Err(); err != nil {
		log.Printf("[MQTT] redis publish failed key=%s err=%v", key, err)
	}
	dispatchRouteAlerts(context.Background(), routeAlerts(msg.Topic(), msg.Payload()), func(key string, ttl time.Duration) bool {
		ok, err := rc.SetNX("fcm:alert:"+key, "1", ttl).Result()
		return err == nil && ok
	}, dispatcher)
}

type normalizedRouteAlert struct{ routeType, routeKey, body, id string }

func dispatchRouteAlerts(ctx context.Context, alerts []normalizedRouteAlert, claim func(string, time.Duration) bool, dispatcher *notificationDispatcher) {
	for _, alert := range alerts {
		key := alert.id
		if key == "" {
			key = fmt.Sprintf("%x", sha256.Sum256([]byte(alert.routeType+"\x00"+alert.routeKey+"\x00"+alert.body)))
		}
		if claim(key, 5*time.Minute) {
			dispatcher.routeAlert(ctx, alert.routeType, alert.routeKey, alert.body)
		}
	}
}

func routeAlerts(topic string, payload []byte) []normalizedRouteAlert {
	routeType := ""
	switch {
	case strings.Contains(topic, "/Bus/"):
		routeType = "bus"
	case strings.Contains(topic, "/Metro/"):
		routeType = "mrt"
	case strings.Contains(topic, "/TRA/"):
		routeType = "tra"
	case strings.Contains(topic, "/THSR/"):
		routeType = "thsr"
	}
	if routeType == "" {
		return nil
	}
	var raw any
	if json.Unmarshal(payload, &raw) != nil {
		return nil
	}
	items, ok := raw.([]any)
	if !ok {
		items = []any{raw}
	}
	seen := map[string]struct{}{}
	out := make([]normalizedRouteAlert, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		key := firstString(m, routeKeyFields(routeType)...)
		if key == "" {
			continue
		}
		body := firstString(m, "Description", "NewsContent", "AlertDescription", "Message")
		if body == "" {
			body = firstString(m, "NewsTitle", "Title")
		}
		if body == "" {
			continue
		}
		dedupe := key + "\x00" + body
		if _, ok := seen[dedupe]; ok {
			continue
		}
		seen[dedupe] = struct{}{}
		out = append(out, normalizedRouteAlert{routeType: routeType, routeKey: key, body: body, id: firstString(m, "NewsID", "AlertID", "UpdateTime")})
	}
	return out
}

func routeKeyFields(routeType string) []string {
	switch routeType {
	case "bus":
		return []string{"SubRouteUID"}
	case "mrt":
		return []string{"LineID"}
	case "tra", "thsr":
		return []string{"LineID"}
	default:
		return nil
	}
}
func firstString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := m[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
