# MiniDBA 非查询 SQL 执行设计与验证

## 背景与问题

当前项目的 SQL 工作台（`home.html`）卡片标题为「查询语句」，路由为 `/query`，导致用户认为只能执行查询。实际上后端 `sqlrun.Run()` 已区分 `Query` 与 `Exec` 路径，但存在以下问题：

1. **前端标签误导**：页面显示「查询语句」，用户不认为可以执行 INSERT/UPDATE/CREATE 等
2. **反馈弱到像崩溃**：执行非查询 SQL 后，页面仅显示一行小字「受影响行：0」，没有明确的「执行成功」感知
3. **脚本边界风险**：`sqlresult.js` / `datatable.js` 在结果表不存在时的防御逻辑可能不足
4. **语句分类瑕疵**：`IsQueryPath()` 用前缀匹配，前导注释会导致误判
5. **零测试覆盖**：`sqlrun` 的 exec 路径没有任何自动化测试

## 目标

让非查询 SQL（DML/DDL）的执行与查询 SQL 一样稳定、可感知、可测试。

## 设计

### 1. 前端标签与结构

**修改 `web/templates/home.html`：**

- 卡片标题从「查询语句」改为「SQL 语句」
- 副标题条件显示：
  - 默认显示「单次结果集最多 {{.MaxRows}} 行」
  - 非查询执行后，结果区上方显示提示「写操作 / DDL 执行后将显示受影响行数」
- 执行结果区改造：
  - 查询结果：保持现有表格 + 筛选工具栏
  - 执行结果：升级为独立卡片，包含绿色成功徽标 + 「执行成功」文本、受影响行数、LastInsertID（如有）、执行耗时

### 2. 后端执行链路健壮化

**保持现有结构**：`sqlrun.Run()` 的分支策略不变，黑名单与只读模式继续生效。

**加固点**：

| 改动 | 说明 |
|------|------|
| 语句分类修正 | `IsQueryPath()` 去除前导注释后再判断前缀，避免 `/* hint */ SELECT 1` 被误判为 exec |
| 执行耗时 | `ExecResult` 新增 `Duration` 字段，`Run()` 入口记录 `time.Now()`，返回时填充 |
| 空结果显式处理 | DDL 的 `RowsAffected()` 常为 0，前端需明确区分「执行成功但 0 行受影响」与「失败」 |
| Handler 兜底 | `handleQuery` 保持现有 `FlashErr` 处理，`ExecInfo` 与 `QueryResult` 互斥（已由 `Run()` 保证） |

### 3. 前端脚本防御与动画

**`sqlresult.js`：**
- `go()` 在找到 `table.sql-result-table` 后，额外校验 `#sql-result-toolbar` 是否存在；不存在则直接返回
- 仅当表格和工具栏同时存在时才初始化筛选器与排序

**`datatable.js`：**
- `initTable` 入口处增加：若 `thead th` 数量为 0，或 `colgroup` 无法重建，直接放弃增强

**CSS 动画（`theme.css`）：**
- 为执行结果卡片定义 `.exec-result-card` 与 `.fade-in` 动画（`opacity: 0 -> 1`，`transform: translateY(-8px) -> 0`，`300ms ease-out`）

### 4. 测试策略

**单元测试（`sqlrun/run_test.go`）：**

引入 `sqlmock` 作为 test-only 依赖，覆盖：

| 场景 | 预期 |
|------|------|
| `SELECT 1` | `QueryResult` 非 nil，`ExecInfo` 为 nil |
| `INSERT INTO t VALUES (1)` | `ExecResult` 非 nil，`QueryResult` 为 nil |
| `CREATE TABLE t (id INT)` | `ExecResult` 非 nil，`RowsAffected` 为 0 |
| `/* comment */ SELECT 1` | 识别为查询，走 `runQueryArgs` |
| `DROP DATABASE x` | 黑名单拦截，返回错误 |
| 只读模式 `DELETE FROM t` | `CheckReadonly` 拦截 |

**集成验证（真实 MySQL）：**

使用数据库：`root:root123456@tcp(127.0.0.1:9217)/backend`

验证语句清单：
```sql
-- DML
INSERT INTO test_table (name) VALUES ('hello');
UPDATE test_table SET name = 'world' WHERE id = 1;
DELETE FROM test_table WHERE id = 1;

-- DDL
CREATE TABLE test_ddl (id INT PRIMARY KEY);
ALTER TABLE test_ddl ADD COLUMN col2 VARCHAR(20);
DROP TABLE test_ddl;
```

每条执行后检查：
- [ ] 页面正常返回，无白屏/无响应
- [ ] 显示「执行成功」动画卡片
- [ ] 受影响行数正确
- [ ] 只读模式下被拦截并显示红色错误提示

## 涉及文件

- `internal/sqlrun/run.go` — 增加 `Duration`，修正 `IsQueryPath`
- `internal/sqlrun/run_test.go` — 新增 exec 路径测试
- `internal/server/handlers.go` — 无逻辑改动（已有 `ExecInfo` 处理），如需传递耗时则微调
- `web/templates/home.html` — 标签、条件副标题、执行结果卡片
- `web/static/css/theme.css` — 执行结果卡片样式与动画
- `web/static/js/sqlresult.js` — 防御式初始化
- `web/static/js/datatable.js` — 空表保护
- `go.mod` / `go.sum` — 引入 `sqlmock`

## 非目标

- 多语句批量执行（分号分隔）
- 事务控制（BEGIN/COMMIT/ROLLBACK UI）
- 存储过程 `CALL` 的多结果集支持
- SQL 语法高亮或格式化
