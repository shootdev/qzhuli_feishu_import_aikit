# 飞书企业自建应用创建套件

基于 [飞书官方 lark-cli](https://github.com/larksuite/cli) `v1.0.100` 的定制工具，包含 CLI 源码、预编译程序包和可复用 AI 技能。定制版 `1.0.100-docs7.2` 并非飞书官方发行版。

默认创建配置：

- 七项文档**用户身份**权限：`docs:document.content:read`、`docs:document.media:download`、`docx:document:readonly`、`drive:drive:readonly`、`offline_access`、`space:document:retrieve`、`wiki:wiki:readonly`。
- 一项**应用身份**权限：`application:application:patch`，用于配置和发布应用自身。
- OAuth 回调：`http://127.0.0.1:4366/oauth/callback`；发布范围：所有员工。

CLI 会以最小模板申请上述权限，并限制用户 OAuth 只请求七项。`application:application:patch` 可以修改应用自身配置；该 CLI 并不会拦截所有通过原始 API 发出的管理操作，使用时仍需核验实际权限清单。

## 下载和安装

选择与接收者系统相符的小包，下载并解压，再按照包内 `README.md` 安装：

| 系统 | 下载 | 大小约 |
| --- | --- | --- |
| Windows Intel / AMD | [Windows x64](dist/feishu-app-kit-1.0.100-docs7.2-windows-x64.zip) | 16.5 MiB |
| Windows ARM | [Windows ARM64](dist/feishu-app-kit-1.0.100-docs7.2-windows-arm64.zip) | 14.8 MiB |
| macOS Apple Silicon / Intel | [Python 脚本包](dist/feishu-app-kit-1.0.100-docs7.2-macos-python.zip) | 纯 Python |
| Linux x64 | [Linux x64](dist/feishu-app-kit-1.0.100-docs7.2-linux-x64.zip) | 16.1 MiB |
| Linux ARM64 | [Linux ARM64](dist/feishu-app-kit-1.0.100-docs7.2-linux-arm64.zip) | 14.7 MiB |

[下载全平台包（含完整源码）](dist/feishu-app-kit-1.0.100-docs7.2-full-with-source.zip)。全平台 ZIP 因大小使用 Git LFS 存储；在 GitHub 页面可下载该文件。使用 Git 克隆并自动获取它需要安装 Git LFS。Windows/Linux 小包包含对应二进制、技能与源码补丁，macOS 小包包含 Python 实现和技能，完整 CLI 源码可从本仓库 `src/lark-cli/` 获取。

**macOS（Intel / Apple Silicon）**

解压 Python 包，在解压目录使用 Python 3.8+ 执行：

```sh
python3 feishu_app.py preview
python3 feishu_app.py create --output "$HOME/feishu-app-credentials.json"
```

同一脚本负责创建确认、核验七加一权限、设置回调、提交全员发布，以及交付本次凭据。可选的 `user-login` 用于员工 OAuth。二维码模块随包提供，直接运行无需 pip、Go 或 Node.js。要安装技能供 Agent 自动发现，可运行 `sh ./install.sh`。[macOS 使用说明](skills/feishu-enterprise-app-setup/references/macos-python.md)

**Windows / Linux**

Windows 小包在 PowerShell 中运行 `./install.ps1 -AddToPath`；Linux 运行 `sh ./install.sh --add-to-path`。默认安装到当前用户目录，安装过程不创建应用或发起授权。安装后重新打开终端或 AI 应用，并运行：

```text
lark-cli --version
lark-cli config init --new --dry-run
```

确认版本为 `1.0.100-docs7.2`，预览包含七项 user 权限、一项 tenant 管理权限、固定回调和全员范围。首次创建仍需本人登录飞书、选择企业并确认；企业要求审核时需完成审核。CLI 提交发布不等于审核通过。

Windows x64 和 Linux x64 原生套件已通过安装与执行验证，Windows/Linux ARM64 已交叉编译。macOS Python 版的测试通过本地模拟服务验证创建和发布流程，真实飞书发布验证记录来自原 Go 版。[验证记录](VALIDATION.md)

## 使用 AI 技能

把小包解压到能访问本机文件和终端的 AI 工作环境，或安装后让其读取 [`skills/feishu-enterprise-app-setup/SKILL.md`](skills/feishu-enterprise-app-setup/SKILL.md)。例如：

```text
使用 $feishu-enterprise-app-setup 创建飞书企业自建应用：
保持七项文档用户权限和一项应用管理权限，使用默认回调，发布给所有员工。
优先通过 CLI 完成；需要我确认企业或审批时告诉我。
完成后交付 App ID、App Secret、重定向 URL，并清理本次本机 CLI 绑定。
```

工具会核对 App ID 与 profile 并将完整凭据导出到用户自己的新文件，不向终端打印密钥。若 AI 环境不能安全地在答复正文中显示 Secret，应提供私密文件并说明。包和仓库不含本次实测应用的真实凭据或登录状态。

## 源码

- [`src/lark-cli/`](src/lark-cli/)：完整定制 CLI 源码快照，修改版提交 `2948de1b8331f041cb1205f99b097b5aaa551f73`，不包含原仓库 Git 历史。
- [`src/credential-export/`](src/credential-export/)：凭据导出工具源码及测试。
- [`src/README.md`](src/README.md)：本地测试和重建命令。Go 模块依赖未 vendored，缓存缺失时需联网。
- [`skills/`](skills/)：主技能和 `lark-shared` 依赖技能。
- [`templates/`](templates/)：固定回调、全员范围和发布请求模板。
- [`VALIDATION.md`](VALIDATION.md)：本地测试与真实应用验证范围。

原始 CLI 的 MIT 许可见 [`LICENSE`](LICENSE)，第三方许可见 [`third-party-licenses/`](third-party-licenses/)。
