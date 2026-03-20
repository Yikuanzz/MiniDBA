// Demo UI：校验 config.yaml 中的 secret_key，通过 Cookie（HMAC 短期令牌）放行静态页。
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	cookieName   = "minidba_auth"
	cookieMaxAge = 86400 * 7
	tokenTTL     = 24 * time.Hour
)

type fileConfig struct {
	SecretKey string `yaml:"secret_key"`
}

func main() {
	cfgPath := os.Getenv("MINIDBA_CONFIG")
	if cfgPath == "" {
		cfgPath = "config.yaml"
	}
	secret, err := loadSecretKey(cfgPath)
	if err != nil {
		log.Fatal(err)
	}

	addr := os.Getenv("MINIDBA_DEMO_ADDR")
	if addr == "" {
		addr = "127.0.0.1:18899"
	}

	demoRoot := "web/demo"
	staticRoot := "web/static"

	mux := http.NewServeMux()

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if ok, _ := readAuthCookie(r, secret); ok {
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
			serveLoginPage(w, r.FormValue("error") != "")
		case http.MethodPost:
			if err := r.ParseForm(); err != nil {
				http.Redirect(w, r, "/login?error=1", http.StatusSeeOther)
				return
			}
			key := strings.TrimSpace(r.FormValue("secret_key"))
			if !secretMatch(key, secret) {
				http.Redirect(w, r, "/login?error=1", http.StatusSeeOther)
				return
			}
			token, err := signToken(secret, time.Now().Add(tokenTTL).Unix())
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			http.SetCookie(w, &http.Cookie{
				Name:     cookieName,
				Value:    token,
				Path:     "/",
				MaxAge:   cookieMaxAge,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
			http.Redirect(w, r, "/", http.StatusSeeOther)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})

	fileServer := http.FileServer(http.Dir(demoRoot))
	staticHandler := http.StripPrefix("/static/", http.FileServer(http.Dir(staticRoot)))

	mux.Handle("/static/", staticHandler)
	mux.Handle("/", authMiddleware(secret, fileServer))

	log.Printf("MiniDBA UI demo → http://%s/  （secret_key 来自 %s）", addr, cfgPath)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func secretMatch(input, secret string) bool {
	hInput := sha256.Sum256([]byte(input))
	hSecret := sha256.Sum256([]byte(secret))
	return subtle.ConstantTimeCompare(hInput[:], hSecret[:]) == 1
}

func loadSecretKey(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取配置 %s: %w", path, err)
	}
	var cfg fileConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return "", fmt.Errorf("解析 YAML: %w", err)
	}
	key := strings.TrimSpace(cfg.SecretKey)
	if key == "" {
		return "", fmt.Errorf("配置 %s 缺少非空 secret_key", path)
	}
	return key, nil
}

func signToken(secret string, expUnix int64) (string, error) {
	expStr := strconv.FormatInt(expUnix, 10)
	mac := hmac.New(sha256.New, []byte(secret))
	if _, err := mac.Write([]byte(expStr)); err != nil {
		return "", err
	}
	sig := mac.Sum(nil)
	return expStr + "." + hex.EncodeToString(sig), nil
}

func verifyToken(secret, token string) bool {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return false
	}
	exp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(parts[0]))
	expected := hex.EncodeToString(mac.Sum(nil))
	if len(parts[1]) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(parts[1]), []byte(expected)) == 1
}

func readAuthCookie(r *http.Request, secret string) (bool, error) {
	c, err := r.Cookie(cookieName)
	if err != nil || c.Value == "" {
		return false, err
	}
	return verifyToken(secret, c.Value), nil
}

func authMiddleware(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/favicon.ico" {
			next.ServeHTTP(w, r)
			return
		}
		ok, _ := readAuthCookie(r, secret)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func serveLoginPage(w http.ResponseWriter, bad bool) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t := template.Must(template.New("login").Parse(loginHTML))
	_ = t.Execute(w, struct{ Bad bool }{Bad: bad})
}

const loginHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<title>MiniDBA · 访问校验（Demo）</title>
<link rel="stylesheet" href="/static/css/theme.css"/>
<style>
.login-page { min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 24px; }
.login-card { width: 100%; max-width: 400px; }
.login-card h1 { margin: 0 0 8px; font-size: 1.25rem; font-weight: 600; }
.login-card .sub { margin: 0 0 20px; color: var(--md-on-surface-secondary); font-size: 0.875rem; }
</style>
</head>
<body>
<div class="login-page">
  <div class="card login-card">
    <div class="card__body">
      <h1>进入 MiniDBA</h1>
      <p class="sub">请输入 <code>config.yaml</code> 中的 <code>secret_key</code>（与服务端一致方可放行）。</p>
      {{if .Bad}}<div class="alert alert--error" role="alert"><span>密钥不正确，请重试。</span></div>{{end}}
      <form method="post" action="/login" autocomplete="off">
        <div class="field">
          <label for="secret_key">secret_key</label>
          <input class="input" type="password" id="secret_key" name="secret_key" required placeholder="与配置文件一致"/>
        </div>
        <button type="submit" class="btn btn--primary">进入</button>
      </form>
    </div>
  </div>
</div>
</body>
</html>
`
