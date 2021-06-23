package mdtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

type markdownunittest struct {
	name     string
	markdown string
	strerror string
	items    int
	inputs   int
}

func TestCollectExamples(t *testing.T) {
	t.Parallel()
	tests := []markdownunittest{
		{
			name: "zq-input",
			markdown: `
~~~zq-input filename
1234
~~~
`,
			inputs: 1},

		{
			name: "zq-input without file name",
			markdown: `
~~~zq-input
1234
~~~
`,
			strerror: "zq-input without file name"},
		{
			name: "zq-input with duplicate file name",
			markdown: `
~~~zq-input filenaame
1234
~~~
~~~zq-input filenaame
1234
~~~
`,
			strerror: "zq-input with duplicate file name"},
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
~~~mdtest-output head
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
			examples, inputs, err := CollectExamples(doc, source)
			if testcase.strerror != "" {
				assert.EqualError(t, err, testcase.strerror)
			}
			if testcase.items != 0 {
				assert.Len(t, examples, testcase.items)
			}
			if testcase.inputs != 0 {
				assert.Len(t, inputs, testcase.inputs)
			}
		})
	}
}

func TestInputs(t *testing.T) {
	t.Parallel()
	out, err := (&ZQExampleTest{
		Command: "cat one two",
		Inputs: map[string]string{
			"one": "1\n",
			"two": "2\n",
		},
	}).Run(t)
	require.NoError(t, err)
	assert.Equal(t, "1\n2\n", out)
}
