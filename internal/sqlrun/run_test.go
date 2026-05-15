package sqlrun

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRunQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

	qres, eres, err := Run(context.Background(), db, "SELECT 1", false, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qres == nil {
		t.Fatal("expected QueryResult")
	}
	if eres != nil {
		t.Fatal("expected ExecInfo to be nil")
	}
	if len(qres.Columns) != 1 || len(qres.Rows) != 1 {
		t.Fatalf("unexpected result shape")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRunExec(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectExec("INSERT INTO t").WillReturnResult(sqlmock.NewResult(1, 1))

	qres, eres, err := Run(context.Background(), db, "INSERT INTO t VALUES (1)", false, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qres != nil {
		t.Fatal("expected QueryResult to be nil")
	}
	if eres == nil {
		t.Fatal("expected ExecResult")
	}
	if eres.RowsAffected != 1 {
		t.Fatalf("expected 1 row affected, got %d", eres.RowsAffected)
	}
	if eres.LastInsertID != 1 {
		t.Fatalf("expected last insert id 1, got %d", eres.LastInsertID)
	}
	if eres.Duration < 0 {
		t.Fatal("expected non-negative duration")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRunExecDDL(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE TABLE").WillReturnResult(sqlmock.NewResult(0, 0))

	qres, eres, err := Run(context.Background(), db, "CREATE TABLE test_ddl (id INT)", false, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qres != nil {
		t.Fatal("expected QueryResult to be nil")
	}
	if eres == nil {
		t.Fatal("expected ExecResult")
	}
	if eres.Duration < 0 {
		t.Fatal("expected non-negative duration")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRunBlacklist(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, _, err = Run(context.Background(), db, "DROP DATABASE backend", false, 100)
	if err == nil {
		t.Fatal("expected blacklist error")
	}
}

func TestRunReadonly(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, _, err = Run(context.Background(), db, "DELETE FROM t", true, 100)
	if err == nil {
		t.Fatal("expected readonly error")
	}
}
