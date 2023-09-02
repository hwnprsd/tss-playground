package squad

import (
	"encoding/hex"
	"log"
	"math/big"

	"github.com/bnb-chain/tss-lib/common"
	"github.com/bnb-chain/tss-lib/ecdsa/signing"
	"github.com/bnb-chain/tss-lib/tss"
)

func (s *Squad) InitSigning(message []byte) (*chan tss.Message, *chan error) {
	shouldContinueInit, outChan, errChan := s.setupSigningParty(message)
	if !shouldContinueInit {
		return nil, nil
	}
	go func() {
		err := (*s.sigParty).Start()
		log.Println("Starting to Sign")
		if err != nil {
			log.Println("SIG_ERROR", err)
		}
	}()
	return outChan, errChan
}

func (s *Squad) setupSigningParty(message []byte) (shouldContinueInit bool, outChan *chan tss.Message, errChan *chan error) {
	// KeyGen is not completed
	if s.keyGenData == nil {
		log.Println("KeyGen Data is NIL")
		return false, nil, nil
	}

	// In an ongoing session. No need to init
	// Or node is in a broken state
	if s.sigParty != nil {
		return false, nil, nil
	}

	msg := new(big.Int).SetBytes(message)
	parties := s.GetPartiesSorted()
	peerCtx := tss.NewPeerContext(parties)
	params := tss.NewParameters(tss.S256(), peerCtx, s.partyId, len(parties), len(parties)-1)

	outChan, errChan = s.setupChannels()
	endChan := make(chan common.SignatureData)

	go func() {
		for {
			select {
			case endData := <-endChan:
				s.handleSessionEnd(&endData)
			}
		}
	}()

	party := signing.NewLocalParty(msg, params, *s.keyGenData, *outChan, endChan)
	s.sigParty = &party
	return true, outChan, errChan
}

func (s *Squad) handleSessionEnd(data *common.SignatureData) {
	log.Println(hex.EncodeToString(data.Signature))
	s.sigParty = nil
}

func (s *Squad) UpdateSigningParty(message UpdateMessage) (*chan tss.Message, *chan error, error) {
	outChan, errChan := s.InitSigning(message.GetSigMessage())
	fromPartyId := s.GetPartyId(message.GetPartyId().Id)
	_, err := (*s.sigParty).UpdateFromBytes(message.GetWireMessage(), fromPartyId, message.GetIsBroadcast())
	if err != nil {
		return nil, nil, err
	}
	return outChan, errChan, nil
}
