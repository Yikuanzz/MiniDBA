package sqlrun

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

func TestIntegrationExec(t *testing.T) {
	dsn := os.Getenv("MINIDBA_TEST_DSN")
	if dsn == "" {
		dsn = "root:root123456@tcp(127.0.0.1:9217)/backend"
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Skipf("skip integration test: cannot open db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skipf("skip integration test: cannot ping db: %v", err)
	}

	ctx := context.Background()

	// Cleanup in case previous run left debris
	_ = execIgnore(ctx, db, "DROP TABLE IF EXISTS test_verify")

	// CREATE TABLE
	qres, eres, err := Run(ctx, db, "CREATE TABLE test_verify (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(50))", false, 100)
	if err != nil {
		t.Fatalf("CREATE TABLE failed: %v", err)
	}
	if qres != nil {
		t.Fatal("CREATE TABLE should not return QueryResult")
	}
	if eres == nil {
		t.Fatal("CREATE TABLE should return ExecResult")
	}
	if eres.Duration < 0 {
		t.Fatal("expected non-negative duration")
	}

	// INSERT
	_, eres, err = Run(ctx, db, "INSERT INTO test_verify (name) VALUES ('hello')", false, 100)
	if err != nil {
		t.Fatalf("INSERT failed: %v", err)
	}
	if eres == nil || eres.RowsAffected != 1 {
		t.Fatalf("INSERT expected 1 row affected, got %+v", eres)
	}
	if eres.LastInsertID != 1 {
		t.Fatalf("INSERT expected LastInsertID 1, got %d", eres.LastInsertID)
	}

	// UPDATE
	_, eres, err = Run(ctx, db, "UPDATE test_verify SET name = 'world' WHERE id = 1", false, 100)
	if err != nil {
		t.Fatalf("UPDATE failed: %v", err)
	}
	if eres == nil || eres.RowsAffected != 1 {
		t.Fatalf("UPDATE expected 1 row affected, got %+v", eres)
	}

	// SELECT to verify UPDATE
	qres, _, err = Run(ctx, db, "SELECT name FROM test_verify WHERE id = 1", false, 100)
	if err != nil {
		t.Fatalf("SELECT failed: %v", err)
	}
	if qres == nil || len(qres.Rows) != 1 || len(qres.Rows[0]) != 1 || qres.Rows[0][0] != "world" {
		t.Fatalf("SELECT expected name='world', got %+v", qres)
	}

	// DELETE
	_, eres, err = Run(ctx, db, "DELETE FROM test_verify WHERE id = 1", false, 100)
	if err != nil {
		t.Fatalf("DELETE failed: %v", err)
	}
	if eres == nil || eres.RowsAffected != 1 {
		t.Fatalf("DELETE expected 1 row affected, got %+v", eres)
	}

	// ALTER TABLE
	_, eres, err = Run(ctx, db, "ALTER TABLE test_verify ADD COLUMN col2 VARCHAR(20)", false, 100)
	if err != nil {
		t.Fatalf("ALTER TABLE failed: %v", err)
	}
	if eres == nil {
		t.Fatal("ALTER TABLE should return ExecResult")
	}

	// DROP TABLE
	_, eres, err = Run(ctx, db, "DROP TABLE test_verify", false, 100)
	if err != nil {
		t.Fatalf("DROP TABLE failed: %v", err)
	}
	if eres == nil {
		t.Fatal("DROP TABLE should return ExecResult")
	}
}

func execIgnore(ctx context.Context, db *sql.DB, sql string) error {
	_, err := db.ExecContext(ctx, sql)
	return err
}

func TestIntegrationReadonly(t *testing.T) {
	dsn := os.Getenv("MINIDBA_TEST_DSN")
	if dsn == "" {
		dsn = "root:root123456@tcp(127.0.0.1:9217)/backend"
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Skipf("skip integration test: cannot open db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skipf("skip integration test: cannot ping db: %v", err)
	}

	ctx := context.Background()
	_, _, err = Run(ctx, db, "DELETE FROM test_verify WHERE id = 999", true, 100)
	if err == nil {
		t.Fatal("expected readonly error")
	}
	if err.Error() != "只读模式禁止该语句" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestIntegrationLeadingCommentSelect(t *testing.T) {
	dsn := os.Getenv("MINIDBA_TEST_DSN")
	if dsn == "" {
		dsn = "root:root123456@tcp(127.0.0.1:9217)/backend"
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Skipf("skip integration test: cannot open db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skipf("skip integration test: cannot ping db: %v", err)
	}

	ctx := context.Background()
	qres, eres, err := Run(ctx, db, "/* test */ SELECT 1 AS col", false, 100)
	if err != nil {
		t.Fatalf("SELECT with leading comment failed: %v", err)
	}
	if qres == nil {
		t.Fatal("expected QueryResult for commented SELECT")
	}
	if eres != nil {
		t.Fatal("expected no ExecResult for SELECT")
	}
	if len(qres.Columns) != 1 || qres.Columns[0] != "col" {
		t.Fatalf("unexpected columns: %v", qres.Columns)
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	// Best-effort cleanup
	if dsn := os.Getenv("MINIDBA_TEST_DSN"); dsn == "" {
		dsn = "root:root123456@tcp(127.0.0.1:9217)/backend"
	}
	os.Exit(code)
}
