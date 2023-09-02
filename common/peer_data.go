package common

import (
	"math/big"

	"github.com/bnb-chain/tss-lib/tss"
	"github.com/hwnprsd/tss/proto"
)

type Peer struct {
	version    *proto.Version
	nodeClient *proto.NodeClient
}

func (p *Peer) GetVersion() *proto.Version {
	return p.version
}

func (p *Peer) SetVersion(version *proto.Version) {
	p.version = version
}

func (p *Peer) GetNodeClient() *proto.NodeClient {
	return p.nodeClient
}

func NewPeer(version *proto.Version, client *proto.NodeClient) *Peer {
	return &Peer{version, client}
}

// Each time I call this, i need to sort the peers, or the index will be -1
func ToPartyId(party *proto.PartyId) *tss.PartyID {
	return tss.NewPartyID(
		party.GetId(),
		party.GetMoniker(),
		new(big.Int).SetBytes(party.GetKey()),
	)
}
