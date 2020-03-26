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

	loopbackData, err := ioutil.ReadFile("suite/pcap/loopback.pcap")
	if err != nil {
		panic(err.Error())
	}
	Test5.Input[0].Data = string(loopbackData)

	// same as in.pcap but with snaplen = 0
	zData, err := ioutil.ReadFile("suite/pcap/zero.pcap")
	if err != nil {
		panic(err.Error())
	}
	Test6.Input[0].Data = string(zData)
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

var out3 = `
1584306020.727648
1584306020.727793
1584306020.727798
1584306020.727823
1584306020.727885
1584306020.727896
1584306020.728452
1584306020.728487
1584306020.730499
1584306020.73051
1584306020.730511
1584306020.730517
1584306020.730921
1584306020.730931
1584306028.07898
1584306028.078995
1584306028.079577
1584306028.079603
1584306030.033299
1584306030.033316
1584306030.033346
1584306030.033348
1584306030.033351
1584306030.033377
1584306031.477171
1584306031.477181
1584306031.50255
1584306031.502586`

// This test exercises the non-Ethernet link layer type ini loopback.pcap.
var Test5 = test.Shell{
	Name:   "pcap-loopback",
	Script: `pcap slice -r loopback.pcap 127.0.0.1:53586 127.0.0.1:9867 | pcap ts -w out3`,
	Input:  []test.File{test.File{Name: "loopback.pcap"}},
	Expected: []test.File{
		test.File{"out3", test.Trim(out3)},
	},
}

// same as Test1 but with zero.pcap which has snaplen=0
var Test6 = test.Shell{
	Name:   "pcap-snap-zero",
	Script: `pcap slice -r zero.pcap -from 1425567047.804914 -to 1425567432.792482 | pcap ts -w out1`,
	Input:  []test.File{test.File{Name: "zero.pcap"}},
	Expected: []test.File{
		test.File{"out1", test.Trim(out1)},
	},
}
