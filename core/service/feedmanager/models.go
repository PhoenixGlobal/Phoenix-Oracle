package feedmanager

import (
	"time"

	"PhoenixOracle/util/crypto"
	"github.com/lib/pq"
	uuid "github.com/satori/go.uuid"
	"gopkg.in/guregu/null.v4"
)

const (
	JobTypeFluxMonitor       = "fluxmonitor"
	JobTypeOffchainReporting = "ocr"
)

type FeedsManager struct {
	ID        int64
	Name      string
	URI       string
	PublicKey crypto.PublicKey
	JobTypes  pq.StringArray `gorm:"type:text[]"`

	IsOCRBootstrapPeer bool

	OCRBootstrapPeerMultiaddr null.String

	IsConnectionActive bool

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (FeedsManager) TableName() string {
	return "feeds_managers"
}

type JobProposalStatus string

const (
	JobProposalStatusPending  JobProposalStatus = "pending"
	JobProposalStatusApproved JobProposalStatus = "approved"
	JobProposalStatusRejected JobProposalStatus = "rejected"
)

type JobProposal struct {
	ID int64

	RemoteUUID uuid.UUID
	Spec       string
	Status     JobProposalStatus

	ExternalJobID  uuid.NullUUID
	FeedsManagerID int64
	Multiaddrs     pq.StringArray `gorm:"type:text[]"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
