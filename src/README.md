# 源码与重建

`lark-cli/` 包含本套件定制版的完整 CLI 源码快照，基于官方 `v1.0.100`，对应本地修改提交 `2948de1b8331f041cb1205f99b097b5aaa551f73`。快照不包含 Git 历史、临时文件或账号数据。

`credential-export/` 包含凭据导出工具源码和测试。两份源码均可跨平台构建。Go 第三方依赖没有打包成 `vendor` 目录；本机缓存缺少依赖时，Go 需要联网下载。构建需要 Go 1.23 或更高版本。

## 重建 CLI

在套件根目录的终端中运行，输出文件名可按实际系统调整：

```text
cd src/lark-cli
go test -mod=readonly -buildvcs=false -ldflags "-X github.com/larksuite/cli/internal/build.PermissionPreset=docs7-managed" -run "^TestDocument" ./internal/auth ./cmd/auth ./cmd/config
go build -mod=readonly -buildvcs=false -trimpath -ldflags "-s -w -X github.com/larksuite/cli/internal/build.Version=1.0.100-docs7.2 -X github.com/larksuite/cli/internal/build.PermissionPreset=docs7-managed" -o lark-cli-rebuilt .
```

Windows 可将输出文件名改为 `lark-cli-rebuilt.exe`。构建后运行其 `--version` 和 `config init --new --dry-run`，确认 `docs7-managed`、七项用户权限及一项应用身份权限。普通 `go build` 不带上述 linker flag 会保留官方默认权限行为，不能作为本套件定制版使用。

## 重建凭据导出工具

导出工具依赖 CLI 的内部包，因此需要先把它的源码复制到 `lark-cli/` 内的临时目录。当前位置仍为 `src/lark-cli`。

macOS / Linux：

```sh
mkdir -p .tmp/export-managed-app
cp ../credential-export/*.go .tmp/export-managed-app/
go test -mod=readonly -buildvcs=false ./.tmp/export-managed-app
go build -mod=readonly -buildvcs=false -trimpath -o feishu-credential-export-rebuilt ./.tmp/export-managed-app
```

Windows PowerShell：

```powershell
New-Item -ItemType Directory -Force .tmp/export-managed-app | Out-Null
Copy-Item ../credential-export/*.go .tmp/export-managed-app/
go test -mod=readonly -buildvcs=false ./.tmp/export-managed-app
go build -mod=readonly -buildvcs=false -trimpath -o feishu-credential-export-rebuilt.exe ./.tmp/export-managed-app
```

这些命令只构建程序，不会创建飞书应用或申请授权。生成的文件不会自动覆盖套件已有程序；如要替换已安装版本，先备份并核验新程序。
