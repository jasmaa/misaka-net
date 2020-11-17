package nodes

import (
	"context"
	"fmt"
	"log"
	"net"

	pb "github.com/jasmaa/misaka-net/internal/grpc"
	"github.com/jasmaa/misaka-net/internal/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// StackNode is a stack node
type StackNode struct {
	stack *utils.IntStack

	ctx       context.Context
	cancel    context.CancelFunc
	isRunning bool

	pushSignal chan interface{}

	certFile, keyFile string

	pb.UnimplementedStackServer
}

// NewStackNode creates a new stack node
func NewStackNode(certFile, keyFile string) *StackNode {
	ctx, cancel := context.WithCancel(context.Background())
	return &StackNode{
		stack:      utils.NewIntStack(),
		ctx:        ctx,
		cancel:     cancel,
		pushSignal: make(chan interface{}),
		certFile:   certFile,
		keyFile:    keyFile,
	}
}

// Start starts stack node
func (s *StackNode) Start() {
	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	creds, err := credentials.NewServerTLSFromFile(s.certFile, s.keyFile)
	if err != nil {
		log.Fatalf("failed to get creds: %v", err)
	}
	server := grpc.NewServer(grpc.Creds(creds))
	pb.RegisterStackServer(server, s)
	log.Printf("starting grpc server...")
	if err := server.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
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

	// Signal push with non-blocking send
	select {
	case s.pushSignal <- nil:
	default:
	}

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

			// Sleep until push occurs
			<-s.pushSignal
		}
	}()

	select {
	case v := <-c:
		return v, nil
	case <-s.ctx.Done():
		return -1, fmt.Errorf("stack pop cancelled")
	}
}
