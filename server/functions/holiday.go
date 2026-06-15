package main

import (
	"encoding/csv"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const holidayCSVURL = "https://data.gov.tw/api/v2/rest/datastore/MOEA_4.csv"

var holidayMap map[string]bool

var taipei *time.Location

func init() {
	var err error
	taipei, err = time.LoadLocation("Asia/Taipei")
	if err != nil {
		panic("cannot load Asia/Taipei: " + err.Error())
	}
}

func loadHolidays() {
	resp, err := http.Get(holidayCSVURL)
	if err != nil {
		log.Printf("[HOLIDAY] fetch error: %v — weekday fallback active", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Printf("[HOLIDAY] fetch error: status=%d — weekday fallback active", resp.StatusCode)
		return
	}
	r := csv.NewReader(resp.Body)
	r.Read()
	m := make(map[string]bool)
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(rec) < 4 {
			continue
		}
		date := strings.TrimSpace(rec[0])
		if len(date) != 8 {
			continue
		}
		t, err := time.Parse("20060102", date)
		if err != nil {
			continue
		}
		m[t.Format("2006-01-02")] = strings.TrimSpace(rec[3]) == "是"
	}
	holidayMap = m
	log.Printf("[HOLIDAY] loaded %d entries", len(m))
}

func isHoliday(t time.Time) bool {
	key := t.In(taipei).Format("2006-01-02")
	if holidayMap != nil {
		return holidayMap[key]
	}
	wd := t.In(taipei).Weekday()
	return wd == time.Saturday || wd == time.Sunday
}
