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

	ctx       context.Context
	cancel    context.CancelFunc
	isRunning bool

	token string

	pb.UnimplementedStackServer
}

// NewStackNode creates a new stack node
func NewStackNode(token string) *StackNode {
	ctx, cancel := context.WithCancel(context.Background())
	return &StackNode{
		stack:  utils.NewIntStack(),
		ctx:    ctx,
		cancel: cancel,
		token:  token,
	}
}

// Start starts stack node
func (s *StackNode) Start() {
	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	server := grpc.NewServer()
	pb.RegisterStackServer(server, s)
	log.Printf("starting grpc server...")
	if err := server.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// GetRequestMetadata gets metadata with token set for request
func (s *StackNode) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": fmt.Sprintf("Bearer %s", s.token),
	}, nil
}

// RequireTransportSecurity indicates credentials are required
func (s *StackNode) RequireTransportSecurity() bool {
	return true
}

// Run runs stack node
func (s *StackNode) Run(ctx context.Context, in *pb.RunRequest) (*pb.CommandReply, error) {
	if !s.isRunning {
		s.isRunning = true
		log.Printf("node was run")
	} else {
		log.Printf("node is already running")
	}
	return &pb.CommandReply{}, nil
}

// Pause pauses stack node
func (s *StackNode) Pause(ctx context.Context, in *pb.PauseRequest) (*pb.CommandReply, error) {
	if s.isRunning {
		s.stopNode()
		log.Printf("node was paused")
	} else {
		log.Printf("node is already paused")
	}
	return &pb.CommandReply{}, nil
}

// Reset resets stack node
func (s *StackNode) Reset(ctx context.Context, in *pb.ResetRequest) (*pb.CommandReply, error) {
	if s.isRunning {
		s.stopNode()
	}
	s.resetNode()
	log.Printf("node was reset")
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

// stopNode stops stack node
func (s *StackNode) stopNode() {
	s.cancel()
	s.isRunning = false

	// Re-create node context
	nodeCtx, cancel := context.WithCancel(context.Background())
	s.ctx = nodeCtx
	s.cancel = cancel
}

// resetNode resets stack node
func (s *StackNode) resetNode() {
	s.stack.Clear()
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
