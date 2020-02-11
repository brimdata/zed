package sort

import (
	"github.com/brimsec/zq/pkg/test"
)

var Internal1 = test.Internal{
	Name:         "sort1",
	Query:        "* | sort x",
	Input:        test.Trim(in1),
	OutputFormat: "zng",
	Expected:     test.Trim(out1),
}

const in1 = `
#0:record[x:set[ip]]
0:[[192.168.1.10;192.168.1.2;192.179.1.1;]]
0:[[192.168.1.10;192.168.1.2;]]
`

const out1 = `
#0:record[x:set[ip]]
0:[[192.168.1.10;192.168.1.2;]]
0:[[192.168.1.10;192.168.1.2;192.179.1.1;]]
`

var Internal2 = test.Internal{
	Name:         "sort2",
	Query:        "* | sort x",
	Input:        test.Trim(in2),
	OutputFormat: "zng",
	Expected:     test.Trim(out2),
}

const in2 = `
#0:record[x:set[ip]]
0:[[192.168.1.10;192.168.1.2;]]
0:[[192.168.1.10;192.168.0.2;]]
`

const out2 = `
#0:record[x:set[ip]]
0:[[192.168.1.10;192.168.0.2;]]
0:[[192.168.1.10;192.168.1.2;]]
`

const in3 = `
#0:record[TTLs:array[duration],count:count]
0:[[0;]18;]
0:[[0;0;]4;]
0:[[0;0;0;]3;]
0:[[0;0;0;0;]1;]
0:[[0;0;0;0;0;]6;]
0:[[0;0;0;0;0;0;]4;]
0:[[0;0;0;0;0;0;0;]7;]
0:[[0;0;0;0;0;0;0;0;]9;]
0:[[0;2;2;2;2;2;2;2;2;]3;]
0:[[0;270;]1;]
0:[[0;40;40;40;40;40;40;]1;]
0:[[0;260;]6;]
0:[[0;39;39;39;]1;]
0:[[0;46;46;46;46;46;46;]1;]
0:[[0;53;]1;]
0:[[0;34;]1;]
0:[[0;418;]1;]
0:[[0;60;]1;]
0:[[0;221;221;221;221;221;221;221;221;]1;]
0:[[0;100;]1;]
0:[[0;24;]2;]
0:[[0;2917;]1;]
0:[[0;608;]2;]
0:[[0;14799;1;]1;]
0:[[0;3094;5;]1;]
0:[[0;2961;]1;]
`

const out3 = `
#0:record[TTLs:array[duration],count:count]
0:[[0;]18;]
0:[[0;0;]4;]
0:[[0;0;0;]3;]
0:[[0;0;0;0;]1;]
0:[[0;0;0;0;0;]6;]
0:[[0;0;0;0;0;0;]4;]
0:[[0;0;0;0;0;0;0;]7;]
0:[[0;0;0;0;0;0;0;0;]9;]
0:[[0;2;2;2;2;2;2;2;2;]3;]
0:[[0;24;]2;]
0:[[0;34;]1;]
0:[[0;39;39;39;]1;]
0:[[0;40;40;40;40;40;40;]1;]
0:[[0;46;46;46;46;46;46;]1;]
0:[[0;53;]1;]
0:[[0;60;]1;]
0:[[0;100;]1;]
0:[[0;221;221;221;221;221;221;221;221;]1;]
0:[[0;260;]6;]
0:[[0;270;]1;]
0:[[0;418;]1;]
0:[[0;608;]2;]
0:[[0;2917;]1;]
0:[[0;2961;]1;]
0:[[0;3094;5;]1;]
0:[[0;14799;1;]1;]
`

var Internal3 = test.Internal{
	Name:         "sort3",
	Query:        "* | sort TTLs",
	Input:        test.Trim(in3),
	OutputFormat: "zng",
	Expected:     test.Trim(out3),
}

const in4 = `
#0:record[s:string]
#1:record[notS:string]
0:[b;]
0:[c;]
0:[a;]
0:[-;]
1:[bleah;]
`

const out4_1 = `
#0:record[s:string]
0:[a;]
0:[b;]
0:[c;]
#1:record[notS:string]
1:[bleah;]
0:[-;]
`

var Internal4_1 = test.Internal{
	Name:         "sort4_1",
	Query:        "sort -nulls last s",
	Input:        test.Trim(in4),
	OutputFormat: "zng",
	Expected:     test.Trim(out4_1),
}

const out4_2 = `
#0:record[notS:string]
0:[bleah;]
#1:record[s:string]
1:[-;]
1:[a;]
1:[b;]
1:[c;]
`

var Internal4_2 = test.Internal{
	Name:         "sort4_2",
	Query:        "sort -nulls first s",
	Input:        test.Trim(in4),
	OutputFormat: "zng",
	Expected:     test.Trim(out4_2),
}

const out4_3 = `
#0:record[s:string]
0:[c;]
0:[b;]
0:[a;]
#1:record[notS:string]
1:[bleah;]
0:[-;]
`

var Internal4_3 = test.Internal{
	Name:         "sort4_3",
	Query:        "sort -r -nulls last s",
	Input:        test.Trim(in4),
	OutputFormat: "zng",
	Expected:     test.Trim(out4_3),
}

const out4_4 = `
#0:record[notS:string]
0:[bleah;]
#1:record[s:string]
1:[-;]
1:[c;]
1:[b;]
1:[a;]
`

var Internal4_4 = test.Internal{
	Name:         "sort4_4",
	Query:        "sort -r -nulls first s",
	Input:        test.Trim(in4),
	OutputFormat: "zng",
	Expected:     test.Trim(out4_4),
}
