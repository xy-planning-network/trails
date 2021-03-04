package postgres

import (
	"fmt"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const cxnStr = "host=%s port=%s user=%s dbname=%s sslmode=disable password=%s"

// CxnConfig holds connection information used to connect to a PostgreSQL database.
type CxnConfig struct {
	Host     string
	IsTestDB bool
	Port     string
	Name     string
	Password string
	URL      string
	User     string
}

// Connect creates a database connection through GORM according to the connection config and runs all migrations.
func Connect(config *CxnConfig, migrations []Migration) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(buildCxnStr(config)), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			NameReplacer: strings.NewReplacer("Table", ""),
		},
	})
	if err != nil {
		return nil, err
	}

	if config.IsTestDB {
		if err := db.Exec("DROP SCHEMA IF EXISTS public CASCADE;").Error; err != nil {
			return nil, err
		}
	}

	if err := migrateUp(db, migrations); err != nil {
		return nil, err
	}

	return db, nil
}

func buildCxnStr(config *CxnConfig) string {
	if config.URL != "" {
		return config.URL
	}

	return fmt.Sprintf(
		cxnStr,
		config.Host,
		config.Port,
		config.User,
		config.Name,
		config.Password,
	)
}

// WipeDB queries for all of the tables and then drops the data in this tables.
func WipeDB(db *gorm.DB) error {
	var tables []string
	err := db.
		Table("information_schema.tables").
		Select("table_name").
		Where("table_schema = ?", "public").
		Pluck("table_name", &tables).Error
	if err != nil {
		return err
	}

	return db.Exec(fmt.Sprintf("TRUNCATE %s CASCADE;", strings.Join(tables, ", "))).Error
}
