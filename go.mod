module github.com/brimsec/zq

go 1.15

require (
	github.com/alecthomas/units v0.0.0-20190717042225-c3de453c63f4
	github.com/alexbrainman/ps v0.0.0-20171229230509-b3e1b4a15894
	github.com/apache/thrift v0.13.0
	github.com/aws/aws-sdk-go v1.36.17
	github.com/axiomhq/hyperloglog v0.0.0-20191112132149-a4c4c47bc57f
	github.com/buger/jsonparser v0.0.0-20191004114745-ee4c978eae7e
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/fsnotify/fsnotify v1.4.9
	github.com/go-pg/pg/extra/pgotel v0.2.0
	github.com/go-pg/pg/v10 v10.7.3
	github.com/go-redis/redis/extra/redisotel v0.2.0
	github.com/go-redis/redis/v8 v8.6.0
	github.com/go-resty/resty/v2 v2.2.0
	github.com/golang-migrate/migrate/v4 v4.14.1
	github.com/golang/mock v1.4.4
	github.com/google/gopacket v1.1.17
	github.com/gorilla/mux v1.8.0
	github.com/gosuri/uilive v0.0.4
	github.com/hashicorp/golang-lru v0.5.4
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/mccanne/charm v0.0.3-0.20191224190439-b05e1b7b1be3
	github.com/mitchellh/mapstructure v1.3.3
	github.com/pbnjay/memory v0.0.0-20190104145345-974d429e7ae4
	github.com/peterh/liner v1.1.0
	github.com/pierrec/lz4/v4 v4.1.0
	github.com/pkg/browser v0.0.0-20180916011732-0a3d74bf9ce4
	github.com/pmezard/go-difflib v1.0.0
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/client_model v0.2.0
	github.com/segmentio/ksuid v1.0.2
	github.com/stretchr/testify v1.7.0
	github.com/xitongsys/parquet-go v1.5.3-0.20200514000040-789bba367841
	github.com/xitongsys/parquet-go-source v0.0.0-20200509081216-8db33acb0acf
	github.com/yuin/goldmark v1.2.1
	go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux v0.17.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.17.0
	go.opentelemetry.io/contrib/propagators/aws v0.17.0
	go.opentelemetry.io/otel v0.17.0
	go.opentelemetry.io/otel/exporters/otlp v0.17.0
	go.opentelemetry.io/otel/sdk v0.17.0
	go.opentelemetry.io/otel/trace v0.17.0
	go.temporal.io/sdk v1.4.1
	go.temporal.io/server v1.6.3
	go.uber.org/multierr v1.6.0
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a
	golang.org/x/sys v0.0.0-20210112080510-489259a85091
	golang.org/x/text v0.3.4
	google.golang.org/grpc v1.36.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/yaml.v3 v3.0.0-20210106172901-c476de37821d
)

replace github.com/minio/minio => github.com/brimsec/minio v0.0.0-20201019191454-3c6f24527f6d
