// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package translator // import "github.com/observeinc/aws-sam-apps/pkg/xrayotel/translator"

import (
	"strconv"

	"go.opentelemetry.io/collector/pdata/pcommon"

	awsxray "github.com/observeinc/aws-sam-apps/pkg/xrayotel/awsxray"
)

func addAWSToResource(aws *awsxray.AWSData, attrs pcommon.Map) {
	if aws == nil {
		// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/c615d2db351929b99e46f7b427f39c12afe15b54/exporter/awsxrayexporter/translator/aws.go#L121
		// this implies that the current segment being processed is not generated
		// by an AWS entity.
		attrs.PutStr("cloud.provider", "unknown")
		return
	}

	attrs.PutStr("cloud.provider", "aws")
	addString(aws.AccountID, "cloud.account.id", attrs)

	// based on https://docs.aws.amazon.com/xray/latest/devguide/xray-api-segmentdocuments.html#api-segmentdocuments-aws
	// it's possible to have all cloudwatch_logs, ec2, ecs and beanstalk fields at the same time.
	if cwl := aws.CWLogs; cwl != nil {
		for _, logGroupMetaData := range cwl {
			addStringSlice(logGroupMetaData.Arn, "aws.log.group.arns", attrs)
			addStringSlice(logGroupMetaData.LogGroup, "aws.log.group.names", attrs)
		}
	}

	if ec2 := aws.EC2; ec2 != nil {
		addString(ec2.AvailabilityZone, "cloud.availability_zone", attrs)
		addString(ec2.InstanceID, "host.id", attrs)
		addString(ec2.InstanceSize, "host.type", attrs)
		addString(ec2.AmiID, "host.image.id", attrs)
	}

	if ecs := aws.ECS; ecs != nil {
		addString(ecs.ContainerName, "container.name", attrs)
		addString(ecs.AvailabilityZone, "cloud.availability_zone", attrs)
		addString(ecs.ContainerID, "container.id", attrs)
	}

	if bs := aws.Beanstalk; bs != nil {
		addString(bs.Environment, "service.namespace", attrs)
		if bs.DeploymentID != nil {
			attrs.PutStr("service.instance.id", strconv.FormatInt(*bs.DeploymentID, 10))
		}
		addString(bs.VersionLabel, "service.version", attrs)
	}

	if eks := aws.EKS; eks != nil {
		addString(eks.ContainerID, "container.id", attrs)
		addString(eks.ClusterName, "k8s.cluster.name", attrs)
		addString(eks.Pod, "k8s.pod.name", attrs)
	}
}

func addAWSToSpan(aws *awsxray.AWSData, attrs pcommon.Map) {
	if aws != nil {
		addString(aws.AccountID, awsxray.AWSAccountAttribute, attrs)
		addString(aws.Operation, awsxray.AWSOperationAttribute, attrs)
		addString(aws.RemoteRegion, awsxray.AWSRegionAttribute, attrs)
		addString(aws.RequestID, awsxray.AWSRequestIDAttribute, attrs)
		addString(aws.QueueURL, awsxray.AWSQueueURLAttribute, attrs)
		addString(aws.TableName, awsxray.AWSTableNameAttribute, attrs)
		addInt64(aws.Retries, awsxray.AWSXrayRetriesAttribute, attrs)
	}
}
