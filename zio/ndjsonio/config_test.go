package ndjsonio

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTypeConfigValidate(t *testing.T) {
	testcases := []struct {
		name string
		in   string
		ok   bool
	}{
		{
			name: "Valid config",
			in: `
                   {
                       "descriptors": {
                           "capture_loss_log": [
                               {
                                   "type": "string",
                                   "name": "_path"
                               },
                               {
                                   "type": "time",
                                   "name": "ts"
                               },
                               {
                                   "name": "id",
                                   "type": {
                                       "type": "ip",
                                       "name": "orig_h"
                                    }
                               }
                           ]
                       },
                       "rules": [
                           {
                               "name": "_path",
                               "value": "capture_loss",
                               "descriptor": "capture_loss_log"
                           }
                       ]
                   }
                   `,
			ok: true,
		},
		{
			name: "Invalid descriptor in matching rule",
			in: `
                   {
                       "descriptors": {
                           "capture_loss_log": [
                               {
                                   "type": "string",
                                   "name": "_path"
                               },
                               {
                                   "type": "time",
                                   "name": "ts"
                               }
                           ]
                       },
                       "rules": [
                           {
                               "name": "_path",
                               "value": "capture_loss",
                               "descriptor": "capture_loss"
                           }
                       ]
                   }
                   `,
			ok: false,
		},
		{
			name: "Use of non-time ts field",
			in: `
                    {
                      "descriptors": {
                           "capture_loss_log": [
                               {
                                   "type": "string",
                                   "name": "_path"
                               },
                               {
                                   "type": "string",
                                   "name": "ts"
                               }
                           ]
                       },
                       "rules": [
                       ]
                   }
                   `,
			ok: false,
		},
		{
			name: "Matching rule refers to absent field",
			in: `
                   {
                       "descriptors": {
                           "capture_loss_log": [
                               {
                                   "type": "string",
                                   "name": "_path"
                               },
                               {
                                   "type": "time",
                                   "name": "ts"
                               }
                           ]
                       },
                       "rules": [
                           {
                               "name": "path",
                               "value": "capture_loss",
                               "descriptor": "capture_loss_log"
                           }
                       ]
                   }
                   `,
			ok: false,
		},
		{
			name: "Matching rule refers to absent field",
			in: `
                   {
                       "descriptors": {
                           "capture_loss_log": [
                               {
                                   "type": "string",
                                   "name": "_path"
                               },
                               {
                                   "type": "time",
                                   "name": "ts"
                               }
                           ]
                       },
                       "rules": [
                           {
                               "name": "path",
                               "value": "capture_loss",
                               "descriptor": "capture_loss_log"
                           }
                       ]
                   }
                   `,
			ok: false,
		},
		{
			name: "Descriptor without _path in first column",
			in: `
                   {
                       "descriptors": {
                           "capture_loss_log": [
                               {
                                   "type": "time",
                                   "name": "ts"
                               }
                           ]
                       },
                       "rules": [
                           {
                               "name": "path",
                               "value": "capture_loss",
                               "descriptor": "capture_loss_log"
                           }
                       ]
                   }
                   `,
			ok: false,
		},
		{
			name: "Descriptor with invalid structure",
			in: `
                   {
                       "descriptors": {
                           "capture_loss_log": [
                               {
                                   "type": "string",
                                   "name": "_path"
                               },
                               {
                                   "d": ["time","ts"]
                               }
                           ]
                       },
                       "rules": [
                           {
                               "name": "path",
                               "value": "capture_loss",
                               "descriptor": "capture_loss_log"
                           }
                       ]
                   }
                   `,
			ok: false,
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			tc := TypeConfig{}
			err := json.Unmarshal([]byte(c.in), &tc)
			require.NoError(t, err)
			err = tc.Validate()
			if c.ok {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
