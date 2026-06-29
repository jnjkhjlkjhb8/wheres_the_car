package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type notificationDB interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

type notificationStore struct{ db notificationDB }

type deviceToken struct{ token string }

type arrivalReminder struct {
	id, token, routeType, routeKey, stopKey, direction string
	leadMinutes                                        int
}

func (s notificationStore) subscribedTokens(ctx context.Context, routeType, routeKey string) ([]deviceToken, error) {
	rows, err := s.db.Query(ctx, `SELECT d.fcm_token FROM firebase_route_subscription s JOIN firebase_device d ON d.install_id=s.install_id WHERE s.route_type=$1 AND s.route_key=$2 AND d.push_enabled AND d.fcm_token<>''`, routeType, routeKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []deviceToken
	for rows.Next() {
		var v deviceToken
		if err := rows.Scan(&v.token); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s notificationStore) activeReminders(ctx context.Context, routeType, routeKey, stopKey, direction string, now time.Time) ([]arrivalReminder, error) {
	rows, err := s.db.Query(ctx, `SELECT r.reminder_id,d.fcm_token,r.route_type,r.route_key,r.stop_key,r.direction,r.lead_minutes FROM firebase_arrival_reminder r JOIN firebase_device d ON d.install_id=r.install_id WHERE r.route_type=$1 AND r.route_key=$2 AND r.stop_key=$3 AND r.direction=$4 AND r.status='pending' AND r.expires_at>$5 AND d.push_enabled AND d.fcm_token<>''`, routeType, routeKey, stopKey, direction, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []arrivalReminder
	for rows.Next() {
		var v arrivalReminder
		if err := rows.Scan(&v.id, &v.token, &v.routeType, &v.routeKey, &v.stopKey, &v.direction, &v.leadMinutes); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func rowsChanged(tag pgconn.CommandTag, err error) (bool, error) { return tag.RowsAffected() == 1, err }
func (s notificationStore) claim(ctx context.Context, id string, now time.Time) (bool, error) {
	tag, err := s.db.Exec(ctx, `UPDATE firebase_arrival_reminder SET status='sending',updated_at=NOW() WHERE reminder_id=$1 AND status='pending' AND expires_at>$2`, id, now)
	return rowsChanged(tag, err)
}
func (s notificationStore) release(ctx context.Context, id string) (bool, error) {
	tag, err := s.db.Exec(ctx, `UPDATE firebase_arrival_reminder SET status='pending',updated_at=NOW() WHERE reminder_id=$1 AND status='sending'`, id)
	return rowsChanged(tag, err)
}
func (s notificationStore) fired(ctx context.Context, id string, now time.Time) (bool, error) {
	tag, err := s.db.Exec(ctx, `UPDATE firebase_arrival_reminder SET status='fired',fired_at=$2,updated_at=NOW() WHERE reminder_id=$1 AND status='sending'`, id, now)
	return rowsChanged(tag, err)
}
func (s notificationStore) invalidate(ctx context.Context, token string) error {
	_, err := s.db.Exec(ctx, `UPDATE firebase_device SET fcm_token='',push_enabled=FALSE,updated_at=NOW() WHERE fcm_token=$1`, token)
	return err
}
