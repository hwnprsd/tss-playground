package node

import (
	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/tss"
)

// Handle multiple signs
func (n *Node) InitKeygen() {
	// Overwrite the localParty and create a new one
	// TODO: Handle different cohorts of local parties?
	// FIXME: Should not overwrite if an ongoing round is happening
	n.SetupKgLocalParty()

	go func() {
		err := (*n.kgParty).Start()
		if err != nil {
			n.logger.Sugar().Fatal(err)
		}
	}()
}

// Setup the local party & it's listeners
func (n *Node) SetupKgLocalParty() {
	n.logger.Info("Setting up KG local party...")

	parties := n.GetPartiesSorted()
	peerCtx := tss.NewPeerContext(parties)
	params := tss.NewParameters(tss.S256(), peerCtx, n.pid, len(parties), len(parties)-1)

	endChan := make(chan keygen.LocalPartySaveData)
	outChan := make(chan tss.Message)
	errChan := make(chan error)

	party := keygen.NewLocalParty(params, outChan, endChan, *n.preParams)
	n.kgParty = &party
	n.logger.Info("KG Local party setup done")

	go func() {
		for {
			select {
			case outMsg := <-outChan:
				n.handleKeygenMessage(outMsg, errChan)
			case endData := <-endChan:
				n.handleKeygenEnd(endData)
				// TODO: Break the loop?
			case err := <-errChan:
				n.logger.Sugar().Fatal(err)
			}
		}
	}()
}

func (n *Node) handleKeygenEnd(data keygen.LocalPartySaveData) {
	n.kgData = &data
	n.logger.Info("Keygen complete")
}

// Messages coming in from the TSS-Lib channels
func (n *Node) handleKeygenMessage(message tss.Message, errChan chan<- error) {
	n.peerLock.RLock()
	// No need to wait for go funcs to complete, as we are only reading the peers
	defer n.peerLock.RUnlock()
	n.logger.Sugar().Infof("[KEYGEN] Received a message from outChan: %+v", message)
	dest := message.GetTo()

	if dest == nil {
		// Broadcast
		for _, peer := range n.peers {
			if peer.version.ListenAddr == n.listenAddress {
				continue
			}
			go n.messagePeer(TSS_KEYGEN, ToWireMessage(message), &peer.nodeClient, errChan)
		}
	} else {
		go n.messagePeer(TSS_KEYGEN, ToWireMessage(message), &n.peers[dest[0].Moniker].nodeClient, errChan)
	}
}
