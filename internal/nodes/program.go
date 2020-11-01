package nodes

import (
	"context"
	"fmt"
	"log"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	pb "github.com/jasmaa/misaka-net/internal/grpc"
	"github.com/jasmaa/misaka-net/internal/tis"
	"github.com/jasmaa/misaka-net/internal/utils"
	"google.golang.org/grpc"
)

// Register buffer size
const bufferSize = 1

// ProgramNode is a program node that interprets TIS-100 asm
type ProgramNode struct {
	masterURI string

	acc int
	bak int
	r0  chan int
	r1  chan int
	r2  chan int
	r3  chan int

	ptr      int
	asm      [][]string
	labelMap map[string]int

	ctx       context.Context
	cancel    context.CancelFunc
	isRunning bool

	pb.UnimplementedProgramServer
}

// NewProgramNode creates a new program node
func NewProgramNode(masterURI string) *ProgramNode {
	ctx, cancel := context.WithCancel(context.Background())
	return &ProgramNode{
		masterURI: masterURI,
		acc:       0,
		bak:       0,
		r0:        make(chan int, bufferSize),
		r1:        make(chan int, bufferSize),
		r2:        make(chan int, bufferSize),
		r3:        make(chan int, bufferSize),
		asm:       [][]string{[]string{"NOP"}},
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start starts program loop and server
func (p *ProgramNode) Start() {
	// Run program loop
	go func() {
		for {
			if p.isRunning {
				err := p.update()
				if err != nil {
					log.Print(err)
				}
			} else {
				time.Sleep(1 * time.Second)
			}
		}
	}()

	// TODO: authenticate commands from master with key
	// TODO: switch to faster protocol (grpc?)

	lis, err := net.Listen("tcp", ":8000")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterProgramServer(s, p)
	log.Printf("starting server...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// Run starts asm execution
func (p *ProgramNode) Run(ctx context.Context, in *pb.RunRequest) (*pb.CommandReply, error) {
	if !p.isRunning {
		p.isRunning = true
		ctx, cancel := context.WithCancel(context.Background())
		p.ctx = ctx
		p.cancel = cancel
		log.Printf("node was run")
	} else {
		log.Printf("node is already running")
	}
	return &pb.CommandReply{}, nil
}

// Pause pauses asm execution
func (p *ProgramNode) Pause(ctx context.Context, in *pb.PauseRequest) (*pb.CommandReply, error) {
	if p.isRunning {
		p.cancel()
		p.isRunning = false
		log.Printf("node was paused")
	} else {
		log.Printf("node is already paused")
	}
	return &pb.CommandReply{}, nil
}

// Reset resets asm execution and registers
func (p *ProgramNode) Reset(ctx context.Context, in *pb.ResetRequest) (*pb.CommandReply, error) {
	p.resetNode()
	log.Printf("node was reset")
	return &pb.CommandReply{}, nil
}

// LoadProgram loads program onto node
func (p *ProgramNode) LoadProgram(s string) error {
	instrArr := strings.Split(s, "\n")
	labelMap, err := tis.GenerateLabelMap(instrArr)
	if err != nil {
		return err
	}

	asm, err := tis.Tokenize(instrArr, labelMap)
	if err != nil {
		return err
	}

	p.asm = asm
	p.labelMap = labelMap
	return nil
}

// Load resets node and loads asm program
func (p *ProgramNode) Load(ctx context.Context, in *pb.LoadRequest) (*pb.CommandReply, error) {
	p.resetNode()
	err := p.LoadProgram(in.Program)
	if err != nil {
		return nil, err
	}
	return &pb.CommandReply{}, nil
}

// SendValue sends value to
func (p *ProgramNode) SendValue(ctx context.Context, in *pb.SendValueRequest) (*pb.CommandReply, error) {
	switch in.Register {
	case 0:
		p.r0 <- int(in.Value)
	case 1:
		p.r1 <- int(in.Value)
	case 2:
		p.r2 <- int(in.Value)
	case 3:
		p.r3 <- int(in.Value)
	default:
		return nil, fmt.Errorf("not a valid register")
	}
	log.Printf("received value")
	return &pb.CommandReply{}, nil
}

// resetNode resets program node
func (p *ProgramNode) resetNode() {
	// Stop program execution
	p.cancel()
	p.isRunning = false

	p.acc = 0
	p.bak = 0
	p.ptr = 0

	p.r0 = make(chan int, bufferSize)
	p.r1 = make(chan int, bufferSize)
	p.r2 = make(chan int, bufferSize)
	p.r3 = make(chan int, bufferSize)
}

// Update steps through asm
func (p *ProgramNode) update() error {
	tokens := p.asm[p.ptr]

	// TODO: remove this
	log.Printf("%v\nACC: %v\nBAK: %v\n", tokens, p.acc, p.bak)

	switch tokens[0] {
	case "NOP":
		// no-op
	case "MOV_VAL_LOCAL":
		// Moves value locally
		v, err := strconv.Atoi(tokens[1])
		if err != nil {
			return err
		}
		switch tokens[2] {
		case "ACC":
			p.acc = v
		case "NIL":
			// no-op
		default:
			return fmt.Errorf("'%s' not a valid local register", tokens[2])
		}
	case "MOV_VAL_NETWORK":
		// Moves value across network
		v, err := strconv.Atoi(tokens[1])
		if err != nil {
			return err
		}
		err = p.sendValue(v, tokens[2])
		if err != nil {
			return err
		}
	case "MOV_SRC_LOCAL":
		// Moves value from src locally
		v, err := p.getFromSrc(tokens[1])
		if err != nil {
			return err
		}
		switch tokens[2] {
		case "ACC":
			p.acc = v
		case "NIL":
			// no-op
		default:
			return fmt.Errorf("'%s' not a valid local register", tokens[2])
		}
	case "MOV_SRC_NETWORK":
		// Moves value from src across network
		v, err := p.getFromSrc(tokens[1])
		if err != nil {
			return err
		}
		err = p.sendValue(v, tokens[2])
		if err != nil {
			return err
		}
	case "SWP":
		// Swaps ACC and BAK
		temp := p.acc
		p.acc = p.bak
		p.bak = temp
	case "SAV":
		// Saves ACC to BAK
		p.bak = p.acc
	case "ADD_VAL":
		// Adds value to ACC
		v, err := strconv.Atoi(tokens[1])
		if err != nil {
			return err
		}
		p.acc += v
	case "SUB_VAL":
		// Subs value from ACC
		v, err := strconv.Atoi(tokens[1])
		if err != nil {
			return err
		}
		p.acc -= v
	case "ADD_SRC":
		// Adds value in src to ACC
		v, err := p.getFromSrc(tokens[1])
		if err != nil {
			return err
		}
		p.acc += v
	case "SUB_SRC":
		// Subs value in src from ACC
		v, err := p.getFromSrc(tokens[1])
		if err != nil {
			return err
		}
		p.acc -= v
	case "NEG":
		// Negates ACC
		p.acc = -p.acc
	case "JMP":
		// Jumps unconditionally to label
		label := tokens[1]
		p.ptr = p.labelMap[label]
		return nil
	case "JEZ":
		// Jump if ACC equals zero
		if p.acc == 0 {
			label := tokens[1]
			p.ptr = p.labelMap[label]
			return nil
		}
	case "JNZ":
		// Jump if ACC not zero
		if p.acc != 0 {
			label := tokens[1]
			p.ptr = p.labelMap[label]
			return nil
		}
	case "JGZ":
		// Jump if ACC greater than zero
		if p.acc > 0 {
			label := tokens[1]
			p.ptr = p.labelMap[label]
			return nil
		}
	case "JLZ":
		// Jump if ACC less than zero
		if p.acc < 0 {
			label := tokens[1]
			p.ptr = p.labelMap[label]
			return nil
		}
	case "JRO_VAL":
		// Jumps by value offset
		v, err := strconv.Atoi(tokens[1])
		if err != nil {
			return err
		}
		p.ptr = utils.IntClamp(p.ptr+v, 0, len(p.asm)-1)
		return nil
	case "JRO_SRC":
		// Jumps by value offset
		v, err := p.getFromSrc(tokens[1])
		if err != nil {
			return err
		}
		p.ptr = utils.IntClamp(p.ptr+v, 0, len(p.asm)-1)
		return nil
	case "PUSH_VAL":
		// Moves value across network
		v, err := strconv.Atoi(tokens[1])
		if err != nil {
			return err
		}
		err = p.pushValue(v, tokens[2])
		if err != nil {
			return err
		}
	case "PUSH_SRC":
		// Moves value from src across network
		v, err := p.getFromSrc(tokens[1])
		if err != nil {
			return err
		}
		err = p.pushValue(v, tokens[2])
		if err != nil {
			return err
		}
	case "POP":
		v, err := p.popValue(tokens[1])
		if err != nil {
			return err
		}
		switch tokens[2] {
		case "ACC":
			p.acc = v
		case "NIL":
			// no-op
		}
	case "IN":
		v, err := p.inputValue()
		if err != nil {
			return err
		}
		switch tokens[1] {
		case "ACC":
			p.acc = v
		case "NIL":
			// no-op
		}
	case "OUT_VAL":
		v, err := strconv.Atoi(tokens[1])
		if err != nil {
			return err
		}
		err = p.outputValue(v)
		if err != nil {
			return err
		}
	case "OUT_SRC":
		v, err := p.getFromSrc(tokens[1])
		if err != nil {
			return err
		}
		err = p.outputValue(v)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("'%v' not a valid instruction", tokens)
	}

	// Move instruction pointer
	p.ptr = (p.ptr + 1) % len(p.asm)

	return nil
}

// Gets value from src
func (p *ProgramNode) getFromSrc(src string) (int, error) {
	switch src {
	case "ACC":
		return p.acc, nil
	case "NIL":
		return 0, nil
	case "R0":
		select {
		case v := <-p.r0:
			return v, nil
		case <-p.ctx.Done():
			return 0, fmt.Errorf("register retrieval cancelled")
		}
	case "R1":
		select {
		case v := <-p.r1:
			return v, nil
		case <-p.ctx.Done():
			return 0, fmt.Errorf("register retrieval cancelled")
		}
	case "R2":
		select {
		case v := <-p.r2:
			return v, nil
		case <-p.ctx.Done():
			return 0, fmt.Errorf("register retrieval cancelled")
		}
	case "R3":
		select {
		case v := <-p.r3:
			return v, nil
		case <-p.ctx.Done():
			return 0, fmt.Errorf("register retrieval cancelled")
		}
	default:
		return 0, fmt.Errorf("'%s' not a valid src", src)
	}
}

// sendValue sends value to target in network
func (p *ProgramNode) sendValue(v int, target string) error {
	if m := regexp.MustCompile(`^(\w+):(R[0123])$`).FindStringSubmatch(target); len(m) > 0 {
		targetURI := m[1]

		// TODO: change parser to get ints
		var register int32
		switch m[2] {
		case "R0":
			register = 0
		case "R1":
			register = 1
		case "R2":
			register = 2
		case "R3":
			register = 3
		}

		conn, err := grpc.Dial(fmt.Sprintf("%s:8000", targetURI), grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}
		defer conn.Close()
		c := pb.NewProgramClient(conn)
		_, err = c.SendValue(p.ctx, &pb.SendValueRequest{Register: register, Value: int32(v)})
		if err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("'%s' not a valid network register", target)
}

// pushValue pushes value to target in network
func (p *ProgramNode) pushValue(v int, targetURI string) error {
	conn, err := grpc.Dial(fmt.Sprintf("%s:8000", targetURI), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewStackClient(conn)
	_, err = c.Push(p.ctx, &pb.PushValueRequest{Value: int32(v)})
	if err != nil {
		return err
	}
	return nil
}

// popValue pops value from source in network
func (p *ProgramNode) popValue(sourceURI string) (int, error) {
	conn, err := grpc.Dial(fmt.Sprintf("%s:8000", sourceURI), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewStackClient(conn)
	r, err := c.Pop(p.ctx, &pb.PopValueRequest{})
	if err != nil {
		return -1, err
	}
	return int(r.GetValue()), nil
}

// inputValue gets an input value from master node
func (p *ProgramNode) inputValue() (int, error) {
	conn, err := grpc.Dial(fmt.Sprintf("%s:8000", p.masterURI), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewMasterClient(conn)
	r, err := c.GetInput(p.ctx, &pb.InputValueRequest{})
	if err != nil {
		return -1, err
	}
	return int(r.GetValue()), nil
}

// outputValue outputs value to master node
func (p *ProgramNode) outputValue(v int) error {
	conn, err := grpc.Dial(fmt.Sprintf("%s:8000", p.masterURI), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewMasterClient(conn)
	_, err = c.SendOutput(p.ctx, &pb.OutputValueRequest{Value: int32(v)})
	if err != nil {
		return err
	}
	return nil
}
