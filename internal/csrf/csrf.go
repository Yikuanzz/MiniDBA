package csrf

import (
	"crypto/subtle"
	"net/http"

	"mini-dba/internal/auth"
)

const (
	cookieName   = "minidba_csrf"
	cookieMaxAge = 86400 * 7
	formField    = "csrf_token"
)

func normCookiePath(p string) string {
	if p == "" {
		return "/"
	}
	return p
}

// Cookie 返回 CSRF double-submit cookie。path 为空时等同于 "/"。
func Cookie(token, path string) *http.Cookie {
	return &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     normCookiePath(path),
		MaxAge:   cookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func ClearCookie(path string) *http.Cookie {
	return &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     normCookiePath(path),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

// FieldName 表单字段名。
func FieldName() string { return formField }

// Ensure 若已登录但无 CSRF Cookie，则写入新令牌（鉴权通过后的 GET 中调用）。
func Ensure(w http.ResponseWriter, r *http.Request, cookiePath string) (token string, err error) {
	c, err := r.Cookie(cookieName)
	if err == nil && c.Value != "" {
		return c.Value, nil
	}
	token, err = auth.RandomHex(32)
	if err != nil {
		return "", err
	}
	http.SetCookie(w, Cookie(token, cookiePath))
	return token, nil
}

// Validate POST：表单 token 必须与 Cookie 一致。
func Validate(r *http.Request) bool {
	if err := r.ParseForm(); err != nil {
		return false
	}
	c, err := r.Cookie(cookieName)
	if err != nil || c.Value == "" {
		return false
	}
	formTok := r.FormValue(formField)
	if len(formTok) != len(c.Value) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(formTok), []byte(c.Value)) == 1
}
