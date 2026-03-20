# MiniDBA 项目设计文档

## 1. 项目定位与目标

**一句话目标**：做一个类似「精简版 Adminer」的内网数据库轻量运维工具。

| 约束 | 说明 |
|------|------|
| 语言 / 形态 | Go，**单二进制**，依赖极少 |
| 数据库 | MVP 仅 MySQL（单连接 / 配置内多库名） |
| 界面 | Web，**服务端模板渲染**；静态资源 **随包分发或内嵌**，支持完全离线 |
| 场景 | 前置机 / 内网穿透 / 最小攻击面下的临时查数、跑 SQL |
| **部署** | **离线环境**：解压发布包后，在 **`config.yaml` 写入 `secret_key`（访问密钥）与 DSN**；首次打开网页输入 **`secret_key`** 通过校验后访问各页（或随后在 Web「设置」中改 DSN）；后续正式后端的 **API 放行** 也以同一 `secret_key` 作为识别依据（见 §6.1、§7.1） |

**不做**：全功能 DBA 平台、多租户、复杂权限模型（后续可延展）。

---

## 2. 核心能力边界（MVP）

仅覆盖约 90% 常见运维场景，避免功能膨胀：

1. **执行 SQL**（查询为主，写操作受配置约束）
2. **查看表结构**（`SHOW TABLES` / `DESCRIBE` / `SHOW CREATE TABLE`）
3. **浏览数据**（分页，严格上限行数）
4. **数据库 / 连接选择**（配置驱动下拉）
5. **连接参数（DSN）**：支持 **`config.yaml` 配置** 与 **Web 设置页修改**，二者对应 **同一套连接配置数据**（见 §4.3、§6.1）
6. **访问密钥 `secret_key`**：写在 **`config.yaml`**；用户进入业务页前在 **`/login`** 输入校验；会话建立后，**后续 SQL、设置及计划中的后端 API** 均须携带有效凭据（Cookie/Header），**识别与放行规则与 `secret_key` 一致**（可与反代 Basic Auth 叠加，见 §7.1）

---

## 3. 整体架构

```text
[浏览器]（可完全离线访问本服务）
   ↓ HTTPS（建议 Nginx 终结证书；纯内网也可用 HTTP + 限定网段）
[Go HTTP Server]
   ↓ html/template + 静态资源（仅本地，无公网 CDN）
   ↓ 读取/写入本地配置（`secret_key`、DSN 等）
   ↓ 会话：`secret_key` 校验 → 签发短期会话（Cookie 等）
[database/sql]
   ↓ 按 DSN 建立 TCP 连接
[MySQL]
```

**设计要点**：

- 业务逻辑分层：`handler`（HTTP）→ `service`（校验、只读策略）→ `dao`（SQL）
- **`secret_key`** 仅存服务端配置，**永不**在 UI 回显、不写进浏览器本地存储明文；所有写操作与 API 须在 **已通过 `secret_key` 建立会话**（或等效 Header）后执行
- **DSN 与数据库账号密码**仅存服务端配置文件；Web 仅通过设置页改写该文件，界面上密码以「置空表示不修改」等方式脱敏展示（§7.2）

---

## 4. 离线发布包与部署流程（核心诉求）

### 4.1 目标体验

运维在 **无外网** 的机器上：

1. 从介质复制 **对应版本、对应 OS/ARCH** 的压缩包；
2. 解压到约定目录；
3. 编辑 **`config.yaml`**：至少设置 **`secret_key`**（访问密钥）与 **`databases`（DSN）**；连接参数也可之后在 Web **`/settings`** 中修改并写回同一文件；
4. 启动二进制，浏览器访问服务端口 → 在 **`/login`**（或统一登录入口）输入 **`secret_key`**，通过后即可使用 SQL 工作台、表列表等；**后续若接入 JSON API**，请求需携带 **由 `secret_key` 校验后的会话 Cookie** 或约定的 **Header（例如 `X-MiniDBA-Secret` / Bearer，与实现一致）**，统一按同一密钥策略放行。

全程 **不要求** `go build`、不要求 `npm install`、不要求访问 Google Fonts 等外网资源。

### 4.2 发布物内容（建议）

每个平台一条构建产物，例如：

| 文件 | 说明 |
|------|------|
| `mini-dba-<version>-<os>-<arch>.tar.gz`（或 `.zip`） | 标准分发包 |
| `mini-dba` / `mini-dba.exe` | 主程序；推荐 `go:embed` 嵌入 `web/templates`、`web/static`，实现「单文件 + 旁路配置」 |
| `config.example.yaml` | 示例配置，可复制为 `config.yaml` |
| `README_OFFLINE.md`（可选） | 离线部署：权限、端口、`config.yaml` 与 Web 设置的关系 |

包内目录示例（静态资源若未 embed 则需带上 `web/`）：

```text
mini-dba-linux-amd64/
├── mini-dba
├── config.example.yaml
├── config.yaml          # 本地生成或复制；可含敏感信息，勿入库
└── README_OFFLINE.md
```

### 4.3 连接参数（DSN）：`config` 与 Web 页面

**同一 `config.yaml` 中**除 **`secret_key`**（§6.1）外，与库相关的「连接配置」指 **DSN** 及其附属字段（展示名、`readonly` 等）；**不包含**独立的主机/DNS 解析配置项。

| 方式 | 说明 |
|------|------|
| **`config.yaml`（或等价路径）** | 启动时加载；运维可直接编辑文件后启动或重启。 |
| **Web「设置 / 连接」页** | 与文件 **同一数据模型**、**同一事实源**：提交后 **校验**并 **原子写回**（如临时文件再 `rename`）到上述配置文件，避免半截配置。 |

两种途径 **等价**：可先写文件再启动，也可启动后在页面里改并保存；再次打开文件应看到与 Web 一致的 DSN 内容。

**说明**：DSN 中 MySQL 地址可为 IP 或主机名；若为主机名，能否解析取决于运行本进程的操作系统网络环境，**不属于 MiniDBA 的「可配置项」**，本文档亦不单独描述 DNS/`hosts` 运维。

**Web 设置页能力边界**：

- **可编辑**：各逻辑连接的 `name`、完整 **`dsn`**（或拆表单的 host/port/user/db 再由服务端拼成 `dsn`）、以及允许暴露的开关（如 `readonly`，若产品设计支持）。
- **密码**：仅在修改时填写；若留空则表示 **不改动** 当前已保存的密码片段；**永不**在响应 HTML 中回显明文密码。
- **生效**：保存后 **重载连接池**，或 MVP 阶段提示 **重启后生效**（二选一，实现时择一写死并在界面提示）。
- **权限**：该页须置于 **强鉴权** 之后；后续可单独区分「仅管理员可改 DSN」。

### 4.4 与前端离线一致

所有字体、图标、CSS、JS **必须**：

- 通过 `go:embed` 打进二进制，或  
- 放在发布包相对路径下由进程 `http.FileServer` 提供  

**禁止**在离线交付物中依赖 `fonts.googleapis.com` 等外链（开发机可暂用 CDN，发布构建需切换为本地资源）。详见 §8.1、§8.7。

---

## 5. 建议目录结构

```text
mini-dba/
├── main.go
├── config.yaml
├── internal/
│   ├── handler/
│   ├── service/
│   ├── dao/
│   ├── config/          # 加载、校验、保存、脱敏
│   └── model/
├── web/
│   ├── templates/
│   └── static/
│       ├── css/
│       ├── js/
│       └── fonts/       # 离线：Roboto、图标字体或内联 SVG
└── docs/
    └── DESIGN.md
```

---

## 6. 功能设计摘要

### 6.1 连接与鉴权（`secret_key` + DSN）

**`secret_key`（访问密钥）**

- 字段名：**`secret_key`**，置于 **`config.yaml` 根级**，字符串；部署时 **足够随机、足够长度**，与源码/仓库隔离。
- **网页**：未建立会话时访问任意受保护路径 → **重定向至 `/login`**；用户输入 **`secret_key`**，服务端与配置值 **常量时间比对**（或对摘要比对）通过后，签发 **短期会话**（推荐 **HttpOnly + SameSite** 的 Cookie；令牌内容由 `secret_key` 参与 **HMAC-SHA256** 等与过期时间绑定，**不把明文密钥写入 Cookie**）。
- **后续后端 / API**：同一 **`secret_key`** 作为 **识别放行的根信任**：浏览器可沿用会话 Cookie；若机器调用 / `fetch` API，可约定 **`Authorization: Bearer <token>`** 或 **`X-MiniDBA-Secret`**（实现时二选一、写清文档），**服务端均以配置中的 `secret_key` 校验或派生验证**；敏感写接口仍建议仅内网或 mTLS。

**配置文件**（节选）示例：

```yaml
secret_key: "请替换为长随机串-勿提交真实环境"
listen: "127.0.0.1:8080"
readonly: true
databases:
  - name: prod
    dsn: "user:pass@tcp(db.example.internal:3306)/appdb"
```

界面：顶部 **数据库选择器**（MUI 风格 Select）；切换逻辑可依赖 Cookie / Session / 表单隐藏字段（须在 **`secret_key` 会话**有效前提下）。

**Web 设置页**：对 **`databases`** 列表增删改（受权限与校验约束）；保存即写回磁盘配置并按 §4.3 生效。**不**在设置页提供修改 **`secret_key`** 的便捷入口（若需轮换，以编辑文件或专用运维流程为准，避免与日常 DSN 编辑混同）。

### 6.2 SQL 执行器

- POST 提交 SQL 文本；服务端解析结果集为列名 + 二维字符串（或 `null` 标记）
- **行数与结果集大小上限**（例如单次最多 100 行、总单元格数上限），防止 OOM
- 区分 `Query` / `Exec`：受影响行数、最后插入 ID 等以结构化信息展示

### 6.3 表结构 / 元数据

统一走固定白名单语句或封装函数：`SHOW TABLES`、`DESCRIBE`、`SHOW CREATE TABLE`，避免拼接用户输入到元数据查询以外路径。

### 6.4 数据浏览

- `SELECT * FROM … LIMIT ? OFFSET ?`，默认页大小与最大页大小可配置
- URL 或表单参数：`table`、`page`，与 SQL 工作台解耦或复用同一结果表格组件（模板 partial）

---

## 7. 安全设计（必须项）

### 7.1 传输与接入

- 生产建议 **仅监听 127.0.0.1**，由 Nginx 反代并做 TLS；离线内网若直连端口，需网络 ACL 限制来源
- **主鉴权**：**`config.yaml` 的 `secret_key`** → 网页登录与会话；**同一密钥语义**延伸至后续 **HTTP API**（Cookie 会话或约定 Header/Bearer），未通过校验的请求一律 **401/403**
- **可选补充**：Nginx **Basic Auth**、IP 白名单等，与 `secret_key` **叠加**亦可

### 7.2 配置与密钥

- **配置落盘**：`chmod` 仅运行用户可读；`**secret_key`** 与 DSN 密码同等级保护
- **Web 保存配置**：校验字段、防路径穿越；**登录与 CSRF**：表单类操作在 Cookie 会话下需 **CSRF Token**（见 §7.4）
- **口令比对**：用户提交的 `secret_key` 与配置比对时，避免按字节长度分支泄露信息；宜对 **双方 SHA-256 摘要** 做 `subtle.ConstantTimeCompare`，或对 HMAC 结果比较

```go
// 示例：校验用户输入的 secret_key 与配置（摘要恒定时间比较）
func secretMatch(input, fromConfig string) bool {
    h1 := sha256.Sum256([]byte(input))
    h2 := sha256.Sum256([]byte(fromConfig))
    return subtle.ConstantTimeCompare(h1[:], h2[:]) == 1
}
```

### 7.3 SQL 风控

- **只读模式**：`readonly: true` 时拒绝写操作与 DDL（MVP 可关键字列表）
- **危险语句拦截**：如 `DROP DATABASE`、`TRUNCATE`（可按环境配置黑名单）

### 7.4 其他

- **`secret_key` 登录后** 的 POST（SQL、设置保存等）建议 **CSRF Token**
- 日志：记录库名、SQL 摘要；**禁止**记录 **`secret_key` 明文**、完整 DSN 密码或登录表单原文

---

## 8. 前端与视觉设计（Material UI 取向 · 离线友好）

### 8.1 与 MUI 的关系

- **[MUI](https://mui.com/)** 本质是 React 组件库；本项目 MVP 为 **Go template**。
- **做法**：遵循 **Material Design 3** 与 MUI 默认主题习惯；**字体与图标文件随二进制或发布包分发**，不使用公网 CDN。

若后续需要：可另起 `web/ui` 使用 React + MUI，构建产物同样 **打包进发布物**，原则与 §4.4 一致。

### 8.2 设计令牌（建议与 MUI 默认 LIGHT 大致对齐）

可在 `web/static/css/theme.css` 集中定义 CSS 变量。

| 令牌 | 建议值 | 说明 |
|------|--------|------|
| `--md-primary` | `#1976d2`（或 M3 主色 `#0061a4` 系） | 主按钮、链接、焦点环 |
| `--md-primary-hover` | 主色加深 8% | Hover |
| `--md-surface` | `#ffffff` | 卡片、表单区背景 |
| `--md-bg` | `#fafafa` | 页面底色 |
| `--md-error` | `#d32f2f` | 错误提示 |
| `--md-on-surface` | `rgba(0,0,0,0.87)` | 主文案 |
| `--md-on-surface-secondary` | `rgba(0,0,0,0.6)` | 次要文案 |
| `--md-divider` | `rgba(0,0,0,0.12)` | 表格线、分割线 |
| `--md-radius-sm` | `4px` | 输入框、小按钮 |
| `--md-radius-md` | `8px` | 卡片 |
| `--md-elevation-1` | `0px 1px 3px rgba(0,0,0,0.12)` | 顶栏 / 浮动卡片 |

**排版**：标题 `600` / `500`，正文 `400`；行高约 `1.5`；基础字号 `14px`。

**间距**：`8 / 16 / 24` 阶梯。

### 8.3 布局与信息架构（IA）

```text
┌─────────────────────────────────────────────┐
│ AppBar：产品名 | 当前库 Select | 只读徽章      │
├────────┬────────────────────────────────────┤
│ Nav    │ 主工作区（Card 包裹）                 │
│ · SQL  │ · SQL 多行输入（类 TextField）        │
│ · 表   │ · Toolbar：执行 / 清空 / 格式化(可选) │
│ · 浏览 │ · 结果：表格（sticky header、斑马纹）  │
│ · 设置 │ · 连接 / 部署说明链接（离线 README）   │
└────────┴────────────────────────────────────┘
```

### 8.4 关键组件映射（模板 + CSS 实现）

| 意图 | MUI 参考 | 模板实现要点 |
|------|-----------|----------------|
| 顶栏 | `AppBar` | 背景 `--md-primary`，标题白字，高度 `56px`–`64px` |
| 侧栏 | `Drawer` + `List` | 当前页 `aria-current` + 左侧主色条 |
| 表单 | `TextField` multiline | `textarea` + 聚焦 `outline` 2px 主色 |
| 主操作 | `Button` contained | 主色底、白字 |
| 次操作 | `Button` outlined | 透明底、主色边框 |
| 结果区 | `Card` + `Table` | 表头 `sticky`、行 hover |
| 消息 | `Alert` | 成功/错误/警告条 |
| 加载 | `CircularProgress` | 纯 CSS / 内联 SVG |

### 8.5 简洁好看的交互细节

- 空状态、错误提示、表格溢出与无障碍同前文
- **设置页**：分组展示连接列表；危险操作（删除连接）二次确认

### 8.6 离线静态资源（必选实现路径）

- **字体**：将 **Roboto**（或 [Roboto Flex](https://fonts.google.com/) _subset 后）的 `woff2` 置于 `web/static/fonts/`，`@font-face` 引用相对路径。
- **图标**：优先 **内嵌 SVG** 或 **单文件图标字体** 本地化；避免依赖 `fonts.googleapis.com` 的 Material Symbols。
- **构建门禁**：CI 中校验发布模板 **不得** 包含 `https://fonts.googleapis.com` 等外链（grep 检测）。

### 8.7 开发阶段可逆说明

开发机为便捷可使用 CDN；合并到 **release** 分支或打离线包前运行「资源本地化」脚本（或由 `//go:build` tag 区分 embed 目录）。文档与 Taskfile / CI 应对齐该约定。

---

## 9. HTTP 路由（草案）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET/POST | `/login` | 输入 **`secret_key`**，建立会话 |
| GET | `/logout` | 清除会话 |
| GET | `/` | SQL 工作台（需已通过 `secret_key` 会话） |
| GET/POST | `/query` | SQL 执行与结果展示 |
| GET | `/tables` | 表列表 |
| GET | `/table/:name/schema` | 表结构 |
| GET | `/table/:name/rows` | 分页数据浏览 |
| POST | `/switch-db` | 切换当前逻辑库 |
| GET/POST | `/settings` | **连接（DSN）配置**：须在有效 **`secret_key` 会话** 下 |
| GET | `/healthz` | 健康检查 |

---

## 10. 构建与部署

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o mini-dba .
```

前置机执行：`./mini-dba -config ./config.yaml`（参数名以实现为准）。Nginx 反代示例：

```nginx
location /dba/ {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

**离线部署检查清单**（可并入 `README_OFFLINE.md`）：

- [ ] 解压目录与可执行权限  
- [ ] `config.yaml` 已设 **非默认、足够强的 `secret_key`**  
- [ ] `databases` / DSN 已配置且 MySQL **网络可达**  
- [ ] 防火墙 / 安全组放行监听端口  
- [ ] （可选）反代 TLS、Basic Auth、IP 限制  

---

## 11. 演进路线

### 第一阶段（当前 MVP）

- 离线包、embed 静态资源、**`secret_key` 登录与会话**、SQL / 表 / 分页、只读 / 黑名单、**Web 连接（DSN）设置**、MUI 取向界面；API 与页面共用放行语义

### 第二阶段

- SQL 历史、导出 CSV、连接 **热重载**（无需重启）

### 第三阶段

- 前端 SQL 高亮（资源仍需本地化）、`EXPLAIN` 可视化、RBAC

### 可选：React + MUI

构建产物必须随发布包分发；禁止仅引用外网 chunk。

---

## 12. 参考代码片段（SQL 结果扫描）

与 MVP 实现思路一致，生产需补足 `rows.Err()`、`context`、类型与 `NULL` 处理等。

```go
func ExecSQL(db *sql.DB, query string) (*Result, error) {
    rows, err := db.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    cols, _ := rows.Columns()
    var results [][]string
    for rows.Next() {
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
            if v == nil {
                row[i] = "NULL"
            } else {
                row[i] = fmt.Sprintf("%v", v)
            }
        }
        results = append(results, row)
    }
    if err := rows.Err(); err != nil {
        return nil, err
    }
    return &Result{Columns: cols, Rows: results}, nil
}
```

---

## 13. 竞品对标边界

**不对标** phpMyAdmin、DBeaver、CloudBeaver 的完整功能集；本工具定位为 **「内网场景下的数据库瑞士军刀」** —— 轻、快、可审计、**可离线投递**、易部署。

---

*文档版本：与 MiniDBA 仓库同步迭代。*
