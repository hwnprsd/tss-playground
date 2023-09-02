package session

import (
	"sync"

	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/tss"
	"github.com/hwnprsd/tss/common"
	"github.com/hwnprsd/tss/proto"
)

type PeerMap map[string]*common.Peer

type Session struct {
	preParams *keygen.LocalPreParams

	peerLock    sync.RWMutex
	messageLock sync.RWMutex
	peers       PeerMap

	keyGenParty *tss.Party
	keyGenData  *keygen.LocalPartySaveData

	// parties map[string]*PeerData

	sigParty *tss.Party

	partyId *tss.PartyID
}

func NewSession(partyId *tss.PartyID, peers PeerMap, preParams *keygen.LocalPreParams) *Session {
	return &Session{
		peers:     peers,
		partyId:   partyId,
		preParams: preParams,
	}
}

func (s *Session) GetPartiesSorted() (parties []*tss.PartyID) {
	s.peerLock.RLock()
	defer s.peerLock.RUnlock()
	for _, p := range s.peers {
		parties = append(parties, common.ToPartyId(p.GetVersion().PartyId))
	}
	parties = append(parties, s.partyId)
	parties = tss.SortPartyIDs(parties)
	return
}

func (s *Session) GetPartyId(id string) *tss.PartyID {
	parties := s.GetPartiesSorted()
	for _, party := range parties {
		if party.Id == id {
			return party
		}
	}
	return nil
}

func (s *Session) GetParties() {

}

type UpdateMessage interface {
	GetWireMessage() []byte
	GetIsBroadcast() bool
	GetPartyId() *proto.PartyId
	GetSigMessage() []byte
}
