package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/go-resty/resty/v2"
)

const cwaBase = "https://opendata.cwa.gov.tw/api/v1/rest/datastore"

type weatherData struct {
	Temperature   float64 `json:"temperature"`
	Precipitation float64 `json:"precipitation"`
	WindSpeed     float64 `json:"wind_speed"`
	Humidity      float64 `json:"humidity"`
}

var cityCoords = map[string][2]float64{
	"Taipei":         {25.04, 121.55},
	"NewTaipei":      {24.99, 121.46},
	"Taoyuan":        {24.99, 121.22},
	"Taichung":       {24.14, 120.67},
	"Tainan":         {22.99, 120.22},
	"Kaohsiung":      {22.65, 120.31},
	"Keelung":        {25.13, 121.73},
	"Hsinchu":        {24.80, 120.97},
	"HsinchuCounty":  {24.67, 121.00},
	"MiaoliCounty":   {24.56, 120.82},
	"ChanghuaCounty": {24.07, 120.52},
	"NantouCounty":   {23.96, 120.68},
	"YunlinCounty":   {23.71, 120.40},
	"Chiayi":         {23.48, 120.45},
	"ChiayiCounty":   {23.45, 120.45},
	"PingtungCounty": {22.54, 120.49},
	"YilanCounty":    {24.70, 121.75},
	"HualienCounty":  {23.99, 121.60},
	"TaitungCounty":  {22.75, 121.11},
}

var countyToCity = map[string]string{
	"臺北市": "Taipei", "新北市": "NewTaipei", "桃園市": "Taoyuan",
	"臺中市": "Taichung", "臺南市": "Tainan", "高雄市": "Kaohsiung",
	"基隆市": "Keelung", "新竹市": "Hsinchu", "新竹縣": "HsinchuCounty",
	"苗栗縣": "MiaoliCounty", "彰化縣": "ChanghuaCounty", "南投縣": "NantouCounty",
	"雲林縣": "YunlinCounty", "嘉義市": "Chiayi", "嘉義縣": "ChiayiCounty",
	"屏東縣": "PingtungCounty", "宜蘭縣": "YilanCounty", "花蓮縣": "HualienCounty",
	"臺東縣": "TaitungCounty",
}

func gridPrecipitation(vals []float64, dimX int, startLon, startLat, res, lon, lat float64) *float64 {
	x := int(math.Round((lon - startLon) / res))
	y := int(math.Round((lat - startLat) / res))
	if x < 0 || y < 0 || x >= dimX || y*dimX+x >= len(vals) {
		return nil
	}
	v := vals[y*dimX+x]
	if v <= -90 {
		return nil
	}
	return &v
}

func weatherSync(rc *redis.Client) {
	cwaKey := os.Getenv("CWA_API_KEY")
	if cwaKey == "" {
		log.Printf("[WEATHER] CWA_API_KEY not set, skipping")
		return
	}
	client := resty.New()
	merged := make(map[string]weatherData)

	var obs struct {
		Records struct {
			Station []struct {
				GeoInfo struct {
					CountyName string `json:"CountyName"`
				} `json:"GeoInfo"`
				ObsTime struct {
					DateTime string `json:"DateTime"`
				} `json:"ObsTime"`
				WeatherElement struct {
					AirTemperature   string `json:"AirTemperature"`
					WindSpeed        string `json:"WindSpeed"`
					RelativeHumidity string `json:"RelativeHumidity"`
				} `json:"WeatherElement"`
			} `json:"Station"`
		} `json:"records"`
	}
	type stationBest struct {
		obsTime time.Time
		data    weatherData
	}
	best := make(map[string]stationBest)

	resp, err := client.R().SetHeader("Authorization", cwaKey).
		Get(cwaBase + "/O-A0003-001")
	if err == nil && resp.StatusCode() == 200 {
		if jsonErr := json.Unmarshal(resp.Body(), &obs); jsonErr == nil {
			for _, s := range obs.Records.Station {
				city, ok := countyToCity[s.GeoInfo.CountyName]
				if !ok {
					continue
				}
				t, _ := time.Parse(time.RFC3339, s.ObsTime.DateTime)
				if prev, exists := best[city]; exists && !t.After(prev.obsTime) {
					continue
				}
				d := weatherData{}
				if v, err := strconv.ParseFloat(s.WeatherElement.AirTemperature, 64); err == nil && v > -90 {
					d.Temperature = v
				}
				if v, err := strconv.ParseFloat(s.WeatherElement.WindSpeed, 64); err == nil && v > -90 {
					d.WindSpeed = v
				}
				if v, err := strconv.ParseFloat(s.WeatherElement.RelativeHumidity, 64); err == nil && v > -90 {
					d.Humidity = v
				}
				best[city] = stationBest{t, d}
			}
		}
		for city, b := range best {
			merged[city] = b.data
		}
	} else if err != nil {
		log.Printf("[WEATHER] O-A0003-001 fetch error: %v", err)
	} else {
		log.Printf("[WEATHER] O-A0003-001 error status=%d", resp.StatusCode())
	}

	var grid struct {
		Cwaopendata struct {
			Dataset struct {
				DatasetInfo struct {
					ParameterSet struct {
						StartPointLongitude string `json:"StartPointLongitude"`
						StartPointLatitude  string `json:"StartPointLatitude"`
						GridResolution      string `json:"GridResolution"`
						GridDimensionX      string `json:"GridDimensionX"`
					} `json:"parameterSet"`
				} `json:"datasetInfo"`
				Contents struct {
					Content string `json:"content"`
				} `json:"contents"`
			} `json:"dataset"`
		} `json:"cwaopendata"`
	}
	resp2, err2 := client.R().SetHeader("Authorization", cwaKey).
		Get(cwaBase + "/F-B0046-001")
	if err2 == nil && resp2.StatusCode() == 200 {
		if jsonErr := json.Unmarshal(resp2.Body(), &grid); jsonErr == nil {
			ps := grid.Cwaopendata.Dataset.DatasetInfo.ParameterSet
			startLon, _ := strconv.ParseFloat(ps.StartPointLongitude, 64)
			startLat, _ := strconv.ParseFloat(ps.StartPointLatitude, 64)
			res, _ := strconv.ParseFloat(ps.GridResolution, 64)
			dimX, _ := strconv.Atoi(ps.GridDimensionX)
			if res <= 0 || dimX <= 0 {
				log.Printf("[WEATHER] F-B0046-001 invalid grid params, skipping")
			} else {
				parts := strings.Split(grid.Cwaopendata.Dataset.Contents.Content, ",")
				vals := make([]float64, len(parts))
				for i, p := range parts {
					vals[i], _ = strconv.ParseFloat(strings.TrimSpace(p), 64)
				}
				for city, coords := range cityCoords {
					p := gridPrecipitation(vals, dimX, startLon, startLat, res, coords[1], coords[0])
					d := merged[city]
					if p != nil {
						d.Precipitation = *p
					}
					merged[city] = d
				}
			}
		}
	} else if err2 != nil {
		log.Printf("[WEATHER] F-B0046-001 fetch error: %v", err2)
	} else {
		log.Printf("[WEATHER] F-B0046-001 error status=%d", resp2.StatusCode())
	}

	for city, d := range merged {
		b, _ := json.Marshal(d)
		rc.Set(fmt.Sprintf("weather:%s", city), string(b), 15*time.Minute)
	}
	log.Printf("[WEATHER] synced %d cities", len(merged))
}
