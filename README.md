<h1 style="display: flex; align-items: center;">
    <img src="docs/last_order.png" width="40rem" height="40rem" /> Misaka Net
</h1>

TIS-100-like distributed computing

![Network diagram](/docs/diagram.png)

## What is This?
Misaka Net is a distributed computing system inspired by both the assembly programing game, TIS-100,
and the Misaka Network from the Raildex franchise.

A Misaka Net is comprised of an arbitrary number of program and stack nodes managed by one master node
which can all communicate with each other. The network does computations using program nodes
which are loaded with assembly programs resembling those found in TIS-100 and run independently on loop.
Program nodes can send data to other program nodes, push/pop data to/from stack nodes, and send output data
to the master node. The network itself can be interacted with by the client through the master node
which broadcasts commands to all nodes and manages IO from the client.

## Setup

Follow [gRPC setup instructions](https://grpc.io/docs/languages/go/quickstart/) to re-generate gRPC Go code with:

    make grpc

Make sure OpenSSL is installed and generate certificate and private key with:

    make cert

### Deploy single node

A single node can be setup by setting proper environment variables and building and running:

    make
    ./app

### Deploy a network with Docker Compose

The provided compose file sets up an example network with
two program nodes and a stack node. In the example network, one program node receives
input from the master node, adds 1 to the input, and passes it to the other program node
which also adds 1 to it, pushes and pops it from the stack node, and then passes it back
to the original program node. The original program node then sends the final
value to the master node's output.

The network can be built and deployed with:

    docker-compose up --build

## Controlling the Network

The network is controlled by sending commands to the master node which
sends them to a particular node or broadcasts them throughout the network.

In order to run all nodes, send a run request to the master node with:

    curl -X POST <DOCKER MACHINE IP>:8000/run

The network's execution can be paused with:

    curl -X POST <DOCKER MACHINE IP>:8000/pause

Or reset with:

    curl -X POST <DOCKER MACHINE IP>:8000/reset

The master node can also be directed to load a program onto a particular node which
will reset the network's execution and update the specified node:

    curl -X POST \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "program=<PROGRAM SOURCE>&targetURI=<TARGET URI>" \
    <DOCKER MACHINE IP>:8000/load

Once the network is running, the client can send inputs and receive computed results through the master node:

    curl -X POST \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "value=<VALUE>" \
    <DOCKER MACHINE IP>:8000/compute