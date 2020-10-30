package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/jasmaa/misaka-net/internal/utils"
)

// StackNode is a stack node
type StackNode struct {
	stack *utils.IntStack

	ctx    context.Context
	cancel context.CancelFunc
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

	http.HandleFunc("/run", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			s.Run()
			log.Printf("Node was run")
			fmt.Fprintf(w, "Success")
		default:
			http.Error(w, "Method GET not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/pause", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			log.Printf("Node was paused")
			fmt.Fprintf(w, "Success")
		default:
			http.Error(w, "Method GET not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/reset", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			s.Reset()
			log.Printf("Node was paused and reset")
			fmt.Fprintf(w, "Success")
		default:
			http.Error(w, "Method GET not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/push", func(w http.ResponseWriter, r *http.Request) {
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

			s.stack.Push(v)

			fmt.Fprintf(w, "Success")
			log.Printf("Value was pushed")
		default:
			http.Error(w, "Method GET not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/pop", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			v, err := s.waitPop()
			if err != nil {
				log.Print(err)
				http.Error(w, "Cannot pop value", http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(popResponse{Value: v})
			log.Printf("Value was popped")
		default:
			http.Error(w, "Method GET not allowed", http.StatusMethodNotAllowed)
		}
	})

	log.Printf("Starting server...")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatal(err)
	}
}

// Run runs stack node
func (s *StackNode) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	s.ctx = ctx
	s.cancel = cancel
}

// Reset resets stack node
func (s *StackNode) Reset() {
	s.cancel()
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
