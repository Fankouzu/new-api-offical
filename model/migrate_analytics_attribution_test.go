package model

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupAnalyticsAttributionTestDB opens an isolated in-memory SQLite database and
// restores the package-level DB / dialect flags on cleanup.
func setupAnalyticsAttributionTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	oldDB := DB
	oldLogDB := LOG_DB
	oldUsingSQLite := common.UsingSQLite
	oldUsingMySQL := common.UsingMySQL
	oldUsingPostgreSQL := common.UsingPostgreSQL

	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	DB = db
	LOG_DB = db

	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		DB = oldDB
		LOG_DB = oldLogDB
		common.UsingSQLite = oldUsingSQLite
		common.UsingMySQL = oldUsingMySQL
		common.UsingPostgreSQL = oldUsingPostgreSQL
	})

	return db
}

// TestMigrateAnalyticsAttributionColumns_AddsMissingColumn reproduces the production
// failure where a database created by an older binary is missing analytics_attribution
// (SQLSTATE 42703 at token-creation time). It simulates the stale schema by dropping
// the column after AutoMigrate, then confirms the migration repairs it on every table
// that stores GA4 conversion attribution.
func TestMigrateAnalyticsAttributionColumns_AddsMissingColumn(t *testing.T) {
	db := setupAnalyticsAttributionTestDB(t)

	// Create all three attribution tables with the current schema.
	require.NoError(t, db.AutoMigrate(&Token{}, &TopUp{}, &SubscriptionOrder{}))

	// Simulate a database created before analytics_attribution existed by dropping
	// the column from every table, the exact state that triggers SQLSTATE 42703.
	for _, m := range analyticsAttributionModels {
		require.NoError(t, db.Migrator().DropColumn(m, "AnalyticsAttribution"),
			"failed to drop analytics_attribution to simulate stale schema")
		require.False(t, db.Migrator().HasColumn(m, "AnalyticsAttribution"),
			"precondition: analytics_attribution should be absent before migration")
	}

	// The repair must add the missing column to every affected table.
	require.NoError(t, migrateAnalyticsAttributionColumns())

	for _, m := range analyticsAttributionModels {
		require.True(t, db.Migrator().HasColumn(m, "AnalyticsAttribution"),
			"analytics_attribution was not restored after migration")
	}
}

// TestMigrateAnalyticsAttributionColumns_SkipsMissingTables confirms the migration never
// errors when a target table has not been created yet (e.g. a partially-migrated schema).
// It must skip absent tables gracefully rather than emitting a confusing
// "relation ... does not exist" from AddColumn — matching the HasTable guard used by the
// sibling migrations migrateTokenModelLimitsToText / migrateSubscriptionPlanPriceAmount.
func TestMigrateAnalyticsAttributionColumns_SkipsMissingTables(t *testing.T) {
	db := setupAnalyticsAttributionTestDB(t)

	// No AutoMigrate — none of the attribution tables exist yet.
	require.False(t, db.Migrator().HasTable(&Token{}), "precondition: tokens table should not exist")

	// Must return nil: every absent table is skipped, not treated as a hard error.
	require.NoError(t, migrateAnalyticsAttributionColumns())
}

// TestMigrateAnalyticsAttributionColumns_Idempotent confirms the migration is a safe
// no-op when the column already exists (the common case after a normal AutoMigrate).
func TestMigrateAnalyticsAttributionColumns_Idempotent(t *testing.T) {
	db := setupAnalyticsAttributionTestDB(t)
	require.NoError(t, db.AutoMigrate(&Token{}, &TopUp{}, &SubscriptionOrder{}))

	// Column already present -> migration must succeed without changing anything.
	require.NoError(t, migrateAnalyticsAttributionColumns())

	for _, m := range analyticsAttributionModels {
		require.True(t, db.Migrator().HasColumn(m, "AnalyticsAttribution"),
			"analytics_attribution should remain present after idempotent run")
	}
}
