package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/go-redis/redis"
	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	pb "github.com/jnjkhjlkjhb8/bus/models"
)

type Bus_RouteServer struct {
	pb.UnimplementedBus_Route_ServiceServer
	mu sync.Mutex
	db *pgxpool.Pool
	rc *redis.Client
}
type Bus_StationServer struct {
	pb.UnimplementedBus_Station_ServiceServer
	mu sync.Mutex
	rc *redis.Client
}
type Bike_Server struct {
	pb.UnimplementedBike_ServiceServer
	mu sync.Mutex
	db *pgxpool.Pool
	rc *redis.Client
}
type Mrt_Server struct {
	pb.UnimplementedMrt_ServiceServer
	mu sync.Mutex
	rc *redis.Client
	db *pgxpool.Pool
}
type Thsr_server struct {
	pb.UnimplementedThsrTimetableServiceServer
	mu sync.Mutex
	db *pgxpool.Pool
}
type Tra_StationServer struct {
	pb.UnimplementedTRAStationServiceServer
	mu sync.Mutex
	db *pgxpool.Pool
}
type Tra_TimetableServer struct {
	pb.UnimplementedTRATimetableServiceServer
	mu sync.Mutex
	db *pgxpool.Pool
}
type Near_Server struct {
	pb.UnimplementedNear_Station_ServiceServer
	mu sync.Mutex
	db *pgxpool.Pool
}

func main() {
	rc := connectredis()
	db := connectdb()
	defer rc.Close()
	defer db.Close()
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
}
func (s *Bus_RouteServer) Bus_route_static(ctx context.Context, in *pb.Bus_Ask_Route) (*pb.Resp_BusStatic, error) {
	log.Printf("call bus_static %s", in.SubRouteUID)
	route := makethatsame(in.SubRouteUID)
	err := s.db.QueryRow(ctx, `SELECT pb FROM bus_static where sub_route_uid like '$1';`, route).Scan(&route)
	if err != nil {
		log.Printf("[gRPC] action=bus_static event=query_failed error=%v", err)
		return nil, err
	}
	resp := &pb.Resp_BusStatic{
		Data: []byte(route),
	}
	return resp, nil
}
func (s *Bus_RouteServer) Bus_route_eta(ctx context.Context, in *pb.Bus_Ask_Route, stream pb.Bus_Route_Service_EtaServer) error {
	log.Printf("call Bus_route_eta %s", in.SubRouteUID)
	route := makethatsame(in.SubRouteUID)
	sub := s.rc.Subscribe(fmt.Sprintf("bus_eta_route:%s", route))
	defer sub.Close()
	for {
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
func (s *Bus_RouteServer) Bus_Station_eta(ctx context.Context, in *pb.Bus_Ask_Station, stream pb.Bus_Station_Service_EtaServer) error {
	log.Printf("call Bus_station_eta %s", in.StationName)
	sub := s.rc.Subscribe(fmt.Sprintf("bus_eta_station:%s:%s", in.City, in.StationName))
	defer sub.Close()
	for {
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
func (s *Bus_RouteServer) Bus_Dailytable(in *pb.Bus_Ask_Route) (*pb.Resp_BusEta, error) {
	log.Printf("call Bus_route_eta %s", in.SubRouteUID)
	route := makethatsame(in.SubRouteUID)
	val, err := s.rc.Get(fmt.Sprintf("bus_dailytable:%s", route)).Result()
	if err != nil {
		log.Printf("[gRPC] action=bus_dailytable event=query_failed error=%v", err)
		return nil, err
	}
	resp := &pb.Resp_BusEta{
		Data: []byte(val),
	}
	return resp, nil
}
func (s *Mrt_Server) Mrt_eta(in *pb.AskMrt, stream pb.Mrt_Service_EtaServer) error {
	log.Printf("call Mrt_eta %s %s", in.System, in.StationID)
	sub := s.rc.Subscribe(fmt.Sprintf("mrt_live:%s:%s", in.System, in.StationID))
	defer sub.Close()
	for {
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
func (s *Bike_Server) Bike_static(ctx context.Context, in *pb.BikeRequest) (*pb.BikeStatic, error) {
	log.Printf("call bike_static %s", in.StationUID)
	var name, address string
	var capacity, service_type int32
	err := s.db.QueryRow(ctx, `SELECT name,capacity,service_type,address FROM bike_stations where station_uid like '$1';`, in.StationUID).Scan(&name, &capacity, &service_type, &address)
	if err != nil {
		log.Printf("[gRPC] action=bike_static event=query_failed error=%v", err)
		return nil, err
	}
	resp := &pb.BikeStatic{
		StationUID:  in.StationUID,
		Name:        name,
		Capacity:    capacity,
		ServiceType: service_type,
		Address:     address,
	}
	return resp, nil
}
func (s *Bike_Server) bike_eta(in *pb.BikeRequest, stream pb.Bike_Service_EtaServer) error {
	log.Printf("call bike_eta %s", in.StationUID)
	sub := s.rc.Subscribe(fmt.Sprintf("bike_availability:%s", in.StationUID))
	defer sub.Close()
	for {
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
func makethatsame(subRouteUID string) string {
	if subRouteUID[0:2] == "THB" {
		temp := subRouteUID[len(subRouteUID)-2:]
		if temp == "01" {
			return subRouteUID[:len(subRouteUID)-2]
		} else if temp == "02" {
			return subRouteUID[:len(subRouteUID)-2]
		}
		return subRouteUID[:len(subRouteUID)-2]
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
