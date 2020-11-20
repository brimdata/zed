package api

type RegisterRequest struct {
	Addr     string `json:"addr"`
	NodeName string `json:"node"`
}

type UnregisterRequest struct {
	Addr string `json:"addr"`
}

type RecruitRequest struct {
	NumberRequested int `json:"N"`
}

type RecruitResponse struct {
	Workers []string `json:"workers"`
}
