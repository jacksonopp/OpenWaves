package db

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open returns a *gorm.DB for the given driver and DSN.
// Supported drivers: "sqlite", "postgres", "mysql", "mssql".
func Open(driver, dsn string) (*gorm.DB, error) {
	cfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
	switch driver {
	case "sqlite":
		return gorm.Open(sqlite.Open(dsn), cfg)
	case "postgres":
		return gorm.Open(postgres.Open(dsn), cfg)
	case "mysql":
		return gorm.Open(mysql.Open(dsn), cfg)
	case "mssql":
		return gorm.Open(sqlserver.Open(dsn), cfg)
	default:
		return nil, fmt.Errorf("db: unsupported driver %q (valid: sqlite, postgres, mysql, mssql)", driver)
	}
}

// Migrate runs AutoMigrate for all given models.
func Migrate(db *gorm.DB, models ...interface{}) error {
	return db.AutoMigrate(models...)
}
