package node

import (
	"encoding/hex"
	"errors"
	"math/big"

	"github.com/bnb-chain/tss-lib/common"
	"github.com/bnb-chain/tss-lib/ecdsa/signing"
	"github.com/bnb-chain/tss-lib/tss"
)

func (n *Node) InitSigning(address []byte, message []byte) {
	shouldContinue, err := n.SetupSigLocalParty(message, address)

	if err != nil {
		n.logger.Sugar().Fatal(err)
	}

	if !shouldContinue {
		return
	}

	sAddress := AddressFromBytes(address)
	go func() {
		err := (*n.sessions[sAddress].sigParty).Start()
		if err != nil {
			n.logger.Sugar().Fatal(err)
		}
	}()
}

func (n *Node) SetupSigLocalParty(message []byte, sessionAddress []byte) (bool, error) {
	sAddress := AddressFromBytes(sessionAddress)
	session, exists := n.sessions[sAddress]
	if !exists {
		return false, errors.New("Session not initiated")
	}
	// If the KeyGen Local data is missing
	if session.kgData == nil {
		return false, errors.New("Complete keygen first")
	}
	// If the signatureParty already exists, don't overwrite
	if session.sigParty != nil {
		return false, nil
	}

	n.logger.Info("Setting up Sig local party...")
	msg := new(big.Int).SetBytes(message)

	parties := n.GetPartiesSorted()
	peerCtx := tss.NewPeerContext(parties)
	params := tss.NewParameters(tss.S256(), peerCtx, n.pid, len(parties), len(parties)-1)

	endChan := make(chan common.SignatureData)
	outChan := make(chan tss.Message)
	errChan := make(chan error)

	party := signing.NewLocalParty(msg, params, *session.kgData, outChan, endChan)
	session.sigParty = &party
	n.logger.Info("Sig Local party setup done")

	go func() {
		for {
			select {
			case outMsg := <-outChan:
				n.handleSigningMessage(outMsg, errChan, message, sessionAddress)
			case endData := <-endChan:
				n.handleSigningEnd(&endData, message, sessionAddress)
				// TODO: Break the loop?
			case err := <-errChan:
				n.logger.Sugar().Fatal(err)
			}
		}
	}()

	return true, nil
}

func (n *Node) handleSigningEnd(data *common.SignatureData, message []byte, sessionAddress []byte) {
	sAddress := AddressFromBytes(sessionAddress)
	n.logger.Info("Sig complete")
	n.logger.Info(hex.EncodeToString(data.Signature))
	session := n.sessions[sAddress]
	session.sigParty = nil
	// x, y := (*n.kgData).ECDSAPub.X(), (*n.kgData).ECDSAPub.Y()
	// pk := ecdsa.PublicKey{
	// 	Curve: tss.EC(),
	// 	X:     x,
	// 	Y:     y,
	// }
	// ok := ecdsa.VerifyASN1(&pk, message, data.GetSignature())
	// pubKeyBytes := elliptic.Marshal(pk.Curve, pk.X, pk.Y)
	// n.logger.Sugar().Infof("Public Key - %s", hex.EncodeToString(pubKeyBytes))
	// n.logger.Sugar().Infof("Is Verified? - %s", ok)
}

// TODO: Add session address
func (n *Node) handleSigningMessage(message tss.Message, errChan chan<- error, msgToSign []byte, sessionAddress []byte) {
	n.peerLock.RLock()
	// No need to wait for go funcs to complete, as we are only reading the peers
	defer n.peerLock.RUnlock()
	n.logger.Sugar().Infof("[SIGNING] Received a message from outChan: %+v", message)
	dest := message.GetTo()

	if dest == nil {
		// Broadcast
		for _, peer := range n.peers {
			if peer.version.ListenAddr == n.listenAddress {
				continue
			}
			go n.messagePeer(TSS_SIGNATURE, ToWireMessage(message), &peer.nodeClient, sessionAddress, errChan, withSigMessage(msgToSign))
		}
	} else {
		go n.messagePeer(TSS_SIGNATURE, ToWireMessage(message), &n.peers[dest[0].Moniker].nodeClient, sessionAddress, errChan, withSigMessage(msgToSign))
	}

}
