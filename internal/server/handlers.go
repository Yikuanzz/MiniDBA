package server

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"mini-dba/internal/auth"
	"mini-dba/internal/config"
	"mini-dba/internal/csrf"
	"mini-dba/internal/sqlrun"

	"github.com/go-sql-driver/mysql"
)

func (s *Server) syncConnCookie(w http.ResponseWriter, r *http.Request) {
	conn := s.currentConnName(r)
	if _, err := auth.ReadConnName(r); err != nil {
		s.ensureConnCookie(w, r, conn)
	}
}

func (s *Server) handleLoginGet(w http.ResponseWriter, r *http.Request) {
	if auth.Authorized(r, s.secret) {
		http.Redirect(w, r, s.absPath("/"), http.StatusSeeOther)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.loginT.Execute(w, struct {
		Bad  bool
		Base string
	}{Bad: r.URL.Query().Get("error") != "", Base: s.basePath()})
}

func (s *Server) handleLoginPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, s.absPath("/login?error=1"), http.StatusSeeOther)
		return
	}
	key := strings.TrimSpace(r.FormValue("secret_key"))
	if !auth.SecretMatch(key, s.secret) {
		http.Redirect(w, r, s.absPath("/login?error=1"), http.StatusSeeOther)
		return
	}
	sess, err := auth.IssueSessionToken(s.secret)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	cp := s.cookiePath()
	http.SetCookie(w, auth.SessionCookie(sess, cp))
	tok, err := auth.RandomHex(32)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, csrf.Cookie(tok, cp))
	http.Redirect(w, r, s.absPath("/"), http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	cp := s.cookiePath()
	http.SetCookie(w, auth.ClearSessionCookie(cp))
	http.SetCookie(w, csrf.ClearCookie(cp))
	http.Redirect(w, r, s.absPath("/login"), http.StatusSeeOther)
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	tok, err := csrf.Ensure(w, r, s.cookiePath())
	if err != nil {
		http.Error(w, "csrf", http.StatusInternalServerError)
		return
	}
	conn := s.currentConnName(r)
	s.syncConnCookie(w, r)
	p := s.basePage(r, "SQL 工作台", "home", conn, tok, "", "")
	p.SQL = strings.TrimSpace(r.URL.Query().Get("sql"))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.homeT.ExecuteTemplate(w, "layout", p); err != nil {
		log.Println(err)
	}
}

func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	if !csrf.Validate(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	tok, err := csrf.Ensure(w, r, s.cookiePath())
	if err != nil {
		http.Error(w, "csrf", http.StatusInternalServerError)
		return
	}
	sqlText := strings.TrimSpace(r.FormValue("sql"))
	conn := s.currentConnName(r)
	db, ok := s.db.DB(conn)
	cfg := s.cfgRL()
	p := s.basePage(r, "SQL 工作台", "home", conn, tok, "", "")
	p.SQL = sqlText
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if !ok {
		p.FlashErr = "未找到当前连接"
		_ = s.homeT.ExecuteTemplate(w, "layout", p)
		return
	}
	if sqlText == "" {
		p.FlashErr = "SQL 为空"
		_ = s.homeT.ExecuteTemplate(w, "layout", p)
		return
	}
	ctx, cancel := s.ctx()
	defer cancel()
	qres, eres, err := sqlrun.Run(ctx, db, sqlText, cfg.Readonly, cfg.MaxResultRows)
	if err != nil {
		p.FlashErr = err.Error()
	} else {
		p.QueryResult = qres
		p.ExecInfo = eres
	}
	if err := s.homeT.ExecuteTemplate(w, "layout", p); err != nil {
		log.Println(err)
	}
}

func (s *Server) handleTables(w http.ResponseWriter, r *http.Request) {
	tok, err := csrf.Ensure(w, r, s.cookiePath())
	if err != nil {
		http.Error(w, "csrf", http.StatusInternalServerError)
		return
	}
	conn := s.currentConnName(r)
	s.syncConnCookie(w, r)
	db, ok := s.db.DB(conn)
	p := s.basePage(r, "表列表", "tables", conn, tok, "", "")
	if !ok {
		p.FlashErr = "未找到当前连接"
	} else {
		ctx, cancel := s.ctx()
		defer cancel()
		rows, err := db.QueryContext(ctx, "SHOW TABLES")
		if err != nil {
			p.FlashErr = err.Error()
		} else {
			defer rows.Close()
			for rows.Next() {
				var name string
				if err := rows.Scan(&name); err != nil {
					p.FlashErr = err.Error()
					break
				}
				p.Tables = append(p.Tables, name)
			}
			_ = rows.Err()
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.tablesT.ExecuteTemplate(w, "layout", p)
}

func (s *Server) handleBrowse(w http.ResponseWriter, r *http.Request) {
	tok, err := csrf.Ensure(w, r, s.cookiePath())
	if err != nil {
		http.Error(w, "csrf", http.StatusInternalServerError)
		return
	}
	conn := s.currentConnName(r)
	s.syncConnCookie(w, r)
	table := strings.TrimSpace(getQuery(r, "table"))
	page := 1
	if p := getQuery(r, "page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}
	cfg := s.cfgRL()
	p := s.basePage(r, "数据浏览", "browse", conn, tok, "", "")
	p.BrowseTable = table
	p.Page = page
	p.PageSize = cfg.PageSize
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if table == "" {
		_ = s.browseT.ExecuteTemplate(w, "layout", p)
		return
	}
	if err := sqlrun.ValidateTableName(table); err != nil {
		p.FlashErr = err.Error()
		_ = s.browseT.ExecuteTemplate(w, "layout", p)
		return
	}
	db, ok := s.db.DB(conn)
	if !ok {
		p.FlashErr = "未找到当前连接"
		_ = s.browseT.ExecuteTemplate(w, "layout", p)
		return
	}
	ctx, cancel := s.ctx()
	defer cancel()

	var total int
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM `%s`", table)
	if err := db.QueryRowContext(ctx, countSQL).Scan(&total); err != nil {
		p.FlashErr = err.Error()
		_ = s.browseT.ExecuteTemplate(w, "layout", p)
		return
	}
	pageCount := (total + cfg.PageSize - 1) / cfg.PageSize
	if pageCount == 0 {
		pageCount = 1
	}
	if page > pageCount {
		page = pageCount
		p.Page = page
	}
	offset := (page - 1) * cfg.PageSize
	p.Offset = offset
	p.PageCount = pageCount

	q := fmt.Sprintf("SELECT * FROM `%s` LIMIT %d OFFSET %d", table, cfg.PageSize, offset)
	qres, _, err := sqlrun.Run(ctx, db, q, true, cfg.MaxPageSize)
	if err != nil {
		p.FlashErr = err.Error()
	} else {
		p.QueryResult = qres
		p.RowCount = len(qres.Rows)
	}
	_ = s.browseT.ExecuteTemplate(w, "layout", p)
}

func getQuery(r *http.Request, k string) string {
	return r.URL.Query().Get(k)
}

func (s *Server) handleSchema(w http.ResponseWriter, r *http.Request) {
	tok, err := csrf.Ensure(w, r, s.cookiePath())
	if err != nil {
		http.Error(w, "csrf", http.StatusInternalServerError)
		return
	}
	name := r.PathValue("name")
	conn := s.currentConnName(r)
	s.syncConnCookie(w, r)
	p := s.basePage(r, fmt.Sprintf("表结构 · %s", name), "tables", conn, tok, "", "")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := sqlrun.ValidateTableName(name); err != nil {
		p.FlashErr = err.Error()
		_ = s.tablesT.ExecuteTemplate(w, "layout", p)
		return
	}
	p.SchemaTable = name
	db, ok := s.db.DB(conn)
	if !ok {
		p.FlashErr = "未找到当前连接"
		_ = s.schemaT.ExecuteTemplate(w, "layout", p)
		return
	}
	ctx, cancel := s.ctx()
	defer cancel()
	descQ := fmt.Sprintf("DESCRIBE `%s`", name)
	dres, _, err := sqlrun.Run(ctx, db, descQ, true, 500)
	if err != nil {
		p.FlashErr = err.Error()
		_ = s.schemaT.ExecuteTemplate(w, "layout", p)
		return
	}
	p.Describe = dres

	var create string
	showQ := fmt.Sprintf("SHOW CREATE TABLE `%s`", name)
	row := db.QueryRowContext(ctx, showQ)
	var tname string
	if err := row.Scan(&tname, &create); err != nil {
		p.FlashErr = err.Error()
		_ = s.schemaT.ExecuteTemplate(w, "layout", p)
		return
	}
	p.CreateSQL = create
	_ = s.schemaT.ExecuteTemplate(w, "layout", p)
}

func (s *Server) handleSwitchDB(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil || !csrf.Validate(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	name := strings.TrimSpace(r.FormValue("conn"))
	cfg := s.cfgRL()
	found := false
	for _, d := range cfg.Databases {
		if d.Name == name {
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "unknown conn", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, auth.ConnCookie(name, s.cookiePath()))
	http.Redirect(w, r, s.safeRefererRedirect(r), http.StatusSeeOther)
}

func (s *Server) safeRefererRedirect(r *http.Request) string {
	fallback := s.absPath("/")
	ref := strings.TrimSpace(r.Header.Get("Referer"))
	if ref == "" {
		return fallback
	}
	u, err := url.Parse(ref)
	if err != nil || u.Path == "" {
		return fallback
	}
	reqHost := r.Host
	if u.Host != "" && !strings.EqualFold(u.Host, reqHost) {
		return fallback
	}
	bp := s.cfgRL().BasePath
	pq := u.Path
	if u.RawQuery != "" {
		pq += "?" + u.RawQuery
	}
	if bp == "" {
		if strings.HasPrefix(u.Path, "/") {
			return pq
		}
		return fallback
	}
	if u.Path == bp || strings.HasPrefix(u.Path, bp+"/") {
		return pq
	}
	return fallback
}

func parseDSNRow(dsn string) (settingsRow, error) {
	var out settingsRow
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return out, err
	}
	out.User = cfg.User
	out.DBName = cfg.DBName
	host, port, err := splitAddr(cfg.Addr)
	if err != nil {
		return out, err
	}
	out.Host = host
	out.Port = port
	return out, nil
}

func splitAddr(addr string) (host, port string, err error) {
	if addr == "" {
		return "", "", fmt.Errorf("empty addr")
	}
	if strings.HasPrefix(addr, "/") {
		return addr, "0", nil
	}
	i := strings.LastIndex(addr, ":")
	if i < 0 {
		return addr, "3306", nil
	}
	return addr[:i], addr[i+1:], nil
}

func (s *Server) handleSettingsGet(w http.ResponseWriter, r *http.Request) {
	tok, err := csrf.Ensure(w, r, s.cookiePath())
	if err != nil {
		http.Error(w, "csrf", http.StatusInternalServerError)
		return
	}
	conn := s.currentConnName(r)
	s.syncConnCookie(w, r)
	cfg := s.cfgRL()
	p := s.basePage(r, "连接设置", "settings", conn, tok, "", "")
	if r.URL.Query().Get("msg") == "saved" {
		p.FlashOK = "已保存并重载连接"
	}
	for _, d := range cfg.Databases {
		row := settingsRow{Name: d.Name}
		if rr, err := parseDSNRow(d.DSN); err == nil {
			row.Host = rr.Host
			row.Port = rr.Port
			row.User = rr.User
			row.DBName = rr.DBName
		}
		p.SettingsRows = append(p.SettingsRows, row)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.settingsT.ExecuteTemplate(w, "layout", p)
}

func (s *Server) handleSettingsPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil || !csrf.Validate(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	diskCfg, err := config.Load(s.cfgPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	names := r.Form["name"]
	hosts := r.Form["host"]
	ports := r.Form["port"]
	users := r.Form["user"]
	dbnames := r.Form["dbname"]
	passes := r.Form["newpass"]
	if len(names) == 0 || len(names) != len(hosts) || len(names) != len(ports) || len(names) != len(users) || len(names) != len(dbnames) {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	for len(passes) < len(names) {
		passes = append(passes, "")
	}
	var dbs []config.Database
	for i := range names {
		mc := mysql.NewConfig()
		mc.Net = "tcp"
		mc.Addr = fmt.Sprintf("%s:%s", hosts[i], ports[i])
		mc.User = users[i]
		mc.DBName = dbnames[i]
		newPass := ""
		if i < len(passes) {
			newPass = passes[i]
		}
		var prevPass string
		for _, od := range diskCfg.Databases {
			if od.Name == strings.TrimSpace(names[i]) {
				if pc, err := mysql.ParseDSN(od.DSN); err == nil {
					prevPass = pc.Passwd
				}
				break
			}
		}
		if newPass != "" {
			mc.Passwd = newPass
		} else {
			mc.Passwd = prevPass
		}
		if mc.Passwd == "" {
			http.Error(w, fmt.Sprintf("连接 %q 需要密码（新连接或无法继承）", names[i]), http.StatusBadRequest)
			return
		}
		dbs = append(dbs, config.Database{Name: strings.TrimSpace(names[i]), DSN: mc.FormatDSN()})
	}
	newCfg := *diskCfg
	newCfg.Databases = dbs
	newCfg.Readonly = r.FormValue("readonly") == "1"
	if err := config.Save(s.cfgPath, &newCfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	loaded, err := config.Load(s.cfgPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.setCfg(loaded)
	if err := s.db.Reload(loaded); err != nil {
		log.Printf("reload db: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, s.absPath("/settings?msg=saved"), http.StatusSeeOther)
}
