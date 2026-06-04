package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis"
	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jnjkhjlkjhb8/wheres_the_car/models"
	"google.golang.org/protobuf/proto"
)

type Tratimetable struct {
	Train_date            time.Time `db:"train_date"`
	Trainno               string    `db:"trainno"`
	Starting_station_id   string    `db:"starting_station_id"`
	Starting_station_name string    `db:"starting_station_name"`
	Ending_station_id     string    `db:"ending_station_id"`
	Ending_station_name   string    `db:"ending_station_name"`
	Train_type_id         string    `db:"train_type_id"`
	Train_type_code       string    `db:"train_type_code"`
	Train_type_name       string    `db:"train_type_name"`
	Tripline              int32     `db:"tripline"`
	Stopsequence          int       `db:"stopsequence"`
	Stationid             string    `db:"stationid"`
	Stationname           string    `db:"stationname"`
	Arrivaltime           time.Time `db:"arrivaltime"`
	Departuretime         time.Time `db:"departuretime"`
	Mask                  int32     `db:"mask"`
	Note                  string    `db:"note"`
}
type Thsrtimetable struct {
	train_date            string `db:"train_date"`
	trainno               string `db:"trainno"`
	Starting_station_id   string `db:"starting_station_id"`
	Starting_station_name string `db:"starting_station_name"`
	Ending_station_id     string `db:"ending_station_id"`
	Ending_station_name   string `db:"ending_station_name"`
	Stopsequence          int    `db:"stop_sequence"`
	Stationid             string `db:"stationid"`
	Stationname           string `db:"stationname"`
	Arrivaltime           string `db:"arrival_time"`
	Departuretime         string `db:"departure_time"`
	Note                  string `db:"note"`
	Overnight             bool   `db:"overnight"`
}
type TraStops struct {
	Stopsequence  int    `db:"stop_sequence"`
	Stationid     string `db:"stationid"`
	Stationname   string `db:"stationname"`
	Arrivaltime   string `db:"arrival_time"`
	Departuretime string `db:"departure_time"`
	Mask          int32  `db:"mask"`
}
type ThsrStops struct {
	Stopsequence  int    `db:"stop_sequence"`
	Stationid     string `db:"stationid"`
	Stationname   string `db:"stationname"`
	Arrivaltime   string `db:"arrival_time"`
	Departuretime string `db:"departure_time"`
}
type trafare struct {
	origin_station_id      string `db:"origin_station_id"`
	destination_station_id string `db:"destination_station_id"`
	ticket_id              string `db:"ticket_id"`
	price                  int32  `db:"price"`
}
type thsrfare struct {
	origin_station_id      string `db:"origin_station_id"`
	destination_station_id string `db:"destination_station_id"`
	ticket_type            uint8  `db:"ticket_type"`
	fare_class             uint8  `db:"fare_class"`
	cabin_class            uint8  `db:"cabin_class"`
	price                  int32  `db:"price"`
}
type raw_trafare struct {
	OriginStationID      string `json:"OriginStationID"`
	DestinationStationID string `json:"DestinationStationID"`
	TicketType           string `json:"TicketType"`
	Price                int32  `json:"Price"`
}
type raw_thsrfare struct {
	OriginStationID      string `json:"OriginStationID"`
	DestinationStationID string `json:"DestinationStationID"`
	TicketType           uint8  `json:"TicketType"`
	FareClass            uint8  `json:"FareClass"`
	CabinClass           uint8  `json:"CabinClass"`
	Price                int32  `json:"Price"`
}
type raw_tra_timetable struct {
	TrainDate      string `json:"TrainDate"`
	DailyTrainInfo struct {
		TrainNo             string `json:"TrainNo"`
		Direction           uint8  `json:"Direction"`
		StartingStationID   string `json:"StartingStationID"`
		StartingStationName struct {
			ZhTw string `json:"Zh_tw"`
		} `json:"StartingStationName"`
		EndingStationID   string `json:"EndingStationID"`
		EndingStationName struct {
			ZhTw string `json:"Zh_tw"`
		} `json:"EndingStationName"`
		TrainTypeID   string `json:"TrainTypeID"`
		TrainTypeCode string `json:"TrainTypeCode"`
		TrainTypeName struct {
			ZhTw string `json:"Zh_tw"`
		} `json:"TrainTypeName"`
		TripLine           uint8 `json:"TripLine"`
		WheelchairFlag     uint8 `json:"WheelchairFlag"`
		PackageServiceFlag uint8 `json:"PackageServiceFlag"`
		DiningFlag         uint8 `json:"DiningFlag"`
		BikeFlag           uint8 `json:"BikeFlag"`
		BreastFeedingFlag  uint8 `json:"BreastFeedingFlag"`
		DailyFlag          uint8 `json:"DailyFlag"`
		ServiceAddedFlag   uint8 `json:"ServiceAddedFlag"`
		SuspendedFlag      uint8 `json:"SuspendedFlag"`
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
		SuspendedFlag uint8  `json:"SuspendedFlag"`
	} `json:"StopTimes"`
}
type raw_thsr_timetable struct {
	TrainDate      string `json:"TrainDate"`
	DailyTrainInfo struct {
		TrainNo             string `json:"TrainNo"`
		Direction           uint8  `json:"Direction"`
		StartingStationID   string `json:"StartingStationID"`
		StartingStationName struct {
			ZhTw string `json:"Zh_tw"`
		} `json:"StartingStationName"`
		EndingStationID   string `json:"EndingStationID"`
		EndingStationName struct {
			ZhTw string `json:"Zh_tw"`
		} `json:"EndingStationName"`
		Note struct {
			ZhTw string `json:"Zh_tw"`
		} `json:"Note"`
		Overnight bool `json:"Overnight"`
	} `json:"DailyTrainInfo"`
	StopTimes []struct {
		StopSequence uint8  `json:"StopSequence"`
		StationID    string `json:"StationID"`
		StationName  struct {
			ZhTw string `json:"Zh_tw"`
		} `json:"StationName"`
		ArrivalTime   string `json:"ArrivalTime"`
		DepartureTime string `json:"DepartureTime"`
	} `json:"StopTimes"`
}

func tra_price(start, end string, ctx context.Context, client *resty.Client, db *pgxpool.Pool, rc *redis.Client) {
	arr := []*models.TraFareItem{}
	rows, _ := db.Query(ctx, `SELECT ticket_type,price FROM tra_fares WHERE origin_station_id = $1 AND destination_station_id = $2;`, start, end)
	gettrafares(ctx, client, rc, db)
	defer rows.Close()
	row, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[trafare])
	if err != nil {
		rows.Close()
		log.Println(err.Error())
	}
	for _, temp := range row {
		arr = append(arr, &models.TraFareItem{
			TicketType: temp.ticket_id,
			Price:      temp.price,
		})
	}
	bytes, _ := proto.Marshal(&models.TraFareItems{Items: arr})
	if err != nil {
		log.Printf("proto marshal error: %v", err)
	} else {
		err = rc.Set(fmt.Sprintf("TRA_Fare:%s:%s", start, end), bytes, 8*time.Hour).Err()
		if err != nil {
			log.Printf("redis set error: %v", err)
		}
	}
}
func thsr_price(start, end string, ctx context.Context, client *resty.Client, db *pgxpool.Pool, rc *redis.Client) {
	arr := []*models.ThsaFare{}
	getthsrfares(ctx, client, rc, db)
	rows, _ := db.Query(ctx, `SELECT ticket_type, fare_class, cabin_class, price FROM thsr_fares WHERE origin_station_id = $1 AND destination_station_id = $2;`, start, end)
	defer rows.Close()
	row, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[thsrfare])
	if err != nil {
		rows.Close()
		log.Println(err.Error())
	}
	for _, temp := range row {
		arr = append(arr, &models.ThsaFare{
			CabinClas:  int32(temp.cabin_class),
			TicketType: int32(temp.ticket_type),
			FareClass:  int32(temp.fare_class),
			Price:      temp.price,
		})
	}
	bytes, _ := proto.Marshal(&models.ThsaFares{Items: arr})
	if err != nil {
		log.Printf("proto marshal error: %v", err)
	} else {
		err = rc.Set(fmt.Sprintf("THSR_Fare:%s:%s", start, end), bytes, 1*time.Hour).Err()
		if err != nil {
			log.Printf("redis set error: %v", err)
		}
	}
}
func tra_stoptimes(trainno string, date time.Time, ctx context.Context, client *resty.Client, db *pgxpool.Pool, rc *redis.Client) {
	arr := []*models.TraStoptime{}
	rows, _ := db.Query(ctx, `SELECT stopsequence, stationid,stationname,arrivaltime,departuretime,mask FROM tra_timetable WHERE trainno = $1 AND updated_at >= $2 AND updated_at <= $2;`, trainno, date)
	if rows == nil {
		get_tra_timetable(ctx, db, client, rc, date.Format(time.DateOnly))
		rows, _ = db.Query(ctx, `SELECT stopsequence, stationid,stationname,arrivaltime,departuretime,mask FROM tra_timetable WHERE trainno = $1 AND updated_at >= $2 AND updated_at <= $2;`, trainno, date)
	}
	defer rows.Close()
	row, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[TraStops])
	if err != nil {
		rows.Close()
		log.Println(err.Error())
	}
	for _, temp := range row {
		arr = append(arr, &models.TraStoptime{
			StopSequence:  int32(temp.Stopsequence),
			StationId:     temp.Stationid,
			StationName:   temp.Stationname,
			ArrivalTime:   temp.Arrivaltime,
			DepartureTime: temp.Departuretime,
			SuspendedFlag: (temp.Mask & (1 << 7)) != 0,
		})
	}
	bytes, _ := proto.Marshal(&models.TraStoptimes{})
	if err != nil {
		log.Printf("proto marshal error: %v", err)
	} else {
		err = rc.Set(fmt.Sprintf("TRA_Stoptimes:%s:%s", date.Format(time.DateOnly), trainno), bytes, 1*time.Hour).Err()
		if err != nil {
			log.Printf("redis set error: %v", err)
		}
	}
}
func thsr_stoptimes(trainno string, date time.Time, ctx context.Context, client *resty.Client, db *pgxpool.Pool, rc *redis.Client) {
	arr := []*models.ThsaStoptime{}
	rows, _ := db.Query(ctx, `SELECT stopsequence, stationid,stationname,arrivaltime,departuretime FROM thsr_timetable WHERE trainno = $1 AND updated_at >= $2 AND updated_at <= $2;`, trainno, date)
	if rows == nil {
		get_thsr_timetable(ctx, db, client, rc, date.Format(time.DateOnly))
		rows, _ = db.Query(ctx, `SELECT stopsequence, stationid,stationname,arrivaltime,departuretime FROM thsr_timetable WHERE trainno = $1 AND updated_at >= $2 AND updated_at <= $2;`, trainno, date)
	}
	defer rows.Close()
	row, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[ThsrStops])
	if err != nil {
		rows.Close()
		log.Println(err.Error())
	}
	for _, temp := range row {
		arr = append(arr, &models.ThsaStoptime{
			StopSequence:  int32(temp.Stopsequence),
			StationId:     temp.Stationid,
			StationName:   temp.Stationname,
			ArrivalTime:   temp.Arrivaltime,
			DepartureTime: temp.Departuretime,
		})
	}
	bytes, _ := proto.Marshal(&models.ThsrStoptimes{})
	if err != nil {
		log.Printf("proto marshal error: %v", err)
	} else {
		err = rc.Set(fmt.Sprintf("THSR_Stoptimes:%s:%s", date.Format(time.DateOnly), trainno), bytes, 1*time.Hour).Err()
		if err != nil {
			log.Printf("redis set error: %v", err)
		}
	}
}
func tra_timetable(start, end string, date time.Time, ctx context.Context, client *resty.Client, db *pgxpool.Pool, rc *redis.Client) {
	mp := make(map[string]*models.TraTimetable)
	arr := []*models.TraTimetable{}
	rows, err := db.Query(ctx, `SELECT train_date,trainno, starting_station_id,starting_station_name,ending_station_id,ending_station_name, stopsequence,train_type_id,train_type_code,train_type_name,tripline,stationid,arrivaltime,stationname,mask,note,departuretime FROM tra_timetable WHERE stationid = $1 AND train_date = $2 AND arrivaltime >= $3;`, start, date.Format(time.DateOnly), date.Format(time.TimeOnly))
	row, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[Tratimetable])
	rows.Close()
	if len(row) == 0 {
		get_tra_timetable(ctx, db, client, rc, date.Format(time.DateOnly))
		rows, _ := db.Query(ctx, `SELECT train_date,trainno, starting_station_id,starting_station_name,ending_station_id,ending_station_name, stopsequence,train_type_id,train_type_code,train_type_name,tripline,stationid,arrivaltime,stationname,mask,note,departuretime FROM tra_timetable WHERE stationid = $1 AND train_date = $2 AND arrivaltime >= $3;`, start, date.Format(time.DateOnly), date.Format(time.TimeOnly))
		row, err = pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[Tratimetable])
		rows.Close()
	}
	for _, temp := range row {
		mp[temp.Trainno] = &models.TraTimetable{
			TrainDate:             temp.Train_date.Format(time.DateOnly),
			TrainNo:               temp.Trainno,
			Starting_Station_Name: temp.Starting_station_name,
			Ending_Station_Name:   temp.Ending_station_name,
			TrainTypeCode:         temp.Train_type_code,
			TrainTypeName:         temp.Train_type_name,
			TrainTypeID:           temp.Train_type_id,
			TripLine:              temp.Tripline,
			Mask:                  temp.Mask,
			Note:                  temp.Note,
			Starting_Time:         temp.Arrivaltime.Format(time.RFC3339),
		}
	}
	rows, _ = db.Query(ctx, `SELECT train_date,trainno, starting_station_id,starting_station_name,ending_station_id,ending_station_name, stopsequence,train_type_id,train_type_code,train_type_name,tripline,stationid,arrivaltime,stationname,mask,note,departuretime FROM tra_timetable WHERE stationid = $1 AND train_date = $2 AND arrivaltime >= $3;`, end, date.Format(time.DateOnly), date.Format(time.TimeOnly))
	defer rows.Close()
	row, err = pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[Tratimetable])
	if err != nil {
		log.Println(err.Error())
	}
	for _, temp := range row {
		seed, ok := mp[temp.Trainno]
		if !ok {
			continue
		}
		w, err := time.Parse(time.RFC3339, seed.Starting_Time)
		if err != nil {
			log.Printf("parse time error: %v", err)
			continue
		}
		t := temp.Arrivaltime
		duration := t.Sub(w)
		if duration < 0 {
			duration = -duration
		}
		seed.Ending_Time = t.Format(time.RFC3339)
		seed.Travel_Time = duration.String()
		arr = append(arr, seed)
	}
	bytes, _ := proto.Marshal(&models.TraTimetables{Items: arr})
	if err != nil {
		log.Printf("proto marshal error: %v", err)
	} else {
		err = rc.Set(fmt.Sprintf("TRA_timetable:%s:%s:%s", date.Format(time.DateOnly), start, end), bytes, 1*time.Hour).Err()
		if err != nil {
			log.Printf("redis set error: %v", err)
		}
	}
}
func thsr_timetable(start, end string, date time.Time, ctx context.Context, client *resty.Client, db *pgxpool.Pool, rc *redis.Client) {
	mp := make(map[string]*models.ThsaTimetable)
	arr := []*models.ThsaTimetable{}

	// First pass: collect departure info from start station
	rows, _ := db.Query(ctx, `SELECT trainno, starting_station_id,starting_station_name,ending_station_id,ending_station_name,arrivaltime,note,overnight FROM thsr_timetable WHERE stationid = $1 AND train_date = $2 AND arrivaltime >= $3;`, start, date.Format(time.DateOnly), date.Format(time.TimeOnly))
	if rows == nil {
		get_thsr_timetable(ctx, db, client, rc, date.Format(time.DateOnly))
		rows, _ = db.Query(ctx, `SELECT trainno, starting_station_id,starting_station_name,ending_station_id,ending_station_name,arrivaltime,note,overnight FROM thsr_timetable WHERE stationid = $1 AND train_date = $2 AND arrivaltime >= $3;`, start, date.Format(time.DateOnly), date.Format(time.TimeOnly))
	}
	if rows != nil {
		row, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[Thsrtimetable])
		rows.Close()
		if err != nil {
			log.Println(err.Error())
		}
		for _, temp := range row {
			mp[temp.trainno] = &models.ThsaTimetable{
				TrainDate:             temp.train_date,
				TrainNo:               temp.trainno,
				Starting_Station_Name: temp.Starting_station_name,
				EndingStationName:     temp.Ending_station_name,
				Note:                  temp.Note,
				Overnight:             temp.Overnight,
				Starting_Time:         temp.Arrivaltime,
			}
		}
	}
	rows2, _ := db.Query(ctx, `SELECT trainno,arrivaltime FROM thsr_timetable WHERE stationid = $1 AND train_date = $2;`, end, date.Format(time.DateOnly))
	if rows2 != nil {
		row2, err := pgx.CollectRows(rows2, pgx.RowToAddrOfStructByName[Thsrtimetable])
		rows2.Close()
		if err != nil {
			log.Println(err.Error())
		}
		for _, temp := range row2 {
			seed, ok := mp[temp.trainno]
			if !ok {
				continue
			}
			w, err := time.Parse(time.TimeOnly, seed.Starting_Time)
			if err != nil {
				log.Printf("parse time error: %v", err)
				continue
			}
			t, err := time.Parse(time.TimeOnly, temp.Arrivaltime)
			if err != nil {
				log.Printf("parse time error: %v", err)
				continue
			}
			duration := t.Sub(w)
			if seed.Overnight {
				duration += 24 * time.Hour
			}
			seed.Ending_Time = temp.Arrivaltime
			seed.Travel_Time = duration.String()
			arr = append(arr, seed)
		}
	}

	bytes, err := proto.Marshal(&models.ThsrTimetables{Items: arr})
	if err != nil {
		log.Printf("proto marshal error: %v", err)
	} else {
		if err = rc.Set(fmt.Sprintf("THSR_timetable:%s:%s:%s", date.Format(time.DateOnly), start, end), bytes, 1*time.Hour).Err(); err != nil {
			log.Printf("redis set error: %v", err)
		}
	}
}
func get_tra_timetable(ctx context.Context, db *pgxpool.Pool, client *resty.Client, rc *redis.Client, Date string) {
	log.Printf("[RAIL] action=tra_timetable event=start")
	dec, comp, err, flipopen := callApi(client, rc, fmt.Sprintf("/v2/Rail/TRA/DailyTimetable/TrainDate/%s", Date), "tra_traindate")
	if err != nil || !comp {
		log.Printf("[RAIL] action=tra_timetable event=skip reason=api_error")
		return
	}
	defer flipopen()
	if _, err := dec.Token(); err != nil {
		log.Printf("[RAIL] action=tra_timetable event=decode_error error=%v", err)
		return
	}
	var row [][]interface{}
	for dec.More() {
		var temp raw_tra_timetable
		if err := dec.Decode(&temp); err == nil {
			for _, stop := range temp.StopTimes {
				at, _ := time.Parse("15:04", stop.ArrivalTime)
				dt, _ := time.Parse("15:04", stop.DepartureTime)
				row = append(row, []interface{}{
					temp.TrainDate,
					temp.DailyTrainInfo.TrainNo,
					temp.DailyTrainInfo.Direction,
					temp.DailyTrainInfo.StartingStationID,
					temp.DailyTrainInfo.StartingStationName.ZhTw,
					temp.DailyTrainInfo.EndingStationID,
					temp.DailyTrainInfo.EndingStationName.ZhTw,
					temp.DailyTrainInfo.TrainTypeID,
					temp.DailyTrainInfo.TrainTypeCode,
					temp.DailyTrainInfo.TrainTypeName.ZhTw,
					temp.DailyTrainInfo.TripLine,
					stop.StopSequence,
					stop.StationID,
					stop.StationName.ZhTw,
					at,
					dt,
					mask(temp.DailyTrainInfo.WheelchairFlag, temp.DailyTrainInfo.PackageServiceFlag, temp.DailyTrainInfo.DiningFlag, temp.DailyTrainInfo.BikeFlag, temp.DailyTrainInfo.BreastFeedingFlag, temp.DailyTrainInfo.DailyFlag, temp.DailyTrainInfo.ServiceAddedFlag, stop.SuspendedFlag),
					temp.DailyTrainInfo.Note.ZhTw,
				})
			}
		}
	}
	if len(row) > 0 {
		c1 := `CREATE TEMP TABLE temp_tra (
					train_date          date not null,
					trainno             text not null,
					direction           integer,
					starting_station_id text not null,	
					starting_station_name text not null,
					ending_station_id   text not null,
					ending_station_name  text not null,
					train_type_id       text,
					train_type_code     text,
					train_type_name     text,
					tripline            integer,
					stopsequence        smallint,
					stationid           text,
					stationname         text,
					arrivaltime         time,
					departuretime       time,
					mask                smallint,
					note                text,
					updated_at          timestamp with time zone
				) ON COMMIT DROP;`
		c2 := `INSERT INTO tra_timetable (
				train_date,
				trainno,
				direction,
				starting_station_id,
				starting_station_name,
				ending_station_id,
				ending_station_name,
				train_type_id,
				train_type_code,
				train_type_name,
				tripline,
				stopsequence,
				stationid,
				stationname,
				arrivaltime,
				departuretime,
				mask,
				note,
				updated_at
			)
			SELECT train_date,trainno,direction,starting_station_id,starting_station_name,ending_station_id,ending_station_name,train_type_id,train_type_code,train_type_name,tripline,stopsequence,stationid,stationname,arrivaltime,departuretime,mask,note,NOW()
			FROM temp_tra
			ON CONFLICT (train_date,trainno,stationid) DO UPDATE SET direction = EXCLUDED.direction, starting_station_id = EXCLUDED.starting_station_id,starting_station_name = excluded.starting_station_name,ending_station_name = excluded.ending_station_name, ending_station_id = EXCLUDED.ending_station_id,train_type_id = excluded.train_type_id, train_type_code = excluded.train_type_code, train_type_name = excluded.train_type_name, tripline = excluded.tripline, stopsequence = excluded.stopsequence, stationname = excluded.stationname, arrivaltime = excluded.arrivaltime, departuretime = excluded.departuretime, mask = excluded.mask, note = excluded.note, updated_at = NOW();`
		b, err := db.Begin(ctx)
		if err != nil {
			log.Printf("[RAIL] action=tra_timetable event=tx_error error=%v", err)
			return
		}
		defer func(b pgx.Tx, ctx context.Context) {
			_ = b.Rollback(ctx)
		}(b, ctx)
		if _, err := b.Exec(ctx, c1); err != nil {
			log.Printf("[RAIL] action=timetable event=create_temp_error error=%v", err)
			return
		}
		_, err = b.CopyFrom(ctx, pgx.Identifier{"temp_tra"}, []string{"train_date", "trainno", "direction", "starting_station_id", "starting_station_name", "ending_station_id", "ending_station_name", "train_type_id", "train_type_code", "train_type_name", "tripline", "stopsequence", "stationid", "stationname", "arrivaltime", "departuretime", "mask", "note"}, pgx.CopyFromRows(row))
		if err == nil {
			if _, err = b.Exec(ctx, c2); err != nil {
				log.Printf("[RAIL] action=tra_timetable event=exec_error error=%v", err)
				return
			}
			if err = b.Commit(ctx); err != nil {
				log.Printf("[RAIL] action=tra_timetable event=commit_error error=%v", err)
			} else {
				log.Printf("[RAIL] action=tra_timetable event=success timetable_count=%d", len(row))
			}
		} else {
			log.Printf("[RAIL] action=tra_timetable event=copyfrom_error error=%v", err)
		}
	} else {
		log.Printf("[RAIL] action=tra_timetable event=complete reason=no_data")
	}
	log.Printf("[RAIL] action=tra_timetable event=complete")
}

func get_thsr_timetable(ctx context.Context, db *pgxpool.Pool, client *resty.Client, rc *redis.Client, Date string) {
	log.Printf("[RAIL] action=thsr_timetable event=start")
	dec, comp, err, flipopen := callApi(client, rc, fmt.Sprintf("/v2/Rail/THSR/DailyTimetable/TrainDate/%s", Date), "thsr_traindate")
	if err != nil || !comp {
		log.Printf("[RAIL] action=thsr_timetable event=skip reason=api_error")
		return
	}
	defer flipopen()
	if _, err := dec.Token(); err != nil {
		log.Printf("[RAIL] action=thsr_timetable event=decode_error error=%v", err)
		return
	}
	var row [][]interface{}
	for dec.More() {
		var temp raw_thsr_timetable
		if err := dec.Decode(&temp); err == nil {
			for _, stop := range temp.StopTimes {
				row = append(row, []interface{}{
					temp.TrainDate,
					temp.DailyTrainInfo.TrainNo,
					temp.DailyTrainInfo.Direction,
					temp.DailyTrainInfo.StartingStationID,
					temp.DailyTrainInfo.StartingStationName.ZhTw,
					temp.DailyTrainInfo.EndingStationID,
					temp.DailyTrainInfo.EndingStationName.ZhTw,
					stop.StationID,
					stop.StationName.ZhTw,
					stop.ArrivalTime,
					stop.DepartureTime,
					temp.DailyTrainInfo.Note.ZhTw,
					temp.DailyTrainInfo.Overnight,
				})
			}
		}
	}
	if len(row) > 0 {
		c1 := `CREATE TEMP TABLE temp_thsr (
					train_date          date not null,
					trainno             text not null,
					direction           integer,
					starting_station_id text not null,
					starting_station_name text not null,
					ending_station_id   text not null,
					ending_station_name  text not null,
					stopsequence        smallint,
					stationid           text,
					stationname         text,
					arrivaltime         time,
					departuretime       time,
					note                text,
					overnight           boolean,
					updated_at          timestamp with time zone
				) ON COMMIT DROP;`
		c2 := `INSERT INTO thsr_timetable (
				train_date,
				trainno,
				direction,
				starting_station_id,
				starting_station_name,
				ending_station_id,
				ending_station_name,
				stopsequence,
				stationid,
				stationname,
				arrivaltime,
				departuretime,
				note,
				overnight,
				updated_at
			)
			SELECT train_date,trainno,direction,starting_station_id,starting_station_name,ending_station_id,ending_station_name,stopsequence,stationid,stationname,arrivaltime,departuretime,note,overnight,NOW()
			FROM temp_thsr
			ON CONFLICT (train_date,trainno,stationid) DO UPDATE SET direction = EXCLUDED.direction, starting_station_id = EXCLUDED.starting_station_id,starting_station_name = excluded.starting_station_name,ending_station_name = excluded.ending_station_name, ending_station_id = EXCLUDED.ending_station_id,stopsequence = excluded.stopsequence, stationname = excluded.stationname, arrivaltime = excluded.arrivaltime, departuretime = excluded.departuretime,note = excluded.note, overnight = excluded.overnight, updated_at = NOW();`
		b, err := db.Begin(ctx)
		if err != nil {
			log.Printf("[RAIL] action=thsr_timetable event=tx_error error=%v", err)
			return
		}
		defer func(b pgx.Tx, ctx context.Context) {
			_ = b.Rollback(ctx)
		}(b, ctx)
		if _, err := b.Exec(ctx, c1); err != nil {
			log.Printf("[RAIL] action=timetable event=create_temp_error error=%v", err)
			return
		}
		_, err = b.CopyFrom(ctx, pgx.Identifier{"temp_thsr"}, []string{"train_date", "trainno", "direction", "starting_station_id", "starting_station_name", "ending_station_id", "ending_station_name", "stopsequence", "stationid", "stationname", "arrivaltime", "departuretime", "note", "overnight"}, pgx.CopyFromRows(row))
		if err == nil {
			if _, err = b.Exec(ctx, c2); err != nil {
				log.Printf("[RAIL] action=thsr_timetable event=exec_error error=%v", err)
				return
			}
			if err = b.Commit(ctx); err != nil {
				log.Printf("[RAIL] action=thsr_timetable event=commit_error error=%v", err)
			} else {
				log.Printf("[RAIL] action=thsr_timetable event=success timetable_count=%d", len(row))
			}
		} else {
			log.Printf("[RAIL] action=thsr_timetable event=copyfrom_error error=%v", err)
		}
	} else {
		log.Printf("[RAIL] action=thsr_timetable event=complete reason=no_data")
	}
	log.Printf("[RAIL] action=thsr_timetable event=complete")
}

func callApi(client *resty.Client, rc *redis.Client, url string, name string) (*json.Decoder, bool, error, func()) {
	since, _ := rc.Get("LastTimeGet_" + name).Result()
	resp, err := client.R().
		SetHeader("If-Modified-Since", since).
		Get(url)
	if err != nil {
		return &json.Decoder{}, false, err, nil
	}
	if resp.StatusCode() == 304 {
		err := resp.RawResponse.Body.Close()
		if err != nil {
			return &json.Decoder{}, false, err, nil
		}
		log.Printf("[RUN] action=no update=%s", name)
		return &json.Decoder{}, false, nil, nil
	}
	rc.Set("LastTimeGet_"+name, resp.Header().Get("Last-Modified"), 0)
	decorder := json.NewDecoder(resp.RawResponse.Body)
	return decorder, true, nil, func() {
		err := resp.RawResponse.Body.Close()
		if err != nil {
			log.Printf("[RUN] action=fail-close-response error=%v", err)
		}
	}
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
func mask(wheel, pack, dining, bike, breast, daily, service, suspended uint8) uint16 {
	var res uint16
	days := []uint8{wheel, pack, dining, bike, breast, daily, service, suspended}
	for i, v := range days {
		if v == 1 {
			res |= 1 << i
		}
	}
	return res
}
func gettrafares(ctx context.Context, client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	log.Printf("[RAIL] action=get_tra_fare event=start")
	dec, comp, err, flipopen := callApi(client, rc, fmt.Sprintf("/v2/Rail/TRA/ODFare"), "tra_fare")
	if err != nil || !comp {
		log.Printf("[RAIL] action=get_tra_fare event=skip reason=api_error")
	}
	func() {
		defer flipopen()
		if _, err := dec.Token(); err != nil {
			log.Printf("[RAIL] action=get_tra_fare event=decode_error error=%v", err)
			return
		}
		var row [][]interface{}
		for dec.More() {
			var temp raw_trafare
			if err := dec.Decode(&temp); err == nil {
				row = append(row, []interface{}{
					temp.OriginStationID,
					temp.DestinationStationID,
					temp.TicketType,
					temp.Price,
				})
			}
		}
		if len(row) > 0 {
			c1 := `CREATE TEMP TABLE temp_tra (
					osi text,
					dsi text,
					ty text,
					p smallint
				) ON COMMIT DROP;`
			c2 := `INSERT INTO tra_fares (
					origin_station_id,
					destination_station_id,
					ticket_type,
					price,
                    updated_at
				)
				SELECT osi, dsi, ty, p,NOW() FROM temp_tra
				ON CONFLICT (origin_station_id,destination_station_id,ticket_type) DO UPDATE SET price = EXCLUDED.price,updated_at = NOW();`
			b, err := db.Begin(ctx)
			if err != nil {
				log.Printf("[rail] action=get_tra_fare event=begin_error error=%v", err)
				return
			}
			defer func(b pgx.Tx, ctx context.Context) {
				_ = b.Rollback(ctx)
			}(b, ctx)
			_, _ = b.Exec(ctx, c1)
			_, err = b.CopyFrom(ctx, pgx.Identifier{"temp_tra"}, []string{"osi", "dsi", "ty", "p"}, pgx.CopyFromRows(row))
			if err == nil {
				if _, execErr := b.Exec(ctx, c2); execErr != nil {
					log.Printf("[RAIL] action=get_tra_fare event=exec_error error=%v", execErr)
				}
				if commitErr := b.Commit(ctx); commitErr != nil {
					log.Printf("[rail] action=get_tra_fare event=commit_error error=%v", commitErr)
				} else {
					log.Printf("[rail] action=get_tra_fare event=success station_count=%d", len(row))
				}
			} else {
				log.Printf("[rail] action=get_tra_fare event=copyfrom_error error=%v", err)
				_ = b.Rollback(ctx)
			}
		}
	}()
	log.Printf("[rail] action=get_tra_fare event=complete")
}
func getthsrfares(ctx context.Context, client *resty.Client, rc *redis.Client, db *pgxpool.Pool) {
	log.Printf("[RAIL] action=get_thsr_fare event=start")
	dec, comp, err, flipopen := callApi(client, rc, fmt.Sprintf("/v2/Rail/THSR/ODFare"), "thsr_fare")
	if err != nil || !comp {
		log.Printf("[RAIL] action=get_thsr_fare event=skip reason=api_error")
	}
	func() {
		defer flipopen()
		if _, err := dec.Token(); err != nil {
			log.Printf("[RAIL] action=get_thsr_fare event=decode_error error=%v", err)
			return
		}
		var row [][]interface{}
		for dec.More() {
			var temp raw_thsrfare
			if err := dec.Decode(&temp); err == nil {
				row = append(row, []interface{}{
					temp.OriginStationID,
					temp.DestinationStationID,
					temp.TicketType,
					temp.FareClass,
					temp.CabinClass,
					temp.Price,
				})
			}
		}
		if len(row) > 0 {
			c1 := `CREATE TEMP TABLE temp_thsr (
					osi text,
					dsi text,
					ty smallint,
					fare smallint,
					cabin smallint,
					p smallint
				) ON COMMIT DROP;`
			c2 := `INSERT INTO thsr_fares (
					origin_station_id,
					destination_station_id,
					ticket_type,
					fare_class,
                    cabin_class,
					price,
					updated_at
				)
				SELECT osi, dsi, ty, fare,cabin,p,NOW() FROM temp_thsr
				ON CONFLICT (origin_station_id,destination_station_id,ticket_type,fare_class,cabin_class) DO UPDATE SET price = EXCLUDED.price,updated_at = NOW();`
			b, err := db.Begin(ctx)
			if err != nil {
				log.Printf("[rail] action=get_tra_fare event=begin_error error=%v", err)
				return
			}
			defer func(b pgx.Tx, ctx context.Context) {
				_ = b.Rollback(ctx)
			}(b, ctx)
			_, _ = b.Exec(ctx, c1)
			_, err = b.CopyFrom(ctx, pgx.Identifier{"temp_thsr"}, []string{"osi", "dsi", "ty", "fare", "cabin", "p"}, pgx.CopyFromRows(row))
			if err == nil {
				if _, execErr := b.Exec(ctx, c2); execErr != nil {
					log.Printf("[RAIL] action=get_thsr_fare event=exec_error error=%v", execErr)
				}
				if commitErr := b.Commit(ctx); commitErr != nil {
					log.Printf("[rail] action=get_thsr_fare event=commit_error error=%v", commitErr)
				} else {
					log.Printf("[rail] action=get_thsr_fare event=success station_count=%d", len(row))
				}
			} else {
				log.Printf("[rail] action=get_thsr_fare event=copyfrom_error error=%v", err)
				_ = b.Rollback(ctx)
			}
		}
	}()
	log.Printf("[rail] action=get_thsr_fare event=complete")
}
