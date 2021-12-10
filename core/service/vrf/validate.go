package vrf

import (
	"bytes"

	"PhoenixOracle/core/service/pipeline"

	uuid "github.com/satori/go.uuid"

	"PhoenixOracle/core/service/job"
	"PhoenixOracle/lib/signatures/secp256k1"
	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

var (
	ErrKeyNotSet = errors.New("key not set")
)

func ValidatedVRFSpec(tomlString string) (job.Job, error) {
	var jb = job.Job{
		ExternalJobID: uuid.NewV4(),
	}

	tree, err := toml.Load(tomlString)
	if err != nil {
		return jb, errors.Wrap(err, "toml error on load")
	}

	err = tree.Unmarshal(&jb)
	if err != nil {
		return jb, errors.Wrap(err, "toml unmarshal error on spec")
	}
	if jb.Type != job.VRF {
		return jb, errors.Errorf("unsupported type %s", jb.Type)
	}

	var spec job.VRFSpec
	err = tree.Unmarshal(&spec)
	if err != nil {
		return jb, errors.Wrap(err, "toml unmarshal error on job")
	}
	var empty secp256k1.PublicKey
	if bytes.Equal(spec.PublicKey[:], empty[:]) {
		return jb, errors.Wrap(ErrKeyNotSet, "publicKey")
	}
	if spec.Confirmations == 0 {
		return jb, errors.Wrap(ErrKeyNotSet, "confirmations")
	}
	if spec.CoordinatorAddress.String() == "" {
		return jb, errors.Wrap(ErrKeyNotSet, "coordinatorAddress")
	}
	var foundVRFTask bool
	for _, t := range jb.Pipeline.Tasks {
		if t.Type() == pipeline.TaskTypeVRF || t.Type() == pipeline.TaskTypeVRFV2 {
			foundVRFTask = true
		}
	}
	if !foundVRFTask {
		return jb, errors.Wrapf(ErrKeyNotSet, "invalid pipeline, expected a vrf task")
	}

	jb.VRFSpec = &spec

	return jb, nil
}
