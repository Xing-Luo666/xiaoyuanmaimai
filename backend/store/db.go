package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/go-sql-driver/mysql"
)

type DBConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbName"`
}

type DBStore struct {
	mu  sync.RWMutex
	db  *sql.DB
	cfg DBConfig
}

func NewDBStore(cfgFile string) *DBStore {
	s := &DBStore{}

	cfg, err := s.loadConfig(cfgFile)
	if err != nil {
		cfg = DBConfig{
			Host:     "127.0.0.1",
			Port:     "3306",
			User:     "root",
			Password: "114514",
			DBName:   "school_trade",
		}
	}
	s.cfg = cfg

	if err := s.connect(); err != nil {
		fmt.Printf("数据库连接失败: %v\n", err)
	}
	if s.db != nil {
		if err := s.initTables(); err != nil {
			fmt.Printf("初始化数据表失败: %v\n", err)
		}
	}

	return s
}

func (s *DBStore) loadConfig(cfgFile string) (DBConfig, error) {
	// 默认值
	cfg := DBConfig{
		Host:     "127.0.0.1",
		Port:     "3306",
		User:     "root",
		Password: "114514",
		DBName:   "school_trade",
	}

	// 环境变量覆盖（优先级最高）
	if v := os.Getenv("DB_HOST"); v != "" {
		cfg.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		cfg.Port = v
	}
	if v := os.Getenv("DB_USER"); v != "" {
		cfg.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		cfg.Password = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		cfg.DBName = v
	}

	// 配置文件（环境变量未设置时生效）
	data, err := os.ReadFile(cfgFile)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	var fileCfg DBConfig
	if err := json.Unmarshal(data, &fileCfg); err != nil {
		return cfg, err
	}
	// 环境变量未设置时才用配置文件的值
	if os.Getenv("DB_HOST") == "" && fileCfg.Host != "" {
		cfg.Host = fileCfg.Host
	}
	if os.Getenv("DB_PORT") == "" && fileCfg.Port != "" {
		cfg.Port = fileCfg.Port
	}
	if os.Getenv("DB_USER") == "" && fileCfg.User != "" {
		cfg.User = fileCfg.User
	}
	if os.Getenv("DB_PASSWORD") == "" && fileCfg.Password != "" {
		cfg.Password = fileCfg.Password
	}
	if os.Getenv("DB_NAME") == "" && fileCfg.DBName != "" {
		cfg.DBName = fileCfg.DBName
	}
	return cfg, nil
}

func (s *DBStore) SaveConfig(cfgFile string, cfg DBConfig) error {
	dir := filepath.Dir(cfgFile)
	os.MkdirAll(dir, 0755)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfgFile, data, 0644)
}

func (s *DBStore) connect() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db != nil {
		s.db.Close()
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		s.cfg.User, s.cfg.Password, s.cfg.Host, s.cfg.Port, s.cfg.DBName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return err
	}

	s.db = db
	return nil
}

func (s *DBStore) Reconnect(cfgFile string, cfg DBConfig) error {
	if err := s.SaveConfig(cfgFile, cfg); err != nil {
		return err
	}
	s.cfg = cfg
	return s.connect()
}

func (s *DBStore) GetConfig() DBConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *DBStore) GetDB() *sql.DB {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.db
}

func (s *DBStore) initTables() error {
	db := s.GetDB()
	if db == nil {
		return fmt.Errorf("数据库未连接")
	}

	tables := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(64) PRIMARY KEY,
			username VARCHAR(64) NOT NULL UNIQUE,
			password VARCHAR(255) NOT NULL,
			nickname VARCHAR(64) NOT NULL,
			avatar VARCHAR(255) DEFAULT '',
			phone VARCHAR(20) DEFAULT '',
			email VARCHAR(64) DEFAULT '',
			role VARCHAR(20) DEFAULT 'student',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS products (
			id VARCHAR(64) PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			description TEXT,
			category VARCHAR(32) NOT NULL,
			price DECIMAL(10,2) NOT NULL,
			ori_price DECIMAL(10,2) DEFAULT 0,
			images JSON,
			cond VARCHAR(20) DEFAULT 'good',
			campus VARCHAR(20) DEFAULT 'main',
			seller_id VARCHAR(64) NOT NULL,
			seller_name VARCHAR(64) NOT NULL,
			status VARCHAR(20) DEFAULT 'selling',
			view_count INT DEFAULT 0,
			like_count INT DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			INDEX idx_category (category),
			INDEX idx_seller (seller_id),
			INDEX idx_status (status)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS orders (
			id VARCHAR(64) PRIMARY KEY,
			product_id VARCHAR(64) NOT NULL,
			product_title VARCHAR(255) NOT NULL,
			buyer_id VARCHAR(64) NOT NULL,
			buyer_name VARCHAR(64) NOT NULL,
			seller_id VARCHAR(64) NOT NULL,
			seller_name VARCHAR(64) NOT NULL,
			price DECIMAL(10,2) NOT NULL,
			status VARCHAR(20) DEFAULT 'pending',
			message TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			INDEX idx_buyer (buyer_id),
			INDEX idx_seller (seller_id),
			INDEX idx_product (product_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}

	for _, ddl := range tables {
		if _, err := db.Exec(ddl); err != nil {
			return fmt.Errorf("建表失败: %w", err)
		}
	}

	return nil
}

func (s *DBStore) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		s.db.Close()
	}
}
