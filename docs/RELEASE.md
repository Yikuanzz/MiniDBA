# MiniDBA 发版说明（GitHub Actions）

本文描述在 **GitHub Releases** 上自动构建、打包多平台制品的流程，并与 [DESIGN.md](./DESIGN.md) §4 一致：**复制压缩包 → 解压 → 配置 `config.yaml` → 启动即可**。静态资源由 `go:embed` 打进二进制，发布包内 **不需要** 单独的 `web/` 目录。

---

## 1. 发版目标与制品形态

| 目标 | 说明 |
|------|------|
| 多平台 | `linux/amd64`、`linux/arm64`、`windows/amd64`、`windows/arm64`、`darwin/amd64`、`darwin/arm64`（由 [`.github/workflows/release.yml`](../.github/workflows/release.yml) 定义）。 |
| 归档格式 | Unix 系为 **`mini-dba-<tag>-<os>-<arch>.tar.gz`**，Windows 为 **`.zip`**（内含 `mini-dba.exe`）。 |
| 包内文件 | **`mini-dba` 或 `mini-dba.exe`**、`config.example.yaml`、`README.md`、`LICENSE`。 |
| 校验 | 同目录提供 **`mini-dba_<tag>_checksums.txt`**（`sha256sum`）。 |

真实的 **`config.yaml`** 不进仓库、不打进包；由用户从示例复制后填写。

---

## 2. GitHub 侧（一次性）

- 仓库在 **GitHub**，默认分支可跑 Actions。
- 工作流 **[`release.yml`](../.github/workflows/release.yml)**：`push` **`tags`** 匹配 **`v*`** 时，在 `ubuntu-latest` 上交叉编译、打包、`softprops/action-gh-release` 上传；需 `contents: write`（已设）。
- 离线资源门禁：`go run ./scripts/check_offline_urls.go`（与 [CI](../.github/workflows/ci.yml) 一致）。

首次可打测试 tag（如 `v0.0.0-test1`）验证 Release 资产后按团队规范清理。

---

## 3. 维护者日常操作

1. 确认主分支与检查通过（含 CI）。
2. **打 tag 并推送**（任选）：
   ```bash
   git tag -a v0.1.0 -m "Release v0.1.0"
   git push origin v0.1.0
   ```
   或使用 Task：`task release:tag VERSION=v0.1.0`（需已配置 `origin`）。
3. 在 Actions 中查看 **Release** workflow；在 Releases 页核对各平台归档与 checksum。
4. 抽检解压：应含 **`mini-dba` / `mini-dba.exe`** 与 **`config.example.yaml`**。

---

## 4. 本地：Task（Windows / Linux）

| 任务 | 说明 |
|------|------|
| `task build` | 本机构建 **Linux amd64** 至 `./release/mini-dba`（与日常开发/前置机构建习惯一致）。 |
| `task release:verify-offline` | 与 CI 相同的公网字体 CDN 检查。 |
| `task release:tag VERSION=v0.1.0` | 打附注 tag 并 `git push origin`（触发远程发版）。 |

多平台包 **仅在推送 tag 后由 CI 生成**，无需在本地安装额外发版工具。

---

## 5. 终端用户：解压与配置

与 [DESIGN.md §4.1](./DESIGN.md) 一致：

1. 下载对应平台归档并解压。
2. 复制 **`config.example.yaml`** → **`config.yaml`**，设置 **`secret_key`**、**`databases`** 等。
3. 启动：
   - Linux / macOS：`./mini-dba -config ./config.yaml`
   - Windows：`mini-dba.exe -config config.yaml`
4. 浏览器访问监听地址，在 **`/login`** 使用 **`secret_key`** 登录。

部署检查清单见 [DESIGN.md §10](./DESIGN.md#10-构建与部署)。

---

## 6. 文档索引

- 发版：**本文** + [`.github/workflows/release.yml`](../.github/workflows/release.yml)。
- 产品设计：**[`docs/DESIGN.md`](./DESIGN.md)**。
