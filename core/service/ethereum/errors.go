package ethereum

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"PhoenixOracle/util"
	"github.com/pkg/errors"
)

type SendError struct {
	fatal bool
	err   error
}

func (s *SendError) Error() string {
	return s.err.Error()
}

func (s *SendError) Fatal() bool {
	return s != nil && s.fatal
}

func (s *SendError) CauseStr() string {
	if s.err != nil {
		return errors.Cause(s.err).Error()
	}
	return ""
}

const (
	NonceTooLow = iota
	ReplacementTransactionUnderpriced
	LimitReached
	TransactionAlreadyInMempool
	TerminallyUnderpriced
	InsufficientEth
	TooExpensive
	FeeTooLow
	FeeTooHigh
	Fatal
)

type ClientErrors = map[int]*regexp.Regexp

var parFatal = regexp.MustCompile(`^Transaction gas is too low. There is not enough gas to cover minimal cost of the transaction|^Transaction cost exceeds current gas limit. Limit:|^Invalid signature|Recipient is banned in local queue.|Supplied gas is beyond limit|Sender is banned in local queue|Code is banned in local queue|Transaction is not permitted|Transaction is too big, see chain specification for the limit|^Invalid RLP data`)
var parity = ClientErrors{
	NonceTooLow:                       regexp.MustCompile("^Transaction nonce is too low. Try incrementing the nonce."),
	ReplacementTransactionUnderpriced: regexp.MustCompile("^Transaction gas price .+is too low. There is another transaction with same nonce in the queue"),
	LimitReached:                      regexp.MustCompile("There are too many transactions in the queue. Your transaction was dropped due to limit. Try increasing the fee."),
	TransactionAlreadyInMempool:       regexp.MustCompile("Transaction with the same hash was already imported."),
	TerminallyUnderpriced:             regexp.MustCompile("^Transaction gas price is too low. It does not satisfy your node's minimal gas price"),
	InsufficientEth:                   regexp.MustCompile("^(Insufficient funds. The account you tried to send transaction from does not have enough funds.|Insufficient balance for transaction.)"),
	Fatal:                             parFatal,
}

var gethFatal = regexp.MustCompile(`(: |^)(exceeds block gas limit|invalid sender|negative value|oversized data|gas uint64 overflow|intrinsic gas too low|nonce too high)$`)
var geth = ClientErrors{
	NonceTooLow:                       regexp.MustCompile(`(: |^)nonce too low$`),
	ReplacementTransactionUnderpriced: regexp.MustCompile(`(: |^)replacement transaction underpriced$`),
	TransactionAlreadyInMempool:       regexp.MustCompile(`(: |^)(?i)(known transaction|already known)`),
	TerminallyUnderpriced:             regexp.MustCompile(`(: |^)transaction underpriced$`),
	InsufficientEth:                   regexp.MustCompile(`(: |^)(insufficient funds for transfer|insufficient funds for gas \* price \+ value|insufficient balance for transfer)$`),
	TooExpensive:                      regexp.MustCompile(`(: |^)tx fee \([0-9\.]+ ether\) exceeds the configured cap \([0-9\.]+ ether\)$`),
	Fatal:                             gethFatal,
}

var arbitrumFatal = regexp.MustCompile(`(: |^)(invalid message format|forbidden sender address|execution reverted: error code)$`)
var arbitrum = ClientErrors{
	// TODO: Arbitrum returns this in case of low or high nonce. Update this when Arbitrum fix it
	NonceTooLow: regexp.MustCompile(`(: |^)invalid transaction nonce$`),
	// TODO: Is it terminally or replacement?
	TerminallyUnderpriced: regexp.MustCompile(`(: |^)gas price too low$`),
	InsufficientEth:       regexp.MustCompile(`(: |^)not enough funds for gas`),
	Fatal:                 arbitrumFatal,
}

var optimism = ClientErrors{
	FeeTooLow:  regexp.MustCompile(`(: |^)fee too low: \d+, use at least tx.gasLimit = \d+ and tx.gasPrice = \d+$`),
	FeeTooHigh: regexp.MustCompile(`(: |^)fee too high: \d+, use less than \d+ \* [0-9\.]+$`),
}

var clients = []ClientErrors{parity, geth, arbitrum, optimism}

func (s *SendError) is(errorType int) bool {
	if s == nil || s.err == nil {
		return false
	}
	str := s.CauseStr()
	for _, client := range clients {
		if _, ok := client[errorType]; !ok {
			continue
		}
		if client[errorType].MatchString(str) {
			return true
		}
	}
	return false
}

var hexDataRegex = regexp.MustCompile(`0x\w+$`)

// IsReplacementUnderpriced indicates that a transaction already exists in the mempool with this nonce but a different gas price or payload
func (s *SendError) IsReplacementUnderpriced() bool {
	return s.is(ReplacementTransactionUnderpriced)
}

func (s *SendError) IsNonceTooLowError() bool {
	return s.is(NonceTooLow)
}

func (s *SendError) IsTransactionAlreadyInMempool() bool {
	return s.is(TransactionAlreadyInMempool)
}

func (s *SendError) IsTerminallyUnderpriced() bool {
	return s.is(TerminallyUnderpriced)
}

func (s *SendError) IsTemporarilyUnderpriced() bool {
	return s.is(LimitReached)
}

func (s *SendError) IsInsufficientEth() bool {
	return s.is(InsufficientEth)
}

func (s *SendError) IsTooExpensive() bool {
	return s.is(TooExpensive)
}

func (s *SendError) IsFeeTooLow() bool {
	return s.is(FeeTooLow)
}

func (s *SendError) IsFeeTooHigh() bool {
	return s.is(FeeTooHigh)
}

func NewFatalSendErrorS(s string) *SendError {
	return &SendError{err: errors.New(s), fatal: true}
}

func NewFatalSendError(e error) *SendError {
	if e == nil {
		return nil
	}
	return &SendError{err: errors.WithStack(e), fatal: true}
}

func NewSendErrorS(s string) *SendError {
	return NewSendError(errors.New(s))
}

func NewSendError(e error) *SendError {
	if e == nil {
		return nil
	}
	fatal := isFatalSendError(e)
	return &SendError{err: errors.WithStack(e), fatal: fatal}
}

func isFatalSendError(err error) bool {
	if err == nil {
		return false
	}
	str := errors.Cause(err).Error()
	for _, client := range clients {
		if _, ok := client[Fatal]; !ok {
			continue
		}
		if client[Fatal].MatchString(str) {
			return true
		}
	}
	return false
}

// go-ethereum@v1.10.0/rpc/json.go
type JsonError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (err *JsonError) Error() string {
	if err.Message == "" {
		return fmt.Sprintf("json-rpc error %d", err.Code)
	}
	return err.Message
}

func ExtractRevertReasonFromRPCError(err error) (string, error) {
	if err == nil {
		return "", errors.New("no error present")
	}
	cause := errors.Cause(err)
	jsonBytes, err := json.Marshal(cause)
	if err != nil {
		return "", errors.Wrap(err, "unable to marshal err to json")
	}
	jErr := JsonError{}
	err = json.Unmarshal(jsonBytes, &jErr)
	if err != nil {
		return "", errors.Wrap(err, "unable to unmarshal json into jsonError struct")
	}
	dataStr, ok := jErr.Data.(string)
	if !ok {
		return "", errors.New("invalid error type")
	}
	matches := hexDataRegex.FindStringSubmatch(dataStr)
	if len(matches) != 1 {
		return "", errors.New("unknown data payload format")
	}
	hexData := utils.RemoveHexPrefix(matches[0])
	if len(hexData) < 8 {
		return "", errors.New("unknown data payload format")
	}
	revertReasonBytes, err := hex.DecodeString(hexData[8:])
	if err != nil {
		return "", errors.Wrap(err, "unable to decode hex to bytes")
	}

	ln := len(revertReasonBytes)
	breaker := time.After(time.Second * 5)
cleanup:
	for {
		select {
		case <-breaker:
			break cleanup
		default:
			revertReasonBytes = bytes.Trim(revertReasonBytes, "\x00")
			revertReasonBytes = bytes.Trim(revertReasonBytes, "\x11")
			revertReasonBytes = bytes.TrimSpace(revertReasonBytes)
			if ln == len(revertReasonBytes) {
				break cleanup
			}
			ln = len(revertReasonBytes)
		}
	}

	revertReason := strings.TrimSpace(string(revertReasonBytes))
	return revertReason, nil
}
