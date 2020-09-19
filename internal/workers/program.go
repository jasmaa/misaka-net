package workers

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ProgramNode is a program node
type ProgramNode struct {
	acc int8
	bak int8
	r1  chan int8
	r2  chan int8
	r3  chan int8
	r4  chan int8

	ptr      int
	asm      []string
	labelMap map[string]int
}

// NewProgramNode creates a new program node
func NewProgramNode() *ProgramNode {
	return &ProgramNode{
		acc: 0,
		bak: 0,
		r1:  make(chan int8, 1),
		r2:  make(chan int8, 1),
		r3:  make(chan int8, 1),
		r4:  make(chan int8, 1),
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
	asm := strings.Split(s, "\n")
	labelMap, err := generateLabelMap(asm)

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
			p.Update()
		}
	}()

	go func() {
		// TODO: put http handler here
	}()

	<-forever
}

// Update steps through asm
func (p *ProgramNode) Update() {
	instr := p.asm[p.ptr]

	// TODO: interpret instr
	_ = instr
	fmt.Println(instr)

	p.ptr = (p.ptr + 1) % len(p.asm)
}

// Nop is a no-op
func (p *ProgramNode) Nop() {

}

// Swp swaps ACC and BAK
func (p *ProgramNode) Swp() {
	temp := p.acc
	p.acc = p.bak
	p.bak = temp
}

// Sav saves ACC to BAK
func (p *ProgramNode) Sav() {
	p.bak = p.acc
}

// Neg negates value in Acc
func (p *ProgramNode) Neg() {
	p.acc = -p.acc
}

// Jmp jumps to label
func (p *ProgramNode) Jmp(label string) {

}

// Finds and generates map of labels to index
func generateLabelMap(asm []string) (map[string]int, error) {
	labelRe := regexp.MustCompile("^\\s*(\\w+):")
	labelMap := make(map[string]int)

	for i, line := range asm {
		matches := labelRe.FindStringSubmatch(line)
		if len(matches) == 2 {
			label := strings.ToUpper(matches[1])
			if _, ok := labelMap[label]; ok {
				return nil, errors.New("Cannot repeat label")
			}
			labelMap[label] = i
		}
	}
	return labelMap, nil
}
