package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type cwa struct {
	records struct {
		Station []struct {
			StationName     string `json:"StationName"`
			RainfallElement struct {
				Now struct {
					Precipitation float64 `json:"Precipitation"`
				} `json:"Now"`
				Past10Min struct {
					Precipitation float64 `json:"Precipitation"`
				}
			} `json:"RainfallElement"`
		} `json:"Station"`
	} `json:"records"`
}

func main() {
	db := connectdb()
	rc := connectredis()
	c := resty.New().
		SetDoNotParseResponse(true).
		SetQueryParams(map[string]string{
			"Authorization":   os.Getenv("WEARTHERAPI"),
			"limit":           "10",
			"RainfallElement": "Now,Past10Min",
		}).
		SetBaseURL("https://opendata.cwa.gov.tw/api")
}
func get_clostest_timetable(rc *redis.Client, db *pgxpool.Pool) {

}
func get_weather(c *resty.Client, rc *redis.Client, city string) (float64, float64, error) {
	val, err := rc.Get("weather:" + city).Result()
	if val != "" {
		temp := strings.Split(val, ":")
		t1, _ := strconv.ParseFloat(temp[0], 32)
		t2, _ := strconv.ParseFloat(temp[1], 32)
		return t1, t2, nil
	}
	resp, err := c.SetHeader("StationName", city).R().Get("v1/rest/datastore/O-A0002-001")
	if err != nil {
		return 0, 0, err
	}
	dec := json.NewDecoder(resp.RawResponse.Body)
	defer resp.RawResponse.Body.Close()
	if _, err = dec.Token(); err != nil {
		return 0, 0, err
	}
	pipe := rc.Pipeline()
	for dec.More() {
		var temp cwa
		if err := dec.Decode(&temp); err == nil {
			t1 := temp.records.Station[0].RainfallElement.Now.Precipitation
			t2 := temp.records.Station[0].RainfallElement.Past10Min.Precipitation
			pipe.Set(fmt.Sprintf("weather:%s", temp.records.Station[0].StationName), fmt.Sprintf("%.2f:%.2f", t1, t2), 10*time.Minute)
			return t1, t2, nil
		}
	}
	_, _ = pipe.Exec()
	return 0, 0, err
}
func connectredis() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	pong, err := client.Ping().Result()
	if err != nil {
		panic(err)
	}
	fmt.Println(pong, err)
	return client
}
func connectdb() *pgxpool.Pool {
	conn, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}
	err = conn.Ping(context.Background())
	if err != nil {
		panic(err)
	}
	return conn
}
