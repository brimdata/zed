package main

import (
	"fmt"
	"io"
	"math/rand"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

const NumRecords = 2

func main() {
	rand.Seed(1234)
	things := make([]Thing, NumRecords)
	for i := 0; i < NumRecords; i++ {
		things[i] = Make()
	}
	m := zson.NewZNGMarshaler()
	m.Decorate(zson.StyleSimple)
	val, err := m.Marshal(things)
	if err != nil {
		panic(err)
	}
	dumpVal(val)
	return
	m = zson.NewZNGMarshaler()
	m.Decorate(zson.StyleSimple)
	val, err = m.Marshal(things[:99])
	if err != nil {
		panic(err)
	}
	dumpVal(val)
}

func dumpVal(val *zed.Value) {
	zed.Walk(val.Type, val.Bytes, func(typ zed.Type, bytes zcode.Bytes) error {
		if zed.IsContainerType(typ) {
			fmt.Println("CONT", zson.FormatType(typ))
			return nil
		}
		fmt.Println("PRIM", zson.String(zed.NewValue(typ, bytes)))
		return nil
	})
}

type Thing interface {
	GetID() int64
}

func Make() Thing {
	switch rand.Intn(3) {
	case 0:
		return NewThingA()
	case 1:
		return NewThingB()
	case 2:
		return NewThingC()
	}
	panic("shouldn't happen")
}

type ThingA struct {
	ID    int64
	Name  string
	Stuff []int64
}

var id int64

func NewThingA() *ThingA {
	id++
	return &ThingA{
		ID:    id,
		Name:  NewName(),
		Stuff: NewInt64s(),
	}
}

func (t *ThingA) GetID() int64 {
	return t.ID
}

type ThingB struct {
	ID    int64
	Name  string
	Stuff []int8
}

func NewThingB() *ThingB {
	id++
	return &ThingB{
		ID:    id,
		Name:  NewName(),
		Stuff: NewInt8s(),
	}
}

func (t *ThingB) GetID() int64 {
	return t.ID
}

type ThingC struct {
	ID    int64
	Name  string
	Stuff []bool
}

func NewThingC() *ThingC {
	id++
	return &ThingC{
		ID:    id,
		Name:  NewName(),
		Stuff: NewBools(),
	}
}

func (t *ThingC) GetID() int64 {
	return t.ID
}

var names = []string{
	"Redding",
	"Burney",
	"Vancouver",
	"Victoria",
	"Berkeley",
	"Oakland",
	"Bryce",
	"Panguitch",
	"Boise",
	"McCall",
	"Jackson",
	"Ashton",
	"Yellowstone",
}

func NewName() string {
	return names[rand.Intn(len(names))]
}

func NewInt64s() []int64 {
	vals := make([]int64, rand.Intn(20))
	for i := 0; i < len(vals); i++ {
		vals[i] = int64(rand.Intn(1_000_000))
	}
	return vals
}

func NewInt8s() []int8 {
	spread := rand.Intn(80) + 10
	vals := make([]int8, rand.Intn(10))
	for i := 0; i < len(vals); i++ {
		vals[i] = int8(rand.Intn(spread) - spread/2)
	}
	return vals
}

func NewBools() []bool {
	vals := make([]bool, rand.Intn(10))
	for i := 0; i < len(vals); i++ {
		if rand.Intn(2) == 0 {
			vals[i] = true
		} else {
			vals[i] = false
		}
	}
	return vals
}

type NopCloser struct {
	io.Writer
}

func (*NopCloser) Close() error {
	return nil
}
