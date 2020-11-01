package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	pb "github.com/jasmaa/misaka-net/internal/grpc"
	"google.golang.org/grpc"
)

// NodeInfo contains information about nodes
type NodeInfo struct {
	Type string `json:"type"`
}

// MasterNode is a master node
type MasterNode struct {
	nodeInfo map[string]NodeInfo
	inChan   chan int
	outChan  chan int

	ctx       context.Context
	cancel    context.CancelFunc
	isRunning bool

	pb.UnimplementedStackServer
}

// inResponse structures response to in request
type inResponse struct {
	Value int `json:"value"`
}

// clientOutResponse structures response to client output request
type clientOutResponse struct {
	Value int `json:"value"`
}

// NewMasterNode creates a new master node
func NewMasterNode(nodeInfo map[string]NodeInfo) *MasterNode {
	ctx, cancel := context.WithCancel(context.Background())
	return &MasterNode{
		nodeInfo: nodeInfo,
		inChan:   make(chan int, bufferSize),
		outChan:  make(chan int, bufferSize),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start starts master node server
func (m *MasterNode) Start() {

	// TODO: authenticate commands from master with key
	// TODO: switch to faster protocol (grpc?)

	http.HandleFunc("/run", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			m.isRunning = true
			ctx, cancel := context.WithCancel(context.Background())
			m.ctx = ctx
			m.cancel = cancel

			err := m.broadcastCommand("run")
			if err != nil {
				log.Print(err)
				http.Error(w, fmt.Sprintf("Error running network: %s", err.Error()), http.StatusBadRequest)
				return
			}

			fmt.Fprintf(w, "Success")
		default:
			http.Error(w, "Method GET not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/pause", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			err := m.broadcastCommand("pause")
			if err != nil {
				log.Print(err)
				http.Error(w, fmt.Sprintf("Error pausing network: %s", err.Error()), http.StatusBadRequest)
				return
			}

			m.cancel()
			m.isRunning = false

			fmt.Fprintf(w, "Success")
		default:
			http.Error(w, "Method GET not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/reset", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			err := m.broadcastCommand("reset")
			if err != nil {
				log.Print(err)
				http.Error(w, fmt.Sprintf("Error resetting network: %s", err.Error()), http.StatusBadRequest)
				return
			}
			m.resetNode()
			fmt.Fprintf(w, "Success")
		default:
			http.Error(w, "Method GET not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/load", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":

			if err := r.ParseForm(); err != nil {
				http.Error(w, "Cannot parse form", http.StatusBadRequest)
				return
			}

			program := r.FormValue("program")
			targetURI := r.FormValue("targetURI")

			// Check if master knows target uri
			if _, ok := m.nodeInfo[targetURI]; !ok {
				err := fmt.Errorf("node %s not valid on this network", targetURI)
				log.Print(err)
				http.Error(w, fmt.Sprintf("error loading program on node %s: %s", targetURI, err.Error()), http.StatusBadRequest)
				return
			}

			// Reset network
			err := m.broadcastCommand("reset")
			if err != nil {
				log.Print(err)
				http.Error(w, fmt.Sprintf("error resetting network: %s", err.Error()), http.StatusBadRequest)
				return
			}
			m.resetNode()

			// Send load command to target node
			conn, err := grpc.Dial(fmt.Sprintf("%s:8000", targetURI), grpc.WithInsecure(), grpc.WithBlock())
			if err != nil {
				log.Fatalf("did not connect: %v", err)
			}
			defer conn.Close()
			c := pb.NewProgramClient(conn)
			_, err = c.Load(m.ctx, &pb.LoadRequest{Program: program})
			if err != nil {
				log.Print(err)
				http.Error(w, fmt.Sprintf("error loading program on node %s: %s", targetURI, err.Error()), http.StatusBadRequest)
				return
			}
			log.Printf("successfully loaded program")
			fmt.Fprintf(w, "Success")
		default:
			http.Error(w, "Method GET not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/in", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			select {
			case v := <-m.inChan:
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(inResponse{Value: v})
				log.Printf("Sent input value")
			case <-m.ctx.Done():
				http.Error(w, "Cannot send input value", http.StatusBadRequest)
				log.Printf("input retrieval cancelled")
			}
		default:
			http.Error(w, "Method GET not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/compute", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			if !m.isRunning {
				http.Error(w, "Network is not running", http.StatusBadRequest)
				return
			}

			if err := r.ParseForm(); err != nil {
				http.Error(w, "Cannot parse form", http.StatusBadRequest)
				return
			}

			v, err := strconv.Atoi(r.FormValue("value"))
			if err != nil {
				http.Error(w, "Cannot parse value", http.StatusBadRequest)
				return
			}

			m.inChan <- v

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(clientOutResponse{Value: <-m.outChan})
			log.Printf("Value outputted")

		default:
			http.Error(w, "Method GET not allowed", http.StatusMethodNotAllowed)
		}
	})

	log.Printf("Starting server...")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatal(err)
	}
}

// GetInput gets input from master node
func (m *MasterNode) GetInput(ctx context.Context, in *pb.InputValueRequest) (*pb.ValueReply, error) {
	select {
	case v := <-m.inChan:
		log.Printf("Sent input value")
		return &pb.ValueReply{Value: int32(v)}, nil
	case <-m.ctx.Done():
		log.Printf("input retrieval cancelled")
		return nil, fmt.Errorf("input retrieval cancelled")
	}
}

// SendOutput sends output to master node
func (m *MasterNode) SendOutput(ctx context.Context, in *pb.OutputValueRequest) (*pb.CommandReply, error) {
	m.outChan <- int(in.Value)
	log.Printf("received output value")
	return &pb.CommandReply{}, nil
}

// resetNode resets master node
func (m *MasterNode) resetNode() {
	m.cancel()
	m.isRunning = false

	m.inChan = make(chan int, bufferSize)
	m.outChan = make(chan int, bufferSize)
}

// broadcastCommand broadcasts specified command to all known nodes in network
func (m *MasterNode) broadcastCommand(cmd string) error {

	c := make(chan error)

	// Send command to all nodes
	for k, v := range m.nodeInfo {
		go func(cmd string, targetURI string, info NodeInfo) {
			switch info.Type {
			case "program":
				c <- m.broadcastCommandProgram(cmd, targetURI)
			case "stack":
				c <- m.broadcastCommandStack(cmd, targetURI)
			default:
				c <- fmt.Errorf("invalid node type")
			}
		}(cmd, k, v)
	}

	// Check if all nodes were successful
	for i := 0; i < len(m.nodeInfo); i++ {
		if err := <-c; err != nil {
			return err
		}
	}

	return nil
}

func (m *MasterNode) broadcastCommandProgram(cmd string, targetURI string) error {
	conn, err := grpc.Dial(fmt.Sprintf("%s:8000", targetURI), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewProgramClient(conn)
	switch cmd {
	case "run":
		_, err = c.Run(m.ctx, &pb.RunRequest{})
		if err != nil {
			return err
		}
	case "pause":
		_, err = c.Pause(m.ctx, &pb.PauseRequest{})
		if err != nil {
			return err
		}
	case "reset":
		_, err = c.Reset(m.ctx, &pb.ResetRequest{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MasterNode) broadcastCommandStack(cmd string, targetURI string) error {
	conn, err := grpc.Dial(fmt.Sprintf("%s:8000", targetURI), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewStackClient(conn)
	switch cmd {
	case "run":
		_, err = c.Run(m.ctx, &pb.RunRequest{})
		if err != nil {
			return err
		}
	case "pause":
		_, err = c.Pause(m.ctx, &pb.PauseRequest{})
		if err != nil {
			return err
		}
	case "reset":
		_, err = c.Reset(m.ctx, &pb.ResetRequest{})
		if err != nil {
			return err
		}
	}
	return nil
}
