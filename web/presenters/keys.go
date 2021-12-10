package presenters

import (
	"time"

	"PhoenixOracle/core/assets"
	"PhoenixOracle/core/keystore/keys/ethkey"
	"PhoenixOracle/core/keystore/keys/csakey"
	"PhoenixOracle/core/keystore/keys/ocrkey"
	"PhoenixOracle/core/keystore/keys/p2pkey"
	"PhoenixOracle/core/keystore/keys/vrfkey"
	"PhoenixOracle/lib/logger"
)

type ETHKeyResource struct {
	JAID
	Address     string       `json:"address"`
	EthBalance  *assets.Eth  `json:"ethBalance"`
	PhbBalance *assets.Phb `json:"phbBalance"`
	IsFunding   bool         `json:"isFunding"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}

func (r ETHKeyResource) GetName() string {
	return "eTHKeys"
}

type NewETHKeyOption func(*ETHKeyResource) error

func NewETHKeyResource(k ethkey.KeyV2, state ethkey.State, opts ...NewETHKeyOption) (*ETHKeyResource, error) {
	r := &ETHKeyResource{
		JAID:        NewJAID(k.Address.Hex()),
		Address:     k.Address.Hex(),
		EthBalance:  nil,
		PhbBalance: nil,
		IsFunding:   state.IsFunding,
		CreatedAt:   state.CreatedAt,
		UpdatedAt:   state.UpdatedAt,
	}

	for _, opt := range opts {
		err := opt(r)

		if err != nil {
			return nil, err
		}
	}

	return r, nil
}

func SetETHKeyEthBalance(ethBalance *assets.Eth) NewETHKeyOption {
	return func(r *ETHKeyResource) error {
		r.EthBalance = ethBalance

		return nil
	}
}

func SetETHKeyPhbBalance(phbBalance *assets.Phb) NewETHKeyOption {
	return func(r *ETHKeyResource) error {
		r.PhbBalance = phbBalance

		return nil
	}
}

type CSAKeyResource struct {
	JAID
	PubKey  string `json:"publicKey"`
	Version int    `json:"version"`
}

func (CSAKeyResource) GetName() string {
	return "csaKeys"
}

func NewCSAKeyResource(key csakey.KeyV2) *CSAKeyResource {
	r := &CSAKeyResource{
		JAID:    NewJAID(key.ID()),
		PubKey:  key.PublicKeyString(),
		Version: 1,
	}

	return r
}

func NewCSAKeyResources(keys []csakey.KeyV2) []CSAKeyResource {
	rs := []CSAKeyResource{}
	for _, key := range keys {
		rs = append(rs, *NewCSAKeyResource(key))
	}

	return rs
}

type OCRKeysBundleResource struct {
	JAID
	OnChainSigningAddress ocrkey.OnChainSigningAddress `json:"onChainSigningAddress"`
	OffChainPublicKey     ocrkey.OffChainPublicKey     `json:"offChainPublicKey"`
	ConfigPublicKey       ocrkey.ConfigPublicKey       `json:"configPublicKey"`
}

func (r OCRKeysBundleResource) GetName() string {
	return "keyV2s"
}

func NewOCRKeysBundleResource(key ocrkey.KeyV2) *OCRKeysBundleResource {
	return &OCRKeysBundleResource{
		JAID:                  NewJAID(key.ID()),
		OnChainSigningAddress: key.OnChainSigning.Address(),
		OffChainPublicKey:     key.OffChainSigning.PublicKey(),
		ConfigPublicKey:       key.PublicKeyConfig(),
	}
}

func NewOCRKeysBundleResources(keys []ocrkey.KeyV2) []OCRKeysBundleResource {
	rs := []OCRKeysBundleResource{}
	for _, key := range keys {
		rs = append(rs, *NewOCRKeysBundleResource(key))
	}

	return rs
}

type P2PKeyResource struct {
	JAID
	PeerID string `json:"peerId"`
	PubKey string `json:"publicKey"`
}

func (P2PKeyResource) GetName() string {
	return "encryptedP2PKeys"
}

func NewP2PKeyResource(key p2pkey.KeyV2) *P2PKeyResource {
	r := &P2PKeyResource{
		JAID:   JAID{ID: key.ID()},
		PeerID: key.PeerID().String(),
		PubKey: key.PublicKeyHex(),
	}

	return r
}

func NewP2PKeyResources(keys []p2pkey.KeyV2) []P2PKeyResource {
	rs := []P2PKeyResource{}
	for _, key := range keys {
		rs = append(rs, *NewP2PKeyResource(key))
	}

	return rs
}

type VRFKeyResource struct {
	JAID
	Compressed   string `json:"compressed"`
	Uncompressed string `json:"uncompressed"`
	Hash         string `json:"hash"`
}

func (VRFKeyResource) GetName() string {
	return "encryptedVRFKeys"
}

func NewVRFKeyResource(key vrfkey.KeyV2) *VRFKeyResource {
	uncompressed, err := key.PublicKey.StringUncompressed()
	if err != nil {
		logger.Error("unable to get uncompressed pk", "err", err)
	}
	return &VRFKeyResource{
		JAID:         NewJAID(key.PublicKey.String()),
		Compressed:   key.PublicKey.String(),
		Uncompressed: uncompressed,
		Hash:         key.PublicKey.MustHash().String(),
	}
}

func NewVRFKeyResources(keys []vrfkey.KeyV2) []VRFKeyResource {
	rs := []VRFKeyResource{}
	for _, key := range keys {
		rs = append(rs, *NewVRFKeyResource(key))
	}

	return rs
}
