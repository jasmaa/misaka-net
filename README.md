# Misaka Net

TIS-100-like distributed computing

## Setup

### Local

Set proper environment variables and build and run:

    make
    ./app

### Deploy with Docker Compose

The provided compose file sets up a master node and two program nodes
that pass an integer back-and-forth, adding 1 to it each time.

The network can be built and deployed with:

    docker-compose up --build

## Controlling the Network

The network is controlled by sending commands to the master node which
sends them to a particular node or broadcasts them throughout the network.

![Network diagram](/docs/diagram.png)

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

Date can be sent and retrieved from the network through:

    # TODO