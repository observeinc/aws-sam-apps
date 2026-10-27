// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package translator // import "github.com/observeinc/aws-sam-apps/pkg/xrayotel/translator"

import (
	"fmt"
	"regexp"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"

	awsxray "github.com/observeinc/aws-sam-apps/pkg/xrayotel/awsxray"
)

func addSQLToSpan(sql *awsxray.SQLData, attrs pcommon.Map) error {
	if sql == nil {
		return nil
	}

	// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/c615d2db351929b99e46f7b427f39c12afe15b54/exporter/awsxrayexporter/translator/sql.go#L60
	if sql.URL != nil {
		dbURL, dbName, err := splitSQLURL(*sql.URL)
		if err != nil {
			return err
		}
		attrs.PutStr("db.connection_string", dbURL)
		attrs.PutStr("db.name", dbName)
	}
	// not handling sql.ConnectionString for now because the X-Ray exporter
	// does not support it
	addString(sql.DatabaseType, "db.system", attrs)
	addString(sql.SanitizedQuery, "db.statement", attrs)
	addString(sql.User, "db.user", attrs)
	return nil
}

// SQL URL is of the format: protocol+transport://host:port/dbName?queryParam or protocol+transport:dbName?queryParam
var re = regexp.MustCompile(`^([^/]+:(?://[^/]+/)?)([^\?]+)\??.*$`)

const (
	dbURLI  = 1
	dbNameI = 2
)

func splitSQLURL(rawURL string) (string, string, error) {
	m := re.FindStringSubmatch(rawURL)
	if len(m) == 0 {
		return "", "", fmt.Errorf(
			"failed to parse out the database name in the \"sql.url\" field, rawUrl: %s",
			rawURL,
		)
	}
	return strings.TrimRight(m[dbURLI], "/"), m[dbNameI], nil
}
