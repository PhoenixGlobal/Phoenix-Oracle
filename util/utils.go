package utils

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	mrand "math/rand"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"PhoenixOracle/lib/logger"
	"go.uber.org/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/jpillora/backoff"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	uuid "github.com/satori/go.uuid"
	"github.com/shopspring/decimal"
	"github.com/tevino/abool"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/sha3"
	null "gopkg.in/guregu/null.v4"
)

const (
	DefaultSecretSize = 48
	EVMWordByteLen = 32
	EVMWordHexLen = EVMWordByteLen * 2
)

// ZeroAddress is an address of all zeroes, otherwise in Ethereum as
// 0x0000000000000000000000000000000000000000
var ZeroAddress = common.Address{}

// EmptyHash is a hash of all zeroes, otherwise in Ethereum as
// 0x0000000000000000000000000000000000000000000000000000000000000000
var EmptyHash = common.Hash{}

func WithoutZeroAddresses(addresses []common.Address) []common.Address {
	var withoutZeros []common.Address
	for _, address := range addresses {
		if address != ZeroAddress {
			withoutZeros = append(withoutZeros, address)
		}
	}
	return withoutZeros
}

func Uint64ToHex(i uint64) string {
	return fmt.Sprintf("0x%x", i)
}

var maxUint256 = common.HexToHash("0x" + strings.Repeat("f", 64)).Big()

func Uint256ToBytes(x *big.Int) (uint256 []byte, err error) {
	if x.Cmp(maxUint256) > 0 {
		return nil, fmt.Errorf("too large to convert to uint256")
	}
	uint256 = common.LeftPadBytes(x.Bytes(), EVMWordByteLen)
	if x.Cmp(big.NewInt(0).SetBytes(uint256)) != 0 {
		panic("failed to round-trip uint256 back to source big.Int")
	}
	return uint256, err
}

func ISO8601UTC(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func NullISO8601UTC(t null.Time) string {
	if t.Valid {
		return ISO8601UTC(t.Time)
	}
	return ""
}

func DurationFromNow(t time.Time) time.Duration {
	return time.Until(t)
}

func FormatJSON(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

func NewBytes32ID() string {
	return strings.ReplaceAll(uuid.NewV4().String(), "-", "")
}

func NewSecret(n int) string {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		panic(errors.Wrap(err, "generating secret failed"))
	}
	return base64.StdEncoding.EncodeToString(b)
}

func RemoveHexPrefix(str string) string {
	if HasHexPrefix(str) {
		return str[2:]
	}
	return str
}

func HasHexPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}

func DecodeEthereumTx(hex string) (types.Transaction, error) {
	var tx types.Transaction
	b, err := hexutil.Decode(hex)
	if err != nil {
		return tx, err
	}
	return tx, rlp.DecodeBytes(b, &tx)
}

func IsEmptyAddress(addr common.Address) bool {
	return addr == ZeroAddress
}

func StringToHex(in string) string {
	return AddHexPrefix(hex.EncodeToString([]byte(in)))
}

func AddHexPrefix(str string) string {
	if len(str) < 2 || len(str) > 1 && strings.ToLower(str[0:2]) != "0x" {
		str = "0x" + str
	}
	return str
}

func IsEmpty(bytes []byte) bool {
	for _, b := range bytes {
		if b != 0 {
			return false
		}
	}
	return true
}

type Sleeper interface {
	Reset()
	Sleep()
	After() time.Duration
	Duration() time.Duration
}

type BackoffSleeper struct {
	backoff.Backoff
	beenRun *abool.AtomicBool
}

func NewBackoffSleeper() *BackoffSleeper {
	return &BackoffSleeper{
		Backoff: backoff.Backoff{
			Min: 1 * time.Second,
			Max: 10 * time.Second,
		},
		beenRun: abool.New(),
	}
}

func (bs *BackoffSleeper) Sleep() {
	if bs.beenRun.SetToIf(false, true) {
		return
	}
	time.Sleep(bs.Backoff.Duration())
}

func (bs *BackoffSleeper) After() time.Duration {
	if bs.beenRun.SetToIf(false, true) {
		return 0
	}
	return bs.Backoff.Duration()
}

func (bs *BackoffSleeper) Duration() time.Duration {
	if !bs.beenRun.IsSet() {
		return 0
	}
	return bs.ForAttempt(bs.Attempt())
}

func (bs *BackoffSleeper) Reset() {
	bs.beenRun.UnSet()
	bs.Backoff.Reset()
}

func RetryWithBackoff(ctx context.Context, fn func() (retry bool)) {
	sleeper := NewBackoffSleeper()
	sleeper.Reset()
	for {
		retry := fn()
		if !retry {
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(sleeper.After()):
			continue
		}
	}
}

func MaxBigs(first *big.Int, bigs ...*big.Int) *big.Int {
	max := first
	for _, n := range bigs {
		if max.Cmp(n) < 0 {
			max = n
		}
	}
	return max
}

func MaxUint32(first uint32, uints ...uint32) uint32 {
	max := first
	for _, n := range uints {
		if n > max {
			max = n
		}
	}
	return max
}

func MaxInt(first int, ints ...int) int {
	max := first
	for _, n := range ints {
		if n > max {
			max = n
		}
	}
	return max
}

func MinUint(first uint, vals ...uint) uint {
	min := first
	for _, n := range vals {
		if n < min {
			min = n
		}
	}
	return min
}

func UnmarshalToMap(input string) (map[string]interface{}, error) {
	var output map[string]interface{}
	err := json.Unmarshal([]byte(input), &output)
	return output, err
}

func MustUnmarshalToMap(input string) map[string]interface{} {
	output, err := UnmarshalToMap(input)
	if err != nil {
		panic(err)
	}
	return output
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func Keccak256(in []byte) ([]byte, error) {
	hash := sha3.NewLegacyKeccak256()
	_, err := hash.Write(in)
	return hash.Sum(nil), err
}

func Sha256(in string) (string, error) {
	hasher := sha3.New256()
	_, err := hasher.Write([]byte(in))
	if err != nil {
		return "", errors.Wrap(err, "sha256 write error")
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func StripBearer(authorizationStr string) string {
	return strings.TrimPrefix(strings.TrimSpace(authorizationStr), "Bearer ")
}

func IsQuoted(input []byte) bool {
	return len(input) >= 2 &&
		((input[0] == '"' && input[len(input)-1] == '"') ||
			(input[0] == '\'' && input[len(input)-1] == '\''))
}

func RemoveQuotes(input []byte) []byte {
	if IsQuoted(input) {
		return input[1 : len(input)-1]
	}
	return input
}

func EIP55CapitalizedAddress(possibleAddressString string) bool {
	if !HasHexPrefix(possibleAddressString) {
		possibleAddressString = "0x" + possibleAddressString
	}
	EIP55Capitalized := common.HexToAddress(possibleAddressString).Hex()
	return possibleAddressString == EIP55Capitalized
}

func ParseEthereumAddress(addressString string) (common.Address, error) {
	if !common.IsHexAddress(addressString) {
		return common.Address{}, fmt.Errorf(
			"not a valid Ethereum address: %s", addressString)
	}
	address := common.HexToAddress(addressString)
	if !EIP55CapitalizedAddress(addressString) {
		return common.Address{}, fmt.Errorf(
			"%s treated as Ethereum address, but it has an invalid capitalization! "+
				"The correctly-capitalized address would be %s, but "+
				"check carefully before copying and pasting! ",
			addressString, address.Hex())
	}
	return address, nil
}

func MustHash(in string) common.Hash {
	out, err := Keccak256([]byte(in))
	if err != nil {
		panic(err)
	}
	return common.BytesToHash(out)
}

func LogListeningAddress(address common.Address) string {
	if address == ZeroAddress {
		return "[all]"
	}
	return address.String()
}

func JustError(_ interface{}, err error) error {
	return err
}

var zero = big.NewInt(0)

func CheckUint256(n *big.Int) error {
	if n.Cmp(zero) < 0 || n.Cmp(maxUint256) >= 0 {
		return fmt.Errorf("number out of range for uint256")
	}
	return nil
}

func HexToUint256(s string) (*big.Int, error) {
	rawNum, err := hexutil.Decode(s)
	if err != nil {
		return nil, errors.Wrapf(err, "while parsing %s as hex: ", s)
	}
	rv := big.NewInt(0).SetBytes(rawNum) // can't be negative number
	if err := CheckUint256(rv); err != nil {
		return nil, err
	}
	return rv, nil
}

func HexToBig(s string) *big.Int {
	n, ok := new(big.Int).SetString(s, 16)
	if !ok {
		panic(fmt.Errorf(`failed to convert "%s" as hex to big.Int`, s))
	}
	return n
}

func Uint256ToHex(n *big.Int) (string, error) {
	if err := CheckUint256(n); err != nil {
		return "", err
	}
	return common.BigToHash(n).Hex(), nil
}

func Uint256ToBytes32(n *big.Int) []byte {
	if n.BitLen() > 256 {
		panic("vrf.uint256ToBytes32: too big to marshal to uint256")
	}
	return common.LeftPadBytes(n.Bytes(), 32)
}

func ToDecimal(input interface{}) (decimal.Decimal, error) {
	switch v := input.(type) {
	case string:
		return decimal.NewFromString(v)
	case int:
		return decimal.New(int64(v), 0), nil
	case int8:
		return decimal.New(int64(v), 0), nil
	case int16:
		return decimal.New(int64(v), 0), nil
	case int32:
		return decimal.New(int64(v), 0), nil
	case int64:
		return decimal.New(v, 0), nil
	case uint:
		return decimal.New(int64(v), 0), nil
	case uint8:
		return decimal.New(int64(v), 0), nil
	case uint16:
		return decimal.New(int64(v), 0), nil
	case uint32:
		return decimal.New(int64(v), 0), nil
	case uint64:
		return decimal.New(int64(v), 0), nil
	case float64:
		return decimal.NewFromFloat(v), nil
	case float32:
		return decimal.NewFromFloat32(v), nil
	case *big.Int:
		return decimal.NewFromBigInt(v, 0), nil
	case decimal.Decimal:
		return v, nil
	case *decimal.Decimal:
		return *v, nil
	default:
		return decimal.Decimal{}, errors.Errorf("type %T cannot be converted to decimal.Decimal (%v)", input, input)
	}
}

func WaitGroupChan(wg *sync.WaitGroup) <-chan struct{} {
	chAwait := make(chan struct{})
	go func() {
		defer close(chAwait)
		wg.Wait()
	}()
	return chAwait
}

func ContextFromChan(chStop <-chan struct{}) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-chStop:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}

func CombinedContext(signals ...interface{}) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	if len(signals) == 0 {
		return ctx, cancel
	}
	signals = append(signals, ctx)

	var cases []reflect.SelectCase
	var cancel2 context.CancelFunc
	for _, signal := range signals {
		var ch reflect.Value

		switch sig := signal.(type) {
		case context.Context:
			ch = reflect.ValueOf(sig.Done())
		case <-chan struct{}:
			ch = reflect.ValueOf(sig)
		case chan struct{}:
			ch = reflect.ValueOf(sig)
		case time.Duration:
			var ctxTimeout context.Context
			ctxTimeout, cancel2 = context.WithTimeout(ctx, sig)
			ch = reflect.ValueOf(ctxTimeout.Done())
		default:
			logger.Errorf("utils.CombinedContext cannot accept a value of type %T, skipping", sig)
			continue
		}
		cases = append(cases, reflect.SelectCase{Chan: ch, Dir: reflect.SelectRecv})
	}

	go func() {
		defer cancel()
		if cancel2 != nil {
			defer cancel2()
		}
		_, _, _ = reflect.Select(cases)
	}()

	return ctx, cancel
}

type DependentAwaiter interface {
	AwaitDependents() <-chan struct{}
	AddDependents(n int)
	DependentReady()
}

type dependentAwaiter struct {
	wg *sync.WaitGroup
	ch <-chan struct{}
}

// NewDependentAwaiter creates a new DependentAwaiter
func NewDependentAwaiter() DependentAwaiter {
	return &dependentAwaiter{
		wg: &sync.WaitGroup{},
	}
}

func (da *dependentAwaiter) AwaitDependents() <-chan struct{} {
	if da.ch == nil {
		da.ch = WaitGroupChan(da.wg)
	}
	return da.ch
}

func (da *dependentAwaiter) AddDependents(n int) {
	da.wg.Add(n)
}

func (da *dependentAwaiter) DependentReady() {
	da.wg.Done()
}

// BoundedQueue is a FIFO queue that discards older items when it reaches its capacity.
type BoundedQueue struct {
	capacity uint
	items    []interface{}
	mu       *sync.RWMutex
}

// NewBoundedQueue creates a new BoundedQueue instance
func NewBoundedQueue(capacity uint) *BoundedQueue {
	return &BoundedQueue{
		capacity: capacity,
		mu:       &sync.RWMutex{},
	}
}

// Add appends items to a BoundedQueue
func (q *BoundedQueue) Add(x interface{}) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = append(q.items, x)
	if uint(len(q.items)) > q.capacity {
		excess := uint(len(q.items)) - q.capacity
		q.items = q.items[excess:]
	}
}

func (q *BoundedQueue) Take() interface{} {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return nil
	}
	x := q.items[0]
	q.items = q.items[1:]
	return x
}

func (q *BoundedQueue) Empty() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.items) == 0
}

// Full checks if a BoundedQueue is over capacity.
func (q *BoundedQueue) Full() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return uint(len(q.items)) >= q.capacity
}

type BoundedPriorityQueue struct {
	queues     map[uint]*BoundedQueue
	priorities []uint
	capacities map[uint]uint
	mu         *sync.RWMutex
}

func NewBoundedPriorityQueue(capacities map[uint]uint) *BoundedPriorityQueue {
	queues := make(map[uint]*BoundedQueue)
	var priorities []uint
	for priority, capacity := range capacities {
		priorities = append(priorities, priority)
		queues[priority] = NewBoundedQueue(capacity)
	}
	sort.Slice(priorities, func(i, j int) bool { return priorities[i] < priorities[j] })
	return &BoundedPriorityQueue{
		queues:     queues,
		priorities: priorities,
		capacities: capacities,
		mu:         &sync.RWMutex{},
	}
}

func (q *BoundedPriorityQueue) Add(priority uint, x interface{}) {
	q.mu.Lock()
	defer q.mu.Unlock()

	subqueue, exists := q.queues[priority]
	if !exists {
		panic(fmt.Sprintf("nonexistent priority: %v", priority))
	}

	subqueue.Add(x)
}

func (q *BoundedPriorityQueue) Take() interface{} {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, priority := range q.priorities {
		queue := q.queues[priority]
		if queue.Empty() {
			continue
		}
		return queue.Take()
	}
	return nil
}

func (q *BoundedPriorityQueue) Empty() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for _, priority := range q.priorities {
		queue := q.queues[priority]
		if !queue.Empty() {
			return false
		}
	}
	return true
}

func WrapIfError(err *error, msg string) {
	if *err != nil {
		*err = errors.Wrap(*err, msg)
	}
}

func LogIfError(err *error, msg string) {
	if *err != nil {
		logger.Errorf(msg+": %+v", *err)
	}
}

func DebugPanic() {
	//revive:disable:defer
	if err := recover(); err != nil {
		pc := make([]uintptr, 10) // at least 1 entry needed
		runtime.Callers(5, pc)
		f := runtime.FuncForPC(pc[0])
		file, line := f.FileLine(pc[0])
		logger.Errorf("Caught panic in %v (%v#%v): %v", f.Name(), file, line, err)
		panic(err)
	}
}

type TickerBase interface {
	Resume()
	Pause()
	Destroy()
	Ticks() <-chan time.Time
}

// PausableTicker stores a ticker with a duration
type PausableTicker struct {
	ticker   *time.Ticker
	duration time.Duration
	mu       *sync.RWMutex
}

// NewPausableTicker creates a new PausableTicker
func NewPausableTicker(duration time.Duration) PausableTicker {
	return PausableTicker{
		duration: duration,
		mu:       &sync.RWMutex{},
	}
}

// Ticks retrieves the ticks from a PausableTicker
func (t PausableTicker) Ticks() <-chan time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.ticker == nil {
		return nil
	}
	return t.ticker.C
}

// Pause pauses a PausableTicker
func (t *PausableTicker) Pause() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.ticker != nil {
		t.ticker.Stop()
		t.ticker = nil
	}
}

// Resume resumes a Ticker
// using a PausibleTicker's duration
func (t *PausableTicker) Resume() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.ticker == nil {
		t.ticker = time.NewTicker(t.duration)
	}
}

// Destroy pauses the PausibleTicker
func (t *PausableTicker) Destroy() {
	t.Pause()
}

type CronTicker struct {
	*cron.Cron
	ch      chan time.Time
	beenRun *abool.AtomicBool
}

func NewCronTicker(schedule string) (CronTicker, error) {
	cron := cron.New(cron.WithSeconds())
	ch := make(chan time.Time, 1)
	_, err := cron.AddFunc(schedule, func() {
		select {
		case ch <- time.Now():
		default:
		}
	})
	if err != nil {
		return CronTicker{beenRun: abool.New()}, err
	}
	return CronTicker{Cron: cron, ch: ch, beenRun: abool.New()}, nil
}

// Start - returns true if the CronTicker was actually started, false otherwise
func (t *CronTicker) Start() bool {
	if t.Cron != nil {
		if t.beenRun.SetToIf(false, true) {
			t.Cron.Start()
			return true
		}
	}
	return false
}

func (t *CronTicker) Stop() bool {
	if t.Cron != nil {
		if t.beenRun.SetToIf(true, false) {
			t.Cron.Stop()
			return true
		}
	}
	return false
}

func (t *CronTicker) Ticks() <-chan time.Time {
	return t.ch
}

func ValidateCronSchedule(schedule string) error {
	if !(strings.HasPrefix(schedule, "CRON_TZ=") || strings.HasPrefix(schedule, "@every ")) {
		return errors.New("cron schedule must specify a time zone using CRON_TZ, e.g. 'CRON_TZ=UTC 5 * * * *', or use the @every syntax, e.g. '@every 1h30m'")
	}
	parser := cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	_, err := parser.Parse(schedule)
	return errors.Wrapf(err, "invalid cron schedule '%v'", schedule)
}

// ResettableTimer stores a timer
type ResettableTimer struct {
	timer *time.Timer
	mu    *sync.RWMutex
}

// NewResettableTimer creates a new ResettableTimer
func NewResettableTimer() ResettableTimer {
	return ResettableTimer{
		mu: &sync.RWMutex{},
	}
}

func (t ResettableTimer) Ticks() <-chan time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.timer == nil {
		return nil
	}
	return t.timer.C
}

func (t *ResettableTimer) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.timer != nil {
		t.timer.Stop()
		t.timer = nil
	}
}

func (t *ResettableTimer) Reset(duration time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.timer != nil {
		t.timer.Stop()
	}
	t.timer = time.NewTimer(duration)
}

func EVMBytesToUint64(buf []byte) uint64 {
	var result uint64
	for _, b := range buf {
		result = result<<8 + uint64(b)
	}
	return result
}

var (
	ErrNotStarted = errors.New("Not started")
)

// StartStopOnce contains a StartStopOnceState integer
type StartStopOnce struct {
	state        atomic.Int32
	sync.RWMutex // lock is held during statup/shutdown, RLock is held while executing functions dependent on a particular state
}

type StartStopOnceState int32

const (
	StartStopOnce_Unstarted StartStopOnceState = iota
	StartStopOnce_Started
	StartStopOnce_Starting
	StartStopOnce_Stopping
	StartStopOnce_Stopped
)

func (once *StartStopOnce) StartOnce(name string, fn func() error) error {
	// SAFETY: We do this compare-and-swap outside of the lock so that
	// concurrent StartOnce() calls return immediately.
	success := once.state.CAS(int32(StartStopOnce_Unstarted), int32(StartStopOnce_Starting))

	if !success {
		return errors.Errorf("%v has already started once", name)
	}

	once.Lock()
	defer once.Unlock()

	err := fn()

	success = once.state.CAS(int32(StartStopOnce_Starting), int32(StartStopOnce_Started))

	if !success {
		// SAFETY: If this is reached, something must be very wrong: once.state
		// was tampered with outside of the lock.
		panic(fmt.Sprintf("%v entered unreachable state, unable to set state to started", name))
	}

	return err
}

func (once *StartStopOnce) StopOnce(name string, fn func() error) error {
	once.Lock()
	defer once.Unlock()

	success := once.state.CAS(int32(StartStopOnce_Started), int32(StartStopOnce_Stopping))

	if !success {
		return errors.Errorf("%v has already stopped once", name)
	}

	err := fn()

	success = once.state.CAS(int32(StartStopOnce_Stopping), int32(StartStopOnce_Stopped))

	if !success {
		// SAFETY: If this is reached, something must be very wrong: once.state
		// was tampered with outside of the lock.
		panic(fmt.Sprintf("%v entered unreachable state, unable to set state to stopped", name))
	}

	return err
}

func (once *StartStopOnce) State() StartStopOnceState {
	state := once.state.Load()
	return StartStopOnceState(state)
}

func (once *StartStopOnce) IfStarted(f func()) (ok bool) {
	once.RLock()
	defer once.RUnlock()

	state := once.state.Load()

	if StartStopOnceState(state) == StartStopOnce_Started {
		f()
		return true
	}
	return false
}

func (once *StartStopOnce) Ready() error {
	if once.State() == StartStopOnce_Started {
		return nil
	}
	return ErrNotStarted
}

// Override this per-service with more specific implementations
func (once *StartStopOnce) Healthy() error {
	if once.State() == StartStopOnce_Started {
		return nil
	}
	return ErrNotStarted
}

func WithJitter(d time.Duration) time.Duration {
	jitter := mrand.Intn(int(d) / 5)
	jitter = jitter - (jitter / 2)
	return time.Duration(int(d) + jitter)
}

type KeyedMutex struct {
	mutexes sync.Map
}

func (m *KeyedMutex) LockInt64(key int64) func() {
	value, _ := m.mutexes.LoadOrStore(key, new(sync.Mutex))
	mtx := value.(*sync.Mutex)
	mtx.Lock()

	return func() { mtx.Unlock() }
}

func BoxOutput(errorMsgTemplate string, errorMsgValues ...interface{}) string {
	errorMsgTemplate = fmt.Sprintf(errorMsgTemplate, errorMsgValues...)
	lines := strings.Split(errorMsgTemplate, "\n")
	maxlen := 0
	for _, line := range lines {
		if len(line) > maxlen {
			maxlen = len(line)
		}
	}
	internalLength := maxlen + 4
	output := "↘" + strings.Repeat("↓", internalLength) + "↙\n" // top line
	output += "→  " + strings.Repeat(" ", maxlen) + "  ←\n"
	readme := strings.Repeat("README ", maxlen/7)
	output += "→  " + readme + strings.Repeat(" ", maxlen-len(readme)) + "  ←\n"
	output += "→  " + strings.Repeat(" ", maxlen) + "  ←\n"
	for _, line := range lines {
		output += "→  " + line + strings.Repeat(" ", maxlen-len(line)) + "  ←\n"
	}
	output += "→  " + strings.Repeat(" ", maxlen) + "  ←\n"
	output += "→  " + readme + strings.Repeat(" ", maxlen-len(readme)) + "  ←\n"
	output += "→  " + strings.Repeat(" ", maxlen) + "  ←\n"
	return "\n" + output + "↗" + strings.Repeat("↑", internalLength) + "↖" + // bottom line
		"\n\n"
}

func Example_boxOutput() {
	fmt.Println()
	fmt.Print(BoxOutput("%s is %d", "foo", 17))
}
