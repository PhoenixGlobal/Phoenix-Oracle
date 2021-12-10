package static

import (
	"fmt"
	"net/url"
	"time"

	uuid "github.com/satori/go.uuid"
)

var Version = "unset"

// Sha string "unset"
var Sha = "unset"

var InstanceUUID uuid.UUID

var InitTime time.Time

const (
	ExternalInitiatorAccessKeyHeader = "X-Phoenix-EA-AccessKey"
	ExternalInitiatorSecretHeader = "X-Phoenix-EA-Secret"
)

func init() {
	InitTime = time.Now()
	InstanceUUID = uuid.NewV4()
}

func buildPrettyVersion() string {
	if Version == "unset" {
		return " "
	}
	return fmt.Sprintf(" %s ", Version)
}

func SetConsumerName(uri *url.URL, name string) {
	q := uri.Query()

	applicationName := fmt.Sprintf("Phoenix%s| %s | %s", buildPrettyVersion(), name, InstanceUUID)
	if len(applicationName) > 63 {
		applicationName = applicationName[:63]
	}
	q.Set("application_name", applicationName)
	uri.RawQuery = q.Encode()
}

const (
	EvmMaxInFlightTransactionsWarningLabel = `WARNING: If this happens a lot, you may need to increase ETH_MAX_IN_FLIGHT_TRANSACTIONS to boost your node's transaction throughput, however you do this at your own risk. You MUST first ensure your ethereum node is configured not to ever evict local transactions that exceed this number otherwise the node can get permanently stuck`
	EvmMaxQueuedTransactionsLabel          = `WARNING: Hitting ETH_MAX_QUEUED_TRANSACTIONS is a sanity limit and should never happen under normal operation. This error is very unlikely to be a problem with Phoenix, and instead more likely to be caused by a problem with your eth node's connectivity. Check your eth node: it may not be broadcasting transactions to the network, or it might be overloaded and evicting Phoenix's transactions from its mempool. Increasing ETH_MAX_QUEUED_TRANSACTIONS is almost certainly not the correct action to take here unless you ABSOLUTELY know what you are doing, and will probably make things worse`
	EthNodeConnectivityProblemLabel        = `WARNING: If this happens a lot, it may be a sign that your eth node has a connectivity problem, and your transactions are not making it to any miners`
)
