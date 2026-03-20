package server

import (
	"context"
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strings"
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
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !auth.Authorized(r, s.secret) {
				if auth.WantsJSON(r) {
					http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
					return
				}
				http.Redirect(w, r, s.absPath("/login"), http.StatusSeeOther)
				return
			}
			h.ServeHTTP(w, r)
		})
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
	http.SetCookie(w, auth.ConnCookie(name, s.cookiePath()))
}

// basePath 返回规范化后的对外路径前缀，无则为 ""。
func (s *Server) basePath() string { return s.cfgRL().BasePath }

func (s *Server) cookiePath() string {
	if s.basePath() == "" {
		return "/"
	}
	return s.basePath()
}

// absPath 将进程内路径 rel（须以 / 开头）拼成浏览器应使用的绝对路径。
func (s *Server) absPath(rel string) string {
	if !strings.HasPrefix(rel, "/") {
		rel = "/" + rel
	}
	bp := s.basePath()
	if bp == "" {
		return rel
	}
	return bp + rel
}

// BrowseHeaderCol 数据浏览表头排序链接。
type BrowseHeaderCol struct {
	Name       string
	SortURL    string
	SortMarker string
}

// BrowsePSLink 每页条数切换链接。
type BrowsePSLink struct {
	Size int
	Href string
}

type pageData struct {
	Base             string
	Title            string
	CSRF             string
	NavActive        string
	ConnName         string
	Readonly         bool
	Databases        []config.Database
	FlashErr         string
	FlashOK          string
	MaxRows          int
	SQL              string
	SQLShortHash     string
	QueryResult      *sqlrun.QueryResult
	ExecInfo         *sqlrun.ExecResult
	Tables           []string
	BrowseTable      string
	Page             int
	PageCount        int
	RowCount         int
	TotalRows        int
	PageSize         int
	Offset           int
	MaxPageSize      int
	BrowseQS         string
	BrowsePrevURL    string
	BrowseNextURL    string
	BrowsePSChoices  []int
	BrowsePSLinks    []BrowsePSLink
	SortCol          string
	SortDir          string
	FilterCol        string
	FilterOp         string
	FilterVal        string
	BrowseHeaderCols []BrowseHeaderCol
	SchemaTable      string
	Describe         *sqlrun.QueryResult
	CreateSQL        string
	SettingsRows     []settingsRow
}

type settingsRow struct {
	Name, Host, Port, User, DBName string
}

func (s *Server) basePage(r *http.Request, title, nav, conn string, tok string, flashErr, flashOK string) pageData {
	cfg := s.cfgRL()
	return pageData{
		Base:        s.basePath(),
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
