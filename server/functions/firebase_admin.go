package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"golang.org/x/oauth2/google"
)

type fcmSender interface {
	Send(context.Context, *messaging.Message) error
}

type firebaseSender struct{ client *messaging.Client }

func (s firebaseSender) Send(ctx context.Context, message *messaging.Message) error {
	_, err := s.client.Send(ctx, message)
	return err
}

func firebaseEnabled() bool {
	return strings.EqualFold(os.Getenv("FIREBASE_ENABLED"), "true") && !strings.EqualFold(os.Getenv("APP_ENV"), "dev")
}

func newFirebaseSender(ctx context.Context) (fcmSender, error) {
	if !firebaseEnabled() {
		return nil, nil
	}
	projectID := strings.TrimSpace(os.Getenv("FIREBASE_PROJECT_ID"))
	if projectID == "" {
		return nil, fmt.Errorf("FIREBASE_PROJECT_ID is required when Firebase is enabled")
	}
	if _, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/firebase.messaging"); err != nil {
		return nil, fmt.Errorf("load Firebase credentials: %w", err)
	}
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("initialize Firebase Admin: %w", err)
	}
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialize Firebase Messaging: %w", err)
	}
	return firebaseSender{client: client}, nil
}
