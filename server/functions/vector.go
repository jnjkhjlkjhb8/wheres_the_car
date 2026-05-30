package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
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
	for _, table := range tables {
		func() {
			var input []string
			var inrow []resp
			switch table {
			case "bus_subroutes":
				rows, _ := db.Query(ctx, `SELECT sub_route_uid, sub_route_name,city,depart,destin FROM bus_static WHERE updated_at >= $1;`, since)
				defer rows.Close()
				row, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[BusSubroutes])
				if err != nil {
					log.Println(err.Error())
				}
				for _, temp := range row {
					text := fmt.Sprintf("類型：公車路線 子路線UID：%s 路線名：%s 縣市：%s 起點站：%s 終點站：%s ",
						temp.SubRouteUid,
						temp.SubRouteName,
						temp.City,
						temp.Departure,
						temp.Destination,
					)
					input = append(input, text)
					inrow = append(inrow, resp{
						Type:         "bus_route",
						UID:          temp.SubRouteUid,
						Name:         temp.SubRouteName,
						City:         temp.City,
						DepartSystem: temp.Departure,
						Destin:       temp.Destination,
					})
				}
			case "bus_stations":
				rows, _ := db.Query(ctx, `SELECT DISTINCT ON (station_name) station_uid, station_name, city, ST_AsText(position) FROM bus_stations WHERE updated_at >= $1;`, since)
				defer rows.Close()
				row, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[BusStationData])
				if err != nil {
					log.Println(err.Error())
				}
				for _, temp := range row {
					text := fmt.Sprintf("類型：公車站牌 站點UID：%s 站點名稱：%s 縣市：%s 位置：%s",
						temp.StationUid,
						temp.StationName,
						temp.City,
						temp.Geom,
					)
					input = append(input, text)
					inrow = append(inrow, resp{
						Type: "bus_station",
						UID:  temp.StationUid,
						Name: temp.StationName,
						City: temp.City,
						Geom: temp.Geom,
					})
				}
			case "bike_stations":
				rows, _ := db.Query(ctx, `SELECT station_uid, name, city, ST_AsText(geom) FROM bike_stations WHERE updated_at >= $1;`, since)
				defer rows.Close()
				row, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[BikeStationData])
				if err != nil {
					log.Println(err.Error())
				}
				for _, temp := range row {
					text := fmt.Sprintf("類型：公共自行車租借站 站點UID：%s 站點名稱：%s 縣市：%s 位置：%s",
						temp.StationUid,
						temp.StationName,
						temp.City,
						temp.Position,
					)
					input = append(input, text)
					inrow = append(inrow, resp{
						Type: "bike_station",
						UID:  temp.StationUid,
						Name: temp.StationName,
						City: temp.City,
						Geom: temp.Position,
					})
				}
			case "mrt_station":
				rows, _ := db.Query(ctx, `SELECT station_id, name, system, ST_AsText(stationposition) FROM mrt_station WHERE updated_at >= $1;`, since)
				defer rows.Close()
				row, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[MrtStationData])
				if err != nil {
					log.Println(err.Error())
				}
				for _, temp := range row {
					text := fmt.Sprintf("類型：捷運站 車站UID：%s 車站名稱：%s 捷運系統：%s 位置：%s",
						temp.StationId,
						temp.Name,
						temp.System,
						temp.Position,
					)
					input = append(input, text)
					inrow = append(inrow, resp{
						Type: "mrt_station",
						UID:  temp.StationId,
						Name: temp.Name,
						City: temp.System,
						Geom: temp.Position,
					})
				}
			case "thsr_stations":
				rows, _ := db.Query(ctx, `SELECT station_id, name, city, ST_AsText(geom) FROM thsr_stations WHERE updated_at >= $1;`, since)
				defer rows.Close()
				row, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[RailStationData])
				if err != nil {
					log.Println(err.Error())
				}
				for _, temp := range row {
					text := fmt.Sprintf("類型：高鐵車站 車站UID：%s 車站名稱：%s 縣市：%s 位置：%s",
						temp.StationId,
						temp.Name,
						temp.City,
						temp.Geom,
					)
					input = append(input, text)
					inrow = append(inrow, resp{
						Type: "thsr_station",
						UID:  temp.StationId,
						Name: temp.Name,
						City: temp.City,
						Geom: temp.Geom,
					})
				}
			case "tra_stations":
				rows, _ := db.Query(ctx, `SELECT station_id, name, city, ST_AsText(geom) FROM tra_stations WHERE updated_at >= $1;`, since)
				defer rows.Close()
				row, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[RailStationData])
				if err != nil {
					log.Println(err.Error())
				}
				for _, temp := range row {
					text := fmt.Sprintf("類型：台鐵車站 車站UID：%s 車站名稱：%s 縣市：%s 位置：%s",
						temp.StationId,
						temp.Name,
						temp.City,
						temp.Geom,
					)
					input = append(input, text)
					inrow = append(inrow, resp{
						Type: "tra_station",
						UID:  temp.StationId,
						Name: temp.Name,
						City: temp.City,
						Geom: temp.Geom,
					})
				}
			}
			long := len(input)
			for i := 0; i < long; i += size {
				end := i + size
				if end > long {
					end = long
				}
				temp1 := input[i:end]
				temp2 := inrow[i:end]
				body, comp, err, flipopen := callhf(temp1)
				if err != nil || !comp {
					log.Printf("[vector] action=vector event=skip reason=api_error,error=%s", err)
					return
				}
				var resp [][]float32
				if err := json.Unmarshal(body, &resp); err != nil {
					log.Printf("[vector] action=vector event=unmarshal_error error=%v", err)
					continue
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
				for j, temp := range resp {
					d := temp2[j]
					b.Queue(c1, d.Type, d.UID, d.Name, d.City, d.DepartSystem, d.Destin, d.Geom, zip(temp))
				}
				batchResults := db.SendBatch(ctx, b)
				if err := batchResults.Close(); err != nil {
					log.Printf("[vector] action=vector event=batch_error error=%v", err)
				}
				log.Printf("[vector] action=vector event=success table=%s count=%d", table, len(resp))
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
		SetHeader("Content-Type", "application/json").
		SetAuthToken(os.Getenv("HF_TOKEN"))
	requestBody := Request{
		Inputs: input,
	}
	resp, err := client.R().
		SetBody(requestBody).
		Post("https://router.huggingface.co/hf-inference/models/BAAI/bge-large-zh-v1.5/pipeline/feature-extraction")
	if err != nil {
		return nil, false, err, nil
	}
	return resp.Body(), true, nil, func() {
		err := resp.RawResponse.Body.Close()
		if err != nil {
			log.Printf("[HF] action=fail-close-response error=%v", err)
		}
	}
}
