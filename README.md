# Misaka Net

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

### Single node locally

A single node can be setup by setting proper environment variables and building and running:

    make
    ./app

### Deploy Network with Docker Compose

The provided compose file sets up an example network with
two program nodes that add and pass an integer back-and-forth
with one of the program nodes also pushing and popping the value to and from a stack node.

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

The client can send and retrieve data through the master node:

    # TODO