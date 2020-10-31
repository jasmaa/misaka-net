package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// MasterNode is a master node
type MasterNode struct {
	nodeURIs []string
	inChan   chan int
	outChan  chan int

	ctx       context.Context
	cancel    context.CancelFunc
	isRunning bool
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
func NewMasterNode(nodeURIs []string) *MasterNode {
	ctx, cancel := context.WithCancel(context.Background())
	return &MasterNode{
		nodeURIs: nodeURIs,
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
			m.Run()
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
			m.Pause()
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
			m.Reset()
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
			isURIFound := false
			for _, v := range m.nodeURIs {
				if v == targetURI {
					isURIFound = true
					break
				}
			}
			if !isURIFound {
				err := fmt.Errorf("node %s not valid on this network", targetURI)
				log.Print(err)
				http.Error(w, fmt.Sprintf("Error loading program on node %s: %s", targetURI, err.Error()), http.StatusBadRequest)
				return
			}

			// Reset network
			err := m.broadcastCommand("reset")
			if err != nil {
				log.Print(err)
				http.Error(w, fmt.Sprintf("Error resetting network: %s", err.Error()), http.StatusBadRequest)
				return
			}
			m.Reset()

			// Send load command to target node
			payload := url.Values{}
			payload.Set("program", program)

			client := http.DefaultClient
			req, _ := http.NewRequestWithContext(
				m.ctx,
				"POST",
				fmt.Sprintf("http://%s:8000/load", targetURI),
				strings.NewReader(payload.Encode()),
			)
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Add("Content-Length", strconv.Itoa(len(payload.Encode())))

			resp, err := client.Do(req)
			if err != nil {
				log.Print(err)
				http.Error(w, fmt.Sprintf("Error loading program on node %s: %s", targetURI, err.Error()), http.StatusBadRequest)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == 200 {
				log.Printf("Program was successfully loaded on node %s", targetURI)
				fmt.Fprintf(w, "Success")
			} else {
				log.Printf("Error loading program on node %s", targetURI)
				http.Error(w, fmt.Sprintf("Error loading program: %s", err.Error()), http.StatusBadRequest)
			}
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

	http.HandleFunc("/out", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Cannot parse form", http.StatusBadRequest)
				return
			}

			v, err := strconv.Atoi(r.FormValue("value"))
			if err != nil {
				http.Error(w, "Cannot parse value", http.StatusBadRequest)
				return
			}

			m.outChan <- v

			fmt.Fprintf(w, "Success")
			log.Printf("Received output value")

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

// Run runs master node
func (m *MasterNode) Run() {
	m.isRunning = true

	ctx, cancel := context.WithCancel(context.Background())
	m.ctx = ctx
	m.cancel = cancel
}

// Pause pauses asm execution
func (m *MasterNode) Pause() {
	m.cancel()
	m.isRunning = false
}

// Reset resets master node
func (m *MasterNode) Reset() {
	m.cancel()
	m.isRunning = false

	m.inChan = make(chan int, bufferSize)
	m.outChan = make(chan int, bufferSize)
}

// broadcastCommand broadcasts specified command to all known nodes in network
func (m *MasterNode) broadcastCommand(cmd string) error {

	c := make(chan error)

	// Send command to all nodes
	for _, v := range m.nodeURIs {
		go func(targetURI string) {
			client := http.DefaultClient
			req, _ := http.NewRequestWithContext(
				m.ctx,
				"POST",
				fmt.Sprintf("http://%s:8000/%s", targetURI, cmd),
				nil,
			)

			log.Printf("Sending command '%s' to node at %s...", cmd, targetURI)
			resp, err := client.Do(req)
			if err != nil {
				c <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == 200 {
				c <- nil
			} else {
				c <- fmt.Errorf("Error sending command '%s' to node at %s", cmd, targetURI)
			}
		}(v)
	}

	// Check if all nodes were successful
	for i := 0; i < len(m.nodeURIs); i++ {
		if err := <-c; err != nil {
			return err
		}
	}

	return nil
}
