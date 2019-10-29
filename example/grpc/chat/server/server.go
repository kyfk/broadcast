package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/kyfk/log"
	"github.com/kyfk/log/format"
	"github.com/kyfk/log/level"
	"github.com/kyfk/broadcast"
	pb "github.com/kyfk/broadcast/example/grpc/chat/proto/chat"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type server struct {
	logger   *log.Logger
	hub      *broadcast.Hub
	messages []*pb.Message
	mux      sync.Mutex
}

type MessageStream struct {
	err      error
	username string
	logger   *log.Logger
	stream   pb.Chat_CommunicateServer
	hub      *broadcast.Hub
}

func (m *MessageStream) GetID() broadcast.ID {
	return m.username
}

func (m *MessageStream) Subscribe() error {
	return m.hub.Publish(func(s broadcast.Subscriber) error {
		return s.(*MessageStream).stream.Send(&pb.RecieveMessageResponse{
			Message: &pb.Message{
				Username: "system",
				Body:     fmt.Sprintf("`%s` joined", m.username),
				SentAt:   ptypes.TimestampNow(),
			},
		})
	})
}

func (m *MessageStream) Unsubscribe() error {
	return m.hub.Publish(func(s broadcast.Subscriber) error {
		return s.(*MessageStream).stream.Send(&pb.RecieveMessageResponse{
			Message: &pb.Message{
				Username: "system",
				Body:     fmt.Sprintf("`%s` left", m.username),
				SentAt:   ptypes.TimestampNow(),
			},
		})
	})
}

func (s *server) Communicate(srv pb.Chat_CommunicateServer) error {
	md, ok := metadata.FromIncomingContext(srv.Context())
	if !ok {
		return status.Errorf(codes.DataLoss, "failed to get metadata")
	}

	ms := &MessageStream{
		username: md["username"][0],
		logger:   s.logger,
		stream:   srv,
		hub:      s.hub,
	}
	err := s.hub.Subscribe(ms)
	if err != nil {
		return err
	}

	for {
		res, err := srv.Recv()
		if err != nil {
			ms.err = err
			s.hub.Unsubscribe(ms)
			return err
		}

		s.mux.Lock()
		s.messages = append(s.messages, res.GetMessage())
		s.mux.Unlock()

		s.hub.Publish(func(s broadcast.Subscriber) error {
			return s.(*MessageStream).stream.Send(&pb.RecieveMessageResponse{
				Message: res.GetMessage(),
			})
		})
	}
}

func (s *server) ListMessages(ctx context.Context, req *empty.Empty) (*pb.ListMessagesResponse, error) {
	return &pb.ListMessagesResponse{Messages: s.messages}, nil
}

func main() {
	logger := log.New(
		log.Format(format.JSONPretty),
		log.MinLevel(level.Debug),
	)

	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	s := grpc.NewServer()
	pb.RegisterChatServer(s, &server{
		hub: broadcast.New(
			broadcast.Concurrency(10),
			broadcast.AllowDuplicableID(true),
		),
		logger: logger,
	})

	if err := s.Serve(lis); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
