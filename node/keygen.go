package node

import (
	"github.com/bnb-chain/tss-lib/tss"
	"github.com/hwnprsd/tss/session"
)

// Setup the local party & it's listeners
// Will only create a new Party if is doesn't exist
// Use delete(session, address) if you want to regenrate
// KeyGen should be init'd once only
func (n *Node) InitKeygen(sessionAddress []byte) {
	sAddress := AddressFromBytes(sessionAddress)
	_, exists := n.sessions[sAddress]

	if exists {
		return
	}
	// Create a new session if not exists
	n.logger.Info("Setting up KG local party...")
	n.sessions[sAddress] = session.NewSession(
		n.pid,
		n.peers,
		n.preParams,
	)
	s := n.sessions[sAddress]

	outChan, errChan := s.InitKeygen()
	// If outChan is nil, that means, a new Keygen party was not
	// Initialized, as it already existed
	if outChan == nil {
		n.logger.Fatal("Error occured initing KeyGen")
	}
	go n.listenKeygenMessages(sessionAddress, outChan, errChan)
}

func (n *Node) listenKeygenMessages(sessionAddress []byte, outChan *chan tss.Message, errChan *chan error) {
	for {
		select {
		case outMsg := <-*outChan:
			n.handleKeygenMessage(outMsg, *errChan, sessionAddress)
		case err := <-*errChan:
			n.logger.Sugar().Fatal(err)
		}
	}
}

// Messages coming in from the TSS-Lib channels
func (n *Node) handleKeygenMessage(message tss.Message, errChan chan error, sessionAddress []byte) {
	// n.logger.Sugar().Infof("[KEYGEN] Received a message from outChan: %+v", message)
	dest := message.GetTo()

	session := n.sessions[AddressFromBytes(sessionAddress)]

	if dest == nil {
		// Broadcast
		for _, peer := range session.GetParties() {
			if peer.GetVersion().ListenAddr == n.listenAddress {
				continue
			}
			go n.messagePeer(TSS_KEYGEN, ToWireMessage(message), peer.GetNodeClient(), sessionAddress, errChan)
		}
	} else {
		go n.messagePeer(TSS_KEYGEN, ToWireMessage(message), n.peers[dest[0].Moniker].GetNodeClient(), sessionAddress, errChan)
	}
}
