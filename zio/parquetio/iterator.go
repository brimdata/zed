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
	"github.com/xitongsys/parquet-go/reader"
	"github.com/xitongsys/parquet-go/source"
)

type valueReader interface {
	nextBoolean() bool
	nextInt32() int32
	nextInt64() int64
	nextFloat() float64
	nextDouble() float64
	nextByteArray() []byte
}

type columnIterator struct {
	pr   *reader.ParquetReader
	name string

	maxRL int32
	maxDL int32

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

func newColumnIterator(pr *reader.ParquetReader, name string, maxRL, maxDL int32) *columnIterator {
	return &columnIterator{
		pr:    pr,
		name:  name,
		maxRL: maxRL,
		maxDL: maxDL,
	}
}

func (i *columnIterator) clearDictionaries() {
	i.byteArrayDict = nil
	i.int64Dict = nil
	i.floatDict = nil
	// XXX clear others as they are added
}

func (i *columnIterator) loadOnePage() (*parquet.PageHeader, []byte, error) {
	if i.groupRead == i.groupTotal {
		if i.rowGroup >= len(i.pr.Footer.RowGroups) {
			return nil, nil, io.EOF
		}
		rg := i.pr.Footer.RowGroups[i.rowGroup]
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
			panic("cannot find column")
		}

		if col.FilePath != nil {
			panic("ColumnChunk refers to external file")
		}

		i.colMetadata = col.MetaData
		offset := i.colMetadata.DataPageOffset
		if i.colMetadata.DictionaryPageOffset != nil {
			offset = *i.colMetadata.DictionaryPageOffset
		}
		size := i.colMetadata.TotalCompressedSize

		// XXX
		i.thriftReader = source.ConvertToThriftReader(i.pr.PFile, offset, size)

		i.clearDictionaries()
	}

	header, err := layout.ReadPageHeader(i.thriftReader)
	if err != nil {
		return nil, nil, err
	}

	// fmt.Printf("header type %s\n", header.GetType())
	// XXX assert on page type

	compressedLen := header.GetCompressedPageSize()

	raw := make([]byte, compressedLen)
	_, err = i.thriftReader.Read(raw)
	if err != nil {
		return nil, nil, err
	}

	page, err := compress.Uncompress(raw, i.colMetadata.GetCodec())
	if err != nil {
		return nil, nil, err
	}

	return header, page, nil
}

func (i *columnIterator) ensureDataPage() {
	if i.pageRead < i.pageTotal {
		return
	}

	// XXX update groupTotal

	for {
		header, page, err := i.loadOnePage()
		if err != nil {
			panic(err)
		}

		switch header.GetType() {
		case parquet.PageType_DICTIONARY_PAGE:
			i.loadDictionaryPage(header, page)

		case parquet.PageType_DATA_PAGE:
			i.initializeDataPage(header, page)
			return

		default:
			panic(fmt.Sprintf("unhandled page type %s", header.GetType()))
		}
	}
}

func (i *columnIterator) loadDictionaryPage(header *parquet.PageHeader, buf []byte) {
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
		// fmt.Printf("skipping dictionary page for type %s\n", i.colMetadata.GetType())
	}
}

func (i *columnIterator) initializeDataPage(header *parquet.PageHeader, buf []byte) {
	i.pageRead = 0
	i.pageTotal = int(header.DataPageHeader.GetNumValues())

	if i.maxRL > 0 {
		width := int(common.BitNum(uint64(i.maxRL)))
		hbuf, n, err := grabLenDenotedBuf(buf)
		if err != nil {
			panic(err)
		}
		buf = buf[n:]
		i.rlReader = newHybridReader(hbuf, width)
	} else {
		i.rlReader = nil
	}

	if i.maxDL > 0 {
		width := int(common.BitNum(uint64(i.maxDL)))
		hbuf, n, err := grabLenDenotedBuf(buf)
		if err != nil {
			panic(err)
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
		//fmt.Printf("instantiate plainReader for %s\n", i.name)
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
			//fmt.Printf("skipping dictionary page of type %s\n", typ)
			i.valReader = &nullReader{}
		}
	default:
		//fmt.Printf("skipping data page with encoding %s\n", enc)
		i.valReader = &nullReader{}
	}
}

func (i *columnIterator) peekDL() int32 {
	i.ensureDataPage()
	if i.dlReader == nil {
		return 0
	}
	return int32(i.dlReader.peekInt64())
}

// advance counter, grab rl and dl
func (i *columnIterator) commonNext() (int32, int32) {
	i.ensureDataPage()
	i.pageRead++

	var rl, dl int32
	if i.rlReader != nil {
		rl = int32(i.rlReader.nextInt64())
	}
	if i.dlReader != nil {
		dl = int32(i.dlReader.nextInt64())
	}

	if rl == i.maxRL {
		i.groupRead++
	}

	return rl, dl
}

func (i *columnIterator) nextBoolean() (bool, int32, int32) {
	rl, dl := i.commonNext()
	var v bool
	if dl == i.maxDL {
		v = i.valReader.nextBoolean()
	}
	return v, rl, dl
}

func (i *columnIterator) nextInt32() (int32, int32, int32) {
	rl, dl := i.commonNext()
	var v int32
	if dl == i.maxDL {
		v = i.valReader.nextInt32()
	}
	return v, rl, dl
}

func (i *columnIterator) nextInt64() (int64, int32, int32) {
	rl, dl := i.commonNext()
	var v int64
	if dl == i.maxDL {
		v = i.valReader.nextInt64()
	}
	return v, rl, dl
}

func (i *columnIterator) nextFloat() (float64, int32, int32) {
	rl, dl := i.commonNext()
	var v float64
	if dl == i.maxDL {
		v = i.valReader.nextFloat()
	}
	return v, rl, dl
}

func (i *columnIterator) nextDouble() (float64, int32, int32) {
	rl, dl := i.commonNext()
	var v float64
	if dl == i.maxDL {
		v = i.valReader.nextDouble()
	}
	return v, rl, dl
}

func (i *columnIterator) nextByteArray() ([]byte, int32, int32) {
	rl, dl := i.commonNext()
	var v []byte
	if dl == i.maxDL {
		v = i.valReader.nextByteArray()
	}
	return v, rl, dl
}

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
		//fmt.Printf("decode rle %d x %d\n", val, n)
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
		//fmt.Printf("decode packed %d*8 %d-bit values\n", groups, r.width)
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

// Handle Parquet PLAIN_DICTIONARY encoding type
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

// Handle Parquet PLAIN_DICTIONARY encoding type
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

// Handle Parquet PLAIN_DICTIONARY encoding type
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
