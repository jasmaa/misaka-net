package main

import "github.com/jasmaa/misaka-net/internal/tis"

func main() {
	t, err := tis.New(`START:
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

	_ = t

}
