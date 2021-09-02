package commits

import (
	"errors"
	"fmt"
	"io"

	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zngbytes"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

var ErrEmptyTransaction = errors.New("empty transaction")

type Object struct {
	Commit  ksuid.KSUID `zng:"commit"`
	Parent  ksuid.KSUID `zng:"parent"`
	Actions []Action    `zng:"actions"`
}

func NewObject(parent ksuid.KSUID, author, message string, retries int) *Object {
	commit := ksuid.New()
	o := &Object{
		Commit: commit,
		Parent: parent,
	}
	o.AppendCommit(commit, parent, retries, nano.Now(), author, message)
	return o
}

func NewAddsObject(parent ksuid.KSUID, retries int, author, message string, segments []segment.Reference) *Object {
	o := NewObject(parent, author, message, retries)
	o.appendAdds(segments)
	return o
}

/*
func NewDeletesObject(commit, parent ksuid.KSUID, ids []ksuid.KSUID) *Object {
	o := newObject(commit, parent, len(ids))
	for _, id := range ids {
		o.AppendDelete(id, retires)
	}
	return o
}
*/

func NewAddIndicesObject(parent ksuid.KSUID, author, message string, retries int, indices []*index.Reference) *Object {
	o := NewObject(parent, author, message, retries)
	for _, index := range indices {
		o.appendAddIndex(index)
	}
	return o
}

func (o *Object) Append(action Action) {
	o.Actions = append(o.Actions, action)
}

func (o *Object) AppendCommit(id, parent ksuid.KSUID, retries int, date nano.Ts, author, message string) {
	o.Append(&Commit{
		ID:      id,
		Parent:  parent,
		Retries: uint8(retries),
		Date:    date,
		Author:  author,
		Message: message,
	})
}

func (o *Object) appendAdds(segments []segment.Reference) {
	for _, s := range segments {
		o.Append(&Add{Commit: o.Commit, Segment: s})
	}
}

func (o *Object) appendAdd(s *segment.Reference) {
	o.Append(&Add{Commit: o.Commit, Segment: *s})
}

func (o *Object) AppendDelete(id ksuid.KSUID) {
	o.Append(&Delete{Commit: o.Commit, ID: id})
}

func (o *Object) appendAddIndex(i *index.Reference) {
	o.Append(&AddIndex{Commit: o.Commit, Index: *i})
}

func (o Object) Serialize() ([]byte, error) {
	writer := zngbytes.NewSerializer()
	writer.Decorate(zson.StylePackage)
	for _, action := range o.Actions {
		if err := writer.Write(action); err != nil {
			writer.Close()
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	b := writer.Bytes()
	if len(b) == 0 {
		return nil, ErrEmptyTransaction
	}
	return b, nil
}

func DecodeObject(r io.Reader) (*Object, error) {
	o := &Object{}
	reader := zngbytes.NewDeserializer(r, ActionTypes)
	for {
		entry, err := reader.Read()
		if err != nil {
			return nil, err
		}
		if entry == nil {
			break
		}
		action, ok := entry.(Action)
		if !ok {
			return nil, badObject(entry)
		}
		o.Append(action)
	}
	// Fill in the commit and parent IDs from the first record,
	// which must always be a Commit action.
	if len(o.Actions) > 0 {
		first, ok := o.Actions[0].(*Commit)
		if !ok {
			return nil, ErrBadCommitObject
		}
		o.Commit = first.ID
		o.Parent = first.Parent
	}
	return o, nil
}

func badObject(entry interface{}) error {
	return fmt.Errorf("internal error: corrupt commit object has unknown entry type %T", entry)
}
