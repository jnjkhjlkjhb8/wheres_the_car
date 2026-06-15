package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type busSubroutes struct {
	SubRouteUid  string `db:"sub_route_uid"`
	SubRouteName string `db:"sub_route_name"`
	City         string `db:"city"`
	Departure    string `db:"depart"`
	Destination  string `db:"destin"`
}
type busStationData struct {
	StationUid  string `db:"station_uid"`
	StationName string `db:"station_name"`
	City        string `db:"city"`
	Geom        string `db:"st_astext"`
}
type bikeStationData struct {
	StationUid  string `db:"station_uid"`
	StationName string `db:"name"`
	City        string `db:"city"`
	Position    string `db:"st_astext"`
}
type mrtStationData struct {
	Position  string `db:"st_astext"`
	System    string `db:"system"`
	Name      string `db:"name"`
	StationId string `db:"station_id"`
}
type railStationData struct {
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
}

var cityNames = map[string]string{
	"Taipei": "台北市", "NewTaipei": "新北市", "Taoyuan": "桃園市",
	"Taichung": "台中市", "Tainan": "台南市", "Kaohsiung": "高雄市",
	"Keelung": "基隆市", "Hsinchu": "新竹市", "HsinchuCounty": "新竹縣",
	"MiaoliCounty": "苗栗縣", "ChanghuaCounty": "彰化縣", "NantouCounty": "南投縣",
	"YunlinCounty": "雲林縣", "ChiayiCounty": "嘉義縣", "Chiayi": "嘉義市",
	"PingtungCounty": "屏東縣", "YilanCounty": "宜蘭縣", "HualienCounty": "花蓮縣",
	"TaitungCounty": "台東縣", "PenghuCounty": "澎湖縣",
	"KinmenCounty": "金門縣", "LienchiangCounty": "連江縣",
	"InterCity": "公路客運",
	"Miaoli":    "苗栗縣", "Changhua": "彰化縣", "Nantou": "南投縣",
	"Yunlin": "雲林縣", "Pingtung": "屏東縣", "Yilan": "宜蘭縣",
	"Hualien": "花蓮縣", "Taitung": "台東縣", "Penghu": "澎湖縣",
	"Kinmen": "金門縣", "Lienchiang": "連江縣",
}

var mrtSystemNames = map[string]string{
	"TRTC": "台北捷運", "KRTC": "高雄捷運", "KLRT": "桃園捷運",
	"TYMC": "台中捷運", "NTDLRT": "淡海輕軌", "KHLRT": "高雄輕軌",
}

func cityName(code string) string {
	if n, ok := cityNames[code]; ok {
		return n
	}
	return code
}

func mrtSystemName(code string) string {
	if n, ok := mrtSystemNames[code]; ok {
		return n
	}
	return code
}

const size = 1024

func changetovector(ctx context.Context, rc *redis.Client, db *pgxpool.Pool) {
	since, _ := rc.Get("LastTimeUpdate").Result()
	if since == "" {
		since = time.Time{}.Format(time.RFC3339)
	}
	tables := []string{"bus_subroutes", "bus_stations", "bike_stations", "mrt_station", "tra_stations", "thsr_stations", "tra_trains", "thsr_trains"}
	processBatch := func(table string, input []string, inrow []resp) bool {
		if len(input) == 0 {
			return true
		}
		body, comp, err, flipopen := callOllama(input)
		if err != nil || !comp {
			log.Printf("[vector] action=vector event=skip reason=api_error,error=%s", err)
			return false
		}
		var result struct {
			Embeddings [][]float32 `json:"embeddings"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			log.Printf("[vector] action=vector event=unmarshal_error error=%v", err)
			flipopen()
			return false
		}
		flipopen()
		resp := result.Embeddings
		b := &pgx.Batch{}
		c1 := `INSERT INTO search_vector(
					type, uid, name, city, depart, destin, geom, embedding, updated_at
				)
				VALUES ($1, $2, $3, $4, $5, $6,
					CASE WHEN $7 = '' THEN NULL ELSE ST_GeomFromText($7, 4326) END,
					$8::vector, NOW())
				ON CONFLICT (type, uid, city)
				DO UPDATE SET name = EXCLUDED.name,
					depart = EXCLUDED.depart,
					destin = EXCLUDED.destin,
					geom = EXCLUDED.geom,
					embedding = EXCLUDED.embedding,
					updated_at = NOW();`
		for i, v := range resp {
			d := inrow[i]
			b.Queue(c1, d.Type, d.UID, d.Name, d.City, d.DepartSystem, d.Destin, d.Geom, toVecLiteral(v))
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
					cn := cityName(city)
					text := fmt.Sprintf("類型：公車路線 子路線UID：%s 路線名：%s 縣市：%s 起點站：%s 終點站：%s", uid, name, cn, depart, destin)
					input = append(input, text)
					inrow = append(inrow, resp{Type: "bus_route", UID: uid, Name: name, City: cn, DepartSystem: depart, Destin: destin})
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
					cn := cityName(city)
					text := fmt.Sprintf("類型：公車站牌 站點UID：%s 站點名稱：%s 縣市：%s 位置：%s", uid, name, cn, geom)
					input = append(input, text)
					inrow = append(inrow, resp{Type: "bus_station", UID: uid, Name: name, City: cn, Geom: geom})
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
					cn := cityName(city)
					text := fmt.Sprintf("類型：公共自行車租借站 站點UID：%s 站點名稱：%s 縣市：%s 位置：%s", uid, name, cn, geom)
					input = append(input, text)
					inrow = append(inrow, resp{Type: "bike_station", UID: uid, Name: name, City: cn, Geom: geom})
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
					cn := mrtSystemName(system)
					text := fmt.Sprintf("類型：捷運站 車站UID：%s 車站名稱：%s 捷運系統：%s 位置：%s", uid, name, cn, geom)
					input = append(input, text)
					inrow = append(inrow, resp{Type: "mrt_station", UID: uid, Name: name, City: cn, Geom: geom})
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
					cn := cityName(city)
					text := fmt.Sprintf("類型：高鐵車站 車站UID：%s 車站名稱：%s 縣市：%s 位置：%s", uid, name, cn, geom)
					input = append(input, text)
					inrow = append(inrow, resp{Type: "thsr_station", UID: uid, Name: name, City: cn, Geom: geom})
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
					cn := cityName(city)
					text := fmt.Sprintf("類型：台鐵車站 車站UID：%s 車站名稱：%s 縣市：%s 位置：%s", uid, name, cn, geom)
					input = append(input, text)
					inrow = append(inrow, resp{Type: "tra_station", UID: uid, Name: name, City: cn, Geom: geom})
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
			case "tra_trains":
				rows, _ := db.Query(ctx, `SELECT DISTINCT ON (trainno) trainno, train_type_name, starting_station_name, ending_station_name FROM tra_timetable WHERE train_date >= CURRENT_DATE AND updated_at >= $1 ORDER BY trainno, train_date;`, since)
				defer rows.Close()
				input := make([]string, 0, size)
				inrow := make([]resp, 0, size)
				for rows.Next() {
					var trainno, typeName, depart, destin string
					if err := rows.Scan(&trainno, &typeName, &depart, &destin); err != nil {
						log.Printf("[vector] action=vector event=scan_error error=%v", err)
						continue
					}
					text := fmt.Sprintf("類型：台鐵車次 車次：%s 車種：%s 起站：%s 終站：%s", trainno, typeName, depart, destin)
					input = append(input, text)
					inrow = append(inrow, resp{Type: "tra_train", UID: trainno, Name: trainno, City: typeName, DepartSystem: depart, Destin: destin})
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
			case "thsr_trains":
				rows, _ := db.Query(ctx, `SELECT DISTINCT ON (trainno) trainno, starting_station_name, ending_station_name FROM thsr_timetable WHERE train_date >= CURRENT_DATE AND updated_at >= $1 ORDER BY trainno, train_date;`, since)
				defer rows.Close()
				input := make([]string, 0, size)
				inrow := make([]resp, 0, size)
				for rows.Next() {
					var trainno, depart, destin string
					if err := rows.Scan(&trainno, &depart, &destin); err != nil {
						log.Printf("[vector] action=vector event=scan_error error=%v", err)
						continue
					}
					text := fmt.Sprintf("類型：高鐵車次 車次：%s 起站：%s 終站：%s", trainno, depart, destin)
					input = append(input, text)
					inrow = append(inrow, resp{Type: "thsr_train", UID: trainno, Name: trainno, City: "高鐵", DepartSystem: depart, Destin: destin})
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

func toVecLiteral(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = strconv.FormatFloat(float64(f), 'f', -1, 32)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func callOllama(input []string) ([]byte, bool, error, func()) {
	client := resty.New().SetHeader("Content-Type", "application/json")
	resp, err := client.R().
		SetBody(map[string]interface{}{
			"model": "qwen3-embedding:0.6b",
			"input": input,
		}).
		Post("http://ollama:11434/api/embed")
	if err != nil {
		return nil, false, err, nil
	}
	return resp.Body(), true, nil, func() {
		if err := resp.RawResponse.Body.Close(); err != nil {
			log.Printf("[embed] action=fail-close-response error=%v", err)
		}
	}
}
