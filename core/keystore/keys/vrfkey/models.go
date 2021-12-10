package vrfkey

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"PhoenixOracle/lib/signatures/secp256k1"

	"PhoenixOracle/util"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type EncryptedVRFKey struct {
	PublicKey secp256k1.PublicKey `gorm:"primary_key"`
	VRFKey    gethKeyStruct       `json:"vrf_key"`
	CreatedAt time.Time           `json:"-"`
	UpdatedAt time.Time           `json:"-"`
	DeletedAt gorm.DeletedAt      `json:"-"`
}

func (e *EncryptedVRFKey) JSON() ([]byte, error) {
	keyJSON, err := json.Marshal(e)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal encrypted key to JSON")
	}
	return keyJSON, nil
}

func (e *EncryptedVRFKey) WriteToDisk(path string) error {
	keyJSON, err := e.JSON()
	if err != nil {
		return errors.Wrapf(err, "while marshaling key to save to %s", path)
	}
	userReadWriteOtherNoAccess := os.FileMode(0600)
	return utils.WriteFileWithMaxPerms(path, keyJSON, userReadWriteOtherNoAccess)
}

// Copied from go-ethereum/accounts/keystore/key.go's encryptedKeyJSONV3
type gethKeyStruct struct {
	Address string              `json:"address"`
	Crypto  keystore.CryptoJSON `json:"crypto"`
	Version int                 `json:"version"`
}

func (k gethKeyStruct) Value() (driver.Value, error) {
	return json.Marshal(&k)
}

func (k *gethKeyStruct) Scan(value interface{}) error {
	// With sqlite gorm driver, we get a []byte, here. With postgres, a string!
	// https://github.com/jinzhu/gorm/issues/2276
	var toUnmarshal []byte
	switch s := value.(type) {
	case []byte:
		toUnmarshal = s
	case string:
		toUnmarshal = []byte(s)
	default:
		return errors.Wrap(
			fmt.Errorf("unable to convert %+v of type %T to gethKeyStruct",
				value, value), "scan failure")
	}
	return json.Unmarshal(toUnmarshal, k)
}
