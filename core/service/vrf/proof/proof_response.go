package proof

import (
	"math/big"

	"PhoenixOracle/internal/gethwrappers/generated/vrf_coordinator_v2"

	"PhoenixOracle/lib/signatures/secp256k1"

	"PhoenixOracle/core/keystore"
	"PhoenixOracle/core/keystore/keys/vrfkey"
	"PhoenixOracle/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

type ProofResponse struct {
	P        vrfkey.Proof
	PreSeed  Seed
	BlockNum uint64
}

const OnChainResponseLength = ProofLength +
	32 // blocknum


type MarshaledOnChainResponse [OnChainResponseLength]byte

func (p *ProofResponse) MarshalForVRFCoordinator() (
	response MarshaledOnChainResponse, err error) {
	solidityProof, err := SolidityPrecalculations(&p.P)
	if err != nil {
		return MarshaledOnChainResponse{}, errors.Wrap(err,
			"while marshaling proof for VRFCoordinator")
	}
	solidityProof.P.Seed = common.BytesToHash(p.PreSeed[:]).Big()
	mProof := solidityProof.MarshalForSolidityVerifier()
	wireBlockNum := utils.EVMWordUint64(p.BlockNum)
	rl := copy(response[:], append(mProof[:], wireBlockNum...))
	if rl != OnChainResponseLength {
		return MarshaledOnChainResponse{}, errors.Errorf(
			"wrong length for response to VRFCoordinator")
	}
	return response, nil
}

func UnmarshalProofResponse(m MarshaledOnChainResponse) (*ProofResponse, error) {
	blockNum := common.BytesToHash(m[ProofLength : ProofLength+32]).Big().Uint64()
	proof, err := UnmarshalSolidityProof(m[:ProofLength])
	if err != nil {
		return nil, errors.Wrap(err, "while parsing ProofResponse")
	}
	preSeed, err := BigToSeed(proof.Seed)
	if err != nil {
		return nil, errors.Wrap(err, "while converting seed to bytes representation")
	}
	return &ProofResponse{P: proof, PreSeed: preSeed, BlockNum: blockNum}, nil
}

func (p ProofResponse) CryptoProof(s PreSeedData) (vrfkey.Proof, error) {
	proof := p.P // Copy P, which has wrong seed value
	proof.Seed = FinalSeed(s)
	valid, err := proof.VerifyVRFProof()
	if err != nil {
		return vrfkey.Proof{}, errors.Wrap(err,
			"could not validate proof implied by on-chain response")
	}
	if !valid {
		return vrfkey.Proof{}, errors.Errorf(
			"proof implied by on-chain response is invalid")
	}
	return proof, nil
}

func GenerateProofResponseFromProof(proof vrfkey.Proof, s PreSeedData) (MarshaledOnChainResponse, error) {
	p := ProofResponse{P: proof, PreSeed: s.PreSeed, BlockNum: s.BlockNum}
	rv, err := p.MarshalForVRFCoordinator()
	if err != nil {
		return MarshaledOnChainResponse{}, err
	}
	return rv, nil
}

func GenerateProofResponseFromProofV2(p vrfkey.Proof, s PreSeedDataV2) (vrf_coordinator_v2.VRFProof, vrf_coordinator_v2.VRFCoordinatorV2RequestCommitment, error) {
	var proof vrf_coordinator_v2.VRFProof
	var rc vrf_coordinator_v2.VRFCoordinatorV2RequestCommitment
	solidityProof, err := SolidityPrecalculations(&p)
	if err != nil {
		return proof, rc, errors.Wrap(err,
			"while marshaling proof for VRFCoordinatorV2")
	}
	solidityProof.P.Seed = common.BytesToHash(s.PreSeed[:]).Big()
	x, y := secp256k1.Coordinates(solidityProof.P.PublicKey)
	gx, gy := secp256k1.Coordinates(solidityProof.P.Gamma)
	cgx, cgy := secp256k1.Coordinates(solidityProof.CGammaWitness)
	shx, shy := secp256k1.Coordinates(solidityProof.SHashWitness)
	return vrf_coordinator_v2.VRFProof{
			Pk:            [2]*big.Int{x, y},
			Gamma:         [2]*big.Int{gx, gy},
			C:             solidityProof.P.C,
			S:             solidityProof.P.S,
			Seed:          common.BytesToHash(s.PreSeed[:]).Big(),
			UWitness:      solidityProof.UWitness,
			CGammaWitness: [2]*big.Int{cgx, cgy},
			SHashWitness:  [2]*big.Int{shx, shy},
			ZInv:          solidityProof.ZInv,
		}, vrf_coordinator_v2.VRFCoordinatorV2RequestCommitment{
			BlockNum:         s.BlockNum,
			SubId:            s.SubId,
			CallbackGasLimit: s.CallbackGasLimit,
			NumWords:         s.NumWords,
			Sender:           s.Sender,
		}, nil
}

func GenerateProofResponse(keystore keystore.VRF, id string, s PreSeedData) (
	MarshaledOnChainResponse, error) {
	seed := FinalSeed(s)
	proof, err := keystore.GenerateProof(id, seed)
	if err != nil {
		return MarshaledOnChainResponse{}, err
	}
	return GenerateProofResponseFromProof(proof, s)
}

func GenerateProofResponseV2(keystore keystore.VRF, id string, s PreSeedDataV2) (
	vrf_coordinator_v2.VRFProof, vrf_coordinator_v2.VRFCoordinatorV2RequestCommitment, error) {
	seedHashMsg := append(s.PreSeed[:], s.BlockHash.Bytes()...)
	seed := utils.MustHash(string(seedHashMsg)).Big()
	proof, err := keystore.GenerateProof(id, seed)
	if err != nil {
		return vrf_coordinator_v2.VRFProof{}, vrf_coordinator_v2.VRFCoordinatorV2RequestCommitment{}, err
	}
	return GenerateProofResponseFromProofV2(proof, s)
}
