package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jnjkhjlkjhb8/wheres_the_car/models"
)

type bikeSations struct {
	StationUid string  `db:"station_uid"`
	Name       string  `db:"name"`
	City       string  `db:"city"`
	Distance   float64 `db:"distance"`
	Lon        float64 `db:"lon"`
	Lat        float64 `db:"lat"`
}
type busStations struct {
	StationUid string  `db:"station_uid"`
	Name       string  `db:"station_name"`
	City       string  `db:"city"`
	Distance   float64 `db:"distance"`
	Lon        float64 `db:"lon"`
	Lat        float64 `db:"lat"`
}
type mrtSations struct {
	StationUid string  `db:"station_id"`
	Name       string  `db:"name"`
	City       string  `db:"city"`
	Distance   float64 `db:"distance"`
	Lon        float64 `db:"lon"`
	Lat        float64 `db:"lat"`
}
type traStations struct {
	StationUid string  `db:"station_id"`
	Name       string  `db:"name"`
	City       string  `db:"city"`
	Distance   float64 `db:"distance"`
	Lon        float64 `db:"lon"`
	Lat        float64 `db:"lat"`
}
type thsrStations struct {
	StationUid string  `db:"station_id"`
	Name       string  `db:"name"`
	City       string  `db:"city"`
	Distance   float64 `db:"distance"`
	Lon        float64 `db:"lon"`
	Lat        float64 `db:"lat"`
}
type osrm struct {
	Code      string      `json:"code"`
	Durations [][]float64 `json:"durations"`
}

func findnearstation(lat, lon float64, size int, ctx context.Context, db *pgxpool.Pool, osrmClient *resty.Client) (*models.RespNear, error) {
	res := models.RespNear{}
	if size <= 0 {
		size = 5
	}
	var wg sync.WaitGroup
	wg.Add(5)
	go func() {
		defer wg.Done()
		rows, _ := db.Query(ctx, `SELECT station_uid, station_name, city, position <-> ST_MakePoint($1,$2)::geography AS distance,ST_X(position) AS lon ,ST_Y(position) AS lat FROM bus_stations ORDER BY position <-> ST_MakePoint($1,$2)::geography LIMIT $3;`, lon, lat, size*10)
		defer rows.Close()
		row, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[busStations])
		if err != nil {
			rows.Close()
			log.Println(err.Error())
		}
		visit := make(map[string]bool)
		cnt := 1
		arr := []string{fmt.Sprintf("%f,%f", lon, lat)}
		ids := make([]string, 0, len(row))
		mp := make(map[string]int)
		for _, r := range row {
			if !visit[r.Name] {
				visit[r.Name] = true
				arr = append(arr, fmt.Sprintf("%f,%f", r.Lon, r.Lat))
				ids = append(ids, strconv.Itoa(cnt))
				mp[r.StationUid] = cnt - 1
				cnt++
			}
		}
		var osrmresp osrm
		yes := false
		c1 := strings.Join(arr, ";")
		c2 := strings.Join(ids, ";")
		resp, err := osrmClient.R().
			SetQueryParam("sources", "0").
			SetQueryParam("destinations", c2).
			SetResult(&osrmresp).
			Get(fmt.Sprintf("http://osrm:5000/table/v1/foot/%s", c1))
		osrmOK := err == nil && resp.IsSuccess() && osrmresp.Code == "Ok" && len(osrmresp.Durations) > 0
		if osrmOK {
			yes = true
		}
		busres := make(map[string]*models.ArrayNear)
		for _, temp := range row {
			var walk int32
			if osrmIdx, exists := mp[temp.StationUid]; exists && yes {
				walk = int32(osrmresp.Durations[0][osrmIdx] / 60)
			}
			ns := &models.NearStation{
				Type:        1,
				StationID:   temp.StationUid,
				StationName: temp.Name,
				City:        temp.City,
				PositionLon: temp.Lon,
				PositionLat: temp.Lat,
				Walk:        walk,
			}
			if _, ok := busres[temp.Name]; !ok {
				busres[temp.Name] = &models.ArrayNear{
					NearStations: []*models.NearStation{},
				}
			}
			busres[temp.Name].NearStations = append(busres[temp.Name].NearStations, ns)
		}
		res.NearBusStations = busres
	}()
	go func() {
		defer wg.Done()
		rows, _ := db.Query(ctx, `SELECT station_uid, name, city, geom <-> ST_MakePoint($1,$2)::geography AS distance,ST_X(geom) AS lon ,ST_Y(geom) AS lat FROM bike_stations ORDER BY geom <-> ST_MakePoint($1,$2)::geography LIMIT $3;`, lon, lat, size*5)
		defer rows.Close()
		row, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[bikeSations])
		if err != nil {
			rows.Close()
			log.Println(err.Error())
		}
		cnt := 1
		arr := []string{fmt.Sprintf("%f,%f", lon, lat)}
		ids := make([]string, 0, len(row))
		mp := make(map[string]int)
		for _, r := range row {
			arr = append(arr, fmt.Sprintf("%f,%f", r.Lon, r.Lat))
			ids = append(ids, strconv.Itoa(cnt))
			mp[r.StationUid] = cnt - 1
			cnt++
		}
		var osrmresp osrm
		yes := false
		c1 := strings.Join(arr, ";")
		c2 := strings.Join(ids, ";")
		resp, err := osrmClient.R().
			SetQueryParam("sources", "0").
			SetQueryParam("destinations", c2).
			SetResult(&osrmresp).
			Get(fmt.Sprintf("http://osrm:5000/table/v1/foot/%s", c1))
		osrmOK := err == nil && resp.IsSuccess() && osrmresp.Code == "Ok" && len(osrmresp.Durations) > 0
		if osrmOK {
			yes = true
		}
		bikeres := make([]*models.NearStation, 0, len(row))
		for _, temp := range row {
			var walk int32
			if osrmIdx, exists := mp[temp.StationUid]; exists && yes {
				walk = int32(osrmresp.Durations[0][osrmIdx] / 60)
			}
			bikeres = append(bikeres, &models.NearStation{
				Type:        2,
				StationID:   temp.StationUid,
				StationName: temp.Name,
				City:        temp.City,
				PositionLon: temp.Lon,
				PositionLat: temp.Lat,
				Walk:        walk,
			})
		}
		res.NearBikeStations = bikeres
	}()
	go func() {
		defer wg.Done()
		rows, _ := db.Query(ctx, `SELECT station_id, name, city, stationposition <-> ST_MakePoint($1,$2)::geography AS distance,ST_X(stationposition) AS lon ,ST_Y(stationposition) AS lat FROM mrt_station ORDER BY stationposition <-> ST_MakePoint($1,$2)::geography LIMIT $3;`, lon, lat, size*5)
		defer rows.Close()
		row, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[mrtSations])
		if err != nil {
			rows.Close()
			log.Println(err.Error())
		}
		cnt := 1
		arr := []string{fmt.Sprintf("%f,%f", lon, lat)}
		ids := make([]string, 0, len(row))
		mp := make(map[string]int)
		for _, r := range row {
			arr = append(arr, fmt.Sprintf("%f,%f", r.Lon, r.Lat))
			ids = append(ids, strconv.Itoa(cnt))
			mp[r.StationUid] = cnt - 1
			cnt++
		}
		var osrmresp osrm
		yes := false
		c1 := strings.Join(arr, ";")
		c2 := strings.Join(ids, ";")
		resp, err := osrmClient.R().
			SetQueryParam("sources", "0").
			SetQueryParam("destinations", c2).
			SetResult(&osrmresp).
			Get(fmt.Sprintf("http://osrm:5000/table/v1/foot/%s", c1))
		osrmOK := err == nil && resp.IsSuccess() && osrmresp.Code == "Ok" && len(osrmresp.Durations) > 0
		if osrmOK {
			yes = true
		}
		mrtres := make([]*models.NearStation, 0, len(row))
		for _, temp := range row {
			var walk int32
			if osrmIdx, exists := mp[temp.StationUid]; exists && yes {
				walk = int32(osrmresp.Durations[0][osrmIdx] / 60)
			}
			mrtres = append(mrtres, &models.NearStation{
				Type:        3,
				StationID:   temp.StationUid,
				StationName: temp.Name,
				City:        temp.City,
				PositionLon: temp.Lon,
				PositionLat: temp.Lat,
				Walk:        walk,
			})
		}
		res.NearMrtStations = mrtres
	}()
	go func() {
		defer wg.Done()
		rows, _ := db.Query(ctx, `SELECT station_id, name, city, geom <-> ST_MakePoint($1,$2)::geography AS distance,ST_X(geom) AS lon ,ST_Y(geom) AS lat FROM tra_stations ORDER BY geom <-> ST_MakePoint($1,$2)::geography LIMIT $3;`, lon, lat, size*5)
		defer rows.Close()
		row, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[traStations])
		if err != nil {
			rows.Close()
			log.Println(err.Error())
		}
		cnt := 1
		arr := []string{fmt.Sprintf("%f,%f", lon, lat)}
		ids := make([]string, 0, len(row))
		mp := make(map[string]int)
		for _, r := range row {
			arr = append(arr, fmt.Sprintf("%f,%f", r.Lon, r.Lat))
			ids = append(ids, strconv.Itoa(cnt))
			mp[r.StationUid] = cnt - 1
			cnt++
		}
		var osrmresp osrm
		yes := false
		c1 := strings.Join(arr, ";")
		c2 := strings.Join(ids, ";")
		resp, err := osrmClient.R().
			SetQueryParam("sources", "0").
			SetQueryParam("destinations", c2).
			SetResult(&osrmresp).
			Get(fmt.Sprintf("http://osrm:5000/table/v1/foot/%s", c1))
		osrmOK := err == nil && resp.IsSuccess() && osrmresp.Code == "Ok" && len(osrmresp.Durations) > 0
		if osrmOK {
			yes = true
		}
		trares := make([]*models.NearStation, 0, len(row))
		for _, temp := range row {
			var walk int32
			if osrmIdx, exists := mp[temp.StationUid]; exists && yes {
				walk = int32(osrmresp.Durations[0][osrmIdx] / 60)
			}
			trares = append(trares, &models.NearStation{
				Type:        4,
				StationID:   temp.StationUid,
				StationName: temp.Name,
				City:        temp.City,
				PositionLon: temp.Lon,
				PositionLat: temp.Lat,
				Walk:        walk,
			})
		}
		res.NearTraStations = trares
	}()
	go func() {
		defer wg.Done()
		rows, _ := db.Query(ctx, `SELECT station_id, name, city, geom <-> ST_MakePoint($1,$2)::geography AS distance,ST_X(geom) AS lon ,ST_Y(geom) AS lat FROM thsr_stations ORDER BY geom <-> ST_MakePoint($1,$2)::geography LIMIT $3;`, lon, lat, size*2)
		defer rows.Close()
		row, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[thsrStations])
		if err != nil {
			rows.Close()
			log.Println(err.Error())
		}
		cnt := 1
		arr := []string{fmt.Sprintf("%f,%f", lon, lat)}
		ids := make([]string, 0, len(row))
		mp := make(map[string]int)
		for _, r := range row {
			arr = append(arr, fmt.Sprintf("%f,%f", r.Lon, r.Lat))
			ids = append(ids, strconv.Itoa(cnt))
			mp[r.StationUid] = cnt - 1
			cnt++
		}
		var osrmresp osrm
		yes := false
		c1 := strings.Join(arr, ";")
		c2 := strings.Join(ids, ";")
		resp, err := osrmClient.R().
			SetQueryParam("sources", "0").
			SetQueryParam("destinations", c2).
			SetResult(&osrmresp).
			Get(fmt.Sprintf("http://osrm:5000/table/v1/foot/%s", c1))
		osrmOK := err == nil && resp.IsSuccess() && osrmresp.Code == "Ok" && len(osrmresp.Durations) > 0
		if osrmOK {
			yes = true
		}
		thsrres := make([]*models.NearStation, 0, len(row))
		for _, temp := range row {
			var walk int32
			if osrmIdx, exists := mp[temp.StationUid]; exists && yes {
				walk = int32(osrmresp.Durations[0][osrmIdx] / 60)
			}
			thsrres = append(thsrres, &models.NearStation{
				Type:        5,
				StationID:   temp.StationUid,
				StationName: temp.Name,
				City:        temp.City,
				PositionLon: temp.Lon,
				PositionLat: temp.Lat,
				Walk:        walk,
			})
		}
		res.NearThsrStations = thsrres
	}()
	wg.Wait()
	return &res, nil
}
