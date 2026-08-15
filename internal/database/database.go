package database

import (
	"fmt"
	"os"
	"path/filepath"

	"quotecore/internal/config"
	"quotecore/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch cfg.Driver {
	case "postgres":
		dialector = postgres.Open(cfg.PostgresDSN)
	case "sqlite":
		if err := os.MkdirAll(filepath.Dir(cfg.SQLitePath), 0o755); err != nil {
			return nil, err
		}
		dialector = sqlite.Open(cfg.SQLitePath)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}
	return gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
}

func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.QuoteTemplate{},
		&model.QuoteTemplateLine{},
		&model.Quote{},
		&model.QuoteItem{},
	); err != nil {
		return err
	}
	if db.Dialector.Name() == "postgres" {
		return db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_quotes_tenant_status ON quotes (tenant_id, status);
			CREATE INDEX IF NOT EXISTS idx_quotes_tenant_no ON quotes (tenant_id, quote_no);
			CREATE INDEX IF NOT EXISTS idx_quote_items_quote ON quote_items (quote_id, sort);
			CREATE INDEX IF NOT EXISTS idx_quote_template_lines_tpl ON quote_template_lines (template_id, sort);
		`).Error
	}
	return nil
}
