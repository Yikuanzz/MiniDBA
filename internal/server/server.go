package server

import (
	"context"
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"sync"
	"time"

	"mini-dba/internal/auth"
	"mini-dba/internal/config"
	"mini-dba/internal/dbmgr"
	"mini-dba/internal/sqlrun"
)

// New 加载配置、连接池与模板。
func New(cfgPath string, assets embed.FS) (*Server, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, err
	}
	db, err := dbmgr.New(cfg)
	if err != nil {
		return nil, err
	}
	funcs := template.FuncMap{
		"inc": func(i int) int { return i + 1 },
		"dec": func(i int) int { return i - 1 },
	}
	parse := func(files ...string) *template.Template {
		return template.Must(template.New("").Funcs(funcs).ParseFS(assets, files...))
	}
	s := &Server{
		cfgPath:   cfgPath,
		cfg:       cfg,
		secret:    cfg.SecretKey,
		db:        db,
		assets:    assets,
		homeT:     parse("web/templates/layout.html", "web/templates/partials/appbar.html", "web/templates/partials/nav.html", "web/templates/home.html"),
		tablesT:   parse("web/templates/layout.html", "web/templates/partials/appbar.html", "web/templates/partials/nav.html", "web/templates/tables.html"),
		browseT:   parse("web/templates/layout.html", "web/templates/partials/appbar.html", "web/templates/partials/nav.html", "web/templates/browse.html"),
		schemaT:   parse("web/templates/layout.html", "web/templates/partials/appbar.html", "web/templates/partials/nav.html", "web/templates/schema.html"),
		settingsT: parse("web/templates/layout.html", "web/templates/partials/appbar.html", "web/templates/partials/nav.html", "web/templates/settings.html"),
		loginT:    template.Must(template.ParseFS(assets, "web/templates/login.html")),
	}
	return s, nil
}

// Server HTTP 服务状态。
type Server struct {
	mu sync.RWMutex

	cfgPath string
	cfg     *config.Config
	secret  string
	db      *dbmgr.Manager

	assets embed.FS

	homeT, tablesT, browseT, schemaT, settingsT *template.Template
	loginT                                      *template.Template
}

func (s *Server) cfgRL() *config.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *Server) setCfg(c *config.Config) {
	s.mu.Lock()
	s.cfg = c
	s.mu.Unlock()
}

// Close 释放连接池。
func (s *Server) Close() { s.db.Close() }

// ListenAndServe 注册路由并监听。
func (s *Server) ListenAndServe() error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})

	staticFS, err := fs.Sub(s.assets, "web/static")
	if err != nil {
		return err
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	mux.HandleFunc("GET /login", s.handleLoginGet)
	mux.HandleFunc("POST /login", s.handleLoginPost)
	mux.HandleFunc("GET /logout", s.handleLogout)

	protected := func(h http.HandlerFunc) http.Handler {
		return auth.RequireAuth(s.secret, h)
	}

	mux.Handle("GET /", protected(s.handleHome))
	mux.Handle("POST /query", protected(s.handleQuery))
	mux.Handle("GET /tables", protected(s.handleTables))
	mux.Handle("GET /browse", protected(s.handleBrowse))
	mux.Handle("GET /table/{name}/schema", protected(s.handleSchema))
	mux.Handle("POST /switch-db", protected(s.handleSwitchDB))
	mux.Handle("GET /settings", protected(s.handleSettingsGet))
	mux.Handle("POST /settings", protected(s.handleSettingsPost))

	addr := s.cfgRL().Listen
	log.Printf("MiniDBA 监听 http://%s/", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) currentConnName(r *http.Request) string {
	cfg := s.cfgRL()
	name, err := auth.ReadConnName(r)
	if err == nil {
		if _, ok := s.db.DB(name); ok {
			return name
		}
	}
	if len(cfg.Databases) > 0 {
		return cfg.Databases[0].Name
	}
	return ""
}

func (s *Server) ensureConnCookie(w http.ResponseWriter, r *http.Request, name string) {
	http.SetCookie(w, auth.ConnCookie(name))
}

type pageData struct {
	Title        string
	CSRF         string
	NavActive    string
	ConnName     string
	Readonly     bool
	Databases    []config.Database
	FlashErr     string
	FlashOK      string
	MaxRows      int
	SQL          string
	QueryResult  *sqlrun.QueryResult
	ExecInfo     *sqlrun.ExecResult
	Tables       []string
	BrowseTable  string
	Page         int
	PageCount    int
	RowCount     int
	PageSize     int
	Offset       int
	MaxPageSize  int
	SchemaTable  string
	Describe     *sqlrun.QueryResult
	CreateSQL    string
	SettingsRows []settingsRow
}

type settingsRow struct {
	Name, Host, Port, User, DBName string
}

func (s *Server) basePage(r *http.Request, title, nav, conn string, tok string, flashErr, flashOK string) pageData {
	cfg := s.cfgRL()
	return pageData{
		Title:       title,
		CSRF:        tok,
		NavActive:   nav,
		ConnName:    conn,
		Readonly:    cfg.Readonly,
		Databases:   cfg.Databases,
		FlashErr:    flashErr,
		FlashOK:     flashOK,
		MaxRows:     cfg.MaxResultRows,
		MaxPageSize: cfg.MaxPageSize,
		PageSize:    cfg.PageSize,
	}
}

func (s *Server) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 60*time.Second)
}
