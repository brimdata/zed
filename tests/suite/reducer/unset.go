package reducer

import "github.com/brimsec/zq/pkg/test"

const unset = `
#0:record[x:uint64]
0:[-;]

#1:record[x:float64]
1:[-;]

#2:record[x:int64]
2:[-;]

#3:record[x:duration]
3:[-;]

#4:record[x:time]
4:[-;]
`

var UnsetAvg = test.Internal{
	Name:  "unset avg",
	Query: "avg(x)",
	Input: test.Trim(unset),
	Expected: test.Trim(`
#0:record[avg:float64]
0:[-;]
`),
}

var UnsetCount = test.Internal{
	Name:  "unset count",
	Query: "count(x)",
	Input: test.Trim(unset),
	Expected: test.Trim(`
#0:record[count:uint64]
0:[5;]
`),
}

var UnsetCountDistinct = test.Internal{
	Name:  "unset countdistinct",
	Query: "countdistinct(x)",
	Input: test.Trim(unset),
	Expected: test.Trim(`
#0:record[countdistinct:uint64]
0:[1;]
`),
}

var UnsetFirst = test.Internal{
	Name:  "unset first",
	Query: "first(x)",
	Input: test.Trim(unset),
	Expected: test.Trim(`
#0:record[first:uint64]
0:[-;]
`),
}

var UnsetLast = test.Internal{
	Name:  "unset last",
	Query: "last(x)",
	Input: test.Trim(unset),
	Expected: test.Trim(`
#0:record[last:time]
0:[-;]
`),
}

var UnsetMax = test.Internal{
	Name:  "unset max",
	Query: "max(x)",
	Input: test.Trim(unset),
	Expected: test.Trim(`
#0:record[max:uint64]
0:[-;]
`),
}

var UnsetMin = test.Internal{
	Name:  "unset min",
	Query: "min(x)",
	Input: test.Trim(unset),
	Expected: test.Trim(`
#0:record[min:uint64]
0:[-;]
`),
}

var UnsetSum = test.Internal{
	Name:  "unset sum",
	Query: "sum(x)",
	Input: test.Trim(unset), Expected: test.Trim(`
#0:record[sum:uint64]
0:[-;]
`),
}
