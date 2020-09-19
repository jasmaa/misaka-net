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
	labelMap, err := generateLabelMap(instrArr)

	if err != nil {
		return err
	}

	// Map instructions
	asm := make([][]string, len(instrArr))
	for i, instr := range instrArr {
		// Get rid of labels and whitespace
		prefixRe := regexp.MustCompile(`^(\s*\w+:)?\s*`)
		if indices := prefixRe.FindStringIndex(instr); indices != nil {
			end := indices[1]
			instr = instr[end:]
		}

		// Convert instr to list
		if len(instr) == 0 {
			// <Label>:
			asm[i] = []string{"NOP"}
		} else if m := regexp.MustCompile(`^#.*$`).FindStringSubmatch(instr); len(m) > 0 {
			// #<Comment>
			asm[i] = []string{"NOP"}
		} else if m := regexp.MustCompile(`^NOP\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// NOP
			asm[i] = []string{"NOP"}
		} else if m := regexp.MustCompile(`^MOV\s+(\d+)\s*,\s+(ACC|NIL|\w+:R[0123])\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// MOV <VAL>, <DST>
			asm[i] = []string{"MOV", m[1], m[2]}
		} else if m := regexp.MustCompile(`^MOV\s+(ACC|NIL|R[0123])\s*,\s+(ACC|NIL|\w+:R[0123])\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// MOV <SRC>, <DST>
			asm[i] = []string{"MOV", m[1], m[2]}
		} else if m := regexp.MustCompile(`^SWP\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// SWP
			asm[i] = []string{"SWP"}
		} else if m := regexp.MustCompile(`^SAV\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// SAV
			asm[i] = []string{"SAV"}
		} else if m := regexp.MustCompile(`^ADD\s+(\d+)\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// ADD <VAL>
			asm[i] = []string{"ADD", m[1]}
		} else if m := regexp.MustCompile(`^ADD\s+(ACC|NIL|R[0123])\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// ADD <SRC>
			asm[i] = []string{"ADD", m[1]}
		} else if m := regexp.MustCompile(`^SUB\s+(\d+)\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// SUB <VAL>
			asm[i] = []string{"SUB", m[1]}
		} else if m := regexp.MustCompile(`^SUB\s+(ACC|NIL|R[0123])\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// SUB <SRC>
			asm[i] = []string{"SUB", m[1]}
		} else if m := regexp.MustCompile(`^NEG\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// NEG
			asm[i] = []string{"NEG"}
		} else if m := regexp.MustCompile(`^JMP\s+(\w+)\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// JMP <LABEL>
			asm[i] = []string{"JMP", strings.ToUpper(m[1])}
		} else if m := regexp.MustCompile(`^JEZ\s+(\w+)\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// JEZ <LABEL>
			asm[i] = []string{"JEZ", strings.ToUpper(m[1])}
		} else if m := regexp.MustCompile(`^JNZ\s+(\w+)\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// JNZ <LABEL>
			asm[i] = []string{"JNZ", strings.ToUpper(m[1])}
		} else if m := regexp.MustCompile(`^JGZ\s+(\w+)\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// JGZ <LABEL>
			asm[i] = []string{"JGZ", strings.ToUpper(m[1])}
		} else if m := regexp.MustCompile(`^JLZ\s+(\w+)\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// JLZ <LABEL>
			asm[i] = []string{"JLZ", strings.ToUpper(m[1])}
		} else if m := regexp.MustCompile(`^JRO\s+(0|-1|2|\w+)\s*$`).FindStringSubmatch(instr); len(m) > 0 {
			// JRO <LABEL>
			asm[i] = []string{"JRO", m[1]}
		} else {
			return errors.New(instr)
		}
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

// Finds and generates map of labels to index
func generateLabelMap(asm []string) (map[string]int, error) {
	labelRe := regexp.MustCompile(`^\s*(\w+):`)
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
