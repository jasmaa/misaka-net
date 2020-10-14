package workers

import (
	"context"
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
	r0       chan int

	ctx    context.Context
	cancel context.CancelFunc
}

// NewMasterNode creates a new master node
func NewMasterNode(nodeURIs []string) *MasterNode {
	ctx, cancel := context.WithCancel(context.Background())
	return &MasterNode{
		nodeURIs: nodeURIs,
		r0:       make(chan int),
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

			// TODO: do proper reset for master as well

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

	http.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":

			if err := r.ParseForm(); err != nil {
				http.Error(w, "Cannot parse form", http.StatusBadRequest)
				return
			}

			rx := r.FormValue("register")
			v, err := strconv.Atoi(r.FormValue("value"))
			if err != nil {
				http.Error(w, "Cannot parse value", http.StatusBadRequest)
				return
			}

			switch rx {
			case "R0":
				m.r0 <- v
			default:
				http.Error(w, "Not a valid register", http.StatusBadRequest)
				return
			}

			fmt.Fprintf(w, "Success")
			log.Printf("Received value")

		default:
			http.Error(w, "Method GET not allowed", http.StatusMethodNotAllowed)
		}
	})

	// TODO: add handler for retrieving output

	log.Printf("Starting server...")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatal(err)
	}
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
