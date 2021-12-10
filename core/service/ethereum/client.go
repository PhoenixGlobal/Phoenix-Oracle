package ethereum

import (
	"context"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"PhoenixOracle/core/assets"
	"PhoenixOracle/db/models"
	"PhoenixOracle/lib/logger"
	"PhoenixOracle/util"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
)

type Client interface {
	Dial(ctx context.Context) error
	Close()

	GetERC20Balance(address common.Address, contractAddress common.Address) (*big.Int, error)
	GetPHBBalance(phbAddress common.Address, address common.Address) (*assets.Phb, error)
	GetEthBalance(ctx context.Context, account common.Address, blockNumber *big.Int) (*assets.Eth, error)

	Call(result interface{}, method string, args ...interface{}) error
	CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
	BatchCallContext(ctx context.Context, b []rpc.BatchElem) error
	RoundRobinBatchCallContext(ctx context.Context, b []rpc.BatchElem) error

	HeadByNumber(ctx context.Context, n *big.Int) (*models.Head, error)
	SubscribeNewHead(ctx context.Context, ch chan<- *models.Head) (ethereum.Subscription, error)

	ChainID(ctx context.Context) (*big.Int, error)
	SendTransaction(ctx context.Context, tx *types.Transaction) error
	PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error)
	PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)
	NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
	FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error)
	SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error)
	EstimateGas(ctx context.Context, call ethereum.CallMsg) (uint64, error)
	SuggestGasPrice(ctx context.Context) (*big.Int, error)
	CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
	CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error)

	// bind.ContractBackend methods
	HeaderByNumber(context.Context, *big.Int) (*types.Header, error)
	SuggestGasTipCap(ctx context.Context) (*big.Int, error)
}

type Subscription interface {
	Err() <-chan error
	Unsubscribe()
}

func DefaultQueryCtx(ctxs ...context.Context) (ctx context.Context, cancel context.CancelFunc) {
	if len(ctxs) > 0 {
		ctx = ctxs[0]
	} else {
		ctx = context.Background()
	}
	return context.WithTimeout(ctx, 15*time.Second)
}

type client struct {
	logger      *logger.Logger
	primary     *node
	secondaries []*secondarynode
	mocked      bool

	roundRobinCount uint32
}

var _ Client = (*client)(nil)

func NewClient(logger *logger.Logger, rpcUrl string, rpcHTTPURL *url.URL, secondaryRPCURLs []url.URL) (*client, error) {
	parsed, err := url.ParseRequestURI(rpcUrl)
	if err != nil {
		return nil, err
	}

	if parsed.Scheme != "ws" && parsed.Scheme != "wss" {
		return nil, errors.Errorf("ethereum url scheme must be websocket: %s", parsed.String())
	}

	c := client{logger: logger}

	c.primary = newNode(*parsed, rpcHTTPURL, "eth-primary-0")

	for i, url := range secondaryRPCURLs {
		if url.Scheme != "http" && url.Scheme != "https" {
			return nil, errors.Errorf("secondary ethereum rpc url scheme must be http(s): %s", url.String())
		}
		s := newSecondaryNode(url, fmt.Sprintf("eth-secondary-%d", i))
		c.secondaries = append(c.secondaries, s)
	}
	return &c, nil
}

func (client *client) Dial(ctx context.Context) error {
	if client.mocked {
		return nil
	}
	if err := client.primary.Dial(ctx); err != nil {
		return errors.Wrap(err, "Failed to dial primary client")
	}

	for _, s := range client.secondaries {
		err := s.Dial()
		if err != nil {
			return errors.Wrapf(err, "Failed to dial secondary client: %v", s.uri)
		}
	}
	return nil
}

func (client *client) Close() {
	client.primary.Close()
}

type CallArgs struct {
	To   common.Address `json:"to"`
	Data hexutil.Bytes  `json:"data"`
}

func (client *client) GetERC20Balance(address common.Address, contractAddress common.Address) (*big.Int, error) {
	result := ""
	numPhbBigInt := new(big.Int)
	functionSelector := models.HexToFunctionSelector("0x70a08231") // balanceOf(address)
	data := utils.ConcatBytes(functionSelector.Bytes(), common.LeftPadBytes(address.Bytes(), utils.EVMWordByteLen))
	args := CallArgs{
		To:   contractAddress,
		Data: data,
	}
	err := client.Call(&result, "eth_call", args, "latest")
	if err != nil {
		return numPhbBigInt, err
	}
	numPhbBigInt.SetString(result, 0)
	return numPhbBigInt, nil
}

func (client *client) GetPHBBalance(phbAddress common.Address, address common.Address) (*assets.Phb, error) {
	balance, err := client.GetERC20Balance(address, phbAddress)
	if err != nil {
		return assets.NewPhb(0), err
	}
	return (*assets.Phb)(balance), nil
}

func (client *client) GetEthBalance(ctx context.Context, account common.Address, blockNumber *big.Int) (*assets.Eth, error) {
	balance, err := client.BalanceAt(ctx, account, blockNumber)
	if err != nil {
		return assets.NewEth(0), err
	}
	return (*assets.Eth)(balance), nil
}

func (client *client) TransactionReceipt(ctx context.Context, txHash common.Hash) (receipt *types.Receipt, err error) {
	receipt, err = client.primary.TransactionReceipt(ctx, txHash)

	if err != nil && strings.Contains(err.Error(), "missing required field") {
		return nil, ethereum.NotFound
	}
	return
}

func (client *client) ChainID(ctx context.Context) (*big.Int, error) {
	return client.primary.ChainID(ctx)
}

func (client *client) HeaderByNumber(ctx context.Context, n *big.Int) (*types.Header, error) {
	return client.primary.HeaderByNumber(ctx, n)
}

func (client *client) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	var wg sync.WaitGroup
	defer wg.Wait()
	for _, s := range client.secondaries {
		wg.Add(1)
		go func(s *secondarynode) {
			defer wg.Done()
			err := NewSendError(s.SendTransaction(ctx, tx))
			if err == nil || err.IsNonceTooLowError() || err.IsTransactionAlreadyInMempool() {
				// Nonce too low or transaction known errors are expected since
				// the primary SendTransaction may well have succeeded already
				return
			}
			client.logger.Warnw("secondary eth client returned error", "err", err, "tx", tx)
		}(s)
	}

	return client.primary.SendTransaction(ctx, tx)
}

func (client *client) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	return client.primary.PendingNonceAt(ctx, account)
}

func (client *client) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	return client.primary.NonceAt(ctx, account, blockNumber)
}

func (client *client) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	return client.primary.PendingCodeAt(ctx, account)
}

func (client *client) EstimateGas(ctx context.Context, call ethereum.CallMsg) (gas uint64, err error) {
	return client.primary.EstimateGas(ctx, call)
}

func (client *client) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return client.primary.SuggestGasPrice(ctx)
}

func (client *client) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	return client.primary.CallContract(ctx, msg, blockNumber)
}

func (client *client) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	return client.primary.CodeAt(ctx, account, blockNumber)
}

func (client *client) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return client.primary.BlockByNumber(ctx, number)
}

func (client *client) HeadByNumber(ctx context.Context, number *big.Int) (head *models.Head, err error) {
	hex := toBlockNumArg(number)
	err = client.primary.CallContext(ctx, &head, "eth_getBlockByNumber", hex, false)
	if err == nil && head == nil {
		err = ethereum.NotFound
	}
	return
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	return hexutil.EncodeBig(number)
}

func (client *client) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	return client.primary.BalanceAt(ctx, account, blockNumber)
}

func (client *client) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	return client.primary.FilterLogs(ctx, q)
}

func (client *client) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	client.logger.Debugw("eth.Client#SubscribeFilterLogs(...)",
		"q", q,
	)
	return client.primary.SubscribeFilterLogs(ctx, q, ch)
}

func (client *client) SubscribeNewHead(ctx context.Context, ch chan<- *models.Head) (ethereum.Subscription, error) {
	return client.primary.EthSubscribe(ctx, ch, "newHeads")
}

func (client *client) EthSubscribe(ctx context.Context, channel interface{}, args ...interface{}) (ethereum.Subscription, error) {
	return client.primary.EthSubscribe(ctx, channel, args...)
}

func (client *client) Call(result interface{}, method string, args ...interface{}) error {
	ctx, cancel := DefaultQueryCtx()
	defer cancel()
	return client.primary.CallContext(ctx, result, method, args...)
}

func (client *client) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	return client.primary.CallContext(ctx, result, method, args...)
}

func (client *client) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	return client.primary.BatchCallContext(ctx, b)
}

func (client *client) RoundRobinBatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	nSecondaries := len(client.secondaries)
	if nSecondaries == 0 {
		return client.BatchCallContext(ctx, b)
	}

	count := atomic.AddUint32(&client.roundRobinCount, 1) - 1
	rr := int(count % uint32(nSecondaries+1))

	if rr == 0 {
		return client.BatchCallContext(ctx, b)
	}
	return client.secondaries[rr-1].BatchCallContext(ctx, b)
}

func (client *client) SuggestGasTipCap(ctx context.Context) (tipCap *big.Int, err error) {
	return client.primary.SuggestGasTipCap(ctx)
}
