package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jnjkhjlkjhb8/bus/models"

	"github.com/go-redis/redis"
	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"
)

type bike_station struct {
	StationUID  string `json:"StationUID"`
	StationID   string `json:"StationID"`
	StationName struct {
		ZhTw string `json:"Zh_tw"`
	} `json:"StationName"`
	StationPosition struct {
		PositionLon float64 `json:"PositionLon"`
		PositionLat float64 `json:"PositionLat"`
	} `json:"StationPosition"`
	StationAddress struct {
		ZhTw string `json:"Zh_tw"`
	} `json:"StationAddress"`
	BikesCapacity uint8 `json:"BikesCapacity"`
	ServiceType   uint8 `json:"ServiceType"`
}
type bike_availability struct {
	StationUID               string `json:"StationUID"`
	StationID                string `json:"StationID"`
	ServiceStatus            uint8  `json:"ServiceStatus"`
	ServiceType              uint8  `json:"ServiceType"`
	AvailableReturnBikes     uint8  `json:"AvailableReturnBikes"`
	AvailableRentBikesDetail struct {
		GeneralBikes  uint8 `json:"GeneralBikes"`
		ElectricBikes uint8 `json:"ElectricBikes"`
	} `json:"AvailableRentBikesDetail"`
}

func bike_static(client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	getbike_station(client, rc, db)
}
func getbike_station(client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	for _, city := range cities {
		if city == "Keelung" || city == "HsinchuCounty" || city == "NantouCounty" || city == "YilanCounty" || city == "PenghuCounty" || city == "KinmenCounty" || city == "LienchiangCounty" || city == "InterCity" {
			continue
		}
		dec, comp, err, flipopen := call_api(client, rc, fmt.Sprintf("/v2/Bike/Station/City/%s", city), "bike_stations")
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
				var temp bike_station
				if err := dec.Decode(&temp); err == nil {
					g := fmt.Sprintf("POINT(%.6f %.6f)", temp.StationPosition.PositionLon, temp.StationPosition.PositionLat)
					row = append(row, []interface{}{
						temp.StationUID,
						temp.StationID,
						strings.TrimPrefix(temp.StationName.ZhTw, "YouBike2.0_"),
						temp.BikesCapacity,
						temp.ServiceType,
						g,
						temp.StationAddress.ZhTw,
					})
				}
			}
			if len(row) > 0 {
				c1 := `CREATE TEMP TABLE temp_bike (
                            uid text,
                            id text,
                            name text,
                            cap int,
                            type int,
                            geom text,
                            addr text
					) ON COMMIT DROP`
				c2 := `INSERT INTO bike_stations (
                           station_uid,
                           station_id,
                           name,
                           capacity,
                           service_type,
                           geom,
                           address,
                           updated_at
					)
					SELECT uid, id, name, cap, type, st_geomfromtext(geom, 4326), address,NOW() FROM temp_bike 
					ON CONFLICT (station_uid) DO UPDATE SET name = EXCLUDED.name,capacity = EXCLUDED.capacity,service_type = EXCLUDED.service_type,geom = EXCLUDED.geom,updated_at = NOW();`
				b, err := db.Begin(ctx)
				if err != nil {
					log.Println(err.Error())
					return
				}
				_, _ = b.Exec(ctx, c1)
				_, err = b.CopyFrom(ctx, pgx.Identifier{"temp_bike"}, []string{"uid", "id", "name", "cap", "type", "geom", "addr"}, pgx.CopyFromRows(row))
				if err == nil {
					b.Exec(ctx, c2)
					b.Commit(ctx)
					log.Println("bike station updated_" + city)
				} else {
					log.Println(err.Error())
					_ = b.Rollback(ctx)
				}
			}
		}()
	}
	log.Println("bike station updated")
}
func bike_eta(client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	for _, city := range cities {
		if city == "Keelung" || city == "HsinchuCounty" || city == "NantouCounty" || city == "YilanCounty" || city == "PenghuCounty" || city == "KinmenCounty" || city == "LienchiangCounty" || city == "InterCity" {
			continue
		}
		dec, comp, err, flipopen := call_api(client, rc, fmt.Sprintf("/v2/Bike/Availability/City/%s", city), "bike_availability")
		if err != nil || !comp {
			continue
		}
		if _, err := dec.Token(); err != nil {
			continue
		}
		func() {
			defer flipopen()
			pipe := rc.Pipeline()
			for dec.More() {
				var temp bike_availability
				if err := dec.Decode(&temp); err == nil {
					raw := &models.BikeEta{
						StationUID:           temp.StationUID,
						ServiceStatus:        int32(temp.ServiceStatus),
						ServiceType:          int32(temp.ServiceType),
						AvailableReturnBikes: int32(temp.AvailableReturnBikes),
						GeneralBikes:         int32(temp.AvailableRentBikesDetail.GeneralBikes),
						ElectricBikes:        int32(temp.AvailableRentBikesDetail.ElectricBikes),
					}
					pb, err := proto.Marshal(raw)
					if err != nil {
						continue
					}
					pipe.Set(fmt.Sprintf("bike_availability:%s", temp.StationUID), pb, 2*time.Minute)
				}
			}
			_, _ = pipe.Exec()
		}()
	}
	log.Println("bike eta done")
}
