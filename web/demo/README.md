# MiniDBA · 静态 UI Demo

依据 `docs/DESIGN.md` 的 MUI 取向与信息架构；**与正式版一致**：先用 **`config.yaml` 中的 `secret_key`** 在 `/login` 登录，再访问静态页。

## 访问校验

1. 仓库根目录确保存在 **`config.yaml`**，且包含非空 **`secret_key`**（可用 `config.example.yaml` 复制改名）。
2. 启动 demo 后打开根路径，会 **302 → `/login`**。
3. 输入与配置一致的 **`secret_key`** 后进入 SQL 工作台等页面；顶栏 **退出** 可清除会话。

默认示例：`config.yaml` 中 `secret_key: "minidba-demo-change-me"`（正式环境务必更换）。

## 页面

| 路径 | 内容 |
|------|------|
| `/login` | 输入 `secret_key` |
| `/logout` | 清除 Cookie |
| `/` | SQL 工作台 |
| `/tables.html` | 表列表 |
| `/browse.html` | 数据浏览 |
| `/settings.html` | 连接（DSN）配置表单 |

## 本地预览

在**仓库根目录**执行：

```bash
task demo
# 或
go run ./cmd/demo
```

浏览器打开终端输出的地址（默认 `http://127.0.0.1:18899/`）。

环境变量：

| 变量 | 含义 |
|------|------|
| `MINIDBA_DEMO_ADDR` | 监听地址，如 `:8080` |
| `MINIDBA_CONFIG` | 配置文件路径，默认 `config.yaml` |

## 样式

- `web/static/css/theme.css`：设计令牌与组件（与设计文档 §8 对齐）

正式开发时可把本目录 HTML 迁为 `html/template`，会话中间件与当前 demo 行为对齐。
