# 七项用户权限加应用管理权的定制 CLI

本技能使用基于官方 `larksuite/cli v1.0.100` 的 `1.0.100-docs7.2` 定制版。创建流程按照官方 SDK 协议传入 `addons.preset=false`，tenant 权限只有 `application:application:patch`，user 权限固定为技能的七项。机器人能力是最小模板的基础能力，不代表开通了机器人业务权限。

## 使用前核验

```text
lark-cli --version
lark-cli config init --new --dry-run
```

版本必须含 `docs7`；预览必须满足 `dry_run=true`、`permission_preset=docs7-managed`、`create_only=true`、`addons.preset=false`、tenant 只有 `application:application:patch`、user 集合恰好是七项。预览本身不联网、不创建应用、不写凭据。原版 CLI 没有本预览能力；不能仅因可执行文件名相同就继续创建。

## 实际创建

```text
lark-cli config init --new --brand feishu --name <本次唯一profile>
```

使用 CLI 原样输出的链接和二维码。浏览器负责用户登录、选择目标企业和创建确认；提交前核验七项 user 清单及 tenant 的 application:application:patch。超出这七加一清单就停止，不批准、不以“稍后删除”替代全程最小权限。

定制版沿用服务端返回的 SDK 确认入口，不强制改成默认 `/page/cli`。缺少可验证的 SDK 链接时失败，不回退到默认模板。SDK 页可能忽略未知 scope；创建后仍需通过 CLI 无额外 scope 的查询和后台确认七项 user 及一项 tenant 均已开通。

`addons` 只支持权限、事件和交互回调，不直接携带 OAuth 回调或员工范围。创建后使用 dry-run 中的 `configuration_defaults` 和 `publication_defaults`，按 [CLI 流程](lark-cli-create-and-publish.md) 以应用身份调用配置与发布 API。管理权限必须生效且应用来源受支持；必要时才用后台处理初始审批或最终验收。`addons.callbacks` 不是 OAuth 回调。

## 用户 OAuth

只有需要实际读取员工资料或验证该链路时才发起。定制版裸 `auth login` 默认请求七项；技能仍使用明确的 `--scope`，便于审阅。额外 scope、`--domain`、`--recommend`、`--exclude` 会被拒绝。

`--device-code` 只能恢复有本机请求清单记录的授权。返回 token 若包含额外 scope 或缺少 scope 声明，定制版拒绝保存；不要隐瞒、不改写 token 声明，不放宽七项限制来完成测试。此检查不能撤销飞书服务端已经存在的历史授权。

## 在其他系统重建

如果使用包含完整源码的全平台套件，优先查看套件根目录的 `src/lark-cli/` 和 `src/README.md`，可直接从源码快照构建（Go 依赖未缓存时仍需联网）。若只安装了技能或使用不含源码快照的小包，则按下方方式从官方源码和补丁重建。

需要 Git 与 Go 1.23+。本改动没有操作系统专属的权限逻辑。使用随技能保存的 [源码补丁](../assets/lark-cli-v1.0.100-docs7.patch)；在新建目录中签出官方标签及精确提交，应用补丁后运行包测试和构建：

```text
git clone --branch v1.0.100 --depth 1 https://github.com/larksuite/cli.git lark-cli-docs7
cd lark-cli-docs7
git rev-parse HEAD
git apply <本技能目录>/assets/lark-cli-v1.0.100-docs7.patch
go test ./internal/auth ./cmd/auth ./cmd/config
go build -trimpath -ldflags "-s -w -X github.com/larksuite/cli/internal/build.Version=1.0.100-docs7.2 -X github.com/larksuite/cli/internal/build.PermissionPreset=docs7-managed" -o <输出文件> .
```

凭据导出工具源代码位于 [assets/credential-export](../assets/credential-export/main.go)。重建时将该目录复制到上述官方源码目录内的 `.tmp/export-managed-app/`，再在官方源码根目录运行 `go test ./.tmp/export-managed-app` 和 `go build -o <导出工具输出文件> ./.tmp/export-managed-app`；这使其能使用官方项目的内部凭据存储代码。与 CLI 一样按目标平台设置可执行文件名。

基线必须是 `ac22a60a23d66c37d43a7ec6d9e99b99a4320356`。输出文件扩展名和安装位置按实际系统选择，先备份原可执行文件，再替换当前 `lark-cli` 启动器实际调用的二进制。不要假设所有系统使用相同 npm 路径。

普通 `go build` 不带上述 linker flag 会保留原版权限行为，不能用于本技能。官方升级或 npm 重装可能覆盖定制版；升级后必须重新检查版本和 dry-run，不能静默使用默认创建模板。

## 验收与清理

将本地模拟测试与真实租户验收分开报告。本地测试检查请求、URL 参数和拒绝行为；不证明平台已接受权限或已发布。真实验收还需确认应用类型、最终七项用户权限、应用身份仅有管理权限、固定回调、已发布版本和所有员工范围。

尚未完成真实验收时，明确标记待办。恢复验证时重新发起注册，不能复用过期链接。

CLI 默认掩码 Secret。清理前使用套件中的 `feishu-credential-export` 将本次 App ID、完整 App Secret 与重定向 URL 导出到新的私密文件；该工具不向终端打印密钥。按 [主流程](lark-cli-create-and-publish.md) 向用户交付三项后，移除本次本机绑定、相应本地凭据和临时授权状态；保留线上应用、其他 profile 及交付文件。

## 官方依据

- [Go SDK 的最小模板与自定义权限](https://github.com/larksuite/oapi-sdk-go/tree/v3.13.0/scene/registration)
- [SDK addons 的含义和限制](https://github.com/larksuite/node-sdk#custom-scopeseventscallbacks-addons-and-updating-an-existing-app)
