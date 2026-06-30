package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-redis/redis"
	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jnjkhjlkjhb8/wheres_the_car/server/obs"
)

func main() {
	defer obs.Init("functions")()
	defer obs.Recover("main")
	//r := cron.New(cron.WithSeconds())
	c := resty.New()
	rc := connectredis()
	db := connectdb()
	ingestDB = db
	sender, err := newFirebaseSender(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	dispatcher := newNotificationDispatcher(notificationStore{db: db}, sender)
	defer func(rc *redis.Client) {
		err := rc.Close()
		if err != nil {
			log.Printf("[REDIS] action=close event=failed error=%v", err)
		}
	}(rc)
	defer db.Close()
	c.SetBaseURL("https://tdx.transportdata.tw/api/basic").
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "br,gzip").
		SetDoNotParseResponse(true).
		SetRetryCount(5).
		SetRetryWaitTime(1 * time.Second).
		SetRetryMaxWaitTime(5 * time.Second).
		AddRetryCondition(
			func(r *resty.Response, err error) bool {
				if err != nil {
					return true
				}
				if r.StatusCode() == 401 {
					rc.Del("TDX_Token")
					return true
				}
				return r.StatusCode() == 429
			},
		).
		OnBeforeRequest(func(_ *resty.Client, req *resty.Request) error {
			req.SetAuthToken(getToken(rc))
			return nil
		})
	busDailyroute(c, rc)
	loadHolidays()
	loadModel()
	if os.Getenv("FREEZE_DATA") != "" {
		ctx := context.Background()
		log.Println("[FREEZE] action=load event=start")
		busStatic(ctx, c, rc, db)
		bikeStatic(ctx, c, rc, db)
		mrtStatic(ctx, c, rc, db)
		railStatic(ctx, c, rc, db)
		changetovector(ctx, rc, db)
		weatherSync(rc)
		bikeEta(c, rc)
		busEta(ctx, c, rc, db, dispatcher)
		mrtEta(c, rc)
		traEta(c, rc)
		keys, _ := rc.Keys("*").Result()
		for _, k := range keys {
			rc.Persist(k)
		}
		log.Printf("[FREEZE] action=load event=end cron=disabled persisted=%d", len(keys))
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		return
	}
	ctx := context.Background()
	busStatic(ctx, c, rc, db)
	bikeStatic(ctx, c, rc, db)
	mrtStatic(ctx, c, rc, db)
	railStatic(ctx, c, rc, db)
	changetovector(ctx, rc, db)
	/*_, _ = r.AddFunc("0 0 3 * * *", func() {
		ctx := context.Background()
		log.Println("[crontab] action=daily event=start")
		busStatic(ctx, c, rc, db)
		bikeStatic(ctx, c, rc, db)
		mrtStatic(ctx, c, rc, db)
		railStatic(ctx, c, rc, db)
		changetovector(ctx, rc, db)
		log.Println("[crontab] action=daily event=end")
	})
	_, _ = r.AddFunc("@every 2m", func() {
		if sleepingwhiledailyupdate() {
			return
		}
		log.Println("[crontab] action=tra event=start")
		traEta(c, rc)
		log.Println("[crontab] action=tra event=end")
	})
	_, _ = r.AddFunc("@every 30s", func() {
		if sleepingwhiledailyupdate() {
			return
		}
		log.Println("[crontab] action=bus&bike event=start")
		bikeEta(c, rc)
		busEta(context.Background(), c, rc, db, dispatcher)
		log.Println("[crontab] action=bus&bike event=end")
	})
	_, _ = r.AddFunc("@every 10s", func() {
		if sleepingwhiledailyupdate() {
			return
		}
		log.Println("[crontab] action=mrt event=start")
		mrtEta(c, rc)
		log.Println("[crontab] action=mrt event=end")
	})
	_, _ = r.AddFunc("@every 10m", func() {
		weatherSync(rc)
	})
	_, _ = r.AddFunc("0 0 4 * * *", func() {
		ctx := context.Background()
		computeTravelAvg(ctx, db)
	})
	_, _ = r.AddFunc("0 30 4 * * *", func() {
		ctx := context.Background()
		cleanupBusHistory(ctx, db)
	})
	r.Start()
	defer r.Stop()
	mqttClient := startMQTT(rc, dispatcher)
	if mqttClient != nil {
		defer mqttClient.Disconnect(500)
	}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("在關了，沒看到嗎？")*/
}

var ingestDB *pgxpool.Pool

var busStaticPrefixes = []string{"bus_StopOfRoute", "bus_Route", "bus_Shape", "bus_Schedule", "bus_Station", "bus_Fare", "bus_Operator", "DailyTimeTable"}

func busStaticCity(name string) (string, bool) {
	for _, p := range busStaticPrefixes {
		if strings.HasPrefix(name, p) {
			return strings.TrimPrefix(name, p), true
		}
	}
	return "", false
}

func dbSince(name string) string {
	if ingestDB == nil {
		return ""
	}
	var q string
	var arg any
	switch {
	case strings.HasPrefix(name, "tra_traindate_"):
		q = "SELECT MAX(updated_at) FROM tra_timetable WHERE train_date=$1"
		arg = strings.TrimPrefix(name, "tra_traindate_")
	case strings.HasPrefix(name, "thsr_traindate_"):
		q = "SELECT MAX(updated_at) FROM thsr_timetable WHERE train_date=$1"
		arg = strings.TrimPrefix(name, "thsr_traindate_")
	case name == "tra_stations":
		q = "SELECT MAX(updated_at) FROM tra_stations"
	case name == "thsr_stations":
		q = "SELECT MAX(updated_at) FROM thsr_stations"
	case name == "tra_fare":
		q = "SELECT MAX(updated_at) FROM tra_fares"
	case name == "thsr_fare":
		q = "SELECT MAX(updated_at) FROM thsr_fares"
	case strings.HasPrefix(name, "mrt_stations"):
		q = "SELECT MAX(updated_at) FROM mrt_station WHERE system=$1"
		arg = strings.TrimPrefix(name, "mrt_stations")
	case strings.HasPrefix(name, "mrt_firstlast"):
		q = "SELECT MAX(updated_at) FROM mrt_schedule WHERE system=$1"
		arg = strings.TrimPrefix(name, "mrt_firstlast")
	case strings.HasPrefix(name, "bike_stations"):
		q = "SELECT MAX(updated_at) FROM bike_stations WHERE city=$1"
		arg = strings.TrimPrefix(name, "bike_stations")
	default:
		city, ok := busStaticCity(name)
		if !ok || city == "" {
			return ""
		}
		q = "SELECT MAX(created_at) FROM raw_bus_route WHERE depart=$1 OR destin=$1"
		arg = city
	}
	ctx := context.Background()
	var t *time.Time
	var err error
	if arg != nil {
		err = ingestDB.QueryRow(ctx, q, arg).Scan(&t)
	} else {
		err = ingestDB.QueryRow(ctx, q).Scan(&t)
	}
	if err != nil || t == nil {
		return ""
	}
	return t.UTC().Format(http.TimeFormat)
}

func callApi(client *resty.Client, rc *redis.Client, url string, name string) (*json.Decoder, bool, error, func()) {
	since, _ := rc.Get("LastTimeGet_" + name).Result()
	if since == "" {
		since = dbSince(name)
	}
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
	if resp.StatusCode() >= 400 {
		_ = resp.RawResponse.Body.Close()
		return &json.Decoder{}, false, fmt.Errorf("tdx %s: status %d", name, resp.StatusCode()), nil
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

/*
	func savebushistory(ctx context.Context, db *pgxpool.Pool, eat []raw_Bus_Esimated, posit []raw_Bus_Position) {
		mp := make(map[string]raw_Bus_Position)
		var rows [][]interface{}
		for _, b := range posit {
			mp[b.PlateNumb] = b
		}
		for _, b := range eat {
			stime, err := time.Parse(time.RFC3339, b.SrcUpdateTime)
			var gps interface{} = nil
			var geom interface{} = nil
			if err != nil {
				stime = time.Now()
			}
			if pos, ok := mp[b.PlateNumb]; ok {
				t, err := time.Parse(time.RFC3339, pos.GPSTime)
				if err != nil {
					t = stime
				}
				gps = t
				geom = fmt.Sprintf("POINT(%.6f %.6f)", pos.BusPosition.PositionLon, pos.BusPosition.PositionLat)
			}
			rows = append(rows, []interface{}{
				b.SubRouteUID,
				b.Direction,
				b.StopUID,
				b.EstimatedTime,
				b.StopStatus,
				b.PlateNumb,
				geom,
				gps,
				stime,
			})
		}
		if len(rows) <= 0 {
			return
		}
		b, err := db.Begin(ctx)
		if err != nil {
			log.Printf("[BUS] Begin tx error: %v", err)
			return
		}
		_, err = b.Exec(ctx, "CREATE TEMP TABLE temp_bus_history (uid text, dir int2, suid text, est int4, sts int2,pl text, geom text,gps timestamptz,stime timestamptz) ON COMMIT DROP;")
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		_, err = b.CopyFrom(ctx, pgx.Identifier{"temp_bus_history"}, []string{"uid", "dir", "suid", "est", "sts", "pl", "geom", "gps", "stime"}, pgx.CopyFromRows(rows))
		if err != nil {
			log.Printf("[BUS] action=savebushistory event=copyfrom_error error=%v row_count=%d", err, len(rows))
			return
		}
		c1 := `INSERT INTO bus_eta_history (
					sub_route_uid,
			 		direction,
					stop_uid,
					estimate_time,
					stop_status,
					plate_numb,
					bus_position,
					gps_time,
				    src_update_time
				)
			SELECT
					uid,
			 		dir,
					suid,
					est,
					sts,
					pl,
					CASE WHEN geom IS NOT NULL THEN ST_GeomFromText(geom, 4326) END,
					gps,
				    stime
			FROM temp_bus_history;`
		_, err = b.Exec(ctx, c1)
		if err != nil {
			log.Printf("[BUS] action=savebushistory event=insert_error error=%v", err)
		}
		if commitErr := b.Commit(ctx); commitErr != nil {
			log.Printf("[BUS] action=savebushistory event=commit_error error=%v", commitErr)
		} else {
			log.Printf("[BUS] action=savebushistory event=complete row_count=%d", len(rows))
		}
	}
*/
func busstaticmp(ctx context.Context, db *pgxpool.Pool, city string) ([]busStationmap, error) {
	query := `SELECT bssm.station_id, bssm.station_name, bssm.sub_route_uid, bssm.route_name,
	                 bssm.direction, bssm.stop_uid, bssm.stop_sequence,
	                 COALESCE(ST_Y(bs.position), 0), COALESCE(ST_X(bs.position), 0)
	          FROM bus_station_stop_map bssm
	          LEFT JOIN bus_stations bs ON bs.station_uid = bssm.station_id
	          WHERE bssm.sub_route_uid LIKE $1`
	rows, err := db.Query(ctx, query, city+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []busStationmap
	for rows.Next() {
		var temp busStationmap
		err := rows.Scan(&temp.StationUID, &temp.StationName, &temp.SubRouteUID,
			&temp.SubRouteName, &temp.Direction, &temp.StopUID, &temp.StopSequence,
			&temp.Lat, &temp.Lon)
		if err != nil {
			fmt.Println(err)
			continue
		}
		list = append(list, temp)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}
func makethatsame(city string, subRouteUID string, Direction uint8) (string, uint8) {
	if city == "InterCity" {
		temp := subRouteUID[len(subRouteUID)-2:]
		if temp == "01" {
			return subRouteUID[:len(subRouteUID)-2], 0
		}
		if temp == "02" {
			return subRouteUID[:len(subRouteUID)-2], 1
		}
		return subRouteUID[:len(subRouteUID)-1], Direction
	}
	return subRouteUID, Direction
}
func connectredis() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:         os.Getenv("REDIS_ADDR"),
		Password:     "",
		DB:           0,
		PoolSize:     20,
		MinIdleConns: 3,
		PoolTimeout:  5 * time.Second,
	})
	pong, err := client.Ping().Result()
	if err != nil {
		log.Printf("[REDIS] action=connect event=failed error=%v", err)
		panic(err)
	}
	log.Printf("[REDIS] action=connect event=success pong=%s", pong)
	return client
}
func connectdb() *pgxpool.Pool {
	config, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Printf("[DB] action=parse_config event=failed error=%v", err)
		panic(err)
	}
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	conn, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Printf("[DB] action=connect event=failed error=%v", err)
		panic(err)
	}
	if err = conn.Ping(context.Background()); err != nil {
		log.Printf("[DB] action=ping event=failed error=%v", err)
		panic(err)
	}
	log.Printf("[DB] action=connect event=success")
	return conn
}
func getToken(rc *redis.Client) string {
	if val, err := rc.Get("TDX_Token").Result(); err == nil && val != "" {
		return val
	}
	authClient := resty.New()
	resp, err := authClient.R().
		SetHeader("content-type", "application/x-www-form-urlencoded").
		SetFormData(map[string]string{
			"grant_type":    "client_credentials",
			"client_id":     os.Getenv("TDX_CLIENT_ID"),
			"client_secret": os.Getenv("TDX_CLIENT_SECRET"),
		}).
		Post("https://tdx.transportdata.tw/auth/realms/TDXConnect/protocol/openid-connect/token")
	if err != nil {
		log.Printf("[TDX] token fetch error: %v", err)
		return ""
	}
	var mp map[string]interface{}
	if err = json.Unmarshal(resp.Body(), &mp); err != nil {
		log.Printf("[TDX] token parse error: %v", err)
		return ""
	}
	token, ok := mp["access_token"].(string)
	if !ok || token == "" {
		log.Printf("[TDX] access_token missing from response")
		return ""
	}
	if err = rc.Set("TDX_Token", token, 6*time.Hour).Err(); err != nil {
		log.Printf("[TDX] token cache error: %v", err)
	}
	return token
}
func processStatic(ctx context.Context, client *resty.Client, rc *redis.Client, db *pgxpool.Pool, city string, api string, processer func([]byte)) {
	var target string
	if city == "InterCity" {
		target = fmt.Sprintf("/v2/Bus/%s/InterCity", api)
	} else {
		target = fmt.Sprintf("/v2/Bus/%s/City/%s", api, city)
	}
	log.Printf("[BUS_STATIC] action=process_static city=%s api=%s event=api_call", city, api)
	cacheKey := "bus_" + api + city
	dec, comp, err, flipopen := callApi(client, rc, target, cacheKey)
	if err == nil && !comp && !hasBusRawStatic(ctx, db, city, api) {
		_ = rc.Del("LastTimeGet_" + cacheKey).Err()
		log.Printf("[BUS_STATIC] action=process_static city=%s api=%s event=retry reason=local_empty", city, api)
		dec, comp, err, flipopen = callApi(client, rc, target, cacheKey)
	}
	if err != nil || !comp {
		log.Printf("[BUS_STATIC] action=process_static city=%s api=%s event=api_skip reason=empty", city, api)
		return
	}
	defer flipopen()
	if _, err := dec.Token(); err != nil {
		log.Printf("[BUS_STATIC] action=process_static city=%s api=%s event=decode_error error=%v", city, api, err)
		return
	}
	var rawRows [][]interface{}
	for dec.More() {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			continue
		}
		processer(raw)
		switch api {
		case "Route":
			var r rawBusRoute
			err := json.Unmarshal(raw, &r)
			if err != nil {
				log.Printf("[BUS_STATIC] action=process_static city=%s api=%s event=unmarshal_error error=%v", city, api, err)
			}
			for _, sub := range r.SubRoutes {
				dep, dest := sub.DepartureStopNameZh, sub.DestinationStopNameZh
				if dep == "" {
					dep = r.DepartureStopNameZh
				}
				if dest == "" {
					dest = r.DestinationStopNameZh
				}
				uid, dir := makethatsame(city, sub.SubRouteUID, sub.Direction)
				rawRows = append(rawRows, []interface{}{
					uid, dir, r.RouteUID, r.RouteName.Zhtw, sub.SubRouteName.Zhtw, dep, dest, api, raw,
				})
			}
		case "StopOfRoute":
			var s rawStopofroute
			err := json.Unmarshal(raw, &s)
			if err != nil {
				fmt.Println(err.Error())
			}
			uid, dir := makethatsame(city, s.SubRouteUID, s.Direction)
			rawRows = append(rawRows, []interface{}{
				uid, dir, s.RouteUID, "", "", "", city, api, raw,
			})
		case "Shape":
			var s rawBusShape
			err := json.Unmarshal(raw, &s)
			if err != nil {
				fmt.Println(err.Error())
			}
			var uid string
			var dir uint8
			if s.SubRouteUID == "" {
				uid, dir = makethatsame(city, s.RouteUID, s.Direction)
			} else {
				uid, dir = makethatsame(city, s.SubRouteUID, s.Direction)
			}
			rawRows = append(rawRows, []interface{}{
				uid, dir, s.RouteUID, "", "", "", city, api, raw,
			})
		case "Schedule":
			var t rawBusSchedule
			err := json.Unmarshal(raw, &t)
			if err != nil {
				fmt.Println(err.Error())
			}
			uid, dir := makethatsame(city, t.SubRouteUID, t.Direction)
			rawRows = append(rawRows, []interface{}{
				uid, dir, t.RouteUID, "", "", "", city, api, raw,
			})
		case "Station":
			var t rawBusStation
			err := json.Unmarshal(raw, &t)
			if err != nil {
				fmt.Println(err.Error())
			}
			rawRows = append(rawRows, []interface{}{
				t.StationUID, -1, t.StationID, t.StationName.Zhtw, "", city, "", api, raw,
			})
		}
	}
	if len(rawRows) > 0 {
		c1 := `CREATE TEMP TABLE temp_bus (
							sub_route_uid  text  not null,
							direction      smallint not null,
							route_uid      text  not null,
							route_name     text,
							sub_route_name text,
							depart         text,
							destin         text,
							type           text  not null,
							content        jsonb not null
				) ON COMMIT DROP;`
		c2 := `INSERT INTO raw_bus_route (
							sub_route_uid,
							direction,
							route_uid,
							route_name,
							sub_route_name,
							depart,
							destin,
							type,
							content,
							created_at
						)
						SELECT DISTINCT ON (sub_route_uid,direction,type) sub_route_uid, direction, route_uid, route_name,sub_route_name, depart,destin,type,content,NOW() FROM temp_bus
						ON CONFLICT (sub_route_uid,direction,type) DO UPDATE SET route_uid = EXCLUDED.route_uid,route_name = excluded.route_name,sub_route_name = EXCLUDED.sub_route_name,depart = excluded.depart,destin = excluded.destin,type = excluded.type,content = excluded.content,created_at = NOW();`
		b, err := db.Begin(ctx)
		if err != nil {
			log.Printf("[MRT] action=process_static city=%s api= %s event=begin_error error=%v", city, api, err)
			return
		}
		defer func(b pgx.Tx, ctx context.Context) {
			_ = b.Rollback(ctx)
		}(b, ctx)
		_, _ = b.Exec(ctx, c1)
		from, err := b.CopyFrom(ctx, pgx.Identifier{"temp_bus"}, []string{"sub_route_uid", "direction", "route_uid", "route_name", "sub_route_name", "depart", "destin", "type", "content"}, pgx.CopyFromRows(rawRows))
		if err == nil {
			if _, execErr := b.Exec(ctx, c2); execErr != nil {
				log.Printf("[BUS_STATIC] action=process_static city=%s api=%s event=exec_error error=%v", city, api, execErr)
			}
			if commitErr := b.Commit(ctx); commitErr != nil {
				log.Printf("[BUS_STATIC] action=process_static city=%s api=%s event=commit_error error=%v", city, api, commitErr)
			} else {
				log.Printf("[BUS_STATIC] action=process_static city=%s api=%s event=success station_count=%d", city, api, from)
			}
		} else {
			log.Printf("[BUS_STATIC] action=process_static city=%s api=%s event=copyfrom_error error=%v", city, api, err)
			_ = b.Rollback(ctx)
		}
	}
}

const busRawStaticExistsSQL = `SELECT COUNT(*) FROM raw_bus_route WHERE type = $1 AND (depart = $2 OR destin = $2)`

func hasBusRawStatic(ctx context.Context, db *pgxpool.Pool, city string, api string) bool {
	var n int
	if err := db.QueryRow(ctx, busRawStaticExistsSQL, api, city).Scan(&n); err != nil {
		log.Printf("[BUS_STATIC] action=raw_exists city=%s api=%s event=query_error error=%v", city, api, err)
		return true
	}
	return n > 0
}

func mask(mon, tues, wed, thur, fri, satur, sun bool, nationalHoliday ...bool) uint8 {
	var res uint8
	days := []bool{mon, tues, wed, thur, fri, satur, sun}
	for i, v := range days {
		if v {
			res |= 1 << i
		}
	}
	if len(nationalHoliday) > 0 && nationalHoliday[0] {
		res |= 1 << 7
	}
	return res
}
func mask2(mon, tues, wed, thur, fri, satur, sun uint8) uint8 {
	var res uint8
	days := []uint8{mon, tues, wed, thur, fri, satur, sun}
	for i, v := range days {
		if v == 1 {
			res |= 1 << i
		}
	}
	return res
}
