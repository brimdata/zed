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

type Transaction []actions.Interface

func NewCommitTxn(id ksuid.KSUID, date nano.Ts, author, message string, segments []segment.Reference) Transaction {
	txn := make(Transaction, 0, len(segments)+1)
	if date == 0 {
		date = nano.Now()
	}
	txn.appendAdds(segments, id)
	txn.AppendCommitMessage(id, date, author, message)
	return txn
}

func NewAddsTxn(id ksuid.KSUID, segments []segment.Reference) Transaction {
	txn := make(Transaction, 0, len(segments))
	txn.appendAdds(segments, id)
	return txn
}

func (t *Transaction) Append(action actions.Interface) {
	*t = append(*t, action)
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
	for _, action := range t {
		if err := writer.Write(action); err != nil {
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

func LoadTransaction(ctx context.Context, uri iosrc.URI) (Transaction, error) {
	r, err := iosrc.NewReader(ctx, uri)
	if err != nil {
		return nil, err
	}
	var t Transaction
	if err := t.Deserialize(r); err != nil {
		return nil, err
	}
	if err := r.Close(); err != nil {
		return nil, err
	}
	return t, nil
}
