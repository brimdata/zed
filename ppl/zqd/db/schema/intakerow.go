package schema

import (
	"fmt"

	"github.com/brimdata/zq/api"
	"github.com/brimdata/zq/ppl/zqd/auth"
	"github.com/segmentio/ksuid"
)

type IntakeRow struct {
	tableName     struct{}      `pg:"intake"` // This is needed so the postgres orm knows the correct table name
	ID            api.IntakeID  `json:"id"`
	Name          string        `json:"name"`
	Shaper        string        `json:"shaper"`
	TargetSpaceID api.SpaceID   `json:"target_space_id"`
	TenantID      auth.TenantID `json:"tenant_id"`
}

func NewIntakeID() api.IntakeID {
	id := ksuid.New()
	return api.IntakeID(fmt.Sprintf("intake_%s", id.String()))
}
