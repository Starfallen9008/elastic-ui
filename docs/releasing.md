# 发布

发布源固定为 `Starfallen9008/elastic-ui` 的 GitHub Releases。客户端只检查正式版，不安装草稿或预发布版。

1. 同时修改 `internal/update/update.go`、`wails.json` 和 `packaging/build-release.ps1` 中的版本号。
2. 运行 `go test ./...`、前端检查与测试。
3. 运行 `packaging/build-release.ps1 -Version 1.0.0`，按实际版本调整参数。脚本验证微软 WebView2 引导安装程序的 Authenticode 签名，并生成安装包和 SHA256 校验文件。
4. 验证安装、原目录升级和卸载。已有进程运行时安装器拒绝覆盖；应用内更新会先退出主程序。
5. 创建 `v1.0.0` 格式的 Git 标签和正式 Release，上传 `elastic-ui-windows-amd64-setup.exe`、`elastic-ui-windows-amd64.exe` 和 `SHA256SUMS`。
6. 校验上传文件的 SHA256，并检查 GitHub latest Release。

更新器只接受固定仓库下的 HTTPS Release 资产，下载完成后校验 SHA256，安装前再次校验。SHA256 校验不能替代发布账号安全和代码签名；当前项目未配置代码签名证书。

安装器保留上一版程序为 `elastic-ui.previous.exe`，替换新程序失败时尝试回退。配置和 Windows 凭据管理器中的凭据不在安装目录内，升级和卸载均保留。

自启是当前用户的交互式登录任务，启用时请求最高可用权限。升级仅刷新已存在的自启任务和右键注册，首次安装不会自动开启。
