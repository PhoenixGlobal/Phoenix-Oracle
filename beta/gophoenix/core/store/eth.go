package store

import (
	"PhoenixOracle/gophoenix/core/store/models"
	"PhoenixOracle/gophoenix/core/utils"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
)

type Eth struct {
	*EthClient
	Subscription *EthClient
	KeyStore *KeyStore
	Config   Config
	ORM      *models.ORM
}

func (self *Eth) CreateTx(to common.Address, data []byte) (*models.Tx, error) {
	account := self.KeyStore.GetAccount()
	nonce, err := self.GetNonce(account)
	if err != nil {
		return nil, err
	}
	txr, err := self.ORM.CreateTx(
		account.Address,
		nonce,
		to,
		data,
		big.NewInt(0),
		500000,
	)
	if err != nil {
		return nil, err
	}
	blkNum, err := self.BlockNumber()
	if err != nil {
		return nil, err
	}
	gasPrice := self.Config.EthGasPriceDefault
	_, err = self.createAttempt(txr,gasPrice,blkNum)
	if err != nil {
		return txr, err
	}

	return txr, nil
}
func (self *Eth) EnsureTxConfirmed(hash common.Hash) (bool, error) {
	blkNum, err := self.BlockNumber()
	if err != nil {
		return false, err
	}
	attempts, err := self.getAttempts(hash)
	if err != nil {
		return false, err
	}
	if len(attempts) == 0 {
		return false, fmt.Errorf("Can only ensure transactions with attempts")
	}
	txr := models.Tx{}
	if err := self.ORM.One("ID", attempts[0].TxID, &txr); err != nil {
		return false, err
	}

	for _, txat := range attempts {
		success, err := self.checkAttempt(&txr, txat, blkNum)
		if success {
			return success, err
		}
	}
	return false, nil
}

func (self *Eth) createAttempt(txr *models.Tx, gasPrice *big.Int, blkNum uint64,) (*models.TxAttempt, error) {
	signable := txr.EthTx(gasPrice)
	signable, err := self.KeyStore.SignTx(signable, self.Config.ChainID)
	if err != nil {
		return nil, err
	}
	a, err := self.ORM.AddAttempt(txr,signable,blkNum)
	if err != nil {
		return nil, err
	}
	return a, self.sendTransaction(signable)
}

func (self *Eth) sendTransaction(tx *types.Transaction) error {
	hex, err := utils.EncodeTxToHex(tx)
	if err != nil {
		return err
	}
	if _, err = self.SendRawTx(hex); err != nil {
		return err
	}
	return nil
}

func (self *Eth) getAttempts(hash common.Hash) ([]*models.TxAttempt, error) {
	attempt := &models.TxAttempt{}
	if err := self.ORM.One("Hash", hash, attempt); err != nil {
		return []*models.TxAttempt{}, err
	}
	attempts, err := self.ORM.AttemptsFor(attempt.TxID)
	if err != nil {
		return []*models.TxAttempt{}, err
	}
	return attempts, nil
}

func (self *Eth) checkAttempt(
	txr *models.Tx,
	txat *models.TxAttempt,
	blkNum uint64,
) (bool, error) {
	receipt, err := self.GetTxReceipt(txat.Hash)
	if err != nil {
		return false, err
	}

	if receipt.Unconfirmed() {
		return self.handleUnconfirmed(txr, txat, blkNum)
	}
	return self.handleConfirmed(txr, txat, receipt, blkNum)
}

func (self *Eth) handleConfirmed(
	txr *models.Tx,
	txat *models.TxAttempt,
	rcpt *TxReceipt,
	blkNum uint64,
) (bool, error) {
	safeAt := rcpt.BlockNumber + self.Config.EthMinConfirmations
	if blkNum < safeAt {
		return false, nil
	}

	if err := self.ORM.ConfirmTx(txr, txat); err != nil {
		return false, err
	}
	return true, nil
}

func (self *Eth) handleUnconfirmed(
	txr *models.Tx,
	txat *models.TxAttempt,
	blkNum uint64,
) (bool, error) {
	bumpable := txr.Hash == txat.Hash
	pastThreshold := blkNum >= txat.SentAt+self.Config.EthGasBumpThreshold
	if bumpable && pastThreshold {
		return false, self.bumpGas(txat, blkNum)
	}
	return false, nil
}

func (self *Eth) bumpGas(txat *models.TxAttempt, blkNum uint64) error {
	txr := &models.Tx{}
	if err := self.ORM.One("ID", txat.TxID, txr); err != nil {
		return err
	}
	gasPrice := new(big.Int).Add(txat.GasPrice, self.Config.EthGasBumpWei)
	_, err :=  self.createAttempt(txr, gasPrice, blkNum)
	if err != nil {
		return err
	}
	return self.ORM.Save(txat)
}