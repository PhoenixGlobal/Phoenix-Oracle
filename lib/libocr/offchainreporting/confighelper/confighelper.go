// Package confighelper provides helpers for converting between the gethwrappers/OffchainAggregator.SetConfig
// event and types.ContractConfig
package confighelper

import (
	"crypto/rand"
	"io"
	"math/big"
	"time"

	"PhoenixOracle/lib/libocr/gethwrappers/offchainaggregator"
	"PhoenixOracle/lib/libocr/offchainreporting/internal/config"
	"PhoenixOracle/lib/libocr/offchainreporting/types"
	"github.com/ethereum/go-ethereum/common"
)

// OracleIdentity is identical to the internal type in package config.
// We intentionally make a copy to make potential future internal modifications easier.
type OracleIdentity struct {
	OnChainSigningAddress types.OnChainSigningAddress
	TransmitAddress       common.Address
	OffchainPublicKey     types.OffchainPublicKey
	PeerID                string
}

// PublicConfig is identical to the internal type in package config.
// We intentionally make a copy to make potential future internal modifications easier.
type PublicConfig struct {
	DeltaProgress    time.Duration
	DeltaResend      time.Duration
	DeltaRound       time.Duration
	DeltaGrace       time.Duration
	DeltaC           time.Duration
	AlphaPPB         uint64
	DeltaStage       time.Duration
	RMax             uint8
	S                []int
	OracleIdentities []OracleIdentity

	F            int
	ConfigDigest types.ConfigDigest
}

func (pc PublicConfig) N() int {
	return len(pc.OracleIdentities)
}

func PublicConfigFromContractConfig(chainID *big.Int, skipChainSpecificChecks bool, change types.ContractConfig) (PublicConfig, error) {
	internalPublicConfig, err := config.PublicConfigFromContractConfig(chainID, skipChainSpecificChecks, change)
	if err != nil {
		return PublicConfig{}, err
	}
	identities := []OracleIdentity{}
	for _, internalIdentity := range internalPublicConfig.OracleIdentities {
		identities = append(identities, OracleIdentity{
			internalIdentity.OnChainSigningAddress,
			internalIdentity.TransmitAddress,
			internalIdentity.OffchainPublicKey,
			internalIdentity.PeerID,
		})
	}
	return PublicConfig{
		internalPublicConfig.DeltaProgress,
		internalPublicConfig.DeltaResend,
		internalPublicConfig.DeltaRound,
		internalPublicConfig.DeltaGrace,
		internalPublicConfig.DeltaC,
		internalPublicConfig.AlphaPPB,
		internalPublicConfig.DeltaStage,
		internalPublicConfig.RMax,
		internalPublicConfig.S,
		identities,
		internalPublicConfig.F,
		internalPublicConfig.ConfigDigest,
	}, nil
}

func ContractConfigFromConfigSetEvent(changed offchainaggregator.OffchainAggregatorConfigSet) types.ContractConfig {
	return types.ContractConfig{
		config.ConfigDigest(
			changed.Raw.Address,
			changed.ConfigCount,
			changed.Signers,
			changed.Transmitters,
			changed.Threshold,
			changed.EncodedConfigVersion,
			changed.Encoded,
		),
		changed.Signers,
		changed.Transmitters,
		changed.Threshold,
		changed.EncodedConfigVersion,
		changed.Encoded,
	}
}

type OracleIdentityExtra struct {
	OracleIdentity
	SharedSecretEncryptionPublicKey types.SharedSecretEncryptionPublicKey
}

// ContractSetConfigArgsForIntegrationTest generates setConfig args for integration tests in core
func ContractSetConfigArgsForIntegrationTest(
	oracles []OracleIdentityExtra,
	f int,
	alphaPPB uint64,
) (
	signers []common.Address,
	transmitters []common.Address,
	threshold uint8,
	encodedConfigVersion uint64,
	encodedConfig []byte,
	err error,
) {
	S := []int{}
	identities := []config.OracleIdentity{}
	sharedSecretEncryptionPublicKeys := []types.SharedSecretEncryptionPublicKey{}
	for _, oracle := range oracles {
		S = append(S, 1)
		identities = append(identities, config.OracleIdentity{
			oracle.PeerID,
			oracle.OffchainPublicKey,
			oracle.OnChainSigningAddress,
			oracle.TransmitAddress,
		})
		sharedSecretEncryptionPublicKeys = append(sharedSecretEncryptionPublicKeys, oracle.SharedSecretEncryptionPublicKey)
	}
	sharedConfig := config.SharedConfig{
		config.PublicConfig{
			2 * time.Second,
			1 * time.Second,
			1 * time.Second,
			500 * time.Millisecond,
			0,
			alphaPPB,
			2 * time.Second,
			3,
			S,
			identities,
			f,
			types.ConfigDigest{},
		},
		&[config.SharedSecretSize]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8},
	}
	return config.XXXContractSetConfigArgsFromSharedConfig(sharedConfig, sharedSecretEncryptionPublicKeys)
}

// ContractSetConfigArgsByNetType generates setConfig args by net type
//netType:1 SlowUpdates, 2 FastUpdates, 3 Testnet
func ContractSetConfigArgsByNetType(
	oracles []OracleIdentityExtra,
	f int,
	alphaPPB uint64,
	deltaC time.Duration,
	netType int,
) (
	signers []common.Address,
	transmitters []common.Address,
	threshold uint8,
	encodedConfigVersion uint64,
	encodedConfig []byte,
	err error,
) {
	S := []int{}
	identities := []config.OracleIdentity{}
	sharedSecretEncryptionPublicKeys := []types.SharedSecretEncryptionPublicKey{}
	for i, oracle := range oracles {
		if i==0{
			S = append(S, 1)
		}else {
			S = append(S, 2)
		}
		identities = append(identities, config.OracleIdentity{
			oracle.PeerID,
			oracle.OffchainPublicKey,
			oracle.OnChainSigningAddress,
			oracle.TransmitAddress,
		})
		sharedSecretEncryptionPublicKeys = append(sharedSecretEncryptionPublicKeys, oracle.SharedSecretEncryptionPublicKey)
	}

	sharedSecret := [config.SharedSecretSize]byte{}
	if _, err := io.ReadFull(rand.Reader, sharedSecret[:]); err != nil {
		return nil, nil, 0, 0, nil, err
	}

	var sharedConfig config.SharedConfig
	switch netType {
	case 1:
		sharedConfig = config.SharedConfig{
			config.PublicConfig{
				35 * time.Second,
				17 * time.Second,
				30 * time.Second,
				12 * time.Second,
				deltaC,
				alphaPPB,
				1 * time.Minute,
				3,
				S,
				identities,
				f,
				types.ConfigDigest{},
			},
			&sharedSecret,
		}
	case 2:
		sharedConfig = config.SharedConfig{
			config.PublicConfig{
				8 * time.Second,
				5 * time.Second,
				5 * time.Second,
				3 * time.Second,
				deltaC,
				alphaPPB,
				5 * time.Second,
				3,
				S,
				identities,
				f,
				types.ConfigDigest{},
			},
			&sharedSecret,
		}
	case 3:
		sharedConfig = config.SharedConfig{
			config.PublicConfig{
				2 * time.Second,
				2 * time.Second,
				1 * time.Second,
				500 * time.Millisecond,
				deltaC,
				alphaPPB,
				5 * time.Second,
				3,
				S,
				identities,
				f,
				types.ConfigDigest{},
			},
			&sharedSecret,
		}
	default:
		sharedConfig = config.SharedConfig{
			config.PublicConfig{
				23 * time.Second,
				10 * time.Second,
				20 * time.Second,
				12 * time.Second,
				deltaC,
				alphaPPB,
				10 * time.Second,
				3,
				S,
				identities,
				f,
				types.ConfigDigest{},
			},
			&sharedSecret,
		}
	}
	return config.XXXContractSetConfigArgsFromSharedConfig(sharedConfig, sharedSecretEncryptionPublicKeys)
}

// ContractSetConfigArgs generates setConfig args from the relevant parameters.
// Only use this for testing, *not* for production.
func ContractSetConfigArgs(
	deltaProgress time.Duration,
	deltaResend time.Duration,
	deltaRound time.Duration,
	deltaGrace time.Duration,
	deltaC time.Duration,
	alphaPPB uint64,
	deltaStage time.Duration,
	rMax uint8,
	s []int,
	oracles []OracleIdentityExtra,
	f int,
) (
	signers []common.Address,
	transmitters []common.Address,
	threshold uint8,
	encodedConfigVersion uint64,
	encodedConfig []byte,
	err error,
) {
	identities := []config.OracleIdentity{}
	sharedSecretEncryptionPublicKeys := []types.SharedSecretEncryptionPublicKey{}
	for _, oracle := range oracles {
		identities = append(identities, config.OracleIdentity{
			oracle.PeerID,
			oracle.OffchainPublicKey,
			oracle.OnChainSigningAddress,
			oracle.TransmitAddress,
		})
		sharedSecretEncryptionPublicKeys = append(sharedSecretEncryptionPublicKeys, oracle.SharedSecretEncryptionPublicKey)
	}

	sharedSecret := [config.SharedSecretSize]byte{}
	if _, err := io.ReadFull(rand.Reader, sharedSecret[:]); err != nil {
		return nil, nil, 0, 0, nil, err
	}

	sharedConfig := config.SharedConfig{
		config.PublicConfig{
			deltaProgress,
			deltaResend,
			deltaRound,
			deltaGrace,
			deltaC,
			alphaPPB,
			deltaStage,
			rMax,
			s,
			identities,
			f,
			types.ConfigDigest{},
		},
		&sharedSecret,
	}
	return config.XXXContractSetConfigArgsFromSharedConfig(sharedConfig, sharedSecretEncryptionPublicKeys)
}
