package lake

// THIS INFO IS ALL THE LOG NOW.  We can use "zed lake log" to get at it
// even in ZNG format.

/*

// statReadCloser implements zio.ReadCloser.
type statReadCloser struct {
	lk       *Lake
	ctx      context.Context
	cancel   context.CancelFunc
	defnames map[ksuid.KSUID]string
	err      error
	recs     chan *zng.Record
	zctx     *zson.MarshalZNGContext
}

func (s *statReadCloser) Read() (*zng.Record, error) {
	select {
	case r, ok := <-s.recs:
		if !ok {
			return nil, s.err
		}
		return r, nil
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	}
}

func (s *statReadCloser) Close() error {
	s.cancel()
	return nil
}

type chunkStat struct {
	Type        string  `zng:"type"`
	LogID       string  `zng:"log_id"`
	First       nano.Ts `zng:"first"`
	Last        nano.Ts `zng:"last"`
	Size        uint64  `zng:"size"`
	RecordCount uint64  `zng:"record_count"`
}

func (s *statReadCloser) chunkRecord(chunk chunk.Chunk) error {
	stat := chunkStat{
		Type:        "chunk",
		LogID:       s.lk.Root.RelPath(chunk.Path()),
		First:       chunk.First,
		Last:        chunk.Last,
		Size:        uint64(chunk.Size),
		RecordCount: chunk.RecordCount,
	}
	rec, err := s.zctx.MarshalRecord(stat)
	if err != nil {
		return err
	}
	select {
	case s.recs <- rec:
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}

type defDesc struct {
	ID          string `zng:"id"`
	Description string `zng:"description"`
}

type indexStat struct {
	Type        string             `zng:"type"`
	LogID       string             `zng:"log_id"`
	First       nano.Ts            `zng:"first"`
	Last        nano.Ts            `zng:"last"`
	Definition  defDesc            `zng:"definition"`
	Size        uint64             `zng:"size"`
	RecordCount uint64             `zng:"record_count"`
	Keys        []zedindex.InfoKey `zng:"keys"`
}

func (s *statReadCloser) indexRecords(chunk chunk.Chunk) error {
	dir := chunk.ZarDir()
	ids, err := index.ListDefinitionIDs(s.ctx, dir)
	if err != nil {
		return err
	}
	for _, id := range ids {
		info, err := zedindex.Stat(s.ctx, index.IndexPath(dir, id))
		if err != nil {
			return err
		}
		defname, ok := s.defnames[id]
		if !ok {
			defname = "[deleted]"
		}
		stat := indexStat{
			Type:       "index",
			LogID:      s.lk.Root.RelPath(chunk.Path()),
			First:      chunk.First,
			Last:       chunk.Last,
			Definition: defDesc{ID: id.String(), Description: defname},
			Size:       uint64(info.Size),
			Keys:       info.Keys,
		}
		rec, err := s.zctx.MarshalRecord(stat)
		if err == nil {
			err = s.send(rec)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *statReadCloser) send(rec *zng.Record) error {
	select {
	case s.recs <- rec:
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}

func (s *statReadCloser) run() {
	defer close(s.recs)

	s.err = Walk(s.ctx, s.lk, func(chunk chunk.Chunk) error {
		if err := s.chunkRecord(chunk); err != nil {
			return err
		}
		if err := s.indexRecords(chunk); err != nil {
			return err
		}
		return nil
	})
}

func RecordCount(ctx context.Context, lk *Lake) (uint64, error) {
	var count uint64
	err := Walk(ctx, lk, func(chunk chunk.Chunk) error {
		count += chunk.RecordCount
		return nil
	})
	return count, err
}

type IndexInfo struct {
	DefinitionID ksuid.KSUID
	IndexCount   uint64
	ChunkCount   uint64
}

func IndexStat(ctx context.Context, lk *Lake, defs []*index.Definition) ([]IndexInfo, error) {
	m := make(map[ksuid.KSUID]IndexInfo)
	for _, def := range defs {
		m[def.ID] = IndexInfo{DefinitionID: def.ID}
	}
	var chunkCount uint64
	err := Walk(ctx, lk, func(chunk chunk.Chunk) error {
		chunkCount++
		ids, err := index.ListDefinitionIDs(ctx, chunk.ZarDir())
		if err != nil {
			return err
		}
		for _, id := range ids {
			if stat, ok := m[id]; ok {
				stat.IndexCount++
				m[id] = stat
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	stats := make([]IndexInfo, 0, len(m))
	for _, stat := range m {
		stat.ChunkCount = chunkCount
		stats = append(stats, stat)
	}
	return stats, nil
}

func Stat(ctx context.Context, zctx *zson.Context, lk *Lake) (zbuf.ReadCloser, error) {
	defs, err := lk.ReadDefinitions(ctx)
	if err != nil {
		return nil, err
	}
	// Make a map of human readable names for the definitions.
	defnames := make(map[ksuid.KSUID]string)
	for _, def := range defs {
		defnames[def.ID] = def.String()
	}
	ctx, cancel := context.WithCancel(ctx)
	mzctx := zson.NewZNGMarshaler()
	mzctx.Context = zctx
	s := &statReadCloser{
		lk:       lk,
		ctx:      ctx,
		cancel:   cancel,
		defnames: defnames,
		recs:     make(chan *zng.Record),
		zctx:     mzctx,
	}
	go s.run()
	return s, nil
}
*/
