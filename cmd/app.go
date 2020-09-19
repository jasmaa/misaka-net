package main

import "github.com/jasmaa/misaka-net/internal/workers"

func main() {
	p := workers.NewProgramNode()
	err := p.Load(`START:
	MOV R0, ACC
	JGZ POSITIVE
	JLZ NEGATIVE
	JMP START
POSITIVE: MOV ACC, comp1:R1
	JMP START
NEGATIVE:
	MOV ACC, comp1:R3
	JMP START`)

	if err != nil {
		panic(err)
	}

	p.Run()
}
