package commits

import (
	"fmt"

	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/segmentio/ksuid"
)

type Action interface {
	CommitID() ksuid.KSUID
	fmt.Stringer
}

var ActionTypes = []interface{}{
	Add{},
	AddIndex{},
	index.AddRule{},
	Delete{},
	index.DeleteRule{},
	index.TypeRule{},
	index.AggRule{},
	index.FieldRule{},
	Commit{},
}

type Add struct {
	Commit ksuid.KSUID `zng:"commit"`
	Object data.Object `zng:"object"`
}

var _ Action = (*Add)(nil)

func (a *Add) CommitID() ksuid.KSUID {
	return a.Commit
}

func (a *Add) String() string {
	return fmt.Sprintf("ADD %s", a.Object)
}

// Note that we store the number of retries in the final commit
// object.  This will allow easily introspection of optimistic
// locking problems under high commit load by simply issuing
// a meta-query and looking at the retry count in the persisted
// commit objects.  If/when this is a problem, we could add
// pessimistic locking mechanisms alongside the optimistic approach.

type Commit struct {
	ID      ksuid.KSUID `zng:"id"`
	Parent  ksuid.KSUID `zng:"parent"`
	Retries uint8       `zng:"retries"`
	Author  string      `zng:"author"`
	Date    nano.Ts     `zng:"date"`
	Message string      `zng:"message"`
}

func (c *Commit) CommitID() ksuid.KSUID {
	return c.ID
}

func (c *Commit) String() string {
	//XXX need to format Message field for single line
	return fmt.Sprintf("COMMIT %s -> %s %s %s %s", c.ID, c.Parent, c.Date, c.Author, c.Message)
}

type Delete struct {
	Commit ksuid.KSUID `zng:"commit"`
	ID     ksuid.KSUID `zng:"id"`
}

func (d *Delete) CommitID() ksuid.KSUID {
	return d.Commit
}

func (d *Delete) String() string {
	return "DEL " + d.ID.String()
}

type AddIndex struct {
	Commit ksuid.KSUID  `zng:"commit"`
	Object index.Object `zng:"object"`
}

func (a *AddIndex) String() string {
	return fmt.Sprintf("ADD_INDEX %s", a.Object)
}

func (a *AddIndex) CommitID() ksuid.KSUID {
	return a.Commit
}
