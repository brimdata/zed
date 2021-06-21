package mdtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

type markdownunittest struct {
	name     string
	markdown string
	strerror string
	items    int
}

func TestCollectExamples(t *testing.T) {
	t.Parallel()
	tests := []markdownunittest{
		{
			name: "mdtest-command only",
			markdown: `
~~~mdtest-command only
1234
~~~
~~~
other code block
~~~
`,
			strerror: "mdtest-command without a following mdtest-output"},
		{
			name: "mdtest-output only",
			markdown: `
~~~mdtest-output only
1234
~~~
~~~
other code block
~~~
`,
			strerror: "mdtest-output without a preceeding mdtest-command"},
		{
			name: "two commands",
			markdown: `
~~~mdtest-command 1
block 1
~~~
~~~mdtest-command 2
block 2
~~~
~~~mdtest-output 2
block 3
~~~
`,
			strerror: "subsequent mdtest-command after another mdtest-command"},
		{
			name: "two items",
			markdown: `
~~~mdtest-command 1
block 1
~~~
~~~mdtest-output 1
block 2
~~~
~~~mdtest-command 2
block 3
~~~
~~~mdtest-output 2
block 4
~~~
`,
			items: 2},
		{
			name: "headed output",
			markdown: `
~~~mdtest-command
block 1
~~~
~~~mdtest-output head:1
block 2
block 2 continued
~~~
`,
			items: 1},
	}
	for _, testcase := range tests {
		testcase := testcase
		t.Run(testcase.name, func(t *testing.T) {
			source := []byte(testcase.markdown)
			reader := text.NewReader(source)
			parser := goldmark.DefaultParser()
			doc := parser.Parse(reader)
			examples, err := CollectExamples(doc, source)
			if testcase.strerror != "" {
				assert.EqualError(t, err, testcase.strerror)
			}
			if testcase.items != 0 {
				assert.Equal(t, len(examples), testcase.items)
			}
		})
	}
}
