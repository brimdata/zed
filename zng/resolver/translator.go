package resolver

import (
	"sync"

	"github.com/brimdata/zq/zng"
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
func (t *Translator) Lookup(id int) (zng.Type, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	outputType := t.Slice.Lookup(id)
	var err error
	if outputType == nil {
		inputType := t.inputCtx.Lookup(id)
		if inputType == nil {
			return nil, nil
		}
		outputType, err = t.outputCtx.TranslateTypeRecord(inputType)
		if err != nil {
			return nil, err
		}
		t.Slice.Enter(id, outputType)
	}
	return outputType, nil
}
