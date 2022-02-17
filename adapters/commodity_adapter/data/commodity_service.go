package data

import (
	"context"

	commodity "github.com/chutommy/commodity-prices/protos/commodity"
	models "github.com/chutommy/market-info/models"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// CommodityService defines the service's client and allows
// the rpc calls. The service provides commodity data.
type CommodityService struct {
	client commodity.CommodityClient
	conn   *grpc.ClientConn
}

// NewCommodity is the CommodityService constructor.
func NewCommodity() *CommodityService {
	return &CommodityService{}
}

// Init starts the server connection and enables rpc calls.
func (cs *CommodityService) Init(target string) error {

	// get connection
	conn, err := grpc.Dial(target, grpc.WithInsecure())
	if err != nil {
		return errors.Wrap(err, "unable to dial commodity service")
	}

	// register client
	cs.client = commodity.NewCommodityClient(conn)
	cs.conn = conn
	return nil
}

// Close cancels the connection between the client and the server.
func (cs *CommodityService) Close() error {
	return cs.conn.Close()
}

// GetCommodity sends the request to the commodity service server
// and returns the latest commodity data.
func (cs *CommodityService) GetCommodity(name string) (*models.Commodity, error) {

	// define the request
	req := &commodity.CommodityRequest{Name: name}

	// call the server
	resp, err := cs.client.GetCommodity(context.Background(), req)
	if err != nil {
		return nil, errors.Wrap(err, "calling the server")
	}

	// construct the Commodity
	cmd := &models.Commodity{
		Name:       resp.GetName(),
		Price:      resp.GetPrice(),
		Currency:   resp.GetCurrency(),
		WeightUnit: resp.GetWeightUnit(),
		ChangeP:    resp.GetChangeP(),
		ChangeN:    resp.GetChangeN(),
		LastUpdate: resp.GetLastUpdate(),
	}

	return cmd, nil
}
