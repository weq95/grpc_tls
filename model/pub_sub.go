package model

import (
	"context"
	"fmt"
	"github.com/docker/docker/pkg/pubsub"
	"google.golang.org/grpc"
	"grpc_tls/proto/src"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

func PubSub() {
	var p = pubsub.NewPublisher(time.Millisecond*100, 10)
	var golang = p.SubscribeTopic(func(v interface{}) bool {
		if key, ok := v.(string); ok {
			if strings.HasPrefix(key, "golang:") {
				return true
			}
		}

		return false
	})

	var docker = p.SubscribeTopic(func(v interface{}) bool {
		if key, ok := v.(string); ok {
			if strings.HasPrefix(key, "docker:") {
				return true
			}
		}

		return false
	})

	go p.Publish("hi")
	go p.Publish("golang: https://golang.org")
	go p.Publish("docker: https://www.docker.com/")
	time.Sleep(1)

	go func() {
		fmt.Println("golang topic:", <-golang)
	}()

	go func() {
		fmt.Println("docker topic:", <-docker)
	}()

}

type PubSubService struct {
	pub *pubsub.Publisher
}

func NewPubSubService() *PubSubService {
	return &PubSubService{
		pub: pubsub.NewPublisher(time.Millisecond*100, 10),
	}
}

func (s *PubSubService) Publish(ctx context.Context, arg *src.String) (*src.String, error) {
	s.pub.Publish(arg.GetValue())

	return &src.String{}, nil
}

func (s *PubSubService) Subscribe(arg *src.String, server src.PubSubService_SubscribeServer) error {
	var ch = s.pub.SubscribeTopic(func(v interface{}) bool {
		if key, ok := v.(string); ok {
			return strings.HasPrefix(key, arg.GetValue())
		}

		return false
	})

	select {
	case v := <-ch:
		if err := server.Send(&src.String{Value: v.(string)}); err != nil {
			return err
		}
	}

	return nil
}

var addr = "localhost:1234"

func Client01() {
	var conn, err = grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close()

	var client = src.NewPubSubServiceClient(conn)
	_, err = client.Publish(context.Background(), &src.String{Value: "golang: hello Go"})
	if err != nil {
		log.Fatal(err)
	}

	_, err = client.Publish(context.Background(), &src.String{Value: "docker: hello Docker"})
	if err != nil {
		log.Fatal("client01 错误信息: ", err)
	}
}

func Client02() {
	var conn, err = grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	var client = src.NewPubSubServiceClient(conn)
	var stream, err01 = client.Subscribe(context.Background(), &src.String{Value: "golang:"})
	if err01 != nil {
		log.Fatal(err01)
	}

	for true {
		var reply, err02 = stream.Recv()
		if err02 != nil {
			if err02 == io.EOF {
				break
			}

			log.Fatal("client02 错误信息: ", err)
		}

		fmt.Println(reply.GetValue())
	}
}

func Server() {
	var grpcServer = grpc.NewServer()
	src.RegisterPubSubServiceServer(grpcServer, NewPubSubService())

	var listen, err = net.Listen("tcp", ":1234")
	if err != nil {
		log.Fatal("rpc net.Listen err: ", err)
	}

	if err = grpcServer.Serve(listen); err != nil {
		log.Fatal("rpc server: ", err)
	}
}
