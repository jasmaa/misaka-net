package workers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jasmaa/misaka-net/internal/tis"
)

// Register buffer size
const bufferSize = 1

// ProgramNode is a program node that interprets TIS-100 asm
type ProgramNode struct {
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
}

// NewProgramNode creates a new program node
func NewProgramNode() *ProgramNode {
	ctx, cancel := context.WithCancel(context.Background())
	return &ProgramNode{
		acc:    0,
		bak:    0,
		r0:     make(chan int, bufferSize),
		r1:     make(chan int, bufferSize),
		r2:     make(chan int, bufferSize),
		r3:     make(chan int, bufferSize),
		asm:    [][]string{[]string{"NOP"}},
		ctx:    ctx,
		cancel: cancel,
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

	http.HandleFunc("/run", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			if !p.isRunning {
				p.Run()
				log.Printf("Node is running")
			} else {
				log.Printf("Node is already running")
			}
			fmt.Fprintf(w, "Success")
		default:
			http.Error(w, "Method GET not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/pause", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			if p.isRunning {
				p.Pause()
				log.Printf("Node is paused")
			} else {
				log.Printf("Node is already paused")
			}
			fmt.Fprintf(w, "Success")
		default:
			http.Error(w, "Method GET not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/reset", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			p.Reset()
			log.Printf("Node was paused and reset")
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
			err := p.Load(program)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error loading program: %s", err.Error()), http.StatusBadRequest)
				return
			}

			fmt.Fprintf(w, "Success")
			log.Printf("Program was loaded")
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
				p.r0 <- v
			case "R1":
				p.r1 <- v
			case "R2":
				p.r2 <- v
			case "R3":
				p.r3 <- v
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

	log.Printf("Starting server...")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatal(err)
	}
}

// Run starts asm execution
func (p *ProgramNode) Run() {
	p.isRunning = true

	ctx, cancel := context.WithCancel(context.Background())
	p.ctx = ctx
	p.cancel = cancel
}

// Pause pauses asm execution
func (p *ProgramNode) Pause() {
	p.cancel()
	p.isRunning = false
}

// Reset resets asm execution and registers
func (p *ProgramNode) Reset() {
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

// Load resets node and loads asm program
func (p *ProgramNode) Load(s string) error {
	// Reset node
	p.Reset()

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

// Update steps through asm
func (p *ProgramNode) update() error {
	tokens := p.asm[p.ptr]

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
		// TODO: reduce code reuse
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
		p.ptr = max(0, min(p.ptr+v, len(p.asm)-1))
		return nil
	case "JRO_SRC":
		// Jumps by value offset
		v, err := p.getFromSrc(tokens[1])
		if err != nil {
			return err
		}
		p.ptr = max(0, min(p.ptr+v, len(p.asm)-1))
		return nil
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
		register := m[2]

		payload := url.Values{}
		payload.Set("register", register)
		payload.Set("value", strconv.Itoa(v))

		client := http.DefaultClient
		req, _ := http.NewRequestWithContext(
			p.ctx,
			"POST",
			fmt.Sprintf("http://%s:8000/send", targetURI),
			strings.NewReader(payload.Encode()),
		)
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("Content-Length", strconv.Itoa(len(payload.Encode())))

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return fmt.Errorf("Network move was not successful")
		}

		return nil
	}

	return fmt.Errorf("'%s' not a valid network register", target)
}

// Max finds maximum of two ints
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Min finds minimum of two ints
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
