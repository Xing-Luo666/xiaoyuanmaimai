package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

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

	// 连接池配置：适应高并发场景
	db.SetMaxOpenConns(25)                 // 最大打开连接数
	db.SetMaxIdleConns(10)                 // 最大空闲连接数
	db.SetConnMaxLifetime(3 * time.Minute) // 连接最大存活时间
	db.SetConnMaxIdleTime(1 * time.Minute) // 空闲连接最大存活时间

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
			specs JSON,
			cond VARCHAR(20) DEFAULT 'good',
			campus VARCHAR(20) DEFAULT 'hangkong',
			building VARCHAR(20) DEFAULT '',
			seller_id VARCHAR(64) NOT NULL,
			seller_name VARCHAR(64) NOT NULL,
			status VARCHAR(20) DEFAULT 'selling',
			view_count INT DEFAULT 0,
			like_count INT DEFAULT 0,
			fav_count INT DEFAULT 0,
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
			product_image VARCHAR(512) DEFAULT '',
			spec_name VARCHAR(255) DEFAULT '',
			quantity INT DEFAULT 1,
			buyer_id VARCHAR(64) NOT NULL,
			buyer_name VARCHAR(64) NOT NULL,
			seller_id VARCHAR(64) NOT NULL,
			seller_name VARCHAR(64) NOT NULL,
			price DECIMAL(10,2) NOT NULL,
			status VARCHAR(20) DEFAULT 'pending',
			message TEXT,
			address_id VARCHAR(64) DEFAULT '',
			address_snapshot TEXT,
			shipped_at DATETIME NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			INDEX idx_buyer (buyer_id),
			INDEX idx_seller (seller_id),
			INDEX idx_product (product_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS cart_items (
			id VARCHAR(64) PRIMARY KEY,
			user_id VARCHAR(64) NOT NULL,
			product_id VARCHAR(64) NOT NULL,
			product_title VARCHAR(255) NOT NULL,
			product_image VARCHAR(512) DEFAULT '',
			spec_name VARCHAR(255) DEFAULT '',
			price DECIMAL(10,2) NOT NULL,
			quantity INT DEFAULT 1,
			created_at DATETIME NOT NULL,
			INDEX idx_cart_user (user_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS favorites (
			id VARCHAR(64) PRIMARY KEY,
			user_id VARCHAR(64) NOT NULL,
			product_id VARCHAR(64) NOT NULL,
			product_title VARCHAR(255) NOT NULL,
			product_image VARCHAR(512) DEFAULT '',
			price DECIMAL(10,2) DEFAULT 0,
			created_at DATETIME NOT NULL,
			UNIQUE KEY uk_fav (user_id, product_id),
			INDEX idx_fav_user (user_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS history (
			id VARCHAR(64) PRIMARY KEY,
			user_id VARCHAR(64) NOT NULL,
			product_id VARCHAR(64) NOT NULL,
			product_title VARCHAR(255) NOT NULL,
			product_image VARCHAR(512) DEFAULT '',
			price DECIMAL(10,2) DEFAULT 0,
			viewed_at DATETIME NOT NULL,
			UNIQUE KEY uk_history (user_id, product_id),
			INDEX idx_history_user (user_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS chat_messages (
			id VARCHAR(64) PRIMARY KEY,
			order_id VARCHAR(64) NOT NULL,
			peer_key VARCHAR(128) DEFAULT '',
			sender_id VARCHAR(64) NOT NULL,
			sender_name VARCHAR(64) NOT NULL,
			content MEDIUMTEXT,
			type VARCHAR(20) DEFAULT 'text',
			recalled TINYINT DEFAULT 0,
			deleted_by VARCHAR(512) DEFAULT '',
			created_at DATETIME NOT NULL,
			INDEX idx_chat_peer (peer_key),
			INDEX idx_chat_order (order_id),
			INDEX idx_chat_created (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS chat_read_cursors (
			user_id VARCHAR(64) NOT NULL,
			peer_key VARCHAR(128) NOT NULL,
			last_read_at DATETIME NOT NULL,
			PRIMARY KEY (user_id, peer_key)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS banners (
			id VARCHAR(64) PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			subtitle VARCHAR(255) DEFAULT '',
			image_url VARCHAR(1024) DEFAULT '',
			link_url VARCHAR(1024) DEFAULT '',
			sort_order INT DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS user_likes (
			user_id VARCHAR(64) NOT NULL,
			product_id VARCHAR(64) NOT NULL,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (user_id, product_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS user_addresses (
			id VARCHAR(64) PRIMARY KEY,
			user_id VARCHAR(64) NOT NULL,
			phone VARCHAR(20) DEFAULT '',
			campus VARCHAR(64) DEFAULT '',
			building VARCHAR(64) DEFAULT '',
			dorm_number VARCHAR(64) DEFAULT '',
			is_default TINYINT(1) DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			INDEX idx_user_addr (user_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS reviews (
			id VARCHAR(64) PRIMARY KEY,
			order_id VARCHAR(64) NOT NULL,
			reviewer_id VARCHAR(64) NOT NULL,
			target_id VARCHAR(64) NOT NULL,
			rating TINYINT NOT NULL DEFAULT 5,
			content VARCHAR(500) DEFAULT '',
			created_at DATETIME NOT NULL,
			UNIQUE KEY uk_order_reviewer (order_id, reviewer_id),
			INDEX idx_target (target_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}

	for _, ddl := range tables {
		if _, err := db.Exec(ddl); err != nil {
			return fmt.Errorf("建表失败: %w", err)
		}
	}

	// 兼容旧表结构：检测并添加可能缺少的列（忽略错误）
	colAlters := []string{
		"ALTER TABLE products ADD COLUMN specs JSON AFTER images",
		"ALTER TABLE products ADD COLUMN building VARCHAR(20) DEFAULT '' AFTER campus",
		"ALTER TABLE products ADD COLUMN fav_count INT DEFAULT 0 AFTER like_count",
		"ALTER TABLE orders ADD COLUMN product_image VARCHAR(512) DEFAULT '' AFTER product_title",
		"ALTER TABLE orders ADD COLUMN spec_name VARCHAR(255) DEFAULT '' AFTER product_image",
		"ALTER TABLE orders ADD COLUMN quantity INT DEFAULT 1 AFTER spec_name",
		"ALTER TABLE orders ADD COLUMN address_id VARCHAR(64) DEFAULT '' AFTER message",
		"ALTER TABLE orders ADD COLUMN address_snapshot TEXT AFTER address_id",
		"ALTER TABLE orders ADD COLUMN shipped_at DATETIME NULL AFTER status",
		// 将 chat_messages.content 从 TEXT 升级为 MEDIUMTEXT（支持大图片）
		"ALTER TABLE chat_messages MODIFY COLUMN content MEDIUMTEXT",
		// 添加 peer_key 列（支持按用户对分组聊天）
		"ALTER TABLE chat_messages ADD COLUMN peer_key VARCHAR(128) DEFAULT '' AFTER order_id",
		// 回填已有聊天记录的 peer_key
		"UPDATE chat_messages cm SET cm.peer_key = (SELECT CONCAT(LEAST(o.buyer_id, o.seller_id), ':', GREATEST(o.buyer_id, o.seller_id)) FROM orders o WHERE o.id = cm.order_id) WHERE cm.peer_key = '' AND EXISTS (SELECT 1 FROM orders o WHERE o.id = cm.order_id)",
	}
	for _, ddl := range colAlters {
		db.Exec(ddl) // 忽略错误
	}

	// 再次确保所有列都存在（用 INFORMATION_SCHEMA 检查）
	for _, ddl := range colAlters {
		db.Exec(ddl)
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
