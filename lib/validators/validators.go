package services

import (
	"fmt"
	"regexp"
	"strings"

	"PhoenixOracle/core/assets"
	"PhoenixOracle/db"
	"PhoenixOracle/db/models"
	"PhoenixOracle/db/orm"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func ValidateBridgeTypeNotExist(bt *models.BridgeTypeRequest, store *db.Store) error {
	fe := models.NewJSONAPIErrors()
	bridge, err := store.ORM.FindBridge(bt.Name)
	if err != nil && err != gorm.ErrRecordNotFound {
		fe.Add(fmt.Sprintf("Error determining if bridge type %v already exists", bt.Name))
	} else if (bridge != models.BridgeType{}) {
		fe.Add(fmt.Sprintf("Bridge Type %v already exists", bt.Name))
	}
	return fe.CoerceEmptyToNil()
}

func ValidateBridgeType(bt *models.BridgeTypeRequest, store *db.Store) error {
	fe := models.NewJSONAPIErrors()
	if len(bt.Name.String()) < 1 {
		fe.Add("No name specified")
	}
	if _, err := models.NewTaskType(bt.Name.String()); err != nil {
		fe.Merge(err)
	}
	u := bt.URL.String()
	if len(strings.TrimSpace(u)) == 0 {
		fe.Add("URL must be present")
	}
	if bt.MinimumContractPayment != nil &&
		bt.MinimumContractPayment.Cmp(assets.NewPhb(0)) < 0 {
		fe.Add("MinimumContractPayment must be positive")
	}
	return fe.CoerceEmptyToNil()
}

var (
	externalInitiatorNameRegexp = regexp.MustCompile("^[a-zA-Z0-9-_]+$")
)

func ValidateExternalInitiator(
	exi *models.ExternalInitiatorRequest,
	store *db.Store,
) error {
	fe := models.NewJSONAPIErrors()
	if len([]rune(exi.Name)) == 0 {
		fe.Add("No name specified")
	} else if !externalInitiatorNameRegexp.MatchString(exi.Name) {
		fe.Add("Name must be alphanumeric and may contain '_' or '-'")
	} else if _, err := store.FindExternalInitiatorByName(exi.Name); err == nil {
		fe.Add(fmt.Sprintf("Name %v already exists", exi.Name))
	} else if err != orm.ErrorNotFound {
		return errors.Wrap(err, "validating external initiator")
	}
	return fe.CoerceEmptyToNil()
}
