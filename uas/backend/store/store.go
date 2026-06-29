package store

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"uas/config"

	_ "github.com/go-sql-driver/mysql"
)

// Store UAS数据库存储
type Store struct {
	mu  sync.RWMutex
	db  *sql.DB
	cfg *config.DBConfig
}

// NewStore 创建数据库连接
func NewStore(cfg *config.DBConfig) *Store {
	s := &Store{cfg: cfg}
	if err := s.Connect(); err != nil {
		fmt.Printf("[UAS] 数据库连接失败: %v\n", err)
	} else {
		fmt.Printf("[UAS] 数据库连接成功: %s:%s/%s\n", cfg.Host, cfg.Port, cfg.DBName)
	}
	return s
}

// Connect 建立数据库连接
func (s *Store) Connect() error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&multiStatements=true",
		s.cfg.User, s.cfg.Password, s.cfg.Host, s.cfg.Port, s.cfg.DBName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return err
	}

	s.mu.Lock()
	s.db = db
	s.mu.Unlock()
	return nil
}

// GetDB 获取数据库连接
func (s *Store) GetDB() *sql.DB {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.db
}

// Ping 健康检查
func (s *Store) Ping() error {
	db := s.GetDB()
	if db == nil {
		return fmt.Errorf("database not connected")
	}
	return db.Ping()
}
