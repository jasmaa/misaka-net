# Notes

## Worker Node Types
  - `interp`: TIS-100-like interpreter
  
  - `workers`: Compute node types
    - `master`: Master node for controlling all nodes on net
      - `POST /start`: Starts computation for all nodes
      - `POST /pause`: Pause computation for all nodes
      - `POST /reset`: Stops and resets computation on all nodes
      - `POST /load`: Makes master load program onto specified program node. Resets all nodes
      - `POST /send`: Sends data to register on node
    
    - `program`: Program node for executing asm
      - `POST /start`: Starts computation
      - `POST /pause`: Pause computation
      - `POST /reset`: Stops and resets computation
      - `POST /load`: Loads program
      - `POST /send`: Sends data to register on node
    
    - `stack`: Stack node
      - `POST /reset`: Clears stack and registers
      - `POST /send`: Pushes data on head