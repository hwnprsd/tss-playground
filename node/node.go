package node

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/hwnprsd/tss/crypto"
	"github.com/hwnprsd/tss/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type PeerData struct {
	version    *proto.Version
	nodeClient proto.NodeClient
}

type Node struct {
	proto.UnimplementedNodeServer

	logger *zap.Logger

	peerLock sync.RWMutex
	peers    map[string]*PeerData

	version       string
	listenAddress string

	// FIXME:
	// Security Hazard
	privateKey crypto.PrivateKey
	partyId    *proto.PartyId

	// TSS PreParams takes time to generate
	isParamsReady bool
}

func NewNode() *Node {
	devConfig := zap.NewDevelopmentConfig()
	devConfig.EncoderConfig.TimeKey = ""
	devConfig.EncoderConfig.CallerKey = ""
	logger, _ := devConfig.Build()
	return &Node{
		version: "solace-kn-1.0.0",
		peers:   make(map[string]*PeerData),
		logger:  logger,
	}
}

func (n *Node) Start(listenAddr string, knownAddresses []string) {
	var (
		opts       = []grpc.ServerOption{}
		grpcServer = grpc.NewServer(opts...)
	)

	ln, err := net.Listen("tcp", listenAddr)
	n.logger = n.logger.Named(listenAddr)
	if err != nil {
		n.logger.Sugar().Fatal(err)
	}

	n.listenAddress = listenAddr

	proto.RegisterNodeServer(grpcServer, n)
	go func() {
		// only connect to nodes after setting up TSS requirements
		n.SetupForTss()
		n.ConnectToNodes(knownAddresses)
	}()
	n.logger.Info(fmt.Sprintf("New node started (%s)", listenAddr))
	n.logger.Sugar().Fatal(grpcServer.Serve(ln))
}

// TODO: Have node limits?
func (n *Node) ConnectToNodes(addrs []string) {
	for _, addr := range addrs {
		// If the address is ourselves / address is already connected to
		// continue
		if addr == n.listenAddress || n.peers[addr] != nil {
			continue
		}

		c, v, err := n.dialRemoteNode(addr)
		if err != nil {
			// TODO: Log and handle Dialing errors
			n.logger.Sugar().Error(err)
			continue
		}
		n.addPeer(c, v)
	}
}

func (n *Node) addPeer(c *proto.NodeClient, version *proto.Version) {
	n.peerLock.Lock()
	defer n.peerLock.Unlock()
	if len(version.PeerList) > 0 {
		go n.ConnectToNodes(version.PeerList)
	}

	n.peers[version.ListenAddr] = &PeerData{
		version:    version,
		nodeClient: *c,
	}
	n.logger.Info(fmt.Sprintf("CONNECTED (%s)", version.ListenAddr))
	// n.logger.Info(fmt.Sprintf("Total Nodes Connected = %d", len(n.peers)))
}

func (n *Node) removePeer(addr string) {
	n.peerLock.Lock()
	defer n.peerLock.Unlock()
	delete(n.peers, addr)
}

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

func (n *Node) dialRemoteNode(addr string) (*proto.NodeClient, *proto.Version, error) {
	c, err := makeNodeClient(addr)
	if err != nil {
		n.logger.Error(fmt.Sprintf("Dial Error for Addr (%s)", addr))
		n.logger.Sugar().Fatal(err)
		return nil, nil, err
	}
	v, err := c.Handshake(context.Background(), n.Version())
	if err != nil {
		n.logger.Sugar().Errorf("Error handshaking for Addr (%s)", addr)
		n.logger.Sugar().Fatal(err)
		return nil, nil, err
	}
	return &c, v, nil
}

func (n *Node) peerList() []string {
	n.peerLock.RLock()
	defer n.peerLock.RUnlock()

	peers := []string{}
	for _, peerData := range n.peers {
		peers = append(peers, peerData.version.ListenAddr)
	}
	return peers
}

func (n *Node) Version() *proto.Version {
	return &proto.Version{
		Version:       n.version,
		ListenAddr:    n.listenAddress,
		PeerList:      n.peerList(),
		IsInitialized: n.isParamsReady,
		PartyId:       n.partyId,
	}
}

func makeNodeClient(listenAddr string) (proto.NodeClient, error) {
	c, err := grpc.Dial(listenAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return proto.NewNodeClient(c), nil
}
