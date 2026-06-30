package main

import (
	"github.com/jnjkhjlkjhb8/wheres_the_car/models"

	"context"
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis"
	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"
)

var cities = []string{
	"Taipei", "NewTaipei", "Taoyuan", "Taichung", "Tainan", "Kaohsiung",
	"InterCity", "Hsinchu", "HsinchuCounty", "MiaoliCounty", "ChanghuaCounty",
	"NantouCounty", "YunlinCounty", "ChiayiCounty", "Chiayi", "PingtungCounty",
	"YilanCounty", "HualienCounty", "TaitungCounty", "PenghuCounty", "KinmenCounty", "LienchiangCounty", "Keelung",
}
var citymap = map[string]string{
	"Taipei": "TPE", "NewTaipei": "NWT", "Taoyuan": "TAO", "Taichung": "TXG",
	"Tainan": "TNN", "Kaohsiung": "KHH", "InterCity": "THB", "Keelung": "KEE",
	"Hsinchu": "HSZ", "HsinchuCounty": "HSQ", "Miaoli": "MIA", "Changhua": "CHA",
	"Nantou": "NAN", "Chiayi": "CYI", "ChiayiCounty": "CYQ", "Yunlin": "YUN",
	"Pingtung": "PIF", "Yilan": "ILA", "Hualien": "HUA", "Taitung": "TTT",
	"Penghu": "PEN", "Kinmen": "KIN", "Lienchiang": "LIE",
}
var citymap2 = map[string]string{
	"TPE": "Taipei", "NWT": "NewTaipei", "TAO": "Taoyuan", "TXG": "Taichung",
	"TNN": "Tainan", "KHH": "Kaohsiung", "THB": "InterCity", "KEE": "Keelung",
	"HSZ": "Hsinchu", "HSQ": "HsinchuCounty", "MIA": "Miaoli", "CHA": "Changhua",
	"NAN": "Nantou", "CYI": "Chiayi", "CYQ": "ChiayiCounty", "YUN": "Yunlin",
	"PIF": "Pingtung", "ILA": "Yilan", "HUA": "Hualien", "TTT": "Taitung",
	"PEN": "Penghu", "KIN": "Kinmen", "LIE": "Lienchiang",
}

func busRouteEtaKey(subRouteUID string) string {
	return fmt.Sprintf("bus_eta_route:%s", subRouteUID)
}

type rawBusRoute struct {
	RouteUID  string `json:"RouteUID"`
	RouteName struct {
		Zhtw string `json:"Zh_tw"`
	} `json:"RouteName"`
	DepartureStopNameZh   string `json:"DepartureStopNameZh"`
	DestinationStopNameZh string `json:"DestinationStopNameZh"`
	SubRoutes             []struct {
		SubRouteUID  string `json:"SubRouteUID"`
		SubRouteID   string `json:"SubRouteID"`
		SubRouteName struct {
			Zhtw string `json:"Zh_tw"`
		} `json:"SubRouteName"`
		Direction             uint8  `json:"Direction"`
		DepartureStopNameZh   string `json:"DepartureStopNameZh,omitempty"`
		DestinationStopNameZh string `json:"DestinationStopNameZh,omitempty"`
		FirstBusTime          string `json:"FirstBusTime"`
		LastBusTime           string `json:"LastBusTime"`
		HolidayFirstBusTime   string `json:"HolidayFirstBusTime"`
		HolidayLastBusTime    string `json:"HolidayLastBusTime"`
	} `json:"SubRoutes"`
}
type rawBusFare struct {
	RouteID         string `json:"RouteID"`
	SubRouteID      string `json:"SubRouteID"`
	FarePricingType uint8  `json:"FarePricingType"`
	IsFreeBus       uint8  `json:"IsFreeBus"`
	SectionFares    []struct {
		SectionSequence uint8 `json:"SectionSequence"`
		BufferZones     []struct {
			StopUID      string `json:"StopUID"`
			StopID       string `json:"StopID"`
			StopSequence int32  `json:"StopSequence"`
		} `json:"BufferZones"`
		Fares []struct {
			TicketType uint8 `json:"TicketType"`
			FareClass  uint8 `json:"FareClass"`
			Price      int32 `json:"Price"`
		} `json:"Fares"`
	} `json:"SectionFares"`
}
type rawStopofroute struct {
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
type rawBusDailytimetable struct {
	SubRouteUID string `json:"SubRouteUID"`
	Direction   uint8  `json:"Direction"`
	Timetables  []struct {
		TripID     string `json:"TripID"`
		IsLowFloor bool   `json:"IsLowFloor"`
		StopTimes  []struct {
			StopSequence  int    `json:"StopSequence"`
			StopUID       string `json:"StopUID"`
			ArrivalTime   string `json:"ArrivalTime"`
			DepartureTime string `json:"DepartureTime"`
		} `json:"StopTimes"`
	} `json:"Timetables"`
}
type rawBusSchedule struct {
	SubRouteUID string `json:"SubRouteUID"`
	RouteUID    string `json:"RouteUID"`
	Direction   uint8  `json:"Direction"`
	Timetables  []struct {
		TripID     string `json:"TripID"`
		IsLowFloor bool   `json:"IsLowFloor"`
		ServiceDay struct {
			Sunday    uint8 `json:"Sunday"`
			Monday    uint8 `json:"Monday"`
			Tuesday   uint8 `json:"Tuesday"`
			Wednesday uint8 `json:"Wednesday"`
			Thursday  uint8 `json:"Thursday"`
			Friday    uint8 `json:"Friday"`
			Saturday  uint8 `json:"Saturday"`
		} `json:"ServiceDay"`
		StopTimes []struct {
			StopSequence int    `json:"StopSequence"`
			StopUID      string `json:"StopUID"`
			StopName     struct {
				Zhtw string `json:"Zh_tw"`
			} `json:"StopName"`
			ArrivalTime   string `json:"ArrivalTime"`
			DepartureTime string `json:"DepartureTime"`
		} `json:"StopTimes"`
	} `json:"Timetables"`
	Frequencys []struct {
		StartTime      string `json:"StartTime"`
		EndTime        string `json:"EndTime"`
		MinHeadwayMins uint8  `json:"MinHeadwayMins"`
		MaxHeadwayMins uint8  `json:"MaxHeadwayMins"`
		ServiceDay     struct {
			Sunday    uint8 `json:"Sunday"`
			Monday    uint8 `json:"Monday"`
			Tuesday   uint8 `json:"Tuesday"`
			Wednesday uint8 `json:"Wednesday"`
			Thursday  uint8 `json:"Thursday"`
			Friday    uint8 `json:"Friday"`
			Saturday  uint8 `json:"Saturday"`
		} `json:"ServiceDay"`
	} `json:"Frequencys"`
}

type rawBusEsimated struct {
	PlateNumb     string `json:"PlateNumb"`
	StopUID       string `json:"StopUID"`
	SubRouteUID   string `json:"SubRouteUID"`
	Direction     uint8  `json:"Direction"`
	EstimatedTime int32  `json:"EstimatedTime"`
	NextBusTime   string `json:"NextBusTime"`
	StopStatus    uint8  `json:"StopStatus"`
	SrcUpdateTime string `json:"SrcUpdateTime"`
}
type rawBusPosition struct {
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
type busStationmap struct {
	StationUID   string
	StationName  string
	SubRouteUID  string
	SubRouteName string
	Direction    uint8
	StopUID      string
	StopSequence uint8
	Lat          float64
	Lon          float64
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
type rawBusShape struct {
	SubRouteUID string `json:"SubRouteUID,omitempty"`
	RouteUID    string `json:"RouteUID"`
	Direction   uint8  `json:"Direction"`
	Geometry    string `json:"Geometry"`
	UpdateTime  string `json:"UpdateTime"`
}

type rawBusStation struct {
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

func busStatic(ctx context.Context, client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	dailyRoute(ctx, rc, client, db)
	busDailyroute(client, rc)
	busFare(ctx, client, rc, db)
}
func dailyRoute(ctx context.Context, rc *redis.Client, c *resty.Client, db *pgxpool.Pool) {
	log.Printf("[BUS] action=dailyRoute event=start")
	for _, city := range cities {
		if city == "LienchiangCounty" {
			continue
		}
		log.Printf("[BUS] action=dailyRoute city=%s event=city_start", city)
		subRoutemap := make(map[string]*models.BusSubroute)
		routeMap := make(map[string][]string)
		syncStart := time.Now()
		_, err := db.Exec(ctx, "DELETE FROM raw_bus_route WHERE destin = $1 OR depart = $1", city)
		if err != nil {
			log.Printf("[BUS] action=dailyRoute city=%s event=cleanup_error error=%v", city, err)
		}
		processStatic(ctx, c, rc, db, city, "Route", func(raw []byte) {
			var r rawBusRoute
			err := json.Unmarshal(raw, &r)
			if err != nil {
				log.Printf("[BUS] action=dailyRoute city=%s api=Route event=unmarshal_error error=%v", city, err)
			}
			for _, sub := range r.SubRoutes {
				uid, dir := makethatsame(city, sub.SubRouteUID, sub.Direction)
				dep, dest := sub.DepartureStopNameZh, sub.DestinationStopNameZh
				if dep == "" {
					dep = r.DepartureStopNameZh
				}
				if dest == "" {
					dest = r.DestinationStopNameZh
				}
				if sub.Direction == 1 {
					dep, dest = dest, dep
				}
				if _, ok := subRoutemap[uid]; !ok {
					subRoutemap[uid] = &models.BusSubroute{
						RouteUID:            r.RouteUID,
						RouteName:           r.RouteName.Zhtw,
						SubRouteUID:         uid,
						SubRouteName:        sub.SubRouteName.Zhtw,
						City:                city,
						DepartureStopName:   dep,
						DestinationStopName: dest,
						Directions:          make(map[int32]*models.Direction),
					}
				}
				subRoutemap[uid].Directions[int32(dir)] = &models.Direction{
					DepartureStopName:   dep,
					DestinationStopName: dest,
					FirstBusTime:        sub.FirstBusTime,
					LastBusTime:         sub.LastBusTime,
					HolidayFirstBusTime: sub.HolidayFirstBusTime,
					HolidayLastBusTime:  sub.HolidayLastBusTime,
				}
				routeMap[r.RouteUID] = append(routeMap[r.RouteUID], uid)
			}
		})
		processStatic(ctx, c, rc, db, city, "StopOfRoute", func(raw []byte) {
			var r rawStopofroute
			err := json.Unmarshal(raw, &r)
			if err != nil {
				log.Printf("[BUS] action=dailyRoute city=%s api=StopOfRoute event=unmarshal_error error=%v", city, err)
			}
			uid, dir := makethatsame(city, r.SubRouteUID, r.Direction)
			if sr, ok := subRoutemap[uid]; ok {
				if d, ok := sr.Directions[int32(dir)]; ok {
					for _, stop := range r.Stops {
						d.Stops = append(d.Stops, &models.BusStop{
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
		processStatic(ctx, c, rc, db, city, "Shape", func(raw []byte) {
			var r rawBusShape
			err := json.Unmarshal(raw, &r)
			if err != nil {
				log.Printf("[BUS] action=dailyRoute event=marshal_error error=%v\n", err)
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
		processStatic(ctx, c, rc, db, city, "Schedule", func(raw []byte) {
			var r rawBusSchedule
			err := json.Unmarshal(raw, &r)
			if err != nil {
				log.Printf("[BUS] action=dailyRoute city=%s api=Schedule event=unmarshal_error error=%v", city, err)
			}
			uid, dir := makethatsame(city, r.SubRouteUID, r.Direction)
			if sr, ok := subRoutemap[uid]; ok {
				if d, ok := sr.Directions[int32(dir)]; ok {
					for _, t := range r.Timetables {
						m := mask2(t.ServiceDay.Monday, t.ServiceDay.Tuesday, t.ServiceDay.Wednesday, t.ServiceDay.Thursday, t.ServiceDay.Friday, t.ServiceDay.Saturday, t.ServiceDay.Sunday)
						for _, temp := range t.StopTimes {
							d.Schedules = append(d.Schedules, &models.Bus_Schedule{
								Type:                        true,
								Tripid:                      t.TripID,
								Islowfloor:                  t.IsLowFloor,
								MinHeadwayMinsArrivalTime:   temp.ArrivalTime,
								MaxHeadwayMinsDepartureTime: temp.DepartureTime,
								ServiceDay:                  int32(m),
							})
						}
					}
					for _, t := range r.Frequencys {
						m := mask2(t.ServiceDay.Monday, t.ServiceDay.Tuesday, t.ServiceDay.Wednesday, t.ServiceDay.Thursday, t.ServiceDay.Friday, t.ServiceDay.Saturday, t.ServiceDay.Sunday)
						d.Schedules = append(d.Schedules, &models.Bus_Schedule{
							Type:                        false,
							Start_Time:                  t.StartTime,
							End_Time:                    t.EndTime,
							MinHeadwayMinsArrivalTime:   strconv.Itoa(int(t.MinHeadwayMins)),
							MaxHeadwayMinsDepartureTime: strconv.Itoa(int(t.MaxHeadwayMins)),
							ServiceDay:                  int32(m),
						})
					}
				}
			}
		})
		processStatic(ctx, c, rc, db, city, "Station", func(raw []byte) {})
		changetodbformat(ctx, db, &subRoutemap)
		savestations(ctx, db, city)
		saveschedule(ctx, db, city)
		if _, delErr := db.Exec(ctx, `DELETE FROM bus_station_stop_map WHERE sub_route_uid LIKE $1`, citymap[city]+"%"); delErr != nil {
			log.Printf("[BUS] action=dailyRoute city=%s event=delete_stop_map_error error=%v", city, delErr)
		}
		savestatictodb(ctx, db, &subRoutemap)
		if _, delErr := db.Exec(ctx, `DELETE FROM bus_subroutes WHERE city = $1 AND updated_at < $2`, city, syncStart); delErr != nil {
			log.Printf("[BUS] action=dailyRoute city=%s event=delete_stale_subroutes_error error=%v", city, delErr)
		}
		if _, delErr := db.Exec(ctx, `DELETE FROM bus_static WHERE city = $1 AND updated_at < $2`, city, syncStart); delErr != nil {
			log.Printf("[BUS] action=dailyRoute city=%s event=delete_stale_static_error error=%v", city, delErr)
		}
		if err != nil {
			log.Printf("[BUS] action=dailyRoute city=%s event=cleanup_raw_error error=%v", city, err)
		}
		log.Printf("[BUS] action=dailyRoute city=%s event=city_complete subroute_count=%d", city, len(subRoutemap))
	}
	invalidateBusStaticMap()
	log.Printf("[BUS] action=dailyRoute event=complete")
}
func changetodbformat(ctx context.Context, db *pgxpool.Pool, raw *map[string]*models.BusSubroute) {
	row := [][]interface{}{}
	for _, sub := range *raw {
		for dir, d := range sub.Directions {
			stops, err := json.Marshal(d.Stops)
			schedules, err := json.Marshal(d.Schedules)
			if err != nil {
				log.Printf("[BUS] action=changetodbformat event=marshal_error error=%v", err)
			}
			row = append(row, []interface{}{
				sub.SubRouteUID,
				sub.RouteUID,
				dir,
				sub.RouteName,
				sub.SubRouteName,
				sub.City,
				d.DepartureStopName,
				d.DestinationStopName,
				d.Geometry,
				stops,
				schedules,
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
			city text,
			depart text,
			destin text,
			geom text,
    		rawstop jsonb,
			schedule jsonb
                          ) ON COMMIT DROP
		    `
	c2 := `
			INSERT INTO bus_subroutes(
				sub_route_uid,
				route_uid,
				direction,
				route_name,
				sub_route_name,
				city,
				depart,
				destin,
				geometry,
				stops,
				schedule
			)
			SELECT uid, rid, d, name1, name2,city,depart,destin,geom,
				   ARRAY(
					   SELECT ROW(
								  s ->> 'StationUID',
								  s ->> 'StopName',
								  (s ->> 'StopSequence')::int,
								  (s ->> 'PositionLon')::float,
								  (s ->> 'PositionLat')::float
							  )::stop
					   FROM jsonb_array_elements(rawstop) AS s
				   ),schedule
			FROM temp_bus 
			ON CONFLICT (sub_route_uid, direction)
			DO UPDATE SET city = excluded.city,geometry = EXCLUDED.geometry,stops = EXCLUDED.stops,depart = EXCLUDED.depart,destin = EXCLUDED.destin,schedule = EXCLUDED.schedule,updated_at = NOW();
			`
	if _, err := b.Exec(ctx, c1); err != nil {
		log.Printf("[BUS] action=changetodbformat event=create_temp_error error=%v", err)
	}
	if _, err := b.CopyFrom(ctx, pgx.Identifier{"temp_bus"}, []string{"uid", "rid", "d", "name1", "name2", "city", "depart", "destin", "geom", "rawstop", "schedule"}, pgx.CopyFromRows(row)); err != nil {
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
func saveschedule(ctx context.Context, db *pgxpool.Pool, city string) {
	syncStart := time.Now()
	rows, err := db.Query(ctx, `SELECT content FROM raw_bus_route WHERE type = 'Schedule' AND destin = $1`, city)
	if err != nil {
		log.Printf("[BUS] action=saveschedule city=%s event=query_error error=%v", city, err)
		return
	}
	defer rows.Close()
	row := [][]interface{}{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		var temp rawBusSchedule
		if err := json.Unmarshal(raw, &temp); err != nil {
			log.Printf("[BUS] action=saveschedule city=%s event=unmarshal_error error=%v", city, err)
			continue
		}
		uid, dir := makethatsame(city, temp.SubRouteUID, temp.Direction)
		for _, t := range temp.Frequencys {
			row = append(row, []interface{}{
				uid, int16(dir), true, "", false, int16(-1), strconv.Itoa(int(t.MinHeadwayMins)), strconv.Itoa(int(t.MaxHeadwayMins)), t.StartTime, t.EndTime, int16(mask2(t.ServiceDay.Monday, t.ServiceDay.Tuesday, t.ServiceDay.Wednesday, t.ServiceDay.Thursday, t.ServiceDay.Friday, t.ServiceDay.Saturday, t.ServiceDay.Sunday)),
			})
		}
		for _, t := range temp.Timetables {
			for _, st := range t.StopTimes {
				row = append(row, []interface{}{
					uid, dir, false, t.TripID, t.IsLowFloor, st.StopSequence, st.StopUID, st.StopName.Zhtw, st.ArrivalTime, st.ArrivalTime, mask2(t.ServiceDay.Monday, t.ServiceDay.Tuesday, t.ServiceDay.Wednesday, t.ServiceDay.Thursday, t.ServiceDay.Friday, t.ServiceDay.Saturday, t.ServiceDay.Sunday),
				})
			}
		}
	}
	if len(row) == 0 {
		log.Printf("[BUS] action=saveschedule city=%s event=skip reason=no_rows", city)
		return
	}
	b, err := db.Begin(ctx)
	if err != nil {
		log.Printf("[BUS] action=saveschedule city=%s event=begin_error error=%v", city, err)
		return
	}
	c1 := `CREATE TEMP TABLE temp_bus (
					uid text,
					dir smallint,
					type bool,
					id text,
					floor bool,
					seq smallint,
					stopuid text,
					stopname text,
					arrival text,
					departure text,
					sdays smallint
				) ON COMMIT DROP`
	c2 := `INSERT INTO bus_schedule (sub_route_uid, direction, type, tripid, islowfloor, stopsequence, "stop_uid/MinHeadwayMins", "stop_name/MaxHeadwayMins", "arrival_time/StartTime", "departure_time/EndTime", service_day, updated_at)
				SELECT uid, dir, type, id, floor, seq, stopuid, stopname, arrival::time, departure::time, sdays, NOW() FROM temp_bus
				ON CONFLICT (sub_route_uid, direction, type, service_day, tripid, "stop_uid/MinHeadwayMins")
				DO UPDATE SET islowfloor = EXCLUDED.islowfloor, stopsequence = EXCLUDED.stopsequence,
					"stop_name/MaxHeadwayMins" = EXCLUDED."stop_name/MaxHeadwayMins",
					"arrival_time/StartTime" = EXCLUDED."arrival_time/StartTime",
					"departure_time/EndTime" = EXCLUDED."departure_time/EndTime",
					updated_at = NOW()`
	defer func(b pgx.Tx, ctx context.Context) {
		_ = b.Rollback(ctx)
	}(b, ctx)
	_, _ = b.Exec(ctx, c1)
	if _, err = b.CopyFrom(ctx, pgx.Identifier{"temp_bus"},
		[]string{"uid", "dir", "type", "id", "floor", "seq", "stopuid", "stopname", "arrival", "departure", "sdays"},
		pgx.CopyFromRows(row)); err != nil {
		log.Printf("[BUS] action=saveschedule city=%s event=copyfrom_error error=%v", city, err)
		return
	}
	if _, err = b.Exec(ctx, c2); err != nil {
		log.Printf("[BUS] action=saveschedule city=%s event=insert_error error=%v", city, err)
		return
	}
	if err = b.Commit(ctx); err != nil {
		log.Printf("[BUS] action=saveschedule city=%s event=commit_error error=%v", city, err)
		return
	}
	if _, delErr := db.Exec(ctx, `DELETE FROM bus_schedule WHERE sub_route_uid LIKE $1 AND updated_at < $2`, citymap[city]+"%", syncStart); delErr != nil {
		log.Printf("[BUS] action=saveschedule city=%s event=cleanup_error error=%v", city, delErr)
	}
	log.Printf("[BUS] action=saveschedule city=%s event=complete row_count=%d", city, len(row))
}

func savestatictodb(ctx context.Context, db *pgxpool.Pool, raw *map[string]*models.BusSubroute) {
	row := [][]interface{}{}
	mp := [][]interface{}{}
	for _, sub := range *raw {
		pb, err := proto.Marshal(sub)
		if err != nil {
			log.Printf("[BUS] action=savestatictodb event=marshal_error subroute=%s error=%v", sub.SubRouteUID, err)
			continue
		}
		row = append(row, []interface{}{
			sub.SubRouteName, sub.RouteName, sub.SubRouteUID, sub.RouteUID, sub.City, sub.DepartureStopName, sub.DestinationStopName, pb,
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
	if err != nil {
		log.Printf("[BUS] action=savestatictodb event=begin_error error=%v", err)
		return
	}
	defer func(b pgx.Tx, ctx context.Context) {
		_ = b.Rollback(ctx)
	}(b, ctx)
	if len(row) > 0 {
		if _, err := b.Exec(ctx, `CREATE TEMP TABLE temp_pb (sname text,rname text,uid text,rid text,city text,depart text,destin text,pb bytea) ON COMMIT DROP`); err != nil {
			log.Printf("[BUS] action=savestatictodb event=create_temp_pb_error error=%v", err)
		}
		if _, err := b.CopyFrom(ctx, pgx.Identifier{"temp_pb"}, []string{"sname", "rname", "uid", "rid", "city", "depart", "destin", "pb"}, pgx.CopyFromRows(row)); err != nil {
			log.Printf("[BUS] action=savestatictodb event=copyfrom_pb_error error=%v row_count=%d", err, len(row))
		}
		if _, err := b.Exec(ctx, `INSERT INTO bus_static (
										sub_route_name,
										route_name,
										sub_route_uid,
										route_uid,
										city,
										depart,
										destin,
										pb
									)
									SELECT sname,rname,uid,rid,city,depart,destin,pb FROM temp_pb
									ON CONFLICT (sub_route_uid) DO UPDATE SET sub_route_name = excluded.sub_route_name,route_name = excluded.route_name, pb = excluded.pb,route_uid = excluded.route_uid,city = excluded.city,depart = excluded.depart,destin = excluded.destin,updated_at = NOW();`); err != nil {
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

func busEta(ctx context.Context, client *resty.Client, rc *redis.Client, db *pgxpool.Pool, dispatcher *notificationDispatcher) {
	log.Printf("[BUS_ETA] action=Bus_eta event=start")
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for _, city := range cities {
		if city == "ChanghuaCounty" || city == "NantouCounty" {
			continue
		}
		wg.Add(1)
		go func(city string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			processBusEtaCity(ctx, client, rc, db, city, dispatcher)
		}(city)
	}
	wg.Wait()
	log.Printf("[BUS_ETA] action=Bus_eta event=complete")
}

func processBusEtaCity(ctx context.Context, client *resty.Client, rc *redis.Client, db *pgxpool.Pool, city string, dispatcher *notificationDispatcher) {
	log.Printf("[BUS_ETA] action=Bus_eta city=%s event=city_start", city)
	prefix := citymap[city]
	mp, cached := cachedBusStaticMap(prefix)
	if !cached {
		var err error
		mp, err = busstaticmp(ctx, db, prefix)
		if err != nil || len(mp) <= 0 {
			log.Printf("[BUS_ETA] action=Bus_eta city=%s event=skip_empty reason=no_stations", city)
			return
		}
		storeBusStaticMap(prefix, mp)
	}
	if len(mp) <= 0 {
		log.Printf("[BUS_ETA] action=Bus_eta city=%s event=skip_empty reason=no_stations", city)
		return
	}
	var eat []rawBusEsimated
	var url string
	if city == "InterCity" {
		url = "/v2/Bus/EstimatedTimeOfArrival/InterCity"
	} else {
		url = fmt.Sprintf("/v2/Bus/EstimatedTimeOfArrival/City/%s", city)
	}
	dec, comp, err, flipopen := callApi(client, rc, url, "bus_EstimatedTimeOfArrival"+city)
	if err != nil || !comp {
		log.Printf("[BUS_ETA] action=Bus_eta city=%s event=skip_eta error=%v", city, err)
		return
	}
	if _, err := dec.Token(); err == nil {
		for dec.More() {
			var e rawBusEsimated
			if err := dec.Decode(&e); err == nil {
				eat = append(eat, e)
			}
		}
	}
	flipopen()
	var posit []rawBusPosition
	if city == "InterCity" {
		url = "/v2/Bus/RealTimeByFrequency/InterCity"
	} else {
		url = fmt.Sprintf("/v2/Bus/RealTimeByFrequency/City/%s", city)
	}
	dec, comp, err, flipopen = callApi(client, rc, url, "bus_RealTimeByFrequency"+city)
	if err != nil || !comp {
		log.Printf("[BUS_ETA] action=Bus_eta city=%s event=skip_position error=%v", city, err)
		return
	}
	if _, err := dec.Token(); err == nil {
		for dec.More() {
			var p rawBusPosition
			if err := dec.Decode(&p); err == nil {
				posit = append(posit, p)
			}
		}
	}
	flipopen()
	busmap := make(map[string][]*models.BusPosition)
	etamap := make(map[string]rawBusEsimated)
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
	totalStops := make(map[string]int)
	for _, b := range mp {
		uid, _ := makethatsame(city, b.SubRouteUID, b.Direction)
		totalStops[uid]++
	}
	var weather *weatherData
	if wjson, wErr := rc.Get("weather:" + city).Result(); wErr == nil {
		var w weatherData
		if json.Unmarshal([]byte(wjson), &w) == nil {
			weather = &w
		}
	}
	now := time.Now().In(taipei)
	holiday := isHoliday(now)
	var fillKeys []routeDirKey
	fillUIDs := make(map[string]bool)
	for _, b := range mp {
		if etaEnt, ok2 := etamap[b.StopUID]; ok2 && etaEnt.StopStatus == 1 && etaEnt.NextBusTime == "" {
			fillKeys = append(fillKeys, routeDirKey{b.SubRouteUID, int32(b.Direction)})
			fillUIDs[b.SubRouteUID] = true
		}
	}
	todTime := now.Format("15:04:05")
	depMap := batchNextDepartures(ctx, db, dedupRouteDirPairs(fillKeys), todTime)
	uidList := make([]string, 0, len(fillUIDs))
	for u := range fillUIDs {
		uidList = append(uidList, u)
	}
	travelAvgMap := batchTravelAvg(ctx, db, uidList, now.Hour(), int(now.Weekday()))
	maxAvgMap := make(map[routeDirKey]int)
	for k, v := range travelAvgMap {
		rk := routeDirKey{k.subRouteUID, k.direction}
		if v > maxAvgMap[rk] {
			maxAvgMap[rk] = v
		}
	}
	var historyRows [][]interface{}
	for _, b := range mp {
		eta, ok := etamap[b.StopUID]
		var status uint8
		var est int32
		var stime string
		uid, _ := makethatsame(city, b.SubRouteUID, b.Direction)
		if ok {
			if srcT, parseErr := time.Parse(time.RFC3339, eta.SrcUpdateTime); parseErr == nil {
				est = eta.EstimatedTime - int32(now.Sub(srcT).Seconds())
			} else {
				est = eta.EstimatedTime
			}
			status = eta.StopStatus
			stime = eta.SrcUpdateTime
		} else {
			status = 67
		}
		/*if eta.NextBusTime == "" {
			err = db.QueryRow(ctx, `SELECT ("arrival_time/StartTime" AT TIME ZONE 'Asia/Taipei')::time from bus_schedule WHERE sub_route_uid = $1 AND direction = $2 AND "arrival_time/StartTime" >= CURRENT_TIME ORDER BY "arrival_time/StartTime" ASC LIMIT 1;`, eta.SubRouteUID, eta.Direction).Scan(&eta.NextBusTime)
		}*/
		if status == 0 {
			uid2, _ := makethatsame(city, b.SubRouteUID, b.Direction)
			ts := totalStops[uid2]
			var plateNumb *string
			var busSpeed *int16
			var busDist *int
			if buses := busmap[uid]; len(buses) > 0 && b.Lat != 0 {
				nearest := buses[0]
				nearestDist := haversine(b.Lat, b.Lon,
					float64(nearest.PositionLat), float64(nearest.PositionLon))
				for _, bus := range buses[1:] {
					d := haversine(b.Lat, b.Lon,
						float64(bus.PositionLat), float64(bus.PositionLon))
					if d < nearestDist {
						nearestDist = d
						nearest = bus
					}
				}
				pn := nearest.PlateNumb
				plateNumb = &pn
				spd := int16(nearest.Speed)
				busSpeed = &spd
				dist := int(nearestDist)
				busDist = &dist
			}
			var srcTime *time.Time
			if stime != "" {
				if t, err := time.Parse(time.RFC3339, stime); err == nil {
					srcTime = &t
				}
			}
			var nextBusTimePtr *string
			if eta.NextBusTime != "" {
				nbtp := eta.NextBusTime
				nextBusTimePtr = &nbtp
			}
			weatherTemp, weatherPrecip, weatherWind, weatherHumid := interface{}(nil), interface{}(nil), interface{}(nil), interface{}(nil)
			if weather != nil {
				weatherTemp = weather.Temperature
				weatherPrecip = weather.Precipitation
				weatherWind = weather.WindSpeed
				weatherHumid = weather.Humidity
			}
			historyRows = append(historyRows, []interface{}{
				b.SubRouteUID, b.StopUID, int16(b.Direction),
				int16(b.StopSequence), int16(ts), est, nextBusTimePtr, srcTime,
				city, int16(now.Hour()), int16(now.Weekday()), holiday,
				weatherTemp, weatherPrecip, weatherWind, weatherHumid,
				plateNumb, busSpeed, busDist,
			})
		}
		if status == 1 && eta.NextBusTime == "" {
			uid2, _ := makethatsame(city, b.SubRouteUID, b.Direction)
			rk := routeDirKey{b.SubRouteUID, int32(b.Direction)}
			avgKey := travelAvgKey{b.SubRouteUID, int32(b.Direction), b.StopUID, now.Hour(), int(now.Weekday())}
			avgVal, hasAvg := travelAvgMap[avgKey]
			eta.NextBusTime = predictNextBusTime(rc,
				busStopCtx{
					subRouteUID:  b.SubRouteUID,
					direction:    int32(b.Direction),
					stopUID:      b.StopUID,
					city:         city,
					stopSequence: int(b.StopSequence),
					totalStops:   totalStops[uid2],
				},
				predictionInputs{
					now:          now,
					nextDep:      depMap[rk],
					travelAvg:    avgVal,
					hasTravelAvg: hasAvg,
					maxTravelAvg: maxAvgMap[rk],
				},
			)
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
			Estimate:      est,
			NextBusTime:   eta.NextBusTime,
			StopStatus:    int32(status),
			SrcUpdateTime: stime,
			Buses:         busmap[uid],
		})
		if shouldDispatchBusArrival(ok, status, est) {
			dispatcher.arrival(ctx, "bus", b.SubRouteUID, b.StopUID, strconv.Itoa(int(b.Direction)), est)
		}
		if _, ok = routes[b.SubRouteUID]; !ok {
			routes[b.SubRouteUID] = &models.Bus_RouteArrival{
				SubRouteUid: b.SubRouteUID,
				Stops:       make([]*models.Bus_RouteEstimate, 0),
			}
		}
		routes[b.SubRouteUID].Stops = append(routes[b.SubRouteUID].Stops, &models.Bus_RouteEstimate{
			StopUid:       b.StopUID,
			Direction:     int32(b.Direction),
			Estimate:      est,
			StopStatus:    int32(status),
			NextBusTime:   eta.NextBusTime,
			SrcUpdateTime: stime,
			Buses:         busmap[uid],
			StopSequence:  int32(b.StopSequence),
		})
	}
	pipe := rc.Pipeline()
	for name, pb := range stations {
		data, _ := proto.Marshal(pb)
		pipe.Set(fmt.Sprintf("bus_eta_station:%s:%s", city, name), data, 180*time.Second)
	}
	for uid, pb := range routes {
		data, _ := proto.Marshal(pb)
		key := busRouteEtaKey(uid)
		pipe.Set(key, data, 180*time.Second)
		pipe.Publish(key, data)
	}
	_, err = pipe.Exec()
	if err != nil {
		log.Printf("[BUS_ETA] action=Bus_eta city=%s event=redis_error error=%v station_count=%d route_count=%d eat_count=%d posit_count=%d", city, err, len(stations), len(routes), len(eat), len(posit))
	} else {
		log.Printf("[BUS_ETA] action=Bus_eta city=%s event=redis_success station_count=%d route_count=%d eat_count=%d posit_count=%d", city, len(stations), len(routes), len(eat), len(posit))
	}
	//savebushistory(ctx, db, eat, posit)
	saveBusEtaHistory(ctx, db, historyRows)
}

func shouldDispatchBusArrival(found bool, status uint8, etaSeconds int32) bool {
	return found && status == 0 && etaSeconds > 0
}
func busDailyroute(client *resty.Client, rc *redis.Client) {
	log.Printf("[bus] action=bus_dailyroute event=start")
	var temp rawBusDailytimetable
	for _, city := range cities {
		if city == "Taipei" || city == "NewTaipei" || city == "Tainan" || city == "KinmenCounty" || city == "LienchiangCounty" {
			continue
		}
		var url string
		if city == "InterCity" {
			url = fmt.Sprintf("/v2/Bus/DailyTimeTable/InterCity")
		} else {
			url = fmt.Sprintf("/v2/Bus/DailyTimeTable/City/%s", city)
		}
		log.Printf("[bus] action=bus_dailyroute city=%s event=city_start", city)
		dec, comp, err, flipopen := callApi(client, rc, url, "DailyTimeTable"+city)
		if err != nil || !comp {
			log.Printf("[bus] action=bus_dailyroute city=%s event=skip reason=api_error,error=%s", city, err)
			continue
		}
		if _, err := dec.Token(); err != nil {
			log.Printf("[bus] action=bus_dailyroute city=%s event=decode_error error=%v", city, err)
			continue
		}
		func() {
			defer flipopen()
			pipe := rc.Pipeline()
			mp := make(map[string]map[int32]*models.Temp, 300)
			for dec.More() {
				temp = rawBusDailytimetable{}
				if err := dec.Decode(&temp); err == nil {
					uid, dir := makethatsame(city, temp.SubRouteUID, temp.Direction)
					if _, exists := mp[uid]; !exists {
						mp[uid] = make(map[int32]*models.Temp, 4)
					}
					if _, exists := mp[uid][int32(dir)]; !exists {
						mp[uid][int32(dir)] = &models.Temp{
							DailyTimetables: make([]*models.Bus_DailyTimetable, 0, 64),
						}
					}
					for _, t := range temp.Timetables {
						stop := make([]*models.Temp_StopTimes, len(t.StopTimes))
						for i, st := range t.StopTimes {
							stop[i] = &models.Temp_StopTimes{
								StopSequence:  int32(st.StopSequence),
								ArrivalTime:   st.ArrivalTime,
								DepartureTime: st.DepartureTime,
							}
						}
						timtable := &models.Bus_DailyTimetable{
							TripID:     t.TripID,
							IsLowFloor: t.IsLowFloor,
							StopTimes:  stop,
						}
						mp[uid][int32(dir)].DailyTimetables = append(mp[uid][int32(dir)].DailyTimetables, timtable)
					}
				}
			}
			for subRouteUID, t := range mp {
				pbRoute := &models.Bus_DailyTimetables{
					SubRouteUID: subRouteUID,
					Direction:   t,
				}
				pb, err := proto.Marshal(pbRoute)
				if err != nil {
					log.Printf("[bus] action=bus_dailyroute subRouteUID=%s event=marshal_error error=%v", subRouteUID, err)
					continue
				}
				pipe.Set(fmt.Sprintf("bus_daily_timetable:%s", subRouteUID), pb, 23*time.Hour+30*time.Minute)
			}
			_, _ = pipe.Exec()
			log.Printf("[bus] action= %s bus_dailyroute event=complete", city)
		}()
	}
	log.Printf("[bus] action=bus_dailyroute event=complete")
}

func busFare(ctx context.Context, client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	log.Printf("[BUS_FARE] action=bus_fare event=start")
	for _, city := range cities {
		if city == "LienchiangCounty" {
			continue
		}
		url := fmt.Sprintf("/v2/Bus/Fare/City/%s", city)
		if city == "InterCity" {
			url = fmt.Sprintf("/v2/Bus/Fare/InterCity")
		}
		dec, comp, err, flipopen := callApi(client, rc, url, "bus_Fare"+city)
		if err != nil || !comp {
			log.Printf("[BUS_FARE] action=bus_fare city=%s event=skip reason=api_error", city)
			continue
		}
		func() {
			defer flipopen()
			if _, err := dec.Token(); err != nil {
				log.Printf("[BUS_FARE] action=bus_fare city=%s event=decode_error error=%v", city, err)
				return
			}
			pre := citymap[city]
			row := [][]interface{}{}
			for dec.More() {
				var f rawBusFare
				if err := dec.Decode(&f); err != nil {
					continue
				}
				if f.SubRouteID == "" {
					continue
				}
				uid := pre + f.SubRouteID
				sf, _ := json.Marshal(f.SectionFares)
				row = append(row, []interface{}{uid, int16(f.FarePricingType), f.IsFreeBus == 1, sf})
			}
			if len(row) == 0 {
				log.Printf("[BUS_FARE] action=bus_fare city=%s event=skip reason=no_data", city)
				return
			}
			c1 := `CREATE TEMP TABLE temp_fare (
						uid text,
						pt smallint,
						free bool,
						sf jsonb
					) ON COMMIT DROP`
			c2 := `INSERT INTO bus_fares (
						sub_route_uid, 
						fare_pricing_type,
						is_free_bus,
						section_fares,
						updated_at
					)
					SELECT uid, pt, free, sf, NOW() FROM temp_fare
					ON CONFLICT (sub_route_uid) DO UPDATE SET fare_pricing_type = EXCLUDED.fare_pricing_type, is_free_bus = EXCLUDED.is_free_bus, section_fares = EXCLUDED.section_fares, updated_at = NOW()`
			b, err := db.Begin(ctx)
			if err != nil {
				log.Printf("[BUS_FARE] action=bus_fare city=%s event=begin_error error=%v", city, err)
				return
			}
			defer func(b pgx.Tx, ctx context.Context) { _ = b.Rollback(ctx) }(b, ctx)
			if _, err := b.Exec(ctx, c1); err != nil {
				return
			}
			if _, err := b.CopyFrom(ctx, pgx.Identifier{"temp_fare"}, []string{"uid", "pt", "free", "sf"}, pgx.CopyFromRows(row)); err != nil {
				log.Printf("[BUS_FARE] action=bus_fare city=%s event=copyfrom_error error=%v", city, err)
				return
			}
			if _, err := b.Exec(ctx, c2); err != nil {
				log.Printf("[BUS_FARE] action=bus_fare city=%s event=insert_error error=%v", city, err)
				return
			}
			if err := b.Commit(ctx); err != nil {
				log.Printf("[BUS_FARE] action=bus_fare city=%s event=commit_error error=%v", city, err)
				return
			}
			log.Printf("[BUS_FARE] action=bus_fare city=%s event=success row_count=%d", city, len(row))
		}()
	}
	log.Printf("[BUS_FARE] action=bus_fare event=complete")
}
