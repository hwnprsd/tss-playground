package node

import (
	"fmt"
	"math/big"

	"github.com/bnb-chain/tss-lib/tss"
	"github.com/hwnprsd/tss/proto"
)

func (n *Node) LogPeers() {
	n.peerLock.Lock()
	defer n.peerLock.Unlock()
	n.logger.Debug(fmt.Sprintf("Peer Count = %d", len(n.peers)))
}

func (n *Node) LogVersion() {
	n.logger.Sugar().Debug(n.Version())
}

func ToPartyId(party *proto.PartyId) *tss.PartyID {
	return tss.NewPartyID(
		party.Id,
		party.Moniker,
		new(big.Int).SetBytes(party.Key),
	)
}
