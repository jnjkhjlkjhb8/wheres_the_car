package main

import (
	"context"

	"github.com/jnjkhjlkjhb8/wheres_the_car/models"

	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis"
	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"
)

type railStation struct {
	StationID   string `json:"StationID"`
	StationName struct {
		ZhTw string `json:"Zh_tw"`
	} `json:"StationName"`
	LocationCityCode string `json:"LocationCityCode"`
	StationPosition  struct {
		PositionLon float64 `json:"PositionLon"`
		PositionLat float64 `json:"PositionLat"`
	} `json:"StationPosition"`
	StationCode string `json:"StationCode"`
}
type tra_fare struct {
	OriginStationID      string `json:"OriginStationID"`
	DestinationStationID string `json:"DestinationStationID"`
	Fares                []struct {
		TicketType string `json:"TicketType"`
		Price      int32  `json:"Price"`
	} `json:"Fares"`
}

type thsr_fare struct {
	OriginStationID      string `json:"OriginStationID"`
	DestinationStationID string `json:"DestinationStationID"`
	Fares                []struct {
		TicketType uint8  `json:"TicketType"`
		FareClass  uint8  `json:"FareClass"`
		CabinClass uint8  `json:"CabinClass"`
		Price      uint16 `json:"Price"`
	} `json:"Fares"`
}

type TraDelay struct {
	TrainNo     string `json:"TrainNo"`
	StationID   string `json:"StationID"`
	StationName struct {
		ZhTw string `json:"Zh_tw"`
	} `json:"StationName"`
	DelayTime     uint16 `json:"DelayTime"`
	SrcUpdateTime string `json:"SrcUpdateTime"`
}
type TraLiveboard struct {
	StationID   string `json:"StationID"`
	StationName struct {
		ZhTw string `json:"Zh_tw"`
	} `json:"StationName"`
	TrainNo       string `json:"TrainNo"`
	Direction     bool   `json:"Direction"`
	TrainTypeID   string `json:"TrainTypeID"`
	TrainTypeCode string `json:"TrainTypeCode"`
	TrainTypeName struct {
		ZhTw string `json:"Zh_tw"`
	} `json:"TrainTypeName"`
	TripLine          uint8  `json:"TripLine"`
	EndingStationID   string `json:"EndingStationID"`
	EndingStationName struct {
		ZhTw string `json:"Zh_tw"`
	} `json:"EndingStationName"`
	ScheduledArrivalTime   string `json:"ScheduledArrivalTime"`
	ScheduledDepartureTime string `json:"ScheduledDepartureTime"`
	DelayTime              uint16 `json:"DelayTime"`
	SrcUpdateTime          string `json:"SrcUpdateTime"`
}

func railStatic(ctx context.Context, client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	log.Printf("[RAIL] action=rail_static event=start")
	traStation(ctx, client, rc, db)
	thsrStation(ctx, client, rc, db)
	traFare(ctx, client, rc, db)
	thsrFare(ctx, client, rc, db)
	log.Printf("[RAIL] action=rail_static event=complete")
}
func traStation(ctx context.Context, client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	log.Printf("[RAIL] action=tra_station event=start")
	dec, comp, err, flipopen := callApi(client, rc, "/v2/Rail/TRA/Station", "tra_stations")
	if err != nil || !comp {
		log.Printf("[RAIL] action=tra_station event=skip reason=api_error")
		return
	}
	defer flipopen()
	if _, err := dec.Token(); err != nil {
		log.Printf("[RAIL] action=tra_station event=decode_error error=%v", err)
		return
	}
	var row [][]interface{}
	for dec.More() {
		var temp railStation
		if err := dec.Decode(&temp); err == nil {
			g := fmt.Sprintf("POINT(%.6f %.6f)", temp.StationPosition.PositionLon, temp.StationPosition.PositionLat)
			row = append(row, []interface{}{
				temp.StationID,
				temp.StationName.ZhTw,
				citymap2[temp.LocationCityCode],
				g,
			})
		}
	}
	if len(row) > 0 {
		c1 := `CREATE TEMP TABLE temp_tra (
					station_id text,
					name text,
					city text,
					geom text
				) ON COMMIT DROP;`
		c2 := `INSERT INTO tra_stations (
					station_id,
					name,
					city,
					geom,
					updated_at
				)
				SELECT station_id, name, city, ST_GeomFromText(geom, 4326),NOW()
				FROM temp_tra
				ON CONFLICT (station_id) DO UPDATE SET name = EXCLUDED.name, city = EXCLUDED.city, geom = EXCLUDED.geom,updated_at = NOW();`
		b, err := db.Begin(ctx)
		if err != nil {
			log.Printf("[RAIL] action=tra_station event=tx_error error=%v", err)
			return
		}
		defer func(b pgx.Tx, ctx context.Context) {
			_ = b.Rollback(ctx)
		}(b, ctx)
		if _, err := b.Exec(ctx, c1); err != nil {
			log.Printf("[RAIL] action=tra_station event=create_temp_error error=%v", err)
			return
		}
		_, err = b.CopyFrom(ctx, pgx.Identifier{"temp_tra"}, []string{"station_id", "name", "city", "geom"}, pgx.CopyFromRows(row))
		if err == nil {
			if _, err = b.Exec(ctx, c2); err != nil {
				log.Printf("[RAIL] action=tra_station event=exec_error error=%v", err)
				return
			}
			if err = b.Commit(ctx); err != nil {
				log.Printf("[RAIL] action=tra_station event=commit_error error=%v", err)
			} else {
				log.Printf("[RAIL] action=tra_station event=success station_count=%d", len(row))
			}
		} else {
			log.Printf("[RAIL] action=tra_station event=copyfrom_error error=%v", err)
		}
	} else {
		log.Printf("[RAIL] action=tra_station event=complete reason=no_data")
	}
	log.Printf("[RAIL] action=tra_station event=complete")
}
func thsrStation(ctx context.Context, client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	log.Printf("[RAIL] action=thsr_station event=start")
	dec, comp, err, flipopen := callApi(client, rc, "/v2/Rail/THSR/Station", "thsr_stations")
	if err != nil || !comp {
		log.Printf("[RAIL] action=thsr_station event=skip reason=api_error")
		return
	}
	defer flipopen()
	if _, err := dec.Token(); err != nil {
		log.Printf("[RAIL] action=thsr_station event=decode_error error=%v", err)
		return
	}
	c1 := `INSERT INTO thsr_stations (
                          station_id,
                          name,
                          city,
                          geom,
                          stationcode,
                          updated_at
                          )
			VALUES ($1, $2, $3,ST_GeomFromText($4, 4326),$5, NOW())
			ON CONFLICT (station_id) DO UPDATE SET name = EXCLUDED.name, updated_at = NOW();`
	batch := &pgx.Batch{}
	for dec.More() {
		var temp railStation
		if err := dec.Decode(&temp); err == nil {
			g := fmt.Sprintf("POINT(%.6f %.6f)", temp.StationPosition.PositionLon, temp.StationPosition.PositionLat)
			batch.Queue(c1, temp.StationID, temp.StationName.ZhTw, citymap2[temp.LocationCityCode], g, temp.StationCode)
		}
	}
	b := db.SendBatch(ctx, batch)
	_ = b.Close()
	log.Printf("[RAIL] action=thsr_station event=complete")
}
func traFare(ctx context.Context, client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	log.Printf("[RAIL] action=tra_fare event=start")
	dec, comp, err, flipopen := callApi(client, rc, "/v2/Rail/TRA/ODFare", "tra_fare")
	if err != nil || !comp {
		log.Printf("[RAIL] action=tra_fare event=skip reason=api_error | noupdate")
		return
	}
	defer flipopen()
	if _, err := dec.Token(); err != nil {
		log.Printf("[RAIL] action=tra_fare event=decode_error error=%v", err)
		return
	}
	var row [][]interface{}
	for dec.More() {
		var temp tra_fare
		if err := dec.Decode(&temp); err == nil {
			for _, t1 := range temp.Fares {
				row = append(row, []interface{}{temp.OriginStationID, temp.DestinationStationID, t1.TicketType, t1.Price})
			}
		}
	}
	c1 := `CREATE TEMP TABLE temp_tra_fare (
				origin_station_id text,
				destination_station_id text,
				ticket_type text,
				price int
			) ON COMMIT DROP;`
	c2 := `INSERT INTO tra_fares (
				origin_station_id,
				destination_station_id,
				ticket_type,
				price
			)
			SELECT origin_station_id, destination_station_id, ticket_type, price FROM temp_tra_fare
			ON CONFLICT (origin_station_id, destination_station_id, ticket_type) 
			DO UPDATE SET price = EXCLUDED.price`
	b, err := db.Begin(ctx)
	if err != nil {
		log.Printf("[RAIL] action=tra_fare event=begin_error error=%v", err)
		return
	}
	defer func(b pgx.Tx, ctx context.Context) {
		_ = b.Rollback(ctx)
	}(b, ctx)
	_, err = b.Exec(ctx, c1)
	if err != nil {
		log.Printf("[RAIL] action=tra_fare event=execute_error error=%v", err)
		return
	}
	_, err = b.CopyFrom(ctx, pgx.Identifier{"temp_tra_fare"}, []string{"origin_station_id", "destination_station_id", "ticket_type", "price"}, pgx.CopyFromRows(row))
	if err == nil {
		if _, execErr := b.Exec(ctx, c2); execErr != nil {
			log.Printf("[RAIL] action=tra_fare event=exec_error error=%v", execErr)
			return
		}
		if commitErr := b.Commit(ctx); commitErr != nil {
			log.Printf("[RAIL] action=tra_fare event=commit_error error=%v", commitErr)
		} else {
			log.Printf("[RAIL] action=tra_fare event=complete row_count=%d", len(row))
		}
	} else {
		log.Printf("[RAIL] action=tra_fare event=copyfrom_error error=%v", err)
	}
}
func thsrFare(ctx context.Context, client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	log.Printf("[RAIL] action=thsr_fare event=start")
	dec, comp, err, flipopen := callApi(client, rc, "/v2/Rail/THSR/ODFare", "thsr_fare")
	if err != nil || !comp {
		log.Printf("[RAIL] action=thsr_fare event=skip reason=api_error | noupdate")
		return
	}
	defer flipopen()
	if _, err := dec.Token(); err != nil {
		log.Printf("[RAIL] action=thsr_fare event=decode_error error=%v", err)
		return
	}
	var row [][]interface{}
	for dec.More() {
		var temp thsr_fare
		if err := dec.Decode(&temp); err == nil {
			for _, t1 := range temp.Fares {
				row = append(row, []interface{}{temp.OriginStationID, temp.DestinationStationID, t1.TicketType, t1.FareClass, t1.CabinClass, t1.Price})
			}
		}
	}
	c1 := `CREATE TEMP TABLE temp_thsr (
				origin_station_id text,
				destination_station_id text,
				ticket_type smallint,
				fare_class smallint,
				cabin_class smallint,
				price int
			) ON COMMIT DROP;`
	c2 := `INSERT INTO thsr_fares (
				origin_station_id,
				destination_station_id,
				ticket_type,
				fare_class,
				cabin_class,
				price
			)
			SELECT origin_station_id, destination_station_id, ticket_type, fare_class,cabin_class,price FROM temp_thsr
			ON CONFLICT (origin_station_id, destination_station_id, ticket_type, fare_class, cabin_class) 
			DO UPDATE SET price = EXCLUDED.price`
	b, err := db.Begin(ctx)
	if err != nil {
		log.Printf("[RAIL] action=thsr_fare event=begin_error error=%v", err)
		return
	}
	defer func(b pgx.Tx, ctx context.Context) {
		_ = b.Rollback(ctx)
	}(b, ctx)
	_, err = b.Exec(ctx, c1)
	if err != nil {
		log.Printf("[RAIL] action=thsr_fare event=execute_error error=%v", err)
		return
	}
	_, err = b.CopyFrom(ctx, pgx.Identifier{"temp_thsr"}, []string{"origin_station_id", "destination_station_id", "ticket_type", "fare_class", "cabin_class", "price"}, pgx.CopyFromRows(row))
	if err == nil {
		if _, execErr := b.Exec(ctx, c2); execErr != nil {
			log.Printf("[RAIL] action=thsr_fare event=exec_error error=%v", execErr)
			return
		}
		if commitErr := b.Commit(ctx); commitErr != nil {
			log.Printf("[RAIL] action=thsr_fare event=commit_error error=%v", commitErr)
		} else {
			log.Printf("[RAIL] action=thsr_fare event=complete row_count=%d", len(row))
		}
	} else {
		log.Printf("[RAIL] action=thsr_fare event=copyfrom_error error=%v", err)
	}
}

func traEta(client *resty.Client, rc *redis.Client) {
	log.Printf("[TRA_ETA] action=tra_eta event=start")
	dec, comp, err, flipopen := callApi(client, rc, "/v2/Rail/TRA/LiveTrainDelay", "tra_delay")
	if err == nil && comp {
		func() {
			defer flipopen()
			if _, err := dec.Token(); err == nil {
				data := &models.TraDelays{
					Delay: make(map[string]int32),
				}
				count := 0
				pipe := rc.Pipeline()
				for dec.More() {
					var temp TraDelay
					if err := dec.Decode(&temp); err == nil {
						count++
						data.Delay[temp.TrainNo] = int32(temp.DelayTime)
						pipe.HSet("tra:delay", temp.TrainNo, temp.DelayTime)
					}
				}
				bytes, _ := proto.Marshal(data)
				pipe.Set("tra:delay_all", bytes, 3*time.Minute)
				pipe.Expire("tra:delay", 3*time.Minute)
				_, pipErr := pipe.Exec()
				if pipErr != nil {
					log.Printf("[TRA_ETA] action=tra_eta event=delay_redis_error error=%v", pipErr)
				} else {
					log.Printf("[TRA_ETA] action=tra_eta event=delay_redis_success count=%d", count)
				}
			}
		}()
	} else {
		log.Printf("[TRA_ETA] action=tra_eta event=skip_delay reason=api_error")
	}
	dec, comp, err, flipopen = callApi(client, rc, "/v2/Rail/TRA/LiveBoard", "tra_liveboard")
	func() {
		if flipopen != nil {
			defer flipopen()
		}
		if err != nil || !comp {
			return
		}
		if _, err := dec.Token(); err != nil {
			return
		}
		delays, _ := rc.HGetAll("tra:delay").Result()
		res := make(map[string][]*models.Tra_LiveBoard)
		for dec.More() {
			var temp TraLiveboard
			if err := dec.Decode(&temp); err == nil {
				if delay, ok := delays[temp.TrainNo]; ok {
					var inter uint16
					if _, err := fmt.Sscanf(delay, "%d", &inter); err == nil {
						temp.DelayTime = inter
					}
				}
				pb := &models.Tra_LiveBoard{
					TrainNo:                temp.TrainNo,
					Direction:              temp.Direction,
					TrainTypeId:            temp.TrainTypeID,
					TrainTypeCode:          temp.TrainTypeCode,
					TrainTypeName:          temp.TrainTypeName.ZhTw,
					EndingStationId:        temp.EndingStationID,
					EndingStationName:      temp.EndingStationName.ZhTw,
					ScheduledArrivalTime:   temp.ScheduledArrivalTime,
					ScheduledDepartureTime: temp.ScheduledDepartureTime,
					Delay:                  int32(temp.DelayTime),
					TripLine:               int32(temp.TripLine),
				}
				res[temp.StationID] = append(res[temp.StationID], pb)
			}
		}
		pipe := rc.Pipeline()
		for a, b := range res {
			pb := &models.Tra_LiveBoards{
				StationId: a,
				Items:     b,
			}
			pbs, err := proto.Marshal(pb)
			if err != nil {
				continue
			}
			pipe.Set(fmt.Sprintf("tra:liveboard:%s", a), pbs, 3*time.Minute)
		}
		_, err = pipe.Exec()
	}()
	log.Printf("[TRA_ETA] action=tra_eta event=complete")
}
