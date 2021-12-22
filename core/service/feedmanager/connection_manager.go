package feedmanager

import (
	"context"
	"crypto/ed25519"
	"sync"

	pb "PhoenixOracle/core/service/feedmanager/proto"
	"PhoenixOracle/lib/gracefulpanic"
	"PhoenixOracle/lib/logger"
	"github.com/pkg/errors"
	"github.com/smartcontractkit/wsrpc"
)

type ConnectionsManager interface {
	Connect(opts ConnectOpts)
	Disconnect(id int64) error
	Close()
	GetClient(id int64) (pb.FeedsManagerClient, error)
	IsConnected(id int64) bool
}

type connectionsManager struct {
	mu       sync.Mutex
	wgClosed sync.WaitGroup

	connections map[int64]*connection
}

type connection struct {
	ctx    context.Context
	cancel context.CancelFunc

	connected bool
	client    pb.FeedsManagerClient
}

func newConnectionsManager() *connectionsManager {
	return &connectionsManager{
		mu:          sync.Mutex{},
		connections: map[int64]*connection{},
	}
}

type ConnectOpts struct {
	FeedsManagerID int64

	URI string

	Privkey []byte

	Pubkey []byte

	Handlers pb.NodeServiceServer

	OnConnect func(pb.FeedsManagerClient)
}


func (mgr *connectionsManager) Connect(opts ConnectOpts) {
	ctx, cancel := context.WithCancel(context.Background())

	conn := &connection{
		ctx:       ctx,
		cancel:    cancel,
		connected: false,
	}

	mgr.wgClosed.Add(1)

	mgr.mu.Lock()
	mgr.connections[opts.FeedsManagerID] = conn
	mgr.mu.Unlock()

	go gracefulpanic.WrapRecover(func() {
		defer mgr.wgClosed.Done()

		logger.Infow("[Feeds] Connecting to Feeds Manager...", "feedsManagerID", opts.FeedsManagerID)

		clientConn, err := wsrpc.DialWithContext(conn.ctx, opts.URI,
			wsrpc.WithTransportCreds(opts.Privkey, ed25519.PublicKey(opts.Pubkey)),
			wsrpc.WithBlock(),
		)
		if err != nil {
			// We only want to log if there was an error that did not occur
			// from a context cancel.
			if conn.ctx.Err() == nil {
				logger.Infof("Error connecting to Feeds Manager server: %v", err)
			} else {
				logger.Infof("Closing wsrpc websocket connection: %v", err)
			}

			return
		}
		defer clientConn.Close()

		logger.Infow("[Feeds] Connected to Feeds Manager", "feedsManagerID", opts.FeedsManagerID)

		// Initialize a new wsrpc client
		mgr.mu.Lock()
		conn.connected = true
		conn.client = pb.NewFeedsManagerClient(clientConn)
		mgr.connections[opts.FeedsManagerID] = conn
		mgr.mu.Unlock()

		// Initialize RPC call handlers
		pb.RegisterNodeServiceServer(clientConn, opts.Handlers)

		if opts.OnConnect != nil {
			opts.OnConnect(conn.client)
		}

		// Wait close
		<-conn.ctx.Done()
	})
}

func (mgr *connectionsManager) Disconnect(id int64) error {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	conn, ok := mgr.connections[id]
	if !ok {
		return errors.New("feeds manager is not connected")
	}

	conn.cancel()
	delete(mgr.connections, id)

	logger.Infow("[Feeds] Disconnected Feeds Manager", "feedsManagerID", id)

	return nil
}

func (mgr *connectionsManager) Close() {
	mgr.mu.Lock()
	for _, conn := range mgr.connections {
		conn.cancel()
	}

	mgr.mu.Unlock()

	mgr.wgClosed.Wait()
}

func (mgr *connectionsManager) GetClient(id int64) (pb.FeedsManagerClient, error) {
	mgr.mu.Lock()
	conn, ok := mgr.connections[id]
	mgr.mu.Unlock()
	if !ok || !conn.connected {
		return nil, errors.New("feeds manager is not connected")
	}

	return conn.client, nil
}

func (mgr *connectionsManager) IsConnected(id int64) bool {
	mgr.mu.Lock()
	conn, ok := mgr.connections[id]
	mgr.mu.Unlock()
	if !ok {
		return false
	}

	return conn.connected
}
