package api

type StatusResponse struct {
	Status string `json:"status"`
}

type WorkerAddr struct {
	Addr string `json:"addr"`
}

type UnreserveRequest struct {
	WorkerAddr
}

type DeregisterRequest struct {
	WorkerAddr
}

type Worker struct {
	WorkerAddr
	NodeName string `json:"node"`
}

type RegisterRequest struct {
	Worker
}

type RecruitRequest struct {
	NumberRequested int `json:"N"`
}

type RecruitResponse struct {
	Workers []Worker `json:"workers"`
}
