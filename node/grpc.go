package node

import (
	"context"
	"errors"
	"fmt"

	"github.com/bnb-chain/tss-lib/tss"
	"github.com/hwnprsd/tss/proto"
)

func (n *Node) Handshake(ctx context.Context, version *proto.Version) (*proto.Version, error) {
	c, err := makeNodeClient(version.ListenAddr)
	if err != nil {
		return nil, err
	}

	// FIXME:
	// There's a potential issue where addPeer gets call on the same peer
	// And because of mutex locks, it gets set twice
	// Not a breaking issue, but still an issue
	n.addPeer(&c, version)

	return n.Version(), nil
}

// FIXME: Big security vuln
func (n *Node) Update(ctx context.Context, version *proto.Version) (*proto.Ack, error) {
	n.peerLock.Lock()
	defer n.peerLock.Unlock()
	// TODO: Check if the client is valid & exists
	// FIXME: Blind update is a security risk - Have some AUTH
	n.peers[version.ListenAddr].version = version
	n.logger.Info(fmt.Sprintf("Updating peer data (%s)", version.PartyId))
	return &proto.Ack{}, nil
}

func (n *Node) StartDKG(context.Context, *proto.Caller) (*proto.Ack, error) {
	n.InitKeygen()
	return &proto.Ack{}, nil
}

func (n *Node) StartSigning(ctx context.Context, data *proto.SignCaller) (*proto.Ack, error) {
	n.InitSigning(data.Data)
	return &proto.Ack{}, nil
}

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
