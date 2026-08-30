package dbclient

import (
	"fmt"

	"github.com/arlogy/deltapilot/internal/envconf"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func ConnectDB(typ string, dsn string) (*DBHandle, error) {
	// https://gorm.io/docs/connecting_to_the_database.html

	var dialector gorm.Dialector
	switch typ {
	case "db_mysql":
		dialector = mysql.Open(dsn)
	case "db_postgresql":
		dialector = postgres.Open(dsn)
	default:
		return nil, fmt.Errorf("failed to select database dialect: unsupported database type: %q", typ)
	}

	config := &gorm.Config{ // https://gorm.io/docs/gorm_config.html
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},

		// prevent accidental logging of sensitive information
		// prevent logging SQL statements from all test cases when only one of them fails, which is confusing
		Logger: logger.Default.LogMode(logger.Silent),

		// https://gorm.io/docs/error_handling.html#Dialect-Translated-Errors
		TranslateError: true,
	}

	db, err := gorm.Open(dialector, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	return NewDBHandle(db), nil
}

func ConnectFromEnv() (*DBHandle, error) {
	if err := envconf.LoadDotEnv(); err != nil {
		return nil, err
	}

	dbType := envconf.GetEnvVar("DB_TYPE")
	dbDSN := envconf.GetEnvVar("DB_DSN")
	return ConnectDB(dbType, dbDSN)
}
