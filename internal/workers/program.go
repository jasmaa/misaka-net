package workers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jasmaa/misaka-net/internal/tis"
)

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
}

// NewProgramNode creates a new program node
func NewProgramNode() *ProgramNode {
	return &ProgramNode{
		acc: 0,
		bak: 0,
		r0:  make(chan int),
		r1:  make(chan int),
		r2:  make(chan int),
		r3:  make(chan int),
	}
}

// Reset resets program node execution
func (p *ProgramNode) Reset() {
	p.acc = 0
	p.bak = 0
	p.ptr = 0
}

// Load loads asm program
func (p *ProgramNode) Load(s string) error {

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

// Run runs program node
func (p *ProgramNode) Run() {
	forever := make(chan interface{})

	go func() {
		for {
			err := p.update()
			if err != nil {
				panic(err)
			}
		}
	}()

	go func() {
		// TODO: put http handler here
		for {
		}
	}()

	// TEMP: test put into register
	go func() {
		for {
			time.Sleep(1 * time.Second)
			p.r0 <- 42
		}
	}()

	<-forever
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

		// TODO: figure out networking
		_ = v

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
