package node

import (
	"encoding/hex"
	"fmt"

	"github.com/bnb-chain/tss-lib/tss"
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
