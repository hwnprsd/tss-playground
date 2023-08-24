package crypto

import "math/big"

type PublicKey struct {
	value []byte
}

func (p PublicKey) BigInt() big.Int {
	return *new(big.Int).SetBytes(p.value)
}
