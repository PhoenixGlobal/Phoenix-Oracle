package p2pkey

import (
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"

	"gorm.io/gorm/schema"

	"gorm.io/gorm"

	keystore "github.com/ethereum/go-ethereum/accounts/keystore"
	cryptop2p "github.com/libp2p/go-libp2p-core/crypto"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
)

type Key struct {
	cryptop2p.PrivKey
}

func (k Key) ToV2() KeyV2 {
	return KeyV2{
		PrivKey: k.PrivKey,
		peerID:  k.PeerID(),
	}
}

type PublicKeyBytes []byte

func (pkb PublicKeyBytes) String() string {
	return hex.EncodeToString(pkb)
}

func (pkb PublicKeyBytes) MarshalJSON() ([]byte, error) {
	return json.Marshal(hex.EncodeToString(pkb))
}

func (pkb *PublicKeyBytes) UnmarshalJSON(input []byte) error {
	var hexString string
	if err := json.Unmarshal(input, &hexString); err != nil {
		return err
	}

	result, err := hex.DecodeString(hexString)
	if err != nil {
		return err
	}

	*pkb = PublicKeyBytes(result)
	return nil
}

func (pkb *PublicKeyBytes) Scan(value interface{}) error {
	switch v := value.(type) {
	case []byte:
		*pkb = v
		return nil
	default:
		return errors.Errorf("invalid public key bytes got %T wanted []byte", v)
	}
}

func (pkb PublicKeyBytes) Value() (driver.Value, error) {
	return []byte(pkb), nil
}

// GormDataType gorm common data type
func (PublicKeyBytes) GormDataType() string {
	return "bytea"
}

func (PublicKeyBytes) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "postgres":
		return "BYTEA"
	}
	return ""
}

func (k Key) GetPeerID() (PeerID, error) {
	peerID, err := peer.IDFromPrivateKey(k)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return PeerID(peerID), err
}

func (k Key) PeerID() PeerID {
	peerID, err := peer.IDFromPrivateKey(k)
	if err != nil {
		panic(err)
	}
	return PeerID(peerID)
}

type EncryptedP2PKey struct {
	ID               int32 `gorm:"primary_key"`
	PeerID           PeerID
	PubKey           PublicKeyBytes `gorm:"type:bytea"`
	EncryptedPrivKey []byte
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt
}

func (EncryptedP2PKey) TableName() string {
	return "encrypted_p2p_keys"
}

func (ep2pk *EncryptedP2PKey) SetID(value string) error {
	result, err := strconv.ParseInt(value, 10, 32)

	if err != nil {
		return err
	}

	ep2pk.ID = int32(result)
	return nil
}

func (ep2pk EncryptedP2PKey) Decrypt(auth string) (k Key, err error) {
	var cryptoJSON keystore.CryptoJSON
	err = json.Unmarshal(ep2pk.EncryptedPrivKey, &cryptoJSON)
	if err != nil {
		return k, errors.Wrapf(err, "invalid JSON for key 0x%x", ep2pk.PubKey)
	}
	marshalledPrivK, err := keystore.DecryptDataV3(cryptoJSON, adulteratedPassword(auth))
	if err != nil {
		return k, errors.Wrapf(err, "could not decrypt key 0x%x", ep2pk.PubKey)
	}
	privK, err := cryptop2p.UnmarshalPrivateKey(marshalledPrivK)
	if err != nil {
		return k, errors.Wrapf(err, "could not unmarshal private key for 0x%x", ep2pk.PubKey)
	}
	return Key{
		privK,
	}, nil
}
