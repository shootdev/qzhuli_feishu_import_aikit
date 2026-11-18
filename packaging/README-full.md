# 飞书企业自建应用创建套件

版本：`1.0.100-docs7.2`。这是基于飞书官方 `lark-cli v1.0.100` 的定制版本，不是飞书官方发行包。

套件包含独立 CLI 程序、凭据安全导出工具、完整的定制 CLI 源码、`feishu-enterprise-app-setup` 主技能、`lark-shared` 依赖技能、安装脚本、配置模板、源码补丁和校验文件。运行 CLI 不需要 Node.js、npm 或 Go；安装脚本也不联网下载程序。

## 支持的系统

| 系统 | 处理器 | 包内目录 |
| --- | --- | --- |
| Windows | Intel / AMD x64 | `bin/windows-amd64` |
| Windows | ARM64 | `bin/windows-arm64` |
| macOS | Intel / Apple Silicon | `feishu_app.py`（Python 3.8+） |
| Linux | x64 | `bin/linux-amd64` |
| Linux | ARM64 | `bin/linux-arm64` |

macOS 使用 Python 脚本。解压后运行 `python3 feishu_app.py preview`，再运行 `python3 feishu_app.py create --output "$HOME/feishu-app-credentials.json"`。具体交付、恢复和用户授权见 `skills/feishu-enterprise-app-setup/references/macos-python.md`。以下原生命令用于 Windows 和 Linux。

## 安装

先完整解压，再在套件目录运行。安装脚本先核验 `SHA256SUMS`，自动选择当前系统/架构。无需管理员权限。

### Windows / PowerShell

```powershell
.\install.ps1 -AddToPath
```

默认将 CLI 放到当前用户的 `%LOCALAPPDATA%\FeishuAppKit\bin`，技能放到 `$CODEX_HOME\skills`；未设置 `CODEX_HOME` 时使用用户目录的 `.codex\skills`。`-AddToPath` 会更新用户 PATH，并更新当前 PowerShell 的 PATH。

自定义目录，或为其他支持 SKILL.md 的 Agent 安装：

```powershell
.\install.ps1 -BinDir 'D:\Tools\FeishuAppKit\bin' -SkillsDir 'D:\MyAgent\skills'
```

如果系统不允许运行安装脚本，可按下方“手动安装”复制程序与技能；不需要修改系统全局执行策略。

### Linux

```sh
sh ./install.sh --add-to-path
```

默认 CLI 目录为 `~/.local/bin`，技能目录为 `${CODEX_HOME:-~/.codex}/skills`。`--add-to-path` 会为 bash、zsh 或 sh 在用户的 shell 配置中追加 PATH；其他 shell 会提示手动配置。完成后打开新终端，并重启需要使用该 CLI 的 Agent 应用。

自定义目录：

```sh
sh ./install.sh --bin-dir "$HOME/tools/feishu/bin" --skills-dir "$HOME/my-agent/skills"
```

两个安装脚本默认保留已有 `lark-shared`。确需替换时，分别加 `-ReplaceSharedSkill` 或 `--replace-shared-skill`。已有主技能和同位置 CLI 会先备份到所选 CLI 目录的 `backups/`，再更新包内文件；不会删除其他技能或应用配置。

## Windows / Linux 安装后检查

```text
lark-cli --version
lark-cli config init --new --dry-run
```

版本应为 `1.0.100-docs7.2`。预览必须包含：

- `permission_preset: docs7-managed`、`preset: false`、`create_only: true`。
- 七项用户权限，见下表。
- 一项应用身份权限：`application:application:patch`。
- 回调 `http://127.0.0.1:4366/oauth/callback`，开启刷新，设置所有员工可用。

这些检查不会创建应用或发起授权。若版本不是预期值，说明 PATH 中另一份 CLI 排在前面；先使用安装脚本输出的完整路径，或调整 PATH 后重开终端/Agent。

## 用技能创建应用

让支持 SKILL.md 的 Agent 加载安装的技能，然后发送：

```text
使用 $feishu-enterprise-app-setup 创建飞书企业自建应用：
七项文档用户权限，加 application:application:patch 应用身份权限；
默认回调不变，发布给所有员工，优先通过 CLI 配置与发布；
任务完成时直接列出 App ID、完整 App Secret 和回调；
交付后清理本次临时本机绑定。
```

首次仍需用自己的飞书账号选择企业并确认创建。创建页应只有约定的七项用户权限和一项应用身份权限，超出就停止。配置和提交发布通过应用身份完成，不需要为这些步骤申请用户管理权限。

`application:application:patch` 可以管理应用自身配置，包括通过原始 API 修改应用权限。该定制版限制默认创建清单和用户 OAuth 请求，但并不拦截所有手工原始 API 写入；使用时仍应按技能核验最终权限。

用户权限仅有：

| 权限 |
| --- |
| `docs:document.content:read` |
| `docs:document.media:download` |
| `docx:document:readonly` |
| `drive:drive:readonly` |
| `offline_access` |
| `space:document:retrieve` |
| `wiki:wiki:readonly` |

只有需要访问员工文档时，才由该员工授权这七项；管理权限始终放在应用身份。CLI 会拒绝用户 OAuth 的其他 scope 及 `--domain` / `--recommend` / `--exclude`。

## Windows / Linux 不使用 Agent 时

先用一个未占用的 profile 名称创建应用：

```text
lark-cli config init --new --brand feishu --name my-doc-import-001
```

确认成功后，记下实际 App ID。将下面的 `APP_ID` 替换成它，并保持 profile 为本次实际名称；在本套件目录执行：

```text
lark-cli api PATCH /open-apis/application/v7/applications/APP_ID/config --profile my-doc-import-001 --as bot --data @templates/config.json --dry-run
lark-cli api PATCH /open-apis/application/v7/applications/APP_ID/config --profile my-doc-import-001 --as bot --data @templates/config.json
lark-cli api POST /open-apis/application/v7/applications/APP_ID/publish --profile my-doc-import-001 --as bot --data @templates/publish.json --dry-run
lark-cli api POST /open-apis/application/v7/applications/APP_ID/publish --profile my-doc-import-001 --as bot --data @templates/publish.json
```

应用身份只能管理这个应用自身。提交成功不等于审批通过，须完成企业要求的审核。版本和范围查询 API 需要默认清单之外的读取权限，因此最终状态可能仍需在开发者后台确认；不要为此自动追加权限。

App Secret 默认不会在终端显示。清理绑定前，在用户自己的私密目录运行（输出文件必须不存在）：

```text
feishu-credential-export --app-id APP_ID --profile my-doc-import-001 --output my-app-credentials.json
```

工具核对 App ID 和 profile，将完整的 App ID、App Secret 和默认重定向 URL 写入新文件，终端仅显示文件位置。若回调由用户明确改为其他地址，增加 `--redirect-url <实际地址>`。在 macOS/Linux 文件权限为仅所有者可读写；Windows 文件继承所在目录的访问控制，应选择仅当前用户可访问的目录。不要把实际凭据文件放入分享包、公共仓库或普通日志。

技能要求任务完成时在用户的私密会话中交付三项；若 Agent 环境无法安全地在答复正文直接展示 Secret，会给出受限文件链接并说明，而不会用掩码冒充完整密钥。

任务结束时只清理本次 profile 和对应本地凭据，保留线上应用；具体步骤见主技能的 `references/lark-cli-create-and-publish.md`。若仍在等待企业审批，任务尚未完成，不要提前破坏继续工作所需的状态。

## 手动安装与恢复

1. Windows / Linux 从 `bin/` 中取与系统和处理器匹配的 `lark-cli` 和 `feishu-credential-export` 两个程序，放入自己的工具目录。Linux 必要时执行 `chmod +x lark-cli feishu-credential-export`。
2. 把 `skills/feishu-enterprise-app-setup` 和 `skills/lark-shared` 复制到 Agent 的实际技能目录。已有文件先备份。
3. 将工具目录加入 PATH，或使用 CLI 完整路径。执行上面的版本和 dry-run 检查。

恢复原版时，使用安装脚本保存的备份，或通过原安装方式重新安装官方 CLI。官方升级/npm 重装可能覆盖定制版，升级后必须重新核对版本与权限预览。

## 校验、源码与边界

- 仅核验包：Windows 用 `.\install.ps1 -VerifyOnly`，macOS/Linux 用 `sh ./install.sh --verify-only`。
- `SHA256SUMS` 校验包内文件；另附 ZIP 的 SHA256 文件，便于核对传输是否完整。校验和不能替代来源可信度判断。
- 可选的 `check-cli.mjs` 需要 Node.js，用于更完整的本地权限拦截检查：`node check-cli.mjs /path/to/lark-cli`。
- `BUILD-INFO.json`、`BINARIES.json` 记录版本、构建参数和目标；`VALIDATION.md` 说明实测范围。
- `src/lark-cli/` 包含完整定制 CLI 源码；`src/credential-export/` 包含导出工具源码。源码重建步骤见 `src/README.md`。Go 依赖未打包，本机缓存缺失时需要联网下载。
- `skills/feishu-enterprise-app-setup/assets/lark-cli-v1.0.100-docs7.patch` 是针对官方精确基线的源码补丁，可选择使用官方源码与补丁重建。
- `LICENSE` 与 `third-party-licenses/` 保留开源许可。
- 本包不包含任何人的 App ID/Secret、token、浏览器会话、授权二维码或 CLI 配置。
