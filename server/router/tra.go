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
)

type raw_tra_timetable struct {
	TrainDate      string `json:"TrainDate"`
	DailyTrainInfo struct {
		TrainNo             string `json:"TrainNo"`
		Direction           bool   `json:"Direction"`
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
		Direction           bool   `json:"Direction"`
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
					stop.StationID,
					stop.StationName.ZhTw,
					stop.ArrivalTime,
					stop.DepartureTime,
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
		_, err = b.CopyFrom(ctx, pgx.Identifier{"temp_tra"}, []string{"train_date", "trainno", "direction", "starting_station_id", "ending_station_id", "train_type_id", "train_type_code", "train_type_name", "tripline", "stopsequence", "stationid", "stationname", "arrivaltime", "departuretime", "mask", "note"}, pgx.CopyFromRows(row))
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
	dec, comp, err, flipopen := callApi(client, rc, fmt.Sprintf("/v2/Rail/THSR/DailyTimetable/TrainDate/%s", Date), "tra_traindate")
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
				updated_at
			)
			SELECT train_date,trainno,direction,starting_station_id,starting_station_name,ending_station_id,ending_station_name,stopsequence,stationid,stationname,arrivaltime,departuretime,note,NOW()
			FROM temp_thsr
			ON CONFLICT (train_date,trainno,stationid) DO UPDATE SET direction = EXCLUDED.direction, starting_station_id = EXCLUDED.starting_station_id,starting_station_name = excluded.starting_station_name,ending_station_name = excluded.ending_station_name, ending_station_id = EXCLUDED.ending_station_id,stopsequence = excluded.stopsequence, stationname = excluded.stationname, arrivaltime = excluded.arrivaltime, departuretime = excluded.departuretime,note = excluded.note, updated_at = NOW();`
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
		_, err = b.CopyFrom(ctx, pgx.Identifier{"temp_thsr"}, []string{"train_date", "trainno", "direction", "starting_station_id", "ending_station_id", "stopsequence", "stationid", "stationname", "arrivaltime", "departuretime", "note"}, pgx.CopyFromRows(row))
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
