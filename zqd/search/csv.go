package search

import (
	"fmt"
	"net/http"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/csvio"
)

// CSVOutput implements the Output inteface and writes csv encoded-output
// directly to the client as text/csv.
type CSVOutput struct {
	response http.ResponseWriter
	writer   *csvio.Writer
}

func NewCSVOutput(response http.ResponseWriter, ctrl bool) *CSVOutput {
	o := &CSVOutput{
		response: response,
		writer:   csvio.NewWriter(zio.NopCloser(response), true, false),
	}
	return o
}

func (r *CSVOutput) flush() {
	r.writer.Flush()
	r.response.(http.Flusher).Flush()
}

func (r *CSVOutput) Collect() interface{} {
	return "TBD" //XXX
}

func (r *CSVOutput) SendBatch(cid int, batch zbuf.Batch) error {
	for _, rec := range batch.Records() {
		if err := r.writer.Write(rec); err != nil {
			// Embed an error in the csv output.  We can't report
			// an http error because we already started successfully
			// streaming records.
			msg := fmt.Sprintf("query error: %s\n", err)
			r.response.Write([]byte(msg))
			return err
		}
	}
	batch.Unref()
	r.flush()
	return nil
}

func (r *CSVOutput) End(ctrl interface{}) error {
	return nil
}

func (r *CSVOutput) SendControl(ctrl interface{}) error {
	return nil
}

func (r *CSVOutput) ContentType() string {
	return MimeTypeCSV
}
