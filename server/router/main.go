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
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type BusRouteserver struct {
	pb.UnimplementedBus_Route_ServiceServer
	mu sync.Mutex
	db *pgxpool.Pool
	rc *redis.Client
}
type BusStationserver struct {
	pb.UnimplementedBus_Station_ServiceServer
	mu sync.Mutex
	rc *redis.Client
}
type BikeServer struct {
	pb.UnimplementedBike_ServiceServer
	mu sync.Mutex
	db *pgxpool.Pool
	rc *redis.Client
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
	mu sync.Mutex
	db *pgxpool.Pool
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
	c.SetAuthToken(getToken(rc, c)).
		SetBaseURL("https://tdx.transportdata.tw/api/basic").
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
				return r.StatusCode() == 429
			},
		)
	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	rl := newRateLimiter()
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(rateLimitInterceptor(rl, 2, time.Second)),
		grpc.StreamInterceptor(rateLimitStreamInterceptor(rl, 2, time.Second)),
	)
	pb.RegisterBus_Route_ServiceServer(grpcServer, &BusRouteserver{db: db, rc: rc})
	pb.RegisterBus_Station_ServiceServer(grpcServer, &BusStationserver{rc: rc})
	pb.RegisterBike_ServiceServer(grpcServer, &BikeServer{db: db, rc: rc})
	pb.RegisterMrt_ServiceServer(grpcServer, &MrtServer{db: db, rc: rc})
	pb.RegisterThsrTimetableServiceServer(grpcServer, &ThsrServer{db: db, client: c, rc: rc})
	pb.RegisterTRAStationServiceServer(grpcServer, &Tra_StationServer{db: db, rc: rc})
	pb.RegisterTRATimetableServiceServer(grpcServer, &Tra_TimetableServer{db: db, client: c, rc: rc})
	pb.RegisterTRA_DetainServiceServer(grpcServer, &Tra_DetainServer{db: db, client: c, rc: rc})
	pb.RegisterThsr_DetainServiceServer(grpcServer, &Thsr_DetainServer{db: db, client: c, rc: rc})
	pb.RegisterNear_Station_ServiceServer(grpcServer, &Near_Server{db: db})
	pb.RegisterAlert_ServiceServer(grpcServer, &AlertServer{rc: rc})
	go startHTTPServer()
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

func (s *BusStationserver) Eta(in *pb.Bus_Ask_Route, stream pb.Bus_Station_Service_EtaServer) error {
	city := ""
	stationName := in.SubRouteUID
	if parts := strings.SplitN(in.SubRouteUID, ":", 2); len(parts) == 2 {
		city = parts[0]
		stationName = parts[1]
	}
	return (&BusRouteserver{rc: s.rc}).BusStationEta(&pb.Bus_Ask_Station{
		StationName: stationName,
		City:        city,
	}, stream)
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
	resp, err := s.traFare(req, ctx)
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
	resp, err := s.traTimetable(in, ctx)
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
	resp, err := s.traStops(in, ctx)
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
	resp, err := s.thsrFare(req, ctx)
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

func (s *ThsrServer) Timetable(ctx context.Context, in *pb.Ask_Thsr) (*pb.ThsrTimetables, error) {
	req := &pb.AskRoute{
		Date:                 in.Date,
		OriginStationId:      in.OriginStationId,
		DestinationStationId: in.DestinationStationId,
	}
	resp, err := s.thsrTimetable(req, ctx)
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
	resp, err := s.thsrStops(in, ctx)
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
	var data []byte
	err := s.db.QueryRow(ctx, `SELECT pb FROM bus_static where sub_route_uid like $1;`, route).Scan(&data)
	if err != nil {
		log.Printf("[gRPC] action=bus_static event=query_failed error=%v", err)
		return nil, err
	}
	resp := &pb.Resp_BusStatic{
		Data: data,
	}
	return resp, nil
}
func (s *BusRouteserver) BusRouteEta(in *pb.Bus_Ask_Route, stream pb.Bus_Route_Service_EtaServer) error {
	log.Printf("call Bus_route_eta %s", in.SubRouteUID)
	route := makethatsame(in.SubRouteUID)
	sub := s.rc.Subscribe(fmt.Sprintf("bus_eta_route:%s", route))
	val := s.rc.Get(fmt.Sprintf("bus_eta_route:%s", route))
	resp := &pb.Resp_BusEta{
		Data: []byte(val.Val()),
	}
	if err := stream.Send(resp); err != nil {
		log.Printf("[gRPC] action=bus_route_eta event=send_failed error=%v", err)
		return err
	}
	defer func(sub *redis.PubSub) {
		_ = sub.Close()
	}(sub)
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
		}
		val, err := sub.ReceiveMessage()
		if err != nil {
			log.Printf("[gRPC] action=bus_route_eta event=query_failed error=%v", err)
			return err
		}
		resp := &pb.Resp_BusEta{
			Data: []byte(val.Payload),
		}
		if err := stream.Send(resp); err != nil {
			log.Printf("[gRPC] action=bus_route_eta event=send_failed error=%v", err)
			return err
		}
	}
}
func (s *BusRouteserver) BusStationEta(in *pb.Bus_Ask_Station, stream pb.Bus_Station_Service_EtaServer) error {
	log.Printf("call Bus_station_eta %s", in.StationName)
	sub := s.rc.Subscribe(fmt.Sprintf("bus_eta_station:%s:%s", in.City, in.StationName))
	val := s.rc.Get(fmt.Sprintf("bus_eta_station:%s:%s", in.City, in.StationName))
	resp := &pb.Resp_BusEta{
		Data: []byte(val.Val()),
	}
	if err := stream.Send(resp); err != nil {
		log.Printf("[gRPC] action=bus_station_eta event=send_failed error=%v", err)
		return err
	}
	defer func(sub *redis.PubSub) {
		_ = sub.Close()
	}(sub)
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
		}
		val, err := sub.ReceiveMessage()
		if err != nil {
			log.Printf("[gRPC] action=bus_station_eta event=query_failed error=%v", err)
			return err
		}
		resp := &pb.Resp_BusEta{
			Data: []byte(val.Payload),
		}
		if err := stream.Send(resp); err != nil {
			log.Printf("[gRPC] action=bus_station_eta event=send_failed error=%v", err)
			return err
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
	sub := s.rc.Subscribe(fmt.Sprintf("mrt_live:%s:%s", in.System, in.StationID))
	val := s.rc.Get(fmt.Sprintf("mrt_live:%s:%s", in.System, in.StationID))
	resp := &pb.Resp_MrtEta{
		Data: []byte(val.Val()),
	}
	if err := stream.Send(resp); err != nil {
		log.Printf("[gRPC] action=mrt_live event=send_failed error=%v", err)
		return err
	}
	defer func(sub *redis.PubSub) {
		_ = sub.Close()
	}(sub)
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
		}
		val, err := sub.ReceiveMessage()
		if err != nil {
			log.Printf("[gRPC] action=mrt_eta event=query_failed error=%v", err)
			return err
		}
		resp := &pb.Resp_MrtEta{
			Data: []byte(val.Payload),
		}
		if err := stream.Send(resp); err != nil {
			log.Printf("[gRPC] action=mrt_eta event=send_failed error=%v", err)
			return err
		}
	}
}
func (s *BikeServer) BikeStatic(ctx context.Context, in *pb.BikeRequest) (*pb.BikeStatic, error) {
	log.Printf("call bike_static %s", in.StationUID)
	var name, address string
	var capacity, serviceType int32
	err := s.db.QueryRow(ctx, `SELECT name,capacity,service_type,address FROM bike_stations where station_uid like $1;`, in.StationUID).Scan(&name, &capacity, &serviceType, &address)
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
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
		}
		val, err := sub.ReceiveMessage()
		if err != nil {
			log.Printf("[gRPC] action=bike_eta event=query_failed error=%v", err)
			return err
		}
		resp := &pb.Resp_BikeEta{
			Data: []byte(val.Payload),
		}
		if err := stream.Send(resp); err != nil {
			log.Printf("[gRPC] action=bike_eta event=send_failed error=%v", err)
			return err
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
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
		}
		val, err := sub.ReceiveMessage()
		if err != nil {
			log.Printf("[gRPC] action=tra_liveboard event=query_failed error=%v", err)
			return err
		}
		resp := &pb.RespTraLiveBoard{
			Data: []byte(val.Payload),
		}
		if err := stream.Send(resp); err != nil {
			log.Printf("[gRPC] action=tra_liveboard event=send_failed error=%v", err)
			return err
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
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
		}
		val, err := sub.ReceiveMessage()
		if err != nil {
			log.Printf("[gRPC] action=tra_delay event=query_failed error=%v", err)
			return err
		}
		resp := &pb.RespTraDelay{
			Data: []byte(val.Payload),
		}
		if err := stream.Send(resp); err != nil {
			log.Printf("[gRPC] action=tra_delay event=send_failed error=%v", err)
			return err
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
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
		}
		val, err := sub.ReceiveMessage()
		if err != nil {
			log.Printf("[gRPC] action=tra_delay event=query_failed error=%v", err)
			return err
		}
		resp := &pb.RespTraDelay{
			Data: []byte(val.Payload),
		}
		if err := stream.Send(resp); err != nil {
			log.Printf("[gRPC] action=tra_delay event=send_failed error=%v", err)
			return err
		}
	}
}
func (s *Tra_TimetableServer) traFare(in *pb.AskRoute, ctx context.Context) (*pb.Resp_Data, error) {
	log.Printf("call tra fare")
	val := s.rc.Get(fmt.Sprintf("TRA_Fare:%s:%s", in.OriginStationId, in.DestinationStationId))
	if errors.Is(val.Err(), redis.Nil) {
		tra_price(in.OriginStationId, in.DestinationStationId, ctx, s.client, s.db, s.rc)
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
func (s *ThsrServer) thsrFare(in *pb.AskRoute, ctx context.Context) (*pb.Resp_Data, error) {
	log.Printf("call thsr fare")
	val := s.rc.Get(fmt.Sprintf("THSR_Fare:%s:%s", in.OriginStationId, in.DestinationStationId))
	if errors.Is(val.Err(), redis.Nil) {
		thsr_price(in.OriginStationId, in.DestinationStationId, ctx, s.client, s.db, s.rc)
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
func (s *Tra_TimetableServer) traTimetable(in *pb.AskRoute, ctx context.Context) (*pb.Resp_Data, error) {
	log.Printf("call tra timetable %s %s", in.OriginStationId, in.DestinationStationId)
	da, _ := time.Parse(time.RFC3339, in.Date)
	val := s.rc.Get(fmt.Sprintf("TRA_timetable:%s:%s:%s", da.Format(time.DateOnly), in.OriginStationId, in.DestinationStationId))
	if errors.Is(val.Err(), redis.Nil) {
		t, _ := time.Parse(time.RFC3339, in.Date)
		tra_timetable(in.OriginStationId, in.DestinationStationId, t, ctx, s.client, s.db, s.rc)
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
func (s *ThsrServer) thsrTimetable(in *pb.AskRoute, ctx context.Context) (*pb.Resp_Data, error) {
	log.Printf("call thsr timetable %s %s", in.OriginStationId, in.DestinationStationId)
	val := s.rc.Get(fmt.Sprintf("THSR_timetable:%s:%s:%s", in.Date, in.OriginStationId, in.DestinationStationId))
	if errors.Is(val.Err(), redis.Nil) {
		t, _ := time.Parse(time.RFC3339, in.Date)
		thsr_timetable(in.OriginStationId, in.DestinationStationId, t, ctx, s.client, s.db, s.rc)
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
func (s *Tra_DetainServer) traStops(in *pb.AskDetain, ctx context.Context) (*pb.Resp_Data, error) {
	log.Printf("call tra stops %s", in.Trainno)
	val := s.rc.Get(fmt.Sprintf("TRA_Stoptimes:%s:%s", in.Date, in.Trainno))
	if errors.Is(val.Err(), redis.Nil) {
		t, _ := time.Parse(time.RFC3339, in.Date)
		tra_stoptimes(in.Trainno, t, ctx, s.client, s.db, s.rc)
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
func (s *Thsr_DetainServer) thsrStops(in *pb.ThsrAskDetain, ctx context.Context) (*pb.Resp_Data, error) {
	log.Printf("call thsr stops %s", in.Trainno)
	val := s.rc.Get(fmt.Sprintf("THSR_Stoptimes:%s:%s", in.Date, in.Trainno))
	if errors.Is(val.Err(), redis.Nil) {
		t, _ := time.Parse(time.RFC3339, in.Date)
		thsr_stoptimes(in.Trainno, t, ctx, s.client, s.db, s.rc)
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
		resp, err := findnearstation(l1, l2, l3, ctx, s.db)
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
func makethatsame(subRouteUID string) string {
	if subRouteUID[0:2] == "THB" {
		temp := subRouteUID[len(subRouteUID)-2:]
		if temp == "01" {
			return subRouteUID[:len(subRouteUID)-2]
		} else if temp == "02" {
			return subRouteUID[:len(subRouteUID)-2]
		}
		return subRouteUID[:len(subRouteUID)-1]
	}
	return subRouteUID
}

func connectredis() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: "",
		DB:       0,
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
	conn, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Printf("[DB] action=connect event=failed error=%v", err)
		panic(err)
	}
	err = conn.Ping(context.Background())
	if err != nil {
		log.Printf("[DB] action=ping event=failed error=%v", err)
		panic(err)
	}
	log.Printf("[DB] action=connect event=success")
	return conn
}
