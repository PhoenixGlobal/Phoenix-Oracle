package pipeline

import (
	"context"
	"crypto/rand"
	"math/big"

	"github.com/pkg/errors"
)

type AnyTask struct {
	BaseTask `mapstructure:",squash"`
}

var _ Task = (*AnyTask)(nil)

func (t *AnyTask) Type() TaskType {
	return TaskTypeAny
}

func (t *AnyTask) Run(_ context.Context, _ Vars, inputs []Result) (result Result) {
	if len(inputs) == 0 {
		return Result{Error: errors.Wrapf(ErrWrongInputCardinality, "AnyTask requires at least 1 input")}
	}

	var answers []interface{}

	for _, input := range inputs {
		if input.Error != nil {
			continue
		}

		answers = append(answers, input.Value)
	}

	if len(answers) == 0 {
		return Result{Error: errors.Wrapf(ErrBadInput, "There were zero non-errored inputs")}
	}

	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(len(answers))))
	if err != nil {
		return Result{Error: errors.Wrapf(err, "Failed to generate random number for picking input")}
	}
	i := int(nBig.Int64())
	answer := answers[i]

	return Result{Value: answer}
}
