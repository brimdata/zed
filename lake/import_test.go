package lake

/*
func TestImportStaleDuration(t *testing.T) {
	t.Run("Stale", func(t *testing.T) {
		testImportStaleDuration(t, 0, 1)
	})
	t.Run("NotStale", func(t *testing.T) {
		testImportStaleDuration(t, math.MaxInt64, 0)
	})
}

func testImportStaleDuration(t *testing.T, stale time.Duration, expected uint64) {
	const data = "{ts:2020-04-21T22:40:50.06466032Z,offset:202}"

	// create archive with a 1 ns ImportFlushTimeout
	lk, err := CreateOrOpenLake(t.TempDir(), nil, nil)
	require.NoError(t, err)

	// write one record to an open archive.Writer and do NOT close it.
	w, err := NewWriter(context.Background(), lk)
	require.NoError(t, err)
	defer w.Close()
	w.SetStaleDuration(stale)
	r := zson.NewReader(strings.NewReader(data), zson.NewContext())
	require.NoError(t, zbuf.Copy(w, r))

	// flush stale writers and ensure data has been written to archive
	err = w.flushStaleWriters()
	require.NoError(t, err)
	count, err := RecordCount(context.Background(), lk)
	require.NoError(t, err)
	assert.EqualValues(t, expected, count)
}
*/
