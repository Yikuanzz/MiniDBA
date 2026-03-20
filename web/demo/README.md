# MiniDBA · 历史静态 UI 草图（参考）

**正式程序已迁移至** 根目录 `go run . -config config.yaml`（模板在 `web/templates/`，逻辑在 `internal/server/`）。本目录 **`.html` 仅作布局参考**，不再维护独立 `cmd/demo` 服务。

样式仍与正式版共用 **`web/static/css/theme.css`**。

---

以下为原 Demo 说明（已过时）：

- 曾使用 `cmd/demo` + 静态页；现请使用主二进制的 `/login`、`/`、`/tables` 等路由。
