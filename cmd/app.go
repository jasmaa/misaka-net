package main

import (
	"fmt"

	"github.com/jasmaa/misaka-net/internal/workers"
)

func main() {

	// TODO: have an interface for nodes
	// TODO: read in config from env var

	nodeType := "master"

	switch nodeType {
	case "program":
		p := workers.NewProgramNode()
		// TEMP: load default program
		p.Load(`START:
		    MOV R0, ACC
		    JGZ POSITIVE
		    JLZ NEGATIVE
		    JMP START
		POSITIVE: MOV ACC, comp1:R1
		    JMP START
		NEGATIVE:
		    MOV ACC, comp1:R3
		    JMP START`)
		p.Start()
	case "stack":
		// TODO: create stack node
	case "master":
		// TEMP: dummy node uris
		nodeURIs := []string{"endpoint1", "endpoint2", "endpoint3"}
		m := workers.NewMasterNode(nodeURIs)
		m.Start()
	default:
		panic(fmt.Errorf("not a valid node type"))
	}
}
