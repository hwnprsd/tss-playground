package node

import (
	"encoding/hex"
	"time"

	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/tss"
	"github.com/hwnprsd/tss/common"
	"github.com/hwnprsd/tss/crypto"
	"github.com/hwnprsd/tss/proto"
)

const (
	TSS_KEYGEN    = 1
	TSS_SIGNATURE = 2
)

func (n *Node) SetupForTss() {
	// TODO: Maybe read the private key from a config file
	// Instead of creating a new one every time
	privKey := crypto.GenerateRandomKey()
	uniqueKey := privKey.Public().BigInt()

	// TOOD: Find a more robust ID structure
	n.partyId = &proto.PartyId{
		Moniker: n.listenAddress,
		Id:      hex.EncodeToString(uniqueKey.Bytes()),
		Key:     uniqueKey.Bytes(),
	}
	n.pid = common.ToPartyId(n.partyId)
	// TODO: Maybe remove this?
	n.isParamsReady = true
	n.privateKey = privKey
}

func (n *Node) GeneratePreParams() {
	preParams, _ := keygen.GeneratePreParams(1 * time.Minute)
	n.preParams = preParams
}

func (n *Node) GetPartiesSorted() (parties []*tss.PartyID) {
	n.peerLock.RLock()
	defer n.peerLock.RUnlock()
	for _, p := range n.peers {
		parties = append(parties, common.ToPartyId(p.GetVersion().PartyId))
	}
	parties = append(parties, n.pid)
	parties = tss.SortPartyIDs(parties)
	return
}

// Get the well constructed partyId for any given id in the peerlist
// This is important, so the index of the party is populated
func (n *Node) GetPartyId(id string) *tss.PartyID {
	parties := n.GetPartiesSorted()
	for _, party := range parties {
		if party.Id == id {
			return party
		}
	}
	return nil
}
