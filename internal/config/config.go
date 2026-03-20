package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-sql-driver/mysql"
	"gopkg.in/yaml.v3"
)

// Database 单条逻辑连接。
type Database struct {
	Name string `yaml:"name"`
	DSN  string `yaml:"dsn"`
}

// Config 与 config.yaml 对应（保存时可能丢失原 YAML 注释）。
type Config struct {
	SecretKey     string     `yaml:"secret_key"`
	Listen        string     `yaml:"listen"`
	Readonly      bool       `yaml:"readonly"`
	Databases     []Database `yaml:"databases"`
	MaxResultRows int        `yaml:"max_result_rows"`
	PageSize      int        `yaml:"page_size"`
	MaxPageSize   int        `yaml:"max_page_size"`
}

// ApplyDefaults 填充缺省上限。
func (c *Config) ApplyDefaults() {
	if c.MaxResultRows <= 0 {
		c.MaxResultRows = 100
	}
	if c.PageSize <= 0 {
		c.PageSize = 50
	}
	if c.MaxPageSize <= 0 {
		c.MaxPageSize = 100
	}
	if c.PageSize > c.MaxPageSize {
		c.PageSize = c.MaxPageSize
	}
	if strings.TrimSpace(c.Listen) == "" {
		c.Listen = "127.0.0.1:8080"
	}
}

// Validate 校验配置与 DSN 形状。
func (c *Config) Validate() error {
	if strings.TrimSpace(c.SecretKey) == "" {
		return fmt.Errorf("secret_key 不能为空")
	}
	if len(c.Databases) == 0 {
		return fmt.Errorf("至少需要一条 databases 配置")
	}
	seen := make(map[string]struct{})
	for i := range c.Databases {
		d := &c.Databases[i]
		name := strings.TrimSpace(d.Name)
		d.Name = name
		d.DSN = strings.TrimSpace(d.DSN)
		if name == "" {
			return fmt.Errorf("databases[%d].name 不能为空", i)
		}
		if d.DSN == "" {
			return fmt.Errorf("databases[%d].dsn 不能为空", i)
		}
		if _, err := mysql.ParseDSN(d.DSN); err != nil {
			return fmt.Errorf("databases[%q].dsn 无效: %w", name, err)
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("重复的连接名: %q", name)
		}
		seen[name] = struct{}{}
	}
	return nil
}

// Load 读取并校验 YAML。
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置 %s: %w", path, err)
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("解析 YAML: %w", err)
	}
	c.ApplyDefaults()
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// Save 原子写入整个配置（Marshal 后注释会丢失）。
func Save(path string, c *Config) error {
	c.ApplyDefaults()
	if err := c.Validate(); err != nil {
		return err
	}
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	tmp, err := os.CreateTemp(dir, "."+base+".*.tmp")
	if err != nil {
		return fmt.Errorf("创建临时文件: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	enc, err := yaml.Marshal(c)
	if err != nil {
		_ = tmp.Close()
		return fmt.Errorf("编码 YAML: %w", err)
	}
	if _, err := tmp.Write(enc); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("写入临时文件: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("关闭临时文件: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("替换配置: %w", err)
	}
	return nil
}
