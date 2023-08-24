package node

import (
	"time"

	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/tss"
	"github.com/hwnprsd/tss/crypto"
	"github.com/hwnprsd/tss/proto"
)

func (n *Node) SetupForTss() {
	preParams, _ := keygen.GeneratePreParams(1 * time.Minute)
	_ = preParams

	// TODO: Maybe read the private key from a config file
	// Instead of creating a new one every time
	privKey := crypto.GenerateRandomKey()
	uniqueKey := privKey.Public().BigInt()

	// TOOD: Find a more robust ID structure
	n.partyId = &proto.PartyId{
		Moniker: n.listenAddress,
		Id:      n.listenAddress,
		Key:     uniqueKey.Bytes(),
	}
	// TODO: Maybe remove this?
	n.isParamsReady = true
	n.privateKey = privKey
}

func (n *Node) PartyId() *tss.PartyID {
	return ToPartyId(n.partyId)
}

func (n *Node) GetParties() (parties []*tss.PartyID) {
	n.peerLock.RLock()
	defer n.peerLock.RUnlock()
	for _, p := range n.peers {
		parties = append(parties, ToPartyId(p.version.PartyId))
	}
	return
}
