package utils

import (
	"fmt"
	"sync"
)

// IntStack is a stack of integers
type IntStack struct {
	stack []int
	mux   sync.RWMutex
}

// NewIntStack creates a new int stack
func NewIntStack() *IntStack {
	return &IntStack{stack: make([]int, 0)}
}

// Push pushes value to stack
func (s *IntStack) Push(v int) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.stack = append(s.stack, v)
}

// Pop pops value at head of stack
func (s *IntStack) Pop() (int, error) {
	if l := len(s.stack); l > 0 {
		s.mux.Lock()
		defer s.mux.Unlock()

		v := s.stack[l-1]
		s.stack = s.stack[:l-1]
		return v, nil
	}

	return -1, fmt.Errorf("stack is empty")
}

// Clear clears stack
func (s *IntStack) Clear() {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.stack = make([]int, 0)
}
