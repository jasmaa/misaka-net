package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/jasmaa/misaka-net/internal/nodes"
)

func main() {

	nodeType := os.Getenv("NODE_TYPE")
	certFile := os.Getenv("CERT_FILE")
	keyFile := os.Getenv("KEY_FILE")

	switch nodeType {
	case "program":
		p := nodes.NewProgramNode(os.Getenv("MASTER_URI"), certFile, keyFile)
		err := p.LoadProgram(os.Getenv("PROGRAM"))
		if err != nil {
			log.Printf("Could not load default program: %s", err.Error())
		}
		p.Start()
	case "stack":
		s := nodes.NewStackNode(certFile, keyFile)
		s.Start()
	case "master":
		var nodeInfo map[string]nodes.NodeInfo
		err := json.Unmarshal([]byte(os.Getenv("NODE_INFO")), &nodeInfo)
		if err != nil {
			panic(fmt.Errorf("invalid node info"))
		}
		m := nodes.NewMasterNode(nodeInfo, certFile, keyFile)
		m.Start()
	default:
		panic(fmt.Errorf("'%s' not a valid node type", nodeType))
	}
}
