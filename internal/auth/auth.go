package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// CookieName 会话 Cookie（与历史 demo 一致）。
	CookieName = "minidba_auth"
	// CookieConn 当前逻辑连接名。
	CookieConn   = "minidba_conn"
	cookieMaxAge = 86400 * 7
	// TokenTTL 令牌过期（墙钟）。
	TokenTTL = 24 * time.Hour
)

// SecretMatch 对用户输入与配置中的 secret_key 做摘要恒定时间比较。
func SecretMatch(input, fromConfig string) bool {
	h1 := sha256.Sum256([]byte(input))
	h2 := sha256.Sum256([]byte(fromConfig))
	return subtle.ConstantTimeCompare(h1[:], h2[:]) == 1
}

// SignToken 签发 expUnix 过期的 HMAC 令牌。
func SignToken(secret string, expUnix int64) (string, error) {
	expStr := strconv.FormatInt(expUnix, 10)
	mac := hmac.New(sha256.New, []byte(secret))
	if _, err := mac.Write([]byte(expStr)); err != nil {
		return "", err
	}
	return expStr + "." + hex.EncodeToString(mac.Sum(nil)), nil
}

// VerifyToken 校验令牌格式与签名、过期时间。
func VerifyToken(secret, token string) bool {
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

// SessionCookie 设置会话 Cookie。
func SessionCookie(value string) *http.Cookie {
	return &http.Cookie{
		Name:     CookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   cookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

// ClearSessionCookie 清除会话。
func ClearSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

// ConnCookie 当前连接。
func ConnCookie(connName string) *http.Cookie {
	return &http.Cookie{
		Name:     CookieConn,
		Value:    connName,
		Path:     "/",
		MaxAge:   cookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func ClearConnCookie() *http.Cookie {
	return &http.Cookie{
		Name:     CookieConn,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

// Authorized 若 Cookie 会话有效，或 Header 携带与配置一致的 secret_key，则返回 true。
func Authorized(r *http.Request, secret string) bool {
	if headerSecretOK(r, secret) {
		return true
	}
	c, err := r.Cookie(CookieName)
	if err != nil || c.Value == "" {
		return false
	}
	return VerifyToken(secret, c.Value)
}

func headerSecretOK(r *http.Request, secret string) bool {
	if s := strings.TrimSpace(r.Header.Get("X-MiniDBA-Secret")); s != "" {
		return SecretMatch(s, secret)
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		t := strings.TrimSpace(auth[7:])
		return SecretMatch(t, secret)
	}
	return false
}

// RequireAuth 未授权则 302 /login（API 风格请求可返回 401）。
func RequireAuth(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !Authorized(r, secret) {
			if wantsJSON(r) {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func wantsJSON(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "application/json")
}

// IssueSessionToken 生成带过期时间的会话令牌字符串。
func IssueSessionToken(secret string) (string, error) {
	return SignToken(secret, time.Now().Add(TokenTTL).Unix())
}

// RandomHex 生成 n 字节随机 hex（用于 CSRF）。
func RandomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

var errNoCookie = errors.New("no cookie")

// ReadConnName 从 Cookie 读取当前连接名；若不存在返回 errNoCookie。
func ReadConnName(r *http.Request) (string, error) {
	c, err := r.Cookie(CookieConn)
	if err != nil || c.Value == "" {
		return "", errNoCookie
	}
	return c.Value, nil
}
