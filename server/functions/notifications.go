package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"firebase.google.com/go/v4/messaging"
)

type notificationStorage interface {
	subscribedTokens(context.Context, string, string) ([]deviceToken, error)
	activeReminders(context.Context, string, string, string, string, time.Time) ([]arrivalReminder, error)
	claim(context.Context, string, time.Time) (bool, error)
	release(context.Context, string) (bool, error)
	fired(context.Context, string, time.Time) (bool, error)
	invalidate(context.Context, string) error
}

type notificationDispatcher struct {
	store  notificationStorage
	sender fcmSender
	now    func() time.Time
}

var isInvalidFCMToken = messaging.IsUnregistered

func newNotificationDispatcher(store notificationStorage, sender fcmSender) *notificationDispatcher {
	if sender == nil {
		return nil
	}
	return &notificationDispatcher{store: store, sender: sender, now: time.Now}
}

func notificationMessage(token, title, body string, data map[string]string) *messaging.Message {
	data["title"] = title
	data["body"] = body
	return &messaging.Message{Token: token, Data: data, Notification: &messaging.Notification{Title: title, Body: body}, Android: &messaging.AndroidConfig{Priority: "high", Notification: &messaging.AndroidNotification{Sound: "default"}}, APNS: &messaging.APNSConfig{Payload: &messaging.APNSPayload{Aps: &messaging.Aps{Sound: "default"}}}}
}

func (d *notificationDispatcher) routeAlert(ctx context.Context, routeType, routeKey, body string) {
	if d == nil || routeType != "bus" || routeKey == "" {
		return
	}
	tokens, err := d.store.subscribedTokens(ctx, routeType, routeKey)
	if err != nil {
		log.Printf("[FCM] route subscriptions: %v", err)
		return
	}
	seen := map[string]struct{}{}
	for _, v := range tokens {
		if _, ok := seen[v.token]; ok {
			continue
		}
		seen[v.token] = struct{}{}
		err = d.sender.Send(ctx, notificationMessage(v.token, "路線異常", body, map[string]string{"kind": "route_alert", "route_type": routeType, "route_key": routeKey}))
		if isInvalidFCMToken(err) {
			_ = d.store.invalidate(ctx, v.token)
		} else if err != nil {
			log.Printf("[FCM] route alert send: %v", err)
		}
	}
}

func (d *notificationDispatcher) arrival(ctx context.Context, routeType, routeKey, stopKey, direction string, etaSeconds int32) {
	if d == nil || routeType != "bus" || etaSeconds < 0 {
		return
	}
	now := d.now()
	reminders, err := d.store.activeReminders(ctx, routeType, routeKey, stopKey, direction, now)
	if err != nil {
		log.Printf("[FCM] arrival reminders: %v", err)
		return
	}
	for _, r := range reminders {
		if etaSeconds > int32(r.leadMinutes*60) {
			continue
		}
		claimed, err := d.store.claim(ctx, r.id, now)
		if err != nil || !claimed {
			continue
		}
		msg := notificationMessage(r.token, "即將到站", fmt.Sprintf("預計 %d 分鐘後到站", (etaSeconds+59)/60), map[string]string{"kind": "arrival_reminder", "route_type": r.routeType, "route_key": r.routeKey, "stop_key": r.stopKey, "direction": r.direction, "lead_minutes": strconv.Itoa(r.leadMinutes)})
		err = d.sender.Send(ctx, msg)
		if err != nil {
			if isInvalidFCMToken(err) {
				_ = d.store.invalidate(ctx, r.token)
			}
			continue
		}
		if _, err = d.store.fired(ctx, r.id, now); err != nil {
			log.Printf("[FCM] mark reminder fired: %v", err)
		}
	}
}
