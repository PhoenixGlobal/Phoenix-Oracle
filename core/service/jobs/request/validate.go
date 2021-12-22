package request

import (
	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"

	"PhoenixOracle/core/assets"
	"PhoenixOracle/core/keystore/keys/ethkey"
	"PhoenixOracle/core/service/job"
	"PhoenixOracle/db/models"
)

type DirectRequestToml struct {
	ContractAddress    ethkey.EIP55Address      `toml:"contractAddress"`
	Requesters         models.AddressCollection `toml:"requesters"`
	MinContractPayment *assets.Phb             `toml:"minContractPaymentPhbJuels"`
}

func ValidatedDirectRequestSpec(tomlString string) (job.Job, error) {
	var jb = job.Job{}
	tree, err := toml.Load(tomlString)
	if err != nil {
		return jb, err
	}
	err = tree.Unmarshal(&jb)
	if err != nil {
		return jb, err
	}
	var spec DirectRequestToml
	err = tree.Unmarshal(&spec)
	if err != nil {
		return jb, err
	}
	jb.DirectRequestSpec = &job.DirectRequestSpec{
		ContractAddress:    spec.ContractAddress,
		Requesters:         spec.Requesters,
		MinContractPayment: spec.MinContractPayment,
	}

	if jb.Type != job.DirectRequest {
		return jb, errors.Errorf("unsupported type %s", jb.Type)
	}
	return jb, nil
}
