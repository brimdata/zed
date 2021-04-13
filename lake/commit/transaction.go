package commit

import (
	"context"
	"fmt"
	"io"

	"github.com/brimdata/zed/lake/commit/actions"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/segmentio/ksuid"
)

type Transaction struct {
	ID      ksuid.KSUID         `zng:"id"`
	Actions []actions.Interface `zng:"actions"`
}

func newTransaction(id ksuid.KSUID, capacity int) Transaction {
	return Transaction{
		ID:      id,
		Actions: make([]actions.Interface, 0, capacity),
	}
}

func NewCommitTxn(id ksuid.KSUID, date nano.Ts, author, message string, segments []segment.Reference) Transaction {
	txn := newTransaction(id, len(segments)+1)
	if date == 0 {
		date = nano.Now()
	}
	txn.appendAdds(segments, id)
	txn.AppendCommitMessage(id, date, author, message)
	return txn
}

func NewAddsTxn(id ksuid.KSUID, segments []segment.Reference) Transaction {
	txn := newTransaction(id, len(segments))
	txn.appendAdds(segments, id)
	return txn
}

func (t *Transaction) Append(action actions.Interface) {
	t.Actions = append(t.Actions, action)
}

func (t *Transaction) AppendCommitMessage(id ksuid.KSUID, date nano.Ts, author, message string) {
	t.Append(&actions.CommitMessage{
		ID:      id,
		Date:    date,
		Author:  author,
		Message: message,
	})
}

func (t *Transaction) appendAdds(segments []segment.Reference, id ksuid.KSUID) {
	for _, s := range segments {
		t.Append(&actions.Add{Commit: id, Segment: s})
	}
}

func (t Transaction) Serialize() ([]byte, error) {
	writer := actions.NewSerializer()
	for _, action := range t.Actions {
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
		return nil, fmt.Errorf("empty transaction")
	}
	return b, nil
}

func (t *Transaction) Deserialize(r io.Reader) error {
	reader := actions.NewDeserializer(r)
	for {
		action, err := reader.Read()
		if err != nil {
			return err
		}
		if action == nil {
			break
		}
		t.Append(action)
	}
	return nil
}

func LoadTransaction(ctx context.Context, id ksuid.KSUID, uri iosrc.URI) (Transaction, error) {
	r, err := iosrc.NewReader(ctx, uri)
	if err != nil {
		return Transaction{}, err
	}
	t := newTransaction(id, 0)
	err = t.Deserialize(r)
	if closeErr := r.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return Transaction{}, err
	}
	return t, nil
}
