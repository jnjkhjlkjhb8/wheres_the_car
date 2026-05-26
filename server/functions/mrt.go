package main

import (
	"context"

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

type mrtStation struct {
	StationPosition struct {
		PositionLon float64 `json:"PositionLon"`
		PositionLat float64 `json:"PositionLat"`
	} `json:"StationPosition"`
	LocationCity       string `json:"LocationCity"`
	serviceType        string
	StationID          string `json:"StationID"`
	BikeAllowOnHoliday bool   `json:"BikeAllowOnHoliday"`
	StationName        struct {
		ZhTw string `json:"Zh_tw"`
	} `json:"StationName"`
}
type mrtFirstlast struct {
	LineID                 string `json:"LineID"`
	StationID              string `json:"StationID"`
	TripHeadSign           string `json:"TripHeadSign"`
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
type mrtLive struct {
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

func mrtStatic(ctx context.Context, client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	log.Printf("[MRT] action=mrt_static event=start")
	getmrtStation(ctx, client, rc, db)
	getmrtFirstlast(ctx, client, rc, db)
	log.Printf("[MRT] action=mrt_static event=complete")
}
func getmrtStation(ctx context.Context, client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	log.Printf("[MRT] action=getmrt_station event=start")
	var systems = []string{"TRTC", "KRTC", "KLRT", "TYMC"}
	for _, system := range systems {
		log.Printf("[MRT] action=getmrt_station system=%s event=system_start", system)
		dec, comp, err, flipopen := callApi(client, rc, fmt.Sprintf("/v2/Rail/Metro/Station/%s", system), "mrt_stations"+system)
		if err != nil || !comp {
			log.Printf("[MRT] action=getmrt_station system=%s event=skip reason=api_error", system)
			continue
		}
		func() {
			defer flipopen()
			if _, err := dec.Token(); err != nil {
				log.Printf("[MRT] action=getmrt_station system=%s event=decode_error error=%v", system, err)
				return
			}
			var row [][]interface{}
			for dec.More() {
				var temp mrtStation
				if err := dec.Decode(&temp); err == nil {
					g := fmt.Sprintf("POINT(%.6f %.6f)", temp.StationPosition.PositionLon, temp.StationPosition.PositionLat)
					row = append(row, []interface{}{
						g,
						system,
						temp.StationName.ZhTw,
						temp.StationID,
						temp.BikeAllowOnHoliday,
					})
				}
			}
			if len(row) > 0 {
				c1 := `CREATE TEMP TABLE temp_mrt (
                                geom text,
                                system text,
								name text,
								id text,
                                bike bool
				) ON COMMIT DROP;`
				c2 := `INSERT INTO mrt_station (
					stationposition,
					system,
					name,
					station_id,
					bikeallowonholiday,
					created_at
				)
				SELECT st_geomfromtext(geom, 4326), system, name, id, bike,NOW() FROM temp_mrt
				ON CONFLICT (station_id) DO UPDATE SET name = EXCLUDED.name,system = EXCLUDED.system,stationposition = EXCLUDED.stationposition,created_at = NOW();`
				b, err := db.Begin(ctx)
				if err != nil {
					log.Printf("[MRT] action=getmrt_station system=%s event=begin_error error=%v", system, err)
					return
				}
				defer func(b pgx.Tx, ctx context.Context) {
					_ = b.Rollback(ctx)
				}(b, ctx)
				_, _ = b.Exec(ctx, c1)
				_, err = b.CopyFrom(ctx, pgx.Identifier{"temp_mrt"}, []string{"geom", "system", "name", "id", "bike"}, pgx.CopyFromRows(row))
				if err == nil {
					if _, execErr := b.Exec(ctx, c2); execErr != nil {
						log.Printf("[MRT] action=getmrt_station system=%s event=exec_error error=%v", system, execErr)
					}
					if commitErr := b.Commit(ctx); commitErr != nil {
						log.Printf("[MRT] action=getmrt_station system=%s event=commit_error error=%v", system, commitErr)
					} else {
						log.Printf("[MRT] action=getmrt_station system=%s event=success station_count=%d", system, len(row))
					}
				} else {
					log.Printf("[MRT] action=getmrt_station system=%s event=copyfrom_error error=%v", system, err)
					_ = b.Rollback(ctx)
				}
			}
		}()
	}
	log.Printf("[MRT] action=getmrt_station event=complete")
}
func getmrtFirstlast(ctx context.Context, client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	log.Printf("[MRT] action=getmrt_firstlast event=start")
	var systems = []string{"TRTC", "KRTC", "KLRT", "TYMC"}
	for _, system := range systems {
		log.Printf("[MRT] action=getmrt_firstlast system=%s event=system_start", system)
		dec, comp, err, flipopen := callApi(client, rc, fmt.Sprintf("/v2/Rail/Metro/FirstLastTimetable/%s", system), "mrt_firstlast"+system)
		if err != nil || !comp {
			log.Printf("[MRT] action=getmrt_firstlast system=%s event=skip reason=api_error", system)
			continue
		}
		func() {
			defer flipopen()
			if _, err := dec.Token(); err != nil {
				log.Printf("[MRT] action=getmrt_firstlast system=%s event=decode_error error=%v", system, err)
				return
			}
			var row [][]interface{}
			for dec.More() {
				var temp mrtFirstlast
				if err := dec.Decode(&temp); err == nil {
					row = append(row, []interface{}{
						temp.StationID,
						temp.LineID,
						temp.TripHeadSign,
						temp.DestinationStaionID,
						temp.DestinationStationName.ZhTw,
						temp.FirstTrainTime,
						temp.LastTrainTime,
						mask(temp.ServiceDay.Monday, temp.ServiceDay.Tuesday, temp.ServiceDay.Wednesday, temp.ServiceDay.Thursday, temp.ServiceDay.Friday, temp.ServiceDay.Saturday, temp.ServiceDay.Sunday),
						system,
					})
				}
			}
			if len(row) > 0 {
				c1 := `CREATE TEMP TABLE temp_mrt (
                               id text,
                               lid text,
							   sign text,
                               dsid text,
                               dsname text,
                               ft text,
                               lt text,
       						   mask int2,
       						   sys text
					) ON COMMIT DROP;`
				c2 := `INSERT INTO mrt_firstlast (
						station_id,
						lineid,
						destinationstaionid,
						destinationstationname,
						firsttraintime,
						lasttraintime,
						serviceday,
						system,
						created_at,
						trip_head_sign
					)
					SELECT id,lid, dsid, dsname, ft, lt,mask,sys,NOW(),sign FROM temp_mrt`
				b, err := db.Begin(ctx)
				if err != nil {
					log.Printf("[MRT] action=getmrt_firstlast system=%s event=begin_error error=%v", system, err)
					return
				}
				defer func(b pgx.Tx, ctx context.Context) {
					_ = b.Rollback(ctx)
				}(b, ctx)
				_, _ = b.Exec(ctx, c1)
				_, err = b.CopyFrom(ctx, pgx.Identifier{"temp_mrt"}, []string{"id", "lid", "sign", "dsid", "dsname", "ft", "lt", "mask", "sys"}, pgx.CopyFromRows(row))
				if err == nil {
					if _, execErr := b.Exec(ctx, c2); execErr != nil {
						log.Printf("[MRT] action=getmrt_firstlast system=%s event=exec_error error=%v", system, execErr)
					}
					if commitErr := b.Commit(ctx); commitErr != nil {
						log.Printf("[MRT] action=getmrt_firstlast system=%s event=commit_error error=%v", system, commitErr)
					} else {
						log.Printf("[MRT] action=getmrt_firstlast system=%s event=success row_count=%d", system, len(row))
					}
				} else {
					log.Printf("[MRT] action=getmrt_firstlast system=%s event=copyfrom_error error=%v", system, err)
					_ = b.Rollback(ctx)
				}
			}
		}()
	}
	log.Printf("[MRT] action=getmrt_firstlast event=complete")
}
func mrtEta(client *resty.Client, rc *redis.Client) {
	log.Printf("[MRT_ETA] action=mrt_eta event=start")
	var systems = []string{"TRTC", "KRTC", "KLRT", "TYMC"}
	for _, system := range systems {
		log.Printf("[MRT_ETA] action=mrt_eta system=%s event=system_start", system)
		dec, comp, err, flipopen := callApi(client, rc, fmt.Sprintf("/v2/Rail/Metro/LiveBoard/%s", system), "mrt_LiveBoard"+system)
		if err != nil || !comp {
			log.Printf("[MRT_ETA] action=mrt_eta system=%s event=skip reason=api_error", system)
			return
		}
		if _, err := dec.Token(); err != nil {
			flipopen()
			log.Printf("[MRT_ETA] action=mrt_eta system=%s event=decode_error error=%v", system, err)
			return
		}
		func() {
			defer flipopen()
			pipe := rc.Pipeline()
			for dec.More() {
				var temp mrtLive
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
	log.Printf("[MRT_ETA] action=mrt_eta event=complete")
}
