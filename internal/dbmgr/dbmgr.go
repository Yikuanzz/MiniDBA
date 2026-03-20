package dbmgr

import (
	"database/sql"
	"fmt"
	"log"
	"sync"

	"mini-dba/internal/config"

	_ "github.com/go-sql-driver/mysql"
)

// Manager 按逻辑名托管 *sql.DB。
type Manager struct {
	mu    sync.RWMutex
	pools map[string]*sql.DB
}

// New 为每条数据库配置打开连接池。
func New(cfg *config.Config) (*Manager, error) {
	m := &Manager{pools: make(map[string]*sql.DB)}
	for i := range cfg.Databases {
		d := cfg.Databases[i]
		db, err := sql.Open("mysql", d.DSN)
		if err != nil {
			m.closeAll()
			return nil, fmt.Errorf("打开 %q: %w", d.Name, err)
		}
		db.SetMaxOpenConns(5)
		db.SetMaxIdleConns(2)
		if err := db.Ping(); err != nil {
			log.Printf("minidba: Ping %q 失败: %v（稍后可重试）", d.Name, err)
		}
		m.pools[d.Name] = db
	}
	return m, nil
}

// Reload 关闭旧池并按新配置重建。
func (m *Manager) Reload(cfg *config.Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, db := range m.pools {
		_ = db.Close()
	}
	m.pools = make(map[string]*sql.DB)
	for i := range cfg.Databases {
		d := cfg.Databases[i]
		db, err := sql.Open("mysql", d.DSN)
		if err != nil {
			m.closeAllLocked()
			return fmt.Errorf("打开 %q: %w", d.Name, err)
		}
		db.SetMaxOpenConns(5)
		db.SetMaxIdleConns(2)
		if err := db.Ping(); err != nil {
			log.Printf("minidba: Ping %q 失败: %v", d.Name, err)
		}
		m.pools[d.Name] = db
	}
	return nil
}

// DB 返回命名连接池。
func (m *Manager) DB(name string) (*sql.DB, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	db, ok := m.pools[name]
	return db, ok
}

// Names 返回所有连接名（顺序与配置一致由调用方保证）。
func (m *Manager) Names() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, 0, len(m.pools))
	for n := range m.pools {
		out = append(out, n)
	}
	return out
}

func (m *Manager) closeAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeAllLocked()
}

func (m *Manager) closeAllLocked() {
	for _, db := range m.pools {
		_ = db.Close()
	}
	m.pools = make(map[string]*sql.DB)
}

// Close 释放所有池。
func (m *Manager) Close() {
	m.closeAll()
}
