package tests

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
			name: "zq-command only",
			markdown: `
~~~zq-command only
1234
~~~
~~~
other code block
~~~
`,
			strerror: "zq-command without a following zq-output"},
		{
			name: "zq-output only",
			markdown: `
~~~zq-output only
1234
~~~
~~~
other code block
~~~
`,
			strerror: "zq-output without a preceeding zq-command"},
		{
			name: "two commands",
			markdown: `
~~~zq-command 1
block 1
~~~
~~~zq-command 2
block 2
~~~
~~~zq-output 2
block 3
~~~
`,
			strerror: "subsequent zq-command after another zq-command"},
		{
			name: "two items",
			markdown: `
~~~zq-command 1
block 1
~~~
~~~zq-output 1
block 2
~~~
~~~zq-command 2
block 3
~~~
~~~zq-output 2
block 4
~~~
`,
			items: 2},
		{
			name: "headed output",
			markdown: `
~~~zq-command
block 1
~~~
~~~zq-output head:1
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
		OutputLineCount: 2,
	}).Run(t)
	require.NoError(t, err)
	assert.Equal(t, "1\n2\n", out)
}
