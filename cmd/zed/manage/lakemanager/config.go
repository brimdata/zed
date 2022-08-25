package lakemanager

import "time"

type Config struct {
	ColdThreshold time.Duration `yaml:"coldthresh"`
}
