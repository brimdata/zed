package lakemanage

import "time"

type Config struct {
	ColdThreshold time.Duration `yaml:"coldthresh"`
}
