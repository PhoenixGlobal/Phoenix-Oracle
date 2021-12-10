package config

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"sync"
	"time"

	"PhoenixOracle/build/static"
	"PhoenixOracle/core/assets"
	"PhoenixOracle/core/chain"
	"PhoenixOracle/core/keystore"
	"PhoenixOracle/core/keystore/keys/ethkey"
	"PhoenixOracle/core/keystore/keys/p2pkey"
	"PhoenixOracle/db/dialects"
	"PhoenixOracle/db/models"
	ocrnetworking "PhoenixOracle/lib/libocr/networking"
	ocrtypes "PhoenixOracle/lib/libocr/offchainreporting/types"
	"PhoenixOracle/lib/logger"
	"PhoenixOracle/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/contrib/sessions"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"
)

const readWritePerms = os.FileMode(0600)

var (
	ErrUnset   = errors.New("env var unset")
	ErrInvalid = errors.New("env var invalid")

	configFileNotFoundError = reflect.TypeOf(viper.ConfigFileNotFoundError{})
)

type GeneralConfig interface {
	AdminCredentialsFile() string
	AllowOrigins() string
	AuthenticatedRateLimit() int64
	AuthenticatedRateLimitPeriod() models.Duration
	BlockBackfillDepth() uint64
	BlockBackfillSkip() bool
	BridgeResponseURL() *url.URL
	CertFile() string

	Chain() *chain.Chain
	ChainID() *big.Int
	ClientNodeURL() string
	CreateProductionLogger() *logger.Logger
	DatabaseBackupDir() string
	DatabaseBackupFrequency() time.Duration
	DatabaseBackupMode() DatabaseBackupMode
	DatabaseBackupURL() *url.URL
	DatabaseListenerMaxReconnectDuration() time.Duration
	DatabaseListenerMinReconnectInterval() time.Duration
	HTTPServerWriteTimeout() time.Duration
	DatabaseMaximumTxDuration() time.Duration
	DatabaseTimeout() models.Duration
	DatabaseURL() url.URL
	DefaultHTTPAllowUnrestrictedNetworkAccess() bool
	DefaultHTTPLimit() int64
	DefaultHTTPTimeout() models.Duration
	DefaultMaxHTTPAttempts() uint
	Dev() bool
	EthereumDisabled() bool
	EthereumHTTPURL() *url.URL
	EthereumSecondaryURLs() []url.URL
	EthereumURL() string
	ExplorerAccessKey() string
	ExplorerSecret() string
	ExplorerURL() *url.URL
	FMDefaultTransactionQueueDepth() uint32
	FeatureCronV2() bool
	FeatureUICSAKeys() bool
	FeatureUIFeedsManager() bool
	FeatureExternalInitiators() bool
	FeatureFluxMonitorV2() bool
	FeatureOffchainReporting() bool
	FeatureWebhookV2() bool
	GetAdvisoryLockIDConfiguredOrDefault() int64
	GetDatabaseDialectConfiguredOrDefault() dialects.DialectName
	GlobalLockRetryInterval() models.Duration
	InsecureFastScrypt() bool
	InsecureSkipVerify() bool
	JSONConsole() bool
	JobPipelineMaxRunDuration() time.Duration
	JobPipelineReaperInterval() time.Duration
	JobPipelineReaperThreshold() time.Duration
	JobPipelineResultWriteQueueDepth() uint64
	KeeperDefaultTransactionQueueDepth() uint32
	KeeperMaximumGracePeriod() int64
	KeeperMinimumRequiredConfirmations() uint64
	KeeperRegistryCheckGasOverhead() uint64
	KeeperRegistryPerformGasOverhead() uint64
	KeeperRegistrySyncInterval() time.Duration
	KeyFile() string
	Layer2Type() string
	LogLevel() LogLevel
	LogSQLMigrations() bool
	LogSQLStatements() bool
	LogToDisk() bool
	MigrateDatabase() bool
	OCRBlockchainTimeout() time.Duration
	OCRBootstrapCheckInterval() time.Duration
	OCRContractPollInterval() time.Duration
	OCRContractSubscribeInterval() time.Duration
	OCRContractTransmitterTransmitTimeout() time.Duration
	OCRDHTLookupInterval() int
	OCRDatabaseTimeout() time.Duration
	OCRDefaultTransactionQueueDepth() uint32
	OCRIncomingMessageBufferSize() int
	OCRKeyBundleID() (string, error)
	OCRMonitoringEndpoint() string
	OCRNewStreamTimeout() time.Duration
	OCRObservationGracePeriod() time.Duration
	OCRObservationTimeout() time.Duration
	OCROutgoingMessageBufferSize() int
	OCRTraceLogging() bool
	OCRTransmitterAddress() (ethkey.EIP55Address, error)
	ORMMaxIdleConns() int
	ORMMaxOpenConns() int
	P2PAnnounceIP() net.IP
	P2PAnnouncePort() uint16
	P2PBootstrapPeers() ([]string, error)
	P2PDHTAnnouncementCounterUserPrefix() uint32
	P2PListenIP() net.IP
	P2PListenPort() uint16
	P2PListenPortRaw() string
	P2PNetworkingStack() (n ocrnetworking.NetworkingStack)
	P2PNetworkingStackRaw() string
	P2PPeerID() p2pkey.PeerID
	P2PPeerIDRaw() string
	P2PPeerstoreWriteInterval() time.Duration
	P2PV2AnnounceAddresses() []string
	P2PV2AnnounceAddressesRaw() []string
	P2PV2Bootstrappers() (locators []ocrtypes.BootstrapperLocator)
	P2PV2BootstrappersRaw() []string
	P2PV2DeltaDial() models.Duration
	P2PV2DeltaReconcile() models.Duration
	P2PV2ListenAddresses() []string
	PhbContractAddress() string
	Port() uint16
	ReaperExpiration() models.Duration
	ReplayFromBlock() int64
	RootDir() string
	SecureCookies() bool
	SessionOptions() sessions.Options
	SessionSecret() ([]byte, error)
	SessionTimeout() models.Duration
	SetDB(*gorm.DB)
	SetLogLevel(ctx context.Context, value string) error
	SetLogSQLStatements(ctx context.Context, sqlEnabled bool) error
	SetDialect(dialects.DialectName)
	StatsPusherLogging() bool
	TelemetryIngressLogging() bool
	TelemetryIngressServerPubKey() string
	TelemetryIngressURL() *url.URL
	TLSCertPath() string
	TLSDir() string
	TLSHost() string
	TLSKeyPath() string
	TLSPort() uint16
	TLSRedirect() bool
	TriggerFallbackDBPollInterval() time.Duration
	UnAuthenticatedRateLimit() int64
	UnAuthenticatedRateLimitPeriod() models.Duration
	Validate() error
}

type generalConfig struct {
	viper            *viper.Viper
	secretGenerator  SecretGenerator
	ORM              *ORM
	ks               keystore.Master
	randomP2PPort    uint16
	randomP2PPortMtx *sync.RWMutex
	dialect          dialects.DialectName
	advisoryLockID   int64
	p2ppeerIDmtx     sync.Mutex
}

const defaultPostgresAdvisoryLockID int64 = 1027321974924625846

func NewGeneralConfig() GeneralConfig {
	v := viper.New()
	c := newGeneralConfigWithViper(v)
	c.secretGenerator = FilePersistedSecretGenerator{}
	c.dialect = dialects.Postgres
	return c
}

func newGeneralConfigWithViper(v *viper.Viper) *generalConfig {
	schemaT := reflect.TypeOf(ConfigSchema{})
	for index := 0; index < schemaT.NumField(); index++ {
		item := schemaT.FieldByIndex([]int{index})
		name := item.Tag.Get("env")
		def, exists := item.Tag.Lookup("default")
		if exists {
			v.SetDefault(name, def)
		}
		_ = v.BindEnv(name, name)
	}

	_ = v.BindEnv("MINIMUM_CONTRACT_PAYMENT")

	config := &generalConfig{
		viper:            v,
		randomP2PPortMtx: new(sync.RWMutex),
	}

	if err := utils.EnsureDirAndMaxPerms(config.RootDir(), os.FileMode(0700)); err != nil {
		logger.Fatalf(`Error creating root directory "%s": %+v`, config.RootDir(), err)
	}

	v.SetConfigName("phoenix")
	v.AddConfigPath(config.RootDir())
	err := v.ReadInConfig()
	if err != nil && reflect.TypeOf(err) != configFileNotFoundError {
		logger.Warnf("Unable to load config file: %v\n", err)
	}

	return config
}

func (c *generalConfig) Validate() error {
	if c.P2PAnnouncePort() != 0 && c.P2PAnnounceIP() == nil {
		return errors.Errorf("P2P_ANNOUNCE_PORT was given as %v but P2P_ANNOUNCE_IP was unset. You must also set P2P_ANNOUNCE_IP if P2P_ANNOUNCE_PORT is set", c.P2PAnnouncePort())
	}

	if c.viper.IsSet("MINIMUM_CONTRACT_PAYMENT") {
		logger.Warn("MINIMUM_CONTRACT_PAYMENT is now deprecated and will be removed from a future release, use MINIMUM_CONTRACT_PAYMENT_PHB_JUELS instead.")
	}

	if _, err := c.OCRKeyBundleID(); errors.Cause(err) == ErrInvalid {
		return err
	}
	if _, err := c.OCRTransmitterAddress(); errors.Cause(err) == ErrInvalid {
		return err
	}
	if peers, err := c.P2PBootstrapPeers(); err == nil {
		for i := range peers {
			if _, err := multiaddr.NewMultiaddr(peers[i]); err != nil {
				return errors.Errorf("p2p bootstrap peer %d is invalid: err %v", i, err)
			}
		}
	}
	if me := c.OCRMonitoringEndpoint(); me != "" {
		if _, err := url.Parse(me); err != nil {
			return errors.Wrapf(err, "invalid monitoring url: %s", me)
		}
	}
	return nil
}

func (c *generalConfig) SetDB(db *gorm.DB) {
	orm := NewORM(db)
	c.ORM = orm
}

func (c *generalConfig) SetDialect(d dialects.DialectName) {
	c.dialect = d
}

func (c *generalConfig) GetAdvisoryLockIDConfiguredOrDefault() int64 {
	return c.advisoryLockID
}

func (c *generalConfig) GetDatabaseDialectConfiguredOrDefault() dialects.DialectName {
	return c.dialect
}

func (c *generalConfig) AllowOrigins() string {
	return c.viper.GetString(EnvVarName("AllowOrigins"))
}

func (c *generalConfig) AdminCredentialsFile() string {
	fieldName := "AdminCredentialsFile"
	file := c.viper.GetString(EnvVarName(fieldName))
	defaultValue, _ := defaultValue(fieldName)
	if file == defaultValue {
		return filepath.Join(c.RootDir(), "apicredentials")
	}
	return file
}

func (c *generalConfig) AuthenticatedRateLimit() int64 {
	return c.viper.GetInt64(EnvVarName("AuthenticatedRateLimit"))
}

func (c *generalConfig) AuthenticatedRateLimitPeriod() models.Duration {
	return models.MustMakeDuration(c.getWithFallback("AuthenticatedRateLimitPeriod", parseDuration).(time.Duration))
}

func (c *generalConfig) BlockBackfillDepth() uint64 {
	return c.getWithFallback("BlockBackfillDepth", parseUint64).(uint64)
}

func (c *generalConfig) BlockBackfillSkip() bool {
	return c.getWithFallback("BlockBackfillSkip", parseBool).(bool)
}

func (c *generalConfig) BridgeResponseURL() *url.URL {
	return c.getWithFallback("BridgeResponseURL", parseURL).(*url.URL)
}

func (c *generalConfig) ChainID() *big.Int {
	return c.getWithFallback("ChainID", parseBigInt).(*big.Int)
}

func (c *generalConfig) Chain() *chain.Chain {
	return chain.ChainFromID(c.ChainID())
}

func (c *generalConfig) ClientNodeURL() string {
	return c.viper.GetString(EnvVarName("ClientNodeURL"))
}

func (c *generalConfig) FeatureCronV2() bool {
	return c.getWithFallback("FeatureCronV2", parseBool).(bool)
}

func (c *generalConfig) FeatureUICSAKeys() bool {
	return c.getWithFallback("FeatureUICSAKeys", parseBool).(bool)
}

func (c *generalConfig) FeatureUIFeedsManager() bool {
	return c.getWithFallback("FeatureUIFeedsManager", parseBool).(bool)
}

func (c *generalConfig) DatabaseListenerMinReconnectInterval() time.Duration {
	return c.getWithFallback("DatabaseListenerMinReconnectInterval", parseDuration).(time.Duration)
}

func (c *generalConfig) DatabaseListenerMaxReconnectDuration() time.Duration {
	return c.getWithFallback("DatabaseListenerMaxReconnectDuration", parseDuration).(time.Duration)
}

func (c *generalConfig) DatabaseMaximumTxDuration() time.Duration {
	return c.getWithFallback("DatabaseMaximumTxDuration", parseDuration).(time.Duration)
}

func (c *generalConfig) DatabaseBackupMode() DatabaseBackupMode {
	return c.getWithFallback("DatabaseBackupMode", parseDatabaseBackupMode).(DatabaseBackupMode)
}

func (c *generalConfig) DatabaseBackupFrequency() time.Duration {
	return c.getWithFallback("DatabaseBackupFrequency", parseDuration).(time.Duration)
}

func (c *generalConfig) DatabaseBackupURL() *url.URL {
	s := c.viper.GetString(EnvVarName("DatabaseBackupURL"))
	if s == "" {
		return nil
	}
	uri, err := url.Parse(s)
	if err != nil {
		logger.Errorf("invalid database backup url %s", s)
		return nil
	}
	return uri
}

func (c *generalConfig) DatabaseBackupDir() string {
	return c.viper.GetString(EnvVarName("DatabaseBackupDir"))
}

func (c *generalConfig) DatabaseTimeout() models.Duration {
	return models.MustMakeDuration(c.getWithFallback("DatabaseTimeout", parseDuration).(time.Duration))
}

func (c *generalConfig) GlobalLockRetryInterval() models.Duration {
	return models.MustMakeDuration(c.getWithFallback("GlobalLockRetryInterval", parseDuration).(time.Duration))
}

func (c *generalConfig) DatabaseURL() url.URL {
	s := c.viper.GetString(EnvVarName("DatabaseURL"))
	uri, err := url.Parse(s)
	if err != nil {
		logger.Error("invalid database url %s", s)
		return url.URL{}
	}
	if uri.String() == "" {
		return *uri
	}
	static.SetConsumerName(uri, "Default")
	return *uri
}

func (c *generalConfig) MigrateDatabase() bool {
	return c.viper.GetBool(EnvVarName("MigrateDatabase"))
}

func (c *generalConfig) DefaultMaxHTTPAttempts() uint {
	return uint(c.getWithFallback("DefaultMaxHTTPAttempts", parseUint64).(uint64))
}

func (c *generalConfig) DefaultHTTPLimit() int64 {
	return c.viper.GetInt64(EnvVarName("DefaultHTTPLimit"))
}

func (c *generalConfig) DefaultHTTPTimeout() models.Duration {
	return models.MustMakeDuration(c.getWithFallback("DefaultHTTPTimeout", parseDuration).(time.Duration))
}

func (c *generalConfig) DefaultHTTPAllowUnrestrictedNetworkAccess() bool {
	return c.viper.GetBool(EnvVarName("DefaultHTTPAllowUnrestrictedNetworkAccess"))
}

// Dev configures "development" mode for phoenix.
func (c *generalConfig) Dev() bool {
	return c.viper.GetBool(EnvVarName("Dev"))
}

func (c *generalConfig) FeatureExternalInitiators() bool {
	return c.viper.GetBool(EnvVarName("FeatureExternalInitiators"))
}

func (c *generalConfig) FeatureFluxMonitorV2() bool {
	return c.getWithFallback("FeatureFluxMonitorV2", parseBool).(bool)
}

func (c *generalConfig) FeatureOffchainReporting() bool {
	return c.viper.GetBool(EnvVarName("FeatureOffchainReporting"))
}

func (c *generalConfig) FeatureWebhookV2() bool {
	return c.getWithFallback("FeatureWebhookV2", parseBool).(bool)
}

func (c *generalConfig) FMDefaultTransactionQueueDepth() uint32 {
	return c.viper.GetUint32(EnvVarName("FMDefaultTransactionQueueDepth"))
}

func (c *generalConfig) EthereumURL() string {
	return c.viper.GetString(EnvVarName("EthereumURL"))
}

func (c *generalConfig) EthereumHTTPURL() (uri *url.URL) {
	urlStr := c.viper.GetString(EnvVarName("EthereumHTTPURL"))
	if urlStr == "" {
		return nil
	}
	var err error
	uri, err = url.Parse(urlStr)
	if err != nil || !(uri.Scheme == "http" || uri.Scheme == "https") {
		logger.Fatalf("Invalid Ethereum HTTP URL: %s, got error: %s", urlStr, err)
	}
	return
}

func (c *generalConfig) EthereumSecondaryURLs() []url.URL {
	oldConfig := c.viper.GetString(EnvVarName("EthereumSecondaryURL"))
	newConfig := c.viper.GetString(EnvVarName("EthereumSecondaryURLs"))

	config := ""
	if newConfig != "" {
		config = newConfig
	} else if oldConfig != "" {
		config = oldConfig
	}

	urlStrings := regexp.MustCompile(`\s*[;,]\s*`).Split(config, -1)
	urls := []url.URL{}
	for _, urlString := range urlStrings {
		if urlString == "" {
			continue
		}
		url, err := url.Parse(urlString)
		if err != nil {
			logger.Fatalf("Invalid Secondary Ethereum URL: %s, got error: %v", urlString, err)
		}
		urls = append(urls, *url)
	}

	return urls
}

func (c *generalConfig) EthereumDisabled() bool {
	return c.viper.GetBool(EnvVarName("EthereumDisabled"))
}

func (c *generalConfig) InsecureFastScrypt() bool {
	return c.viper.GetBool(EnvVarName("InsecureFastScrypt"))
}

func (c *generalConfig) InsecureSkipVerify() bool {
	return c.viper.GetBool(EnvVarName("InsecureSkipVerify"))
}

func (c *generalConfig) TriggerFallbackDBPollInterval() time.Duration {
	return c.getWithFallback("TriggerFallbackDBPollInterval", parseDuration).(time.Duration)
}

func (c *generalConfig) JobPipelineMaxRunDuration() time.Duration {
	return c.getWithFallback("JobPipelineMaxRunDuration", parseDuration).(time.Duration)
}

func (c *generalConfig) JobPipelineResultWriteQueueDepth() uint64 {
	return c.getWithFallback("JobPipelineResultWriteQueueDepth", parseUint64).(uint64)
}

func (c *generalConfig) JobPipelineReaperInterval() time.Duration {
	return c.getWithFallback("JobPipelineReaperInterval", parseDuration).(time.Duration)
}

func (c *generalConfig) JobPipelineReaperThreshold() time.Duration {
	return c.getWithFallback("JobPipelineReaperThreshold", parseDuration).(time.Duration)
}

func (c *generalConfig) KeeperRegistryCheckGasOverhead() uint64 {
	return c.getWithFallback("KeeperRegistryCheckGasOverhead", parseUint64).(uint64)
}

func (c *generalConfig) KeeperRegistryPerformGasOverhead() uint64 {
	return c.getWithFallback("KeeperRegistryPerformGasOverhead", parseUint64).(uint64)
}

func (c *generalConfig) KeeperDefaultTransactionQueueDepth() uint32 {
	return c.viper.GetUint32(EnvVarName("KeeperDefaultTransactionQueueDepth"))
}

func (c *generalConfig) KeeperRegistrySyncInterval() time.Duration {
	return c.getWithFallback("KeeperRegistrySyncInterval", parseDuration).(time.Duration)
}

func (c *generalConfig) KeeperMinimumRequiredConfirmations() uint64 {
	return c.viper.GetUint64(EnvVarName("KeeperMinimumRequiredConfirmations"))
}

func (c *generalConfig) KeeperMaximumGracePeriod() int64 {
	return c.viper.GetInt64(EnvVarName("KeeperMaximumGracePeriod"))
}

func (c *generalConfig) JSONConsole() bool {
	return c.viper.GetBool(EnvVarName("JSONConsole"))
}

func (c *generalConfig) ExplorerURL() *url.URL {
	rval := c.getWithFallback("ExplorerURL", parseURL)
	switch t := rval.(type) {
	case nil:
		return nil
	case *url.URL:
		return t
	default:
		logger.Panicf("invariant: ExplorerURL returned as type %T", rval)
		return nil
	}
}

func (c *generalConfig) ExplorerAccessKey() string {
	return c.viper.GetString(EnvVarName("ExplorerAccessKey"))
}

func (c *generalConfig) ExplorerSecret() string {
	return c.viper.GetString(EnvVarName("ExplorerSecret"))
}

func (c *generalConfig) TelemetryIngressURL() *url.URL {
	rval := c.getWithFallback("TelemetryIngressURL", parseURL)
	switch t := rval.(type) {
	case nil:
		return nil
	case *url.URL:
		return t
	default:
		logger.Panicf("invariant: TelemetryIngressURL returned as type %T", rval)
		return nil
	}
}

func (c *generalConfig) TelemetryIngressServerPubKey() string {
	return c.viper.GetString(EnvVarName("TelemetryIngressServerPubKey"))
}

func (c *generalConfig) TelemetryIngressLogging() bool {
	return c.getWithFallback("TelemetryIngressLogging", parseBool).(bool)
}

func (c *generalConfig) OCRBootstrapCheckInterval() time.Duration {
	return c.getWithFallback("OCRBootstrapCheckInterval", parseDuration).(time.Duration)
}

func (c *generalConfig) OCRContractTransmitterTransmitTimeout() time.Duration {
	return c.getWithFallback("OCRContractTransmitterTransmitTimeout", parseDuration).(time.Duration)
}

func (c *generalConfig) getDuration(field string) time.Duration {
	return c.getWithFallback(field, parseDuration).(time.Duration)
}

func (c *generalConfig) OCRObservationTimeout() time.Duration {
	return c.getDuration("OCRObservationTimeout")
}

func (c *generalConfig) OCRObservationGracePeriod() time.Duration {
	return c.getWithFallback("OCRObservationGracePeriod", parseDuration).(time.Duration)
}

func (c *generalConfig) OCRBlockchainTimeout() time.Duration {
	return c.getDuration("OCRBlockchainTimeout")
}

func (c *generalConfig) OCRContractSubscribeInterval() time.Duration {
	return c.getDuration("OCRContractSubscribeInterval")
}

func (c *generalConfig) OCRContractPollInterval() time.Duration {
	return c.getDuration("OCRContractPollInterval")
}

func (c *generalConfig) OCRDatabaseTimeout() time.Duration {
	return c.getWithFallback("OCRDatabaseTimeout", parseDuration).(time.Duration)
}

func (c *generalConfig) OCRDHTLookupInterval() int {
	return int(c.getWithFallback("OCRDHTLookupInterval", parseUint16).(uint16))
}

func (c *generalConfig) OCRIncomingMessageBufferSize() int {
	return int(c.getWithFallback("OCRIncomingMessageBufferSize", parseUint16).(uint16))
}

func (c *generalConfig) OCRNewStreamTimeout() time.Duration {
	return c.getWithFallback("OCRNewStreamTimeout", parseDuration).(time.Duration)
}

func (c *generalConfig) OCROutgoingMessageBufferSize() int {
	return int(c.getWithFallback("OCRIncomingMessageBufferSize", parseUint16).(uint16))
}

func (c *generalConfig) OCRTraceLogging() bool {
	return c.viper.GetBool(EnvVarName("OCRTraceLogging"))
}

func (c *generalConfig) OCRMonitoringEndpoint() string {
	return c.viper.GetString(EnvVarName("OCRMonitoringEndpoint"))
}

func (c *generalConfig) OCRDefaultTransactionQueueDepth() uint32 {
	return c.viper.GetUint32(EnvVarName("OCRDefaultTransactionQueueDepth"))
}

func (c *generalConfig) OCRTransmitterAddress() (ethkey.EIP55Address, error) {
	taStr := c.viper.GetString(EnvVarName("OCRTransmitterAddress"))
	if taStr != "" {
		ta, err := ethkey.NewEIP55Address(taStr)
		if err != nil {
			return "", errors.Wrapf(ErrInvalid, "OCR_TRANSMITTER_ADDRESS is invalid EIP55 %v", err)
		}
		return ta, nil
	}
	return "", errors.Wrap(ErrUnset, "OCR_TRANSMITTER_ADDRESS")
}

func (c *generalConfig) OCRKeyBundleID() (string, error) {
	kbStr := c.viper.GetString(EnvVarName("OCRKeyBundleID"))
	if kbStr != "" {
		_, err := models.Sha256HashFromHex(kbStr)
		if err != nil {
			return "", errors.Wrapf(ErrInvalid, "OCR_KEY_BUNDLE_ID is an invalid sha256 hash hex string %v", err)
		}
	}
	return kbStr, nil
}

func (c *generalConfig) ORMMaxOpenConns() int {
	return int(c.getWithFallback("ORMMaxOpenConns", parseUint16).(uint16))
}

func (c *generalConfig) ORMMaxIdleConns() int {
	return int(c.getWithFallback("ORMMaxIdleConns", parseUint16).(uint16))
}

func (*generalConfig) Layer2Type() string {
	val, ok := lookupEnv(EnvVarName("Layer2Type"), parseString)
	if !ok || val == nil {
		return ""
	}
	return val.(string)
}

func (*generalConfig) PhbContractAddress() string {
	val, ok := lookupEnv("PHB_CONTRACT_ADDRESS", parseString)
	if !ok || val == nil {
		return ""
	}
	return val.(string)
}

// LogLevel represents the maximum level of log messages to output.
func (c *generalConfig) LogLevel() LogLevel {
	if c.ORM != nil {
		var value LogLevel
		if err := c.ORM.GetConfigValue("LogLevel", &value); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warnw("Error while trying to fetch LogLevel.", "error", err)
		} else if err == nil {
			return value
		}
	}
	return c.getWithFallback("LogLevel", parseLogLevel).(LogLevel)
}

// SetLogLevel saves a runtime value for the default logger level
func (c *generalConfig) SetLogLevel(ctx context.Context, value string) error {
	if c.ORM == nil {
		return errors.New("SetLogLevel: No runtime store installed")
	}
	var ll LogLevel
	err := ll.Set(value)
	if err != nil {
		return err
	}
	return c.ORM.SetConfigStrValue(ctx, "LogLevel", ll.String())
}

// LogToDisk configures disk preservation of logs.
func (c *generalConfig) LogToDisk() bool {
	return c.viper.GetBool(EnvVarName("LogToDisk"))
}

// LogSQLStatements tells phoenix to log all SQL statements made using the default logger
func (c *generalConfig) LogSQLStatements() bool {
	if c.ORM != nil {
		logSqlStatements, err := c.ORM.GetConfigBoolValue("LogSQLStatements")
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warnw("Error while trying to fetch LogSQLStatements.", "error", err)
		} else if err == nil {
			return *logSqlStatements
		}
	}
	return c.viper.GetBool(EnvVarName("LogSQLStatements"))
}

// SetLogSQLStatements saves a runtime value for enabling/disabling logging all SQL statements on the default logger
func (c *generalConfig) SetLogSQLStatements(ctx context.Context, sqlEnabled bool) error {
	if c.ORM == nil {
		return errors.New("SetLogSQLStatements: No runtime store installed")
	}

	return c.ORM.SetConfigStrValue(ctx, "LogSQLStatements", strconv.FormatBool(sqlEnabled))
}

// LogSQLMigrations tells phoenix to log all SQL migrations made using the default logger
func (c *generalConfig) LogSQLMigrations() bool {
	return c.viper.GetBool(EnvVarName("LogSQLMigrations"))
}

// P2PListenIP is the ip that libp2p willl bind to and listen on
func (c *generalConfig) P2PListenIP() net.IP {
	return c.getWithFallback("P2PListenIP", parseIP).(net.IP)
}

// P2PListenPort is the port that libp2p will bind to and listen on
func (c *generalConfig) P2PListenPort() uint16 {
	if c.viper.IsSet(EnvVarName("P2PListenPort")) {
		return uint16(c.viper.GetUint32(EnvVarName("P2PListenPort")))
	}
	// Fast path in case it was already set
	c.randomP2PPortMtx.RLock()
	if c.randomP2PPort > 0 {
		c.randomP2PPortMtx.RUnlock()
		return c.randomP2PPort
	}
	c.randomP2PPortMtx.RUnlock()
	// Path for initial set
	c.randomP2PPortMtx.Lock()
	defer c.randomP2PPortMtx.Unlock()
	if c.randomP2PPort > 0 {
		return c.randomP2PPort
	}
	r, err := rand.Int(rand.Reader, big.NewInt(65535-1023))
	if err != nil {
		logger.Fatalw("unexpected error generating random port", "err", err)
	}
	randPort := uint16(r.Int64() + 1024)
	logger.Warnw(fmt.Sprintf("P2P_LISTEN_PORT was not set, listening on random port %d. A new random port will be generated on every boot, for stability it is recommended to set P2P_LISTEN_PORT to a fixed value in your environment", randPort), "p2pPort", randPort)
	c.randomP2PPort = randPort
	return c.randomP2PPort
}

// P2PListenPortRaw returns the raw string value of P2P_LISTEN_PORT
func (c *generalConfig) P2PListenPortRaw() string {
	return c.viper.GetString(EnvVarName("P2PListenPort"))
}

func (c *generalConfig) P2PAnnounceIP() net.IP {
	str := c.viper.GetString(EnvVarName("P2PAnnounceIP"))
	return net.ParseIP(str)
}

func (c *generalConfig) P2PAnnouncePort() uint16 {
	return uint16(c.viper.GetUint32(EnvVarName("P2PAnnouncePort")))
}

func (c *generalConfig) P2PDHTAnnouncementCounterUserPrefix() uint32 {
	return c.viper.GetUint32(EnvVarName("P2PDHTAnnouncementCounterUserPrefix"))
}

func (c *generalConfig) P2PPeerstoreWriteInterval() time.Duration {
	return c.getWithFallback("P2PPeerstoreWriteInterval", parseDuration).(time.Duration)
}

func (c *generalConfig) P2PPeerID() p2pkey.PeerID {
	var pid p2pkey.PeerID
	pidStr := c.viper.GetString(EnvVarName("P2PPeerID"))
	if pidStr != "" {
		err := pid.UnmarshalText([]byte(pidStr))
		if err != nil {
			logger.Error(errors.Wrapf(ErrInvalid, "P2P_PEER_ID is invalid %v", err))
		}
	}
	return pid
}

// P2PPeerIDRaw returns the string value of whatever P2P_PEER_ID was set to with no parsing
func (c *generalConfig) P2PPeerIDRaw() string {
	return c.viper.GetString(EnvVarName("P2PPeerID"))
}

func (c *generalConfig) P2PBootstrapPeers() ([]string, error) {
	if c.viper.IsSet(EnvVarName("P2PBootstrapPeers")) {
		bps := c.viper.GetStringSlice(EnvVarName("P2PBootstrapPeers"))
		if bps != nil {
			return bps, nil
		}
		return nil, errors.Wrap(ErrUnset, "P2P_BOOTSTRAP_PEERS")
	}
	return []string{}, nil
}

// P2PNetworkingStack returns the preferred networking stack for libocr
func (c *generalConfig) P2PNetworkingStack() (n ocrnetworking.NetworkingStack) {
	str := c.P2PNetworkingStackRaw()
	err := n.UnmarshalText([]byte(str))
	if err != nil {
		logger.Fatalf("P2PNetworkingStack failed to unmarshal '%s': %s", str, err)
	}
	return n
}

func (c *generalConfig) P2PNetworkingStackRaw() string {
	return c.viper.GetString(EnvVarName("P2PNetworkingStack"))
}

func (c *generalConfig) P2PV2ListenAddresses() []string {
	return c.viper.GetStringSlice(EnvVarName("P2PV2ListenAddresses"))
}

func (c *generalConfig) P2PV2AnnounceAddresses() []string {
	if c.viper.IsSet(EnvVarName("P2PV2AnnounceAddresses")) {
		return c.viper.GetStringSlice(EnvVarName("P2PV2AnnounceAddresses"))
	}
	return c.P2PV2ListenAddresses()
}

func (c *generalConfig) P2PV2AnnounceAddressesRaw() []string {
	return c.viper.GetStringSlice(EnvVarName("P2PV2AnnounceAddresses"))
}

func (c *generalConfig) P2PV2Bootstrappers() (locators []ocrtypes.BootstrapperLocator) {
	bootstrappers := c.P2PV2BootstrappersRaw()
	for _, s := range bootstrappers {
		var locator ocrtypes.BootstrapperLocator
		err := locator.UnmarshalText([]byte(s))
		if err != nil {
			logger.Fatalf("invalid format for bootstrapper '%s', got error: %s", s, err)
		}
		locators = append(locators, locator)
	}
	return
}

func (c *generalConfig) P2PV2BootstrappersRaw() []string {
	return c.viper.GetStringSlice(EnvVarName("P2PV2Bootstrappers"))
}

// P2PV2DeltaDial controls how far apart Dial attempts are
func (c *generalConfig) P2PV2DeltaDial() models.Duration {
	return models.MustMakeDuration(c.getWithFallback("P2PV2DeltaDial", parseDuration).(time.Duration))
}

// P2PV2DeltaReconcile controls how often a Reconcile message is sent to every peer.
func (c *generalConfig) P2PV2DeltaReconcile() models.Duration {
	return models.MustMakeDuration(c.getWithFallback("P2PV2DeltaReconcile", parseDuration).(time.Duration))
}

func (c *generalConfig) Port() uint16 {
	return c.getWithFallback("Port", parseUint16).(uint16)
}

func (c *generalConfig) HTTPServerWriteTimeout() time.Duration {
	return c.getWithFallback("HTTPServerWriteTimeout", parseDuration).(time.Duration)
}

// ReaperExpiration represents
func (c *generalConfig) ReaperExpiration() models.Duration {
	return models.MustMakeDuration(c.getWithFallback("ReaperExpiration", parseDuration).(time.Duration))
}

func (c *generalConfig) ReplayFromBlock() int64 {
	return c.viper.GetInt64(EnvVarName("ReplayFromBlock"))
}

func (c *generalConfig) RootDir() string {
	return c.getWithFallback("RootDir", parseHomeDir).(string)
}

// SecureCookies allows toggling of the secure cookies HTTP flag
func (c *generalConfig) SecureCookies() bool {
	return c.viper.GetBool(EnvVarName("SecureCookies"))
}

// SessionTimeout is the maximum duration that a user session can persist without any activity.
func (c *generalConfig) SessionTimeout() models.Duration {
	return models.MustMakeDuration(c.getWithFallback("SessionTimeout", parseDuration).(time.Duration))
}

// StatsPusherLogging toggles very verbose logging of raw messages for the StatsPusher (also telemetry)
func (c *generalConfig) StatsPusherLogging() bool {
	return c.getWithFallback("StatsPusherLogging", parseBool).(bool)
}

func (c *generalConfig) TLSCertPath() string {
	return c.viper.GetString(EnvVarName("TLSCertPath"))
}

// TLSHost represents the hostname to use for TLS clients. This should match
// the TLS certificate.
func (c *generalConfig) TLSHost() string {
	return c.viper.GetString(EnvVarName("TLSHost"))
}

func (c *generalConfig) TLSKeyPath() string {
	return c.viper.GetString(EnvVarName("TLSKeyPath"))
}

func (c *generalConfig) TLSPort() uint16 {
	return c.getWithFallback("TLSPort", parseUint16).(uint16)
}

// TLSRedirect forces TLS redirect for unencrypted connections
func (c *generalConfig) TLSRedirect() bool {
	return c.viper.GetBool(EnvVarName("TLSRedirect"))
}

// UnAuthenticatedRateLimit defines the threshold to which requests unauthenticated requests get limited
func (c *generalConfig) UnAuthenticatedRateLimit() int64 {
	return c.viper.GetInt64(EnvVarName("UnAuthenticatedRateLimit"))
}

// UnAuthenticatedRateLimitPeriod defines the period to which unauthenticated requests get limited
func (c *generalConfig) UnAuthenticatedRateLimitPeriod() models.Duration {
	return models.MustMakeDuration(c.getWithFallback("UnAuthenticatedRateLimitPeriod", parseDuration).(time.Duration))
}

func (c *generalConfig) TLSDir() string {
	return filepath.Join(c.RootDir(), "tls")
}

// KeyFile returns the path where the server key is kept
func (c *generalConfig) KeyFile() string {
	if c.TLSKeyPath() == "" {
		return filepath.Join(c.TLSDir(), "server.key")
	}
	return c.TLSKeyPath()
}

// CertFile returns the path where the server certificate is kept
func (c *generalConfig) CertFile() string {
	if c.TLSCertPath() == "" {
		return filepath.Join(c.TLSDir(), "server.crt")
	}
	return c.TLSCertPath()
}

// CreateProductionLogger returns a custom logger for the config's root
// directory and LogLevel, with pretty printing for stdout. If LOG_TO_DISK is
// false, the logger will only log to stdout.
func (c *generalConfig) CreateProductionLogger() *logger.Logger {
	return logger.CreateProductionLogger(c.RootDir(), c.JSONConsole(), c.LogLevel().Level, c.LogToDisk())
}

func (c *generalConfig) SessionSecret() ([]byte, error) {
	return c.secretGenerator.Generate(c.RootDir())
}

func (c *generalConfig) SessionOptions() sessions.Options {
	return sessions.Options{
		Secure:   c.SecureCookies(),
		HttpOnly: true,
		MaxAge:   86400 * 30,
	}
}

func (c *generalConfig) getWithFallback(name string, parser func(string) (interface{}, error)) interface{} {
	str := c.viper.GetString(EnvVarName(name))
	defaultValue, hasDefault := defaultValue(name)
	if str != "" {
		v, err := parser(str)
		if err == nil {
			return v
		}
		logger.Errorw(
			fmt.Sprintf("Invalid value provided for %s, falling back to default.", name),
			"value", str,
			"default", defaultValue,
			"error", err)
	}

	if !hasDefault {
		return zeroValue(name)
	}

	v, err := parser(defaultValue)
	if err != nil {
		log.Fatalf(`Invalid default for %s: "%s" (%s)`, name, defaultValue, err)
	}
	return v
}

func parseString(str string) (interface{}, error) {
	return str, nil
}

func parseAddress(str string) (interface{}, error) {
	if str == "" {
		return nil, nil
	} else if common.IsHexAddress(str) {
		val := common.HexToAddress(str)
		return &val, nil
	} else if i, ok := new(big.Int).SetString(str, 10); ok {
		val := common.BigToAddress(i)
		return &val, nil
	}
	return nil, fmt.Errorf("unable to parse '%s' into EIP55-compliant address", str)
}

func parsePhb(str string) (interface{}, error) {
	i, ok := new(assets.Phb).SetString(str, 10)
	if !ok {
		return i, fmt.Errorf("unable to parse '%v' into *assets.Phb(base 10)", str)
	}
	return i, nil
}

func parseLogLevel(str string) (interface{}, error) {
	var lvl LogLevel
	err := lvl.Set(str)
	return lvl, err
}

func parseUint16(s string) (interface{}, error) {
	v, err := strconv.ParseUint(s, 10, 16)
	return uint16(v), err
}

func parseUint32(s string) (interface{}, error) {
	v, err := strconv.ParseUint(s, 10, 32)
	return uint32(v), err
}

func parseUint64(s string) (interface{}, error) {
	v, err := strconv.ParseUint(s, 10, 64)
	return v, err
}

func parseF32(s string) (interface{}, error) {
	v, err := strconv.ParseFloat(s, 32)
	return v, err
}

func parseURL(s string) (interface{}, error) {
	return url.Parse(s)
}

func parseIP(s string) (interface{}, error) {
	return net.ParseIP(s), nil
}

func parseDuration(s string) (interface{}, error) {
	return time.ParseDuration(s)
}

func parseBool(s string) (interface{}, error) {
	return strconv.ParseBool(s)
}

func parseBigInt(str string) (interface{}, error) {
	i, ok := new(big.Int).SetString(str, 10)
	if !ok {
		return i, fmt.Errorf("unable to parse %v into *big.Int(base 10)", str)
	}
	return i, nil
}

func parseHomeDir(str string) (interface{}, error) {
	exp, err := homedir.Expand(str)
	if err != nil {
		return nil, err
	}
	return filepath.ToSlash(exp), nil
}

// LogLevel determines the verbosity of the events to be logged.
type LogLevel struct {
	zapcore.Level
}

type DatabaseBackupMode string

var (
	DatabaseBackupModeNone DatabaseBackupMode = "none"
	DatabaseBackupModeLite DatabaseBackupMode = "lite"
	DatabaseBackupModeFull DatabaseBackupMode = "full"
)

func parseDatabaseBackupMode(s string) (interface{}, error) {
	switch DatabaseBackupMode(s) {
	case DatabaseBackupModeNone, DatabaseBackupModeLite, DatabaseBackupModeFull:
		return DatabaseBackupMode(s), nil
	default:
		return "", fmt.Errorf("unable to parse %v into DatabaseBackupMode. Must be one of values: \"%s\", \"%s\", \"%s\"", s, DatabaseBackupModeNone, DatabaseBackupModeLite, DatabaseBackupModeFull)
	}
}
