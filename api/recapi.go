package api

type Worker struct {
	Addr     string `json:"addr"`
	NodeName string `json:"node"`
}

type RegisterRequest struct {
	Worker
}

type StatusResponse struct {
	Status string `json:"status"`
}

type DeregisterRequest struct {
	Addr string `json:"addr"`
}

type RecruitRequest struct {
	NumberRequested int `json:"N"`
}

type RecruitResponse struct {
	Workers []Worker `json:"workers"`
}
