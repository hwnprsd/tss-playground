package node

import (
	"context"
	"errors"

	"github.com/bnb-chain/tss-lib/tss"
	"github.com/hwnprsd/tss/proto"
)

// GRPC Handler
func (n *Node) HandleTSSMessage(ctx context.Context, message *proto.TSSData) (*proto.Ack, error) {
	n.messageLock.Lock()
	defer n.messageLock.Unlock()
	// TODO: What to do if the localparty is outdated?
	// Check if the parties matches the incoming message

	fromPartyId := n.GetPartyId(message.PartyId.Id)

	switch message.Type {
	case TSS_KEYGEN:
		if n.kgParty == nil {
			n.InitKeygen()
		}

		// Send broadcast info over the network as well
		_, err := (*n.kgParty).UpdateFromBytes(message.WireMessage, fromPartyId, message.IsBroadcast)
		if err != nil {
			return nil, err
		}
		return &proto.Ack{}, nil
	case TSS_SIGNATURE:
		if n.sigParty == nil {
			n.InitSigning(message.SigMessage)
		}

		// Send broadcast info over the network as well
		_, err := (*n.sigParty).UpdateFromBytes(message.WireMessage, fromPartyId, message.IsBroadcast)
		if err != nil {
			return nil, err
		}
		return &proto.Ack{}, nil
	default:
		return nil, errors.New("Invalid TSS Message Type")
	}

}

type TSSMessageOpt struct {
	SigMessage []byte
}

type TSSMessageOptFunc func(*TSSMessageOpt)

func withSigMessage(message []byte) func(*TSSMessageOpt) {
	return func(opt *TSSMessageOpt) {
		opt.SigMessage = message
	}
}

func (n *Node) updateTSSPeer(messageType int, message tss.Message, node *proto.NodeClient, errChan chan<- error, opts ...TSSMessageOptFunc) {
	data, _, _ := message.WireBytes()
	opt := TSSMessageOpt{}
	for _, o := range opts {
		o(&opt)
	}
	msg := &proto.TSSData{
		WireMessage: data,
		PartyId:     n.partyId,
		IsBroadcast: message.IsBroadcast(),
		Type:        int32(messageType),
		SigMessage:  opt.SigMessage,
	}
	_, err := (*node).HandleTSSMessage(context.Background(), msg)
	if err != nil {
		errChan <- err
	}
}
