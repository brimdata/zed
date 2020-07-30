package parquetio

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/xitongsys/parquet-go/common"
	"github.com/xitongsys/parquet-go/compress"
	"github.com/xitongsys/parquet-go/layout"
	"github.com/xitongsys/parquet-go/parquet"
	"github.com/xitongsys/parquet-go/source"
)

// Helper to avoid sprinkling fmt.Printf() calls around the code.
// Enable debug statements by uncommenting the call to Printf()
func debugf(msg string, args ...interface{}) {
	//fmt.Printf(msg, args...)
}

// valueReader is the interface used for anything that can iterate over
// a series of values of one of the parquet primitive types.  Note that
// not all methods can be used on all implementations of this interface.
type valueReader interface {
	nextBoolean() bool
	nextInt32() int32
	nextInt64() int64
	nextFloat() float64
	nextDouble() float64
	nextByteArray() []byte
}

// columnIterator reads and emits all the values from an individual
// column inside a parquet file.  This implementation is not yet complete,
// it only handles PLAIN and PLAIN_DICTIONARY encodings for a few
// primitive data types.
type columnIterator struct {
	name   string
	footer *parquet.FileMetaData
	file   source.ParquetFile

	maxRepetitionLevel int32
	maxDefinitionLevel int32

	rowGroup int

	groupRead    int
	groupTotal   int
	thriftReader *thrift.TBufferedTransport
	colMetadata  *parquet.ColumnMetaData

	pageRead  int
	pageTotal int

	rlReader  *hybridReader
	dlReader  *hybridReader
	valReader valueReader

	// only used for PageType_DICTIONARY_PAGE
	int64Dict     []int64
	floatDict     []float64
	byteArrayDict [][]byte
	// XXX need other types
}

func newColumnIterator(name string, footer *parquet.FileMetaData, file source.ParquetFile, maxRL, maxDL int32) *columnIterator {
	return &columnIterator{
		name:               name,
		footer:             footer,
		file:               file,
		maxRepetitionLevel: maxRL,
		maxDefinitionLevel: maxDL,
	}
}

func (i *columnIterator) clearDictionaries() {
	i.byteArrayDict = nil
	i.int64Dict = nil
	i.floatDict = nil
	// XXX clear others as they are added
}

// loadOnePage reads the next page that is part of this column.
func (i *columnIterator) loadOnePage() (*parquet.PageHeader, []byte, error) {
	// If we've reached the end of a row group (or if we haven't
	// yet read the first row group), find the right offset inside
	// the parquet file for this column in the next row group.
	if i.groupRead == i.groupTotal {
		if i.rowGroup >= len(i.footer.RowGroups) {
			return nil, nil, io.EOF
		}
		rg := i.footer.RowGroups[i.rowGroup]
		i.rowGroup++

		i.groupRead = 0
		i.groupTotal = int(rg.NumRows)

		var col *parquet.ColumnChunk
		for _, c := range rg.Columns {
			if c.MetaData.PathInSchema[0] == i.name {
				col = c
				break
			}
		}
		if col == nil {
			return nil, nil, fmt.Errorf("cannot find ColumnChunk for %s", i.name)
		}

		if col.FilePath != nil {
			return nil, nil, fmt.Errorf("ColumnChunk for %s refers to external file", i.name)
		}

		i.colMetadata = col.MetaData
		offset := i.colMetadata.DataPageOffset
		if i.colMetadata.DictionaryPageOffset != nil {
			offset = *i.colMetadata.DictionaryPageOffset
		}
		size := i.colMetadata.TotalCompressedSize

		// XXX
		i.thriftReader = source.ConvertToThriftReader(i.file, offset, size)

		i.clearDictionaries()
	}

	// Pages within a row group are sequential in the file.
	// The thriftReader object keeps track of the file offset,
	// for this column.  Read the header for the next page from it.
	header, err := layout.ReadPageHeader(i.thriftReader)
	if err != nil {
		return nil, nil, err
	}

	raw := make([]byte, header.GetCompressedPageSize())
	if _, err = i.thriftReader.Read(raw); err != nil {
		return nil, nil, err
	}

	page, err := compress.Uncompress(raw, i.colMetadata.GetCodec())
	if err != nil {
		return nil, nil, err
	}

	return header, page, nil
}

// ensureDataPage loads a new data page if the current page has been
// completely read (or if no page has yet been loaded).  This may include
// loading and parsing dictionary pages, but when this function returns,
// the caller may assume there are valid values available in this struct's
// data structures to decode.
func (i *columnIterator) ensureDataPage() error {
	if i.pageRead < i.pageTotal {
		return nil
	}

	for {
		header, page, err := i.loadOnePage()
		if err != nil {
			return err
		}

		debugf("read page type %s for %s\n", header.GetType(), i.name)
		switch header.GetType() {
		case parquet.PageType_DICTIONARY_PAGE:
			if err := i.loadDictionaryPage(header, page); err != nil {
				return err
			}

		case parquet.PageType_DATA_PAGE:
			return i.initializeDataPage(header, page)

		default:
			return fmt.Errorf("unhandled page type %s", header.GetType())
		}
	}
}

func (i *columnIterator) loadDictionaryPage(header *parquet.PageHeader, buf []byte) error {
	n := int(header.DictionaryPageHeader.GetNumValues())
	switch i.colMetadata.GetType() {
	case parquet.Type_INT64:
		i.int64Dict = make([]int64, n)
		r := plainReader{buf}
		for j := 0; j < n; j++ {
			i.int64Dict[j] = r.nextInt64()
		}

	case parquet.Type_DOUBLE:
		i.floatDict = make([]float64, n)
		r := plainReader{buf}
		for j := 0; j < n; j++ {
			i.floatDict[j] = r.nextDouble()
		}

	case parquet.Type_BYTE_ARRAY:
		i.byteArrayDict = make([][]byte, n)
		r := plainReader{buf}
		for j := 0; j < n; j++ {
			i.byteArrayDict[j] = r.nextByteArray()
		}

	default:
		debugf("skipping dictionary page for type %s\n", i.colMetadata.GetType())
	}
	return nil
}

// grabLenDenotedBuf extracts a chunk of data that represents
// repetition level (RL) or definition level (DL) data inside a data page.
// This consists of a 4 byte integer that represents the size of the
// encoded data followed by the actual encoded data.  Returns the
// length-denoted encoded data and the total number of bytes that were
// required from the original buffer.
func grabLenDenotedBuf(buf []byte) ([]byte, int, error) {
	if len(buf) < 4 {
		return nil, 0, fmt.Errorf("buffer is too short (%d)", len(buf))
	}
	ln := binary.LittleEndian.Uint32(buf[:4])
	total := int(4 + ln)
	if len(buf) < total {
		return nil, 0, fmt.Errorf("buffer is too short (%d, need %d)", len(buf), total)
	}
	return buf[4:total], total, nil
}

// initializeDataPage sets up the rlReader, dlReader, and valReader
// members of this struct for a new data page.
func (i *columnIterator) initializeDataPage(header *parquet.PageHeader, buf []byte) error {
	i.pageRead = 0
	i.pageTotal = int(header.DataPageHeader.GetNumValues())

	// Per https://github.com/apache/parquet-format#data-pages
	// a page contains the optional RL data, the optional DL data,
	// and the encoded values, all back-to-back in the page.
	if i.maxRepetitionLevel > 0 {
		width := int(common.BitNum(uint64(i.maxRepetitionLevel)))
		hbuf, n, err := grabLenDenotedBuf(buf)
		if err != nil {
			return err
		}
		buf = buf[n:]
		i.rlReader = newHybridReader(hbuf, width)
	} else {
		i.rlReader = nil
	}

	if i.maxDefinitionLevel > 0 {
		width := int(common.BitNum(uint64(i.maxDefinitionLevel)))
		hbuf, n, err := grabLenDenotedBuf(buf)
		if err != nil {
			return err
		}
		buf = buf[n:]
		i.dlReader = newHybridReader(hbuf, width)
	} else {
		i.dlReader = nil
	}

	enc := header.DataPageHeader.GetEncoding()
	typ := i.colMetadata.GetType()
	switch enc {
	case parquet.Encoding_PLAIN:
		if typ == parquet.Type_BOOLEAN {
			i.valReader = &plainBooleanReader{buf: buf}
		} else {
			i.valReader = &plainReader{buf}
		}
	case parquet.Encoding_PLAIN_DICTIONARY:
		switch typ {
		case parquet.Type_INT64:
			i.valReader = newDictionaryInt64Reader(buf, i.int64Dict)

		case parquet.Type_DOUBLE:
			i.valReader = newDictionaryDoubleReader(buf, i.floatDict)

		case parquet.Type_BYTE_ARRAY:
			i.valReader = newDictionaryByteArrayReader(buf, i.byteArrayDict)
		default:
			debugf("skipping dictionary page of type %s\n", typ)
			i.valReader = &nullReader{}
		}
	default:
		debugf("skipping data page with encoding %s\n", enc)
		i.valReader = &nullReader{}
	}
	return nil
}

// peekDL returns the definition level (DL) for the next value on this
// page without advancing the iterator
func (i *columnIterator) peekDL() (int32, error) {
	if err := i.ensureDataPage(); err != nil {
		return 0, err
	}
	if i.dlReader == nil {
		return 0, nil
	}
	return int32(i.dlReader.peekInt64()), nil
}

// peekRL returns the repetition level (RL) for the next value on this
// page without advancing the iterator
func (i *columnIterator) peekRL() (int32, error) {
	if err := i.ensureDataPage(); err != nil {
		return 0, err
	}
	if i.rlReader == nil {
		return 0, nil
	}
	return int32(i.rlReader.peekInt64()), nil
}

// advance counter, grab rl and dl.  (to keep everything consistent,
// the caller should also advance valReader when this is called if the
// dl value indicates a value is present.)
func (i *columnIterator) commonNext() (int32, int32) {
	if err := i.ensureDataPage(); err != nil {
		if err == io.EOF {
			return 0, 0
		} else {
			panic(err)
		}
	}
	i.pageRead++

	var rl, dl int32
	if i.rlReader != nil {
		rl = int32(i.rlReader.nextInt64())
	}
	if i.dlReader != nil {
		dl = int32(i.dlReader.nextInt64())
	}

	if rl == i.maxRepetitionLevel {
		i.groupRead++
	}

	return rl, dl
}

func (i *columnIterator) nextBoolean() (bool, int32, int32) {
	rl, dl := i.commonNext()
	var v bool
	if dl == i.maxDefinitionLevel {
		v = i.valReader.nextBoolean()
	}
	return v, rl, dl
}

func (i *columnIterator) nextInt32() (int32, int32, int32) {
	rl, dl := i.commonNext()
	var v int32
	if dl == i.maxDefinitionLevel {
		v = i.valReader.nextInt32()
	}
	return v, rl, dl
}

func (i *columnIterator) nextInt64() (int64, int32, int32) {
	rl, dl := i.commonNext()
	var v int64
	if dl == i.maxDefinitionLevel {
		v = i.valReader.nextInt64()
	}
	return v, rl, dl
}

func (i *columnIterator) nextFloat() (float64, int32, int32) {
	rl, dl := i.commonNext()
	var v float64
	if dl == i.maxDefinitionLevel {
		v = i.valReader.nextFloat()
	}
	return v, rl, dl
}

func (i *columnIterator) nextDouble() (float64, int32, int32) {
	rl, dl := i.commonNext()
	var v float64
	if dl == i.maxDefinitionLevel {
		v = i.valReader.nextDouble()
	}
	return v, rl, dl
}

func (i *columnIterator) nextByteArray() ([]byte, int32, int32) {
	rl, dl := i.commonNext()
	var v []byte
	if dl == i.maxDefinitionLevel {
		v = i.valReader.nextByteArray()
	}
	return v, rl, dl
}

// hybridReader decodes 64-bit inntegers encoded using the hybrid
// RLE/bit-packing encoding described at:
// https://github.com/apache/parquet-format/blob/master/Encodings.md#run-length-encoding--bit-packing-hybrid-rle--3
type hybridReader struct {
	buf   []byte
	width int
	vals  []int64
}

func newHybridReader(buf []byte, width int) *hybridReader {
	return &hybridReader{buf, width, nil}
}

func makeMask(bits int) uint64 {
	return uint64(1<<bits) - 1
}

func (r *hybridReader) fillVals() {
	if len(r.vals) > 0 {
		return
	}
	hdr, n := binary.Uvarint(r.buf)
	if n == 0 {
		panic("could not decode varint")
	}
	r.buf = r.buf[n:]
	if hdr&1 == 0 {
		// RLE encoded
		raw := []byte{0, 0, 0, 0}
		bytes := (r.width + 7) / 8
		for i := 0; i < bytes; i++ {
			raw[i] = r.buf[i]
		}
		val := int32(binary.LittleEndian.Uint32(raw))
		r.buf = r.buf[bytes:]

		n := int(hdr >> 1)
		if cap(r.vals) >= n {
			r.vals = r.vals[:n]
		} else {
			r.vals = make([]int64, n, n)
		}
		for i := 0; i < n; i++ {
			r.vals[i] = int64(val)
		}
	} else {
		// bit packed
		groups := int(hdr >> 1)
		n := groups * 8
		if cap(r.vals) >= n {
			r.vals = r.vals[:n]
		} else {
			r.vals = make([]int64, n, n)
		}

		mask := makeMask(r.width)
		var iv uint64
		havebits := 0

		if len(r.buf) < (groups * r.width) {
			panic(fmt.Sprintf("need %d bytes for packed but have %d", groups*r.width, len(r.buf)))
		}
		bi := 0
		for i := 0; i < n; i++ {
			for havebits < r.width {
				iv |= uint64(r.buf[bi]) << havebits
				havebits += 8
				bi++
			}
			r.vals[i] = int64(iv & mask)
			iv >>= r.width
			havebits -= r.width
		}
		r.buf = r.buf[r.width*groups:]
	}
}

func (r *hybridReader) peekInt64() int64 {
	r.fillVals()
	return r.vals[0]
}

func (r *hybridReader) nextInt64() int64 {
	r.fillVals()
	ret := r.vals[0]
	r.vals = r.vals[1:]
	return ret
}

// nullReader repeatedly returns a default value for some data type.
// This is a development tool that allows us to "handle" unsupported
// encodings without panicing or failing.  Eventually this should be
// removed.
type nullReader struct {
}

func (r *nullReader) nextBoolean() bool {
	return false
}

func (r *nullReader) nextInt32() int32 {
	return 0
}

func (r *nullReader) nextInt64() int64 {
	return 0
}

func (r *nullReader) nextFloat() float64 {
	return 0
}

func (r *nullReader) nextDouble() float64 {
	return 0
}

func (r *nullReader) nextByteArray() []byte {
	return nil
}

// plainBooleanReader decodes BOOLEAN-typed values encoded with the
// parquet PLAIN encoding described at:
// https://github.com/apache/parquet-format/blob/master/Encodings.md#plain-plain--0
type plainBooleanReader struct {
	buf     []byte
	current byte
	bits    int
}

func (r *plainBooleanReader) nextBoolean() bool {
	if r.bits == 0 {
		r.current = r.buf[0]
		r.buf = r.buf[1:]
		r.bits = 8
	}
	b := (r.current & 1) == 1
	r.current >>= 1
	r.bits -= 1
	return b
}

func (r *plainBooleanReader) nextInt32() int32 {
	panic("cannot read INT32 from PLAIN BOOLEAN reader")
}

func (r *plainBooleanReader) nextInt64() int64 {
	panic("cannot read INT64 from PLAIN BOOLEAN reader")
}

func (r *plainBooleanReader) nextFloat() float64 {
	panic("cannot read FLOAT from PLAIN BOOLEAN reader")
}

func (r *plainBooleanReader) nextDouble() float64 {
	panic("cannot read DOUBLE from PLAIN BOOLEAN reader")
}

func (r *plainBooleanReader) nextByteArray() []byte {
	panic("cannot read BYTE_ARRAY from PLAIN BOOLEAN reader")
}

// plainReader decodes values encoded with the parquet PLAIN encoding
// for all types other than BOOLEAN.  PLAIN encoding is described at:
// https://github.com/apache/parquet-format/blob/master/Encodings.md#plain-plain--0
// Handle Parquet PLAIN encoding type
type plainReader struct {
	buf []byte
}

func (r *plainReader) nextBoolean() bool {
	// panic("implement plain boolean reader")
	return false
}

func (r *plainReader) nextInt32() int32 {
	if len(r.buf) < 4 {
		panic("underflow in PLAIN INT32")
	}

	ret := binary.LittleEndian.Uint32(r.buf[:4])
	r.buf = r.buf[4:]
	return int32(ret)
}

func (r *plainReader) nextInt64() int64 {
	if len(r.buf) < 8 {
		panic("underflow in PLAIN INT64")
	}

	ret := binary.LittleEndian.Uint64(r.buf[:8])
	r.buf = r.buf[8:]
	return int64(ret)
}

func (r *plainReader) nextFloat() float64 {
	if len(r.buf) < 4 {
		panic("underflow in PLAIN FLOAT")
	}
	v := binary.LittleEndian.Uint32(r.buf[:4])
	r.buf = r.buf[4:]
	return float64(math.Float32frombits(v))
}

func (r *plainReader) nextDouble() float64 {
	if len(r.buf) < 8 {
		panic("underflow in PLAIN DOUBLE")
	}
	v := binary.LittleEndian.Uint64(r.buf[:8])
	r.buf = r.buf[8:]
	return math.Float64frombits(v)
}

func (r *plainReader) nextByteArray() []byte {
	if len(r.buf) < 4 {
		panic("underflow in PLAIN BYTE_ARRAY")
	}
	ln := binary.LittleEndian.Uint32(r.buf[:4])
	total := int(4 + ln)
	if len(r.buf) < total {
		panic("underflow in PLAIN BYTE_ARRAY")
	}
	ret := r.buf[4:total]
	r.buf = r.buf[total:]
	return ret
}

// dictionaryInt64Reader decodes INT64-typed values encoded with the
// parquet PLAIN_DICTIONARY encoding described at:
// https://github.com/apache/parquet-format/blob/master/Encodings.md#dictionary-encoding-plain_dictionary--2-and-rle_dictionary--8
type dictionaryInt64Reader struct {
	dict        []int64
	indexReader *hybridReader
}

func newDictionaryInt64Reader(buf []byte, dict []int64) *dictionaryInt64Reader {
	width := int(buf[0])
	reader := newHybridReader(buf[1:], width)
	return &dictionaryInt64Reader{dict, reader}
}

func (r *dictionaryInt64Reader) nextBoolean() bool {
	panic("cannot read BOOLEAN from INT64 dictionary reader")
}

func (r *dictionaryInt64Reader) nextInt32() int32 {
	panic("cannot read INT32 from INT64 dictionary reader")
}

func (r *dictionaryInt64Reader) nextInt64() int64 {
	i := int(r.indexReader.nextInt64())
	if i > len(r.dict) {
		panic(fmt.Sprintf("dictionary index too large (%d>%d)", i, len(r.dict)))
	}
	return r.dict[i]
}

func (r *dictionaryInt64Reader) nextFloat() float64 {
	panic("cannot read FLOAT from INT64 dictionary reader")
}

func (r *dictionaryInt64Reader) nextDouble() float64 {
	panic("cannot read DOUBLE from INT64 dictionary reader")
}

func (r *dictionaryInt64Reader) nextByteArray() []byte {
	panic("cannot read BYTE_ARRAY from INT64 dictionary reader")
}

// dictionaryDoubleReader decodes DOUBLE-typed values encoded with the
// parquet PLAIN_DICTIONARY encoding described at:
// https://github.com/apache/parquet-format/blob/master/Encodings.md#dictionary-encoding-plain_dictionary--2-and-rle_dictionary--8
type dictionaryDoubleReader struct {
	dict        []float64
	indexReader *hybridReader
}

func newDictionaryDoubleReader(buf []byte, dict []float64) *dictionaryDoubleReader {
	width := int(buf[0])
	reader := newHybridReader(buf[1:], width)
	return &dictionaryDoubleReader{dict, reader}
}

func (r *dictionaryDoubleReader) nextBoolean() bool {
	panic("cannot read BOOLEAN from DOUBLE dictionary reader")
}

func (r *dictionaryDoubleReader) nextInt32() int32 {
	panic("cannot read INT32 from DOUBLE dictionary reader")
}

func (r *dictionaryDoubleReader) nextInt64() int64 {
	panic("cannot read INT64 from DOUBLE dictionary reader")
}

func (r *dictionaryDoubleReader) nextFloat() float64 {
	panic("cannot read FLOAT from DOUBLE dictionary reader")
}

func (r *dictionaryDoubleReader) nextDouble() float64 {
	i := int(r.indexReader.nextInt64())
	if i > len(r.dict) {
		panic(fmt.Sprintf("dictionary index too large (%d>%d)", i, len(r.dict)))
	}
	return r.dict[i]
}

func (r *dictionaryDoubleReader) nextByteArray() []byte {
	panic("cannot read BYTE_ARRAY from DOUBLE dictionary reader")
}

// dictionaryByteArrayReader decodes BYTE_ARRAY-typed values encoded with the
// parquet PLAIN_DICTIONARY encoding described at:
// https://github.com/apache/parquet-format/blob/master/Encodings.md#dictionary-encoding-plain_dictionary--2-and-rle_dictionary--8
type dictionaryByteArrayReader struct {
	dict        [][]byte
	indexReader *hybridReader
}

func newDictionaryByteArrayReader(buf []byte, dict [][]byte) *dictionaryByteArrayReader {
	width := int(buf[0])
	reader := newHybridReader(buf[1:], width)
	return &dictionaryByteArrayReader{dict, reader}
}

func (r *dictionaryByteArrayReader) nextBoolean() bool {
	panic("cannot read BOOLEAN from BYTE_ARRAY dictionary reader")
}

func (r *dictionaryByteArrayReader) nextInt32() int32 {
	panic("cannot read INT32 from BYTE_ARRAY dictionary reader")
}

func (r *dictionaryByteArrayReader) nextInt64() int64 {
	panic("cannot read INT64 from BYTE_ARRAY dictionary reader")
}

func (r *dictionaryByteArrayReader) nextFloat() float64 {
	panic("cannot read FLOAT from BYTE_ARRAY dictionary reader")
}

func (r *dictionaryByteArrayReader) nextDouble() float64 {
	panic("cannot read DOUBLE from BYTE_ARRAY dictionary reader")
}

func (r *dictionaryByteArrayReader) nextByteArray() []byte {
	i := int(r.indexReader.nextInt64())
	if i > len(r.dict) {
		panic(fmt.Sprintf("dictionary index too large (%d>%d)", i, len(r.dict)))
	}
	return r.dict[i]
}
