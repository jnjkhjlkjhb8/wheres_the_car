package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/go-redis/redis"
	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BusSubroutes struct {
	SubRouteUid  string `db:"sub_route_uid"`
	SubRouteName string `db:"sub_route_name"`
	City         string `db:"city"`
	Departure    string `db:"depart"`
	Destination  string `db:"destin"`
}
type BusStationData struct {
	StationUid  string `db:"station_uid"`
	StationName string `db:"station_name"`
	City        string `db:"city"`
	Geom        string `db:"st_astext"`
}
type BikeStationData struct {
	StationUid  string `db:"station_uid"`
	StationName string `db:"name"`
	City        string `db:"city"`
	Position    string `db:"st_astext"`
}
type MrtStationData struct {
	Position  string `db:"st_astext"`
	System    string `db:"system"`
	Name      string `db:"name"`
	StationId string `db:"station_id"`
}
type RailStationData struct {
	StationId string `db:"station_id"`
	Name      string `db:"name"`
	City      string `db:"city"`
	Geom      string `db:"st_astext"`
}
type resp struct {
	Type         string
	UID          string
	Name         string
	City         string
	DepartSystem string
	Destin       string
	Geom         string
	Blob         []byte
}
type Request struct {
	Inputs []string `json:"inputs"`
}

const size = 1024

func changetovector(ctx context.Context, rc *redis.Client, db *pgxpool.Pool) {
	since, _ := rc.Get("LastTimeUpdate").Result()
	if since == "" {
		since = time.Time{}.Format(time.RFC3339)
	}
	tables := []string{"bus_subroutes", "bus_stations", "bike_stations", "mrt_station", "tra_stations", "thsr_stations"}
	processBatch := func(table string, input []string, inrow []resp) bool {
		if len(input) == 0 {
			return true
		}
		body, comp, err, flipopen := callhf(input)
		if err != nil || !comp {
			log.Printf("[vector] action=vector event=skip reason=api_error,error=%s", err)
			return false
		}
		var resp [][]float32
		if err := json.Unmarshal(body, &resp); err != nil {
			log.Printf("[vector] action=vector event=unmarshal_error error=%v", err)
			flipopen()
			return false
		}
		flipopen()
		b := &pgx.Batch{}
		c1 := `INSERT INTO search_vector(
					type,
					uid,
					name,
					city,
					depart,
					destin,
					geom,
					blob,
					updated_at
				)
				VALUES ($1, $2, $3, $4, $5, $6,CASE WHEN $7 = '' THEN NULL ELSE ST_GeomFromText($7, 4326) END,$8,NOW())
				ON CONFLICT (type,uid,city)
				DO UPDATE SET name = EXCLUDED.name,
					depart = EXCLUDED.depart,
					destin = EXCLUDED.destin,
					geom = EXCLUDED.geom,
					blob = EXCLUDED.blob,
					updated_at = NOW();`
		for i, v := range resp {
			d := inrow[i]
			b.Queue(c1, d.Type, d.UID, d.Name, d.City, d.DepartSystem, d.Destin, d.Geom, zip(v))
		}
		batchResults := db.SendBatch(ctx, b)
		if err := batchResults.Close(); err != nil {
			log.Printf("[vector] action=vector event=batch_error error=%v", err)
			return false
		}
		log.Printf("[vector] action=vector event=success table=%s count=%d", table, len(resp))
		return true
	}
	for _, table := range tables {
		func() {
			switch table {
			case "bus_subroutes":
				rows, _ := db.Query(ctx, `SELECT sub_route_uid, sub_route_name,city,depart,destin FROM bus_static WHERE updated_at >= $1;`, since)
				defer rows.Close()
				input := make([]string, 0, size)
				inrow := make([]resp, 0, size)
				for rows.Next() {
					var uid, name, city, depart, destin string
					if err := rows.Scan(&uid, &name, &city, &depart, &destin); err != nil {
						log.Printf("[vector] action=vector event=scan_error error=%v", err)
						continue
					}
					text := fmt.Sprintf("類型：公車路線 子路線UID：%s 路線名：%s 縣市：%s 起點站：%s 終點站：%s ", uid, name, city, depart, destin)
					input = append(input, text)
					inrow = append(inrow, resp{
						Type:         "bus_route",
						UID:          uid,
						Name:         name,
						City:         city,
						DepartSystem: depart,
						Destin:       destin,
					})
					if len(input) >= size {
						if !processBatch(table, input, inrow) {
							return
						}
						input = input[:0]
						inrow = inrow[:0]
					}
				}
				if !processBatch(table, input, inrow) {
					return
				}
			case "bus_stations":
				rows, _ := db.Query(ctx, `SELECT DISTINCT ON (station_name) station_uid, station_name, city, ST_AsText(position) FROM bus_stations WHERE updated_at >= $1;`, since)
				defer rows.Close()
				input := make([]string, 0, size)
				inrow := make([]resp, 0, size)
				for rows.Next() {
					var uid, name, city, geom string
					if err := rows.Scan(&uid, &name, &city, &geom); err != nil {
						log.Printf("[vector] action=vector event=scan_error error=%v", err)
						continue
					}
					text := fmt.Sprintf("類型：公車站牌 站點UID：%s 站點名稱：%s 縣市：%s 位置：%s", uid, name, city, geom)
					input = append(input, text)
					inrow = append(inrow, resp{
						Type: "bus_station",
						UID:  uid,
						Name: name,
						City: city,
						Geom: geom,
					})
					if len(input) >= size {
						if !processBatch(table, input, inrow) {
							return
						}
						input = input[:0]
						inrow = inrow[:0]
					}
				}
				if !processBatch(table, input, inrow) {
					return
				}
			case "bike_stations":
				rows, _ := db.Query(ctx, `SELECT station_uid, name, city, ST_AsText(geom) FROM bike_stations WHERE updated_at >= $1;`, since)
				defer rows.Close()
				input := make([]string, 0, size)
				inrow := make([]resp, 0, size)
				for rows.Next() {
					var uid, name, city, geom string
					if err := rows.Scan(&uid, &name, &city, &geom); err != nil {
						log.Printf("[vector] action=vector event=scan_error error=%v", err)
						continue
					}
					text := fmt.Sprintf("類型：公共自行車租借站 站點UID：%s 站點名稱：%s 縣市：%s 位置：%s", uid, name, city, geom)
					input = append(input, text)
					inrow = append(inrow, resp{
						Type: "bike_station",
						UID:  uid,
						Name: name,
						City: city,
						Geom: geom,
					})
					if len(input) >= size {
						if !processBatch(table, input, inrow) {
							return
						}
						input = input[:0]
						inrow = inrow[:0]
					}
				}
				if !processBatch(table, input, inrow) {
					return
				}
			case "mrt_station":
				rows, _ := db.Query(ctx, `SELECT station_id, name, system, ST_AsText(stationposition) FROM mrt_station WHERE updated_at >= $1;`, since)
				defer rows.Close()
				input := make([]string, 0, size)
				inrow := make([]resp, 0, size)
				for rows.Next() {
					var uid, name, system, geom string
					if err := rows.Scan(&uid, &name, &system, &geom); err != nil {
						log.Printf("[vector] action=vector event=scan_error error=%v", err)
						continue
					}
					text := fmt.Sprintf("類型：捷運站 車站UID：%s 車站名稱：%s 捷運系統：%s 位置：%s", uid, name, system, geom)
					input = append(input, text)
					inrow = append(inrow, resp{
						Type: "mrt_station",
						UID:  uid,
						Name: name,
						City: system,
						Geom: geom,
					})
					if len(input) >= size {
						if !processBatch(table, input, inrow) {
							return
						}
						input = input[:0]
						inrow = inrow[:0]
					}
				}
				if !processBatch(table, input, inrow) {
					return
				}
			case "thsr_stations":
				rows, _ := db.Query(ctx, `SELECT station_id, name, city, ST_AsText(geom) FROM thsr_stations WHERE updated_at >= $1;`, since)
				defer rows.Close()
				input := make([]string, 0, size)
				inrow := make([]resp, 0, size)
				for rows.Next() {
					var uid, name, city, geom string
					if err := rows.Scan(&uid, &name, &city, &geom); err != nil {
						log.Printf("[vector] action=vector event=scan_error error=%v", err)
						continue
					}
					text := fmt.Sprintf("類型：高鐵車站 車站UID：%s 車站名稱：%s 縣市：%s 位置：%s", uid, name, city, geom)
					input = append(input, text)
					inrow = append(inrow, resp{
						Type: "thsr_station",
						UID:  uid,
						Name: name,
						City: city,
						Geom: geom,
					})
					if len(input) >= size {
						if !processBatch(table, input, inrow) {
							return
						}
						input = input[:0]
						inrow = inrow[:0]
					}
				}
				if !processBatch(table, input, inrow) {
					return
				}
			case "tra_stations":
				rows, _ := db.Query(ctx, `SELECT station_id, name, city, ST_AsText(geom) FROM tra_stations WHERE updated_at >= $1;`, since)
				defer rows.Close()
				input := make([]string, 0, size)
				inrow := make([]resp, 0, size)
				for rows.Next() {
					var uid, name, city, geom string
					if err := rows.Scan(&uid, &name, &city, &geom); err != nil {
						log.Printf("[vector] action=vector event=scan_error error=%v", err)
						continue
					}
					text := fmt.Sprintf("類型：台鐵車站 車站UID：%s 車站名稱：%s 縣市：%s 位置：%s", uid, name, city, geom)
					input = append(input, text)
					inrow = append(inrow, resp{
						Type: "tra_station",
						UID:  uid,
						Name: name,
						City: city,
						Geom: geom,
					})
					if len(input) >= size {
						if !processBatch(table, input, inrow) {
							return
						}
						input = input[:0]
						inrow = inrow[:0]
					}
				}
				if !processBatch(table, input, inrow) {
					return
				}
			}
		}()
	}
	rc.Set("LastTimeUpdate", time.Now().Format(time.RFC3339), 0)
	log.Printf("[vector] action=vector event=complete")
}
func zip(vector []float32) []byte {
	blob := make([]byte, len(vector)*4)
	for i, f := range vector {
		bit := math.Float32bits(f)
		binary.LittleEndian.PutUint32(blob[i*4:(i+1)*4], bit)
	}
	return blob
}
func callhf(input []string) ([]byte, bool, error, func()) {
	client := resty.New().
		SetHeader("Content-Type", "application/json")
	resp, err := client.R().
		SetBody(Request{Inputs: input}).
		Post("http://embed:8082/embed")
	if err != nil {
		return nil, false, err, nil
	}
	return resp.Body(), true, nil, func() {
		err := resp.RawResponse.Body.Close()
		if err != nil {
			log.Printf("[embed] action=fail-close-response error=%v", err)
		}
	}
}
