---
name: feishu-enterprise-app-setup
description: 以 lark-shared 和定制 lark-cli 创建配置飞书企业自建应用，默认七项文档用户权限加一项应用身份管理权限、固定 OAuth 回调、发布给所有员工，任务结束清理本机绑定；浏览器仅处理必要确认与验收，不用于妙搭应用。
---

# 飞书企业自建应用：CLI 创建、配置与发布

以 `lark-shared` 和 `docs7-managed` 定制版 `lark-cli` 完成创建、配置与提交发布。浏览器只处理登录、创建确认、接口不支持的步骤及必要的最终验收。`lark-cli apps +create` 创建妙搭应用，不用于本流程。

## 固定默认值

除非用户明确修改，直接使用以下配置，不重复询问：

- 用户身份权限恰好七项：`docs:document.content:read`、`docs:document.media:download`、`docx:document:readonly`、`drive:drive:readonly`、`offline_access`、`space:document:retrieve`、`wiki:wiki:readonly`。
- 应用身份权限恰好一项：`application:application:patch`。它用于应用自身配置和发布，不放进用户 OAuth 清单。
- OAuth 重定向 URL：`http://127.0.0.1:4366/oauth/callback`。
- 可用范围：目标企业的 **所有员工**；完成正式发布与启用，并据实际证据报告状态。
- 每次任务真正结束时，清理本次使用的本机 CLI 绑定及对应本地凭据；保留线上应用和用户要求交付的密钥文件。

用户最新明确要求优先于以上默认。这里的权限总计是七项 user 加一项 tenant；不得再申请 `application:application:self_manage`、版本读取、通讯录或任何其他权限，也不得借用残留更大权限的旧 token。全员可用不等于获得全员文档或通讯录，员工资料仍使用员工自己的 user_access_token 和资源权限。

## 执行前

1. 读取当前环境的 `lark-shared/SKILL.md` 及本次触发的配置、身份权限和输出契约参考。通用技能中的 `--domain all`、`--recommend` 不适用于本流程。
2. 从项目、任务上下文确认目标企业、应用用途和已有配置；用户要求新建时新建，要求复用时按 App ID 精确定位。只询问确实无法推断的必要信息。
3. 阅读 [CLI 流程](references/lark-cli-create-and-publish.md) 和 [定制版说明](references/docs7-cli.md)。运行版本及 dry-run 核验，不能仅凭可执行文件名认定已安装正确版本。
4. 只要求编写技能或检查方案时，不创建或发布真实应用。用户已要求创建、配置、发布及验证时，按已有授权完成，不逐个按钮重复征求同意。

## 创建与权限

1. `lark-cli --version` 应为 `1.0.100-docs7.2` 或经核验的兼容版本；`config init --new --dry-run` 的 `permission_preset` 必须是 `docs7-managed`，`preset:false`、`create_only:true`，user 为七项，tenant 只有 `application:application:patch`。
2. 使用唯一命名 profile 发起 `config init --new --brand feishu --name <profile>`，按 `lark-shared` 展示原样链接和二维码。在创建确认页核对企业、应用类型及七项 user 加一项 tenant 后再提交。额外权限不能先批准再删除；缺项不能悄悄替换为其他 scope。
3. 官方 SDK 最小模板保留机器人能力，但这不等于有机器人业务权限；不添加发消息、事件或长连接权限。
4. 取得 App ID 后，确认它与本次 profile、品牌和企业一致。Secret 通过安全输入及 CLI 凭据存储处理，不在命令行参数、日志、前端或仓库泄露。用户要求交付 Secret 时按其要求安全交付。
5. 用无需额外 scope 的 CLI 权限查询核验实际开通清单。平台可能忽略不认识的 scope，不能把请求清单等同实际开通结果。若管理权限尚待初始发布/审批生效，等待正常流程，不追加用户身份管理权限。

## CLI 配置与提交发布

1. 从 dry-run 输出保存 `configuration_defaults` 和 `publication_defaults` 两个 JSON 请求体，按 [CLI 流程](references/lark-cli-create-and-publish.md) 先预览再调用正式 API。操作使用目标应用自己的 `--as bot --profile <profile>`。
2. 配置请求添加固定回调，开启允许刷新，并设置 `visibility.is_visible_to_all=true`；保留其他已存在回调，不改通讯录数据范围。目标 App ID 必须等于 profile 绑定的 App ID。
3. 配置成功后提交发布，保存返回的版本 ID/版本号。不要把提交成功当作审批通过或已上线，也不要在结果不确定时重复提交制造多个版本。
4. 官方配置/发布接口限定支持的自建应用来源。若 SDK 创建的应用被接口拒绝，记录真实错误，在已授权范围内使用必要后台步骤；不声称 CLI 全链路已经通过。
5. 查询最终版本、范围或启用状态若需要其他 scope，不申请；使用已登录后台或用户确认取得证据。企业审批不能跳过，不能修改企业全局审核策略。
6. 已启用不反复切换；仅在本次授权范围内处理目标应用。不更改全员工作台布局，不把“未置顶”当成“不可用”。

## 验证与交付

按 [发布验收](references/publish-and-verify.md) 区分：本地测试通过、真实应用创建、权限生效、回调配置成功、发布已提交、审批通过、实际全员可用。证据来源是 API、后台直读还是用户确认，应准确说明。

创建和发布本身使用应用身份，不为例行验收发起用户 OAuth。只有实际读取员工资料或用户明确要求该链路测试时，才请求原七项用户权限；已有有效授权则复用。详见 [权限与 OAuth](references/permissions-and-oauth.md)。不把开发者账号成功当成普通员工已实测，不进行全量文档导出。

任务完成时，在最终答复中明确列出该应用对应的 **App ID、完整 App Secret、重定向 URL**，并说明七项 user 加一项 tenant、真实版本与审批/启用/可用范围状态及具体待办。优先通过当前环境支持的私密凭据交付机制在最终答复直接展示，不将 Secret 写入终端、普通工具输出、日志、仓库或分享包。若无法安全地在答复正文中取得并展示完整 Secret，则交付包含完整三项的受限本地文件链接，明确说明正文未显示，不能用掩码冒充完整 Secret。若用户对本任务指定其他交付方式，从其要求。缺证据的项标为待验证。

## 结束时清理

完成验证及凭据交付后，按 [CLI 流程中的清理步骤](references/lark-cli-create-and-publish.md) 移除本次准确 App ID/profile 及对应本地凭据并复核。本技能默认使用本次任务专用的临时 profile 并在结束时清理。清理任务开始前已存在的连接须有用户明确授权。取消或无法继续时也清理本次临时绑定；正在等待用户创建确认或管理员审批而仍需继续的任务尚未结束，可暂留必要状态。

保留无关 profile、线上应用及其七加一权限配置、用户要求交付的密钥文件。CLI 自动改选其他默认 profile 时如实报告，不顺带删除。清理失败则说明具体范围和原因。清理后下次可能需要重新绑定或授权。
