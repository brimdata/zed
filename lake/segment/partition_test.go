package segment_test

/* TBD: re-do these tests with ztest by adding a way to scan the scan intervals...
   We need this functionality anyway to do pool introspection...
   See issue #2538

func kid(s string) ksuid.KSUID {
	var b [20]byte
	copy(b[:], s)
	k, _ := ksuid.FromBytes(b[:])
	return k
}

func createLake(t *testing.T) *lake.Pool {
	rootPath := iosrc.MustParseURI("file://" + t.TempDir())
	lakeName := "test-" + ksuid.New().String()
	ctx := context.Background()
	lk, err := lake.Create(ctx, rootPath, lakeName)
	require.NoError(t, err)
	pool, err := lk.CreatePool(ctx, "test", zbuf.OrderAsc)
	require.NoError(t, err)
	return pool
}

func importTzng(t *testing.T, pool *lake.Pool, s string) {
	ctx := context.Background()
	zctx := resolver.NewContext()
	reader := tzngio.NewReader(strings.NewReader(s), zctx)
	commits, err := pool.Add(ctx, zctx.Context, reader)
	require.NoError(t, err)
	err = pool.Commit(ctx, commits)
	require.NoError(t, err)
}

func seg(id string, first, last int) *segment.Reference {
	s := segment.NewReference(iosrc.URI{}, kid(id), 0)
	s.First = nano.Ts(first)
	s.Last = nano.Ts(last)
	return s
}

func TestPartitionSegments(t *testing.T) {
	cases := []struct {
		segments []*segment.Reference
		filter   nano.Span
		order    zbuf.Order
		exp      []journal.Range
	}{
		{
			segments: []*segment.Reference{
				seg("a", 0, 0),
				seg("b", 1, 1),
			},
			filter: nano.MaxSpan,
			order:  zbuf.OrderAsc,
			exp: []journal.Range{
				{First: 0, Last: 0, Segments: []*segment.Reference{seg("a", 0, 0)}},
				{Span: nano.Span{Ts: 1, Dur: 1}, Segments: []*segment.Referemce{seg("b", 1, 1)}},
			},
		},
		{
			chunks: []segment.Segment{
				{ID: kid("a"), First: 0, Last: 1},
				{ID: kid("b"), First: 1, Last: 2},
			},
			filter: nano.MaxSpan,
			order:  zbuf.OrderAsc,
			exp: []SpanInfo{
				{Span: nano.Span{Ts: 0, Dur: 1}, Chunks: []segment.Segment{{ID: kid("a"), First: 0, Last: 1}}},
				{Span: nano.Span{Ts: 1, Dur: 1}, Chunks: []segment.Segment{{ID: kid("a"), First: 0, Last: 1}, {ID: kid("b"), First: 1, Last: 2}}},
				{Span: nano.Span{Ts: 2, Dur: 1}, Chunks: []segment.Segment{{ID: kid("b"), First: 1, Last: 2}}},
			},
		},
		{
			chunks: []segment.Segment{
				{ID: kid("a"), First: 0, Last: 3},
				{ID: kid("b"), First: 1, Last: 2},
			},
			filter: nano.MaxSpan,
			order:  zbuf.OrderAsc,
			exp: []SpanInfo{
				{Span: nano.Span{Ts: 0, Dur: 1}, Chunks: []segment.Segment{{ID: kid("a"), First: 0, Last: 3}}},
				{Span: nano.Span{Ts: 1, Dur: 2}, Chunks: []segment.Segment{{ID: kid("a"), First: 0, Last: 3}, {ID: kid("b"), First: 1, Last: 2}}},
				{Span: nano.Span{Ts: 3, Dur: 1}, Chunks: []segment.Segment{{ID: kid("a"), First: 0, Last: 3}}},
			},
		},
		{
			chunks: []segment.Segment{
				{ID: kid("a"), First: 0, Last: 3},
				{ID: kid("b"), First: 1, Last: 2},
			},
			filter: nano.Span{Ts: 1, Dur: 2},
			order:  zbuf.OrderAsc,
			exp: []SpanInfo{
				{Span: nano.Span{Ts: 1, Dur: 2}, Chunks: []segment.Segment{{ID: kid("a"), First: 0, Last: 3}, {ID: kid("b"), First: 1, Last: 2}}},
			},
		},
		{
			chunks: []segment.Segment{
				{ID: kid("a"), First: 9, Last: 7},
				{ID: kid("b"), First: 5, Last: 3},
			},
			filter: nano.MaxSpan,
			order:  zbuf.OrderDesc,
			exp: []SpanInfo{
				{Span: nano.Span{Ts: 7, Dur: 3}, Chunks: []segment.Segment{{ID: kid("a"), First: 9, Last: 7}}},
				{Span: nano.Span{Ts: 3, Dur: 3}, Chunks: []segment.Segment{{ID: kid("b"), First: 5, Last: 3}}},
			},
		},
		{
			chunks: []segment.Segment{
				{ID: kid("a"), First: 9, Last: 5},
				{ID: kid("b"), First: 7, Last: 3},
			},
			filter: nano.MaxSpan,
			order:  zbuf.OrderDesc,
			exp: []SpanInfo{
				{Span: nano.Span{Ts: 8, Dur: 2}, Chunks: []segment.Segment{{ID: kid("a"), First: 9, Last: 5}}},
				{Span: nano.Span{Ts: 5, Dur: 3}, Chunks: []segment.Segment{{ID: kid("a"), First: 9, Last: 5}, {ID: kid("b"), First: 7, Last: 3}}},
				{Span: nano.Span{Ts: 3, Dur: 2}, Chunks: []segment.Segment{{ID: kid("b"), First: 7, Last: 3}}},
			},
		},
		{
			chunks: []segment.Segment{
				{ID: kid("b"), First: 0, Last: 0},
				{ID: kid("a"), First: 0, Last: 0},
				{ID: kid("d"), First: 0, Last: 0},
				{ID: kid("c"), First: 0, Last: 0},
			},
			filter: nano.MaxSpan,
			order:  zbuf.OrderAsc,
			exp: []SpanInfo{
				{Span: nano.Span{Ts: 0, Dur: 1}, Chunks: []segment.Segment{
					{ID: kid("a"), First: 0, Last: 0},
					{ID: kid("b"), First: 0, Last: 0},
					{ID: kid("c"), First: 0, Last: 0},
					{ID: kid("d"), First: 0, Last: 0}}},
			},
		},
		{
			chunks: []segment.Segment{
				{ID: kid("a"), First: 0, Last: 5},
				{ID: kid("b"), First: 1, Last: 8},
				{ID: kid("c"), First: 6, Last: 6},
				{ID: kid("d"), First: 7, Last: 10},
			},
			filter: nano.MaxSpan,
			order:  zbuf.OrderAsc,
			exp: []SpanInfo{
				{Span: nano.Span{Ts: 0, Dur: 1}, Chunks: []segment.Segment{
					{ID: kid("a"), First: 0, Last: 5}}},
				{Span: nano.Span{Ts: 1, Dur: 5}, Chunks: []segment.Segment{
					{ID: kid("a"), First: 0, Last: 5},
					{ID: kid("b"), First: 1, Last: 8}}},
				{Span: nano.Span{Ts: 6, Dur: 1}, Chunks: []segment.Segment{
					{ID: kid("b"), First: 1, Last: 8},
					{ID: kid("c"), First: 6, Last: 6}}},
				{Span: nano.Span{Ts: 7, Dur: 2}, Chunks: []segment.Segment{
					{ID: kid("b"), First: 1, Last: 8},
					{ID: kid("d"), First: 7, Last: 10}}},
				{Span: nano.Span{Ts: 9, Dur: 2}, Chunks: []segment.Segment{
					{ID: kid("d"), First: 7, Last: 10}}},
			},
		},
		{
			chunks: []segment.Segment{
				{ID: kid("a"), First: 0, Last: 10},
				{ID: kid("b"), First: 1, Last: 10},
				{ID: kid("c"), First: 2, Last: 10},
			},
			filter: nano.MaxSpan,
			order:  zbuf.OrderAsc,
			exp: []SpanInfo{
				{Span: nano.Span{Ts: 0, Dur: 1}, Chunks: []segment.Segment{
					{ID: kid("a"), First: 0, Last: 10}}},
				{Span: nano.Span{Ts: 1, Dur: 1}, Chunks: []segment.Segment{
					{ID: kid("a"), First: 0, Last: 10},
					{ID: kid("b"), First: 1, Last: 10}}},
				{Span: nano.Span{Ts: 2, Dur: 9}, Chunks: []segment.Segment{
					{ID: kid("a"), First: 0, Last: 10},
					{ID: kid("b"), First: 1, Last: 10},
					{ID: kid("c"), First: 2, Last: 10}}},
			},
		},
		{
			chunks: nil,
			filter: nano.MaxSpan,
			order:  zbuf.OrderAsc,
			exp:    nil,
		},
	}
	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, c.exp, alignChunksToSpans(c.chunks, c.order, c.filter))
		})
	}
}

func TestOverlapWalking(t *testing.T) {
	datapath, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(datapath)

	lk, err := CreateOrOpenLake(datapath, &CreateOptions{}, nil)
	require.NoError(t, err)

	const data1 = `
{ts:1970-01-01T00:00:00Z,v:0}
{ts:1970-01-01T00:00:00.000000005Z,v:5}
`
	const data2 = `
{ts:1970-01-01T00:00:00.00000001Z,v:10}
{ts:1970-01-01T00:00:00.00000002Z,v:20}
`
	const data3 = `
{ts:1970-01-01T00:00:00.000000015Z,v:15}
{ts:1970-01-01T00:00:00.000000025Z,v:25}
`
	dataChunkSpans := []nano.Span{{Ts: 15, Dur: 11}, {Ts: 10, Dur: 11}, {Ts: 0, Dur: 6}}
	importZSON(t, lk, data2)
	importZSON(t, lk, data1)
	importZSON(t, lk, data3)

	{
		var chunks []segment.Segment
		err = tsDirVisit(context.Background(), lk, nano.MaxSpan, func(tsd tsDir, c []segment.Segment) error {
			chunks = append(chunks, c...)
			return nil
		})
		require.NoError(t, err)
		require.Len(t, chunks, 3)
		segment.Sort(lk.DataOrder, chunks)
		var spans []nano.Span
		for _, c := range chunks {
			spans = append(spans, c.Span())
		}
		require.Equal(t, dataChunkSpans, spans)
	}
	{
		var chunks []segment.Segment
		err = Walk(context.Background(), lk, func(c segment.Segment) error {
			chunks = append(chunks, c)
			return nil
		})
		require.NoError(t, err)
		require.Len(t, chunks, 3)
		var spans []nano.Span
		for _, c := range chunks {
			spans = append(spans, c.Span())
		}
		require.Equal(t, dataChunkSpans, spans)
	}
	{
		var chunks []segment.Segment
		err = tsDirVisit(context.Background(), lk, nano.Span{Ts: 12, Dur: 20}, func(tsd tsDir, c []segment.Segment) error {
			chunks = append(chunks, c...)
			return nil
		})
		require.NoError(t, err)
		assert.Len(t, chunks, 2)
		segment.Sort(lk.DataOrder, chunks)
		var spans []nano.Span
		for _, c := range chunks {
			spans = append(spans, c.Span())
		}
		assert.Equal(t, []nano.Span{{Ts: 15, Dur: 11}, {Ts: 10, Dur: 11}}, spans)
	}
	{
		type sispan struct {
			si         nano.Span
			chunkSpans []nano.Span
		}
		var sispans []sispan
		err = SpanWalk(context.Background(), lk, nano.Span{Ts: 12, Dur: 10}, func(si SpanInfo) error {
			var chunkSpans []nano.Span
			for _, c := range si.Chunks {
				chunkSpans = append(chunkSpans, c.Span())
			}
			sispans = append(sispans, sispan{si: si.Span, chunkSpans: chunkSpans})
			return nil
		})
		require.NoError(t, err)
		assert.Len(t, sispans, 2)
		exp := []sispan{
			{si: nano.Span{Ts: 15, Dur: 7}, chunkSpans: []nano.Span{{Ts: 15, Dur: 11}, {Ts: 10, Dur: 11}}},
			{si: nano.Span{Ts: 12, Dur: 3}, chunkSpans: []nano.Span{{Ts: 10, Dur: 11}}},
		}
		assert.Equal(t, exp, sispans)
	}
}

func TestMergeChunksToSpans(t *testing.T) {
	cases := []struct {
		in     []segment.Segment
		filter nano.Span
		order  zbuf.Order
		exp    []SpanInfo
	}{
		{
			in: []segment.Segment{
				{ID: kid("a"), First: 0, Last: 3, RecordCount: 10},
				{ID: kid("b"), First: 1, Last: 3, RecordCount: 20},
			},
			filter: nano.MaxSpan,
			order:  zbuf.OrderAsc,
			exp: []SpanInfo{
				{Span: nano.Span{Ts: 0, Dur: 1}, Chunks: []segment.Segment{
					{ID: kid("a"), First: 0, Last: 3, RecordCount: 10}}},
				{Span: nano.Span{Ts: 1, Dur: 3}, Chunks: []segment.Segment{
					{ID: kid("a"), First: 0, Last: 3, RecordCount: 10},
					{ID: kid("b"), First: 1, Last: 3, RecordCount: 20}}},
			},
		},
		{
			in: []segment.Segment{
				{ID: kid("a"), First: 0, Last: 3, RecordCount: 20},
				{ID: kid("b"), First: 2, Last: 5, RecordCount: 10},
			},
			filter: nano.MaxSpan,
			order:  zbuf.OrderAsc,
			exp: []SpanInfo{
				{Span: nano.Span{Ts: 0, Dur: 4}, Chunks: []segment.Segment{
					{ID: kid("a"), First: 0, Last: 3, RecordCount: 20},
					{ID: kid("b"), First: 2, Last: 5, RecordCount: 10}}},
				{Span: nano.Span{Ts: 4, Dur: 2}, Chunks: []segment.Segment{
					{ID: kid("b"), First: 2, Last: 5, RecordCount: 10}}},
			},
		},
		{
			in: []segment.Segment{
				{ID: kid("b"), First: 0, Last: 0, RecordCount: 10},
				{ID: kid("a"), First: 0, Last: 0, RecordCount: 10},
			},
			filter: nano.MaxSpan,
			order:  zbuf.OrderAsc,
			exp: []SpanInfo{
				{Span: nano.Span{Ts: 0, Dur: 1}, Chunks: []segment.Segment{
					{ID: kid("a"), First: 0, Last: 0, RecordCount: 10},
					{ID: kid("b"), First: 0, Last: 0, RecordCount: 10}}},
			},
		},
		{
			in:     nil,
			filter: nano.MaxSpan,
			order:  zbuf.OrderAsc,
			exp:    nil,
		},
	}
	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, c.exp, mergeChunksToSpans(c.in, c.order, c.filter))
		})
	}
}

*/
