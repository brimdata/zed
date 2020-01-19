package resolver

import (
	"sync"

	"github.com/mccanne/zq/zng"
)

type Translator struct {
	mu sync.Mutex
	Slice
	inputCtx  *Context
	outputCtx *Context
}

func NewTranslator(in, out *Context) *Translator {
	return &Translator{
		inputCtx:  in,
		outputCtx: out,
	}
}

// Lookup implements zng.Resolver
func (t *Translator) Lookup(id int) *zng.TypeRecord {
	t.mu.Lock()
	defer t.mu.Unlock()
	outputType := t.lookup(id)
	if outputType == nil {
		inputType := t.inputCtx.Lookup(id)
		if inputType == nil {
			return nil
		}
		outputType = t.outputCtx.LookupByColumns(inputType.Columns)
		t.enter(id, outputType)
	}
	return outputType
}
