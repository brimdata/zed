package mdtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseMarkdown(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		markdown string
		strerror string
		inputs   map[string]string
		tests    []*Test
	}{
		{
			name: "mdtest-input",
			markdown: `
~~~mdtest-input filename
1234
~~~
`,
			inputs: map[string]string{"filename": "1234\n"},
		},

		{
			name: "mdtest-input without file name",
			markdown: `
~~~mdtest-input
1234
~~~
`,
			strerror: "mdtest-input without file name",
		},
		{
			name: "mdtest-input with duplicate file name",
			markdown: `
~~~mdtest-input filenaame
1234
~~~
~~~mdtest-input filenaame
1234
~~~
`,
			strerror: "mdtest-input with duplicate file name",
		},
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
			strerror: "line 2: unpaired mdtest-command",
		},
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
			strerror: "line 2: unpaired mdtest-output",
		},
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
			strerror: "line 2: unpaired mdtest-command",
		},
		{
			name: "two tests",
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
			tests: []*Test{
				{Command: "block 1\n", Dir: "1", Expected: "block 2\n", Line: 2},
				{Command: "block 3\n", Dir: "2", Expected: "block 4\n", Line: 8},
			},
		},
		{
			name: "headed output",
			markdown: `
~~~mdtest-command
block 1
~~~
~~~mdtest-output head
block 2
...
~~~
`,
			tests: []*Test{
				{Command: "block 1\n", Expected: "block 2\n", Line: 2, Head: true},
			},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			inputs, tests, err := parseMarkdown([]byte(tc.markdown))
			if tc.strerror != "" {
				assert.EqualError(t, err, tc.strerror)
			} else {
				assert.Equal(t, inputs, tc.inputs)
				assert.Equal(t, tests, tc.tests)
			}
		})
	}
}
