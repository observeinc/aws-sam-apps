package tracing

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
	"go.opentelemetry.io/otel/attribute"
)

var AttributeSetters = []otelaws.AttributeSetter{AttributeSetter}

func AttributeSetter(_ context.Context, in middleware.InitializeInput) (attrs []attribute.KeyValue) {
	// see https://opentelemetry.io/docs/specs/semconv/object-stores/s3/
	switch v := in.Parameters.(type) {
	case *s3.GetObjectInput:
		attrs = append(attrs,
			attribute.String("aws.s3.bucket", aws.ToString(v.Bucket)),
			attribute.String("aws.s3.key", aws.ToString(v.Key)),
		)
	case *s3.CopyObjectInput:
		attrs = append(attrs,
			attribute.String("aws.s3.bucket", aws.ToString(v.Bucket)),
			attribute.String("aws.s3.key", aws.ToString(v.Key)),
			attribute.String("aws.s3.copy_source", aws.ToString(v.CopySource)),
		)
	case *s3.PutObjectInput:
		attrs = append(attrs,
			attribute.String("aws.s3.bucket", aws.ToString(v.Bucket)),
			attribute.String("aws.s3.key", aws.ToString(v.Key)),
		)
	}
	return
}
