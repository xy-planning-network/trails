package postgres

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Migration is used to hold the database key and function for creating the migration.
type Migration struct {
	Executor func(*gorm.DB) error
	Key      string
}

func (m Migration) execute(db *gorm.DB) error {
	// Start transaction
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// Run migration logic
	if err := m.Executor(tx); err != nil {
		tx.Rollback()
		return err
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return err
	}

	return nil
}

func MigrateUp(db *gorm.DB, schema string, migrations []Migration) error {
	// Ensure schema exists
	ensureSchema(db, schema)

	// Ensure migrations table exists
	ensureMigrationsTable(db)

	// Run migrations
	migrationsToRun := determineMigrationsToRun(db, migrations)
	for _, m := range migrationsToRun {
		if err := m.execute(db); err != nil {
			fmt.Println(m.Key)
			panic(err)
		}

		// There was no error, so create a record for the migration
		createMigrationRecord(db, m.Key)
	}

	return nil
}

func ensureSchema(db *gorm.DB, schema string) {
	err := db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schema)).Error
	if err != nil {
		panic(fmt.Sprintf("Error creating %s schema. Cannot continue: %s", schema, err))
	}
}

func ensureMigrationsTable(db *gorm.DB) {
	err := db.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id SERIAL PRIMARY KEY,
			ran_at bigint,
			key text,
			CONSTRAINT migrations_key UNIQUE (key)
		)
	`).Error
	if err != nil {
		panic(fmt.Sprintf("Error creating migrations table. Cannot continue: %s", err))
	}
}

type migrationKeyCol struct {
	Key string
}

func determineMigrationsToRun(db *gorm.DB, allMigrations []Migration) []Migration {
	ranMigrations := []migrationKeyCol{}
	r := db.Raw("SELECT key FROM migrations;")
	if r.Error != nil {
		panic(fmt.Sprintf("Error fetching ran migrations. Cannot continue: %s", r.Error))
	}
	r.Scan(&ranMigrations)

	// If key is empty, we haven't ran *any* migrations
	if len(ranMigrations) == 0 {
		return allMigrations
	}

	// Compare ran migration keys to all migration keys to determine which need to run
	migrationsToRun := []Migration{}
	for _, migrationToCheck := range allMigrations {
		itsBeenRun := false
		// If this migration has already ran, continue to next iteration (don't add to list to run)
		for _, ranMigration := range ranMigrations {
			if migrationToCheck.Key == ranMigration.Key {
				itsBeenRun = true
				continue
			}
		}

		if !itsBeenRun {
			migrationsToRun = append(migrationsToRun, migrationToCheck)
		}
	}

	return migrationsToRun
}

func createMigrationRecord(db *gorm.DB, key string) {
	err := db.Exec(`INSERT INTO migrations (key, ran_at) VALUES (?, ?)`, key, time.Now().Unix()).Error
	if err != nil {
		panic(fmt.Sprintf("Error creating migration. Cannot continue: %s", err))
	}
}
