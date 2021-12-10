package presenters

import (
	"strconv"
	"time"

	"PhoenixOracle/core/service/feedmanager"
)

type JobProposalResource struct {
	JAID
	Spec           string                  `json:"spec"`
	Status         feedmanager.JobProposalStatus `json:"status"`
	ExternalJobID  *string                 `json:"external_job_id"`
	FeedsManagerID string                  `json:"feeds_manager_id"`
	Multiaddrs     []string                `json:"multiaddrs"`
	CreatedAt      time.Time               `json:"createdAt"`
}

func (r JobProposalResource) GetName() string {
	return "job_proposals"
}

func NewJobProposalResource(jp feedmanager.JobProposal) *JobProposalResource {
	res := &JobProposalResource{
		JAID:           NewJAIDInt64(jp.ID),
		Status:         jp.Status,
		Spec:           jp.Spec,
		FeedsManagerID: strconv.FormatInt(jp.FeedsManagerID, 10),
		Multiaddrs:     jp.Multiaddrs,
		CreatedAt:      jp.CreatedAt,
	}

	if jp.ExternalJobID.Valid {
		uuid := jp.ExternalJobID.UUID.String()
		res.ExternalJobID = &uuid
	}

	return res
}

func NewJobProposalResources(jps []feedmanager.JobProposal) []JobProposalResource {
	rs := []JobProposalResource{}

	for _, jp := range jps {
		rs = append(rs, *NewJobProposalResource(jp))
	}

	return rs
}
