package zed_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
	"github.com/kr/pretty"
)

const NumRecords = 100
const NumRuns = 20

func dump(b []byte) {
	f, err := os.OpenFile("t.zng", os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	f.Write(b)
	f.Close()
}

func main() {
	/*f, err := os.Create("PROF")
	if err != nil {
		panic(err)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()*/

	zedThings := make([]Thing, NumRecords)
	for i := 0; i < NumRecords; i++ {
		zedThings[i] = Make()
	}
	//fmt.Println(MarshalZSON(zedThings))
	dump(MarshalZNG(zedThings))
	return
	jsonThings := convertToJSON(zedThings)
	run("MarshalJSON", func() {
		MarshalJSON(jsonThings)
	})
	encodedJSON := MarshalJSON(jsonThings)
	run("UnmarshalJSON", func() {
		UnmarshalJSON(encodedJSON)
	})
	fmt.Println("JSON size", len(encodedJSON))
	run("MashalZSON", func() {
		MarshalZSON(zedThings)
	})
	encodedZSON := MarshalZSON(zedThings)
	run("UnmarshalZSON", func() {
		UnmarshalZSON(encodedZSON)
	})
	fmt.Println("ZSON size", len(encodedZSON))
	run("MarshalZNG", func() {
		MarshalZNG(zedThings)
	})
	encodedZNG := MarshalZNG(zedThings)
	run("UnmarshalZNG", func() {
		UnmarshalZNG(encodedZNG)
	})
	fmt.Println("ZNG size", len(encodedZNG))
}

func run(which string, marshal func()) {
	start := time.Now()
	for i := 0; i < NumRuns; i++ {
		marshal()
	}
	fmt.Println(which, time.Since(start))
}

func MarshalJSON(things []JSONThing) []byte {
	for i := 0; i < 10; i++ {
		pretty.Println(things[i])
	}
	b, err := json.Marshal(things)
	if err != nil {
		panic(err)
	}
	return b
}

func MarshalZSON(vals []Thing) string {
	m := zson.NewMarshaler()
	m.Decorate(zson.StyleSimple)
	s, err := m.Marshal(vals)
	if err != nil {
		panic(err)
	}
	return s
}

func UnmarshalZSON(s string) []Thing {
	u := zson.NewUnmarshaler()
	u.Bind(ThingA{}, ThingB{}, ThingC{})
	var things []Thing
	if err := u.Unmarshal(s, &things); err != nil {
		panic(err)
	}
	return things
}

func MarshalZNG(things []Thing) []byte {
	m := zson.NewZNGMarshaler()
	var out bytes.Buffer
	writer := zngio.NewWriter(&NopCloser{&out})
	m.Decorate(zson.StyleSimple)
	val, err := m.Marshal(things)
	if err != nil {
		panic(err)
	}
	dumpVal(val)
	if err := writer.Write(val); err != nil {
		panic(err)
	}
	if err := writer.Close(); err != nil {
		panic(err)
	}
	return out.Bytes()
}

func dumpVal(val *zed.Value) {
	zed.Walk(val.Type, val.Bytes, func(typ zed.Type, bytes zcode.Bytes) error {
		if zed.IsContainerType(typ) {
			fmt.Println(zson.FormatType(typ))
			return nil
		}
		fmt.Println(zson.String(zed.NewValue(typ, bytes)))
		return nil
	})
	panic("x")
}

func UnmarshalZNG(b []byte) []Thing {
	u := zson.NewZNGUnmarshaler()
	u.Bind(ThingA{}, ThingB{}, ThingC{})
	reader := zngio.NewReader(zed.NewContext(), bytes.NewReader(b))
	val, err := reader.Read()
	if err != nil {
		panic(err)
	}
	var things []Thing
	if err := u.Unmarshal(val, &things); err != nil {
		panic(err)
	}
	return things
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

type JSONThing interface {
	GetID() int64
}

func convertToJSON(things []Thing) []JSONThing {
	out := make([]JSONThing, 0, len(things))
	for _, thing := range things {
		switch thing := thing.(type) {
		case *ThingA:
			out = append(out, &JSONThingA{
				Kind:  "A",
				ID:    thing.ID,
				Name:  thing.Name,
				Stuff: thing.Stuff,
			})
		case *ThingB:
			out = append(out, &JSONThingB{
				Kind:  "B",
				ID:    thing.ID,
				Name:  thing.Name,
				Stuff: thing.Stuff,
			})
		case *ThingC:
			out = append(out, &JSONThingC{
				Kind:  "C",
				ID:    thing.ID,
				Name:  thing.Name,
				Stuff: thing.Stuff,
			})
		default:
			panic("bad type in convertToJSON")
		}
	}
	return out
}

type JSONThingA struct {
	Kind  string
	ID    int64
	Name  string
	Stuff []int64
}

func (j *JSONThingA) GetID() int64 {
	return j.ID
}

type JSONThingB struct {
	Kind  string
	ID    int64
	Name  string
	Stuff []int8
}

func (j *JSONThingB) GetID() int64 {
	return j.ID
}

type JSONThingC struct {
	Kind  string
	ID    int64
	Name  string
	Stuff []bool
}

func (j *JSONThingC) GetID() int64 {
	return j.ID
}

type Header struct {
	Kind string
}

func UnmarshalJSON(b []byte) []JSONThing {
	var rawMessages []*json.RawMessage
	if err := json.Unmarshal(b, &rawMessages); err != nil {
		panic(err)
	}
	var things []JSONThing
	for _, message := range rawMessages {
		var hdr Header
		if err := json.Unmarshal(*message, &hdr); err != nil {
			panic(err)
		}
		var thing JSONThing
		switch hdr.Kind {
		case "A":
			var a JSONThingA
			if err := json.Unmarshal(*message, &a); err != nil {
				panic(err)
			}
			thing = &a
		case "B":
			var b JSONThingB
			if err := json.Unmarshal(*message, &b); err != nil {
				panic(err)
			}
			thing = &b
		case "C":
			var c JSONThingC
			if err := json.Unmarshal(*message, &c); err != nil {
				panic(err)
			}
			thing = &c
		default:
			fmt.Println(string(*message))
			panic("bad kind in JSON unmarshaler: " + hdr.Kind)
		}
		things = append(things, thing)
	}
	return things
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
	spread := rand.Intn(200) + 1
	vals := make([]int8, rand.Intn(10))
	for i := 0; i < len(vals); i++ {
		vals[i] = int8(rand.Intn(spread))
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
