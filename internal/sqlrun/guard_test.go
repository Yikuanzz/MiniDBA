package sqlrun

import "testing"

func TestValidateTableName(t *testing.T) {
	if ValidateTableName("users") != nil {
		t.Fatal()
	}
	if ValidateTableName("bad;drop") == nil {
		t.Fatal()
	}
}

func TestCheckBlacklist(t *testing.T) {
	if CheckBlacklist("SELECT 1") != nil {
		t.Fatal()
	}
	if CheckBlacklist("DROP DATABASE x") == nil {
		t.Fatal()
	}
}

func TestCheckReadonly(t *testing.T) {
	if CheckReadonly("SELECT 1") != nil {
		t.Fatal()
	}
	if CheckReadonly("DELETE FROM t") == nil {
		t.Fatal()
	}
}

func TestIsQueryPath(t *testing.T) {
	if !IsQueryPath("SELECT 1") {
		t.Fatal()
	}
	if !IsQueryPath("SHOW TABLES") {
		t.Fatal()
	}
	if IsQueryPath("UPDATE t SET a=1") {
		t.Fatal()
	}
}

func TestFormatSQLCell(t *testing.T) {
	if got := formatSQLCell(nil); got != "NULL" {
		t.Fatal(got)
	}
	if got := formatSQLCell([]byte("hello")); got != "hello" {
		t.Fatal(got)
	}
	if got := formatSQLCell("plain"); got != "plain" {
		t.Fatal(got)
	}
	if got := formatSQLCell(int64(42)); got != "42" {
		t.Fatal(got)
	}
}
