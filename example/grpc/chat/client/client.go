package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/kyfk/broadcast/example/grpc/chat/proto/chat"
	"github.com/kyfk/log"
	"github.com/kyfk/log/format"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	username = flag.String("username", "username", "the username to show")
	addr     = flag.String("addr", "localhost:50052", "the address to connect to")
	// see https://github.com/grpc/grpc/blob/master/doc/service_config.md to know more about service config
	retryPolicy = `{
		"methodConfig": [{
		  "name": [{"service": "grpc.examples.echo.Echo"}],
		  "waitForReady": true,
		  "retryPolicy": {
			  "MaxAttempts": 4,
			  "InitialBackoff": ".01s",
			  "MaxBackoff": ".01s",
			  "BackoffMultiplier": 1.0,
			  "RetryableStatusCodes": [ "UNAVAILABLE" ]
		  }
		}]}`
)

// use grpc.WithDefaultServiceConfig() to set service config
func retryDial() (*grpc.ClientConn, error) {
	return grpc.Dial(*addr, grpc.WithInsecure(), grpc.WithDefaultServiceConfig(retryPolicy))
}

func main() {
	flag.Parse()
	log.SetFormat(format.JSONPretty)

	// Set up a connection to the server.
	conn, err := retryDial()
	if err != nil {
		log.Error(fmt.Errorf("did not connect: %v", err))
		return
	}
	defer func() {
		if e := conn.Close(); e != nil {
			log.Error(fmt.Errorf("did not connect: %v", err))
		}
	}()

	c := pb.NewChatClient(conn)

	ms, err := c.ListMessages(context.Background(), &empty.Empty{})
	if err != nil {
		log.Error(err)
		return
	}
	for _, m := range ms.GetMessages() {
		printMessage(m)
	}

	md := metadata.Pairs("username", *username)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	stream, err := c.Communicate(ctx)
	if err != nil {
		log.Error(err)
		return
	}

	errCh := make(chan error)
	go func() {
		for {
			res, err := stream.Recv()
			if err != nil {
				errCh <- err
				return
			}
			printMessage(res.GetMessage())
		}
	}()

	fmt.Println("input text:")
	go func() {
		for {
			var msg string
			fmt.Scanf("%s", &msg)
			err := stream.Send(&pb.SendMessageRequest{
				Message: &pb.Message{
					Username: *username,
					Body:     msg,
					SentAt:   ptypes.TimestampNow(),
				},
			})
			if err != nil {
				errCh <- err
				return
			}
		}
	}()

	err = <-errCh
	log.Error(err)
	os.Exit(1)
}

func printMessage(m *pb.Message) {
	sentAt := time.Unix(m.SentAt.GetSeconds(), int64(m.SentAt.GetNanos()))
	fmt.Printf("[%s]%s: %s\n", sentAt.Format(time.RFC3339), m.GetUsername(), m.GetBody())
}
