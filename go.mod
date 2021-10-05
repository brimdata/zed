module github.com/brimdata/zed

go 1.16

require (
	github.com/alecthomas/units v0.0.0-20190717042225-c3de453c63f4
	github.com/araddon/dateparse v0.0.0-20210207001429-0eec95c9db7e
	github.com/aws/aws-sdk-go v1.36.17
	github.com/axiomhq/hyperloglog v0.0.0-20191112132149-a4c4c47bc57f
	github.com/fraugster/parquet-go v0.3.0
	github.com/go-redis/redis/v8 v8.4.11
	github.com/go-resty/resty/v2 v2.2.0
	github.com/golang-jwt/jwt v3.2.1+incompatible
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/golang/snappy v0.0.2 // indirect
	github.com/gorilla/mux v1.7.5-0.20200711200521-98cb6bf42e08
	github.com/gosuri/uilive v0.0.4
	github.com/hashicorp/golang-lru v0.5.4
	github.com/kr/text v0.2.0
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/pbnjay/memory v0.0.0-20190104145345-974d429e7ae4
	github.com/peterh/liner v1.1.0
	github.com/pierrec/lz4/v4 v4.1.0
	github.com/pmezard/go-difflib v1.0.0
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/client_model v0.2.0
	github.com/segmentio/ksuid v1.0.2
	github.com/stretchr/testify v1.7.0
	github.com/yuin/goldmark v1.2.1
	go.uber.org/multierr v1.6.0
	go.uber.org/zap v1.16.0
	golang.org/x/lint v0.0.0-20201208152925-83fdc39ff7b5 // indirect
	golang.org/x/mod v0.4.0 // indirect
	golang.org/x/net v0.0.0-20201224014010-6772e930b67b // indirect
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9
	golang.org/x/sys v0.0.0-20210105210732-16f7687f5001
	golang.org/x/term v0.0.0-20210317153231-de623e64d2a6
	golang.org/x/text v0.3.4
	golang.org/x/tools v0.0.0-20201229013931-929a8494cf60 // indirect
	google.golang.org/protobuf v1.25.1-0.20201020201750-d3470999428b // indirect
	gopkg.in/check.v1 v1.0.0-20200902074654-038fdea0a05b // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210106172901-c476de37821d
	honnef.co/go/tools v0.1.0 // indirect
)

replace github.com/fraugster/parquet-go => github.com/brimdata/parquet-go v0.3.1
