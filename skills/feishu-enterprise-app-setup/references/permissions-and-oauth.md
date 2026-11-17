# 文档读取权限与 OAuth 核对

维护日期：2026-11-17。用于员工授权后读取云盘、我的文档库、企业知识库并导入文档。调用前查当前官方接口文档和应用后台；下列标识不代表已经在某个租户开通。

## 四层范围不要混淆

| 层次 | 管什么 | 本技能的处理 |
| --- | --- | --- |
| 应用可用范围 | 谁能使用应用 | 按用户要求；未指定时默认目标企业的所有员工。 |
| API 权限 / 数据范围 | 应用能申请或执行哪些 API 能力 | 默认开通下列七项用户权限及应用身份的 `application:application:patch`；不附加其他权限。用户以后明确修改要求时从其要求。 |
| 用户 OAuth 授权 | 这位员工给这个应用哪些访问能力 | 使用其本人授权的 user_access_token，核对实际授予的 scope。 |
| 文档资源权限 | 该身份对具体目录、文档、素材是否有权 | 遵循资源所有者设置；不能靠扩大应用范围绕过。 |

“我的文档库”必须用用户身份定位。给机器人添加一个知识库的读取资格，不会让机器人拥有每位员工的个人文档库。

## 权限矩阵：按接口选择

这里给出接口和权限核对信息。除非用户明确指定其他权限，本技能的用户授权仅开通七项；应用身份另开通 `application:application:patch` 用于管理自身，即使某些接口支持任选其一，也不因此删减默认七项。不要因为接口文档列出其他候选权限而自动申请；七项无法满足时说明限制。

| 能力 / 接口 | 身份与权限核对起点 | 是否默认需要 |
| --- | --- | --- |
| 云盘目录：`GET /open-apis/drive/v1/files` | 用户身份；核对 `space:document:retrieve` 和 `drive:drive:readonly`。接口通常满足其一即可调用，但两项均属于本技能默认权限。 | 需要云盘入口时必需。 |
| 列知识空间：`GET /open-apis/wiki/v2/spaces` | 用户身份；默认使用七项中的 `wiki:wiki:readonly`，该接口也支持其他替代权限。 | 需要企业知识库入口时核对，不自动追加权限。 |
| 解析我的文档库：`GET /open-apis/wiki/v2/spaces/my_library`；读取指定空间信息 | 用户身份；核对该“获取知识空间信息”接口的当前权限。可从兼容的 `wiki:wiki:readonly` 或当前细粒度空间读取权限查起；不要把“列空间”和“读空间详情”当成同一个权限项。 | 需要我的文档库时必需。 |
| 列子节点：`GET /open-apis/wiki/v2/spaces/{space_id}/nodes` | 用户身份；核对七项中的 `wiki:wiki:readonly` 是否为当前接口支持的只读替代项。 | 默认不追加 `wiki:node:retrieve`；七项不满足时说明限制。 |
| 新版文档元数据、正文块、纯文本：`/docx/v1/documents/{id}`、`/blocks`、`/raw_content` | 用户身份；核对 `docx:document:readonly` 及接口当前提供的细粒度只读项。 | 新版文档导入必需；覆盖元数据与分页块读取。 |
| 官方 Markdown 内容：`GET /open-apis/docs/v1/content` | 核对 `docs:document.content:read`，同时核实当前企业自建应用能否开通、是否支持所需 doc_type / content_type。 | 项目调用该接口时检查。不可用时保留已有“正文块转 Markdown”方案，不能虚报已授权。 |
| 图片 / 文档内附件：`GET /open-apis/drive/v1/medias/{file_token}/download` | 核对 `docs:document.media:download` 及该接口实际支持的只读替代项；同时检查用户对素材的下载权。 | 需要完整素材导入时必需。 |
| 云盘原始文件下载：`GET /open-apis/drive/v1/files/{file_token}/download` | 核对七项中的 `drive:drive:readonly` 是否为当前接口支持的替代项。 | 默认不追加 `drive:file:download`；七项不满足时说明限制。 |
| 长期保持连接 | `offline_access`，核对是否需要后台开通及授权端点当前要求。 | 默认七项之一；刷新链路是否可用需单独验证。 |
| 电子表格 / 多维表格 / 白板等内容 | 对应产品的读取接口及只读权限，逐个核对。 | 只有业务要求获取其正文 / 数据时再加；列出名称不等于已支持导入。 |

不要仅凭 `drive:drive.metadata:readonly` 宣称目录遍历已授权。目录清单、元数据、文档正文、素材下载是不同能力。错误返回中的多个可接受 scope 可能是“或”关系。

### 文档导入七项权限

除非用户明确指定其他权限，否则以下七项均为默认值，应全部按 **user** 身份申请并逐项核验，不能因其中部分权限对某接口互为替代而自行删除：

```text
docs:document.content:read
docs:document.media:download
docx:document:readonly
drive:drive:readonly
offline_access
space:document:retrieve
wiki:wiki:readonly
```

这是默认且唯一的用户身份权限集合。应用身份仅增加 `application:application:patch` 用于配置自身与提交发布，不放入用户 OAuth；不申请其他权限。用户日后明确修改范围时按其要求处理。`offline_access`、允许刷新开关、实际获得 refresh token 和刷新成功是不同检查点。配置和发布使用已经允许的应用身份管理权限；需要其他读取权限的验收步骤仍使用必要后台核验。

开发者后台支持 JSON 批量导入时，按当前界面校验以下格式；应用身份页签应只有 `application:application:patch`。批量导入是增量操作，不能证明旧权限已移除：

```json
{
  "scopes": {
    "tenant": ["application:application:patch"],
    "user": [
      "docs:document.content:read",
      "docs:document.media:download",
      "docx:document:readonly",
      "drive:drive:readonly",
      "offline_access",
      "space:document:retrieve",
      "wiki:wiki:readonly"
    ]
  }
}
```

### 从项目已有配置生成最终清单

1. 搜索项目实际调用的 API 路径和 OAuth scope 配置；只读 `.env.example` 等示例，不把真实密钥文件打印出来。
2. 输出一张当前任务的小表：实际接口 → 选定 scope → 身份类型 → 当前平台依据 → 开通状态。
3. 常见旧配置只有 Docx、Markdown、素材和 Wiki 读取；新增云盘时要补 Drive 目录读取能力，不能只改页面入口。
4. 控制台有“批量导入权限”时，可按其当前格式生成 JSON。用户文档 scope 放到用户身份列表；tenant 只放 `application:application:patch`，不混用身份。以导入后的实际权限列表核对结果。
5. 权限项不存在时先核对企业 / 中国版飞书 / 身份页签 / 当前 API 支持，不通过猜测同名写权限替换。
6. 记录默认七项或用户明确指定的最终 scope 集合，再同步到项目授权请求；本表中的额外候选项不自动追加。

## OAuth 配置与 token 生命周期

- 核对主页、已注册回调、应用 App ID 及环境的一致性。
- 优先沿用项目当前有效链路；如新增或迁移 OAuth，再核对官方现行授权端点和 token 接口，不混搭旧版、新版参数。
- 现行核对入口：`https://accounts.feishu.cn/open-apis/authen/v1/authorize`；新接入的换取和刷新凭证核对 `POST https://accounts.feishu.cn/oauth/v3/token`，旧 v2 接口已标记不推荐。既有项目迁移时核对当前官方文档和实际兼容性。
- 授权请求带本次实际需要的 scope、精确 redirect_uri 和用于关联会话的 state；按项目和当前规范处理 PKCE。换码由可信服务端持有 App Secret，不把它交给网页。
- OAuth 完成后核对用户身份、实际授予 scope、有效期、是否存在 refresh_token。校验 state，授权码按一次性凭证处理。
- 已有同一应用、同一用户的有效授权时优先复用；仅在需要用户身份且没有有效授权时发起授权。刷新仅在需要且有对应用户授权时执行。按响应安全保存新的 access token、过期时间以及返回的新 refresh token；refresh token 一次性使用，成功后旧值失效。如果没有新 refresh token，不用空值覆盖旧值；按当前文档处理失效或不确定结果。
- 对自行维护 OAuth 的业务服务，飞书 v3 刷新接口支持用 `scope` 将新 access token 缩减到已授权权限的子集，可能无需重新让用户同意；它不撤销用户的原始授权，原 access token 在到期前仍可能有效。刷新会轮换 refresh token，必须可靠保存新值。不要为了缩减 scope 手工改写 `lark-cli` 内部凭据或假设 CLI 支持该操作；需要立即撤销旧 token 时按平台支持的撤销流程处理，并说明可能需要重新授权。
- 新开权限不会自动扩展老 token。需要重新授权 / 增量授权；刷新 token 也不能凭空取得原来没授权的权限。
- 不把 tenant_access_token 当作员工登录凭证，不以开发者本人成功代替所有员工可用。

## 三类文档来源的只读验证

仅在已有适用用户授权或用户明确要求用户侧连通性测试时执行；单纯创建、配置和发布应用不为了测试强制触发用户 OAuth。

### 云盘

`GET /open-apis/drive/v1/files?folder_token={已授权的文件夹token}&page_size=1`

验证指定文件夹的一页；若用户明确选择云盘根目录，可省略 folder_token。普通文件夹分页看 has_more / next_page_token；根目录的分页、快捷方式覆盖限制需依当前文档记录。不能把一页当全量。

### 我的文档库

1. 用户 token 调用 `GET /open-apis/wiki/v2/spaces/my_library`。
2. 取 `data.space.space_id`，用真实 ID 请求 `GET /open-apis/wiki/v2/spaces/{space_id}/nodes?page_size=1`。
3. node_token 用于子页面遍历，obj_token 与 obj_type 用于实际内容读取。
4. 普通知识空间列表不包含 my_library。直接把 my_library 拼入 nodes 路径、得到空清单后宣称无文档，都不作为正确验证。

### 企业知识库、正文与素材

- 用该用户列一页可访问空间；选择本次允许测试的空间读一页节点。每个来源记录“成功且有样例 / 成功但为空 / 无权限 / 未验证”。
- 从实际返回的一篇 docx 取 obj_token，读取元数据及一页内容块。页面本身可能同时有正文与子页面，不能只把叶子节点当文档。
- 如果本次需要素材，从这篇文档真实返回的素材 token 测试一个允许下载的资源。不探测或枚举范围外的资源。
- 初始联通验证不需要自动递归导出整个企业。

## 官方核对入口

以下是读取原始资料的链接，不是必须照搬的第三方工作流：

- [飞书应用登录与业务授权的区别](https://open.feishu.cn/document/client-docs/build-login-free-system-)
- [获取用户访问凭证](https://open.feishu.cn/document/authentication-management/access-token/get-user-access-token)
- [刷新用户访问凭证](https://open.feishu.cn/document/authentication-management/access-token/refresh-user-access-token)
- [获取文件夹中的文件清单](https://open.feishu.cn/document/server-docs/docs/drive-v1/file/list)
- [获取知识空间信息](https://open.feishu.cn/document/server-docs/docs/wiki-v2/space/get)
- [获取知识空间子节点列表](https://open.feishu.cn/document/server-docs/docs/wiki-v2/space-node/list)
- [Lark 官方 CLI：空间列表与权限替代关系](https://github.com/larksuite/cli/blob/main/shortcuts/wiki/wiki_space_list.go)
- [Lark 官方 CLI：my_library 先解析再列节点](https://github.com/larksuite/cli/blob/main/shortcuts/wiki/wiki_node_list.go)
- [Lark 官方 SDK：Markdown 内容接口的请求定义](https://github.com/larksuite/oapi-sdk-python/blob/v2_main/lark_oapi/api/docs/v1/model/get_content_request.py)
- [Lark 官方 SDK：新版文档块列表](https://github.com/larksuite/oapi-sdk-python/blob/v2_main/lark_oapi/api/docx/v1/model/list_document_block_request.py)

官网页面无法直接抽取正文时，不能把页面标题当成权限已核实的证据；查浏览器中的官方文档或当前控制台。
