package format

import (
	"github.com/brimsec/zq/pkg/test"
)

var JsonTypes = test.Internal{
	Name:         "format",
	Query:        "*",
	Input:        test.Trim(jsonInput),
	OutputFormat: "ndjson",
	Expected:     test.Trim(jsonExpected),
}

// This test covers serializing all the different zng types to json.
const jsonInput = `
#0:record[a:ip,a2:ip,b:bool,c:uint64,f:float64,i:int32,interval:duration,p:uint16,s:string,t:time]
0:[10.1.1.1;fe80::eef4:bbff:fe51:89ec;t;517;3.14159;18;60.0;443;Hello, world!;1578407783.487;]
`

const jsonExpected = `
{"a":"10.1.1.1","a2":"fe80::eef4:bbff:fe51:89ec","b":true,"c":517,"f":3.14159,"i":18,"interval":"60","p":443,"s":"Hello, world!","t":"1578407783.487"}
`
