package pipeline

import (
	"context"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/multierr"
)

type MeanTask struct {
	BaseTask      `mapstructure:",squash"`
	Values        string `json:"values"`
	AllowedFaults string `json:"allowedFaults"`
	Precision     string `json:"precision"`
}

var _ Task = (*MeanTask)(nil)

func (t *MeanTask) Type() TaskType {
	return TaskTypeMean
}

func (t *MeanTask) Run(_ context.Context, vars Vars, inputs []Result) (result Result) {
	var (
		maybeAllowedFaults MaybeUint64Param
		maybePrecision     MaybeInt32Param
		valuesAndErrs      SliceParam
		decimalValues      DecimalSliceParam
		allowedFaults      int
		faults             int
	)
	err := multierr.Combine(
		errors.Wrap(ResolveParam(&maybeAllowedFaults, From(t.AllowedFaults)), "allowedFaults"),
		errors.Wrap(ResolveParam(&maybePrecision, From(VarExpr(t.Precision, vars), t.Precision)), "precision"),
		errors.Wrap(ResolveParam(&valuesAndErrs, From(VarExpr(t.Values, vars), JSONWithVarExprs(t.Values, vars, true), Inputs(inputs))), "values"),
	)
	if err != nil {
		return Result{Error: err}
	}

	if allowed, isSet := maybeAllowedFaults.Uint64(); isSet {
		allowedFaults = int(allowed)
	} else {
		allowedFaults = len(valuesAndErrs) - 1
	}

	values, faults := valuesAndErrs.FilterErrors()
	if faults > allowedFaults {
		return Result{Error: errors.Wrapf(ErrTooManyErrors, "Number of faulty inputs %v to mean task > number allowed faults %v", faults, allowedFaults)}
	} else if len(values) == 0 {
		return Result{Error: errors.Wrap(ErrWrongInputCardinality, "values")}
	}

	err = decimalValues.UnmarshalPipelineParam(values)
	if err != nil {
		return Result{Error: errors.Wrapf(ErrBadInput, "values: %v", err)}
	}

	total := decimal.NewFromInt(0)
	for _, val := range decimalValues {
		total = total.Add(val)
	}

	numValues := decimal.NewFromInt(int64(len(decimalValues)))

	if precision, isSet := maybePrecision.Int32(); isSet {
		return Result{Value: total.DivRound(numValues, precision)}
	}
	return Result{Value: total.Div(numValues)}
}
