package workers

import (
	"fmt"
	"strings"

	"github.com/jasmaa/misaka-net/internal/tis"
)

// ProgramNode is a program node that interprets TIS-100 asm
type ProgramNode struct {
	acc int8
	bak int8
	r0  chan int8
	r1  chan int8
	r2  chan int8
	r3  chan int8

	ptr      int
	asm      [][]string
	labelMap map[string]int
}

// NewProgramNode creates a new program node
func NewProgramNode() *ProgramNode {
	return &ProgramNode{
		acc: 0,
		bak: 0,
		r0:  make(chan int8, 1),
		r1:  make(chan int8, 1),
		r2:  make(chan int8, 1),
		r3:  make(chan int8, 1),
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
