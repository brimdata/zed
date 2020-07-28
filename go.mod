module github.com/brimsec/zq

go 1.14

require (
	github.com/alecthomas/units v0.0.0-20190717042225-c3de453c63f4
	github.com/alexbrainman/ps v0.0.0-20171229230509-b3e1b4a15894
	github.com/apache/thrift v0.0.0-20181112125854-24918abba929
	github.com/aws/aws-sdk-go v1.30.19
	github.com/axiomhq/hyperloglog v0.0.0-20191112132149-a4c4c47bc57f
	github.com/buger/jsonparser v0.0.0-20191004114745-ee4c978eae7e
	github.com/go-resty/resty/v2 v2.2.0
	github.com/golang/mock v1.4.3
	github.com/golang/snappy v0.0.1 // indirect
	github.com/google/gopacket v1.1.17
	github.com/gorilla/mux v1.7.5-0.20200711200521-98cb6bf42e08
	github.com/gosuri/uilive v0.0.4
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/klauspost/compress v1.10.3 // indirect
	github.com/mattn/go-isatty v0.0.8 // indirect
	github.com/mccanne/charm v0.0.3-0.20191224190439-b05e1b7b1be3
	github.com/mccanne/joe v0.0.0-20181124064909-25770742c256
	github.com/peterh/liner v1.1.0
	github.com/pmezard/go-difflib v1.0.0
	github.com/prometheus/client_golang v1.7.1
	github.com/segmentio/ksuid v1.0.2
	github.com/stretchr/testify v1.5.1
	github.com/xitongsys/parquet-go v1.5.3-0.20200514000040-789bba367841
	github.com/xitongsys/parquet-go-source v0.0.0-20200509081216-8db33acb0acf
	github.com/yuin/goldmark v1.1.27
	go.uber.org/multierr v1.5.0
	go.uber.org/zap v1.15.0
	golang.org/x/crypto v0.0.0-20200709230013-948cd5f35899
	golang.org/x/net v0.0.0-20200707034311-ab3426394381 // indirect
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/sys v0.0.0-20200625212154-ddb9806d33ae // indirect
	golang.org/x/text v0.3.3
	golang.org/x/tools v0.0.0-20200425043458-8463f397d07c // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/yaml.v2 v2.2.8 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200121175148-a6ecf24a6d71
)

replace github.com/minio/minio => github.com/brimsec/minio v0.0.0-20200716214025-90d56627f750
