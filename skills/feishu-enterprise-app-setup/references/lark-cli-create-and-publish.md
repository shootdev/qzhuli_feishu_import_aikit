# 七项用户权限加一项应用管理权限的 CLI 流程

## 使用前

读取 `lark-shared` 的身份权限、配置和输出契约参考。只允许七项 document 用户权限及 tenant 的 `application:application:patch`；用户 OAuth 仍不允许任何第八项。

按 [定制版说明](docs7-cli.md) 核对版本与 `lark-cli config init --new --dry-run`。保存预览里的 `configuration_defaults` 为 `config.json`、`publication_defaults` 为 `publish.json`。它们不含密钥；文件参数使用当前工作目录内的相对路径。

## 创建并绑定

```text
lark-cli config init --new --brand feishu --name <本次唯一profile>
```

新建流程强制 createOnly。创建页确认目标企业及七项 user 加一项 tenant 后再提交；不能把旧应用或妙搭应用当作本次目标。只处理本次子进程中误继承的 Hermes/OpenClaw 环境，不修改用户级环境或其他 Agent 绑定。取得 App ID 后，所有后续操作显式指定本次 profile。

```text
lark-cli auth scopes --profile <profile> --json
lark-cli api GET /open-apis/application/v6/scopes --profile <profile> --as bot
```

按当前文档，应用权限查询不需要增加 scope；它仍需要已保存的应用凭据。核验返回的身份与权限清单，不能只核验请求体。缺项或未生效按平台要求处理；不增加范围外的权限。

## 配置回调和全员范围

```text
lark-cli api PATCH /open-apis/application/v7/applications/<本次AppID>/config --profile <profile> --as bot --data @config.json --dry-run
lark-cli api PATCH /open-apis/application/v7/applications/<本次AppID>/config --profile <profile> --as bot --data @config.json
```

配置正文：

```json
{
  "security": {
    "add": {"redirect_urls": ["http://127.0.0.1:4366/oauth/callback"]},
    "allow_refresh_token": true
  },
  "visibility": {"is_visible_to_all": true}
}
```

`application:application:patch` 必须作为应用身份权限生效；tenant_access_token 只能修改自身。回调地址修改可即时生效，范围等配置需按平台规则发布后生效。用户身份管理权限、通讯录权限均不需要添加。

## 提交发布

```text
lark-cli api POST /open-apis/application/v7/applications/<本次AppID>/publish --profile <profile> --as bot --data @publish.json --dry-run
lark-cli api POST /open-apis/application/v7/applications/<本次AppID>/publish --profile <profile> --as bot --data @publish.json
```

使用预览的发布说明及 `bot` 默认能力，符合 SDK 最小模板；若用户明确修改了应用能力，先核对真实能力后按要求设置。版本号可省略，由接口处理已有待发布版本或创建版本，避免猜测版本号。记录返回版本标识；仅成功响应不能证明审核通过。正在审核、应用来源不支持或网络结果不确定时，先查明现状，不反复提交。

该 API 与配置 API 都要求 tenant 的 `application:application:patch`。官方文档限定开发者后台创建的自建应用，SDK 一键创建来源是否被接受必须实测。

## 最终验收的权限边界

- 当前版本详情/列表 API 要求 `application:application:self_manage` 或 `application:application.app_version:readonly`，都不在本次授权中。
- 企业可用范围读取接口要求 `admin:app.info:readonly`，也不在本次授权中。
- 不为验收追加这些 scope。用必要后台页面或用户确认核实正式版本、审批、全员范围、启用及回调；报告证据来源。
- 创建时的一次确认、企业审批、用户文档 OAuth 是不同环节。配置和发布用应用身份，不要求用户重复 OAuth。

## 确需用户 OAuth 时

```text
lark-cli auth login --profile <profile> --scope "docs:document.content:read docs:document.media:download docx:document:readonly drive:drive:readonly offline_access space:document:retrieve wiki:wiki:readonly" --no-wait --json
```

仅在需要访问员工资料、已有授权不可复用时执行。定制版拒绝额外 user scope，管理权限也不能放入这条命令。返回 token 的 scope 为空或超出七项时拒绝保存，报告实际情况，不改写声明。遵循 `lark-shared` 的链接、二维码和恢复授权规则。

## 交付 App ID、App Secret 与重定向 URL

任务完成时向用户交付本次实际 App ID、完整 App Secret 和重定向 URL。CLI 默认只显示掩码；在清理绑定前，使用套件提供的 `feishu-credential-export --app-id <本次AppID> --profile <本次profile> --output <当前目录下的新文件>` 安全导出。若使用隔离配置目录，导出时沿用相同的 `LARKSUITE_CLI_CONFIG_DIR`。该命令验证 App ID 与 profile、拒绝覆盖文件，并将三项写入文件，不向终端打印 Secret。

在支持私密凭据交付的环境里，按用户要求在最终答复直接展示完整 Secret；如果无法安全直接展示，则提供受限本地文件链接并明确说明。文件应放在用户自己的私密目录，不放入分享包、版本库或普通日志；用户另有要求时按其要求处理。确认凭据已妥善交付或安全保留后再清理本机绑定。

## 结束时定向清理

完成验证与凭据交付后：

1. 确认本次准确 App ID、profile 和当前 CLI 配置目录；确认密钥是否被其他配置共享。
2. 当前版本可用 `lark-cli config tenant-access-token remove --app-id <本次AppID>` 清理单独保存的 tenant token。
3. 多 profile 时用 `lark-cli profile remove <本次profile或AppID>`，核验本次连接及对应用户凭据、App Secret 已清理。它可能改选其他默认连接，应如实说明。
4. 唯一 profile 无法用 profile remove 删除时，仅在已确认配置只含本次目标且不会误删共享凭据的情况下使用 `lark-cli config remove`。若留下仅有空 apps 的配置文件而导致列表报错，核对实际文件路径和内容后只移除这个空文件。
5. 只清理本次授权记录、二维码和临时配置，保留其他应用、线上应用及其七加一配置、交付密钥文件；不批量清空共享缓存。核验实际绑定列表。

等待用户确认或企业审批而任务尚未完成时可暂留必要状态；结束时再清理。清理失败如实报告，不把本机清理当作线上撤销授权。

## 官方规范

- [修改应用配置信息](https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/application-v7/application-v7/application-config/patch.md)
- [提交发布自建应用](https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/application-v7/application-v7/application-publish/create.md)
- [获取版本详情](https://open.feishu.cn/document/server-docs/application-v6/application/get-2.md)
- [查询企业可用范围](https://open.feishu.cn/document/server-docs/application-v6/admin/obtain-the-app-availability-in-an-organization.md)
