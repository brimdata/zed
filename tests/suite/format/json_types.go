package format

import (
	"github.com/mccanne/zq/pkg/test"
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
#0:record[a:addr,a2:addr,b:bool,c:count,d:double,e:enum,i:int,interval:interval,p:port,s:string,t:time]
0:[10.1.1.1;fe80::eef4:bbff:fe51:89ec;t;517;3.14159;foo;18;60.0;443;Hello, world!;1578407783.487;]
`

const jsonExpected = `
{"a":"10.1.1.1","a2":"fe80::eef4:bbff:fe51:89ec","b":true,"c":517,"d":3.14159,"e":"foo","i":18,"interval":60000000000,"p":443,"s":"Hello, world!","t":1578407783487000000}
`
