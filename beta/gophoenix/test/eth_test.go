package test

import (
	strpkg "PhoenixOracle/gophoenix/core/store"
	"PhoenixOracle/gophoenix/core/store/models"
	"PhoenixOracle/gophoenix/core/utils"
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEthCreateTx(t *testing.T) {
	t.Parallel()
	app := NewApplicationWithKeyStore()
	store := app.Store
	defer app.Stop()
	manager := store.Eth

	to := NewEthAddress()
	data, err := hex.DecodeString("0000abcdef")
	assert.Nil(t, err)
	hash := NewTxHash()
	sentAt := uint64(23456)
	nonce := uint64(256)
	ethMock := app.MockEthClient()
	ethMock.Register("eth_getTransactionCount", utils.Uint64ToHex(nonce)) // 256
	ethMock.Register("eth_sendRawTransaction", hash)
	ethMock.Register("eth_blockNumber", utils.Uint64ToHex(sentAt))

	a, err := manager.CreateTx(to, data)
	assert.Nil(t, err)
	tx := models.Tx{}
	assert.Nil(t, store.One("ID", a.TxID, &tx))
	assert.Nil(t, err)
	assert.Equal(t, nonce, tx.Nonce)
	assert.Equal(t, data, tx.Data)
	assert.Equal(t, to, tx.To)

	assert.Nil(t, store.One("From", tx.From, &tx))
	assert.Equal(t, nonce, tx.Nonce)
	attempts, err := store.AttemptsFor(tx.ID)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(attempts))

	assert.True(t, ethMock.AllCalled())
}

func TestEthEnsureTxConfirmedBeforeThreshold(t *testing.T) {
	t.Parallel()
	app := NewApplicationWithKeyStore()
	store := app.Store
	defer app.Stop()
	config := store.Config
	eth := store.Eth
	sentAt := uint64(23456)
	from := store.KeyStore.GetAccount().Address

	ethMock := app.MockEthClient()
	ethMock.Register("eth_getTransactionReceipt", strpkg.TxReceipt{})
	ethMock.Register("eth_blockNumber", utils.Uint64ToHex(sentAt+config.EthGasBumpThreshold-1))

	txr := CreateTxAndAttempt(store,from, sentAt)
	attempts, err := store.AttemptsFor(txr.ID)
	assert.Nil(t, err)
	a := attempts[0]

	confirmed, err := eth.EnsureTxConfirmed(a.Hash)
	assert.Nil(t, err)
	assert.False(t, confirmed)
	assert.Nil(t, store.One("ID", txr.ID, txr))
	attempts, err = store.AttemptsFor(txr.ID)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(attempts))

	assert.True(t, ethMock.AllCalled())
}
func TestEthEnsureTxConfirmedAtThreshold(t *testing.T) {
	t.Parallel()
	app := NewApplicationWithKeyStore()
	store := app.Store
	defer app.Stop()
	config := store.Config
	eth := store.Eth

	sentAt := uint64(23456)
	from := store.KeyStore.GetAccount().Address

	ethMock := app.MockEthClient()
	ethMock.Register("eth_getTransactionReceipt", strpkg.TxReceipt{})
	ethMock.Register("eth_blockNumber", utils.Uint64ToHex(sentAt+config.EthGasBumpThreshold))
	ethMock.Register("eth_sendRawTransaction", NewTxHash())

	txr := CreateTxAndAttempt(store, from, sentAt)
	attempts, err := store.AttemptsFor(txr.ID)
	assert.Nil(t, err)
	a := attempts[0]

	confirmed, err := eth.EnsureTxConfirmed(a.Hash)
	assert.Nil(t, err)
	assert.False(t, confirmed)
	assert.Nil(t, store.One("ID", txr.ID, txr))
	attempts, err = store.AttemptsFor(txr.ID)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(attempts))

	assert.True(t, ethMock.AllCalled())
}

func TestEthEnsureTxConfirmedWhenSafe(t *testing.T) {
	t.Parallel()
	app := NewApplicationWithKeyStore()
	store := app.Store
	defer app.Stop()
	config := store.Config
	eth := store.Eth

	sentAt := uint64(23456)
	from := store.KeyStore.GetAccount().Address

	ethMock := app.MockEthClient()
	ethMock.Register("eth_getTransactionReceipt", strpkg.TxReceipt{
		Hash:        NewTxHash(),
		BlockNumber: sentAt,
	})
	ethMock.Register("eth_blockNumber", utils.Uint64ToHex(sentAt+config.EthMinConfirmations))

	txr := CreateTxAndAttempt(store, from, sentAt)
	a := models.TxAttempt{}
	assert.Nil(t, store.One("TxID", txr.ID, &a))

	confirmed, err := eth.EnsureTxConfirmed(a.Hash)
	assert.Nil(t, err)
	assert.True(t, confirmed)
	assert.Nil(t, store.One("ID", txr.ID, txr))
	attempts, err := store.AttemptsFor(txr.ID)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(attempts))

	assert.True(t, ethMock.AllCalled())
}

func TestEthEnsureTxConfirmedWhenWithConfsButNotSafe(t *testing.T) {
	t.Parallel()
	app := NewApplicationWithKeyStore()
	store := app.Store
	defer app.Stop()
	config := store.Config
	eth := store.Eth

	sentAt := uint64(23456)
	from := store.KeyStore.GetAccount().Address

	ethMock := app.MockEthClient()
	ethMock.Register("eth_getTransactionReceipt", strpkg.TxReceipt{
		Hash:        NewTxHash(),
		BlockNumber: sentAt,
	})
	ethMock.Register("eth_blockNumber", utils.Uint64ToHex(sentAt+config.EthMinConfirmations-1))

	txr := CreateTxAndAttempt(store, from, sentAt)
	a := models.TxAttempt{}
	assert.Nil(t, store.One("TxID", txr.ID, &a))

	confirmed, err := eth.EnsureTxConfirmed(a.Hash)
	assert.Nil(t, err)
	assert.False(t, confirmed)
	assert.Nil(t, store.One("ID", txr.ID, txr))
	attempts, err := store.AttemptsFor(txr.ID)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(attempts))

	assert.True(t, ethMock.AllCalled())
}
