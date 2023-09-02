package session

import (
	"log"

	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/tss"
)

func (s *Session) InitKeygen() (*chan tss.Message, *chan error) {
	shouldContinueInit, outChan, errChan := s.setupKeygenParty()
	if !shouldContinueInit {
		return nil, nil
	}
	go func() {
		err := (*s.keyGenParty).Start()
		if err != nil {
			log.Println("ERR", err)
		}
	}()
	return outChan, errChan
}

// Should continue init
func (s *Session) setupKeygenParty() (shouldContinueInit bool, outChan *chan tss.Message, errChan *chan error) {
	// Keygen Party exists for this session
	if s.keyGenParty != nil {
		return false, nil, nil
	}

	parties := s.GetPartiesSorted()

	peerCtx := tss.NewPeerContext(parties)

	params := tss.NewParameters(tss.S256(), peerCtx, s.partyId, len(parties), len(parties)-1)

	outChan, errChan = s.setupChannels()
	endChan := make(chan keygen.LocalPartySaveData)
	party := keygen.NewLocalParty(params, *outChan, endChan, *s.preParams)
	s.keyGenParty = &party

	go func() {
		for {
			select {
			case endData := <-endChan:
				s.handleKeygenEnd(endData)
			}
		}
	}()

	return true, outChan, errChan
}

func (s *Session) handleKeygenEnd(data keygen.LocalPartySaveData) {
	s.keyGenData = &data
	log.Println("Keygen Complete")
	// x, y := data.ECDSAPub.X(), data.ECDSAPub.Y()
	// pk := ecdsa.PublicKey{
	// 	Curve: tss.EC(),
	// 	X:     x,
	// 	Y:     y,
	// }
	// pubKeyBytes := elliptic.Marshal(pk.Curve, pk.X, pk.Y)
	// n.logger.Sugar().Infof("Session - %s", sAddress)
	// n.logger.Sugar().Infof("Public Key - %s", hex.EncodeToString(pubKeyBytes))
}

func (s *Session) UpdateKeygenParty(
	message UpdateMessage,
) (*chan tss.Message, *chan error, error) {
	outChan, errChan := s.InitKeygen()
	fromPartyId := s.GetPartyId(message.GetPartyId().Id)
	_, err := (*s.keyGenParty).UpdateFromBytes(message.GetWireMessage(), fromPartyId, message.GetIsBroadcast())
	if err != nil {
		return nil, nil, err
	}
	return outChan, errChan, nil
}

func (n *Session) setupChannels() (*chan tss.Message, *chan error) {
	outChan := make(chan tss.Message)
	errChan := make(chan error)
	return &outChan, &errChan
}
