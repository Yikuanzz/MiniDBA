# MiniDBA

内网场景下的轻量 MySQL Web 运维小工具（Go，单二进制 + 本地配置）。设计说明见 [docs/DESIGN.md](docs/DESIGN.md)。

## 运行（开发）

```bash
go run . -config config.yaml
```

或使用 [Task](https://taskfile.dev/)：

```bash
task run
```

从 `config.example.yaml` 复制为 `config.yaml`，设置 `secret_key` 与 `databases`；若挂在反代子路径下，另设 **`base_path`**（见 `docs/DESIGN.md` §10）。

## 构建与发版

- 本地 Linux amd64：`task build`
- 发版流程（GitHub Actions，推送 `v*` tag）：见 [docs/RELEASE.md](docs/RELEASE.md)

预编译包见 **GitHub Releases**。
