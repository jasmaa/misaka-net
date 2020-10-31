package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jasmaa/misaka-net/internal/nodes"
)

func main() {

	nodeType := os.Getenv("NODE_TYPE")

	switch nodeType {
	case "program":
		p := nodes.NewProgramNode(os.Getenv("MASTER_URI"))
		err := p.Load(os.Getenv("PROGRAM"))
		if err != nil {
			log.Printf("Could not load default program: %s", err.Error())
		}
		p.Start()
	case "stack":
		s := nodes.NewStackNode()
		s.Start()
	case "master":
		nodeURIs := strings.Split(os.Getenv("NODE_URIS"), ",")
		m := nodes.NewMasterNode(nodeURIs)
		m.Start()
	default:
		panic(fmt.Errorf("'%s' not a valid node type", nodeType))
	}
}
