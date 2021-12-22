package utils

import (
	"PhoenixOracle/lib/libocr/offchainreporting/confighelper"
	ocrtypes "PhoenixOracle/lib/libocr/offchainreporting/types"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/curve25519"
	"testing"
	"time"
)

func TestSetupConfig(t *testing.T)  {
	var oracles      []confighelper.OracleIdentityExtra
	var transmitters=[]common.Address{
		common.HexToAddress("0x6f7bADc7eD84D64DB7bF3bB8db6A6dADb52607e2"),
		common.HexToAddress("0x22D2a184da4E94625E3FdAd977b983e91312519E"),
		common.HexToAddress("0xdEc9025bAeBE9Ece7F1c07dCBf9852E218aB7A6e"),
		common.HexToAddress("0x4652C4a429Ba63a129b96ceea042660bC0017088"),
	}
	var signAddresses=[]common.Address{
		common.HexToAddress("0x8511adc4a1c6da31d3e50e533f4cca2ec1d081ab"),
		common.HexToAddress("0x865b601a966bd7ff8d554f545cf295c176c230f0"),
		common.HexToAddress("0xb87b4ddb1ef07e7a4dc97cf3764387518ab3571e"),
		common.HexToAddress("0x0cbf28aa7f8809b6a03f310fdcb4c084fe841ca6"),
	}
	var publicKeyOffChains=[]string{
		"ad46563af4762b90fcc20a831da8a4575243ac17fe521e8d50d81241790487b6",
		"5ecec424b832d6c9a5185eb192b7ae333b59c97c9ec7bceadd7b1c6a746b321f",
		"113b47c42c44542c2db532fc3bc1d256a9648d62c1390fe318011df630269514",
		"7f5dbf1ed958bc7cb730ed182a655b3fc6feb66130bbfdd9ac884d253992937f",
	}
	var publicKeyConfigs=[]string{
		"8119536ef0f9877f46dbbd6971c1c49176bc0266a83138b164d5e6c2ccc4225a",
		"1f109ba50b9c3d53ac28297aeee5ddd296fb20796b548ef83b82ff2af0150b0d",
		"f63f880372b42fb7595e445ba0da4ed833d83aaa34ffd85f3b1aa66bec2dcd30",
		"706d2b50968c9808cfb27ce15f6900002927b1501f9f98e5c959334e05e9c359",
	}
	var peerIDs=[]string{
		"12D3KooWLCGTxShexbb3wFu8qfnn3NmfnyK4KRE9SeFY3mNSCQBa",
		"12D3KooWRp5dTPx8QNDqTQMMqCWQMAY3uHfuoU7rw3cjKg9QMmh6",
		"12D3KooWGMRjindteF4ksd6oUqZ2TmURpijf4i5XrMDBsq4Tuwt1",
		"12D3KooWQj8VgLSwRWSbxfza6qtHmwu1gor6CjcZHdPAQ1zkX2Dy",
	}
	for i := 0; i < 4; i++ {
		publicKeyOffChain, _ :=hex.DecodeString(publicKeyOffChains[i])
		publicKeyOffChain2:=ocrtypes.OffchainPublicKey(publicKeyOffChain)

		publicKeyConfig,_:=hex.DecodeString(publicKeyConfigs[i])
		var rvFixed [curve25519.PointSize]byte
		copy(rvFixed[:], publicKeyConfig)

		oracles = append(oracles, confighelper.OracleIdentityExtra{
			OracleIdentity: confighelper.OracleIdentity{
				OnChainSigningAddress: ocrtypes.OnChainSigningAddress(signAddresses[i]),
				TransmitAddress:       transmitters[i],
				OffchainPublicKey:     ocrtypes.OffchainPublicKey(publicKeyOffChain2),
				PeerID:                peerIDs[i],
			},

			SharedSecretEncryptionPublicKey: ocrtypes.SharedSecretEncryptionPublicKey(rvFixed),
		})
	}
	deltaC:=10 * time.Minute
	netType:=1
	signers, transmitters, threshold, encodedConfigVersion, encodedConfig, err := confighelper.ContractSetConfigArgsByNetType(
		oracles,
		1,
		1000000000/100, // threshold PPB
		deltaC,  // deltaC
		netType, // netType:1 SlowUpdates, 2 FastUpdates, 3 Testnet
	)
	fmt.Println(1111000,err)
	fmt.Println(111,signers)
	fmt.Println(222,transmitters)
	fmt.Println(333,threshold)
	fmt.Println(444,encodedConfigVersion)

	fmt.Println(len(encodedConfig),555,encodedConfig)
	encodedConfig2:=hex.EncodeToString(encodedConfig)
	fmt.Println(len(encodedConfig2),666,encodedConfig2)
	return
}