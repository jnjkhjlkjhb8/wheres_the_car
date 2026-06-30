package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis"
	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	pb "github.com/jnjkhjlkjhb8/wheres_the_car/models"
	"github.com/jnjkhjlkjhb8/wheres_the_car/server/obs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func makethatsame(subRouteUID string) string {
	if len(subRouteUID) >= 3 && subRouteUID[:3] == "THB" {
		temp := subRouteUID[len(subRouteUID)-2:]
		switch temp {
		case "01", "02":
			return subRouteUID[:len(subRouteUID)-2]
		}
		return subRouteUID[:len(subRouteUID)-1]
	}
	return subRouteUID
}

func busRouteEtaKey(subRouteUID string) string {
	return fmt.Sprintf("bus_eta_route:%s", makethatsame(subRouteUID))
}

func usableBusEtaPayload(data []byte) bool {
	return len(data) > 0
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
	config.MaxConns = 20
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

type BusRouteserver struct {
	pb.UnimplementedBus_Route_ServiceServer
	mu    sync.Mutex
	db    *pgxpool.Pool
	rc    *redis.Client
	cache *ttlCache
}
type BusStationserver struct {
	pb.UnimplementedBus_Station_ServiceServer
	mu sync.Mutex
	db *pgxpool.Pool
	rc *redis.Client
}
type BikeServer struct {
	pb.UnimplementedBike_ServiceServer
	mu    sync.Mutex
	db    *pgxpool.Pool
	rc    *redis.Client
	cache *ttlCache
}
type MrtServer struct {
	pb.UnimplementedMrt_ServiceServer
	mu sync.Mutex
	rc *redis.Client
	db *pgxpool.Pool
}
type ThsrServer struct {
	pb.UnimplementedThsrTimetableServiceServer
	mu     sync.Mutex
	db     *pgxpool.Pool
	client *resty.Client
	rc     *redis.Client
}
type Tra_StationServer struct {
	pb.UnimplementedTRAStationServiceServer
	mu sync.Mutex
	db *pgxpool.Pool
	rc *redis.Client
}
type Tra_TimetableServer struct {
	pb.UnimplementedTRATimetableServiceServer
	mu     sync.Mutex
	db     *pgxpool.Pool
	rc     *redis.Client
	client *resty.Client
}
type Tra_DetainServer struct {
	pb.UnimplementedTRA_DetainServiceServer
	mu     sync.Mutex
	db     *pgxpool.Pool
	rc     *redis.Client
	client *resty.Client
}
type Thsr_DetainServer struct {
	pb.UnimplementedThsr_DetainServiceServer
	mu     sync.Mutex
	db     *pgxpool.Pool
	rc     *redis.Client
	client *resty.Client
}
type Near_Server struct {
	pb.UnimplementedNear_Station_ServiceServer
	mu         sync.Mutex
	db         *pgxpool.Pool
	osrmClient *resty.Client
}

type rateLimiter struct {
	mu       sync.Mutex
	counts   map[string]int
	windowAt time.Time
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		counts:   make(map[string]int, 128),
		windowAt: time.Now(),
	}
}

func (r *rateLimiter) allow(ip string, limit int, window time.Duration) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	if now.Sub(r.windowAt) >= window {
		r.windowAt = now
		for k := range r.counts {
			delete(r.counts, k)
		}
	}
	r.counts[ip]++
	return r.counts[ip] <= limit
}
func rateLimitInterceptor(rl *rateLimiter, limit int, window time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if !allowRequest(ctx, rl, limit, window) {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}
		return handler(ctx, req)
	}
}
func rateLimitStreamInterceptor(rl *rateLimiter, limit int, window time.Duration) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !allowRequest(ss.Context(), rl, limit, window) {
			return status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}
		return handler(srv, ss)
	}
}
func allowRequest(ctx context.Context, rl *rateLimiter, limit int, window time.Duration) bool {
	peerInfo, ok := peer.FromContext(ctx)
	if !ok || peerInfo.Addr == nil {
		return true
	}
	addr := peerInfo.Addr.String()
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	return rl.allow(host, limit, window)
}

func main() {
	defer obs.Init("router")()
	rc := connectredis()
	db := connectdb()
	c := resty.New()
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
	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	rl := newRateLimiter()
	tlsCredentials, err := firebaseTLSCredentialsFromEnv()
	if err != nil {
		log.Fatalf("gRPC TLS initialization failed: %v", err)
	}
	appCheckVerifier, enforceAppCheck, err := firebaseAppCheckFromEnv(context.Background())
	if err != nil {
		log.Fatalf("Firebase Admin initialization failed: %v", err)
	}
	serverOptions := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			obs.UnaryInterceptor(),
			rateLimitInterceptor(rl, 30, time.Second),
			appCheckUnaryInterceptor(appCheckVerifier, enforceAppCheck),
		),
		grpc.ChainStreamInterceptor(
			obs.StreamInterceptor(),
			rateLimitStreamInterceptor(rl, 30, time.Second),
			appCheckStreamInterceptor(appCheckVerifier, enforceAppCheck),
		),
	}
	if tlsCredentials != nil {
		serverOptions = append(serverOptions, grpc.Creds(tlsCredentials))
	}
	grpcServer := grpc.NewServer(serverOptions...)
	pb.RegisterBus_Route_ServiceServer(grpcServer, &BusRouteserver{db: db, rc: rc, cache: newTTLCache()})
	pb.RegisterBus_Station_ServiceServer(grpcServer, &BusStationserver{db: db, rc: rc})
	pb.RegisterBike_ServiceServer(grpcServer, &BikeServer{db: db, rc: rc, cache: newTTLCache()})
	pb.RegisterMrt_ServiceServer(grpcServer, &MrtServer{db: db, rc: rc})
	pb.RegisterThsrTimetableServiceServer(grpcServer, &ThsrServer{db: db, client: c, rc: rc})
	pb.RegisterTRAStationServiceServer(grpcServer, &Tra_StationServer{db: db, rc: rc})
	pb.RegisterTRATimetableServiceServer(grpcServer, &Tra_TimetableServer{db: db, client: c, rc: rc})
	pb.RegisterTRA_DetainServiceServer(grpcServer, &Tra_DetainServer{db: db, client: c, rc: rc})
	pb.RegisterThsr_DetainServiceServer(grpcServer, &Thsr_DetainServer{db: db, client: c, rc: rc})
	pb.RegisterNear_Station_ServiceServer(grpcServer, &Near_Server{db: db, osrmClient: resty.New().SetTimeout(5 * time.Second)})
	pb.RegisterAlert_ServiceServer(grpcServer, &AlertServer{rc: rc})
	pb.RegisterMaasServiceServer(grpcServer, newMaasServer(rc, db, func() string { return getToken(rc) }))
	pb.RegisterFirebase_ServiceServer(grpcServer, &FirebaseServer{store: newFirebaseStore(db), now: time.Now})
	go startHTTPServer(db)
	log.Printf("gRPC server is running on port %d", 50051)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func (s *BusRouteserver) Static(ctx context.Context, in *pb.Bus_Ask_Route) (*pb.Resp_BusStatic, error) {
	return s.BusRouteStatic(ctx, in)
}

func (s *BusRouteserver) Daily(_ context.Context, in *pb.Bus_Ask_Route) (*pb.Resp_BusDailyTimetable, error) {
	resp, err := s.BusDailytable(in)
	if err != nil {
		return nil, err
	}
	return &pb.Resp_BusDailyTimetable{Data: resp.Data}, nil
}

func (s *BusRouteserver) Eta(in *pb.Bus_Ask_Route, stream pb.Bus_Route_Service_EtaServer) error {
	return s.BusRouteEta(in, stream)
}

func (s *BusRouteserver) Fare(ctx context.Context, in *pb.Bus_Ask_Route) (*pb.Bus_Fare, error) {
	log.Printf("[gRPC] action=bus_fare uid=%s", in.SubRouteUID)
	key := fmt.Sprintf("bus_fare:%s", in.SubRouteUID)
	if val, err := s.rc.Get(key).Bytes(); err == nil {
		var fare pb.Bus_Fare
		if err := proto.Unmarshal(val, &fare); err == nil {
			return &fare, nil
		}
	}
	var type2 int16
	var free bool
	var select2 []byte
	err := s.db.QueryRow(ctx,
		`SELECT fare_pricing_type, is_free_bus, section_fares FROM bus_fares WHERE sub_route_uid = $1`,
		in.SubRouteUID,
	).Scan(&type2, &free, &select2)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "no fare data for %s", in.SubRouteUID)
	}
	resp := &pb.Bus_Fare{
		SubRouteUid:      in.SubRouteUID,
		FarePricingType:  int32(type2),
		IsFreeBus:        free,
		SectionFaresJson: select2,
	}
	if data, err := proto.Marshal(resp); err == nil {
		s.rc.Set(key, data, 24*time.Hour)
	}
	return resp, nil
}

func (s *BusStationserver) Eta(in *pb.Bus_Ask_Route, stream pb.Bus_Station_Service_EtaServer) error {
	return (&BusRouteserver{db: s.db, rc: s.rc}).BusStationEta(in, stream)
}

func (s *BusStationserver) Group(ctx context.Context, in *pb.Bus_Ask_Route) (*pb.Bus_StationGroup, error) {
	groupUID := in.SubRouteUID
	if groupUID == "" {
		return nil, status.Error(codes.InvalidArgument, "group_uid required")
	}
	resp := &pb.Bus_StationGroup{GroupUid: groupUID}
	err := s.db.QueryRow(ctx, `
		SELECT group_name, city, ST_X(position), ST_Y(position)
		FROM bus_station_groups
		WHERE group_uid = $1`, groupUID).Scan(&resp.GroupName, &resp.City, &resp.PositionLon, &resp.PositionLat)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "station group not found: %s", groupUID)
	}
	rows, err := s.db.Query(ctx, `
		SELECT station_uid, station_id, station_name, ST_X(position), ST_Y(position)
		FROM bus_station_group_members
		WHERE group_uid = $1
		ORDER BY station_id, station_uid`, groupUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "station group members: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		m := &pb.Bus_StationGroupMember{}
		if err := rows.Scan(&m.StationUid, &m.StationId, &m.StationName, &m.PositionLon, &m.PositionLat); err != nil {
			return nil, status.Errorf(codes.Internal, "scan station group member: %v", err)
		}
		resp.Members = append(resp.Members, m)
	}
	return resp, nil
}

func (s *BikeServer) Static(ctx context.Context, in *pb.BikeRequest) (*pb.BikeStatic, error) {
	return s.BikeStatic(ctx, in)
}

func (s *BikeServer) Eta(in *pb.BikeRequest, stream pb.Bike_Service_EtaServer) error {
	return s.bikeEta(in, stream)
}

func (s *Tra_StationServer) LiveBoard(in *pb.AskStaiton, stream pb.TRAStationService_LiveBoardServer) error {
	return s.traLiveboard(in, stream)
}

func (s *Tra_TimetableServer) Delay(_ *pb.AskRoute, stream pb.TRATimetableService_DelayServer) error {
	return s.traDelay(stream)
}

func (s *Tra_TimetableServer) Fare(ctx context.Context, in *pb.AskStaiton) (*pb.TraFareItem, error) {
	if in.StationId == "" || in.Date == "" {
		return nil, status.Error(codes.InvalidArgument, "origin and destination are required")
	}
	req := &pb.AskRoute{
		OriginStationId:      in.StationId,
		DestinationStationId: in.Date,
	}
	resp, err := s.traFare(ctx, req)
	if err != nil {
		return nil, err
	}
	items := &pb.TraFareItems{}
	if err := proto.Unmarshal(resp.Data, items); err != nil {
		return nil, status.Errorf(codes.Internal, "decode fare: %v", err)
	}
	if len(items.Items) == 0 {
		return nil, status.Error(codes.NotFound, "fare not found")
	}
	return items.Items[0], nil
}

func (s *Tra_TimetableServer) Timetable(ctx context.Context, in *pb.AskRoute) (*pb.TraTimetables, error) {
	resp, err := s.traTimetable(ctx, in)
	if err != nil {
		return nil, err
	}
	items := &pb.TraTimetables{}
	if err := proto.Unmarshal(resp.Data, items); err != nil {
		return nil, status.Errorf(codes.Internal, "decode timetable: %v", err)
	}
	if len(items.Items) == 0 {
		return nil, status.Error(codes.NotFound, "timetable not found")
	}
	return items, nil
}

func (s *Tra_DetainServer) Delay(in *pb.AskDetain, stream pb.TRA_DetainService_DelayServer) error {
	return s.traDdelay(in, stream)
}

func (s *Tra_DetainServer) Stops(ctx context.Context, in *pb.AskDetain) (*pb.TraStoptimes, error) {
	resp, err := s.traStops(ctx, in)
	if err != nil {
		return nil, err
	}
	items := &pb.TraStoptimes{}
	if err := proto.Unmarshal(resp.Data, items); err != nil {
		return nil, status.Errorf(codes.Internal, "decode stops: %v", err)
	}
	return items, nil
}

func (s *ThsrServer) Fare(ctx context.Context, in *pb.Ask_Thsr) (*pb.ThsaFare, error) {
	req := &pb.AskRoute{
		OriginStationId:      in.OriginStationId,
		DestinationStationId: in.DestinationStationId,
	}
	resp, err := s.thsrFare(ctx, req)
	if err != nil {
		return nil, err
	}
	items := &pb.ThsaFares{}
	if err := proto.Unmarshal(resp.Data, items); err != nil {
		return nil, status.Errorf(codes.Internal, "decode fare: %v", err)
	}
	if len(items.Items) == 0 {
		return nil, status.Error(codes.NotFound, "fare not found")
	}
	return items.Items[0], nil
}

func (s *ThsrServer) AvailableSeats(in *pb.Ask_Thsr, stream grpc.ServerStreamingServer[pb.RespThsrSeats]) error {
	log.Printf("[gRPC] action=thsr_available_seats date=%s", in.Date)
	channel := fmt.Sprintf("thsr_seats:%s:*", in.Date)
	var cursor uint64
	for {
		if err := stream.Context().Err(); err != nil {
			return err
		}
		keys, next, err := s.rc.Scan(cursor, channel, 20).Result()
		if err != nil {
			break
		}
		for _, i := range keys {
			val, err := s.rc.Get(i).Bytes()
			if err != nil {
				continue
			}
			resp := &pb.RespThsrSeats{Data: val}
			if err = stream.Send(resp); err != nil {
				return err
			}
		}
		cursor = next
		if cursor == 0 {
			t, _ := time.Parse(time.RFC3339, in.Date)
			get_thsr_availableseatstatus(s.client, s.rc, t.Format(time.DateOnly))
			break
		}
	}
	sub := s.rc.PSubscribe(fmt.Sprintf("thsr_seats:%s:*", in.Date))
	defer func(sub *redis.PubSub) { _ = sub.Close() }(sub)
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
		}
		m, err := sub.ReceiveMessage()
		if err != nil {
			return err
		}
		resp := &pb.RespThsrSeats{Data: []byte(m.Payload)}
		if err = stream.Send(resp); err != nil {
			return err
		}
	}
}

func (s *ThsrServer) Timetable(ctx context.Context, in *pb.Ask_Thsr) (*pb.ThsrTimetables, error) {
	req := &pb.AskRoute{
		Date:                 in.Date,
		OriginStationId:      in.OriginStationId,
		DestinationStationId: in.DestinationStationId,
	}
	resp, err := s.thsrTimetable(ctx, req)
	if err != nil {
		return nil, err
	}
	items := &pb.ThsrTimetables{}
	if err := proto.Unmarshal(resp.Data, items); err != nil {
		return nil, status.Errorf(codes.Internal, "decode timetable: %v", err)
	}
	return items, nil
}

func (s *Thsr_DetainServer) Stops(ctx context.Context, in *pb.ThsrAskDetain) (*pb.ThsrStoptimes, error) {
	resp, err := s.thsrStops(ctx, in)
	if err != nil {
		return nil, err
	}
	items := &pb.ThsrStoptimes{}
	if err := proto.Unmarshal(resp.Data, items); err != nil {
		return nil, status.Errorf(codes.Internal, "decode stops: %v", err)
	}
	return items, nil
}

func (s *Near_Server) Near(stream pb.Near_Station_Service_NearServer) error {
	return s.FindNear(stream)
}
func (s *BusRouteserver) BusRouteStatic(ctx context.Context, in *pb.Bus_Ask_Route) (*pb.Resp_BusStatic, error) {
	log.Printf("call bus_static %s", in.SubRouteUID)
	route := makethatsame(in.SubRouteUID)
	if s.cache != nil {
		if data, ok := s.cache.get("bus_static:" + route); ok {
			return &pb.Resp_BusStatic{Data: data}, nil
		}
	}
	var data []byte
	err := s.db.QueryRow(ctx, `SELECT pb FROM bus_static WHERE sub_route_uid = $1;`, route).Scan(&data)
	if err != nil {
		log.Printf("[gRPC] action=bus_static event=query_failed error=%v", err)
		return nil, err
	}
	if s.cache != nil {
		s.cache.set("bus_static:"+route, data, time.Hour)
	}
	return &pb.Resp_BusStatic{Data: data}, nil
}
func (s *BusRouteserver) BusRouteEta(in *pb.Bus_Ask_Route, stream pb.Bus_Route_Service_EtaServer) error {
	log.Printf("call Bus_route_eta %s", in.SubRouteUID)
	key := busRouteEtaKey(in.SubRouteUID)
	sub := s.rc.Subscribe(key)
	if val, err := s.rc.Get(key).Bytes(); err == nil && usableBusEtaPayload(val) {
		resp := &pb.Resp_BusEta{Data: val}
		if err := stream.Send(resp); err != nil {
			log.Printf("[gRPC] action=bus_route_eta event=send_failed error=%v", err)
			return err
		}
	}
	defer func(sub *redis.PubSub) {
		_ = sub.Close()
	}(sub)
	ch := sub.Channel()
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case val := <-ch:
			data := []byte(val.Payload)
			if !usableBusEtaPayload(data) {
				continue
			}
			resp := &pb.Resp_BusEta{Data: data}
			if err := stream.Send(resp); err != nil {
				log.Printf("[gRPC] action=bus_route_eta event=send_failed error=%v", err)
				return err
			}
		}
	}
}
func (s *BusRouteserver) BusStationEta(in *pb.Bus_Ask_Route, stream pb.Bus_Station_Service_EtaServer) error {
	log.Printf("call Bus_station_eta %s", in.SubRouteUID)
	city, groupUID, ok := strings.Cut(in.SubRouteUID, ":")
	if !ok {
		groupUID = in.SubRouteUID
	}
	if groupUID == "" {
		return status.Error(codes.InvalidArgument, "station eta key must include group_uid")
	}
	if s.db != nil {
		var dbCity string
		if err := s.db.QueryRow(stream.Context(), `SELECT city FROM bus_station_groups WHERE group_uid = $1`, groupUID).Scan(&dbCity); err == nil {
			city = dbCity
		}
	}
	if city == "" {
		return status.Error(codes.InvalidArgument, "station eta key must include city or known group_uid")
	}
	key := fmt.Sprintf("bus_eta_station:%s:%s", city, groupUID)
	sub := s.rc.Subscribe(key)
	if val, err := s.rc.Get(key).Bytes(); err == nil && usableBusEtaPayload(val) {
		resp := &pb.Resp_BusEta{Data: val}
		if err := stream.Send(resp); err != nil {
			log.Printf("[gRPC] action=bus_station_eta event=send_failed error=%v", err)
			return err
		}
	}
	defer func(sub *redis.PubSub) {
		_ = sub.Close()
	}(sub)
	ch := sub.Channel()
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case val := <-ch:
			data := []byte(val.Payload)
			if !usableBusEtaPayload(data) {
				continue
			}
			resp := &pb.Resp_BusEta{Data: data}
			if err := stream.Send(resp); err != nil {
				log.Printf("[gRPC] action=bus_station_eta event=send_failed error=%v", err)
				return err
			}
		}
	}
}
func (s *BusRouteserver) BusDailytable(in *pb.Bus_Ask_Route) (*pb.Resp_BusEta, error) {
	log.Printf("call Bus_route_eta %s", in.SubRouteUID)
	route := makethatsame(in.SubRouteUID)
	val, err := s.rc.Get(fmt.Sprintf("bus_daily_timetable:%s", route)).Result()
	if err != nil {
		log.Printf("[gRPC] action=bus_dailytable event=query_failed error=%v", err)
		return nil, err
	}
	resp := &pb.Resp_BusEta{
		Data: []byte(val),
	}
	return resp, nil
}
func (s *MrtServer) MrtEta(in *pb.AskMrt, stream pb.Mrt_Service_EtaServer) error {
	log.Printf("call Mrt_eta %s %s", in.System, in.StationID)
	channel := fmt.Sprintf("mrt_live:%s:%s", in.System, in.StationID)
	sub := s.rc.Subscribe(channel)
	defer func(sub *redis.PubSub) { _ = sub.Close() }(sub)
	key := fmt.Sprintf("mrt_live:%s:%s:*", in.System, in.StationID)
	var cursor uint64
	for {
		keys, next, err := s.rc.Scan(cursor, key, 20).Result()
		if err != nil {
			break
		}
		for _, i := range keys {
			val := s.rc.Get(i)
			if val.Err() != nil {
				continue
			}
			resp := &pb.Resp_MrtEta{Data: []byte(val.Val())}
			if err := stream.Send(resp); err != nil {
				log.Printf("[gRPC] action=mrt_live event=send_failed error=%v", err)
				return err
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	ch := sub.Channel()
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case val := <-ch:
			resp := &pb.Resp_MrtEta{Data: []byte(val.Payload)}
			if err := stream.Send(resp); err != nil {
				log.Printf("[gRPC] action=mrt_eta event=send_failed error=%v", err)
				return err
			}
		}
	}
}
func (s *BikeServer) BikeStatic(ctx context.Context, in *pb.BikeRequest) (*pb.BikeStatic, error) {
	log.Printf("call bike_static %s", in.StationUID)
	if s.cache != nil {
		if data, ok := s.cache.get("bike_static:" + in.StationUID); ok {
			var resp pb.BikeStatic
			if proto.Unmarshal(data, &resp) == nil {
				return &resp, nil
			}
		}
	}
	var name, address string
	var capacity, serviceType int32
	err := s.db.QueryRow(ctx, `SELECT name,capacity,service_type,address FROM bike_stations WHERE station_uid = $1;`, in.StationUID).Scan(&name, &capacity, &serviceType, &address)
	if err != nil {
		log.Printf("[gRPC] action=bike_static event=query_failed error=%v", err)
		return nil, err
	}
	resp := &pb.BikeStatic{
		StationUID:  in.StationUID,
		Name:        name,
		Capacity:    capacity,
		ServiceType: serviceType,
		Address:     address,
	}
	if s.cache != nil {
		if data, err := proto.Marshal(resp); err == nil {
			s.cache.set("bike_static:"+in.StationUID, data, time.Hour)
		}
	}
	return resp, nil
}
func (s *BikeServer) bikeEta(in *pb.BikeRequest, stream pb.Bike_Service_EtaServer) error {
	log.Printf("call bike_eta %s", in.StationUID)
	sub := s.rc.Subscribe(fmt.Sprintf("bike_availability:%s", in.StationUID))
	val := s.rc.Get(fmt.Sprintf("bike_availability:%s", in.StationUID))
	resp := &pb.Resp_BikeEta{
		Data: []byte(val.Val()),
	}
	if err := stream.Send(resp); err != nil {
		log.Printf("[gRPC] action=bike_eta event=send_failed error=%v", err)
		return err
	}
	defer func(sub *redis.PubSub) {
		_ = sub.Close()
	}(sub)
	ch := sub.Channel()
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case val := <-ch:
			resp := &pb.Resp_BikeEta{Data: []byte(val.Payload)}
			if err := stream.Send(resp); err != nil {
				log.Printf("[gRPC] action=bike_eta event=send_failed error=%v", err)
				return err
			}
		}
	}
}
func (s *Tra_StationServer) traLiveboard(in *pb.AskStaiton, stream pb.TRAStationService_LiveBoardServer) error {
	log.Printf("call tra_liveboard %s", in.StationId)
	sub := s.rc.Subscribe(fmt.Sprintf("tra:liveboard:%s", in.StationId))
	val := s.rc.Get(fmt.Sprintf("tra:liveboard:%s", in.StationId))
	resp := &pb.RespTraLiveBoard{
		Data: []byte(val.Val()),
	}
	if err := stream.Send(resp); err != nil {
		log.Printf("[gRPC] action=tra_liveboard event=send_failed error=%v", err)
		return err
	}
	defer func(sub *redis.PubSub) {
		_ = sub.Close()
	}(sub)
	ch := sub.Channel()
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case val := <-ch:
			resp := &pb.RespTraLiveBoard{Data: []byte(val.Payload)}
			if err := stream.Send(resp); err != nil {
				log.Printf("[gRPC] action=tra_liveboard event=send_failed error=%v", err)
				return err
			}
		}
	}
}
func (s *Tra_TimetableServer) traDelay(stream pb.TRATimetableService_DelayServer) error {
	log.Printf("call tra_delay")
	sub := s.rc.Subscribe("tra:delay:all")
	val := s.rc.Get(fmt.Sprintf("tra:delay:all"))
	resp := &pb.RespTraDelay{
		Data: []byte(val.Val()),
	}
	if err := stream.Send(resp); err != nil {
		log.Printf("[gRPC] action=tra_delay event=send_failed error=%v", err)
		return err
	}
	defer func(sub *redis.PubSub) {
		_ = sub.Close()
	}(sub)
	ch := sub.Channel()
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case val := <-ch:
			resp := &pb.RespTraDelay{Data: []byte(val.Payload)}
			if err := stream.Send(resp); err != nil {
				log.Printf("[gRPC] action=tra_delay event=send_failed error=%v", err)
				return err
			}
		}
	}
}
func (s *Tra_DetainServer) traDdelay(in *pb.AskDetain, stream pb.TRA_DetainService_DelayServer) error {
	log.Printf("call tra_delay %s", in.Trainno)
	sub := s.rc.Subscribe(fmt.Sprintf("tra:delay:%s", in.Trainno))
	val := s.rc.Get(fmt.Sprintf("tra:delay:%s", in.Trainno))
	resp := &pb.RespTraDelay{
		Data: []byte(val.Val()),
	}
	if err := stream.Send(resp); err != nil {
		log.Printf("[gRPC] action=tra_delay event=send_failed error=%v", err)
		return err
	}
	defer func(sub *redis.PubSub) {
		_ = sub.Close()
	}(sub)
	ch := sub.Channel()
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case val := <-ch:
			resp := &pb.RespTraDelay{Data: []byte(val.Payload)}
			if err := stream.Send(resp); err != nil {
				log.Printf("[gRPC] action=tra_delay event=send_failed error=%v", err)
				return err
			}
		}
	}
}
func (s *Tra_TimetableServer) traFare(ctx context.Context, in *pb.AskRoute) (*pb.Resp_Data, error) {
	log.Printf("call tra fare")
	val := s.rc.Get(fmt.Sprintf("TRA_Fare:%s:%s", in.OriginStationId, in.DestinationStationId))
	if errors.Is(val.Err(), redis.Nil) {
		tra_price(ctx, in.OriginStationId, in.DestinationStationId, s.client, s.db, s.rc)
		val = s.rc.Get(fmt.Sprintf("TRA_Fare:%s:%s", in.OriginStationId, in.DestinationStationId))
	}
	if val.Err() != nil {
		log.Printf("[gRPC] action=tra_fare event=query_failed error=%v", val.Err())
		return nil, val.Err()
	}
	res, _ := val.Result()
	resp := &pb.Resp_Data{
		Data: []byte(res),
	}
	return resp, nil
}
func (s *ThsrServer) thsrFare(ctx context.Context, in *pb.AskRoute) (*pb.Resp_Data, error) {
	log.Printf("call thsr fare")
	val := s.rc.Get(fmt.Sprintf("THSR_Fare:%s:%s", in.OriginStationId, in.DestinationStationId))
	if errors.Is(val.Err(), redis.Nil) {
		thsr_price(ctx, in.OriginStationId, in.DestinationStationId, s.client, s.db, s.rc)
		val = s.rc.Get(fmt.Sprintf("THSR_Fare:%s:%s", in.OriginStationId, in.DestinationStationId))
	}
	if val.Err() != nil {
		log.Printf("[gRPC] action=thsr_fare event=query_failed error=%v", val.Err())
		return nil, val.Err()
	}
	res, _ := val.Result()
	resp := &pb.Resp_Data{
		Data: []byte(res),
	}
	return resp, nil
}
func (s *Tra_TimetableServer) traTimetable(ctx context.Context, in *pb.AskRoute) (*pb.Resp_Data, error) {
	log.Printf("call tra timetable %s %s", in.OriginStationId, in.DestinationStationId)
	da, _ := time.Parse(time.RFC3339, in.Date)
	val := s.rc.Get(fmt.Sprintf("TRA_timetable:%s:%s:%s", da.Format(time.DateOnly), in.OriginStationId, in.DestinationStationId))
	if errors.Is(val.Err(), redis.Nil) {
		t, _ := time.Parse(time.RFC3339, in.Date)
		tra_timetable(ctx, in.OriginStationId, in.DestinationStationId, t, s.client, s.db, s.rc)
		val = s.rc.Get(fmt.Sprintf("TRA_timetable:%s:%s:%s", da.Format(time.DateOnly), in.OriginStationId, in.DestinationStationId))
	}
	if val.Err() != nil {
		log.Printf("[gRPC] action=tra_timetable event=query_failed error=%v", val.Err())
		return nil, val.Err()
	}
	res, _ := val.Result()
	resp := &pb.Resp_Data{
		Data: []byte(res),
	}
	return resp, nil
}
func (s *ThsrServer) thsrTimetable(ctx context.Context, in *pb.AskRoute) (*pb.Resp_Data, error) {
	log.Printf("call thsr timetable %s %s", in.OriginStationId, in.DestinationStationId)
	val := s.rc.Get(fmt.Sprintf("THSR_timetable:%s:%s:%s", in.Date, in.OriginStationId, in.DestinationStationId))
	if errors.Is(val.Err(), redis.Nil) {
		t, _ := time.Parse(time.RFC3339, in.Date)
		thsr_timetable(ctx, in.OriginStationId, in.DestinationStationId, t, s.client, s.db, s.rc)
		val = s.rc.Get(fmt.Sprintf("THSR_timetable:%s:%s:%s", in.Date, in.OriginStationId, in.DestinationStationId))
	}
	if val.Err() != nil {
		log.Printf("[gRPC] action=thsr_timetable event=query_failed error=%v", val.Err())
		return nil, val.Err()
	}
	res, _ := val.Result()
	resp := &pb.Resp_Data{
		Data: []byte(res),
	}
	return resp, nil
}
func (s *Tra_DetainServer) traStops(ctx context.Context, in *pb.AskDetain) (*pb.Resp_Data, error) {
	log.Printf("call tra stops %s", in.Trainno)
	val := s.rc.Get(fmt.Sprintf("TRA_Stoptimes:%s:%s", in.Date, in.Trainno))
	if errors.Is(val.Err(), redis.Nil) {
		t, _ := time.Parse(time.RFC3339, in.Date)
		tra_stoptimes(ctx, in.Trainno, t, s.client, s.db, s.rc)
		val = s.rc.Get(fmt.Sprintf("TRA_Stoptimes:%s:%s", in.Date, in.Trainno))
	}
	if val.Err() != nil {
		log.Printf("[gRPC] action=stops event=query_failed error=%v", val.Err())
		return nil, val.Err()
	}
	res, _ := val.Result()
	resp := &pb.Resp_Data{
		Data: []byte(res),
	}
	return resp, nil
}
func (s *Thsr_DetainServer) thsrStops(ctx context.Context, in *pb.ThsrAskDetain) (*pb.Resp_Data, error) {
	log.Printf("call thsr stops %s", in.Trainno)
	val := s.rc.Get(fmt.Sprintf("THSR_Stoptimes:%s:%s", in.Date, in.Trainno))
	if errors.Is(val.Err(), redis.Nil) {
		t, _ := time.Parse(time.RFC3339, in.Date)
		thsr_stoptimes(ctx, in.Trainno, t, s.client, s.db, s.rc)
		val = s.rc.Get(fmt.Sprintf("THSR_Stoptimes:%s:%s", in.Date, in.Trainno))
	}
	if val.Err() != nil {
		log.Printf("[gRPC] action=stops event=query_failed error=%v", val.Err())
		return nil, val.Err()
	}
	res, _ := val.Result()
	resp := &pb.Resp_Data{
		Data: []byte(res),
	}
	return resp, nil
}
func (s *Near_Server) FindNear(stream pb.Near_Station_Service_NearServer) error {
	ctx := stream.Context()
	var l1, l2 float64
	var l3 int
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		lon := in.PositionLon
		lat := in.PositionLat
		r := in.Radius
		l1 = lon
		l2 = lat
		l3 = int(r)
		log.Printf("[gRPC] received location: lon=%f lat=%f radius=%d", lon, lat, r)
		resp, err := findnearstation(l2, l1, l3, ctx, s.db, s.osrmClient)
		if err != nil {
			log.Printf("[gRPC] action=findnearstation failed error=%v", err)
			return err
		}
		if err := stream.Send(resp); err != nil {
			log.Printf("[gRPC] action=send_newdata failed error=%v", err)
			return err
		}
		log.Printf("[gRPC] action=send_newdata event=success")
	}
}
