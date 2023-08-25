package node

import (
	"errors"
	"math/big"

	"github.com/bnb-chain/tss-lib/common"
	"github.com/bnb-chain/tss-lib/ecdsa/signing"
	"github.com/bnb-chain/tss-lib/tss"
)

func (n *Node) InitSigning(message string) {
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

func (n *Node) SetupSigLocalParty(message string) error {
	if n.kgData == nil {
		return errors.New("Complete keygen first")
	}

	n.logger.Info("Setting up Sig local party...")
	bytes := []byte(message)
	msg := new(big.Int).SetBytes(bytes)

	parties := n.GetPartiesSorted()
	peerCtx := tss.NewPeerContext(parties)
	params := tss.NewParameters(tss.S256(), peerCtx, n.pid, len(parties), len(parties)-1)

	endChan := make(chan common.SignatureData)
	outChan := make(chan tss.Message)
	errChan := make(chan error)

	signing.NewLocalParty(msg, params, *n.kgData, outChan, endChan)

	go func() {
		for {
			select {
			case outMsg := <-outChan:
				n.handleSigningMessage(outMsg, errChan)
			case endData := <-endChan:
				n.handleSigningEnd(endData)
				// TODO: Break the loop?
			case err := <-errChan:
				n.logger.Sugar().Fatal(err)
			}
		}
	}()

	return nil
}

func (n *Node) handleSigningEnd(data common.SignatureData) {
}

func (n *Node) handleSigningMessage(message tss.Message, errChan chan<- error) {}
