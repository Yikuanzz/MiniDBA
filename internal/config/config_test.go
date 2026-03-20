package config

import (
	"testing"
)

func TestNormalizeBasePath(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"  ", ""},
		{"/dba", "/dba"},
		{"/dba/", "/dba"},
		{"dba", "/dba"},
		{"/a/b", "/a/b"},
		{"/", ""},
	}
	for _, tc := range tests {
		var c Config
		c.BasePath = tc.in
		c.ApplyDefaults()
		if c.BasePath != tc.want {
			t.Fatalf("BasePath %q: got %q, want %q", tc.in, c.BasePath, tc.want)
		}
	}
}

func TestValidateBasePath_direct(t *testing.T) {
	bad := []string{
		"/dba/",
		"/a/../b",
		"/a//b",
		"relative",
	}
	for _, p := range bad {
		if err := validateBasePath(p); err == nil {
			t.Fatalf("expected error for %q", p)
		}
	}
	if err := validateBasePath("/dba"); err != nil {
		t.Fatal(err)
	}
	if err := validateBasePath(""); err != nil {
		t.Fatal(err)
	}
}

func TestValidateBasePath_okWithDSN(t *testing.T) {
	c := Config{
		SecretKey: "sufficient-secret",
		BasePath:  "/minidba",
		Databases: []Database{{Name: "a", DSN: "u:p@tcp(127.0.0.1:3306)/db"}},
	}
	c.ApplyDefaults()
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if c.BasePath != "/minidba" {
		t.Fatalf("got %q", c.BasePath)
	}
}
