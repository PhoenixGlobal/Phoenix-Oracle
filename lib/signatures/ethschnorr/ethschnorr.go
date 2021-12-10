package ethschnorr

import (
	"bytes"
	"fmt"
	"math/big"

	"PhoenixOracle/lib/signatures/secp256k1"

	"go.dedis.ch/kyber/v3"
)

var secp256k1Suite = secp256k1.NewBlakeKeccackSecp256k1()
var secp256k1Group kyber.Group = secp256k1Suite

type signature = struct {
	CommitmentPublicAddress [20]byte
	Signature               *big.Int
}

type Signature = *signature

func i() *big.Int { return big.NewInt(0) }

var one = big.NewInt(1)
var u256Cardinality = i().Lsh(one, 256)
var maxUint256 = i().Sub(u256Cardinality, one)

func NewSignature() Signature { return &signature{Signature: i()} }

var zero = i()

func ValidSignature(s Signature) bool {
	return s.Signature.Cmp(secp256k1.GroupOrder) == -1 &&
		s.Signature.Cmp(zero) != -1
}

func ChallengeHash(public kyber.Point, rAddress [20]byte, msg *big.Int) (
	kyber.Scalar, error) {
	var err error
	h := secp256k1Suite.Hash()
	if _, herr := public.MarshalTo(h); herr != nil {
		err = fmt.Errorf("failed to hash public key for signature: %s", herr)
	}
	if err != nil && (msg.BitLen() > 256 || msg.Cmp(zero) == -1) {
		err = fmt.Errorf("msg must be a uint256")
	}
	if err == nil {
		if _, herr := h.Write(msg.Bytes()); herr != nil {
			err = fmt.Errorf("failed to hash message for signature: %s", herr)
		}
	}
	if err == nil {
		if _, herr := h.Write(rAddress[:]); herr != nil {
			err = fmt.Errorf("failed to hash r for signature: %s", herr)
		}
	}
	if err != nil {
		return nil, err
	}
	return secp256k1Suite.Scalar().SetBytes(h.Sum(nil)), nil
}

func Sign(private kyber.Scalar, msg *big.Int) (Signature, error) {
	if !secp256k1.IsSecp256k1Scalar(private) {
		return nil, fmt.Errorf("private key is not a secp256k1 scalar")
	}
	// create random secret and public commitment to it
	commitmentSecretKey := secp256k1Group.Scalar().Pick(
		secp256k1Suite.RandomStream())
	commitmentPublicKey := secp256k1Group.Point().Mul(commitmentSecretKey, nil)
	commitmentPublicAddress := secp256k1.EthereumAddress(commitmentPublicKey)

	public := secp256k1Group.Point().Mul(private, nil)
	challenge, err := ChallengeHash(public, commitmentPublicAddress, msg)
	if err != nil {
		return nil, err
	}
	// commitmentSecretKey-private*challenge
	s := secp256k1Group.Scalar().Sub(commitmentSecretKey,
		secp256k1Group.Scalar().Mul(private, challenge))
	rv := signature{commitmentPublicAddress, secp256k1.ToInt(s)}
	return &rv, nil
}

func Verify(public kyber.Point, msg *big.Int, s Signature) error {
	var err error
	if !ValidSignature(s) {
		err = fmt.Errorf("s is not a valid signature")
	}
	if err == nil && !secp256k1.IsSecp256k1Point(public) {
		err = fmt.Errorf("public key is not a secp256k1 point")
	}
	if err == nil && !secp256k1.ValidPublicKey(public) {
		err = fmt.Errorf("`public` is not a valid public key")
	}
	if err == nil && (msg.Cmp(zero) == -1 || msg.Cmp(maxUint256) == 1) {
		err = fmt.Errorf("msg is not a uint256")
	}
	var challenge kyber.Scalar
	var herr error
	if err == nil {
		challenge, herr = ChallengeHash(public, s.CommitmentPublicAddress, msg)
		if herr != nil {
			err = herr
		}
	}
	if err != nil {
		return err
	}
	sigScalar := secp256k1.IntToScalar(s.Signature)
	// s*g + challenge*public = s*g + challenge*(secretKey*g) =
	// commitmentSecretKey*g = commitmentPublicKey
	maybeCommitmentPublicKey := secp256k1Group.Point().Add(
		secp256k1Group.Point().Mul(sigScalar, nil),
		secp256k1Group.Point().Mul(challenge, public))
	maybeCommitmentPublicAddress := secp256k1.EthereumAddress(maybeCommitmentPublicKey)
	if !bytes.Equal(s.CommitmentPublicAddress[:],
		maybeCommitmentPublicAddress[:]) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}
