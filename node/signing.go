package node

import (
	"errors"

	"github.com/bnb-chain/tss-lib/tss"
)

func (n *Node) InitSigning(sessionAddress []byte, message []byte) (bool, error) {
	sAddress := AddressFromBytes(sessionAddress)
	s, exists := n.sessions[sAddress]
	if !exists {
		return false, errors.New("Session not initiated")
	}

	// If the KeyGen Local data is missing

	outChan, errChan := s.InitSigning(message)

	if outChan != nil {
		go n.listenSigningMessages(message, sessionAddress, outChan, errChan)
	}

	return true, nil
}

func (n *Node) listenSigningMessages(message []byte, sessionAddress []byte, outChan *chan tss.Message, errChan *chan error) {
	for {
		select {
		case outMsg := <-*outChan:
			n.handleSigningMessage(outMsg, *errChan, message, sessionAddress)
		case err := <-*errChan:
			n.logger.Sugar().Fatal(err)
		}
	}
}

// TODO: Add session address
func (n *Node) handleSigningMessage(message tss.Message, errChan chan<- error, msgToSign []byte, sessionAddress []byte) {
	n.peerLock.RLock()
	// No need to wait for go funcs to complete, as we are only reading the peers
	defer n.peerLock.RUnlock()
	// n.logger.Sugar().Infof("[SIGNING] Received a message from outChan: %+v", message)
	dest := message.GetTo()

	if dest == nil {
		// Broadcast
		for _, peer := range n.peers {
			if peer.GetVersion().ListenAddr == n.listenAddress {
				continue
			}
			go n.messagePeer(TSS_SIGNATURE, ToWireMessage(message), peer.GetNodeClient(), sessionAddress, errChan, withSigMessage(msgToSign))
		}
	} else {
		go n.messagePeer(TSS_SIGNATURE, ToWireMessage(message), n.peers[dest[0].Moniker].GetNodeClient(), sessionAddress, errChan, withSigMessage(msgToSign))
	}

}
