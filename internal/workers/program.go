package workers

import (
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

	isRunning bool
	resetFlag bool
}

// NewProgramNode creates a new program node
func NewProgramNode() *ProgramNode {
	return &ProgramNode{
		acc: 0,
		bak: 0,
		r0:  make(chan int, bufferSize),
		r1:  make(chan int, bufferSize),
		r2:  make(chan int, bufferSize),
		r3:  make(chan int, bufferSize),
		asm: [][]string{[]string{"NOP"}},
	}
}

// Start starts program loop and server
func (p *ProgramNode) Start() {
	// Run program loop
	go func() {
		for {
			if p.isRunning {

				// Cancel send if node is reset
				// TODO: make this less jank, use context.WithCancel??

				c := make(chan error)

				go func() {
					c <- p.update()
				}()

				go func() {
					for {
						if p.resetFlag {
							c <- nil
						}
						time.Sleep(1 * time.Second)
					}
				}()

				if err := <-c; err != nil {
					log.Fatal(err)
				}
			}
		}
	}()

	http.HandleFunc("/run", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			p.Run()
			fmt.Fprintf(w, "Success")
			log.Printf("Node is running")
		default:
			http.Error(w, "Method GET not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/pause", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			p.Pause()
			fmt.Fprintf(w, "Success")
			log.Printf("Node is paused")
		default:
			http.Error(w, "Method GET not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/reset", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			p.Reset()
			fmt.Fprintf(w, "Success")
			log.Printf("Node was reset")
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

	fmt.Println("Starting server...")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatal(err)
	}
}

// Run starts asm execution
func (p *ProgramNode) Run() {
	p.isRunning = true
	p.resetFlag = false
}

// Pause pauses asm execution
func (p *ProgramNode) Pause() {
	p.isRunning = false
}

// Reset resets asm execution and registers
func (p *ProgramNode) Reset() {
	p.isRunning = false
	p.resetFlag = true

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

	fmt.Println(tokens)
	fmt.Printf("ACC: %v\n", p.acc)
	fmt.Printf("BAK: %v\n", p.bak)
	fmt.Println("---")

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

		if m := regexp.MustCompile(`^(\w+):(R[0123])$`).FindStringSubmatch(tokens[2]); len(m) > 0 {
			targetURI := m[1]
			register := m[2]

			payload := url.Values{}
			payload.Set("register", register)
			payload.Set("value", strconv.Itoa(v))

			resp, err := http.PostForm(fmt.Sprintf("http://%s:8000/send", targetURI), payload)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("Network move was not successful")
			}
		} else {
			return fmt.Errorf("'%s' not a valid network register", tokens[2])
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
		// TODO: figure out networking
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
		return <-p.r0, nil
	case "R1":
		return <-p.r1, nil
	case "R2":
		return <-p.r2, nil
	case "R3":
		return <-p.r3, nil
	default:
		return 0, fmt.Errorf("'%s' not a valid src", src)
	}
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
