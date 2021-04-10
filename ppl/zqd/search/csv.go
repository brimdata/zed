package search

import (
	"fmt"
	"net/http"

	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/csvio"
)

// CSVOutput implements the Output inteface and writes csv encoded-output
// directly to the client as text/csv.
type CSVOutput struct {
	response http.ResponseWriter
	writer   *csvio.Writer
}

func NewCSVOutput(response http.ResponseWriter, ctrl bool) *CSVOutput {
	return &CSVOutput{
		response: response,
		writer:   csvio.NewWriter(zio.NopCloser(response), csvio.WriterOpts{UTF8: true}),
	}
}

func (r *CSVOutput) Collect() interface{} {
	return "TBD" //XXX
}

func (r *CSVOutput) SendBatch(cid int, batch zbuf.Batch) error {
	for _, rec := range batch.Records() {
		if err := r.writer.Write(rec); err != nil {
			r.error(err)
			return err
		}
	}
	batch.Unref()
	err := r.writer.Flush()
	if err != nil {
		r.error(err)
	}
	if f, ok := r.response.(http.Flusher); ok {
		f.Flush()
	}
	return err
}

// error embeds an error in the CSV output.  We can't report an HTTP error
// because we already started successfully streaming records.
func (r *CSVOutput) error(err error) {
	fmt.Fprintf(r.response, "query error: %s\n", err)
}

func (r *CSVOutput) End(ctrl interface{}) error {
	return r.writer.Close()
}

func (r *CSVOutput) SendControl(ctrl interface{}) error {
	return nil
}

func (r *CSVOutput) ContentType() string {
	return MimeTypeCSV
}
