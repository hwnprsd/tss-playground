package crypto

import (
	"github.com/ethereum/go-ethereum/crypto"
)

type PrivateKey struct {
	value []byte
}

func GenerateRandomKey() PrivateKey {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}
	return PrivateKey{
		value: crypto.FromECDSA(privateKey),
	}
}

func (p PrivateKey) Public() PublicKey {
	pk, err := crypto.ToECDSA(p.value)
	if err != nil {
		panic(err)
	}
	return PublicKey{
		value: crypto.FromECDSAPub(&pk.PublicKey),
	}
}
