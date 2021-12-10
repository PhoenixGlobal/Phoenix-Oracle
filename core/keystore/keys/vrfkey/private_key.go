package vrfkey

import (
	"encoding/json"
	"fmt"
	"math/big"

	"PhoenixOracle/lib/signatures/secp256k1"
	keystore "github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.dedis.ch/kyber/v3"
)

type PrivateKey struct {
	k         kyber.Scalar
	PublicKey secp256k1.PublicKey
}

func newPrivateKey(rawKey *big.Int) (*PrivateKey, error) {
	if rawKey.Cmp(secp256k1.GroupOrder) >= 0 || rawKey.Cmp(big.NewInt(0)) <= 0 {
		return nil, fmt.Errorf("secret key must be in {1, ..., #secp256k1 - 1}")
	}
	sk := &PrivateKey{}
	sk.k = secp256k1.IntToScalar(rawKey)
	pk, err := suite.Point().Mul(sk.k, nil).MarshalBinary()
	if err != nil {
		panic(errors.Wrapf(err, "could not marshal public key"))
	}
	if len(pk) != secp256k1.CompressedPublicKeyLength {
		panic(fmt.Errorf("public key %x has wrong length", pk))
	}
	if l := copy(sk.PublicKey[:], pk); l != secp256k1.CompressedPublicKeyLength {
		panic(fmt.Errorf("failed to copy correct length in serialized public key"))
	}
	return sk, nil
}

func (k PrivateKey) ToV2() KeyV2 {
	return KeyV2{
		k:         &k.k,
		PublicKey: k.PublicKey,
	}
}

func fromGethKey(gethKey *keystore.Key) *PrivateKey {
	secretKey := secp256k1.IntToScalar(gethKey.PrivateKey.D)
	rawPublicKey, err := secp256k1.ScalarToPublicPoint(secretKey).MarshalBinary()
	if err != nil {
		panic(err) // Only way this can happen is out-of-memory failure
	}
	var publicKey secp256k1.PublicKey
	copy(publicKey[:], rawPublicKey)
	return &PrivateKey{secretKey, publicKey}
}

func (k *PrivateKey) String() string {
	return fmt.Sprintf("PrivateKey{k: <redacted>, PublicKey: %s}", k.PublicKey)
}

func (k *PrivateKey) GoString() string {
	return k.String()
}

func Decrypt(e EncryptedVRFKey, auth string) (*PrivateKey, error) {
	// NOTE: We do this shuffle to an anonymous struct
	// solely to add a a throwaway UUID, so we can leverage
	// the keystore.DecryptKey from the geth which requires it
	// as of 1.10.0.
	keyJSON, err := json.Marshal(struct {
		Address string              `json:"address"`
		Crypto  keystore.CryptoJSON `json:"crypto"`
		Version int                 `json:"version"`
		Id      string              `json:"id"`
	}{
		Address: e.VRFKey.Address,
		Crypto:  e.VRFKey.Crypto,
		Version: e.VRFKey.Version,
		Id:      uuid.New().String(),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "while marshaling key for decryption")
	}
	gethKey, err := keystore.DecryptKey(keyJSON, adulteratedPassword(auth))
	if err != nil {
		return nil, errors.Wrapf(err, "could not decrypt key %s",
			e.PublicKey.String())
	}
	return fromGethKey(gethKey), nil
}
