package ethdss

import (
	"bytes"
	"errors"
	"math/big"

	"PhoenixOracle/lib/signatures/ethschnorr"
	"PhoenixOracle/lib/signatures/secp256k1"

	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/share"
)

type Suite interface {
	kyber.Group
	kyber.HashFactory
	kyber.Random
}

var secp256k1Suite = secp256k1.NewBlakeKeccackSecp256k1()
var secp256k1Group kyber.Group = secp256k1Suite

type DistKeyShare interface {
	PriShare() *share.PriShare
	Commitments() []kyber.Point
}

type DSS struct {
	secret kyber.Scalar
	public kyber.Point
	index int
	participants []kyber.Point
	T int
	long DistKeyShare
	random DistKeyShare
	longPoly *share.PubPoly
	randomPoly *share.PubPoly
	msg *big.Int
	partials []*share.PriShare
	partialsIdx map[int]bool
	signed bool
	sessionID []byte
}

type DSSArgs = struct {
	secret       kyber.Scalar
	participants []kyber.Point
	long         DistKeyShare
	random       DistKeyShare
	msg          *big.Int
	T            int
}

type PartialSig struct {
	Partial   *share.PriShare
	SessionID []byte
	Signature ethschnorr.Signature
}

func NewDSS(args DSSArgs) (*DSS, error) {
	public := secp256k1Group.Point().Mul(args.secret, nil)
	var i int
	var found bool
	for j, p := range args.participants {
		if p.Equal(public) {
			found = true
			i = j
			break
		}
	}
	if !found {
		return nil, errors.New("dss: public key not found in list of participants")
	}
	return &DSS{
		secret:       args.secret,
		public:       public,
		index:        i,
		participants: args.participants,
		long:         args.long,
		longPoly: share.NewPubPoly(secp256k1Suite,
			secp256k1Group.Point().Base(), args.long.Commitments()),
		random: args.random,
		randomPoly: share.NewPubPoly(secp256k1Suite,
			secp256k1Group.Point().Base(), args.random.Commitments()),
		msg:         args.msg,
		T:           args.T,
		partialsIdx: make(map[int]bool),
		sessionID:   sessionID(secp256k1Suite, args.long, args.random),
	}, nil
}

func (d *DSS) PartialSig() (*PartialSig, error) {
	secretPartialLongTermKey := d.long.PriShare().V     // ɑᵢ, in the paper
	secretPartialCommitmentKey := d.random.PriShare().V // βᵢ, in the paper
	fullChallenge := d.hashSig()                        // h(m‖V), in the paper
	secretChallengeMultiple := secp256k1Suite.Scalar().Mul(
		fullChallenge, secretPartialLongTermKey) // ɑᵢh(m‖V)G, in the paper
	// Corresponds to ɣᵢG=βᵢG+ɑᵢh(m‖V)G in the paper, but NB, in its notation, we
	// use ɣᵢG=βᵢG-ɑᵢh(m‖V)G. (Subtract instead of add.)
	partialSignature := secp256k1Group.Scalar().Sub(
		secretPartialCommitmentKey, secretChallengeMultiple)
	ps := &PartialSig{
		Partial:   &share.PriShare{V: partialSignature, I: d.index},
		SessionID: d.sessionID,
	}
	var err error
	ps.Signature, err = ethschnorr.Sign(d.secret, ps.Hash()) // sign share
	if !d.signed {
		d.partialsIdx[d.index] = true
		d.partials = append(d.partials, ps.Partial)
		d.signed = true
	}
	return ps, err
}

func (d *DSS) ProcessPartialSig(ps *PartialSig) error {
	var err error
	public, ok := findPub(d.participants, ps.Partial.I)
	if !ok {
		err = errors.New("dss: partial signature with invalid index")
	}
	// nothing secret here
	if err == nil && !bytes.Equal(ps.SessionID, d.sessionID) {
		err = errors.New("dss: session id do not match")
	}
	if err == nil {
		if vrr := ethschnorr.Verify(public, ps.Hash(), ps.Signature); vrr != nil {
			err = vrr
		}
	}
	if err == nil {
		if _, ok := d.partialsIdx[ps.Partial.I]; ok {
			err = errors.New("dss: partial signature already received from peer")
		}
	}
	if err != nil {
		return err
	}
	hash := d.hashSig() // h(m‖V), in the paper's notation
	idx := ps.Partial.I
	// βᵢG=sum(cₖi^kG), in the paper, defined as sᵢ in step 2 of section 2.4
	randShare := d.randomPoly.Eval(idx)
	// ɑᵢG=sum(bₖi^kG), defined as sᵢ in step 2 of section 2.4
	longShare := d.longPoly.Eval(idx)
	// h(m‖V)(Y+...) term from equation (3) of the paper. AKA h(m‖V)ɑᵢG
	challengeSummand := secp256k1Group.Point().Mul(hash, longShare.V)
	// RHS of equation (3), except we subtract the second term instead of adding.
	// AKA (βᵢ-ɑᵢh(m‖V))G, which should equal ɣᵢG, according to equation (3)
	maybePartialSigCommitment := secp256k1Group.Point().Sub(randShare.V,
		challengeSummand)
	// Check that equation (3) holds (ɣᵢ is represented as ps.Partial.V, here.)
	partialSigCommitment := secp256k1Group.Point().Mul(ps.Partial.V, nil)
	if !partialSigCommitment.Equal(maybePartialSigCommitment) {
		return errors.New("dss: partial signature not valid")
	}
	d.partialsIdx[ps.Partial.I] = true
	d.partials = append(d.partials, ps.Partial)
	return nil
}

func (d *DSS) EnoughPartialSig() bool {
	return len(d.partials) >= d.T
}

func (d *DSS) Signature() (ethschnorr.Signature, error) {
	if !d.EnoughPartialSig() {
		return nil, errors.New("dkg: not enough partial signatures to sign")
	}
	// signature corresponds to σ in step 4 of section 4.2
	signature, err := share.RecoverSecret(secp256k1Suite, d.partials, d.T,
		len(d.participants))
	if err != nil {
		return nil, err
	}
	rv := ethschnorr.NewSignature()
	rv.Signature = secp256k1.ToInt(signature)
	// commitmentPublicKey corresponds to V in step 4 of section 4.2
	commitmentPublicKey := d.random.Commitments()[0]
	rv.CommitmentPublicAddress = secp256k1.EthereumAddress(commitmentPublicKey)
	return rv, nil
}

func (d *DSS) hashSig() kyber.Scalar {
	v := d.random.Commitments()[0] // Public-key commitment, in signature from d
	vAddress := secp256k1.EthereumAddress(v)
	publicKey := d.long.Commitments()[0]
	rv, err := ethschnorr.ChallengeHash(publicKey, vAddress, d.msg)
	if err != nil {
		panic(err)
	}
	return rv
}

func Verify(public kyber.Point, msg *big.Int, sig ethschnorr.Signature) error {
	return ethschnorr.Verify(public, msg, sig)
}

func (ps *PartialSig) Hash() *big.Int {
	h := secp256k1Suite.Hash()
	_, _ = h.Write(ps.Partial.Hash(secp256k1Suite))
	_, _ = h.Write(ps.SessionID)
	return (&big.Int{}).SetBytes(h.Sum(nil))
}

func findPub(list []kyber.Point, i int) (kyber.Point, bool) {
	if i >= len(list) {
		return nil, false
	}
	return list[i], true
}

func sessionID(s Suite, a, b DistKeyShare) []byte {
	h := s.Hash()
	for _, p := range a.Commitments() {
		_, _ = p.MarshalTo(h)
	}

	for _, p := range b.Commitments() {
		_, _ = p.MarshalTo(h)
	}

	return h.Sum(nil)
}
