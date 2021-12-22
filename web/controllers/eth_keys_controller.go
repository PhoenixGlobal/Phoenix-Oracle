package controllers

import (
	"PhoenixOracle/web"
	"context"
	"io/ioutil"
	"net/http"
	"strconv"

	"PhoenixOracle/core/assets"
	"PhoenixOracle/core/keystore/keys/ethkey"
	"PhoenixOracle/core/service/phoenix"
	"PhoenixOracle/lib/logger"
	"PhoenixOracle/web/presenters"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type ETHKeysController struct {
	App phoenix.Application
}

func (ekc *ETHKeysController) Index(c *gin.Context) {
	ethKeyStore := ekc.App.GetKeyStore().Eth()
	var keys []ethkey.KeyV2
	var err error
	if ekc.App.GetStore().Config.Dev() {
		keys, err = ethKeyStore.GetAll()
	} else {
		keys, err = ethKeyStore.SendingKeys()
	}
	if err != nil {
		err = errors.Errorf("error getting unlocked keys: %v", err)
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}
	states, err := ethKeyStore.GetStatesForKeys(keys)
	if err != nil {
		err = errors.Errorf("error getting key states: %v", err)
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}
	var resources []presenters.ETHKeyResource
	for _, state := range states {
		key, err := ethKeyStore.Get(state.Address.Hex())
		if err != nil {
			web.JsonAPIError(c, http.StatusInternalServerError, err)
			return
		}
		r, err := presenters.NewETHKeyResource(key, state,
			ekc.setEthBalance(c.Request.Context(), key.Address.Address()),
			ekc.setPhbBalance(key.Address.Address()),
		)
		if err != nil {
			web.JsonAPIError(c, http.StatusInternalServerError, err)
			return
		}

		resources = append(resources, *r)
	}

	web.JsonAPIResponse(c, resources, "keys")
}

func (ekc *ETHKeysController) Create(c *gin.Context) {
	ethKeyStore := ekc.App.GetKeyStore().Eth()
	key, err := ethKeyStore.Create()
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}
	state, err := ethKeyStore.GetState(key.ID())
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}
	r, err := presenters.NewETHKeyResource(key, state,
		ekc.setEthBalance(c.Request.Context(), key.Address.Address()),
		ekc.setPhbBalance(key.Address.Address()),
	)
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponseWithStatus(c, r, "account", http.StatusCreated)
}

func (ekc *ETHKeysController) Delete(c *gin.Context) {
	ethKeyStore := ekc.App.GetKeyStore().Eth()
	var hardDelete bool
	var err error

	if c.Query("hard") != "" {
		hardDelete, err = strconv.ParseBool(c.Query("hard"))
		if err != nil {
			web.JsonAPIError(c, http.StatusUnprocessableEntity, err)
			return
		}
	}

	if !hardDelete {
		web.JsonAPIError(c, http.StatusUnprocessableEntity, errors.New("hard delete only"))
		return
	}

	if !common.IsHexAddress(c.Param("keyID")) {
		web.JsonAPIError(c, http.StatusBadRequest, errors.New("hard delete only"))
		return
	}
	keyID := c.Param("keyID")
	state, err := ethKeyStore.GetState(keyID)
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	key, err := ethKeyStore.Delete(keyID)
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	r, err := presenters.NewETHKeyResource(key, state,
		ekc.setEthBalance(c.Request.Context(), key.Address.Address()),
		ekc.setPhbBalance(key.Address.Address()),
	)
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponse(c, r, "account")
}

func (ekc *ETHKeysController) Import(c *gin.Context) {
	ethKeyStore := ekc.App.GetKeyStore().Eth()
	defer logger.ErrorIfCalling(c.Request.Body.Close)

	bytes, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		web.JsonAPIError(c, http.StatusBadRequest, err)
		return
	}
	oldPassword := c.Query("oldpassword")

	key, err := ethKeyStore.Import(bytes, oldPassword)
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	state, err := ethKeyStore.GetState(key.ID())
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	r, err := presenters.NewETHKeyResource(key, state,
		ekc.setEthBalance(c.Request.Context(), key.Address.Address()),
		ekc.setPhbBalance(key.Address.Address()),
	)
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponse(c, r, "account")
}

func (ekc *ETHKeysController) Export(c *gin.Context) {
	defer logger.ErrorIfCalling(c.Request.Body.Close)

	address := c.Param("address")
	newPassword := c.Query("newpassword")

	bytes, err := ekc.App.GetKeyStore().Eth().Export(address, newPassword)
	if err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}
	c.Data(http.StatusOK, web.MediaType, bytes)
}

func (ekc *ETHKeysController) setEthBalance(ctx context.Context, accountAddr common.Address) presenters.NewETHKeyOption {
	ethClient := ekc.App.GetEthClient()
	bal, err := ethClient.BalanceAt(ctx, accountAddr, nil)

	return func(r *presenters.ETHKeyResource) error {
		if err != nil {
			return errors.Errorf("error calling getEthBalance on Ethereum node: %v", err)
		}

		r.EthBalance = (*assets.Eth)(bal)

		return nil
	}
}

func (ekc *ETHKeysController) setPhbBalance(accountAddr common.Address) presenters.NewETHKeyOption {
	ethClient := ekc.App.GetEthClient()
	addr := common.HexToAddress(ekc.App.GetEVMConfig().PhbContractAddress())
	bal, err := ethClient.GetPHBBalance(addr, accountAddr)

	return func(r *presenters.ETHKeyResource) error {
		if err != nil {
			return errors.Errorf("error calling getLINKBalance on Ethereum node: %v", err)
		}

		r.PhbBalance = bal

		return nil
	}
}
