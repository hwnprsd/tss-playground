package node

import (
	"encoding/hex"
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

func (n *Node) PeerCount() int {
	n.peerLock.Lock()
	defer n.peerLock.Unlock()
	return len(n.peers)
}

func (n *Node) LogVersion() {
	n.logger.Sugar().Debug(n.Version())
}

// Each time I call this, i need to sort the peers, or the index will be -1
func ToPartyId(party *proto.PartyId) *tss.PartyID {
	return tss.NewPartyID(
		party.Id,
		party.Moniker,
		new(big.Int).SetBytes(party.Key),
	)
}

type wireMessageData struct {
	tss.Message
}

func (w wireMessageData) Bytes() ([]byte, error) {
	b, _, e := w.WireBytes()
	return b, e
}

func ToWireMessage(message tss.Message) WireMessage {
	return wireMessageData{message}
}

func AddressFromBytes(b []byte) string {
	return hex.EncodeToString(b)
}
