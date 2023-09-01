package node

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"errors"
	"math/big"

	"github.com/bnb-chain/tss-lib/common"
	"github.com/bnb-chain/tss-lib/ecdsa/signing"
	"github.com/bnb-chain/tss-lib/tss"
)

func (n *Node) InitSigning(message []byte) {
	if err := n.SetupSigLocalParty(message); err != nil {
		n.logger.Sugar().Fatal(err)
	}

	go func() {
		err := (*n.sigParty).Start()
		if err != nil {
			n.logger.Sugar().Fatal(err)
		}
	}()
}

func (n *Node) SetupSigLocalParty(message []byte) error {
	if n.kgData == nil {
		return errors.New("Complete keygen first")
	}

	n.logger.Info("Setting up Sig local party...")
	msg := new(big.Int).SetBytes(message)

	parties := n.GetPartiesSorted()
	peerCtx := tss.NewPeerContext(parties)
	params := tss.NewParameters(tss.S256(), peerCtx, n.pid, len(parties), len(parties)-1)

	endChan := make(chan common.SignatureData)
	outChan := make(chan tss.Message)
	errChan := make(chan error)

	party := signing.NewLocalParty(msg, params, *n.kgData, outChan, endChan)
	n.sigParty = &party
	n.logger.Info("Sig Local party setup done")

	go func() {
		for {
			select {
			case outMsg := <-outChan:
				n.handleSigningMessage(outMsg, errChan, message)
			case endData := <-endChan:
				n.handleSigningEnd(&endData, message)
				// TODO: Break the loop?
			case err := <-errChan:
				n.logger.Sugar().Fatal(err)
			}
		}
	}()

	return nil
}

func (n *Node) handleSigningEnd(data *common.SignatureData, message []byte) {
	n.logger.Info("Sig complete")
	x, y := (*n.kgData).ECDSAPub.X(), (*n.kgData).ECDSAPub.Y()
	pk := ecdsa.PublicKey{
		Curve: tss.EC(),
		X:     x,
		Y:     y,
	}
	ok := ecdsa.VerifyASN1(&pk, message, data.GetSignature())
	pubKeyBytes := elliptic.Marshal(pk.Curve, pk.X, pk.Y)
	n.logger.Sugar().Infof("Public Key - %s", hex.EncodeToString(pubKeyBytes))
	n.logger.Sugar().Infof("Is Verified? - %s", ok)
}

func (n *Node) handleSigningMessage(message tss.Message, errChan chan<- error, msgToSign []byte) {
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
			go n.updateTSSPeer(TSS_SIGNATURE, message, &peer.nodeClient, errChan, withSigMessage(msgToSign))
		}
	} else {
		go n.updateTSSPeer(TSS_SIGNATURE, message, &n.peers[dest[0].Moniker].nodeClient, errChan, withSigMessage(msgToSign))
	}

}
