package collector

import (
	"context"
)

// Platform is the kind of Postgres service being scraped.
type Platform string

// Platforms the exporter can detect.
const (
	PlatformCommunity     Platform = "community"
	PlatformRDS           Platform = "rds"
	PlatformAurora        Platform = "aurora"
	PlatformCloudSQL      Platform = "cloudsql"
	PlatformAzureFlexible Platform = "azure_flexible"
)

const platformQuery = `
SELECT to_regproc('aurora_version') IS NOT NULL                             AS aurora
     , EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'rds_superuser')     AS rds
     , EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'cloudsqlsuperuser') AS cloudsql
     , EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'azure_pg_admin')    AS azure /*postgres_exporter*/`

// DetectPlatform identifies the Postgres service. Any error falls back to PlatformCommunity.
func DetectPlatform(ctx context.Context, db Querier) (Platform, error) {
	var aurora, rds, cloudsql, azure bool
	if err := db.QueryRow(ctx, platformQuery).Scan(&aurora, &rds, &cloudsql, &azure); err != nil {
		return PlatformCommunity, err
	}
	return platformFrom(aurora, rds, cloudsql, azure), nil
}

// platformFrom resolves the detection flags. Aurora also has rds_superuser, so it is checked first.
func platformFrom(aurora, rds, cloudsql, azure bool) Platform {
	switch {
	case aurora:
		return PlatformAurora
	case rds:
		return PlatformRDS
	case cloudsql:
		return PlatformCloudSQL
	case azure:
		return PlatformAzureFlexible
	default:
		return PlatformCommunity
	}
}
