## Design Overview

This is the high-level architecture of ThemiX, it consists of three components:
- `noise layer`
- `consensus layer`
- `proposer`

![themix-overview](./picture/framework.jpg)

### Noise Layer
Noise layer uses a p2p library to communication with other themix nodes. In current implementation, the noise layer has many components:
- Noise node, which broadcasts and receives messages from other themix nodes.
- Verify pool, which verifies message signature.

### Consensus Layer
Consensus layer is the layer to run themix consensus. This layer consists of a queue of themix. Themix adapts ACS (Asynchronous Common Subset) framework, which consists of RBC (Reliable Broadcast) and ABA (Asynchronous Binary Agreement).

### Proposer
Proposer communicates with client using gRPC, the message sent between them is in the form of protobuf.

## Under Construction
This repo is still under construction.

### Main Tasks
- [ ] Liveness is broken in some case, not stable

### Evaluation Tasks
- [ ] Signature collection verification 
