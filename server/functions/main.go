package main

import (
	"backend/models"
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
	"google.golang.org/protobuf/proto"
)

var ctx = context.Background()
var cities = []string{
	"Taipei", "NewTaipei", "Taoyuan", "Taichung", "Tainan", "Kaohsiung",
	"Keelung", "Hsinchu", "HsinchuCounty", "MiaoliCounty", "ChanghuaCounty",
	"NantouCounty", "YunlinCounty", "ChiayiCounty", "Chiayi", "PingtungCounty",
	"YilanCounty", "HualienCounty", "TaitungCounty", "PenghuCounty", "KinmenCounty", "LienchiangCounty", "InterCity",
}

type raw_Route struct {
	RouteUID  string `json:"RouteUID"`
	RouteName struct {
		Zhtw string `json:"Zh_tw"`
	} `json:"RouteName"`
	DepartureStopNameZh   string `json:"DepartureStopNameZh"`
	DestinationStopNameZh string `json:"DestinationStopNameZh"`
	SubRoutes             []struct {
		SubRouteUID  string `json:"SubRouteUID"`
		SubRouteName struct {
			Zhtw string `json:"Zh_tw"`
		} `json:"SubRouteName"`
		Direction             uint8  `json:"Direction"`
		DepartureStopNameZh   string `json:"DepartureStopNameZh,omitempty"`
		DestinationStopNameZh string `json:"DestinationStopNameZh,omitempty"`
	} `json:"SubRoutes"`
}
type raw_Stopofroute struct {
	RouteUID    string `json:"RouteUID"`
	SubRouteUID string `json:"SubRouteUID"`
	Direction   uint8  `json:"Direction"`
	Stops       []struct {
		StopName struct {
			Zhtw string `json:"Zh_tw"`
		} `json:"StopName"`
		StopSequence uint8 `json:"StopSequence"`
		StopPosition struct {
			PositionLon float64 `json:"PositionLon"`
			PositionLat float64 `json:"PositionLat"`
		} `json:"StopPosition"`
		StationID        string `json:"StationID"`
		LocationCityCode string `json:"LocationCityCode"`
	} `json:"Stops"`
}

type raw_Shape struct {
	SubRouteUID string `json:"SubRouteUID,omitempty"`
	RouteUID    string `json:"RouteUID"`
	Direction   uint8  `json:"Direction"`
	Geometry    string `json:"Geometry"`
	UpdateTime  string `json:"UpdateTime"`
}

type raw_Station struct {
	StationUID  string `json:"StationUID"`
	StationID   string `json:"StationID"`
	StationName struct {
		Zhtw string `json:"Zh_tw"`
	} `json:"StationName"`
	StationPosition struct {
		PositionLon float64 `json:"PositionLon"`
		PositionLat float64 `json:"PositionLat"`
	} `json:"StationPosition"`
}

type raw_Schedule struct {
	SubRouteUID string          `json:"SubRouteUID"`
	RouteUID    string          `json:"RouteUID"`
	Direction   uint8           `json:"Direction"`
	Frequencys  json.RawMessage `json:"Frequencys,omitempty"`
	Timetables  json.RawMessage `json:"Timetables,omitempty"`
}

type raw_Bus_Esimated struct {
	PlateNumb     string `json:"PlateNumb"`
	StopUID       string `json:"StopUID"`
	SubRouteUID   string `json:"SubRouteUID"`
	Direction     uint8  `json:"Direction"`
	EstimatedTime int32  `json:"EstimatedTime"`
	StopStatus    int32  `json:"StopStatus"`
	SrcUpdateTime string `json:"SrcUpdateTime"`
}
type raw_Bus_Position struct {
	PlateNumb   string `json:"PlateNumb"`
	SubRouteUID string `json:"SubRouteUID"`
	StopUID     string `json:"StopUID"`
	Direction   uint8  `json:"Direction"`
	BusPosition struct {
		PositionLon float64 `json:"PositionLon"`
		PositionLat float64 `json:"PositionLat"`
	} `json:"BusPosition"`
	Azimuth    int     `json:"Azimuth"`
	Speed      float64 `json:"Speed"`
	DutyStatus int32   `json:"DutyStatus"`
	BusStatus  int32   `json:"BusStatus"`
}

func main() {
	r := gin.Default()
	c := resty.New()
	rc := connectredis()
	db := connectdb()
	c.SetBaseURL("https://tdx.transportdata.tw/api/basic").
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "br,gzip").
		SetAuthToken(getToken(rc, c))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	r.Run(":8080")
	defer rc.Close()
}

// static

func dailyRoute(rc *redis.Client, c *resty.Client, db *pgxpool.Pool) {
	fmt.Println("Daily Route")
	for _, city := range cities {
		subRoutemap := make(map[string]*models.Subroute)
		_, err := db.Exec(ctx, "DELETE FROM raw_bus_route WHERE destin = $1 OR depart = $1", city)
		if err != nil {
			fmt.Println(err.Error())
		}
		since, _ := rc.Get("Last_RouteUpdate_" + city).Result()
		process_static(c, city, db, "Route", since, func(raw []byte) {
			var r raw_Route
			err := json.Unmarshal(raw, &r)
			if err != nil {
				fmt.Println(err.Error())
			}
			for _, sub := range r.SubRoutes {
				uid, dir := makethatsame(city, sub.SubRouteUID, sub.Direction)
				if _, ok := subRoutemap[uid]; !ok {
					subRoutemap[uid] = &models.Subroute{
						RouteUID:     r.RouteUID,
						RouteName:    r.RouteName.Zhtw,
						SubRouteUID:  uid,
						SubRouteName: sub.SubRouteName.Zhtw,
						Directions:   make(map[int32]*models.Direction),
					}
				}
				dep, dest := sub.DepartureStopNameZh, sub.DestinationStopNameZh
				if dep == "" {
					dep = r.DepartureStopNameZh
				}
				if dest == "" {
					dest = r.DestinationStopNameZh
				}
				subRoutemap[uid].Directions[int32(dir)] = &models.Direction{
					DepartureStopName:   dep,
					DestinationStopName: dest,
				}
			}
		})
		rc.Set("Last_RouteUpdate_"+city, time.Now(), 0)
		since, _ = rc.Get("Last_StopOfRouteUpdate_" + city).Result()
		process_static(c, city, db, "StopOfRoute", since, func(raw []byte) {
			var r raw_Stopofroute
			err := json.Unmarshal(raw, &r)
			if err != nil {
				fmt.Println(err.Error())
			}
			uid, dir := makethatsame(city, r.SubRouteUID, r.Direction)
			if sr, ok := subRoutemap[uid]; ok {
				if d, ok := sr.Directions[int32(dir)]; ok {
					for _, stop := range r.Stops {
						d.Stops = append(d.Stops, &models.Stop{
							StopName:     stop.StopName.Zhtw,
							StopSequence: int32(stop.StopSequence),
							PositionLat:  stop.StopPosition.PositionLat,
							PositionLon:  stop.StopPosition.PositionLon,
							StationID:    stop.StationID,
						})
					}
				}
			}
		})
		rc.Set("Last_StopOfRouteUpdate_"+city, time.Now(), 0)
		since, _ = rc.Get("Last_ShapeUpdate_" + city).Result()
		process_static(c, city, db, "Shape", since, func(raw []byte) {
			var r raw_Shape
			json.Unmarshal(raw, &r)
			uid, dir := makethatsame(city, r.SubRouteUID, r.Direction)
			if sr, ok := subRoutemap[uid]; ok {
				if d, ok := sr.Directions[int32(dir)]; ok {
					d.Geometry = r.Geometry
				}
			}
		})
		rc.Set("Last_ShapeUpdate_"+city, time.Now(), 0)
		since, _ = rc.Get("Last_ScheduleUpdate_" + city).Result()
		process_static(c, city, db, "Schedule", since, func(raw []byte) {})
		rc.Set("Last_ScheduleUpdate_"+city, time.Now(), 0)
		since, _ = rc.Get("Last_StationUpdate_" + city).Result()
		process_static(c, city, db, "Station", since, func(raw []byte) {})
		rc.Set("Last_StationUpdate_"+city, time.Now(), 0)
		changetodbformat(db, subRoutemap)
		savestations(db, city)
		saveschedule(db, city)
		savestatictodb(db, subRoutemap)
		fmt.Println("%s done", city)
		time.Sleep(1 * time.Second)
	}
	log.Println("done all cities")
}
func changetodbformat(db *pgxpool.Pool, raw map[string]*models.Subroute) {
	var row [][]interface{}
	for _, sub := range raw {
		for _, d := range sub.Directions {
			stops, err := json.Marshal(d.Stops)
			if err != nil {
				fmt.Println(err.Error())
			}
			row = append(row, []interface{}{
				sub.SubRouteUID,
				sub.RouteUID,
				d,
				sub.RouteName,
				sub.SubRouteName,
				d.Geometry,
				stops,
			})
		}
	}
	c1 := `
			CREATE TEMP TABLE temp_bus (
    		uid text,
    		rid text,
			d int,
			name1 text,
			name2 text,
			geom text,
    		rawstop jsonb
                          ) ON COMMIT DROP
		    `
	c2 := `
			INSERT INTO bus_subroutes(
                          sub_route_uid,
                          route_uid,
                          direction,
                          route_name,
                          sub_route_name,
                          geometry,
                          stops
			)
			SELECT uid,rid,d,name1,name2,st_geomfromtext(geom,4326),
				   Array(
						   Select ROW (s ->> 'StationUID',s ->> 'StopName',(s ->> 'StopSequence')::int),st_setsrid(st_makepoint((s ->> 'PositionLon')::float, (s ->> 'PositionLat')::float, 4326))::stop_type
						   FROM jsonb_array_elements(rawstop) AS s
				   )
			FROM temp_bus ON CONFLICT (sub_route_uid,direction) DO UPDATE SET geometry = excluded.geometry,stops = excluded.stops;
			`
	db.Exec(ctx, c1)
	db.CopyFrom(ctx, pgx.Identifier{"temp_bus"}, []string{"uid", "rid", "d", "name1", "name2", "geom", "rawstop"}, pgx.CopyFromRows(row))
	db.Exec(ctx, c2)
}
func savestations(db *pgxpool.Pool, city string) {
	c1 := fmt.Sprintf(`
			INSERT INTO bus_stations (
									  station_uid,
									  station_name,
									  city,
									  position
			)
			SELECT sub_route_uid,route_name,depart,
				   ST_SetSRID(ST_MakePoint((content->'StationPosition'->>'PositionLon')::float,(content->'StationPosition'->>'PositionLat')::float), 4326)
			FROM raw_bus_route WHERE type = 'Station' AND depart = '%s'
			ON CONFLICT (station_uid) DO UPDATE SET station_name = EXCLUDED.station_name, position = EXCLUDED.position
			`, city)
	db.Exec(ctx, c1)
}
func saveschedule(db *pgxpool.Pool, city string) {
	c1 := fmt.Sprintf(`
			INSERT INTO bus_schedule(
									   sub_route_uid,
									   route_id,
									   direction,
									   frequencys,
									   timetables
			)
			SELECT sub_route_uid,route_uid,(depart)::int,content->'Frequencys',content->'Timetables'
			FROM raw_bus_route WHERE type = 'Schedule' and destin = '%s'
			ON CONFLICT (sub_route_uid,direction) DO  UPDATE  SET frequencys = excluded.frequencys,timetables = excluded.timetables;
			`, city)
	db.Exec(ctx, c1)
}
func savestatictodb(db *pgxpool.Pool, raw map[string]*models.Subroute) {
	var row [][]interface{}
	var mp [][]interface{}
	for _, sub := range raw {
		pb, err := proto.Marshal(sub)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		row = append(row, []interface{}{
			sub.RouteUID, sub.RouteUID, pb,
		})
		for dir, d := range sub.Directions {
			for _, stop := range d.Stops {
				mp = append(mp, []interface{}{
					stop.StationID, sub.SubRouteUID, dir, sub.SubRouteName, stop.StopSequence,
				})
			}
		}
	}
	if len(row) > 0 {
		db.Exec(ctx, `"CREATE TEMP TABLE temp_pb (sub_route_uid text,route_uid text,pb bytea) ON COMMIT DROP"`)
		db.CopyFrom(ctx, pgx.Identifier{"temp_pb"}, []string{"uid", "rid", "pb"}, pgx.CopyFromRows(row))
		db.Exec(ctx, `INSERT INTO bus_static (
                        sub_route_uid,
                        route_uid,
                        pb
						)
						SELECT uid,rid,pb FROM temp_pb
						ON CONFLICT (sub_route_uid) DO UPDATE SET pb = excluded.pb,updated_at = NOW()
						`)
	}
	if len(mp) > 0 {
		db.Exec(ctx, `"CREATE TEMP TABLE temp_map(sid text,sruid text,dir int,rname text,suit text,seq int) ON COMMIT DROP"`)
		db.CopyFrom(ctx, pgx.Identifier{"temp_map"}, []string{"sid", "sruid", "dir", "rname", "seq"}, pgx.CopyFromRows(mp))
		db.Exec(ctx, `INSERT INTO bus_station_stop_map (
                                  station_id,
                                  sub_route_uid,
                                  direction,
                                  route_name,
                                  stop_uid,
                                  stop_sequence
							)
							SELECT sid, sruid, dir, rname, suid, seq FROM temp_map
							ON CONFLICT (station_id, sub_route_uid, stop_uid) DO UPDATE
							SET route_name = EXCLUDED.route_name, stop_sequence = EXCLUDED.stop_sequence;`)
	}
}

// eta
func Get_Bus_eta(client *resty.Client, rc *redis.Client, db *pgxpool.Pool, city string) {
	since, _ := rc.Get("Last_EtaUpdate_" + city).Result()
	url := fmt.Sprintf("/v2/Bus/EstimatedTimeOfArrival/City/%s", city)
	if resp, err := client.R().SetDoNotParseResponse(true).SetHeader("If-Modified-Since", since).Get(url); err != nil && resp.StatusCode() != 200 {
		defer resp.RawBody().Close()
		var etas []raw_Bus_Esimated
		decoder := json.NewDecoder(resp.RawBody())
		if err := decoder.Decode(&etas); err != nil {
			redisPipe := rc.Pipeline()
			var history [][]interface{}
			for _, eta := range etas {
				uid, dir := makethatsame(city, eta.SubRouteUID, eta.Direction)
				redisKey := fmt.Sprintf("bus:eta:%s", uid)
				bytes, _ := json.Marshal(eta)
				redisPipe.HSet(redisKey, eta.StopUID, string(bytes))
				redisPipe.Expire(redisKey, 120*time.Second)
				t, parseErr := time.Parse(time.RFC3339, eta.SrcUpdateTime)
				if parseErr == nil {
					history = append(history, []interface{}{
						uid, eta.StopUID, dir, eta.PlateNumb, eta.EstimatedTime, eta.StopStatus, t,
					})
				}

			}
		}
	}
}

// basic func

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
	defer conn.Close()
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
		val = mp["access_token"].(string)
		err = json.Unmarshal([]byte(resp.String()), &mp)
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
func process_static(client *resty.Client, city string, db *pgxpool.Pool, api string, since string, processer func([]byte)) {
	var target string
	if city == "InterCity" {
		target = fmt.Sprintf("/v2/Bus/%s/InterCity", api)
	} else {
		target = fmt.Sprintf("/v2/Bus/%s/City/%s", api, city)
	}
	req := client.R().
		SetDoNotParseResponse(true).
		SetHeader("If-Modified-Since", since)
	resp, err := req.Get(target)
	defer resp.RawResponse.Body.Close()
	if resp.StatusCode() == 304 {
		return
	}
	if resp.StatusCode() == 429 {
		time.Sleep(1 * time.Second)
	}
	dec := json.NewDecoder(resp.RawResponse.Body)
	var raw_rows [][]interface{}
	_, err = dec.Token()
	if err != nil {
		fmt.Println(err.Error())
	}
	if err != nil {
		fmt.Println(err.Error())
	}
	for dec.More() {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			continue
		}
		processer(raw)
		switch api {
		case "Route":
			var r raw_Route
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
			var s raw_Shape
			err := json.Unmarshal(raw, &s)
			if err != nil {
				fmt.Println(err.Error())
			}
			uid, _ := makethatsame(city, s.SubRouteUID, s.Direction)
			raw_rows = append(raw_rows, []interface{}{
				uid, s.RouteUID, "", "", "", city, api, raw,
			})
		case "Schedule":
			var t raw_Schedule
			err := json.Unmarshal(raw, &t)
			if err != nil {
				fmt.Println(err.Error())
			}
			uid, dir := makethatsame(city, t.SubRouteUID, t.Direction)
			raw_rows = append(raw_rows, []interface{}{
				uid, t.RouteUID, "", "", fmt.Sprintf("%d", dir), city, api, raw,
			})
		case "Station":
			var t raw_Station
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
