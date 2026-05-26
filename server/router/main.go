package main

import (
	"context"
	"log"
	"os"
	"sync"

	"github.com/go-redis/redis"
	"github.com/jackc/pgx/v5/pgxpool"
	pb "github.com/jnjkhjlkjhb8/bus/models"
)

type Bus_RouteServer struct {
	pb.UnimplementedBusRouteServer
	mu sync.Mutex
}
type Bus_StationServer struct {
	pb.UnimplementedBusStationServer
	mu sync.Mutex
}
type Bike_request struct {
	pb.UnimplementedRequestBikeServer
	mu sync.Mutex
}
type Mrt_request struct {
	pb.UnimplementedRequestMrtServer
	mu sync.Mutex
}
type Thsr_request struct {
	pb.UnimplementedRequestThsrServer
	mu sync.Mutex
}
type Tra_StationServer struct {
	pb.UnimplementedRquestStationServer
	mu sync.Mutex
}
type Tra_RoutesServer struct {
	pb.UnimplementedRequestRoutesServer
	mu sync.Mutex
}
type Near_StationServer struct {
	pb.UnimplementedNear_StationServer
	mu sync.Mutex
}

func main() {
	rc := connectredis()
	db := connectdb()
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
