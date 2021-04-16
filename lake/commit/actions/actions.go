package actions

import (
	"fmt"

	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/segmentio/ksuid"
)

type Interface interface {
	CommitID() ksuid.KSUID
	fmt.Stringer
}

var actions = []interface{}{
	Add{},
	Delete{},
	CommitMessage{},
}

type Add struct {
	Commit  ksuid.KSUID       `zng:"commit"`
	Segment segment.Reference `zng:"segment"`
}

func (a *Add) CommitID() ksuid.KSUID {
	return a.Commit
}

func (a *Add) String() string {
	return fmt.Sprintf("ADD %s", a.Segment)
}

type CommitMessage struct {
	ID      ksuid.KSUID `zng:"id"`
	Author  string      `zng:"author"`
	Date    nano.Ts     `zng:"date"`
	Message string      `zng:"message"`
}

func (c *CommitMessage) CommitID() ksuid.KSUID {
	return c.ID
}

func (c *CommitMessage) String() string {
	return fmt.Sprintf("COMMIT %s %s %s %s", c.ID, c.Date, c.Author, c.Message)
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
