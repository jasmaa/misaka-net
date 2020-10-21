# Notes

## Architecture

  - Program Nodes
    - `ACC`: Read-write register for ints.
    - `BAK`: Register for ints. Only accessible via `SAV` and `SWP`.
    - `RX`: Host read-only, peer write-only registers.
    - `some_comp_name:RX`: Host write-only, peer read-only registers. Represent `RX` on another machine.

  - Stack Nodes
    - `stack`: Holds some number of ints in a stack.

## Added Instructions
  - `PUSH <VAL>, <LOC>`: Pushes `<VAL>` to stack node at `<LOC>`. Fails if `<LOC>` not stack node.
  - `PUSH <SRC>, <LOC>`: Pushes value in `<SRC>` to stack node at `<LOC>`. Fails if `<LOC>` not stack node.
  - `POP <LOC>, <SRC>`: Pops head from stack node at `<LOC>` to `<SRC>` on machine. Fails if `<LOC>` not stack node.

## Worker Node Types
  - `tis`: Functions to work with TIS-100-like asm
  
  - `workers`: Compute node types
    - `master`: Master node for controlling all nodes on net
      - `POST /run`: Starts computation for all nodes
      - `POST /pause`: Pause computation for all nodes
      - `POST /reset`: Stops and resets computation on all nodes
      - `POST /load`: Makes master load program onto specified program node. Resets all nodes
      - `POST /send`: Sends data to register on node
    
    - `program`: Program node for executing asm
      - `POST /run`: Starts computation
      - `POST /pause`: Pause computation
      - `POST /reset`: Stops and resets computation
      - `POST /load`: Loads program
      - `POST /send`: Sends data to register on node
    
    - `stack`: Stack node
      - `POST /reset`: Clears stack and registers
      - `POST /push`: Pushes data on head
      - `POST /pop`: Pops data from head