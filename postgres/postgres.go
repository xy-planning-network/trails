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
	Host        string
	IsTestDB    bool
	MaxIdleCxns int
	Name        string
	Password    string
	Port        string
	Schema      string
	SSLMode     string
	URL         string
	User        string
}

// Connect creates a database connection through GORM according to the connection config.
//
// Run migrations by passing DB into MigrateUp.
func Connect(config *CxnConfig, env trails.Environment) (*gorm.DB, error) {
	if config.Schema == "" {
		config.Schema = "public"
	}
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

	gormDB, err := gorm.Open(postgres.Open(buildCxnStr(config)), &gorm.Config{
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

	db, err := gormDB.DB()
	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(config.MaxIdleCxns)

	if config.IsTestDB {
		if err := gormDB.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", config.Schema)).Error; err != nil {
			return nil, err
		}
	}

	ensureSchema(gormDB, config.Schema)

	return gormDB, nil
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
func WipeDB(db *gorm.DB, schema string) error {
	var tables []string
	err := db.
		Table("information_schema.tables").
		Select("table_name").
		Where("table_schema = ?", schema).
		Not("table_type = ?", "VIEW").
		Pluck("table_name", &tables).
		Error
	if err != nil {
		return err
	}

	if len(tables) == 0 {
		return nil
	}

	return db.Exec(fmt.Sprintf("TRUNCATE %s CASCADE;", strings.Join(tables, ", "))).Error
}

// A Scope returns a function that fulfills the interface expected by gorm.Scopes.
//
// A Scope can be converted to a subquery by passing in a *gorm.DB instance.
// For example, given this scope:
//
//	func ActiveUsers() Scope {
//	    return func(dbx *gorm.DB) *gorm.DB {
//	       return dbx.Where("state = ?", "active")
//	    }
//	}
//
// The scope turns into a subquery like so:
//
//	db.Preload("Members", ActiveUsers()(db)).Where("role = ?", "owner").Find(&owners)
//
// Cf. [*gorm.DB.Scopes], https://gorm.io/docs/scopes.html
type Scope func(*gorm.DB) *gorm.DB
