package vrfkey

import (
	"crypto/ecdsa"
	"encoding/json"

	"PhoenixOracle/lib/signatures/secp256k1"
	"PhoenixOracle/util"
	keystore "github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

func FromEncryptedJSON(keyJSON []byte, password string) (KeyV2, error) {
	var export EncryptedVRFKeyExport
	if err := json.Unmarshal(keyJSON, &export); err != nil {
		return KeyV2{}, err
	}

	keyJSON, err := json.Marshal(struct {
		Address string              `json:"address"`
		Crypto  keystore.CryptoJSON `json:"crypto"`
		Version int                 `json:"version"`
		Id      string              `json:"id"`
	}{
		Address: export.VRFKey.Address,
		Crypto:  export.VRFKey.Crypto,
		Version: export.VRFKey.Version,
		Id:      uuid.New().String(),
	})
	if err != nil {
		return KeyV2{}, errors.Wrapf(err, "while marshaling key for decryption")
	}

	gethKey, err := keystore.DecryptKey(keyJSON, adulteratedPassword(password))
	if err != nil {
		return KeyV2{}, errors.Wrapf(err, "could not decrypt key %s", export.PublicKey.String())
	}

	key := Raw(gethKey.PrivateKey.D.Bytes()).Key()
	return key, nil
}

type EncryptedVRFKeyExport struct {
	PublicKey secp256k1.PublicKey `json:"PublicKey"`
	VRFKey    gethKeyStruct       `json:"vrf_key"`
}

func (key KeyV2) ToEncryptedJSON(password string, scryptParams utils.ScryptParams) (export []byte, err error) {
	cryptoJSON, err := keystore.EncryptKey(key.toGethKey(), adulteratedPassword(password), scryptParams.N, scryptParams.P)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to encrypt key %s", key.ID())
	}
	var gethKey gethKeyStruct
	err = json.Unmarshal(cryptoJSON, &gethKey)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal key %s", key.ID())
	}
	encryptedOCRKExport := EncryptedVRFKeyExport{
		PublicKey: key.PublicKey,
		VRFKey:    gethKey,
	}
	return json.Marshal(encryptedOCRKExport)
}

func (key KeyV2) toGethKey() *keystore.Key {
	return &keystore.Key{
		Address:    key.PublicKey.Address(),
		PrivateKey: &ecdsa.PrivateKey{D: secp256k1.ToInt(*key.k)},
	}
}

var passwordPrefix = "don't mix VRF and Ethereum keys!"

func adulteratedPassword(password string) string {
	return passwordPrefix + password
}
