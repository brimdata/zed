package pcapdemo

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/brimsec/zq/pcap"
	"github.com/brimsec/zq/pkg/nano"
)

// PacketSearch are the query string args to the packet endpoint when searching
// for packets within a connection 5-tuple.
type PacketSearch struct {
	Span    nano.Span
	Proto   string `validate:"required"`
	SrcHost string `validate:"required"`
	SrcPort *uint16
	DstHost string `validate:"required"`
	DstPort *uint16
}

// ToQuery transforms a packet search into a url.Values.
func (ps *PacketSearch) ToQuery() url.Values {
	tssec, tsns := ps.Span.Ts.Split()
	dursec := int(ps.Span.Dur / 1000000000)
	durns := int(int64(ps.Span.Dur) - int64(dursec)*1000000000)
	q := url.Values{}
	q.Add("ts_sec", strconv.Itoa(int(tssec)))
	q.Add("ts_ns", strconv.Itoa(int(tsns)))
	q.Add("duration_sec", strconv.Itoa(dursec))
	q.Add("duration_ns", strconv.Itoa(durns))
	q.Add("proto", ps.Proto)
	q.Add("src_host", ps.SrcHost)
	q.Add("dst_host", ps.DstHost)
	if ps.SrcPort != nil {
		q.Add("src_port", strconv.Itoa(int(*ps.SrcPort)))
	}
	if ps.DstPort != nil {
		q.Add("dst_port", strconv.Itoa(int(*ps.DstPort)))
	}
	return q
}

// FromQuery parses a query string and populates the receiver's values.
func (ps *PacketSearch) FromQuery(v url.Values) error {
	var err error
	var tsSec, tsNs, durSec, durNs int64
	if tsSec, err = strconv.ParseInt(v.Get("ts_sec"), 10, 64); err != nil {
		return err
	}
	if tsNs, err = strconv.ParseInt(v.Get("ts_ns"), 10, 64); err != nil {
		return err
	}
	if durSec, err = strconv.ParseInt(v.Get("duration_sec"), 10, 64); err != nil {
		return err
	}
	if durNs, err = strconv.ParseInt(v.Get("duration_ns"), 10, 64); err != nil {
		return err
	}
	if v.Get("src_port") != "" {
		p, err := strconv.ParseUint(v.Get("src_port"), 10, 16)
		if err != nil {
			return err
		}
		sp := uint16(p)
		ps.SrcPort = &sp
	}
	if v.Get("dst_port") != "" {
		p, err := strconv.ParseUint(v.Get("dst_port"), 10, 16)
		if err != nil {
			return err
		}
		sp := uint16(p)
		ps.DstPort = &sp
	}
	span := nano.Span{
		Ts:  nano.Unix(tsSec, tsNs),
		Dur: nano.Duration(durSec, durNs),
	}
	ps.Span = span
	ps.Proto = v.Get("proto")
	ps.SrcHost = v.Get("src_host")
	ps.DstHost = v.Get("dst_host")
	return err
}

func parse(url *url.URL) (*pcap.Search, error) {
	search := &PacketSearch{}
	if err := search.FromQuery(url.Query()); err != nil {
		return nil, err
	}
	return pcap.NewSearch(
		search.Span,
		search.Proto,
		search.SrcHost,
		search.SrcPort,
		search.DstHost,
		search.DstPort,
	)
}

// GetPackets is an endpoint that returns the packets for a given conn_id as a
// pcap file.
func HandleGet(w http.ResponseWriter, r *http.Request, spaceName string) {
	if r.Method != http.MethodGet {
		http.Error(w, "bad method", http.StatusBadRequest)
		return
	}
	pcapPath, pcapIndexPath, err := pcapFiles(spaceName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	search, err := parse(r.URL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	index, err := pcap.LoadIndex(pcapIndexPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	f, err := os.Open(pcapPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer f.Close()
	slicer, err := pcap.NewSlicer(f, index, search.Span())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.tcpdump.pcap")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s.pcap", search.ID()))
	err = search.Run(w, slicer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func HasPcaps(spaceName string) bool {
	return isFile(filepath.Join(".", spaceName, "packets.idx.json"))
}

func pcapFiles(spaceName string) (string, string, error) {
	//XXX demo placeholders
	pcapPath := filepath.Join(".", spaceName, "packets.pcap")
	indexPath := filepath.Join(".", spaceName, "packets.idx.json")
	if !isFile(pcapPath) || !isFile(indexPath) {
		return "", "", fmt.Errorf("%s: space has no pcaps", spaceName)
	}
	return pcapPath, indexPath, nil
}
