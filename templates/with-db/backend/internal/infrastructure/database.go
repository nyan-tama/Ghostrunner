package infrastructure

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Database はDB接続を管理する構造体。
// グローバル変数を使わず、構造体ベースのDIで依存性を注入する。
type Database struct {
	db *gorm.DB
}

// NewDatabase は PostgreSQL への接続を確立し、Database 構造体を返す。
func NewDatabase(databaseURL string) (*Database, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &Database{db: db}, nil
}

// DB は GORM DB インスタンスを返す。
func (d *Database) DB() *gorm.DB {
	return d.db
}

// Close はDB接続をクローズする。
func (d *Database) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	return sqlDB.Close()
}

// AutoMigrate は指定されたモデルのマイグレーションを実行する。
func (d *Database) AutoMigrate(models ...interface{}) error {
	if err := d.db.AutoMigrate(models...); err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}
	return nil
}
