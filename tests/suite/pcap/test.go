package pcap

import (
	"io/ioutil"

	"github.com/brimsec/zq/pkg/test"
)

var pcapData []byte

func init() {
	pcapData, err := ioutil.ReadFile("suite/pcap/in.pcap")
	if err != nil {
		panic(err.Error())
	}
	Test1.Input[0].Data = string(pcapData)
	Test2.Input[0].Data = string(pcapData)
	ngData, err := ioutil.ReadFile("suite/pcap/ng.pcap")
	if err != nil {
		panic(err.Error())
	}
	Test3.Input[0].Data = string(ngData)
	Test4.Input[0].Data = string(ngData)
}

// XXX note these tests don't test the pcap slice index file since there
// isn't a way yet to look at whether or not the index was used

// test simple time range of odd-sorted pcap
var Test1 = test.Shell{
	Name:   "pcap-command",
	Script: `pcap slice -r in.pcap -from 1425567047.804914 -to 1425567432.792482 | pcap ts -w out1`,
	Input:  []test.File{test.File{Name: "in.pcap"}},
	Expected: []test.File{
		test.File{"out1", test.Trim(out1)},
	},
}

var out1 = `
1425567432.792481
1425567047.804914
`

// 80.239.174.91.443 (aka [::ffff:50ef:ae5b]:443) > 192.168.0.51.33773

// test simple flow extraction
var Test2 = test.Shell{
	Name:   "pcap-command",
	Script: `pcap slice -r in.pcap [::ffff:50ef:ae5b]:443 192.168.0.51:33773 | pcap ts -w out2`,
	Input:  []test.File{test.File{Name: "in.pcap"}},
	Expected: []test.File{
		test.File{"out2", test.Trim(out2)},
	},
}

var out2 = `
1425567047.803929
1425567047.804906
1425567047.804914
`

// test ng version
var Test3 = test.Shell{
	Name:   "pcap-command",
	Script: `pcap slice -r ng.pcap -from 1425567047.804914 -to 1425567432.792482 | pcap ts -w out1`,
	Input:  []test.File{test.File{Name: "ng.pcap"}},
	Expected: []test.File{
		test.File{"out1", test.Trim(out1)},
	},
}

var Test4 = test.Shell{
	Name:   "pcap-command",
	Script: `pcap slice -r ng.pcap [::ffff:50ef:ae5b]:443 192.168.0.51:33773 | pcap ts -w out2`,
	Input:  []test.File{test.File{Name: "ng.pcap"}},
	Expected: []test.File{
		test.File{"out2", test.Trim(out2)},
	},
}
