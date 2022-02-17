package handlers

import (
	data "github.com/chutommy/market-info/data"
	"github.com/pkg/errors"
)

// Handler handle all clients requests.
type Handler struct {
	commoditySrv *data.CommodityService
	currencySrv  *data.CurrencyService
	cryptoSrv    *data.CryptoService
}

// New is a constructor for the Handler.
func New() *Handler {
	return &Handler{}
}

// Init initilize all needed services.
func (h *Handler) Init(commodityTarget string, currencyTarget string, cryptoTarget string) error {

	// initialize commodity service
	h.commoditySrv = data.NewCommodity()
	err := h.commoditySrv.Init(commodityTarget)
	if err != nil {
		return errors.Wrap(err, "initialize commodity service")
	}

	// initialize currency service
	h.currencySrv = data.NewCurrency()
	err = h.currencySrv.Init(currencyTarget)
	if err != nil {
		return errors.Wrap(err, "initialize currency service")
	}

	// initialize crypto service
	h.cryptoSrv = data.NewCrypto()
	err = h.cryptoSrv.Init(cryptoTarget)
	if err != nil {
		return errors.Wrap(err, "initialize crypto service")
	}

	return nil
}

// Stop closes all clients connection.
func (h *Handler) Stop() error {

	// stop commodity service
	err := h.commoditySrv.Close()
	if err != nil {
		return errors.Wrap(err, "closing commodity service")
	}

	// stop currency service
	err = h.currencySrv.Close()
	if err != nil {
		return errors.Wrap(err, "closing currency service")
	}

	// stop crypto service
	err = h.cryptoSrv.Close()
	if err != nil {
		return errors.Wrap(err, "closing crypto service")
	}

	return nil
}
