package node

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hwnprsd/tss/proto"
)

// Handlers for all the GRPC Methods Supported

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

	// TODO: Handle setting target version
	n.peers[version.ListenAddr].SetVersion(version)
	n.logger.Info(fmt.Sprintf("Updating peer data (%s)", version.PartyId))
	return &proto.Ack{}, nil
}

func (n *Node) StartDKG(ctx context.Context, caller *proto.Caller) (*proto.Ack, error) {
	n.InitKeygen(caller.Address)
	return &proto.Ack{}, nil
}

func (n *Node) StartSigning(ctx context.Context, data *proto.SignCaller) (*proto.Ack, error) {
	n.InitSigning(data.Address, data.Data)
	return &proto.Ack{}, nil
}

// GRPC Handler
func (n *Node) HandleTSSMessage(ctx context.Context, message *proto.TSSData) (*proto.Ack, error) {
	time.Sleep(500 * time.Millisecond)
	n.messageLock.Lock()
	defer n.messageLock.Unlock()
	// TODO: What to do if the localparty is outdated?
	// Check if the parties matches the incoming message

	// TODO: Setup the session if it doesn't exist

	// fromPartyId := n.GetPartyId(message.PartyId.Id)
	sAddress := AddressFromBytes(message.Address)

	switch message.Type {
	case TSS_KEYGEN:
		n.InitKeygen(message.Address)
		session := n.sessions[sAddress]
		outChan, errChan, err := session.UpdateKeygenParty(message)
		// Send broadcast info over the network as well
		if err != nil {
			return nil, err
		}
		// First time init
		if outChan != nil {
			go n.listenKeygenMessages(message.Address, outChan, errChan)
		}
		return &proto.Ack{}, nil
	case TSS_SIGNATURE:
		_, err := n.InitSigning(message.Address, message.SigMessage)
		if err != nil {
			n.logger.Sugar().Fatal(err)
		}

		session := n.sessions[sAddress]

		// Send broadcast info over the network as well
		outChan, errChan, err := (*session).UpdateSigningParty(message)
		if err != nil {
			return nil, err
		}
		// First time init
		if outChan != nil {
			go n.listenSigningMessages(message.SigMessage, message.Address, outChan, errChan)
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

type WireMessage interface {
	Bytes() ([]byte, error)
	IsBroadcast() bool
}

func (n *Node) messagePeer(messageType int, message WireMessage, node *proto.NodeClient, sessionAddress []byte, errChan chan<- error, opts ...TSSMessageOptFunc) {
	data, err := message.Bytes()
	if err != nil {
		errChan <- err
		n.logger.Sugar().Fatal(err)
	}
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
		Address:     sessionAddress,
	}
	_, err = (*node).HandleTSSMessage(context.Background(), msg)
	if err != nil {
		errChan <- err
	}
}
