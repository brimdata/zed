package index

/*

type Table struct {
	store *kvs.Store
}

var ErrBadTable = errors.New("system error: corrupt index rule in key-value store")

func (t *Table) Lookup(ctx context.Context, id ksuid.KSUID) (*Index, error) {
	vals, err := t.store.Values(ctx)
	if err != nil {
		return nil, err
	}
	for _, v := range vals {
		index, ok := v.(*Index)
		if !ok {
			return nil, ErrBadTable
		}
		if index.ID == id {
			return index, nil
		}
	}
	return nil, errors.New("index rule not found")
}

//XXX update comment
// Add checks if the table already has an equivalent Index and if it does not
// returns Indices with the Index appended to it. Returns a non-nil Index pointer
// if an equivalent Index is found.
func (t *Table) Add(ctx context.Context, index Index) error {
	err := t.store.Set(ctx, index.Name, &index)
	if err == kvs.ErrKeyExists {
		err = fmt.Errorf("index rule named %q already exists", index.Name)
	}
	return err
}

//XXX update comment
// LookupDelete checks the Indices list for a index matching the provided ID
// returning the deleted index if found.
func (t *Table) Delete(ctx context.Context, id ksuid.KSUID) error {
        index, err := t.Lookup(ctx, id)
        if err != nil {
                return err
        }
        t.store.MatchAndDelete(ctx, func (key string, val interface{}) bool {
                target, ok := val.(*Index)
                //XXX this is goofy... not all
                if !ok || (index.Name != "" && index.Name != target.Name) {
                        return false
                }
                if
                        return
                }
                return target.Equivalent(index)
                 key == index.Name &&
        })
	if i := indices.indexOf(id); i >= 0 {
		index := indices[i]
		return append(indices[:i], indices[i+1:]...), &index
	}
	return indices, nil
}

func (indices Indices) IDs() []ksuid.KSUID {
	ids := make([]ksuid.KSUID, len(indices))
	for k, index := range indices {
		ids[k] = index.ID
	}
	return ids
}

*/
