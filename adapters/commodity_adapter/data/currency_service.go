package data

import (
	"context"

	currency "github.com/chutommy/currencies/protos/currency"
	models "github.com/chutommy/market-info/models"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// CurrencyService allow the rpc calls. The service provides currency data.
type CurrencyService struct {
	client currency.CurrencyClient
	conn   *grpc.ClientConn
}

// NewCurrency is the CurrencyService constructor.
func NewCurrency() *CurrencyService {
	return &CurrencyService{}
}

// Init starts the server connection and enables rpc calls.
func (cs *CurrencyService) Init(target string) error {

	// get connection
	conn, err := grpc.Dial(target, grpc.WithInsecure())
	if err != nil {
		return errors.Wrap(err, "unable to dial ")
	}

	// register client
	cs.client = currency.NewCurrencyClient(conn)
	cs.conn = conn
	return nil
}

// Close cancels the connection between the client and the server.
func (cs *CurrencyService) Close() error {
	return cs.conn.Close()
}

// GetCurrency sends the request to the currency service server
// and returns the latest currency data.
func (cs *CurrencyService) GetCurrency(name string) (*models.Currency, error) {

	// define the request
	req := &currency.GetCurrencyRequest{Name: name}

	// call the server
	resp, err := cs.client.GetCurrency(context.Background(), req)
	if err != nil {
		return nil, errors.Wrap(err, "calling the server")
	}

	// construct the Currency
	ccy := &models.Currency{
		Name:        resp.GetName(),
		Country:     resp.GetCountry(),
		Description: resp.GetDescription(),
		Change:      resp.GetChange(),
		RateUSD:     resp.GetRateUSD(),
		UpdatedAt:   resp.GetUpdatedAt(),
	}

	return ccy, nil
}

// GetRate sends the request to the currency service server
// and returns the current exchange rate of two given currencies.
func (cs *CurrencyService) GetRate(base, dest string) (*models.ExchangeRate, error) {

	// define request
	res := &currency.GetRateRequest{
		Base:        base,
		Destination: dest,
	}

	// call the server
	resp, err := cs.client.GetRate(context.Background(), res)
	if err != nil {
		return nil, errors.Wrap(err, "calling the server")
	}

	// construct the  Rate
	rte := &models.ExchangeRate{Rate: resp.GetRate()}

	return rte, nil
}
