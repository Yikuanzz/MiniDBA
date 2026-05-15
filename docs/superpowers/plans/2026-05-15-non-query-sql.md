# 非查询 SQL 执行实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复并增强 MiniDBA 的非查询 SQL（DML/DDL）执行能力，使其稳定、可感知、可测试。

**Architecture:** 保持后端 `sqlrun.Run()` 的 Query/Exec 分支结构不变，加固语句分类边界、补充执行耗时、增强前端无结果表时的脚本防御，并为执行结果添加显眼的成功卡片与动画。

**Tech Stack:** Go 1.24, `database/sql`, `go-sql-driver/mysql`, `html/template`, `github.com/DATA-DOG/go-sqlmock`

---

## 文件结构

| 文件 | 动作 | 说明 |
|------|------|------|
| `internal/sqlrun/guard.go` | 修改 | `IsQueryPath()` 增加去除前导注释逻辑 |
| `internal/sqlrun/guard_test.go` | 修改 | 补充前导注释场景测试 |
| `internal/sqlrun/run.go` | 修改 | `ExecResult` 增加 `Duration`，`Run()` 增加耗时记录 |
| `internal/sqlrun/run_test.go` | 新建 | 使用 sqlmock 覆盖 query/exec/黑名单/只读场景 |
| `internal/server/server.go` | 修改 | 模板函数增加 `dur` 用于格式化耗时 |
| `web/templates/home.html` | 修改 | 标签改为「SQL 语句」，增加执行结果卡片 |
| `web/static/css/theme.css` | 修改 | 增加 `.exec-result-card`、`.fade-in` 动画 |
| `web/static/js/sqlresult.js` | 修改 | `go()` 增加工具栏存在性校验 |
| `web/static/js/datatable.js` | 修改 | `initTable` 增加空表头保护 |
| `go.mod` / `go.sum` | 修改 | 引入 `sqlmock` 测试依赖 |

---

### Task 1: 加固 `IsQueryPath` — 处理前导注释

**Files:**
- Modify: `internal/sqlrun/guard.go`
- Test: `internal/sqlrun/guard_test.go`

- [ ] **Step 1: 编写 `stripLeadingComments` 辅助函数**

在 `internal/sqlrun/guard.go` 中，`CheckBlacklist` 函数之前新增：

```go
// stripLeadingComments 去除 SQL 前导的 /* ... */ 与 -- 注释及空白。
func stripLeadingComments(sql string) string {
	s := strings.TrimSpace(sql)
	for {
		if strings.HasPrefix(s, "/*") {
			idx := strings.Index(s, "*/")
			if idx < 0 {
				break
			}
			s = strings.TrimSpace(s[idx+2:])
			continue
		}
		if strings.HasPrefix(s, "--") {
			idx := strings.Index(s, "\n")
			if idx < 0 {
				break
			}
			s = strings.TrimSpace(s[idx+1:])
			continue
		}
		break
	}
	return s
}
```

- [ ] **Step 2: 修改 `IsQueryPath` 使用新函数**

将 `IsQueryPath` 改为：

```go
func IsQueryPath(sql string) bool {
	s := stripLeadingComments(sql)
	if s == "" {
		return false
	}
	u := strings.ToUpper(s)
	return strings.HasPrefix(u, "SELECT") ||
		strings.HasPrefix(u, "WITH") ||
		strings.HasPrefix(u, "SHOW") ||
		strings.HasPrefix(u, "DESCRIBE") ||
		strings.HasPrefix(u, "DESC ") ||
		strings.HasPrefix(u, "EXPLAIN")
}
```

- [ ] **Step 3: 在 `guard_test.go` 中添加测试**

```go
func TestIsQueryPathWithComments(t *testing.T) {
	if !IsQueryPath("/* hint */ SELECT 1") {
		t.Fatal("want query for leading block comment")
	}
	if !IsQueryPath("-- comment\nSELECT 1") {
		t.Fatal("want query for leading line comment")
	}
	if IsQueryPath("/* hint */ INSERT INTO t VALUES (1)") {
		t.Fatal("want exec for commented insert")
	}
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test ./internal/sqlrun/ -run TestIsQueryPathWithComments -v
```

Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/sqlrun/guard.go internal/sqlrun/guard_test.go
git commit -m "fix(sqlrun): IsQueryPath now strips leading comments before checking prefix

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 2: `ExecResult` 增加执行耗时

**Files:**
- Modify: `internal/sqlrun/run.go`

- [ ] **Step 1: 修改 `ExecResult` 结构体**

在 `internal/sqlrun/run.go` 中，将：

```go
type ExecResult struct {
	RowsAffected int64
	LastInsertID int64
}
```

改为：

```go
type ExecResult struct {
	RowsAffected int64
	LastInsertID int64
	Duration     time.Duration
}
```

- [ ] **Step 2: 修改 `Run()` 记录耗时**

将 `Run()` 函数体改为：

```go
func Run(ctx context.Context, db *sql.DB, sqlText string, readonly bool, maxRows int) (*QueryResult, *ExecResult, error) {
	if err := CheckBlacklist(sqlText); err != nil {
		return nil, nil, err
	}
	if readonly {
		if err := CheckReadonly(sqlText); err != nil {
			return nil, nil, err
		}
	}
	start := time.Now()
	if IsQueryPath(sqlText) {
		res, err := runQueryArgs(ctx, db, sqlText, nil, maxRows)
		return res, nil, err
	}
	if readonly {
		return nil, nil, fmt.Errorf("只读模式禁止执行该语句")
	}
	ex, err := runExec(ctx, db, sqlText)
	if ex != nil {
		ex.Duration = time.Since(start)
	}
	return nil, ex, err
}
```

- [ ] **Step 3: 编译确认无错**

```bash
go build ./...
```

Expected: 无报错

- [ ] **Step 4: 提交**

```bash
git add internal/sqlrun/run.go
git commit -m "feat(sqlrun): add Duration to ExecResult

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 3: 为 `sqlrun` 引入 sqlmock 单元测试

**Files:**
- Create: `internal/sqlrun/run_test.go`
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: 安装 sqlmock**

```bash
go get github.com/DATA-DOG/go-sqlmock
```

- [ ] **Step 2: 编写 `run_test.go`**

```go
package sqlrun

import (
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

	qres, eres, err := Run(nil, db, "SELECT 1", false, 100)
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

	qres, eres, err := Run(nil, db, "INSERT INTO t VALUES (1)", false, 100)
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
	if eres.Duration <= 0 {
		t.Fatal("expected positive duration")
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

	qres, eres, err := Run(nil, db, "CREATE TABLE test_ddl (id INT)", false, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qres != nil {
		t.Fatal("expected QueryResult to be nil")
	}
	if eres == nil {
		t.Fatal("expected ExecResult")
	}
	if eres.Duration <= 0 {
		t.Fatal("expected positive duration")
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

	_, _, err = Run(nil, db, "DROP DATABASE backend", false, 100)
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

	_, _, err = Run(nil, db, "DELETE FROM t", true, 100)
	if err == nil {
		t.Fatal("expected readonly error")
	}
}
```

- [ ] **Step 3: 运行测试**

```bash
go test ./internal/sqlrun/ -v
```

Expected: 全部 PASS（5 个测试）

- [ ] **Step 4: 提交**

```bash
git add go.mod go.sum internal/sqlrun/run_test.go
git commit -m "test(sqlrun): add sqlmock tests for query, exec, DDL, blacklist, readonly

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 4: 模板函数增加耗时格式化

**Files:**
- Modify: `internal/server/server.go`

- [ ] **Step 1: 在模板函数中注册 `dur`**

在 `internal/server/server.go` 的 `New` 函数中，将 `funcs` 改为：

```go
funcs := template.FuncMap{
	"inc": func(i int) int { return i + 1 },
	"dec": func(i int) int { return i - 1 },
	"dur": func(d time.Duration) string {
		if d < time.Millisecond {
			return d.Round(time.Microsecond).String()
		}
		return d.Round(time.Millisecond).String()
	},
}
```

- [ ] **Step 2: 编译确认无错**

```bash
go build ./...
```

Expected: 无报错

- [ ] **Step 3: 提交**

```bash
git add internal/server/server.go
git commit -m "feat(server): add dur template func for formatting execution duration

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 5: 更新首页模板

**Files:**
- Modify: `web/templates/home.html`

- [ ] **Step 1: 修改卡片标题与副标题**

将原代码：

```html
<h1 class="page-title">SQL 工作台</h1>
<p class="page-sub">单次结果集最多 {{.MaxRows}} 行</p>

<div class="card">
  <div class="card__header">查询语句</div>
```

替换为：

```html
<h1 class="page-title">SQL 工作台</h1>
<p class="page-sub">单次结果集最多 {{.MaxRows}} 行 · 写操作与 DDL 也将在此执行</p>

<div class="card">
  <div class="card__header">SQL 语句</div>
```

- [ ] **Step 2: 替换 `ExecInfo` 展示区块**

将原代码：

```html
{{if .ExecInfo}}
<div class="toolbar-meta">
  <span>受影响行：<strong>{{.ExecInfo.RowsAffected}}</strong></span>
  {{if ne .ExecInfo.LastInsertID 0}}<span>LastInsertId：<strong>{{.ExecInfo.LastInsertID}}</strong></span>{{end}}
</div>
{{end}}
```

替换为：

```html
{{if .ExecInfo}}
<div class="exec-result-card fade-in">
  <div class="exec-result-card__header">
    <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
    </svg>
    执行成功
  </div>
  <div class="exec-result-card__body">
    <span>受影响行：<strong>{{.ExecInfo.RowsAffected}}</strong></span>
    {{if ne .ExecInfo.LastInsertID 0}}
    <span style="margin-left: 16px;">LastInsertId：<strong>{{.ExecInfo.LastInsertID}}</strong></span>
    {{end}}
    {{if .ExecInfo.Duration}}
    <span style="margin-left: 16px;">耗时：<strong>{{dur .ExecInfo.Duration}}</strong></span>
    {{end}}
  </div>
</div>
{{end}}
```

- [ ] **Step 3: 提交**

```bash
git add web/templates/home.html
git commit -m "feat(ui): rename query card to SQL statement, add exec result card with animation

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 6: 添加执行结果卡片 CSS

**Files:**
- Modify: `web/static/css/theme.css`

- [ ] **Step 1: 在 `theme.css` 末尾追加样式**

在文件末尾（`@media (max-width: 768px)` 之后）追加：

```css
/* Exec result card */
.exec-result-card {
  background: var(--md-surface);
  border-radius: var(--md-radius-md);
  box-shadow: var(--md-elevation-1);
  margin-bottom: 16px;
  overflow: hidden;
  border-left: 4px solid var(--md-success);
}
.exec-result-card__header {
  padding: 12px 16px;
  font-weight: 600;
  color: var(--md-success);
  display: flex;
  align-items: center;
  gap: 8px;
}
.exec-result-card__body {
  padding: 0 16px 12px;
  font-size: 0.875rem;
  color: var(--md-on-surface-secondary);
}

@keyframes fadeInDown {
  from {
    opacity: 0;
    transform: translateY(-8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
.fade-in {
  animation: fadeInDown 300ms ease-out;
}

@media (prefers-reduced-motion: reduce) {
  .fade-in {
    animation: none;
  }
}
```

- [ ] **Step 2: 提交**

```bash
git add web/static/css/theme.css
git commit -m "feat(css): add exec-result-card and fade-in animation styles

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 7: 加固 `sqlresult.js`

**Files:**
- Modify: `web/static/js/sqlresult.js`

- [ ] **Step 1: 修改 `go()` 增加工具栏校验**

将 `go()` 函数：

```javascript
function go() {
    var t = document.querySelector("table.sql-result-table");
    if (t) init(t);
}
```

改为：

```javascript
function go() {
    var t = document.querySelector("table.sql-result-table");
    var tb = document.getElementById("sql-result-toolbar");
    if (t && tb) init(t);
}
```

- [ ] **Step 2: 提交**

```bash
git add web/static/js/sqlresult.js
git commit -m "fix(js): sqlresult only initializes when both table and toolbar exist

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 8: 加固 `datatable.js`

**Files:**
- Modify: `web/static/js/datatable.js`

- [ ] **Step 1: 修改 `initTable` 增加空表头保护**

在 `initTable` 函数中，在现有 `if (!ths.length) return;` 之后、在 `colgroup` 处理之前，增加：

```javascript
if (!ths.length) return;
```

（实际上代码已有这个检查，但需要确认它确实在 `colgroup` 重建之前。当前代码顺序是：获取 key → 获取 ths → 检查 !ths.length → 检查 data-dt-enhanced → 处理 colgroup。这个顺序是正确的。）

经核查，`datatable.js` 第 19-21 行已有：
```javascript
var ths = table.querySelectorAll("thead th");
if (!ths.length) return;
```

**无需修改**，该文件已具备空表头保护。

- [ ] **Step 2: 提交（如无需改动则跳过）**

若确认无需修改，直接跳过此任务。

---

### Task 9: 集成验证（真实 MySQL）

**Files:**
- 无文件修改，纯验证

- [ ] **Step 1: 确保本地 MySQL 可连接**

DSN: `root:root123456@tcp(127.0.0.1:9217)/backend`

在 `config.yaml` 中配置一个测试连接，或临时启动一个测试配置。

- [ ] **Step 2: 启动服务**

```bash
go run . -config config.yaml
```

- [ ] **Step 3: 浏览器验证 DML**

在 SQL 工作台依次执行：

```sql
CREATE TABLE IF NOT EXISTS test_verify (
  id INT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(50)
);
INSERT INTO test_verify (name) VALUES ('hello');
UPDATE test_verify SET name = 'world' WHERE id = 1;
DELETE FROM test_verify WHERE id = 1;
DROP TABLE test_verify;
```

每条执行后确认：
- [ ] 页面正常返回，无白屏/无响应
- [ ] 绿色「执行成功」卡片以动画形式出现
- [ ] 受影响行数显示正确
- [ ] 耗时显示合理

- [ ] **Step 4: 验证只读拦截**

临时将 `config.yaml` 的 `readonly` 改为 `true`，重启后执行：

```sql
DELETE FROM test_verify WHERE id = 1;
```

确认：
- [ ] 页面显示红色错误提示「只读模式禁止该语句」

- [ ] **Step 5: 验证前导注释 SELECT 不走 exec**

```sql
/* test */ SELECT 1 AS col;
```

确认：
- [ ] 正常返回结果表（说明被正确识别为查询，未走 ExecContext）

---

## Self-Review Checklist

### 1. Spec 覆盖

| 设计小节 | 对应任务 |
|----------|----------|
| 语句分类边界修正（前导注释） | Task 1 |
| 执行耗时 | Task 2, 4 |
| 空结果/零影响行显式处理 | Task 5（卡片始终展示，不受 0 值影响） |
| 前端标签「SQL 语句」 | Task 5 |
| 执行结果卡片 + 动画 | Task 5, 6 |
| 脚本防御 | Task 7, 8 |
| sqlmock 单元测试 | Task 3 |
| 集成验证 | Task 9 |

### 2. Placeholder 扫描

- 无 TBD/TODO
- 无 "add appropriate error handling"
- 无 "write tests for the above"（Task 3 含完整测试代码）
- 所有步骤含实际代码或命令

### 3. 类型一致性

- `ExecResult.Duration` 为 `time.Duration`，模板使用 `dur` 函数格式化，一致。
- `Run()` 返回签名未变，Handler 层无需改动。

---

**Plan complete and saved to `docs/superpowers/plans/2026-05-15-non-query-sql.md`.**
