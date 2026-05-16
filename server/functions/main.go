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
	"YilanCounty", "HualienCounty", "TaitungCounty", "PenghuCounty", "KinmenCounty", "LienchiangCounty",
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
		Direction             int32  `json:"Direction"`
		DepartureStopNameZh   string `json:"DepartureStopNameZh,omitempty"`
		DestinationStopNameZh string `json:"DestinationStopNameZh,omitempty"`
	} `json:"SubRoutes"`
}
type raw_Stopofroute struct {
	RouteUID    string `json:"RouteUID"`
	SubRouteUID string `json:"SubRouteUID"`
	Direction   int32  `json:"Direction"`
	Stops       []struct {
		StopName struct {
			Zhtw string `json:"Zh_tw"`
		} `json:"StopName"`
		StopSequence int32 `json:"StopSequence"`
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
	Direction   int32  `json:"Direction"`
	Geometry    string `json:"Geometry"`
	UpdateTime  string `json:"UpdateTime"`
}

type raw_Station struct {
	StationUID  string `json:"StationUID"`
	StationName struct {
		Zhtw string `json:"Zh_tw"`
	}
	StationPosition struct {
		PositionLon float64 `json:"PositionLon"`
		PositionLat float64 `json:"PositionLat"`
	} `json:"StationPosition"`
}

type raw_Timeable struct {
	StationUID  string `json:"StationUID"`
	RouteUID    string `json:"RouteUID"`
	Direction   int32  `json:"Direction"`
	SubRouteUID struct {
		Zhtw string `json:"Zh_tw"`
	} `json:"SubRouteUID"`
	Frequencys json.RawMessage `json:"Frequencys,omitempty"`
	Timetables json.RawMessage `json:"Timetables,omitempty"`
}

func main() {
	r := gin.Default()
	c := resty.New()
	rc := connectredis()
	db := connectdb()
	c.SetBaseURL("https://tdx.transportdata.tw/api/basic").
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding:", "br,gzip").
		SetAuthToken(getToken(rc, c))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	r.Run(":8080")
	defer rc.Close()
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
		err = rc.Set(val, resp.String(), 86390).Err()
		if err != nil {
			fmt.Println(err.Error())
		}
	}
	return val
}
func dailyRoute(rc *redis.Client, c *resty.Client, db *pgxpool.Pool) {
	fmt.Println("Daily Route")
	for _, city := range cities {
		subRoutemap := make(map[string]*models.Subroute)
		since, err := rc.Get("Last_RouteUpdate").Result()
		if err != nil {
			fmt.Println(err.Error())
		}
		process_static(c, city, db, "Route", since, func(raw []byte) {
			var r raw_Route
			err := json.Unmarshal(raw, &r)
			if err != nil {
				fmt.Println(err.Error())
			}
			for _, sub := range r.SubRoutes {
				if _, ok := subRoutemap[sub.SubRouteUID]; !ok {
					subRoutemap[sub.SubRouteUID] = &models.Subroute{
						RouteUID:     r.RouteUID,
						RouteName:    r.RouteName.Zhtw,
						SubRouteUID:  r.RouteUID,
						SubRouteName: r.RouteName.Zhtw,
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
				subRoutemap[sub.SubRouteUID].Directions[sub.Direction] = &models.Direction{
					DepartureStopName:   dep,
					DestinationStopName: dest,
				}
			}
		})
		rc.Set("Last_RouteUpdate", time.Now(), 0)
		since, err = rc.Get("Last_StopOfRouteUpdate").Result()
		if err != nil {
			fmt.Println(err.Error())
		}
		process_static(c, city, db, "StopOfRoute", since, func(raw []byte) {
			var r raw_Stopofroute
			err := json.Unmarshal(raw, &r)
			if err != nil {
				fmt.Println(err.Error())
			}
			if sr, ok := subRoutemap[r.SubRouteUID]; ok {
				if d, ok := sr.Directions[r.Direction]; ok {
					for _, stop := range r.Stops {
						d.Stops = append(d.Stops, &models.Stop{
							StopName:     stop.StopName.Zhtw,
							StopSequence: stop.StopSequence,
							PositionLat:  stop.StopPosition.PositionLat,
							PositionLon:  stop.StopPosition.PositionLon,
							StationID:    stop.StationID,
						})
					}
				}
			}
		})
		rc.Set("Last_StopOfRouteUpdate", time.Now(), 0)
		since, err = rc.Get("Last_ShapeUpdate").Result()
		if err != nil {
			fmt.Println(err.Error())
		}
		process_static(c, city, db, "Shape", since, func(raw []byte) {
			var r raw_Shape
			json.Unmarshal(raw, &r)
			if sr, ok := subRoutemap[r.SubRouteUID]; ok {
				if d, ok := sr.Directions[r.Direction]; ok {
					d.Geometry = r.Geometry
				}
			}
		})
		rc.Set("Last_ShapeUpdate", time.Now(), 0)
		since, err = rc.Get("Last_TimetableUpdate").Result()
		if err != nil {
			fmt.Println(err.Error())
		}
		process_static(c, city, db, "Timetable", since, func(raw []byte) {})
		rc.Set("Last_TimetableUpdate", time.Now(), 0)
		since, err = rc.Get("Last_StationUpdate").Result()
		if err != nil {
			fmt.Println(err.Error())
		}
		process_static(c, city, db, "Station", since, func(raw []byte) {})
		rc.Set("Last_StationUpdate", time.Now(), 0)
		changetodbformat(db, subRoutemap)
		savestations(db, city)
		savetimetable(db, city)
		savetostaticredis(rc, subRoutemap)
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
	c1 := `"
			CREATE TEMP TABLE temp_bus (
    		uid text,
    		rid text,
			d int,
			name1 text,
			name2 text,
			geom text,
    		rawstop jsonb
                          ) ON COMMIT DROP
		    "`
	c2 := `"
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
						   Select ROW (s ->> 'StationUID',s ->> 'StopName',(s ->> 'StopSequence')::int),
								  st_setsrid(st_makepoint((s ->> 'PositionLon')::float, (s ->> 'PositionLat')::float, 4326)
								  )::stop_type
						   FROM jsonb_array_element(rawstop) s
				   )
			FROM temp_bus ON CONFLICT (sub_route_uid,direction) DO UPDATE SET geometry = excluded.geometry,stops = excluded.stops;
			"`
	db.Exec(ctx, c1)
	db.CopyFrom(ctx, pgx.Identifier{"temp_bus"}, []string{"uid", "rid", "d", "name", "geom", "rawstop"}, pgx.CopyFromRows(row))
	db.Exec(ctx, c2)
}
func savestations(db *pgxpool.Pool, city string) {
	c1 := `"
			INSERT INTO bus_stations (
									  station_uid,
									  station_name,
									  city,
									  position
			)
			SELECT sub_route_uid,route_uid,route_name,
				   ST_SetSRID(ST_MakePoint((content->'StationPosition'->>'PositionLon')::float,(content->'StationPosition'->>'PositionLat')::float), 4326)
			FROM raw_bus_route WHERE type = 'Station'
			ON CONFLICT (station_uid) DO UPDATE SET station_name = EXCLUDED.station_name, position = EXCLUDED.position
			"`
	db.Exec(ctx, c1, city)
}
func savetimetable(db *pgxpool.Pool, city string) {
	c1 := `"
			INSERT INTO bus_timetables(
									   sub_route_uid,
									   route_id,
									   direction,
									   frequencys,
									   timetables
			)
			SELECT sub_route_uid,route_uid,content->'Frequencys',content->'Timetables'
			FROM raw_bus_route WHERE type = 'Timetables'
			ON CONFLICT (sub_route_uid,direction) DO  UPDATE  SET frequencys = excluded.frequencys,timetables = excluded.timetables;
			"'`
	db.Exec(ctx, c1, city)
}
func savetostaticredis(rc *redis.Client, subroutemap map[string]*models.Subroute) {
	temp := make([]*models.Subroute, 0, len(subroutemap))
	for _, sub := range subroutemap {
		temp = append(temp, sub)
	}
	temp = slices.DeleteFunc(temp, func(sub *models.Subroute) bool {
		return sub.SubRouteUID != sub.RouteUID
	})
	for _, sub := range temp {
		pb, err := proto.Marshal(sub)
		if err != nil {
			fmt.Println(err)
		}
		rc.Set(fmt.Sprintf("bus:subroute:%s", sub.SubRouteUID), pb, 0)
	}
}
func process_static(client *resty.Client, city string, db *pgxpool.Pool, api string, since string, processer func([]byte)) {
	resp, err := client.R().
		SetDoNotParseResponse(true).
		SetHeader("If-Modified-Since", since).
		SetPathParam("api", api).
		SetPathParam("city", city).
		Get("/v2/Bus/{api}/City/{city}")
	defer resp.RawResponse.Body.Close()
	dec := json.NewDecoder(resp.RawResponse.Body)
	var raw_rows [][]interface{}
	dec.Token()
	if err != nil {
		fmt.Println(err.Error())
	}
	for dec.More() {
		var raw json.RawMessage
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		processer(raw)
		switch api {
		case "Routes":
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
				raw_rows = append(raw_rows, []interface{}{
					sub.SubRouteUID,
					r.RouteUID,
					r.RouteName.Zhtw,
					sub.SubRouteName.Zhtw,
					dep,
					dest,
					api,
					raw,
				})
			}
		case "StopOfRoute":
			var s raw_Stopofroute
			err := json.Unmarshal(raw, &s)
			if err != nil {
				fmt.Println(err.Error())
			}
			raw_rows = append(raw_rows, []interface{}{
				s.SubRouteUID, s.RouteUID, "", "", "", api, raw,
			})
		case "Shapes":
			var s raw_Shape
			err := json.Unmarshal(raw, &s)
			if err != nil {
				fmt.Println(err.Error())
			}
			raw_rows = append(raw_rows, []interface{}{
				s.SubRouteUID, s.RouteUID, "", "", "", api, raw,
			})
		case "Schedule":
			var t raw_Stopofroute
			err := json.Unmarshal(raw, &t)
			if err != nil {
				fmt.Println(err.Error())
			}
			raw_rows = append(raw_rows, []interface{}{
				t.SubRouteUID, t.RouteUID, "", "", "", api, raw,
			})
		}
		if len(raw_rows) > 0 {
			from, err := db.CopyFrom(ctx, pgx.Identifier{"raw_bus_route"}, []string{"sub_route_uid", "route_uid", "route_name", "sub_route_name", "depart", "destin", "type", "content"}, pgx.CopyFromRows(raw_rows))
			fmt.Println(from, err)
			if err != nil {
				fmt.Println(from, err)
			}
		}
	}
}
