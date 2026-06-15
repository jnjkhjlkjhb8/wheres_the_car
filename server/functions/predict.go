package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/dmitryikh/leaves"
	"github.com/go-redis/redis"
	"github.com/jackc/pgx/v5/pgxpool"
)

var etaModel *leaves.Ensemble

var modelEncoders struct {
	City      map[string]int `json:"city"`
	PlateNumb map[string]int `json:"plate_numb"`
}

func loadModel() {
	path := os.Getenv("BUS_ETA_MODEL_PATH")
	if path == "" {
		path = "./model/bus_eta.json"
	}
	m, err := leaves.XGEnsembleFromFile(path, true)
	if err != nil {
		log.Printf("[MODEL] not loaded (file: %s): %v", path, err)
		return
	}
	encPath := strings.TrimSuffix(path, ".json") + "_encoders.json"
	encData, err := os.ReadFile(encPath)
	if err != nil {
		log.Printf("[MODEL] encoders not found at %s: %v", encPath, err)
	} else {
		json.Unmarshal(encData, &modelEncoders)
	}
	etaModel = m
	log.Printf("[MODEL] loaded from %s", path)
}

func fillNextBusTime(
	ctx context.Context,
	db *pgxpool.Pool,
	rc *redis.Client,
	subRouteUID string,
	direction int32,
	stopUID string,
	city string,
	stopSequence int,
	totalStops int,
	now time.Time,
) string {
	if etaModel == nil {
		return ""
	}

	todTime := now.In(taipei).Format("15:04:05")
	var depTimePG time.Time
	err := db.QueryRow(ctx, `
		SELECT "arrival_time/StartTime"
		FROM bus_schedule
		WHERE sub_route_uid = $1 AND direction = $2
		  AND type = true AND stopsequence = 0
		  AND "arrival_time/StartTime" >= $3::time
		ORDER BY "arrival_time/StartTime" ASC LIMIT 1`,
		subRouteUID, direction, todTime).Scan(&depTimePG)
	if err != nil {
		return ""
	}
	t := now.In(taipei)
	nextDep := time.Date(t.Year(), t.Month(), t.Day(),
		depTimePG.Hour(), depTimePG.Minute(), depTimePG.Second(), 0, taipei)

	var travelSec int
	err = db.QueryRow(ctx, `
		SELECT avg_seconds FROM bus_travel_avg
		WHERE sub_route_uid = $1 AND direction = $2 AND stop_uid = $3
		  AND hour = $4 AND day_of_week = $5`,
		subRouteUID, direction, stopUID, t.Hour(), int(t.Weekday())).Scan(&travelSec)
	if err != nil {
		var maxSec int
		err2 := db.QueryRow(ctx, `
			SELECT COALESCE(MAX(avg_seconds), 0) FROM bus_travel_avg
			WHERE sub_route_uid = $1 AND direction = $2`,
			subRouteUID, direction).Scan(&maxSec)
		if err2 != nil || maxSec == 0 {
			return nextDep.Format(time.RFC3339)
		}
		ratio := 0.0
		if totalStops > 0 {
			ratio = float64(stopSequence) / float64(totalStops)
		}
		travelSec = int(ratio * float64(maxSec))
	}

	var wd weatherData
	if wjson, wErr := rc.Get("weather:" + city).Result(); wErr == nil {
		json.Unmarshal([]byte(wjson), &wd)
	}

	cityEnc := -1.0
	if v, ok := modelEncoders.City[city]; ok {
		cityEnc = float64(v)
	}
	ratio := 0.0
	if totalStops > 0 {
		ratio = float64(stopSequence) / float64(totalStops)
	}
	features := []float64{
		float64(t.Hour()),
		float64(t.Weekday()),
		boolToFloat64(isHoliday(t)),
		wd.Temperature,
		wd.Precipitation,
		wd.WindSpeed,
		wd.Humidity,
		float64(direction),
		float64(stopSequence),
		float64(totalStops),
		ratio,
		cityEnc,
		-1,
		-1,
		-1,
	}
	correction := etaModel.PredictSingle(features, 0)
	eta := nextDep.Add(
		time.Duration(travelSec)*time.Second +
			time.Duration(correction)*time.Second,
	)
	return eta.Format(time.RFC3339)
}

func boolToFloat64(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
