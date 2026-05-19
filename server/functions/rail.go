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

type rail_Station struct {
	StationID   string `json:"StationID"`
	StationName struct {
		ZhTw string `json:"Zh_tw"`
	} `json:"StationName"`
	LocationCity    string `json:"LocationCity"`
	StationPosition struct {
		PositionLon float64 `json:"PositionLon"`
		PositionLat float64 `json:"PositionLat"`
	} `json:"StationPosition"`
	StationCode string `json:"StationID"`
}
type rail_Fare struct {
	OriginStationID      string `json:"OriginStationID"`
	DestinationStationID string `json:"DestinationStationID"`
	Fares                []struct {
		TicketType int32 `json:"TicketType"`
		FareClass  int32 `json:"FareClass"`
		CabinClass int32 `json:"CabinClass"`
		Price      int32 `json:"Price"`
	} `json:"Fares"`
}
type rail_DailyTrain struct {
	TrainDate      string `json:"TrainDate"`
	DailyTrainInfo struct {
		TrainNo           string `json:"TrainNo"`
		Direction         bool   `json:"Direction"`
		StartingStationID string `json:"StartingStationID"`
		EndingStationID   string `json:"EndingStationID"`
		TrainTypeID       string `json:"TrainTypeID"`
		TrainTypeCode     string `json:"TrainTypeCode"`
		TrainTypeName     struct {
			ZhTw string `json:"Zh_tw"`
		} `json:"TrainTypeName"`
		TripLine           uint8 `json:"TripLine"`
		WheelchairFlag     bool  `json:"WheelchairFlag"`
		PackageServiceFlag bool  `json:"PackageServiceFlag"`
		DiningFlag         bool  `json:"DiningFlag"`
		BikeFlag           bool  `json:"BikeFlag"`
		BreastFeedingFlag  bool  `json:"BreastFeedingFlag"`
		DailyFlag          bool  `json:"DailyFlag"`
		ServiceAddedFlag   bool  `json:"ServiceAddedFlag"`
		SuspendedFlag      bool  `json:"SuspendedFlag"`
		Note               struct {
			ZhTw string `json:"Zh_tw"`
		} `json:"Note"`
	} `json:"DailyTrainInfo"`
	StopTimes []struct {
		StopSequence uint8  `json:"StopSequence"`
		StationID    string `json:"StationID"`
		StationName  struct {
			ZhTw string `json:"Zh_tw"`
		} `json:"StationName"`
		ArrivalTime   string `json:"ArrivalTime"`
		DepartureTime string `json:"DepartureTime"`
		SuspendedFlag bool   `json:"SuspendedFlag"`
	} `json:"StopTimes"`
}
type Tra_Delay struct {
	TrainNo     string `json:"TrainNo"`
	StationID   string `json:"StationID"`
	StationName struct {
		ZhTw string `json:"Zh_tw"`
	} `json:"StationName"`
	DelayTime     uint16 `json:"DelayTime"`
	SrcUpdateTime string `json:"SrcUpdateTime"`
}
type Tra_LiveBoard struct {
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

type thsr_availableseatstatuslist struct {
	AvailableSeats []struct {
		TrainNo           string `json:"TrainNo"`
		Direction         bool   `json:"Direction"`
		StationID         string `json:"StationID"`
		DepartureTime     string `json:"DepartureTime"`
		EndingStationID   string `json:"EndingStationID"`
		EndingStationName struct {
			ZhTw string `json:"Zh_tw"`
		} `json:"EndingStationName"`
		StopStations []struct {
			StationID   string `json:"StationID"`
			StationName struct {
				ZhTw string `json:"Zh_tw"`
			}
			StandardSeatStatus string `json:"StandardSeatStatus"`
			BusinessSeatStatus string `json:"BusinessSeatStatus"`
		} `json:"StopStations"`
	} `json:"AvailableSeats"`
}

func rail_static(client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	tra_station(client, rc, db)
	thsr_station(client, rc, db)
	tra_fare(client, rc, db)
}
func tra_station(client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	dec, comp, err, flipopen := call_api(client, rc, "/v2/Rail/TRA/Station", "tra_stations")
	if err != nil || !comp {
		return
	}
	defer flipopen()
	if _, err := dec.Token(); err != nil {
		return
	}
	c1 := `INSERT INTO tra_stations (
                          station_id,
                          name,
                          city,
                          geom,
                          updated_at
                          )
			VALUES ($1, $2, $3,ST_GeomFromText($4, 4326), NOW())
			ON CONFLICT (station_id) DO UPDATE SET name = EXCLUDED.name, updated_at = NOW();`
	batch := &pgx.Batch{}
	for dec.More() {
		var temp rail_Station
		if err := dec.Decode(&temp); err == nil {
			g := fmt.Sprintf("POINT(%.6f %.6f)", temp.StationPosition.PositionLon, temp.StationPosition.PositionLat)
			batch.Queue(c1, temp.StationID, temp.StationName.ZhTw, temp.LocationCity, g)
		}
	}
	b := db.SendBatch(ctx, batch)
	_ = b.Close()
	log.Println("updated tra_stations")
}
func thsr_station(client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	dec, comp, err, flipopen := call_api(client, rc, "/v2/Rail/THSR/Station", "thsr_stations")
	if err != nil || !comp {
		return
	}
	defer flipopen()
	if _, err := dec.Token(); err != nil {
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
		var temp rail_Station
		if err := dec.Decode(&temp); err == nil {
			g := fmt.Sprintf("POINT(%.6f %.6f)", temp.StationPosition.PositionLon, temp.StationPosition.PositionLat)
			batch.Queue(c1, temp.StationID, temp.StationName.ZhTw, temp.LocationCity, g, temp.StationCode)
		}
	}
	b := db.SendBatch(ctx, batch)
	_ = b.Close()
	log.Println("updated thsr_stations")
}
func tra_fare(client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	dec, comp, err, flipopen := call_api(client, rc, "/v2/Rail/TRA/ODFare", "tra_fare")
	if err != nil || !comp {
		return
	}
	defer flipopen()
	if _, err := dec.Token(); err != nil {
		return
	}
	var rows [][]interface{}
	for dec.More() {
		var temp rail_Fare
		if err := dec.Decode(&temp); err == nil {
			for _, t1 := range temp.Fares {
				rows = append(rows, []interface{}{temp.OriginStationID, temp.DestinationStationID, t1.TicketType, t1.Price})
			}
		}
	}
	b, err := db.Begin(ctx)
	if err != nil {
		return
	}
	defer b.Rollback(ctx)
	b.Exec(ctx, `TRUNCATE TABLE tra_fares;`)
	_, err = b.CopyFrom(ctx, pgx.Identifier{"tra_fares"}, []string{"origin_station_id", "destination_station_id", "ticket_type", "price"}, pgx.CopyFromRows(rows))
	if err == nil {
		_ = b.Commit(ctx)
		log.Println("updated tra_fares")
	}
}

func tra_eta(client *resty.Client, rc *redis.Client) {
	dec, comp, err, flipopen := call_api(client, rc, "/v2/Rail/TRA/LiveTrainDelay", "tra_delay")
	if err == nil && comp {
		func() {
			defer flipopen()
			if _, err := dec.Token(); err == nil {
				pipe := rc.Pipeline()
				for dec.More() {
					var temp Tra_Delay
					if err := dec.Decode(&temp); err == nil {
						pipe.HSet("tra:delay", temp.TrainNo, temp.DelayTime)
					}
				}
				_, _ = pipe.Exec()
			}
		}()
	}
	dec, comp, err, flipopen = call_api(client, rc, "/v2/Rail/TRA/LiveBoard", "tra_liveboard")
	func() {
		defer flipopen()
		if err != nil && !comp {
			return
		}
		if _, err := dec.Token(); err != nil {
			return
		}
		delays, _ := rc.HGetAll("tra:delay").Result()
		res := make(map[string][]*models.Tra_LiveBoard)
		for dec.More() {
			var temp Tra_LiveBoard
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
	log.Println("updated tra eta updated")
}
