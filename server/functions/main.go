package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	r := gin.Default()
	c := resty.New()
	rc := connectredis()
	db := connectdb()
	c.SetBaseURL("https://tdx.transportdata.tw/api/basic").
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "br,gzip").
		SetDoNotParseResponse(true).
		SetAuthToken(getToken(rc, c)).
		SetRetryCount(5).
		SetRetryWaitTime(1 * time.Second).
		SetRetryMaxWaitTime(5 * time.Second).
		AddRetryCondition(
			func(r *resty.Response, err error) bool {
				if err != nil {
					return true
				}
				return r.StatusCode() == 429
			},
		)
	defer rc.Close()
	defer db.Close()
	r.POST("/daily", func(ctx *gin.Context) {
		log.Println("⚡ Azure Trigger: Daily static sync job started.")
		bus_static(c, rc, db)
		bike_static(c, rc, db)
		mrt_static(c, rc, db)
		rail_static(c, rc, db)
	})
	r.POST("/2min", func(ctx *gin.Context) {
		log.Println("⚡ Azure Trigger: 2min static sync job started.")
		tra_eta(c, rc)
	})
	r.POST("/1min", func(ctx *gin.Context) {
		bike_eta(c, rc, db)
	})
	r.POST("/10s", func(ctx *gin.Context) {
		Bus_eta(c, rc, db)
		mrt_eta(c, rc, db)
	})
}

// basic func
func call_api(client *resty.Client, rc *redis.Client, url string, name string) (json.Decoder, bool, error, func()) {
	since, _ := rc.Get("LastTimeGet_" + name).Result()
	resp, err := client.SetHeader("If-Modified-Since", since).R().Get(url)
	if err != nil {
		return json.Decoder{}, false, err, nil
	}
	if resp.StatusCode() == 304 {
		resp.RawResponse.Body.Close()
		return json.Decoder{}, false, nil, nil
	}
	rc.Set("LastTimeGet_"+name, time.Now(), 0)
	decorder := json.NewDecoder(resp.RawResponse.Body)
	return *decorder, true, nil, func() { resp.RawResponse.Body.Close() }
}
func savebushistory(db *pgxpool.Pool, eat []raw_Bus_Esimated, posit []raw_Bus_Position, city string) {
	mp := make(map[string]raw_Bus_Position)
	var rows [][]interface{}
	for _, b := range posit {
		mp[b.PlateNumb] = b
	}
	for _, b := range eat {
		stime, err := time.Parse(time.RFC3339, b.SrcUpdateTime)
		var gps interface{} = nil
		var geom interface{} = nil
		if err != nil {
			stime = time.Now()
		}
		if pos, ok := mp[b.PlateNumb]; ok {
			t, err := time.Parse(time.RFC3339, pos.GPSTime)
			if err != nil {
				t = stime
			}
			gps = t
			geom = fmt.Sprintf("POINT(%.6f %.6f)", pos.BusPosition.PositionLon, pos.BusPosition.PositionLat)
		}
		rows = append(rows, []interface{}{
			b.SubRouteUID,
			b.Direction,
			b.StopUID,
			b.EstimatedTime,
			b.StopStatus,
			b.PlateNumb,
			geom,
			gps,
			stime,
		})
	}
	if len(rows) <= 0 {
		return
	}
	b, _ := db.Begin(ctx)
	_, err := b.Exec(ctx, "CREATE TEMP TABLE temp_bus_history (uid text, dir int2, suid text, est int4, sts int2,pl text, geom text,gps timestamptz,stime timestamptz) ON COMMIT DROP;")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	_, err = b.CopyFrom(ctx, pgx.Identifier{"temp_bus_history"}, []string{"sub_route_uid", "direction", "stop_uid", "estimate_time", "stop_status", "plate_numb", "bus_position", "gps_time", "src_update_time"}, pgx.CopyFromRows(rows))
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	c1 := `INSERT INTO bus_eta_history (
				sub_route_uid,
		 		direction,
				stop_uid,
				estimate_time,
				stop_status,
				plate_numb,
				bus_position,
				gps_time,
			    src_update_time
			)
		SELECT
				uid,
		 		dir,
				suid,
				est,
				sts,
				pl,
				CASE WHEN geom IS NOT NULL THEN ST_GeomFromText(geom, 4326) END,
				gps,
			    stime
		FROM temp_bus_history;`
	_, err = b.Exec(ctx, c1)
	if err != nil {
		fmt.Println(err.Error())
	}
	b.Commit(ctx)
}
func busstaticmp(db *pgxpool.Pool, city string) ([]Bus_stationmap, error) {
	var query string
	var arg string
	if city == "interCity" {
		query = "SELECT station_id, station_name, sub_route_uid, route_name, direction, stop_uid, stop_sequence FROM bus_station_stop_map WHERE stop_uid LIKE $1"
		arg = "THB%"
	} else {
		query = "SELECT station_id, station_name, sub_route_uid, route_name, direction, stop_uid, stop_sequence FROM bus_station_stop_map WHERE stop_uid LIKE $1"
		arg = city + "%"
	}
	rows, err := db.Query(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Bus_stationmap
	for rows.Next() {
		var temp Bus_stationmap
		err := rows.Scan(&temp.StationUID, &temp.StationName, &temp.SubRouteUID, &temp.SubRouteName, &temp.Direction, &temp.StopUID, &temp.StopSequence)
		if err != nil {
			fmt.Println(err)
		}
		list = append(list, temp)
	}
	return list, nil
}
func makethatsame(city string, subRouteUID string, Direction uint8) (string, uint8) {
	if city == "InterCity" {
		temp := subRouteUID[len(subRouteUID)-2:]
		if temp == "01" {
			return subRouteUID[:len(subRouteUID)-2], 0
		} else if temp == "02" {
			return subRouteUID[:len(subRouteUID)-2], 1
		}
		return subRouteUID[:len(subRouteUID)-2], Direction
	}
	return subRouteUID, Direction
}
func connectredis() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	pong, err := client.Ping().Result()
	if err != nil {
		panic(err)
	}
	fmt.Println(pong, err)
	return client
}
func connectdb() *pgxpool.Pool {
	conn, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}
	err = conn.Ping(context.Background())
	if err != nil {
		panic(err)
	}
	return conn
}
func getToken(rc *redis.Client, c *resty.Client) string {
	val, err := rc.Get("TDX_Token").Result()
	if err != nil {
		fmt.Println("Token eq Empty")
		resp, err := c.R().
			SetHeader("content-type", "application/x-www-form-urlencoded").
			SetFormData(map[string]string{
				"grant_type":    "client_credentials",
				"client_id":     os.Getenv("TDX_CLIENT_ID"),
				"client_secret": os.Getenv("TDX_CLIENT_SECRET"),
			}).
			Post("https://tdx.transportdata.tw/auth/realms/TDXConnect/protocol/openid-connect/token")
		if err != nil {
			fmt.Println(err.Error())
		}
		var mp map[string]interface{}
		err = json.Unmarshal([]byte(resp.String()), &mp)
		val = mp["access_token"].(string)
		if err != nil {
			fmt.Println(err.Error())
		}
		err = rc.Set("TDX_Token", val, 6*time.Hour).Err()
		if err != nil {
			fmt.Println(err.Error())
		}
	}
	return val
}
func process_static(client *resty.Client, rc *redis.Client, db *pgxpool.Pool, city string, api string, processer func([]byte)) {
	var target string
	if city == "InterCity" {
		target = fmt.Sprintf("/v2/Bus/%s/InterCity", api)
	} else {
		target = fmt.Sprintf("/v2/Bus/%s/City/%s", api, city)
	}
	dec, comp, err, flipopen := call_api(client, rc, target, "bus_"+api+city)
	if err != nil || !comp {
		return
	}
	defer flipopen()
	if _, err := dec.Token(); err != nil {
		return
	}
	var raw_rows [][]interface{}
	for dec.More() {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			continue
		}
		processer(raw)
		switch api {
		case "Route":
			var r raw_Bus_Route
			err := json.Unmarshal(raw, &r)
			if err != nil {
				fmt.Println(err.Error())
			}
			for _, sub := range r.SubRoutes {
				dep, dest := sub.DepartureStopNameZh, sub.DestinationStopNameZh
				if dep == "" {
					dep = r.DepartureStopNameZh
				}
				if dest == "" {
					dest = r.DestinationStopNameZh
				}
				uid, _ := makethatsame(city, sub.SubRouteUID, sub.Direction)
				raw_rows = append(raw_rows, []interface{}{
					uid, r.RouteUID, r.RouteName.Zhtw, sub.SubRouteName.Zhtw, dep, dest, api, raw,
				})
			}
		case "StopOfRoute":
			var s raw_Stopofroute
			err := json.Unmarshal(raw, &s)
			if err != nil {
				fmt.Println(err.Error())
			}
			uid, _ := makethatsame(city, s.SubRouteUID, s.Direction)
			raw_rows = append(raw_rows, []interface{}{
				uid, s.RouteUID, "", "", "", city, api, raw,
			})
		case "Shape":
			var s raw_Bus_Shape
			err := json.Unmarshal(raw, &s)
			if err != nil {
				fmt.Println(err.Error())
			}
			uid, _ := makethatsame(city, s.SubRouteUID, s.Direction)
			raw_rows = append(raw_rows, []interface{}{
				uid, s.RouteUID, "", "", "", city, api, raw,
			})
		case "Schedule":
			var t raw_Bus_Schedule
			err := json.Unmarshal(raw, &t)
			if err != nil {
				fmt.Println(err.Error())
			}
			uid, dir := makethatsame(city, t.SubRouteUID, t.Direction)
			raw_rows = append(raw_rows, []interface{}{
				uid, t.RouteUID, "", "", fmt.Sprintf("%d", dir), city, api, raw,
			})
		case "Station":
			var t raw_Bus_Station
			err := json.Unmarshal(raw, &t)
			if err != nil {
				fmt.Println(err.Error())
			}
			raw_rows = append(raw_rows, []interface{}{
				t.StationUID, t.StationID, t.StationName.Zhtw, "", city, "", api, raw,
			})
		}
	}
	if len(raw_rows) > 0 {
		from, err := db.CopyFrom(ctx, pgx.Identifier{"raw_bus_route"}, []string{"sub_route_uid", "route_uid", "route_name", "sub_route_name", "depart", "destin", "type", "content"}, pgx.CopyFromRows(raw_rows))
		fmt.Println(from, err)
		if err != nil {
			fmt.Println(from, err)
		}
	}
}
func mask(mon, tues, wed, thur, fri, satur, sun, national bool) uint8 {
	var res uint8
	days := []bool{mon, tues, wed, thur, fri, satur, sun, national}
	for i, v := range days {
		if v {
			res |= 1 << i
		}
	}
	return res
}
