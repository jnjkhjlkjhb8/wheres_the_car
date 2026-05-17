package main

import (
	"backend/models"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"slices"
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
		StopUID  string `json:"StopUID"`
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
	StopStatus    uint8  `json:"StopStatus"`
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
	DutyStatus uint8   `json:"DutyStatus"`
	BusStatus  uint8   `json:"BusStatus"`
	GPSTime    string  `json:"GPSTime"`
}
type Bus_StationMapping struct {
	StationUID   string
	StationName  string
	SubRouteUID  string
	SubRouteName string
	Direction    uint8
	StopUID      string
	StopSequence uint8
}
type Bus_History struct {
	SubRouteUID   string
	Direction     uint8
	StopUID       string
	EstimatedTime int32
	StopStatus    uint8
	PlateNumb     string
	PositionLon   float64
	PositionLat   float64
	GPSTime       time.Time
	SrcUpdateTime time.Time
}

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
		fmt.Printf("%s done\n", city)
		time.Sleep(1 * time.Second)
	}
	log.Println("done all cities")
}
func changetodbformat(db *pgxpool.Pool, raw map[string]*models.Subroute) {
	var row [][]interface{}
	for _, sub := range raw {
		for dir, d := range sub.Directions {
			stops, err := json.Marshal(d.Stops)
			if err != nil {
				fmt.Println(err.Error())
			}
			row = append(row, []interface{}{
				sub.SubRouteUID,
				sub.RouteUID,
				dir,
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
					stop.StationID, stop.StopName, sub.SubRouteUID, sub.SubRouteName, dir, stop.StopUID, stop.StopSequence,
				})
			}
		}
	}
	if len(row) > 0 {
		db.Exec(ctx, `CREATE TEMP TABLE temp_pb (sub_route_uid text,route_uid text,pb bytea) ON COMMIT DROP`)
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
		db.Exec(ctx, `CREATE TEMP TABLE temp_map(sid text, sname text, sruid text, rname text, dir int, suid text, seq int) ON COMMIT DROP`)
		db.CopyFrom(ctx, pgx.Identifier{"temp_map"}, []string{"sid", "sname", "sruid", "rname", "dir", "suid", "seq"}, pgx.CopyFromRows(mp))
		db.Exec(ctx, `INSERT INTO bus_station_stop_map (
                                  station_id,
                                  station_name,
                                  sub_route_uid,
                                  route_name,
                                  direction,
                                  stop_uid,
                                  stop_sequence
							)
							SELECT sid, sname, sruid, rname, dir, suid, seq FROM temp_map
							ON CONFLICT (station_id, sub_route_uid, stop_uid, direction) DO UPDATE
							SET station_name = EXCLUDED.station_name, route_name = EXCLUDED.route_name, stop_sequence = EXCLUDED.stop_sequence;`)
	}
}

// eta
func Get_Bus_eta(client *resty.Client, rc *redis.Client, db *pgxpool.Pool, city string) {
	mp, err := busstaticmp(db, city)
	if err != nil || len(mp) <= 0 {
		return
	}
	var eat []raw_Bus_Esimated
	var url string
	if city == "InterCity" {
		url = "/v2/Bus/EstimatedTimeOfArrival/InterCity"
	} else {
		url = fmt.Sprintf("/v2/Bus/EstimatedTimeOfArrival/City/%s", city)
	}
	if resp, err := client.R().Get(url); err == nil && resp.StatusCode() == 200 {
		defer resp.RawBody().Close()
		_ = json.Unmarshal(resp.Body(), &eat)
	}
	var posit []raw_Bus_Position
	if city == "InterCity" {
		url = "/v2/Bus/RealTimeByFrequency/InterCity"
	} else {
		url = fmt.Sprintf("/v2/Bus/RealTimeByFrequency/City/%s", city)
	}
	if resp, err := client.R().Get(url); err == nil && resp.StatusCode() == 200 {
		defer resp.RawBody().Close()
		_ = json.Unmarshal(resp.Body(), &posit)
	}
	busmap := make(map[string][]*models.BusPosition)
	etamap := make(map[string]raw_Bus_Esimated)
	stations := make(map[string]*models.Bus_StationArrival)
	routes := make(map[string]*models.Bus_RouteArrival)
	for _, eat := range eat {
		etamap[eat.StopUID] = eat
	}
	for _, b := range posit {
		uid, _ := makethatsame(city, b.SubRouteUID, b.Direction)
		pb := &models.BusPosition{
			PlateNumb:   b.PlateNumb,
			PositionLon: b.BusPosition.PositionLon,
			PositionLat: b.BusPosition.PositionLat,
			Speed:       int32(b.Speed),
			Azimuth:     int32(b.Azimuth),
			DutyStatus:  int32(b.DutyStatus),
			BusStatus:   int32(b.BusStatus),
			GpsTime:     b.GPSTime,
		}
		busmap[uid] = append(busmap[uid], pb)
	}
	for _, b := range mp {
		eta, ok := etamap[b.SubRouteUID]
		var est, status uint8
		var time string
		if ok {
			est = uint8(eta.EstimatedTime)
			status = eta.StopStatus
			time = eta.SrcUpdateTime
		} else {
			status = 67
		}
		if _, ok = stations[b.SubRouteUID]; !ok {
			stations[b.StationName] = &models.Bus_StationArrival{
				StationName: b.StationName,
				StationUid:  make([]string, 0),
				Routes:      make([]*models.Bus_StopEstimate, 0),
			}
		}
		station := stations[b.StationName]
		if !slices.Contains(station.StationUid, b.StationUID) {
			station.StationUid = append(station.StationUid, b.StationUID)
		}
		station.Routes = append(station.Routes, &models.Bus_StopEstimate{
			StopUid:       b.StopUID,
			SubRouteUid:   b.SubRouteUID,
			RouteName:     b.SubRouteName,
			Direction:     int32(b.Direction),
			Estimate:      int32(est),
			StopStatus:    int32(status),
			SrcUpdateTime: time,
			Buses:         busmap[b.SubRouteUID],
		})
		if _, ok = routes[b.SubRouteUID]; !ok {
			routes[b.SubRouteUID] = &models.Bus_RouteArrival{
				SubRouteUid: b.SubRouteUID,
				Stops:       make([]*models.Bus_RouteEstimate, 0),
			}
		}
		routes[b.SubRouteUID].Stops = append(routes[b.SubRouteUID].Stops, &models.Bus_RouteEstimate{
			StopUid:       b.StopUID,
			Direction:     int32(b.Direction),
			Estimate:      int32(est),
			StopStatus:    int32(status),
			SrcUpdateTime: time,
			Buses:         busmap[b.SubRouteUID],
		})
	}
	pipe := rc.Pipeline()
	for name, pb := range stations {
		byte, _ := proto.Marshal(pb)
		pipe.Set(fmt.Sprintf("bus_eta_station:%s:%s", city, name), byte, 180*time.Second)
	}
	for uid, pb := range routes {
		byte, _ := proto.Marshal(pb)
		pipe.Set(fmt.Sprintf("bus_eta_route:%s:%s", city, uid), byte, 180*time.Second)
	}
	_, err = pipe.Exec()
	if err != nil {
		fmt.Println(err.Error())
	}
}

// basic func
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
	_, err := db.Exec(ctx, "CREATE TEMP TABLE temp_bus_history (uid text, dir int2, suid text, est int4, sts int2,pl text, geom text,gps timestamptz,stime timestamptz) ON COMMIT DROP;")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	_, err = db.CopyFrom(ctx, pgx.Identifier{"temp_bus_history"}, []string{"sub_route_uid", "direction", "stop_uid", "estimate_time", "stop_status", "plate_numb", "bus_position", "gps_time", "src_update_time"}, pgx.CopyFromRows(rows))
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
	_, err = db.Exec(ctx, c1)
	if err != nil {
		fmt.Println(err.Error())
	}
}
func busstaticmp(db *pgxpool.Pool, city string) ([]Bus_StationMapping, error) {
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
	var list []Bus_StationMapping
	for rows.Next() {
		var temp Bus_StationMapping
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
