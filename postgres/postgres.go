package postgres

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/xy-planning-network/trails"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// PG Docs: https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-PARAMKEYWORDS
const cxnStr = "host=%s port=%s dbname=%s user=%s password=%s sslmode=%s"

// CxnConfig holds connection information used to connect to a PostgreSQL database.
type CxnConfig struct {
	IsTestDB bool
	URL      string
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
}

// Connect creates a database connection through GORM according to the connection config and runs all migrations.
func Connect(config *CxnConfig, migrations []Migration, env trails.Environment) (*gorm.DB, error) {
	// https://gorm.io/docs/logger.html
	c := logger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  logger.Warn,
		IgnoreRecordNotFoundError: true,
		Colorful:                  false,
	}

	if env.IsDevelopment() {
		c.Colorful = true
	}

	db, err := gorm.Open(postgres.Open(buildCxnStr(config)), &gorm.Config{
		Logger: logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), c),
		NamingStrategy: schema.NamingStrategy{
			NameReplacer: strings.NewReplacer("Table", ""),
		},
		NowFunc: func() time.Time {
			return time.Now().Truncate(time.Microsecond)
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

	if config.SSLMode == "" {
		// PG Docs: https://www.postgresql.org/docs/current/libpq-ssl.html#LIBPQ-SSL-SSLMODE-STATEMENTS
		config.SSLMode = "prefer"
	}

	return fmt.Sprintf(
		cxnStr,
		config.Host,
		config.Port,
		config.Name,
		config.User,
		config.Password,
		config.SSLMode,
	)
}

// WipeDB queries for all of the tables and then drops the data in this tables.
func WipeDB(db *gorm.DB) error {
	var tables []string
	err := db.
		Table("information_schema.tables").
		Select("table_name").
		Where("table_schema = ?", "public").
		Not("table_type = ?", "VIEW").
		Pluck("table_name", &tables).
		Error
	if err != nil {
		return err
	}

	return db.Exec(fmt.Sprintf("TRUNCATE %s CASCADE;", strings.Join(tables, ", "))).Error
}
