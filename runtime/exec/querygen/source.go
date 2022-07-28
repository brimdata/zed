package querygen

/*
type Source struct {
	engine storage.Engine
	lake   *lake.Root
}

func NewSource(engine storage.Engine, lake *lake.Root) *Source {
	return &Source{
		engine: engine,
		lake:   lake,
	}
}

func (s *Source) IsLake() bool {
	return s.lake != nil
}

func (s *Source) Lake() *lake.Root {
	return s.lake
}

func (s *Source) PoolID(ctx context.Context, id string) (ksuid.KSUID, error) {
	if s.lake != nil {
		return s.lake.PoolID(ctx, id)
	}
	return ksuid.Nil, nil
}

func (s *Source) xCommitObject(ctx context.Context, id ksuid.KSUID, name string) (ksuid.KSUID, error) {
	if s.lake != nil {
		return s.lake.CommitObject(ctx, id, name)
	}
	return ksuid.Nil, nil
}

func (s *Source) xLayout(ctx context.Context, src dag.Source) order.Layout {
	if s.lake != nil {
		return s.lake.Layout(ctx, src)
	}
	return order.Nil
}
*/
