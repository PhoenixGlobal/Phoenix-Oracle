package vrf

import (
	"context"
	"fmt"
	"sync"

	heaps "github.com/theodesp/go-heaps"
	"github.com/theodesp/go-heaps/pairing"

	"PhoenixOracle/lib/gracefulpanic"

	"PhoenixOracle/core/keystore"
	"PhoenixOracle/core/log"
	"PhoenixOracle/core/service/ethereum"
	"PhoenixOracle/core/service/job"
	"PhoenixOracle/core/service/pipeline"
	"PhoenixOracle/core/service/txmanager"
	"PhoenixOracle/db/models"
	"PhoenixOracle/internal/gethwrappers/generated/vrf_coordinator_v2"
	httypes "PhoenixOracle/lib/headtracker/types"
	"PhoenixOracle/lib/logger"
	"PhoenixOracle/lib/postgres"
	"PhoenixOracle/util"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const (
	GasAfterPaymentCalculation = 5000 +
		2100 +
		20000 +
		2*2100 -
		4800 +
		21000 +
		8890
)

var (
	_ log.Listener = &listenerV2{}
	_ job.Service  = &listenerV2{}
)

type pendingRequest struct {
	confirmedAtBlock uint64
	req              *vrf_coordinator_v2.VRFCoordinatorV2RandomWordsRequested
	lb               log.Broadcast
}

type listenerV2 struct {
	utils.StartStopOnce
	cfg             Config
	l               logger.Logger
	abi             abi.ABI
	ethClient       ethereum.Client
	logBroadcaster  log.Broadcaster
	txm             txmanager.TxManager
	headBroadcaster httypes.HeadBroadcasterRegistry
	coordinator     *vrf_coordinator_v2.VRFCoordinatorV2
	pipelineRunner  pipeline.Runner
	pipelineORM     pipeline.ORM
	vorm            keystore.VRFORM
	job             job.Job
	db              *gorm.DB
	vrfks           keystore.VRF
	gethks          keystore.Eth
	reqLogs         *utils.Mailbox
	chStop          chan struct{}
	waitOnStop      chan struct{}
	newHead         chan struct{}
	latestHead      uint64
	latestHeadMu    sync.RWMutex

	reqsMu   sync.Mutex
	reqs     []pendingRequest
	reqAdded func()

	respCountMu sync.Mutex
	respCount   map[string]uint64
	blockNumberToReqID *pairing.PairHeap
}

func (lsn *listenerV2) Start() error {
	return lsn.StartOnce("VRFListenerV2", func() error {
		minConfs := lsn.cfg.MinIncomingConfirmations()
		if lsn.job.VRFSpec.Confirmations > lsn.cfg.MinIncomingConfirmations() {
			minConfs = lsn.job.VRFSpec.Confirmations
		}
		unsubscribeLogs := lsn.logBroadcaster.Register(lsn, log.ListenerOpts{
			Contract: lsn.coordinator.Address(),
			ParseLog: lsn.coordinator.ParseLog,
			LogsWithTopics: map[common.Hash][][]log.Topic{
				vrf_coordinator_v2.VRFCoordinatorV2RandomWordsRequested{}.Topic(): {
					{
						log.Topic(lsn.job.VRFSpec.PublicKey.MustHash()),
					},
				},
			},
		})

		latestHead, unsubscribeHeadBroadcaster := lsn.headBroadcaster.Subscribe(lsn)
		if latestHead != nil {
			lsn.setLatestHead(*latestHead)
		}

		go gracefulpanic.WrapRecover(func() {
			lsn.runLogListener([]func(){unsubscribeLogs}, minConfs)
		})
		go gracefulpanic.WrapRecover(func() {
			lsn.runHeadListener(unsubscribeHeadBroadcaster)
		})
		return nil
	})
}

func (lsn *listenerV2) Connect(head *models.Head) error {
	lsn.latestHead = uint64(head.Number)
	return nil
}

func (lsn *listenerV2) extractConfirmedLogs() []pendingRequest {
	lsn.reqsMu.Lock()
	defer lsn.reqsMu.Unlock()
	var toProcess, toKeep []pendingRequest
	for i := 0; i < len(lsn.reqs); i++ {
		if lsn.reqs[i].confirmedAtBlock <= lsn.getLatestHead() {
			toProcess = append(toProcess, lsn.reqs[i])
		} else {
			toKeep = append(toKeep, lsn.reqs[i])
		}
	}
	lsn.reqs = toKeep
	return toProcess
}

func (lsn *listenerV2) OnNewLongestChain(_ context.Context, head models.Head) {
	lsn.setLatestHead(head)
	select {
	case lsn.newHead <- struct{}{}:
	default:
	}
}

func (lsn *listenerV2) setLatestHead(h models.Head) {
	lsn.latestHeadMu.Lock()
	defer lsn.latestHeadMu.Unlock()
	num := uint64(h.Number)
	if num > lsn.latestHead {
		lsn.latestHead = num
	}
}

func (lsn *listenerV2) getLatestHead() uint64 {
	lsn.latestHeadMu.RLock()
	defer lsn.latestHeadMu.RUnlock()
	return lsn.latestHead
}

type fulfilledReqV2 struct {
	blockNumber uint64
	reqID       string
}

func (a fulfilledReqV2) Compare(b heaps.Item) int {
	a1 := a
	a2 := b.(fulfilledReqV2)
	switch {
	case a1.blockNumber > a2.blockNumber:
		return 1
	case a1.blockNumber < a2.blockNumber:
		return -1
	default:
		return 0
	}
}

func (lsn *listenerV2) pruneConfirmedRequestCounts() {
	lsn.respCountMu.Lock()
	defer lsn.respCountMu.Unlock()
	min := lsn.blockNumberToReqID.FindMin()
	for min != nil {
		m := min.(fulfilledReqV2)
		if m.blockNumber > (lsn.getLatestHead() - 10000) {
			break
		}
		delete(lsn.respCount, m.reqID)
		lsn.blockNumberToReqID.DeleteMin()
		min = lsn.blockNumberToReqID.FindMin()
	}
}

func (lsn *listenerV2) runHeadListener(unsubscribe func()) {
	for {
		select {
		case <-lsn.chStop:
			unsubscribe()
			lsn.waitOnStop <- struct{}{}
			return
		case <-lsn.newHead:
			toProcess := lsn.extractConfirmedLogs()
			for _, r := range toProcess {
				lsn.ProcessV2VRFRequest(r.req, r.lb)
			}
			lsn.pruneConfirmedRequestCounts()
		}
	}
}

func (lsn *listenerV2) runLogListener(unsubscribes []func(), minConfs uint32) {
	lsn.l.Infow("VRFListenerV2: listening for run requests",
		"minConfs", minConfs)
	for {
		select {
		case <-lsn.chStop:
			for _, f := range unsubscribes {
				f()
			}
			lsn.waitOnStop <- struct{}{}
			return
		case <-lsn.reqLogs.Notify():
			// Process all the logs in the queue if one is added
			for {
				i, exists := lsn.reqLogs.Retrieve()
				if !exists {
					break
				}
				lb, ok := i.(log.Broadcast)
				if !ok {
					panic(fmt.Sprintf("VRFListenerV2: invariant violated, expected log.Broadcast got %T", i))
				}
				lsn.handleLog(lb, minConfs)
			}
		}
	}
}

func (lsn *listenerV2) shouldProcessLog(lb log.Broadcast) bool {
	ctx, cancel := postgres.DefaultQueryCtx()
	defer cancel()
	consumed, err := lsn.logBroadcaster.WasAlreadyConsumed(lsn.db.WithContext(ctx), lb)
	if err != nil {
		lsn.l.Errorw("VRFListenerV2: could not determine if log was already consumed", "error", err, "txHash", lb.RawLog().TxHash)
		// Do not process, let lb resend it as a retry mechanism.
		return false
	}
	return !consumed
}

func (lsn *listenerV2) getConfirmedAt(req *vrf_coordinator_v2.VRFCoordinatorV2RandomWordsRequested, minConfs uint32) uint64 {
	lsn.respCountMu.Lock()
	defer lsn.respCountMu.Unlock()
	newConfs := uint64(minConfs) * (1 << lsn.respCount[req.RequestId.String()])
	if newConfs > 200 {
		newConfs = 200
	}
	if lsn.respCount[req.RequestId.String()] > 0 {
		lsn.l.Warnw("VRFListenerV2: duplicate request found after fulfillment, doubling incoming confirmations",
			"txHash", req.Raw.TxHash,
			"blockNumber", req.Raw.BlockNumber,
			"blockHash", req.Raw.BlockHash,
			"reqID", req.RequestId.String(),
			"newConfs", newConfs)
	}
	return req.Raw.BlockNumber + uint64(minConfs)*(1<<lsn.respCount[req.RequestId.String()])
}

func (lsn *listenerV2) handleLog(lb log.Broadcast, minConfs uint32) {
	if v, ok := lb.DecodedLog().(*vrf_coordinator_v2.VRFCoordinatorV2RandomWordsFulfilled); ok {
		if !lsn.shouldProcessLog(lb) {
			return
		}
		lsn.respCountMu.Lock()
		lsn.respCount[v.RequestId.String()]++
		lsn.respCountMu.Unlock()
		lsn.blockNumberToReqID.Insert(fulfilledReqV2{
			blockNumber: v.Raw.BlockNumber,
			reqID:       v.RequestId.String(),
		})
		lsn.markLogAsConsumed(lb)
		return
	}

	req, err := lsn.coordinator.ParseRandomWordsRequested(lb.RawLog())
	if err != nil {
		lsn.l.Errorw("VRFListenerV2: failed to parse log", "err", err, "txHash", lb.RawLog().TxHash)
		if !lsn.shouldProcessLog(lb) {
			return
		}
		lsn.markLogAsConsumed(lb)
		return
	}

	confirmedAt := lsn.getConfirmedAt(req, minConfs)
	lsn.reqsMu.Lock()
	lsn.reqs = append(lsn.reqs, pendingRequest{
		confirmedAtBlock: confirmedAt,
		req:              req,
		lb:               lb,
	})
	lsn.reqAdded()
	lsn.reqsMu.Unlock()
}

func (lsn *listenerV2) markLogAsConsumed(lb log.Broadcast) {
	ctx, cancel := postgres.DefaultQueryCtx()
	defer cancel()
	err := lsn.logBroadcaster.MarkConsumed(lsn.db.WithContext(ctx), lb)
	lsn.l.ErrorIf(errors.Wrapf(err, "VRFListenerV2: unable to mark log %v as consumed", lb.String()))
}

func (lsn *listenerV2) ProcessV2VRFRequest(req *vrf_coordinator_v2.VRFCoordinatorV2RandomWordsRequested, lb log.Broadcast) {
	// Check if the vrf req has already been fulfilled
	callback, err := lsn.coordinator.GetCommitment(nil, req.RequestId)
	if err != nil {
		lsn.l.Errorw("VRFListenerV2: unable to check if already fulfilled, processing anyways", "err", err, "txHash", req.Raw.TxHash)
	} else if utils.IsEmpty(callback[:]) {
		// If seedAndBlockNumber is zero then the response has been fulfilled
		// and we should skip it
		lsn.l.Infow("VRFListenerV2: request already fulfilled", "txHash", req.Raw.TxHash, "subID", req.SubId, "callback", callback)
		lsn.markLogAsConsumed(lb)
		return
	}

	lsn.l.Infow("VRFListenerV2: received log request",
		"log", lb.String(),
		"reqID", req.RequestId.String(),
		"txHash", req.Raw.TxHash,
		"blockNumber", req.Raw.BlockNumber,
		"blockHash", req.Raw.BlockHash,
		"seed", req.PreSeed)

	vars := pipeline.NewVarsFrom(map[string]interface{}{
		"jobSpec": map[string]interface{}{
			"databaseID":    lsn.job.ID,
			"externalJobID": lsn.job.ExternalJobID,
			"name":          lsn.job.Name.ValueOrZero(),
			"publicKey":     lsn.job.VRFSpec.PublicKey[:],
		},
		"jobRun": map[string]interface{}{
			"logBlockHash":   req.Raw.BlockHash[:],
			"logBlockNumber": req.Raw.BlockNumber,
			"logTxHash":      req.Raw.TxHash,
			"logTopics":      req.Raw.Topics,
			"logData":        req.Raw.Data,
		},
	})
	run := pipeline.NewRun(*lsn.job.PipelineSpec, vars)
	if _, err = lsn.pipelineRunner.Run(context.Background(), &run, lsn.l, true, func(tx *gorm.DB) error {
		// Always mark consumed regardless of whether the proof failed or not.
		if err = lsn.logBroadcaster.MarkConsumed(tx, lb); err != nil {
			logger.Errorw("VRFListenerV2: failed mark consumed", "err", err)
		}
		return nil
	}); err != nil {
		logger.Errorw("VRFListenerV2: failed executing run", "err", err)
	}
}

// Close complies with job.Service
func (lsn *listenerV2) Close() error {
	return lsn.StopOnce("VRFListenerV2", func() error {
		close(lsn.chStop)
		<-lsn.waitOnStop
		return nil
	})
}

func (lsn *listenerV2) HandleLog(lb log.Broadcast) {
	wasOverCapacity := lsn.reqLogs.Deliver(lb)
	if wasOverCapacity {
		logger.Error("VRFListenerV2: log mailbox is over capacity - dropped the oldest log")
	}
}

// Job complies with log.Listener
func (lsn *listenerV2) JobID() int32 {
	return lsn.job.ID
}
