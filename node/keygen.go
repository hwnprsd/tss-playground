package node

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"

	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/tss"
)

// Handle multiple signs
func (n *Node) InitKeygen(sessionAddress []byte) {
	// TODO: Handle different cohorts of local parties?
	// FIXME: Should not overwrite if an ongoing round is happening
	exist := n.SetupKeygenParty(sessionAddress)
	if exist {
		return
	}

	sAddress := AddressFromBytes(sessionAddress)

	go func() {
		err := (*n.sessions[sAddress].keyGenParty).Start()
		if err != nil {
			n.logger.Sugar().Fatal(err)
		}
	}()
}

// Setup the local party & it's listeners
// Will only create a new Party if is doesn't exist
// Use delete(session, address) if you want to regenrate
func (n *Node) SetupKeygenParty(sessionAddress []byte) bool {
	n.logger.Info("Setting up KG local party...")

	sAddress := AddressFromBytes(sessionAddress)
	session, exists := n.sessions[sAddress]

	if exists && session.keyGenParty != nil {
		// n.logger.Info("KG Party for this session already exists")
		return true
	}

	// TODO: Get Parties based on the session. Not everyone connected
	parties := n.GetPartiesSorted()
	peerCtx := tss.NewPeerContext(parties)
	params := tss.NewParameters(tss.S256(), peerCtx, n.pid, len(parties), len(parties)-1)
	// x, y := (*n.kgData).ECDSAPub.X(), (*n.kgData).ECDSAPub.Y()
	// pk := ecdsa.PublicKey{
	// 	Curve: tss.EC(),
	// 	X:     x,
	// 	Y:     y,
	// }
	// ok := ecdsa.VerifyASN1(&pk, message, data.GetSignature())
	// pubKeyBytes := elliptic.Marshal(pk.Curve, pk.X, pk.Y)
	// n.logger.Sugar().Infof("Public Key - %s", hex.EncodeToString(pubKeyBytes))

	endChan := make(chan keygen.LocalPartySaveData)
	outChan := make(chan tss.Message)
	errChan := make(chan error)

	party := keygen.NewLocalParty(params, outChan, endChan, *n.preParams)
	// Only create the local party if the session doesn't exist
	n.sessions[sAddress] = &Session{
		keyGenParty: &party,
	}
	n.logger.Info("KG Local party setup done")

	go func() {
		for {
			select {
			case outMsg := <-outChan:
				n.handleKeygenMessage(outMsg, errChan, sessionAddress)
			case endData := <-endChan:
				n.handleKeygenEnd(endData, sessionAddress)
				// TODO: Break the loop?
			case err := <-errChan:
				n.logger.Sugar().Fatal(err)
			}
		}
	}()
	return false
}

func (n *Node) handleKeygenEnd(data keygen.LocalPartySaveData, sessionAddress []byte) {
	sAddress := AddressFromBytes(sessionAddress)
	session, exists := n.sessions[sAddress]
	if !exists {
		n.logger.Fatal("Session not initialized")
	}
	session.kgData = &data
	x, y := data.ECDSAPub.X(), data.ECDSAPub.Y()
	pk := ecdsa.PublicKey{
		Curve: tss.EC(),
		X:     x,
		Y:     y,
	}
	pubKeyBytes := elliptic.Marshal(pk.Curve, pk.X, pk.Y)
	n.logger.Sugar().Infof("Session - %s", sAddress)
	n.logger.Sugar().Infof("Public Key - %s", hex.EncodeToString(pubKeyBytes))
}

// Messages coming in from the TSS-Lib channels
func (n *Node) handleKeygenMessage(message tss.Message, errChan chan<- error, sessionAddress []byte) {
	n.peerLock.RLock()
	// No need to wait for go funcs to complete, as we are only reading the peers
	defer n.peerLock.RUnlock()
	n.logger.Sugar().Infof("[KEYGEN] Received a message from outChan: %+v", message)
	dest := message.GetTo()

	if dest == nil {
		// Broadcast
		for _, peer := range n.peers {
			if peer.version.ListenAddr == n.listenAddress {
				continue
			}
			go n.messagePeer(TSS_KEYGEN, ToWireMessage(message), &peer.nodeClient, sessionAddress, errChan)
		}
	} else {
		go n.messagePeer(TSS_KEYGEN, ToWireMessage(message), &n.peers[dest[0].Moniker].nodeClient, sessionAddress, errChan)
	}
}
