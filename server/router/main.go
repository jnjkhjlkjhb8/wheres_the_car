package main

import (
	"context"
	"log"
	"os"

	"github.com/go-redis/redis"
	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	pb "github.com/jnjkhjlkjhb8/bus/models"
	"google.golang.org/grpc"
)

type server struct {
	db    *pgxpool.Pool
	redis *redis.Client
	tdx   *resty.Client
}

type bus_returnroute interface {
	BusRouteStatic(context.Context, *pb.Requestroute) (*pb.Subroute, error)
	BusRouteEta(*pb.Requestroute, requestRouteEtaServer) error
}

type bus_returnstation interface {
	BusStationEta(*pb.Requeststation, requestStationEtaServer) error
}

type bike_return interface {
	BikeStatic(context.Context, *pb.BikeRequest) (*pb.BikeStatic, error)
	BikeEta(*pb.BikeRequest, requestBikeEtaServer) error
}

type mrt_return interface {
	MrtEta(*pb.Requestmrt, requestMrtEtaServer) error
}

type thsr_return interface {
	ThsrFare(context.Context, *pb.AskStatic) (*pb.ThsaFare, error)
	ThsrTimetable(context.Context, *pb.AskStatic) (*pb.ThsaTimetable, error)
}

type tra_returnstation interface {
	TraStationLiveBoard(*pb.AskStaiton, rquestStationLiveBoardServer) error
	TraStationTimetable(context.Context, *pb.AskStaiton) (*pb.TraTimetable, error)
}

type tra_return interface {
	TraRouteTimetable(context.Context, *pb.AskRoute) (*pb.TraTimetable, error)
	TraRouteFare(context.Context, *pb.AskStaiton) (*pb.TraFareItem, error)
	TraRouteDelay(*pb.AskRoute, requestRoutesDelayServer) error
}

type bus_requestroute interface {
	Send(*pb.Bus_RouteArrival) error
	grpc.ServerStream
}

type bus_requestation interface {
	Send(*pb.Bus_StationArrival) error
	grpc.ServerStream
}

type bike_request interface {
	Send(*pb.BikeEta) error
	grpc.ServerStream
}

type mrt_request interface {
	Send(*pb.MrtLive) error
	grpc.ServerStream
}

type tra_requestation interface {
	Send(*pb.Tra_LiveBoards) error
	grpc.ServerStream
}

type tra_requestdelay interface {
	Send(*pb.TraDelay) error
	grpc.ServerStream
}

func run() error {
	redisClient := connectredis()
	dbPool := connectdb()

	srv := grpc.NewServer()
	pb.RegisterBusServer(srv, &server{
		db:    dbPool,
		redis: redisClient,
		tdx:   nil,
	})

	return srv.ListenAndServe()
}

func makethatsame(city string, subRouteUID string, Direction uint8) (string, uint8) {
	if city == "InterCity" {
		temp := subRouteUID[len(subRouteUID)-2:]
		if temp == "01" {
			return subRouteUID[:len(subRouteUID)-2], 0
		} else if temp == "02" {
			return subRouteUID[:len(subRouteUID)-2], 1
		}
		return subRouteUID[:len(subRouteUID)-2], Direction
	}
	return subRouteUID, Direction
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
		//panic(err)
	}
	log.Printf("[REDIS] action=connect event=success pong=%s", pong)
	return client
}

func connectdb() *pgxpool.Pool {
	conn, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Printf("[DB] action=connect event=failed error=%v", err)
		//panic(err)
	}
	err = conn.Ping(context.Background())
	if err != nil {
		log.Printf("[DB] action=ping event=failed error=%v", err)
		//panic(err)
	}
	log.Printf("[DB] action=connect event=success")
	return conn
}
