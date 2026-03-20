<div align="center">

![MiniDBA banner](pic/icon_banner.png)

### MiniDBA

**内网场景下的轻量 MySQL Web 运维工具** — 单二进制、`go:embed` 离线前端、YAML 配置即可跑。

[![CI](https://github.com/Yikuanzz/MiniDBA/actions/workflows/ci.yml/badge.svg)](https://github.com/Yikuanzz/MiniDBA/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Yikuanzz/MiniDBA?sort=semver)](https://github.com/Yikuanzz/MiniDBA/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://go.dev/)

[设计文档](docs/DESIGN.md) · [发版说明](docs/RELEASE.md)

</div>

---

## 简介

MiniDBA 面向 **前置机 / 内网 / 最小暴露面** 下的临时查数、执行 SQL（受只读与黑名单约束）、浏览表数据与维护连接配置。界面为服务端模板渲染，**不依赖 npm 与外链静态资源**，适合无外网环境部署。

## 特性

| | |
| --- | --- |
| **单文件交付** | 一个可执行文件 + 本地 `config.yaml`，模板与静态资源内置 |
| **多连接** | 多个逻辑库（DSN），顶栏切换；Web「连接设置」写回配置文件并热重载连接池 |
| **安全基线** | `secret_key` 登录、会话 Cookie、POST CSRF；支持 `X-MiniDBA-Secret` / Bearer 头 |
| **反代子路径** | 可选 `base_path`，与 Nginx `location` + 尾斜杠 `proxy_pass` 配合（见设计文档 §10） |

## 环境要求

- **Go** `1.24+`（仅开发与从源码构建时需要）
- **MySQL**（由 DSN 指定），运行 MiniDBA 的机器需能访问数据库端口

## 快速开始

### 从源码运行

```bash
cp config.example.yaml config.yaml
# 编辑 config.yaml：设置 secret_key 与 databases
go run . -config config.yaml
```

使用 [Task](https://taskfile.dev/)：

```bash
task run
```

### 预编译包

从 [Releases](https://github.com/Yikuanzz/MiniDBA/releases) 下载对应平台归档，解压后同样复制并编辑 `config.yaml`，执行：

```bash
./mini-dba -config ./config.yaml
```

## 配置要点

| 项 | 说明 |
| --- | --- |
| `secret_key` | 访问密钥；`/login` 与后续会话；勿提交入库 |
| `listen` | 监听地址，如 `127.0.0.1:18080`；反代场景常仅本机 |
| `databases` | 逻辑名 + MySQL DSN |
| `readonly`、`max_result_rows` 等 | 见 [`config.example.yaml`](config.example.yaml) 注释 |
| `base_path` | 挂在子路径反代时填写，如 `/dba`（与 Nginx 前缀一致，无尾斜杠） |

完整行为与安全说明见 **[`docs/DESIGN.md`](docs/DESIGN.md)**。

## 构建

```bash
task build    # Linux amd64 → ./release/mini-dba
```

本地也可：`CGO_ENABLED=0 go build -ldflags="-s -w" -o mini-dba .`

## 测试

```bash
go test ./...
```

## 相关文档

- [**设计文档**](docs/DESIGN.md) — 架构、路由、部署与 Nginx 示例
- [**发版流程**](docs/RELEASE.md) — 推送 `v*` tag 触发 GitHub Actions 多平台打包

## 许可证

[MIT](LICENSE)
