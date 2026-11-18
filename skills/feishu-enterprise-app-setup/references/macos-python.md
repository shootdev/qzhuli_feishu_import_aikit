# macOS 用法

Intel Mac 和 Apple Silicon 使用同一个 Python 包。需要可用的 Python 3.8 或更高版本。解压后在套件根目录运行：

```sh
python3 feishu_app.py preview
python3 feishu_app.py create --output "$HOME/feishu-app-credentials.json"
```

预览时核对七项用户权限、应用身份的 `application:application:patch`、固定回调和全员范围。创建命令先输出飞书确认链接和临时二维码文件路径。将原样链接及二维码交给本次操作者，进程保持运行，等其选择正确企业并核实权限后确认。确认成功时，脚本立即将 App ID、完整 App Secret 和固定回调写入指定的新文件（权限 0600）。选择自己私密目录里未占用的文件名。

随后脚本用应用身份查验全部已开通权限，配置 `http://127.0.0.1:4366/oauth/callback`、允许刷新、全员可用，提交发布并返回版本号。需按企业审批流程确认最终上线状态。向用户交付凭据和发布结果。此实现将设备码和应用 token 保存在进程内，结束时释放，临时二维码目录自动清理。持久文件仅用于交付与记录发布结果。

若创建已成功而配置环节失败，使用同一个凭据文件继续：

```sh
python3 feishu_app.py verify --credentials "$HOME/feishu-app-credentials.json"
python3 feishu_app.py publish --credentials "$HOME/feishu-app-credentials.json"
```

同一交付路径的 `.publish.json` 文件用于防止重复提交。状态为 `submitted` 时检查其中的版本号及后台审批状态。状态为 `submission_unconfirmed` 时先核实飞书后台，只有确认没有提交成功才移走该记录并重试。

如确实需用员工身份授权，运行 `python3 feishu_app.py user-login --credentials <应用凭据文件> --output <新的私密用户凭据文件>`。它固定请求原七项用户权限。此文件包含用户 token，请交付给本次授权用户或其业务服务。

可直接将包里的 `skills/feishu-enterprise-app-setup`、`skills/lark-shared` 安装到 Agent 的技能目录，也可以运行 `sh ./install.sh`。从已安装技能执行时，将命令里的 `feishu_app.py` 换成主技能目录内的 `scripts/feishu_app.py`。脚本使用 Python 标准库和随包提供的纯 Python 二维码模块。
