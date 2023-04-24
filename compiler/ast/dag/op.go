package dag

// This module is derived from the GO AST design pattern in
// https://golang.org/pkg/go/ast/
//
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import (
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
	"github.com/segmentio/ksuid"
)

type Op interface {
	OpNode()
}

var PassOp = &Pass{Kind: "Pass"}

// Ops

type (
	Combine struct {
		Kind string `json:"kind" unpack:""`
	}
	Cut struct {
		Kind  string       `json:"kind" unpack:""`
		Args  []Assignment `json:"args"`
		Quiet bool         `json:"quiet"`
	}
	Drop struct {
		Kind string `json:"kind" unpack:""`
		Args []Expr `json:"args"`
	}
	Explode struct {
		Kind string `json:"kind" unpack:""`
		Args []Expr `json:"args"`
		Type string `json:"type"`
		As   Expr   `json:"as"`
	}
	Filter struct {
		Kind string `json:"kind" unpack:""`
		Expr Expr   `json:"expr"`
	}
	Fuse struct {
		Kind string `json:"kind" unpack:""`
	}
	Head struct {
		Kind  string `json:"kind" unpack:""`
		Count int    `json:"count"`
	}
	Join struct {
		Kind     string       `json:"kind" unpack:""`
		Style    string       `json:"style"`
		LeftKey  Expr         `json:"left_key"`
		RightKey Expr         `json:"right_key"`
		Args     []Assignment `json:"args"`
	}
	Load struct {
		Kind    string `json:"kind" unpack:""`
		Pool    string `json:"pool"`
		Author  string `json:author`
		Message string `json:message`
		Meta    string `json:meta`
	}
	Merge struct {
		Kind  string      `json:"kind" unpack:""`
		Expr  Expr        `json:"expr"`
		Order order.Which `json:"order"`
	}
	Parallel struct {
		Kind string `json:"kind" unpack:""`
		Ops  []Op   `json:"ops"`
		Any  bool   `json:"any"`
	}
	Pass struct {
		Kind string `json:"kind" unpack:""`
	}
	Put struct {
		Kind string       `json:"kind" unpack:""`
		Args []Assignment `json:"args"`
	}
	Rename struct {
		Kind string       `json:"kind" unpack:""`
		Args []Assignment `json:"args"`
	}
	Scope struct {
		Kind   string      `json:"kind" unpack:""`
		Consts []Def       `json:"consts"`
		Funcs  []*Func     `json:"funcs"`
		Body   *Sequential `json:"body"`
	}
	Sequential struct {
		Kind string `json:"kind" unpack:""`
		Ops  []Op   `json:"ops"`
	}
	Shape struct {
		Kind string `json:"kind" unpack:""`
	}
	Sort struct {
		Kind       string      `json:"kind" unpack:""`
		Args       []Expr      `json:"args"`
		Order      order.Which `json:"order"`
		NullsFirst bool        `json:"nullsfirst"`
	}
	Summarize struct {
		Kind         string       `json:"kind" unpack:""`
		Limit        int          `json:"limit"`
		Keys         []Assignment `json:"keys"`
		Aggs         []Assignment `json:"aggs"`
		InputSortDir int          `json:"input_sort_dir,omitempty"`
		PartialsIn   bool         `json:"partials_in,omitempty"`
		PartialsOut  bool         `json:"partials_out,omitempty"`
	}
	Switch struct {
		Kind  string `json:"kind" unpack:""`
		Expr  Expr   `json:"expr"`
		Cases []Case `json:"cases"`
	}
	Tail struct {
		Kind  string `json:"kind" unpack:""`
		Count int    `json:"count"`
	}
	Top struct {
		Kind  string `json:"kind" unpack:""`
		Limit int    `json:"limit"`
		Args  []Expr `json:"args"`
		Flush bool   `json:"flush"`
	}
	Over struct {
		Kind  string      `json:"kind" unpack:""`
		Defs  []Def       `json:"defs"`
		Exprs []Expr      `json:"exprs"`
		Vars  []Def       `json:"vars"`
		Body  *Sequential `json:"body"`
	}
	Uniq struct {
		Kind  string `json:"kind" unpack:""`
		Cflag bool   `json:"cflag"`
	}
	Yield struct {
		Kind  string `json:"kind" unpack:""`
		Exprs []Expr `json:"exprs"`
	}
)

// Input structure

type (
	Lister struct {
		Kind      string      `json:"kind" unpack:""`
		Pool      ksuid.KSUID `json:"pool"`
		Commit    ksuid.KSUID `json:"commit"`
		KeyPruner Expr        `json:"key_pruner"`
	}
	Slicer struct {
		Kind string `json:"kind" unpack:""`
	}
	SeqScan struct {
		Kind      string      `json:"kind" unpack:""`
		Pool      ksuid.KSUID `json:"pool"`
		Filter    Expr        `json:"filter"`
		KeyPruner Expr        `json:"key_pruner"`
	}
	Deleter struct {
		Kind      string      `json:"kind" unpack:""`
		Pool      ksuid.KSUID `json:"pool"`
		Where     Expr        `json:"where"`
		KeyPruner Expr        `json:"key_pruner"`
	}

	// Leaf sources

	FileScan struct {
		Kind    string        `json:"kind" unpack:""`
		Path    string        `json:"path"`
		Format  string        `json:"format"`
		SortKey order.SortKey `json:"sort_key"`
		Filter  Expr          `json:"filter"`
	}
	HTTPScan struct {
		Kind    string        `json:"kind" unpack:""`
		URL     string        `json:"url"`
		Format  string        `json:"format"`
		SortKey order.SortKey `json:"sort_key"`
	}
	PoolScan struct {
		Kind   string      `json:"kind" unpack:""`
		ID     ksuid.KSUID `json:"id"`
		Commit ksuid.KSUID `json:"commit"`
	}
	PoolMetaScan struct {
		Kind string      `json:"kind" unpack:""`
		ID   ksuid.KSUID `json:"id"`
		Meta string      `json:"meta"`
	}
	CommitMetaScan struct {
		Kind      string      `json:"kind" unpack:""`
		Pool      ksuid.KSUID `json:"pool"`
		Commit    ksuid.KSUID `json:"commit"`
		Meta      string      `json:"meta"`
		Tap       bool        `json:"tap"`
		KeyPruner Expr        `json:"key_pruner"`
	}
	LakeMetaScan struct {
		Kind string `json:"kind" unpack:""`
		Meta string `json:"meta"`
	}
)

var LakeMetas = map[string]struct{}{
	"branches":    {},
	"index_rules": {},
	"pools":       {},
}

var PoolMetas = map[string]struct{}{
	"branches": {},
}

var CommitMetas = map[string]struct{}{
	"indexes":    {},
	"log":        {},
	"objects":    {},
	"partitions": {},
	"rawlog":     {},
	"vectors":    {},
}

func (*FileScan) OpNode()       {}
func (*HTTPScan) OpNode()       {}
func (*PoolScan) OpNode()       {}
func (*LakeMetaScan) OpNode()   {}
func (*PoolMetaScan) OpNode()   {}
func (*CommitMetaScan) OpNode() {}

func (*Lister) OpNode()  {}
func (*Slicer) OpNode()  {}
func (*SeqScan) OpNode() {}
func (*Deleter) OpNode() {}

// Various Op fields

type (
	Case struct {
		Expr Expr `json:"expr"`
		Op   Op   `json:"op"`
	}
	Def struct {
		Name string `json:"name"`
		Expr Expr   `json:"expr"`
	}
)

func (*Sequential) OpNode() {}
func (*Scope) OpNode()      {}
func (*Parallel) OpNode()   {}
func (*Switch) OpNode()     {}
func (*Sort) OpNode()       {}
func (*Cut) OpNode()        {}
func (*Drop) OpNode()       {}
func (*Head) OpNode()       {}
func (*Tail) OpNode()       {}
func (*Pass) OpNode()       {}
func (*Filter) OpNode()     {}
func (*Uniq) OpNode()       {}
func (*Summarize) OpNode()  {}
func (*Top) OpNode()        {}
func (*Put) OpNode()        {}
func (*Rename) OpNode()     {}
func (*Fuse) OpNode()       {}
func (*Join) OpNode()       {}
func (*Shape) OpNode()      {}
func (*Explode) OpNode()    {}
func (*Over) OpNode()       {}
func (*Yield) OpNode()      {}
func (*Merge) OpNode()      {}
func (*Load) OpNode()       {}
func (*Combine) OpNode()    {}

// NewFilter returns a filter node for e.
func NewFilter(e Expr) *Filter {
	return &Filter{
		Kind: "Filter",
		Expr: e,
	}
}

func (seq *Sequential) Prepend(front Op) {
	seq.Ops = append([]Op{front}, seq.Ops...)
}

func (seq *Sequential) Append(op Op) {
	seq.Ops = append(seq.Ops, op)
}

func (seq *Sequential) Delete(at, length int) {
	seq.Ops = append(seq.Ops[0:at], seq.Ops[at+length:]...)
}

func FanIn(op Op) int {
	switch op := op.(type) {
	case *Sequential:
		return FanIn(op.Ops[0])
	case *Join:
		return 2
	}
	return 1
}

func (t *This) String() string {
	return field.Path(t.Path).String()
}
