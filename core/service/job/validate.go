package job

import (
	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

var (
	ErrNoPipelineSpec       = errors.New("pipeline spec not specified")
	ErrInvalidJobType       = errors.New("invalid job type")
	ErrInvalidSchemaVersion = errors.New("invalid schema version")
	jobTypes                = map[Type]struct{}{
		Cron:              {},
		DirectRequest:     {},
		FluxMonitor:       {},
		OffchainReporting: {},
		Keeper:            {},
		VRF:               {},
		Webhook:           {},
	}
)

func ValidateSpec(ts string) (Type, error) {
	var jb Job

	tree, err := toml.Load(ts)
	if err != nil {
		return "", err
	}
	err = tree.Unmarshal(&jb)
	if err != nil {
		return "", err
	}
	if _, ok := jobTypes[jb.Type]; !ok {
		return "", ErrInvalidJobType
	}
	if jb.SchemaVersion != 1 {
		return "", ErrInvalidSchemaVersion
	}
	if jb.Type.RequiresPipelineSpec() && (jb.Pipeline.Source == "") {
		return "", ErrNoPipelineSpec
	}
	if jb.Pipeline.RequiresPreInsert() && !jb.Type.SupportsAsync() {
		return "", errors.Errorf("async=true tasks are not supported for %v", jb.Type)
	}
	return jb.Type, nil
}
