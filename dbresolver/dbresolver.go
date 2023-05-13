package dbresolver

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Expected requst header or query
var apiKeyHeader = "x-api-key"

// Use custom type as context key to avoid collisions.
type ContextValue string

// The context key used to access each database connection from req context.
var ConnectionContextKey ContextValue = "connection"

// The context key used to access each database name from req context.
var DatabaseContextKey ContextValue = "database"

type Driver string

const (
	Sqlite   Driver = "sqlite"
	MySQL    Driver = "mysql"
	Postgres Driver = "postgres"
)

type DBDriver struct {
	Driver   Driver
	Database string
}

// DBResolver stores all database connections and config.
// Call ResolveConnection to get the underlying database connection for an API key.
type DBResolver struct {
	conns          map[string]*gorm.DB
	databaseConfig DatabaseConfig
	config         *gorm.Config
}

type Option func(resolver *DBResolver)

func GormConfig(c *gorm.Config) Option {
	return func(resolver *DBResolver) {
		resolver.config = c
	}
}

// Changes the expected header or query param for the API key.
func SetHeaderName(name string) {
	apiKeyHeader = name
}

// Initialize a new DBResolver with a database config, driver, and *gorm.Config.
// The driver argument should be one of "sqlite", "mysql", or "postgres".
// Default ApiKey header/query expected is x-api-key.
// call dbresolver.SetHeaderName to change it.
func New(c DatabaseConfig, options ...Option) (*DBResolver, error) {
	resolver := &DBResolver{
		conns:          make(map[string]*gorm.DB),
		databaseConfig: c,
		config:         &gorm.Config{},
	}

	// Apply all the options
	for _, option := range options {
		option(resolver)
	}

	for _, dbDriver := range c.DatabaseDrivers() {
		database := dbDriver.Database
		var dialect gorm.Dialector

		switch dbDriver.Driver {
		case Sqlite:
			dialect = sqlite.Open(string(database))
		case MySQL:
			dialect = mysql.Open(string(database))
		case Postgres:
			dialect = postgres.Open(string(database))
		default:
			return nil, fmt.Errorf("unsupported database driver: %s", dbDriver.Driver)
		}

		// Create database connection with correct dialect.
		conn, err := createDB(dialect, resolver.config)
		if err != nil {
			return nil, err
		}

		// Add database connection to map of connections
		resolver.conns[database] = conn
	}
	return resolver, nil

}

// createDB connects to a database with the provided driver and returns the connection.
// If the database cannot be created, it panics.
func createDB(dialector gorm.Dialector, config *gorm.Config) (*gorm.DB, error) {
	db, err := gorm.Open(dialector, config)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// ResolveDatabase resolves the database connection from the request APIKey.
func (resolver *DBResolver) resolveConnection(apiKey string) (*gorm.DB, error) {
	databaseMap, ok := resolver.databaseConfig[apiKey]
	if !ok {
		return nil, fmt.Errorf("no database configuration found for API key: %q", apiKey)
	}

	// Get the database databaseName
	databaseName, ok := databaseMap["database"]
	if !ok {
		return nil, fmt.Errorf("no database found for API key %q", apiKey)
	}

	conn, exists := resolver.conns[databaseName]
	if !exists {
		return nil, fmt.Errorf("no valid connection exists for API key: %s", apiKey)
	}
	return conn, nil
}

// ResolveDatabase resolves the database name from the request APIKey.
func (resolver *DBResolver) resolveDatabaseName(apiKey string) (string, error) {
	databaseMap, ok := resolver.databaseConfig[apiKey]
	if !ok {
		return "", fmt.Errorf("no database configuration found for API key: %q", apiKey)
	}
	// Get the database databaseName
	databaseName, ok := databaseMap["database"]
	if !ok {
		return "", fmt.Errorf("no database found for API key %q", apiKey)
	}
	return databaseName, nil
}

func (resolver *DBResolver) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get database name from request API key.
		apiKey := r.Header.Get(apiKeyHeader)
		if apiKey == "" {
			apiKey = r.URL.Query().Get(apiKeyHeader)
		}

		// Get underlying *gorm.DB for API key
		db, err := resolver.resolveConnection(apiKey)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Set the database connection in context
		ctx := context.WithValue(r.Context(), ConnectionContextKey, db)

		// Set the database name in context
		// If we have a connection, there is no expected error
		dbname, _ := resolver.resolveDatabaseName(apiKey)
		ctx = context.WithValue(ctx, DatabaseContextKey, dbname)

		// Serve the request
		next.ServeHTTP(w, r.WithContext(ctx))
	})

}

// DB retrieves the current database connection from the request context.
// This assumes that the handler was run after the middleware.
// Otherwise, this will panic.
func (resolver *DBResolver) DB(r *http.Request) *gorm.DB {
	db := r.Context().Value(ConnectionContextKey).(*gorm.DB)
	return db
}

// ResolveDatabase resolves the database name from the request APIKey.
func (resolver *DBResolver) DBName(r *http.Request) string {
	dbName := r.Context().Value(DatabaseContextKey).(string)
	return dbName
}

// Runs *gorm.DB.AutoMigrate(...) on all databases, creating all the tables.
// The error callback is only called if error is not nil.
// The error callback allows you to ignore certain errors and return true.
// If the error callback returns false this function will panic.
func (resolver *DBResolver) AutoMigrate(models []interface{}, errorCallback func(error) bool) {
	for _, conn := range resolver.conns {
		err := conn.AutoMigrate(models...)
		if err != nil && errorCallback(err) {
			panic(err)
		}
	}
}
