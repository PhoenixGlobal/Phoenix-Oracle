package data

import (
	"context"

	crypto "github.com/chutommy/crypto-currencies/protos/crypto"
	models "github.com/chutommy/market-info/models"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// CryptoService allows the rpc calls. The service provides cryptocurrency data.
type CryptoService struct {
	client crypto.CryptoClient
	conn   *grpc.ClientConn
}

// NewCrypto is the CryptoService constructor.
func NewCrypto() *CryptoService {
	return &CryptoService{}
}

// Init starts the server connection and enables rpc calls.
func (cs *CryptoService) Init(target string) error {

	// get connection
	conn, err := grpc.Dial(target, grpc.WithInsecure())
	if err != nil {
		return errors.Wrap(err, "unable to dial crypto service")
	}

	// register client
	cs.client = crypto.NewCryptoClient(conn)
	cs.conn = conn
	return nil
}

// Close cancels the connection between the client and the server.
func (cs *CryptoService) Close() error {
	return cs.conn.Close()
}

// GetCrypto sends the request to the cryptocurrency service server
// and returns the latest cryptocurrency data.
func (cs *CryptoService) GetCrypto(name string) (*models.Crypto, error) {

	// define the request
	req := &crypto.GetCryptoRequest{Name: name}

	// call the server
	resp, err := cs.client.GetCrypto(context.Background(), req)
	if err != nil {
		return nil, errors.Wrap(err, "calling the server")
	}

	// construct the Crypto
	cpto := &models.Crypto{
		Name:              resp.GetName(),
		Symbol:            resp.GetSymbol(),
		MarketCapUSD:      resp.GetMarketCapUSD(),
		Price:             resp.GetPrice(),
		CirculatingSupply: resp.GetCirculatingSupply(),
		Mineable:          resp.GetMineable(),
		Volume:            resp.GetVolume(),
		ChangeHour:        resp.GetChangeHour(),
		ChangeDay:         resp.GetChangeDay(),
		ChangeWeek:        resp.GetChangeWeek(),
	}

	return cpto, nil
}
