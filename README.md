# Elastic UI

Elastic Defend 的 Windows 本机控制台。查看当前主机的防护状态、调查告警、扫描文件、管理隔离文件和信任应用。集中策略、规则管理与解除主机隔离在 Kibana 中完成。

本项目是独立客户端，不是 Elastic 官方产品，也不包含 Elastic Agent。使用前需要安装并注册 Elastic Agent / Elastic Defend，并具备 Kibana 和 Elasticsearch 的相应访问权限。

## 安装与更新

从 [GitHub Releases](https://github.com/Dreamend1ng/elastic-ui/releases/latest) 下载：

- `elastic-ui-windows-amd64-setup.exe`：安装包，提供桌面和开始菜单入口、升级与卸载。
- `elastic-ui-windows-amd64.exe`：独立程序。
- `SHA256SUMS`：下载文件的 SHA256 校验值。

支持 Windows 10 / 11 x64。安装器在缺少 WebView2 时调用微软安装程序下载运行时。程序及安装器需要管理员权限，当前发布未配置代码签名证书。

设置中的“关于 → 检查更新”读取本仓库的最新正式版。点击“更新并重启”后下载、校验安装包，再退出程序、原目录升级并重新启动。升级保留用户配置、凭据、恢复记录与审计日志；替换失败时保留原程序。卸载不删除用户数据，也不卸载 Elastic Agent。

“通知与托盘 → 开机自启”可设置当前用户登录时启动。通过计划任务以管理员权限运行，默认进入托盘。该选项初始关闭。

## 功能

- 自动绑定本机 Agent，显示 Agent / Endpoint 运行状态、版本和异常详情。
- 查看恶意软件、勒索、内存、行为防护、设备控制、防篡改和策略应用状态。
- 支持连接 Kibana / Elasticsearch，使用用户名密码或 API Key。
- 查看本机威胁历史、MITRE 信息、实际处置结果和完整文件路径。
- 使用可缩放、平移的进程图调查告警，查看完整命令行及进程元数据。
- 浏览进程、文件、网络、注册表和安全事件，筛选时间、类别与关键字。
- 发起路径扫描、右键扫描，查看扫描历史及关联威胁文件。
- 查看隔离记录，按隔离路径合并，读取本机文件存在状态和当前策略信任状态。
- 单个或批量找回隔离文件、单个或批量加白，显示逐项结果；跨页保留选择。
- 持久保存本客户端的恢复时间和 ZIP 保存位置，重新打开后仍可查看。
- 管理本机策略的信任应用；全局规则只读，支持批量校验、添加和删除。
- 创建、查看和重命名本机独立策略。
- 应急隔离当前主机，解除隔离由管理员在 Kibana 中操作。
- 可切换 Elastic 原生通知或 UI 通知，支持托盘、关闭到托盘和开机自启。
- 深浅主题平滑切换，尊重系统的减少动态效果设置。
- 导出诊断、清理本地缓存，保留本地审计。

## 隔离文件

文件找回使用 Elastic 官方 `get-file` 响应动作。客户端校验加密 ZIP 的内容及原始文件 SHA256 后保存，ZIP 密码为 `elastic`。文件保存在 ZIP 内，不会自动解压到原位置。

文件存在、ZIP 保存记录和信任规则分别显示。找回 ZIP 不会自动删除 Endpoint 隔离文件；“已配置信任”表示当前策略存在对应 SHA256 信任应用规则，不表示已经确认 Endpoint 应用成功，也不覆盖所有类型的检测例外。

隔离区依据本机成功隔离告警构建，尚不包含未上传到 Elasticsearch 的隔离文件。文件状态只检查本机隔离目录的元数据；权限不足显示无法读取。仅记录本客户端实施的 ZIP 保存结果。文件去重后的总数由 Elasticsearch cardinality 聚合计算，大数据量下可能近似；分页上限为 9900。

## 首次连接

填写 Kibana、Elasticsearch 地址和凭据。连接后自动匹配本机 Agent；按界面完成独立策略设置。

凭据保存在 Windows 凭据管理器。配置、运行日志、审计和恢复记录位于 `%APPDATA%\ElasticDefendConsole`。服务端账号或 API Key 的权限决定实际可访问范围，客户端的本机过滤不代替服务端权限控制。

## 构建与验证

需要 Go 1.26、Node.js 22+、Wails v2 和 Windows 构建环境。安装包构建还需要 NSIS。

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.1
cd frontend
npm ci
npm run check
npm test
cd ..
go test ./...
wails build
```

生成程序位于 `build/bin/elastic-ui.exe`。开发模式使用 `wails dev`。

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File packaging/build-release.ps1
```

发布安装包、独立程序和校验文件位于 `build/bin`。发布流程见 [docs/releasing.md](docs/releasing.md)。

## 限制

- Endpoint 响应动作和 API 可用性取决于 Elastic 版本、许可证与账号权限。
- 扫描告警按主机、执行时间、`event.action=demand` 和路径关联；重叠扫描可能共享结果。
- Windows 11 的传统右键入口位于“显示更多选项”。
- 手动启动要求 UAC；登录自启使用用户主动启用的管理员计划任务。
- 本机信任区目前提供信任应用，其他例外类型在 Kibana 管理。
- 程序只控制运行它的主机，不提供解除主机隔离入口。
