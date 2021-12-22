package adapters

import (
	"PhoenixOracle/gophoenix/core/store"
	"PhoenixOracle/gophoenix/core/store/models"
	"PhoenixOracle/gophoenix/core/utils"
	"encoding/hex"
)

type EthTx struct {
	Address    string `json:"address"`
	FunctionID string `json:"functionID"`
}

func (self *EthTx) Perform(input models.RunResult, store *store.Store) models.RunResult {
	if !input.Pending {
		return createTxRunResult(self, input, store)
	} else {
		return ensureTxRunResult(input, store)
	}
}

func createTxRunResult(e *EthTx,
	input models.RunResult,
	store *store.Store,
	) models.RunResult {
	recipient, err := utils.StringToAddress(e.Address)
	if err != nil {
		return models.RunResultWithError(err)
	}
	data, err := hex.DecodeString(e.FunctionID + input.Value())
	if err != nil {
		return models.RunResultWithError(err)
	}

	attempt, err := store.Eth.CreateTx(recipient, data)
	if err != nil {
		return models.RunResultWithError(err)
	}

	sendResult := models.RunResultWithValue(attempt.Hash.String())
	return ensureTxRunResult(sendResult, store)
}

func ensureTxRunResult(input models.RunResult, store *store.Store) models.RunResult {
	hash, err := utils.StringToHash(input.Value())
	if err != nil {
		return models.RunResultWithError(err)
	}

	confirmed, err := store.Eth.EnsureTxConfirmed(hash)
	if err != nil {
		return models.RunResultWithError(err)
	} else if !confirmed {
		return models.RunResultPending(input)
	}
	return models.RunResultWithValue(hash.String())
}
