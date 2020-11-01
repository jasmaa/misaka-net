package nodes

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	pb "github.com/jasmaa/misaka-net/internal/grpc"
	"github.com/jasmaa/misaka-net/internal/utils"
	"google.golang.org/grpc"
)

// StackNode is a stack node
type StackNode struct {
	stack *utils.IntStack

	ctx    context.Context
	cancel context.CancelFunc

	pb.UnimplementedStackServer
}

// popResponse structures response to pop request
type popResponse struct {
	Value int `json:"value"`
}

// NewStackNode creates a new stack node
func NewStackNode() *StackNode {
	ctx, cancel := context.WithCancel(context.Background())
	return &StackNode{
		stack:  utils.NewIntStack(),
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start starts stack node
func (s *StackNode) Start() {
	lis, err := net.Listen("tcp", ":8000")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	server := grpc.NewServer()
	pb.RegisterStackServer(server, s)
	log.Printf("starting server...")
	if err := server.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// Run runs stack node
func (s *StackNode) Run(ctx context.Context, in *pb.RunRequest) (*pb.CommandReply, error) {
	ctx, cancel := context.WithCancel(context.Background())
	s.ctx = ctx
	s.cancel = cancel
	return &pb.CommandReply{}, nil
}

// Pause pauses stack node
func (s *StackNode) Pause(ctx context.Context, in *pb.PauseRequest) (*pb.CommandReply, error) {
	s.cancel()
	return &pb.CommandReply{}, nil
}

// Reset resets stack node
func (s *StackNode) Reset(ctx context.Context, in *pb.ResetRequest) (*pb.CommandReply, error) {
	s.cancel()
	s.stack.Clear()
	return &pb.CommandReply{}, nil
}

// Push pushes value onto stack node
func (s *StackNode) Push(ctx context.Context, in *pb.PushValueRequest) (*pb.CommandReply, error) {
	s.stack.Push(int(in.Value))
	return &pb.CommandReply{}, nil
}

// Pop pops value from stack node
func (s *StackNode) Pop(ctx context.Context, in *pb.PopValueRequest) (*pb.ValueReply, error) {
	v, err := s.waitPop()
	if err != nil {
		return nil, err
	}
	return &pb.ValueReply{Value: int32(v)}, nil
}

// waitPop waits until value can be popped from stack and returns value
func (s *StackNode) waitPop() (int, error) {
	c := make(chan int, 1)
	go func() {
		for {
			v, err := s.stack.Pop()
			if err == nil {
				c <- v
				return
			}
			time.Sleep(1 * time.Second)
		}
	}()

	select {
	case v := <-c:
		return v, nil
	case <-s.ctx.Done():
		return -1, fmt.Errorf("stack pop cancelled")
	}
}
