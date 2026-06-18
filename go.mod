module github.com/observeinc/aws-sam-apps

go 1.25.7

require (
	github.com/aws/aws-lambda-go v1.54.0
	github.com/aws/aws-sdk-go-v2 v1.41.7
	github.com/aws/aws-sdk-go-v2/config v1.31.9
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.17.63
	github.com/aws/aws-sdk-go-v2/service/amp v1.40.1
	github.com/aws/aws-sdk-go-v2/service/apigateway v1.35.4
	github.com/aws/aws-sdk-go-v2/service/apigatewayv2 v1.32.4
	github.com/aws/aws-sdk-go-v2/service/autoscaling v1.59.1
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.50.1
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.45.14
	github.com/aws/aws-sdk-go-v2/service/databasemigrationservice v1.57.4
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.253.0
	github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi v1.30.4
	github.com/aws/aws-sdk-go-v2/service/s3 v1.77.1
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.34.19
	github.com/aws/aws-sdk-go-v2/service/shield v1.34.4
	github.com/aws/aws-sdk-go-v2/service/sqs v1.42.27
	github.com/aws/aws-sdk-go-v2/service/storagegateway v1.42.4
	github.com/aws/smithy-go v1.26.0
	github.com/go-logr/logr v1.4.3
	github.com/google/go-cmp v0.7.0
	github.com/hashicorp/go-retryablehttp v0.7.8
	github.com/klauspost/pgzip v1.2.6
	github.com/lithammer/dedent v1.1.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/pkg/profile v1.7.0
	github.com/prometheus-community/yet-another-cloudwatch-exporter v0.63.0
	github.com/sethvargo/go-envconfig v1.3.0
	go.opentelemetry.io/contrib/detectors/aws/lambda v0.69.0
	go.opentelemetry.io/contrib/exporters/autoexport v0.69.0
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda v0.69.0
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws v0.69.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.69.0
	go.opentelemetry.io/contrib/propagators/b3 v1.44.0
	go.opentelemetry.io/otel v1.44.0
	go.opentelemetry.io/otel/sdk v1.44.0
	go.opentelemetry.io/otel/trace v1.44.0
	golang.org/x/sync v0.20.0
	golang.org/x/time v0.15.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/aws/aws-sdk-go v1.55.7 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.10 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.18.13 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.7 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.33 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.57.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.6.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.12.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.18.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/sns v1.39.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.29.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.34.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.38.4 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/felixge/fgprof v0.9.3 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/pprof v0.0.0-20211214055906-6f57359322fd // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grafana/regexp v0.0.0-20240607082908-2cb410fa05da // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.5 // indirect
	github.com/prometheus/otlptranslator v1.0.0 // indirect
	github.com/prometheus/procfs v0.20.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/bridges/prometheus v0.69.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.20.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.20.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/prometheus v0.66.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutlog v0.20.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.44.0 // indirect
	go.opentelemetry.io/otel/log v0.20.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.20.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.44.0 // indirect
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	golang.org/x/exp v0.0.0-20240823005443-9b4947da3948 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/grpc v1.81.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
