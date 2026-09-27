package dbclient

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// DBHandle provides a stable abstraction over internal database tooling.
type DBHandle struct {
	DB *gorm.DB
}

func NewDBHandle(db *gorm.DB) *DBHandle {
	return &DBHandle{
		DB: db,
	}
}

func (h *DBHandle) ConfigureDBPool(maxOpenConns int, maxIdleConns int) error {
	if maxOpenConns <= 0 {
		return errors.New("maxOpenConns must be greater than zero")
	}

	if maxIdleConns <= 0 {
		return errors.New("maxIdleConns must be greater than zero")
	}

	if maxIdleConns > maxOpenConns {
		return errors.New("maxIdleConns cannot exceed maxOpenConns")
	}

	// https://gorm.io/docs/generic_interface.html#Connection-Pool

	sqlDB, err := h.DB.DB()
	if err != nil {
		return err
	}

	// maximum number of open connections
	sqlDB.SetMaxOpenConns(maxOpenConns)

	// maximum number of idle connections (those kept open while not being used)
	sqlDB.SetMaxIdleConns(maxIdleConns)

	// maximum amount of time a connection may be reused
	sqlDB.SetConnMaxLifetime(time.Hour) // arbitrary lifetime for now

	return nil
}
