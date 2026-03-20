package sqlrun

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// QueryResult 查询结果。
type QueryResult struct {
	Columns   []string
	Rows      [][]string
	Truncated bool
}

// ExecResult 执行结果。
type ExecResult struct {
	RowsAffected int64
	LastInsertID int64
}

// Run 根据只读与语句类型执行；maxRows 为查询最大行数。
func Run(ctx context.Context, db *sql.DB, sqlText string, readonly bool, maxRows int) (*QueryResult, *ExecResult, error) {
	if err := CheckBlacklist(sqlText); err != nil {
		return nil, nil, err
	}
	if readonly {
		if err := CheckReadonly(sqlText); err != nil {
			return nil, nil, err
		}
	}
	if IsQueryPath(sqlText) {
		res, err := runQueryArgs(ctx, db, sqlText, nil, maxRows)
		return res, nil, err
	}
	if readonly {
		return nil, nil, fmt.Errorf("只读模式禁止执行该语句")
	}
	ex, err := runExec(ctx, db, sqlText)
	return nil, ex, err
}

// RunReadonlyQuery 执行只读参数化查询（占位符 ?）；maxRows 为最多返回行数。
func RunReadonlyQuery(ctx context.Context, db *sql.DB, sqlText string, args []interface{}, maxRows int) (*QueryResult, error) {
	if err := CheckBlacklist(sqlText); err != nil {
		return nil, err
	}
	if err := CheckReadonly(sqlText); err != nil {
		return nil, err
	}
	if !IsQueryPath(sqlText) {
		return nil, fmt.Errorf("仅支持查询语句")
	}
	return runQueryArgs(ctx, db, sqlText, args, maxRows)
}

func runQueryArgs(ctx context.Context, db *sql.DB, sqlText string, args []interface{}, maxRows int) (*QueryResult, error) {
	rows, err := db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	out := &QueryResult{Columns: cols}
	n := 0
	truncated := false
	for rows.Next() {
		if n >= maxRows {
			truncated = true
			break
		}
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make([]string, len(cols))
		for i, v := range vals {
			row[i] = formatSQLCell(v)
		}
		out.Rows = append(out.Rows, row)
		n++
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out.Truncated = truncated
	return out, nil
}

// formatSQLCell 将 Scan 得到的值转为展示用字符串（MySQL 驱动常把文本列扫成 []byte，不能用 %v 直接打印）。
func formatSQLCell(v interface{}) string {
	if v == nil {
		return "NULL"
	}
	switch x := v.(type) {
	case []byte:
		return string(x)
	case string:
		return x
	case time.Time:
		return x.Format("2006-01-02 15:04:05.000000")
	default:
		return fmt.Sprint(x)
	}
}

func runExec(ctx context.Context, db *sql.DB, sqlText string) (*ExecResult, error) {
	res, err := db.ExecContext(ctx, sqlText)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	id, _ := res.LastInsertId()
	return &ExecResult{RowsAffected: n, LastInsertID: id}, nil
}
