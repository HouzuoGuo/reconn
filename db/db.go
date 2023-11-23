package db

import (
	"database/sql"
	"fmt"

	"github.com/HouzuoGuo/reconn-voice-clone/db/dbgen"
	_ "github.com/lib/pq"
)

// Config describes database connection parameters.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

// Connect to the postgresql database with TLS verification.
func Connect(conf Config) (*sql.DB, *dbgen.Queries, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=verify-full", conf.Host, conf.Port, conf.User, conf.Password, conf.Database)
	lowLevelDB, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, nil, err
	}
	if err := lowLevelDB.Ping(); err != nil {
		return nil, nil, err
	}
	reconnDB := dbgen.New(lowLevelDB)
	return lowLevelDB, reconnDB, nil
}
