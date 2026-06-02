package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/go-redis/redis"
	pb "github.com/jnjkhjlkjhb8/wheres_the_car/models"
	"google.golang.org/grpc"
)

type AlertServer struct {
	pb.UnimplementedAlert_ServiceServer
	mu sync.Mutex
	rc *redis.Client
}

func (s *AlertServer) BusNews(in *pb.Alert_Bus_Ask, stream grpc.ServerStreamingServer[pb.Alert_Msg]) error {
	key := fmt.Sprintf("mqtt:v2:Bus:News:City:%s", in.City)
	return streamAlert(s.rc, key, stream)
}

func (s *AlertServer) MetroAlert(in *pb.Alert_Metro_Ask, stream grpc.ServerStreamingServer[pb.Alert_Msg]) error {
	key := fmt.Sprintf("mqtt:v2:Rail:Metro:Alert:%s", in.System)
	return streamAlert(s.rc, key, stream)
}

func (s *AlertServer) TraAlert(_ *pb.Alert_Ask, stream grpc.ServerStreamingServer[pb.Alert_Msg]) error {
	return streamAlert(s.rc, "mqtt:v3:Rail:TRA:Alert", stream)
}

func (s *AlertServer) ThsrAlert(_ *pb.Alert_Ask, stream grpc.ServerStreamingServer[pb.Alert_Msg]) error {
	return streamAlert(s.rc, "mqtt:v2:Rail:THSR:AlertInfo", stream)
}

func streamAlert(rc *redis.Client, key string, stream grpc.ServerStreamingServer[pb.Alert_Msg]) error {
	sub := rc.Subscribe(key)
	defer func() { _ = sub.Close() }()

	if val := rc.Get(key); val.Val() != "" {
		if err := stream.Send(&pb.Alert_Msg{Data: []byte(val.Val())}); err != nil {
			log.Printf("[gRPC] action=alert event=send_cached_failed key=%s err=%v", key, err)
			return err
		}
	}

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
		}
		msg, err := sub.ReceiveMessage()
		if err != nil {
			log.Printf("[gRPC] action=alert event=receive_failed key=%s err=%v", key, err)
			return err
		}
		if err := stream.Send(&pb.Alert_Msg{Data: []byte(msg.Payload)}); err != nil {
			log.Printf("[gRPC] action=alert event=send_failed key=%s err=%v", key, err)
			return err
		}
	}
}
