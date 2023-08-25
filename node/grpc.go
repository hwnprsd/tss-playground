package node

import (
	"context"

	"github.com/hwnprsd/tss/proto"
)

// GRPC Handler
func (n *Node) DKGMessage(ctx context.Context, message *proto.DKGData) (*proto.Ack, error) {
	n.messageLock.Lock()
	defer n.messageLock.Unlock()
	// TODO: What to do if the localparty is outdated?
	// Check if the parties matches the incoming message
	if n.kgParty == nil {
		n.InitKeygen()
	}

	fromPartyId := n.GetPartyId(message.PartyId.Id)

	// Send broadcast info over the network as well
	_, err := (*n.kgParty).UpdateFromBytes(message.WireMessage, fromPartyId, message.IsBroadcast)
	if err != nil {
		return nil, err
	}
	return &proto.Ack{}, nil
}
