package api

type WorkerChunkRequest struct {
	SearchRequest
	ChunkPaths []string `json:"chunk_paths"`
	DataPath   string   `json:"data_path"`
}

type WorkerRootRequest struct {
	SearchRequest
	NumberOfWorkers int `json:"number_of_workers"`
}
