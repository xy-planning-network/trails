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
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

const (
	// PG Docs: https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-PARAMKEYWORDS
	cxnStr       = "host=%s port=%s dbname=%s user=%s password=%s sslmode=%s"
	violatesFK   = "violates foreign key constraint"
	violatesUniq = "duplicate key value violates unique constraint"

	// Associations can be passed to Preload
	// and triggers it to load all top-level associations.
	Associations = clause.Associations
)

// Config holds connection information used to connect to a PostgreSQL database.
type Config struct {
	Env         trails.Environment
	Host        string
	IsTestDB    bool
	MaxIdleCxns int
	Name        string
	Password    string
	Port        string
	Schema      string
	LogSilent   bool
	SSLMode     string
	URL         string
	User        string
}

// Connect creates a database connection through GORM according to the connection config.
//
// Run migrations by passing DB into MigrateUp.
// func Connect(cfg Config) (*Conn, error) {
func Connect(cfg Config) (*DB, error) {
	if cfg.Schema == "" {
		cfg.Schema = "public"
	}
	// https://gorm.io/docs/logger.html
	c := logger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  logger.Warn,
		IgnoreRecordNotFoundError: true,
		Colorful:                  false,
	}

	if cfg.Env.IsDevelopment() {
		c.Colorful = true
	}
	l := logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), c)
	if cfg.LogSilent {
		l = logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), c)
	}

	gormDB, err := gorm.Open(postgres.Open(cfg.conn()), &gorm.Config{
		Logger: l,
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

	db.SetMaxIdleConns(cfg.MaxIdleCxns)

	if cfg.IsTestDB {
		if err := gormDB.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", cfg.Schema)).Error; err != nil {
			return nil, err
		}
	}

	ensureSchema(gormDB, cfg.Schema)

	return NewDB(gormDB), nil
}

func (cfg Config) conn() string {
	if cfg.URL != "" {
		return cfg.URL
	}

	if cfg.SSLMode == "" {
		// PG Docs: https://www.postgresql.org/docs/current/libpq-ssl.html#LIBPQ-SSL-SSLMODE-STATEMENTS
		cfg.SSLMode = "prefer"
	}

	return fmt.Sprintf(
		cxnStr,
		cfg.Host,
		cfg.Port,
		cfg.Name,
		cfg.User,
		cfg.Password,
		cfg.SSLMode,
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
//	    return func(dbx *postgres.DB) *postgres.DB {
//	       return dbx.Where("state = ?", "active")
//	    }
//	}
//
// The scope turns into a subquery like so:
//
//	db.Preload("Members", ActiveUsers()(db)).Where("role = ?", "owner").Find(&owners)
//
// Cf. [*gorm.DB.Scopes], https://gorm.io/docs/scopes.html
type Scope func(*DB) *DB
