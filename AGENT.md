# MiniDBA — 协作说明（面向人与编码代理）

本文件帮助在仓库内快速对齐目标、约束与命令。**实现细节与路由、安全、离线包要求以 `docs/DESIGN.md` 为准**；若与本文冲突，以设计文档为准。

## 产品定位

- **Go 单二进制**，内网轻量 MySQL 运维 Web：**执行 SQL**、表元数据、分页浏览、多连接（DSN）配置。
- **`config.yaml`**：`secret_key`（网页登录 / 后续 API 放行）、`databases`（DSN）、`readonly` 等。
- **前端**：MUI 取向的 Material 风，服务端模板 + 随包静态资源（无公网 CDN）；当前 **`web/demo/`** 为定稿用静态 Demo，**`cmd/demo`** 提供登录与静态托管。

## 仓库结构（预期演进）

```text
main.go              # 正式入口（当前占位）
cmd/demo/            # UI 预览：secret_key 登录 + 静态页
internal/            # 正式开发：handler / service / dao / config（待建）
web/demo/            # 静态 Demo HTML
web/static/css/      # theme.css，可迁模板后复用
docs/DESIGN.md       # 设计规格
config.example.yaml  # 可提交的示例配置（复制为 config.yaml）
```

## 本地命令

| 用途 | 命令 |
|------|------|
| UI Demo | 根目录：复制 `config.example.yaml` → `config.yaml` 并设 `secret_key`，再 `task demo` 或 `go run ./cmd/demo` |
| Demo 监听 / 配置 | `MINIDBA_DEMO_ADDR`、`MINIDBA_CONFIG`（默认 `config.yaml`） |
| 格式化 | `task golangci:fmt` 或 `go fmt ./...` |
| 主程序占位 | `task run` → `go run main.go` |

## 配置与密钥

- **`config.yaml` 已加入 `.gitignore`，勿提交。** 团队仅用 **`config.example.yaml`** 入库。
- 含真实 `secret_key`、DSN 密码的文件限制权限，与设计文档 §7 一致。

## 编码约定

- **Go 版本**：见 `go.mod`。
- **改动范围**：与设计/MVP 相关；避免无关大范围重构。
- **依赖**：新增依赖保持理由清晰；离线构建与设计 §4 一致。
- **正式 HTTP**：设计文档 §9（`/login`、`/logout`、`/query`、`/settings` 等）；Demo 路由以 `cmd/demo` 为准直至合并进主程序。

## 代理工作协议（建议）

1. 接到需求时先核对 **`docs/DESIGN.md`** 对应章节。
2. 改配置模型或安全行为时，同步更新 **设计文档 + `config.example.yaml`**。
3. UI 变更可先对齐 **`web/demo`** 或 **`web/static/css/theme.css`**，再接入 `html/template`。

---

*随里程碑更新本文；大块架构变更必须反映到 `docs/DESIGN.md`。*
