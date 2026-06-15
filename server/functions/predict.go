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

type routeDirKey struct {
	subRouteUID string
	direction   int32
}

type travelAvgKey struct {
	subRouteUID string
	direction   int32
	stopUID     string
	hour        int
	dayOfWeek   int
}

func dedupRouteDirPairs(keys []routeDirKey) []routeDirKey {
	seen := make(map[routeDirKey]bool, len(keys))
	out := make([]routeDirKey, 0, len(keys))
	for _, k := range keys {
		if !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	return out
}

func batchNextDepartures(ctx context.Context, db *pgxpool.Pool, keys []routeDirKey, todTime string) map[routeDirKey]time.Time {
	out := make(map[routeDirKey]time.Time, len(keys))
	if len(keys) == 0 {
		return out
	}
	uids := make([]string, len(keys))
	dirs := make([]int32, len(keys))
	for i, k := range keys {
		uids[i] = k.subRouteUID
		dirs[i] = k.direction
	}
	rows, err := db.Query(ctx, `
		SELECT sub_route_uid, direction, MIN("arrival_time/StartTime") AS dep
		FROM bus_schedule
		WHERE type = true AND stopsequence = 0
		  AND "arrival_time/StartTime" >= $3::time
		  AND (sub_route_uid, direction) IN (
		      SELECT unnest($1::text[]), unnest($2::int[])
		  )
		GROUP BY sub_route_uid, direction`,
		uids, dirs, todTime)
	if err != nil {
		log.Printf("[MODEL] batchNextDepartures error: %v", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var uid string
		var dir int32
		var dep time.Time
		if err := rows.Scan(&uid, &dir, &dep); err == nil {
			out[routeDirKey{subRouteUID: uid, direction: dir}] = dep
		}
	}
	return out
}

func batchTravelAvg(ctx context.Context, db *pgxpool.Pool, uids []string, hour, dayOfWeek int) map[travelAvgKey]int {
	out := make(map[travelAvgKey]int)
	if len(uids) == 0 {
		return out
	}
	rows, err := db.Query(ctx, `
		SELECT sub_route_uid, direction, stop_uid, avg_seconds
		FROM bus_travel_avg
		WHERE hour = $2 AND day_of_week = $3
		  AND sub_route_uid = ANY($1::text[])`,
		uids, hour, dayOfWeek)
	if err != nil {
		log.Printf("[MODEL] batchTravelAvg error: %v", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var uid, stop string
		var dir int32
		var sec int
		if err := rows.Scan(&uid, &dir, &stop, &sec); err == nil {
			out[travelAvgKey{subRouteUID: uid, direction: dir, stopUID: stop, hour: hour, dayOfWeek: dayOfWeek}] = sec
		}
	}
	return out
}

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

type busStopCtx struct {
	subRouteUID  string
	direction    int32
	stopUID      string
	city         string
	stopSequence int
	totalStops   int
}

type predictionInputs struct {
	now          time.Time
	nextDep      time.Time
	travelAvg    int
	hasTravelAvg bool
	maxTravelAvg int
}

func predictNextBusTime(rc *redis.Client, stop busStopCtx, inputs predictionInputs) string {
	if etaModel == nil || inputs.nextDep.IsZero() {
		return ""
	}
	t := inputs.now.In(taipei)
	dep := time.Date(t.Year(), t.Month(), t.Day(),
		inputs.nextDep.Hour(), inputs.nextDep.Minute(), inputs.nextDep.Second(), 0, taipei)
	travelSec := inputs.travelAvg
	ratio := 0.0
	if stop.totalStops > 0 {
		ratio = float64(stop.stopSequence) / float64(stop.totalStops)
	}
	if !inputs.hasTravelAvg {
		if inputs.maxTravelAvg == 0 {
			return dep.Format(time.RFC3339)
		}
		travelSec = int(ratio * float64(inputs.maxTravelAvg))
	}
	var wd weatherData
	if wjson, wErr := rc.Get("weather:" + stop.city).Result(); wErr == nil {
		json.Unmarshal([]byte(wjson), &wd)
	}
	cityEnc := -1.0
	if v, ok := modelEncoders.City[stop.city]; ok {
		cityEnc = float64(v)
	}
	features := []float64{
		float64(t.Hour()),
		float64(t.Weekday()),
		boolToFloat64(isHoliday(t)),
		wd.Temperature,
		wd.Precipitation,
		wd.WindSpeed,
		wd.Humidity,
		float64(stop.direction),
		float64(stop.stopSequence),
		float64(stop.totalStops),
		ratio,
		cityEnc,
		-1,
		-1,
		-1,
	}
	correction := etaModel.PredictSingle(features, 0)
	eta := dep.Add(time.Duration(travelSec)*time.Second + time.Duration(correction)*time.Second)
	return eta.Format(time.RFC3339)
}

func boolToFloat64(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
