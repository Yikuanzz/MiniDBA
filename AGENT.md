# MiniDBA — 协作说明（面向人与编码代理）

本文件帮助在仓库内快速对齐目标、约束与命令。**实现细节与路由、安全、离线包要求以 `docs/DESIGN.md` 为准**；若与本文冲突，以设计文档为准。

## 产品定位

- **Go 单二进制**，内网轻量 MySQL 运维 Web：**执行 SQL**、表列表、表结构、分页浏览、多连接（DSN）与 **`/settings` 保存写回 `config.yaml`**。
- **`config.yaml`**：`secret_key`（`/login` 与会话 Cookie；亦可 **`X-MiniDBA-Secret`** / **`Authorization: Bearer`**）、`databases`（DSN）、`readonly`、`max_result_rows` 等。
- **前端**：`html/template` + [`web/static/css/theme.css`](web/static/css/theme.css)；**`go:embed`** 打包 `web/templates` 与 `web/static`。历史目录 **`web/demo/`** 仅作可选参考，正式路由以主程序为准。

## 仓库结构

```text
main.go              # 入口：-config 指定 yaml
internal/
  auth/              # HMAC 会话、Header 放行
  csrf/              # Double-submit Cookie
  config/            # 加载/校验/原子保存
  dbmgr/             # 连接池与热重载
  server/            # 路由与页面
  sqlrun/            # 只读/黑名单/执行
web/templates/       # 布局与各页
web/static/          # 主题 CSS
docs/DESIGN.md
config.example.yaml
```

## 本地命令

| 用途 | 命令 |
|------|------|
| 运行 | 复制 `config.example.yaml` → `config.yaml`，`task run` 或 `go run . -config config.yaml` |
| 测试 | `go test ./...` |
| Linux 构建 | `task build` → `./release/minidba` |
| 格式化 | `task golangci:fmt` / `go fmt ./...` |

## 配置说明

- **`config.yaml` 勿提交**（见 `.gitignore`）；团队以 **`config.example.yaml`** 为范本。
- **`config.Save` 整文件重写后 YAML 注释会丢失**（MVP 已知限制）。
- 修改 **`secret_key` 仅手改文件并重启进程**，设置页不轮换该字段。

## 编码约定

- 改动与设计/MVP 对齐；避免无关大范围重构。
- 所有 **POST**（除 `/login`）需 **CSRF**；业务接口需 **`secret_key` 会话或 Header**。

## 代理工作协议（建议）

1. 接到需求时先核对 **`docs/DESIGN.md`**。
2. 改配置模型或安全行为时，同步更新 **设计文档 + `config.example.yaml`**。
3. 样式以 **`theme.css`** 与现有模板块为准。

---

*随里程碑更新本文；大块架构变更必须反映到 `docs/DESIGN.md`。*
