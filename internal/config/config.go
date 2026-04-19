package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Viettel  ViettelConfig
	MISA     MISAConfig
	Provider ProviderConfig
	Worker   WorkerConfig
	Log      LogConfig
	Seller   SellerConfig
}

type ServerConfig struct {
	Port         int           `mapstructure:"SERVER_PORT"`
	ReadTimeout  time.Duration `mapstructure:"SERVER_READ_TIMEOUT"`
	WriteTimeout time.Duration `mapstructure:"SERVER_WRITE_TIMEOUT"`
}

type DatabaseConfig struct {
	Host         string `mapstructure:"DB_HOST"`
	Port         int    `mapstructure:"DB_PORT"`
	User         string `mapstructure:"DB_USER"`
	Password     string `mapstructure:"DB_PASSWORD"`
	DBName       string `mapstructure:"DB_NAME"`
	SSLMode      string `mapstructure:"DB_SSLMODE"`
	MaxOpenConns int    `mapstructure:"DB_MAX_OPEN_CONNS"`
	MaxIdleConns int    `mapstructure:"DB_MAX_IDLE_CONNS"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.SSLMode,
	)
}

type ViettelConfig struct {
	BaseURL           string        `mapstructure:"THIRD_PARTY_BASE_URL"`
	AuthURL           string        `mapstructure:"THIRD_PARTY_AUTH_URL"`
	CreateInvoicePath string        `mapstructure:"THIRD_PARTY_CREATE_PATH"`
	QueryStatusPath   string        `mapstructure:"THIRD_PARTY_QUERY_PATH"`
	ReportToAuthorityPath string    `mapstructure:"THIRD_PARTY_REPORT_TO_AUTHORITY_PATH"`
	GetFilePath           string    `mapstructure:"THIRD_PARTY_GET_FILE_PATH"`
	SupplierCode      string        `mapstructure:"THIRD_PARTY_SUPPLIER_CODE"`
	Username          string        `mapstructure:"THIRD_PARTY_USERNAME"`
	Password          string        `mapstructure:"THIRD_PARTY_PASSWORD"`
	APIKey            string        `mapstructure:"THIRD_PARTY_API_KEY"` // reserved for API-key-based providers
	Timeout           time.Duration `mapstructure:"THIRD_PARTY_TIMEOUT"`
	TemplateCode      string        `mapstructure:"THIRD_PARTY_TEMPLATE_CODE"`
	InvoiceSeries     string        `mapstructure:"THIRD_PARTY_INVOICE_SERIES"`
	InvoiceType       string        `mapstructure:"THIRD_PARTY_INVOICE_TYPE"`
}

// MISAConfig holds configuration for the MISA MeInvoice integration API.
type MISAConfig struct {
	BaseURL         string        `mapstructure:"MISA_BASE_URL"`
	AppID           string        `mapstructure:"MISA_APP_ID"`
	TaxCode         string        `mapstructure:"MISA_TAX_CODE"`
	Username        string        `mapstructure:"MISA_USERNAME"`
	Password        string        `mapstructure:"MISA_PASSWORD"`
	InvoiceWithCode bool          `mapstructure:"MISA_INVOICE_WITH_CODE"`
	InvoiceCalcu    bool          `mapstructure:"MISA_INVOICE_CALCU"`
	SignType         int          `mapstructure:"MISA_SIGN_TYPE"`
	Timeout         time.Duration `mapstructure:"MISA_TIMEOUT"`
}

// ProviderConfig controls which e-invoice provider is used by default.
type ProviderConfig struct {
	Default string `mapstructure:"DEFAULT_PROVIDER"`
}

type WorkerConfig struct {
	PoolSize     int           `mapstructure:"WORKER_POOL_SIZE"`
	QueueSize    int           `mapstructure:"WORKER_QUEUE_SIZE"`
	PollInterval time.Duration `mapstructure:"WORKER_POLL_INTERVAL"`
	MaxRetries   int           `mapstructure:"WORKER_MAX_RETRIES"`
}

type LogConfig struct {
	Level  string `mapstructure:"LOG_LEVEL"`
	Format string `mapstructure:"LOG_FORMAT"`
}

type SellerConfig struct {
	LegalName   string `mapstructure:"SELLER_LEGAL_NAME"`
	TaxCode     string `mapstructure:"SELLER_TAX_CODE"`
	Address     string `mapstructure:"SELLER_ADDRESS"`
	PhoneNumber string `mapstructure:"SELLER_PHONE"`
	Email       string `mapstructure:"SELLER_EMAIL"`
	BankName    string `mapstructure:"SELLER_BANK_NAME"`
	BankAccount string `mapstructure:"SELLER_BANK_ACCOUNT"`
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	// Server defaults
	viper.SetDefault("SERVER_PORT", 8080)
	viper.SetDefault("SERVER_READ_TIMEOUT", "15s")
	viper.SetDefault("SERVER_WRITE_TIMEOUT", "15s")

	// Database defaults
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", 5432)
	viper.SetDefault("DB_USER", "taxmodule")
	viper.SetDefault("DB_PASSWORD", "secret")
	viper.SetDefault("DB_NAME", "tax_module")
	viper.SetDefault("DB_SSLMODE", "disable")
	viper.SetDefault("DB_MAX_OPEN_CONNS", 25)
	viper.SetDefault("DB_MAX_IDLE_CONNS", 5)

	viper.SetDefault("THIRD_PARTY_BASE_URL", "https://api-vinvoice.viettel.vn/services/einvoiceapplication/api")
	viper.SetDefault("THIRD_PARTY_AUTH_URL", "https://api-vinvoice.viettel.vn/auth/login")
	viper.SetDefault("THIRD_PARTY_CREATE_PATH", "/InvoiceAPI/InvoiceWS/createInvoice")
	viper.SetDefault("THIRD_PARTY_QUERY_PATH", "/InvoiceAPI/InvoiceWS/searchInvoiceByTransactionUuid")
	viper.SetDefault("THIRD_PARTY_REPORT_TO_AUTHORITY_PATH", "/InvoiceAPI/InvoiceWS/sendInvoiceByTransactionUuid")
	viper.SetDefault("THIRD_PARTY_GET_FILE_PATH", "/InvoiceAPI/InvoiceUtilsWS/getInvoiceRepresentationFile")
	viper.SetDefault("THIRD_PARTY_SUPPLIER_CODE", "")
	viper.SetDefault("THIRD_PARTY_USERNAME", "")
	viper.SetDefault("THIRD_PARTY_PASSWORD", "")
	viper.SetDefault("THIRD_PARTY_API_KEY", "")
	viper.SetDefault("THIRD_PARTY_TIMEOUT", "95s")
	viper.SetDefault("THIRD_PARTY_TEMPLATE_CODE", "")
	viper.SetDefault("THIRD_PARTY_INVOICE_SERIES", "")
	viper.SetDefault("THIRD_PARTY_INVOICE_TYPE", "1")

	// MISA MeInvoice defaults
	viper.SetDefault("MISA_BASE_URL", "https://api.meinvoice.vn/api/integration")
	viper.SetDefault("MISA_APP_ID", "")
	viper.SetDefault("MISA_TAX_CODE", "")
	viper.SetDefault("MISA_USERNAME", "")
	viper.SetDefault("MISA_PASSWORD", "")
	viper.SetDefault("MISA_INVOICE_WITH_CODE", true)
	viper.SetDefault("MISA_INVOICE_CALCU", false)
	viper.SetDefault("MISA_SIGN_TYPE", 2)
	viper.SetDefault("MISA_TIMEOUT", "30s")

	viper.SetDefault("DEFAULT_PROVIDER", "viettel")

	viper.SetDefault("WORKER_POOL_SIZE", 10)
	viper.SetDefault("WORKER_QUEUE_SIZE", 100)
	viper.SetDefault("WORKER_POLL_INTERVAL", "60s")
	viper.SetDefault("WORKER_MAX_RETRIES", 5)

	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("LOG_FORMAT", "console")

	// Seller defaults
	viper.SetDefault("SELLER_LEGAL_NAME", "")
	viper.SetDefault("SELLER_TAX_CODE", "")
	viper.SetDefault("SELLER_ADDRESS", "")
	viper.SetDefault("SELLER_PHONE", "")
	viper.SetDefault("SELLER_EMAIL", "")
	viper.SetDefault("SELLER_BANK_NAME", "")
	viper.SetDefault("SELLER_BANK_ACCOUNT", "")

	// Read .env file (ignore error if not found — env vars still work)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			if !strings.Contains(err.Error(), "no such file") {
				return nil, fmt.Errorf("reading config file: %w", err)
			}
		}
	}

	cfg := &Config{
		Server: ServerConfig{
			Port:         viper.GetInt("SERVER_PORT"),
			ReadTimeout:  viper.GetDuration("SERVER_READ_TIMEOUT"),
			WriteTimeout: viper.GetDuration("SERVER_WRITE_TIMEOUT"),
		},
		Database: DatabaseConfig{
			Host:         viper.GetString("DB_HOST"),
			Port:         viper.GetInt("DB_PORT"),
			User:         viper.GetString("DB_USER"),
			Password:     viper.GetString("DB_PASSWORD"),
			DBName:       viper.GetString("DB_NAME"),
			SSLMode:      viper.GetString("DB_SSLMODE"),
			MaxOpenConns: viper.GetInt("DB_MAX_OPEN_CONNS"),
			MaxIdleConns: viper.GetInt("DB_MAX_IDLE_CONNS"),
		},
		Viettel: ViettelConfig{
			BaseURL:               viper.GetString("THIRD_PARTY_BASE_URL"),
			AuthURL:               viper.GetString("THIRD_PARTY_AUTH_URL"),
			CreateInvoicePath:     viper.GetString("THIRD_PARTY_CREATE_PATH"),
			QueryStatusPath:       viper.GetString("THIRD_PARTY_QUERY_PATH"),
			ReportToAuthorityPath: viper.GetString("THIRD_PARTY_REPORT_TO_AUTHORITY_PATH"),
			GetFilePath:           viper.GetString("THIRD_PARTY_GET_FILE_PATH"),
			SupplierCode:          viper.GetString("THIRD_PARTY_SUPPLIER_CODE"),
			Username:              viper.GetString("THIRD_PARTY_USERNAME"),
			Password:              viper.GetString("THIRD_PARTY_PASSWORD"),
			APIKey:                viper.GetString("THIRD_PARTY_API_KEY"),
			Timeout:               viper.GetDuration("THIRD_PARTY_TIMEOUT"),
			TemplateCode:          viper.GetString("THIRD_PARTY_TEMPLATE_CODE"),
			InvoiceSeries:         viper.GetString("THIRD_PARTY_INVOICE_SERIES"),
			InvoiceType:           viper.GetString("THIRD_PARTY_INVOICE_TYPE"),
		},
		MISA: MISAConfig{
			BaseURL:         viper.GetString("MISA_BASE_URL"),
			AppID:           viper.GetString("MISA_APP_ID"),
			TaxCode:         viper.GetString("MISA_TAX_CODE"),
			Username:        viper.GetString("MISA_USERNAME"),
			Password:        viper.GetString("MISA_PASSWORD"),
			InvoiceWithCode: viper.GetBool("MISA_INVOICE_WITH_CODE"),
			InvoiceCalcu:    viper.GetBool("MISA_INVOICE_CALCU"),
			SignType:        viper.GetInt("MISA_SIGN_TYPE"),
			Timeout:         viper.GetDuration("MISA_TIMEOUT"),
		},
		Provider: ProviderConfig{
			Default: viper.GetString("DEFAULT_PROVIDER"),
		},
		Worker: WorkerConfig{
			PoolSize:     viper.GetInt("WORKER_POOL_SIZE"),
			QueueSize:    viper.GetInt("WORKER_QUEUE_SIZE"),
			PollInterval: viper.GetDuration("WORKER_POLL_INTERVAL"),
			MaxRetries:   viper.GetInt("WORKER_MAX_RETRIES"),
		},
		Log: LogConfig{
			Level:  viper.GetString("LOG_LEVEL"),
			Format: viper.GetString("LOG_FORMAT"),
		},
		Seller: SellerConfig{
			LegalName:   viper.GetString("SELLER_LEGAL_NAME"),
			TaxCode:     viper.GetString("SELLER_TAX_CODE"),
			Address:     viper.GetString("SELLER_ADDRESS"),
			PhoneNumber: viper.GetString("SELLER_PHONE"),
			Email:       viper.GetString("SELLER_EMAIL"),
			BankName:    viper.GetString("SELLER_BANK_NAME"),
			BankAccount: viper.GetString("SELLER_BANK_ACCOUNT"),
		},
	}

	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	return cfg, nil
}

func validate(cfg *Config) error {
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid SERVER_PORT: %d", cfg.Server.Port)
	}
	if cfg.Database.Host == "" {
		return fmt.Errorf("DB_HOST is required")
	}
	if cfg.Database.DBName == "" {
		return fmt.Errorf("DB_NAME is required")
	}
	return nil
}
