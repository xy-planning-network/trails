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
	cxnStr = "host=%s port=%s dbname=%s user=%s password=%s sslmode=%s"
	// FIXME(dlk): change to PostgreSQL error codes
	violatesFK = "violates foreign key constraint"

	// Associations can be passed to Preload
	// and triggers it to load all top-level associations.
	Associations = clause.Associations
)

var (
	// safeGORMSession is the accepted set of parameters to pass to *gorm.DB.Session
	// when needing a clean pointer but want to preserve the current query.
	//
	// For Initialized and NewDB, the risk is losing *gorm.Statement values set earlier in the query chain
	// due to subtlties in GORM's cloning of *gorm.Statement.
	//
	// Frankly, the GORM code is a bit inscrutable and so this is a best guess to be improved upon.
	// Relevant code:
	// - https://github.com/go-gorm/gorm/blob/b88148363a954f69fa680b152dfd96a94ffea1e1/gorm.go#L324-L348
	// - https://github.com/go-gorm/gorm/blob/b88148363a954f69fa680b152dfd96a94ffea1e1/gorm.go#L434-L461
	// - https://github.com/go-gorm/gorm/blob/b88148363a954f69fa680b152dfd96a94ffea1e1/statement.go#L529-L581
	safeGORMSession = &gorm.Session{Initialized: true, NewDB: false}
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

// Connect creates a database connection and *DB
// through GORM according to the connection config.
//
// Run migrations by passing DB into MigrateUp.
func Connect(cfg Config) (*DB, error) {
	gdb, err := ConnectRaw(cfg)
	if err != nil {
		return nil, err
	}

	return NewDB(gdb), nil
}

// ConnectRaw creates a database connection and *gorm.DB
// through GORM according to the connection config.
//
// Run migrations by passing DB into MigrateUp.
func ConnectRaw(cfg Config) (*gorm.DB, error) {
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

	if cfg.LogSilent {
		c.LogLevel = logger.Silent
	}

	// FIXME(dlk): use slog, trails/ranger's newSlogger?
	l := logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), c)

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

	return gormDB, nil
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
