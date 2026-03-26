package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func NewMySQL(dsn string) (*gorm.DB, error) {
	cfg, err := mysqlDriver.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid MYSQL_DSN: %w", err)
	}
	if strings.TrimSpace(cfg.DBName) == "" {
		return nil, fmt.Errorf("MYSQL_DSN must include a database name")
	}

	bootstrapCfg := *cfg
	bootstrapCfg.DBName = ""

	sqlDB, err := sql.Open("mysql", bootstrapCfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open mysql bootstrap connection: %w", err)
	}
	defer sqlDB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect mysql server for bootstrap: %w", err)
	}

	createDBSQL := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
		quoteMySQLIdentifier(cfg.DBName),
	)
	if _, err := sqlDB.ExecContext(ctx, createDBSQL); err != nil {
		return nil, fmt.Errorf("failed to create database %q: %w", cfg.DBName, err)
	}

	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}

func quoteMySQLIdentifier(identifier string) string {
	return "`" + strings.ReplaceAll(identifier, "`", "``") + "`"
}
