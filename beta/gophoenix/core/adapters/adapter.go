package adapters

import (
	"PhoenixOracle/gophoenix/core/store"
	"PhoenixOracle/gophoenix/core/store/models"
	"encoding/json"
	"fmt"
)

type Adapter interface {
	Perform(models.RunResult, *store.Store) models.RunResult
}

func For(task models.Task) (ac Adapter, err error) {
	switch task.Type {
	case "HttpGet":
		ac = &HttpGet{}
		err = json.Unmarshal(task.Params, ac)
	case "JsonParse":
		ac = &JsonParse{}
		err = json.Unmarshal(task.Params, ac)
	case "EthBytes32":
		ac = &EthBytes32{}
		err = unmarshalOrEmpty(task.Params, ac)
	case "EthTx":
		ac = &EthTx{}
		err = unmarshalOrEmpty(task.Params, ac)
	case "NoOpPend":
		ac = &NoOpPend{}
		err = unmarshalOrEmpty(task.Params, ac)
	case "NoOp":
		ac = &NoOp{}
		err = unmarshalOrEmpty(task.Params, ac)
	default:
		return nil, fmt.Errorf("%s is not a supported adapter type", task.Type)
	}
	return ac, err

}

func unmarshalOrEmpty(params json.RawMessage, dst interface{}) error {
	if len(params) > 0 {
		return json.Unmarshal(params, dst)
	}
	return nil
}

func Validate(job models.Job) error {
	var err error
	for _, task := range job.Tasks {
		err = validateTask(task)
		if err != nil {
			break
		}
	}

	return err
}

func validateTask(task models.Task) error {
	_, err := For(task)
	return err
}
