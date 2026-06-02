package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jnjkhjlkjhb8/wheres_the_car/models"

	"github.com/go-redis/redis"
	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"
)

type bikeStation struct {
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
type bikeAvailability struct {
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

func bikeStatic(ctx context.Context, client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	log.Printf("[BIKE] action=bike_static event=start")
	getbikeStation(ctx, client, rc, db)
	log.Printf("[BIKE] action=bike_static event=complete")
}
func getbikeStation(ctx context.Context, client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	log.Printf("[BIKE] action=getbike_station event=start")
	for _, city := range cities {
		if city == "Keelung" || city == "HsinchuCounty" || city == "NantouCounty" || city == "YilanCounty" || city == "PenghuCounty" || city == "KinmenCounty" || city == "LienchiangCounty" || city == "InterCity" || city == "HualienCounty" {
			continue
		}
		log.Printf("[BIKE] action=getbike_station city=%s event=city_start", city)
		dec, comp, err, flipopen := callApi(client, rc, fmt.Sprintf("/v2/Bike/Station/City/%s", city), "bike_stations"+city)
		func() {
			if flipopen != nil {
				defer flipopen()
			}
			if err != nil || !comp {
				log.Printf("[BIKE] action=getbike_station city=%s event=skip reason=api_error=%s", city, err)
				return
			}
			if _, err := dec.Token(); err != nil {
				log.Printf("[BIKE] action=getbike_station city=%s event=decode_error error=%v", city, err.Error())
				return
			}
			var row [][]interface{}
			for dec.More() {
				var temp bikeStation
				if err := dec.Decode(&temp); err == nil {
					g := fmt.Sprintf("POINT(%.6f %.6f)", temp.StationPosition.PositionLon, temp.StationPosition.PositionLat)
					row = append(row, []interface{}{
						temp.StationUID,
						temp.StationID,
						strings.TrimPrefix(temp.StationName.ZhTw, "YouBike2.0_"),
						temp.BikesCapacity,
						temp.ServiceType,
						city,
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
							city text,
                            geom text,
                            addr text
					) ON COMMIT DROP`
				c2 := `INSERT INTO bike_stations (
                           station_uid,
                           station_id,
                           name,
                           capacity,
                           service_type,
						   city,
                           geom,
                           address,
                           updated_at
					)
					SELECT uid, id, name, cap, type, city,st_geomfromtext(geom, 4326) AS temp, addr AS address,NOW() FROM temp_bike 
					ON CONFLICT (station_uid) DO UPDATE SET name = EXCLUDED.name,capacity = EXCLUDED.capacity,service_type = EXCLUDED.service_type,city = excluded.city,geom = EXCLUDED.geom,address = EXCLUDED.address,updated_at = NOW();`
				b, err := db.Begin(ctx)
				if err != nil {
					log.Println(err.Error())
					return
				}
				_, _ = b.Exec(ctx, c1)
				_, err = b.CopyFrom(ctx, pgx.Identifier{"temp_bike"}, []string{"uid", "id", "name", "cap", "type", "city", "geom", "addr"}, pgx.CopyFromRows(row))
				if err == nil {
					if _, execErr := b.Exec(ctx, c2); execErr != nil {
						log.Printf("[BIKE] action=getbike_station city=%s event=exec_error error=%v", city, execErr)
					}
					if commitErr := b.Commit(ctx); commitErr != nil {
						log.Printf("[BIKE] action=getbike_station city=%s event=commit_error error=%v", city, commitErr)
					} else {
						log.Printf("[BIKE] action=getbike_station city=%s event=success station_count=%d", city, len(row))
					}
				} else {
					log.Printf("[BIKE] action=getbike_station city=%s event=copyfrom_error error=%v", city, err)
					_ = b.Rollback(ctx)
				}
			}
		}()
	}
	log.Printf("[BIKE] action=getbike_station event=complete")
}
func bikeEta(client *resty.Client, rc *redis.Client) {
	log.Printf("[BIKE_ETA] action=bike_eta event=start")
	for _, city := range cities {
		if city == "Keelung" || city == "HsinchuCounty" || city == "NantouCounty" || city == "YilanCounty" || city == "PenghuCounty" || city == "KinmenCounty" || city == "LienchiangCounty" || city == "InterCity" || city == "HualienCounty" {
			continue
		}
		log.Printf("[BIKE_ETA] action=bike_eta city=%s event=city_start", city)
		dec, comp, err, flipopen := callApi(client, rc, fmt.Sprintf("/v2/Bike/Availability/City/%s", city), "bike_availability"+city)
		if !comp {
			log.Printf("[BIKE_ETA] action=bike_eta city=%s event=skip reason=no updated", city)
			continue
		}
		if err != nil {
			log.Printf("[BIKE_ETA] action=bike_eta city=%s event=skip reason=api_error,error=%s", city, err)
			continue
		}
		if _, err := dec.Token(); err != nil {
			log.Printf("[BIKE_ETA] action=bike_eta city=%s event=decode_error error=%v", city, err)
			continue
		}
		func() {
			defer flipopen()
			pipe := rc.Pipeline()
			for dec.More() {
				var temp bikeAvailability
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
			log.Printf("[BIKE_ETA] action= %s bike_eta event=complete", city)
		}()
	}
	log.Printf("[BIKE_ETA] action=bike_eta event=complete")
}
