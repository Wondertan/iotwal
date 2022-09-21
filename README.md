# If Only Tendermint Was A Library

## Manifest
Tendermint software is empowering the Cosmos ecosystem and holds billions of value worldwide. Its main goal is providing
BFT POS consensus engine/service for anyone to build custom blockchains on top. The software accomplishes the goal good
enough together with Cosmos SDK. However...

As everything in our world, nothing is perfect, and Tendermint is not an exception. Particularly, it's unbelievably far 
from that point. It has been in development for many years(8 from the moment of writing) and haven't seen any bottom to top 
revisions for too long. Recently, a ray of light broke through and there was an [attempt](https://medium.com/tendermint/tendermint-v0-35-introduces-prioritized-mempool-a-makeover-to-the-peer-to-peer-network-more-61eea6ec572d) 
to do such a revision for the bottom layer of the stack - p2p networking, opening possibility for more pervasive improvements.

But..., it was a complete [failure](https://interchain-io.medium.com/discontinuing-tendermint-v0-35-a-postmortem-on-the-new-networking-layer-3696c811dabc)
and PITA for teams trying to use it. As a result, the failed attempt prompted lots of internal drama, which in the end
makes all the involved stakeholders even more conservative to drastic changes like this, tightening nuts for any fundamental 
revisions that the software starves for. Bags are full(of comfort obviously) so why would anyone need to risk?

Regret it or not, but the poor decisions on the lower level of the stack affect the whole stack built or not, no matter
what tech it is. The stack over Tendermint's legacy blooms and flourishes(Cosmos SDK, IBC, IgniteCLI, Gno), millions are
invested in the ecosystem to buidl, to build over ABCI(++)... ABCI is a general purpose interface 

* ABCI ++ example
* Mempool support
* Fraud Provability

We can now see this happening with ABCI - the main abstraction in Tendermint allowing everyone to build

general purpose applications over the engine. Its [original motivation](https://docs.tendermint.com/v0.34/introduction/what-is-tendermint.html#motivation)
is to be modular 

* Monolithic
* Inefficient
  * Bandwidth
* Opinionated
* Limited
* Visibility to errors



Modularization Steps
* Extract consensus pkg
  * Remove metrics and WAL
    * Will be added back at later stages of IOTWAL
  * Remove BaseService
    * Pkg global logger
  * Remove Blockstore
    * Consensus is only responsible for finding consensus! 
  * Extract required types into consensus pkg
    * Not block
      * Pulls whole type pkg what we want to avoid
  * Remove BaseReactor + migrate to libp2p
