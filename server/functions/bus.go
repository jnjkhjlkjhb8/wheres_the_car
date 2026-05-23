package main

import (
	"github.com/jnjkhjlkjhb8/bus/models"

	"context"
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/go-redis/redis"
	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"
)

var cities = []string{
	"Taipei", "NewTaipei", "Taoyuan", "Taichung", "Tainan", "Kaohsiung",
	"Keelung", "Hsinchu", "HsinchuCounty", "MiaoliCounty", "ChanghuaCounty",
	"NantouCounty", "YunlinCounty", "ChiayiCounty", "Chiayi", "PingtungCounty",
	"YilanCounty", "HualienCounty", "TaitungCounty", "PenghuCounty", "KinmenCounty", "LienchiangCounty", "InterCity",
}

type raw_Bus_Route struct {
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

type raw_Bus_Schedule struct {
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
	NextBusTime   int32  `json:"NextBusTime"`
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
type Bus_stationmap struct {
	StationUID   string
	StationName  string
	SubRouteUID  string
	SubRouteName string
	Direction    uint8
	StopUID      string
	StopSequence uint8
}

/*
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
*/
type raw_Bus_Shape struct {
	SubRouteUID string `json:"SubRouteUID,omitempty"`
	RouteUID    string `json:"RouteUID"`
	Direction   uint8  `json:"Direction"`
	Geometry    string `json:"Geometry"`
	UpdateTime  string `json:"UpdateTime"`
}

type raw_Bus_Station struct {
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

type bus_schedule_freq struct {
	StartTime      string `json:"StartTime"`
	EndTime        string `json:"EndTime"`
	MinHeadwayMins int    `json:"MinHeadwayMins"`
	MaxHeadwayMins int    `json:"MaxHeadwayMins"`
	ServiceDay     struct {
		Sunday    bool `json:"Sunday"`
		Monday    bool `json:"Monday"`
		Tuesday   bool `json:"Tuesday"`
		Wednesday bool `json:"Wednesday"`
		Thursday  bool `json:"Thursday"`
		Friday    bool `json:"Friday"`
		Saturday  bool `json:"Saturday"`
	} `json:"ServiceDay"`
}

type bus_schedule_tb struct {
	ServiceDay struct {
		Sunday    bool `json:"Sunday"`
		Monday    bool `json:"Monday"`
		Tuesday   bool `json:"Tuesday"`
		Wednesday bool `json:"Wednesday"`
		Thursday  bool `json:"Thursday"`
		Friday    bool `json:"Friday"`
		Saturday  bool `json:"Saturday"`
	} `json:"ServiceDay"`
	StopTimes []struct {
		StopSequence  int    `json:"StopSequence"`
		DepartureTime string `json:"DepartureTime"`
	} `json:"StopTimes"`
}

func bus_static(ctx context.Context, client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	dailyRoute(ctx, rc, client, db)
}
func dailyRoute(ctx context.Context, rc *redis.Client, c *resty.Client, db *pgxpool.Pool) {
	log.Printf("[BUS] action=dailyRoute event=start")
	for _, city := range cities {
		if city == "LienchiangCounty" {
			continue
		}
		log.Printf("[BUS] action=dailyRoute city=%s event=city_start", city)
		subRoutemap := make(map[string]*models.Subroute)
		routeMap := make(map[string][]string)
		_, err := db.Exec(ctx, "DELETE FROM raw_bus_route WHERE destin = $1 OR depart = $1", city)
		if err != nil {
			log.Printf("[BUS] action=dailyRoute city=%s event=cleanup_error error=%v", city, err)
		}
		process_static(ctx, c, rc, db, city, "Route", func(raw []byte) {
			var r raw_Bus_Route
			err := json.Unmarshal(raw, &r)
			if err != nil {
				log.Printf("[BUS] action=dailyRoute city=%s api=Route event=unmarshal_error error=%v", city, err)
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
				routeMap[r.RouteUID] = append(routeMap[r.RouteUID], uid)
			}
		})
		process_static(ctx, c, rc, db, city, "StopOfRoute", func(raw []byte) {
			var r raw_Stopofroute
			err := json.Unmarshal(raw, &r)
			if err != nil {
				log.Printf("[BUS] action=dailyRoute city=%s api=StopOfRoute event=unmarshal_error error=%v", city, err)
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
							StopUID:      stop.StopUID,
						})
					}
				}
			}
		})
		process_static(ctx, c, rc, db, city, "Shape", func(raw []byte) {
			var r raw_Bus_Shape
			err := json.Unmarshal(raw, &r)
			if err != nil {
				log.Println("[BUS] action=dailyRoute event=marshal_error error=%v", err)
			}
			if r.SubRouteUID != "" {
				uid, dir := makethatsame(city, r.SubRouteUID, r.Direction)
				if sr, ok := subRoutemap[uid]; ok {
					if d, ok := sr.Directions[int32(dir)]; ok {
						d.Geometry = r.Geometry
					}
				}
			} else if subUIDs, exists := routeMap[r.RouteUID]; exists {
				for _, uid := range subUIDs {
					if sr, ok := subRoutemap[uid]; ok {
						if d, ok := sr.Directions[int32(r.Direction)]; ok {
							d.Geometry = r.Geometry
						}
					}
				}
			}
		})
		//process_static(c, rc, db, city, "Schedule", func(raw []byte) {})
		process_static(ctx, c, rc, db, city, "Station", func(raw []byte) {})
		changetodbformat(ctx, db, &subRoutemap)
		savestations(ctx, db, city)
		saveschedule(db, city)
		savestatictodb(ctx, db, &subRoutemap)
		_, err = db.Exec(ctx, "truncate raw_bus_route;")
		if err != nil {
			log.Printf("[BUS] action=dailyRoute city=%s event=cleanup_raw_error error=%v", city, err)
		}
		log.Printf("[BUS] action=dailyRoute city=%s event=city_complete subroute_count=%d", city, len(subRoutemap))
	}
	log.Printf("[BUS] action=dailyRoute event=complete")
}
func changetodbformat(ctx context.Context, db *pgxpool.Pool, raw *map[string]*models.Subroute) {
	var row [][]interface{}
	for _, sub := range *raw {
		for dir, d := range sub.Directions {
			stops, err := json.Marshal(d.Stops)
			if err != nil {
				log.Printf("[BUS] action=changetodbformat event=marshal_error error=%v", err)
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
	b, err := db.Begin(ctx)
	if err != nil {
		log.Printf("[BUS] action=changetodbformat event=begin_error error=%v", err)
		return
	}
	defer func() {
		_ = b.Rollback(ctx)
	}()
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
			SELECT uid, rid, d, name1, name2,geom,
				   ARRAY(
					   SELECT ROW(
								  s ->> 'StationUID',
								  s ->> 'StopName',
								  (s ->> 'StopSequence')::int,
								  (s ->> 'PositionLon')::float,
								  (s ->> 'PositionLat')::float
							  )::stop
					   FROM jsonb_array_elements(rawstop) AS s
				   )
			FROM temp_bus 
			ON CONFLICT (sub_route_uid, direction) 
			DO UPDATE SET geometry = EXCLUDED.geometry,stops = EXCLUDED.stops;
			`
	if _, err := b.Exec(ctx, c1); err != nil {
		log.Printf("[BUS] action=changetodbformat event=create_temp_error error=%v", err)
	}
	if _, err := b.CopyFrom(ctx, pgx.Identifier{"temp_bus"}, []string{"uid", "rid", "d", "name1", "name2", "geom", "rawstop"}, pgx.CopyFromRows(row)); err != nil {
		log.Printf("[BUS] action=changetodbformat event=copyfrom_error error=%v row_count=%d", err, len(row))
	}
	if _, err := b.Exec(ctx, c2); err != nil {
		log.Printf("[BUS] action=changetodbformat event=insert_error error=%v", err)
	}
	if err := b.Commit(ctx); err != nil {
		log.Printf("[BUS] action=changetodbformat event=commit_error error=%v", err)
	}
}
func savestations(ctx context.Context, db *pgxpool.Pool, city string) {
	c1 := `
			INSERT INTO bus_stations (
									  station_uid,
									  station_name,
									  city,
									  position
			)
			SELECT sub_route_uid,route_name,depart,
				   ST_SetSRID(ST_MakePoint((content->'StationPosition'->>'PositionLon')::float,(content->'StationPosition'->>'PositionLat')::float), 4326)
			FROM raw_bus_route WHERE type = 'Station' AND depart = $1
			ON CONFLICT (station_uid) DO UPDATE SET station_name = EXCLUDED.station_name, position = EXCLUDED.position
			`
	if _, err := db.Exec(ctx, c1, city); err != nil {
		log.Printf("[BUS] action=savestations city=%s event=insert_error error=%v", city, err)
	} else {
		log.Printf("[BUS] action=savestations city=%s event=complete", city)
	}
}
func saveschedule(db *pgxpool.Pool, city string) {

}

func savestatictodb(ctx context.Context, db *pgxpool.Pool, raw *map[string]*models.Subroute) {
	var row [][]interface{}
	var mp [][]interface{}
	for _, sub := range *raw {
		pb, err := proto.Marshal(sub)
		if err != nil {
			log.Printf("[BUS] action=savestatictodb event=marshal_error subroute=%s error=%v", sub.SubRouteUID, err)
			continue
		}
		row = append(row, []interface{}{
			sub.SubRouteUID, sub.RouteUID, pb,
		})
		for dir, d := range sub.Directions {
			for _, stop := range d.Stops {
				var temp string
				if len(sub.SubRouteUID) >= 3 {
					temp = sub.SubRouteUID[:3]
				}
				mp = append(mp, []interface{}{
					temp + stop.StationID, stop.StopName, sub.SubRouteUID, sub.SubRouteName, dir, stop.StopUID, stop.StopSequence,
				})
			}
		}
	}
	b, err := db.Begin(ctx)
	defer b.Rollback(ctx)
	if err != nil {
		log.Printf("[BUS] action=savestatictodb event=begin_error error=%v", err)
		return
	}
	if len(row) > 0 {
		if _, err := b.Exec(ctx, `CREATE TEMP TABLE temp_pb (uid text,rid text,pb bytea) ON COMMIT DROP`); err != nil {
			log.Printf("[BUS] action=savestatictodb event=create_temp_pb_error error=%v", err)
		}
		if _, err := b.CopyFrom(ctx, pgx.Identifier{"temp_pb"}, []string{"uid", "rid", "pb"}, pgx.CopyFromRows(row)); err != nil {
			log.Printf("[BUS] action=savestatictodb event=copyfrom_pb_error error=%v row_count=%d", err, len(row))
		}
		if _, err := b.Exec(ctx, `INSERT INTO bus_static (
                        sub_route_uid,
                        route_uid,
                        pb
					)
					SELECT uid,rid,pb FROM temp_pb
					ON CONFLICT (sub_route_uid) DO UPDATE SET pb = excluded.pb,updated_at = NOW()
					`); err != nil {
			log.Printf("[BUS] action=savestatictodb event=insert_pb_error error=%v", err)
		} else {
			log.Printf("[BUS] action=savestatictodb event=insert_pb_success row_count=%d", len(row))
		}
	}
	if len(mp) > 0 {
		if _, err := b.Exec(ctx, `CREATE TEMP TABLE temp_map(sid text, sname text, sruid text, rname text, dir int, suid text, seq int) ON COMMIT DROP`); err != nil {
			log.Printf("[BUS] action=savestatictodb event=create_temp_map_error error=%v", err)
		}
		if _, err := b.CopyFrom(ctx, pgx.Identifier{"temp_map"}, []string{"sid", "sname", "sruid", "rname", "dir", "suid", "seq"}, pgx.CopyFromRows(mp)); err != nil {
			log.Printf("[BUS] action=savestatictodb event=copyfrom_map_error error=%v row_count=%d", err, len(mp))
		}
		if _, err := b.Exec(ctx, `INSERT INTO bus_station_stop_map (
                                  station_id,
                                  station_name,
                                  sub_route_uid,
                                  route_name,
                                  direction,
                                  stop_uid,
                                  stop_sequence
									)
									SELECT DISTINCT ON (sruid, suid, dir) sid, sname, sruid, rname, dir, suid, seq FROM temp_map
									ON CONFLICT (sub_route_uid, stop_uid, direction) DO UPDATE
									SET station_name = EXCLUDED.station_name, route_name = EXCLUDED.route_name, stop_sequence = EXCLUDED.stop_sequence;`); err != nil {
			log.Printf("[BUS] action=savestatictodb event=insert_map_error error=%v", err)
		} else {
			log.Printf("[BUS] action=savestatictodb event=insert_map_success row_count=%d", len(mp))
		}
	}
	if err := b.Commit(ctx); err != nil {
		log.Printf("[BUS] action=savestatictodb event=commit_error error=%v", err)
	} else {
		log.Printf("[BUS] action=savestatictodb event=complete pb_rows=%d map_rows=%d", len(row), len(mp))
	}
}

func Bus_eta(ctx context.Context, client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	log.Printf("[BUS_ETA] action=Bus_eta event=start")
	for _, city := range cities {
		if city == "ChanghuaCounty" || city == "NantouCounty" {
			continue
		}
		log.Printf("[BUS_ETA] action=Bus_eta city=%s event=city_start", city)
		mp, err := busstaticmp(ctx, db, city)
		if err != nil || len(mp) <= 0 {
			log.Printf("[BUS_ETA] action=Bus_eta city=%s event=skip_empty reason=no_stations", city)
			continue
		}
		var eat []raw_Bus_Esimated
		var url string
		if city == "InterCity" {
			url = "/v2/Bus/EstimatedTimeOfArrival/InterCity"
		} else {
			url = fmt.Sprintf("/v2/Bus/EstimatedTimeOfArrival/City/%s", city)
		}
		dec, comp, err, flipopen := call_api(client, rc, url, "bus_EstimatedTimeOfArrival"+city)
		if err == nil && comp {
			if _, err := dec.Token(); err == nil {
				for dec.More() {
					var e raw_Bus_Esimated
					if err := dec.Decode(&e); err == nil {
						eat = append(eat, e)
					}
				}
			}
			flipopen()
		}
		var posit []raw_Bus_Position
		if city == "InterCity" {
			url = "/v2/Bus/RealTimeByFrequency/InterCity"
		} else {
			url = fmt.Sprintf("/v2/Bus/RealTimeByFrequency/City/%s", city)
		}
		dec, comp, err, flipopen = call_api(client, rc, url, "bus_RealTimeByFrequency"+city)
		if err == nil && comp {
			if _, err := dec.Token(); err == nil {
				for dec.More() {
					var p raw_Bus_Position
					if err := dec.Decode(&p); err == nil {
						posit = append(posit, p)
					}
				}
			}
			flipopen()
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
			eta, ok := etamap[b.StopUID]
			var est, status uint8
			var stime string
			if ok {
				est = uint8(eta.EstimatedTime)
				status = eta.StopStatus
				stime = eta.SrcUpdateTime
			} else {
				status = 67
			}
			if _, ok = stations[b.StationName]; !ok {
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
				NextBusTime:   string(eta.NextBusTime),
				StopStatus:    int32(status),
				SrcUpdateTime: stime,
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
				NextBusTime:   string(eta.NextBusTime),
				SrcUpdateTime: stime,
				Buses:         busmap[b.SubRouteUID],
			})
		}
		pipe := rc.Pipeline()
		for name, pb := range stations {
			data, _ := proto.Marshal(pb)
			pipe.Set(fmt.Sprintf("bus_eta_station:%s:%s", city, name), data, 180*time.Second)
		}
		for uid, pb := range routes {
			data, _ := proto.Marshal(pb)
			pipe.Set(fmt.Sprintf("bus_eta_route:%s:%s", city, uid), data, 180*time.Second)
		}
		_, err = pipe.Exec()
		if err != nil {
			log.Printf("[BUS_ETA] action=Bus_eta city=%s event=redis_error error=%v station_count=%d route_count=%d eat_count=%d posit_count=%d", city, err, len(stations), len(routes), len(eat), len(posit))
		} else {
			log.Printf("[BUS_ETA] action=Bus_eta city=%s event=redis_success station_count=%d route_count=%d eat_count=%d posit_count=%d", city, len(stations), len(routes), len(eat), len(posit))
		}
		//savebushistory(ctx, db, eat, posit)
	}
	log.Printf("[BUS_ETA] action=Bus_eta event=complete")
}
