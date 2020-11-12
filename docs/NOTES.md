# Notes

## Project Layout
  - `grpc`: Protobuf definitions and generated code to work with gRPC
  - `tis`: Functions to work with TIS-100-like asm
  - `nodes`: Code for master, program, and stack nodes
  - `utils`: Utility functions


## Architecture
  - Program Nodes
    - `ACC`: Read-write register for ints.
    - `BAK`: Register for ints. Only accessible via `SAV` and `SWP`.
    - `RX`: Host read-only, peer write-only registers.
    - `some_comp_name:RX`: Host write-only, peer read-only registers. Represent `RX` on another machine.

  - Stack Nodes
    - `stack`: Holds some number of ints in a stack.


## Added ASM Instructions
  - `PUSH <VAL>, <DST>`: Pushes `<VAL>` to stack node at `<DST>`. Fails if `<DST>` not stack node.
  - `PUSH <SRC>, <DST>`: Pushes value in `<SRC>` to stack node at `<LOC>`. Fails if `<DST>` not stack node.
  - `POP <SRC>`: Pops head from stack node at `<SRC>` to `ACC` on machine. Fails if `<SRC>` not stack node.
  - `IN <DST>`: Moves a value from input in master to `<DST>`
  - `OUT <VAL/SRC>`: Moves `<VAL/SRC>` in master output


## Node Types and Methods
  - Master: Node for controlling all nodes on net
    - Client methods:
      - `POST /run`: Starts computation for all nodes
      - `POST /pause`: Pause computation for all nodes
      - `POST /reset`: Stops and resets computation on all nodes
      - `POST /load`: Makes master load program onto specified program node. Resets all nodes
      - `POST /compute`: Puts received value into input and waits for network to compute output
    - RPC:
      - `rpc GetInput`: Returns value in input to requester
      - `rpc SendOutput`: Puts recevied value from requester into output
    
  - Program: Node for executing asm
      - `rpc Run`: Starts computation
      - `rpc Pause`: Pause computation
      - `rpc Reset`: Stops and resets computation
      - `rpc Load`: Loads program
      - `rpc SendValue`: Sends data to register on node
    
  - Stack: Node for stack storage
      - `rpc Run`: Starts computation
      - `rpc Pause`: Pause computation
      - `rpc Reset`: Clears stack and registers
      - `rpc Push`: Pushes data on head
      - `rpc Pop`: Pops data from head


## Adding Nodes to the Network in Docker Compose
  - Add new node as a service in `docker-compose.yml` with proper env vars
  - Update master node's `NODE_INFO` in `docker-compose.yml` to include name and type of new node
  - Update `alt_names` in `./openssl/certificate.conf` to include name of new service