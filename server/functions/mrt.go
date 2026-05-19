package main

import (
	"github.com/jnjkhjlkjhb8/bus/models"

	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis"
	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"
)

type mrt_station struct {
	StationPosition struct {
		PositionLon float64 `json:"PositionLon"`
		PositionLat float64 `json:"PositionLat"`
	} `json:"StationPosition"`
	city               string `json:"LocationCity"`
	serviceType        string
	StationID          string `json:"StationID"`
	BikeAllowOnHoliday bool   `json:"BikeAllowOnHoliday"`
}
type mrt_firstlast struct {
	LineID                 string `json:"LineID"`
	StationID              string `json:"StationID"`
	DestinationStaionID    string `json:"DestinationStaionID"`
	DestinationStationName struct {
		ZhTw string `json:"Zh_tw"`
	} `json:"DestinationStationName"`
	FirstTrainTime string `json:"FirstTrainTime"`
	LastTrainTime  string `json:"LastTrainTime"`
	ServiceDay     struct {
		Monday           bool `json:"Monday"`
		Tuesday          bool `json:"Tuesday"`
		Wednesday        bool `json:"Wednesday"`
		Thursday         bool `json:"Thursday"`
		Friday           bool `json:"Friday"`
		Saturday         bool `json:"Saturday"`
		Sunday           bool `json:"Sunday"`
		NationalHolidays bool `json:"NationalHolidays"`
	} `json:"ServiceDay"`
}
type mrt_live struct {
	LineID                 string `json:"LineID"`
	StationID              string `json:"StationID"`
	TripHeadSign           string `json:"TripHeadSign"`
	DestinationStaionID    string `json:"DestinationStaionID"`
	DestinationStationName struct {
		ZhTw string `json:"Zh_tw"`
	} `json:"DestinationStationName"`
	ServiceStatus uint8 `json:"ServiceStatus"`
	EstimateTime  int32 `json:"EstimateTime"`
}

func mrt_static(client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	getmrt_station(client, rc, db)
	getmrt_firstlast(client, rc, db)
}
func getmrt_station(client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	var systems = []string{"TRTC", "KRTC", "KLRT", "TYMC"}
	for _, system := range systems {
		dec, comp, err, flipopen := call_api(client, rc, fmt.Sprintf("/v2/Rail/Station/%s", system), "mrt_stations")
		if err != nil || !comp {
			return
		}
		if _, err := dec.Token(); err != nil {
			flipopen()
			return
		}
		var row [][]interface{}
		for dec.More() {
			var temp mrt_station
			if err := dec.Decode(&temp); err == nil {
				g := fmt.Sprintf("POINT(%.6f %.6f)", temp.StationPosition.PositionLon, temp.StationPosition.PositionLat)
				row = append(row, []interface{}{
					g,
					system,
					temp.serviceType,
					temp.StationID,
					temp.BikeAllowOnHoliday,
				})
			}
		}
		if len(row) > 0 {
			c1 := `CREATE TEMP TABLE temp_mrt (
                                geom text,
                                system text,
                                type text,
                                id text,
                                bike bool
					) ON COMMIT DROP;`
			c2 := `INSERT INTO mrt_station (
						stationposition,
						system,
						servicetype,
						stationid,
						bikeallowonholiday,
						"created_at"
					)
					SELECT st_geomfromtext(geom, 4326), system, type, id, bike, address,NOW() FROM temp_mrt
					ON CONFLICT (station_uid) DO UPDATE SET name = EXCLUDED.name,system = EXCLUDED.system,servicetype = EXCLUDED.servicetype,stationposition = EXCLUDED.stationposition,updated_at = NOW();`
			b, err := db.Begin(ctx)
			if err != nil {
				log.Println(err.Error())
				continue
			}
			_, _ = b.Exec(ctx, c1)
			_, err = b.CopyFrom(ctx, pgx.Identifier{"temp_mrt"}, []string{"geom", "system", "type", "id", "bike"}, pgx.CopyFromRows(row))
			if err == nil {
				b.Exec(ctx, c2)
				b.Commit(ctx)
				log.Println("mrt station updated_" + system)
			} else {
				log.Println(err.Error())
				_ = b.Rollback(ctx)
			}
		}
		flipopen()
	}
}
func getmrt_firstlast(client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	var systems = []string{"TRTC", "KRTC", "KLRT", "TYMC"}
	for _, system := range systems {
		dec, comp, err, flipopen := call_api(client, rc, fmt.Sprintf("/v2/Rail/FirstLastTimetable/%s", system), "mrt_firstlast")
		if err != nil || !comp {
			return
		}
		if _, err := dec.Token(); err != nil {
			flipopen()
			return
		}
		func() {
			defer flipopen()
			var row [][]interface{}

			for dec.More() {
				var temp mrt_firstlast
				if err := dec.Decode(&temp); err == nil {
					row = append(row, []interface{}{
						temp.StationID,
						temp.LineID,
						temp.DestinationStaionID,
						temp.DestinationStationName,
						temp.FirstTrainTime,
						temp.LastTrainTime,
						mask(temp.ServiceDay.Monday, temp.ServiceDay.Tuesday, temp.ServiceDay.Wednesday, temp.ServiceDay.Thursday, temp.ServiceDay.Friday, temp.ServiceDay.Saturday, temp.ServiceDay.Sunday, temp.ServiceDay.NationalHolidays),
						system,
					})
				}
			}
			if len(row) > 0 {
				c1 := `CREATE TEMP TABLE temp_mrt (
                               id text,
                               lid text,
                               dsid text,
                               dsname text,
                               ft text,
                               lt text,
       						   mask int2,
       						   sys text
					) ON COMMIT DROP;`
				c2 := `INSERT INTO mrt_firstlast (
						stationid,
						lineid,
						destinationstaionid,
						destinationstationname,
						firsttraintime,
						lasttraintime,
						serviceday,
						system,
						created_at
					)
					SELECT id,lid, dsid, dsname, ft, lt,mask,sys,NOW() FROM temp_mrt;`
				b, err := db.Begin(ctx)
				if err != nil {
					log.Println(err.Error())
					return
				}
				_, _ = b.Exec(ctx, c1)
				_, err = b.CopyFrom(ctx, pgx.Identifier{"temp_mrt"}, []string{"id", "lid", "dsid", "dsname", "ft", "lt", "mask", "sys"}, pgx.CopyFromRows(row))
				if err == nil {
					b.Exec(ctx, c2)
					b.Commit(ctx)
					log.Println("mrt firtlast updated_" + system)
				} else {
					log.Println(err.Error())
					_ = b.Rollback(ctx)
				}
			}
		}()
	}
}
func mrt_eta(client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	var systems = []string{"TRTC", "KRTC", "KLRT", "TYMC"}
	for _, system := range systems {
		dec, comp, err, flipopen := call_api(client, rc, fmt.Sprintf("/v2/Rail/Metro/LiveBoard/%s", system), "mrt_LiveBoard")
		if err != nil || !comp {
			return
		}
		if _, err := dec.Token(); err != nil {
			flipopen()
			return
		}
		func() {
			defer flipopen()
			pipe := rc.Pipeline()
			for dec.More() {
				var temp mrt_live
				if err := dec.Decode(&temp); err == nil {
					raw := &models.MrtLive{
						LineID:                 temp.LineID,
						StationID:              temp.StationID,
						System:                 system,
						TripHeadSign:           temp.TripHeadSign,
						DestinationStaionID:    temp.DestinationStaionID,
						DestinationStationName: temp.DestinationStationName.ZhTw,
						ServiceStatus:          int32(temp.ServiceStatus),
						EstimateTime:           temp.EstimateTime,
					}
					pb, err := proto.Marshal(raw)
					if err != nil {
						continue
					}
					pipe.Set(fmt.Sprintf("mrt_live:%s", temp.StationID), pb, 2*time.Minute)
				}
			}
			_, _ = pipe.Exec()
		}()
	}
	log.Println("mrt eta done")
}
