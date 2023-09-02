package node

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/tss"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/hwnprsd/tss/common"
	"github.com/hwnprsd/tss/crypto"
	"github.com/hwnprsd/tss/proto"
	"github.com/hwnprsd/tss/session"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Node struct {
	proto.UnimplementedNodeServer

	logger *zap.Logger

	peerLock    sync.RWMutex
	messageLock sync.RWMutex
	peers       map[string]*common.Peer

	version       string
	listenAddress string

	// Map of smart-wallet data to Session Data
	sessions map[string]*session.Session

	// FIXME:
	// Security Hazard
	privateKey crypto.PrivateKey
	partyId    *proto.PartyId

	// kgParty  *tss.Party
	// kgData   *keygen.LocalPartySaveData
	// sigParty *tss.Party
	// TODO: Redundant, but pid stores sorted index
	pid *tss.PartyID
	// This needs to be generated for each DKG event
	preParams *keygen.LocalPreParams

	// TSS PreParams takes time to generate
	// Not sure if this is needed in the nodes's state
	isParamsReady bool
}

func NewNode() *Node {
	devConfig := zap.NewDevelopmentConfig()
	devConfig.EncoderConfig.TimeKey = ""
	devConfig.EncoderConfig.CallerKey = ""
	logger, _ := devConfig.Build()
	return &Node{
		version:  "solace-kn-1.0.0",
		peers:    make(map[string]*common.Peer),
		logger:   logger,
		sessions: make(map[string]*session.Session),
	}
}

func (n *Node) StartGrpcServer() {
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := proto.RegisterNodeHandlerFromEndpoint(context.Background(), mux, n.listenAddress, opts)
	if err != nil {
		panic(err)
	}
	n.logger.Info(fmt.Sprintf("New mux server started (%s)", ":5050"))
	err = http.ListenAndServe(":5050", mux)
	if err != nil {
		panic(err)
	}
}

func (n *Node) Start(listenAddr string, knownAddresses []string, startServer bool) {
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
	// This should be done in real time
	n.SetupForTss()
	n.GeneratePreParams()
	go func() {
		n.ConnectToNodes(knownAddresses)
	}()
	n.logger.Info(fmt.Sprintf("New node started (%s)", listenAddr))
	if startServer {
		go n.StartGrpcServer()
	}
	n.logger.Sugar().Fatal(grpcServer.Serve(ln))
}

// TODO: Have node limits?
func (n *Node) ConnectToNodes(addrs []string) {
	for _, addr := range addrs {
		// If the address is ourselves / address is already connected to
		// continue
		_, exists := n.peers[addr]
		if addr == n.listenAddress || exists {
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
	_, exists := n.peers[version.ListenAddr]
	if exists {
		return
	}

	n.peers[version.ListenAddr] = common.NewPeer(version, c)
	n.logger.Info(fmt.Sprintf("CONNECTED (%s)", version.ListenAddr))
	// n.logger.Info(fmt.Sprintf("Total Nodes Connected = %d", len(n.peers)))
}

func (n *Node) removePeer(addr string) {
	n.peerLock.Lock()
	defer n.peerLock.Unlock()
	delete(n.peers, addr)
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
		peers = append(peers, peerData.GetVersion().ListenAddr)
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
