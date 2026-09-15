# zerotier-webui

zerotier-one 本地自建控制器的 Web 管理面板，**单个可执行文件**即为完整服务：前端产物用 `go:embed` 内置，无需 PHP、Node、数据库或任何运行时依赖，拷贝到目标机器直接运行。

管理对象是 zerotier-one 内置控制器的本地 REST API（默认 `http://127.0.0.1:9993`），不是官方 my.zerotier.com（Central）。

## 特性

- 单文件交付：约 8 MB，Windows / Linux / macOS × amd64 / arm64 六个目标
- 网络管理：列表、创建、编辑（结构化表单 + 原始 JSON 双向同步）、删除
- 成员管理：授权、改名、备注、IP 分配、移除
- 配置字段尽量透传：网络 config 与成员对象整体往返，只剥离服务端只读字段，不裁剪可配置面
- 零凭据落盘：二进制不写文件、不存密钥；令牌只存在浏览器 localStorage，随请求头透传

## 快速开始

```bash
zerotier-webui-windows-amd64.exe            # 默认监听 127.0.0.1:9090 并打印访问地址
zerotier-webui-linux-amd64 --open          # 启动后自动打开浏览器
```

打开面板后进入「连接」页，填入控制器访问令牌即可。令牌取自控制器所在机器的 `authtoken.secret`：

| 平台 | 路径 |
| --- | --- |
| Windows | `%LOCALAPPDATA%\ZeroTier\authtoken.secret`（`C:\ProgramData\ZeroTier\One\` 下那份需管理员权限读取） |
| Linux | `/var/lib/zerotier-one/authtoken.secret` |
| macOS | `/var/db/zerotier-one/authtoken.secret` |

控制器不在本机时，在「连接」页填控制器地址（如 `http://10.0.0.7:9993`）；留空则沿用桥启动时的 `--url`。

## 命令行参数

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `--listen` | `127.0.0.1:9090` | 桥自身监听地址，可用 `ZT_LISTEN` 覆盖 |
| `--url` | `http://127.0.0.1:9993` | 控制器基址，可用 `ZT_URL` 覆盖 |
| `--token` | 空 | 兜底令牌，供 curl / 健康检查等无浏览器场景使用（`ZT_TOKEN`） |
| `--node-address` | 空 | 控制器节点地址（10 位 hex）；留空则自动取 `/status` 的 `address` |
| `--timeout` | `15s` | 上游请求超时 |
| `--open` | `false` | 启动后打开默认浏览器 |
| `--version` | — | 打印版本信息后退出 |

## 开发

```bash
pnpm -C web install
pnpm -C web dev       # Vite 开发服务器，/api 与 /_forge 代理到 127.0.0.1:9090
go run ./cmd/zerotier-webui
```

两个进程各自独立跑：Vite 提供热更新的前端，Go 提供 API 与控制器透传。

## 构建

```bash
pnpm -C web build                                   # 产出 web/dist/，供 go:embed 读取
go build -trimpath -ldflags "-s -w" -o bin/zerotier-webui ./cmd/zerotier-webui
pwsh -NoProfile -File scripts/build.ps1             # 交叉编译六个目标，版本取自 git describe
```

`web/dist/` 只保留一个占位文件入库，因此 `go build` 之前必须先 `pnpm -C web build`，否则二进制里没有前端产物（启动时会打印提示，访问 `/` 返回 503 与可执行的修复指引）。

## 架构

```
cmd/zerotier-webui/   入口：参数解析、路由登记表、优雅退出
internal/buildinfo/   构建期注入的版本/提交/时间
internal/config/      命令行与环境变量的解析归一
internal/forge/       路由元登记表：同一份表既注册 HTTP 路由，也生成前端消费的文档
internal/server/      静态产物托管、SPA 回落、元信息端点、/api 处理器与统一信封
internal/zt/          zerotier-one 本地控制器客户端与业务组装
web/                  前端工程（Vite 根）
  src/                Vue 3 + Element Plus + Pug 源码
  dist/               构建产物，由 web/embed.go 用 go:embed 打进二进制
scripts/build.ps1     交叉编译脚本
.github/workflows/    打 tag 自动发布二进制
```

四处值得注意的实现约束：

- **登记表同源**：`/_forge/routes` 文档与实际 HTTP 路由都从 `forge.Registry` 生成，前端按命名路由调用（`@route-forge/vue` builtin adapter），二者不可能漂移。文档字段与 route-forge/laravel 的下发格式逐字段对齐（`schemeVersion` / `levels` / `parameter_defaults` 等）。
- **数值保真**：上游 JSON 一律以 `json.RawMessage` 承载，需要改写的对象走 `map[string]json.RawMessage`。若经 `float64` 往返，`creationTime` 这类毫秒时间戳会被写成科学计数法。
- **nwid 规则**：本地控制器要求新建网络的 `nwid` = 控制器节点地址（10 位 hex）+ 6 位随机 hex，桥会自动完成拼接。
- **Pug 类名书写**：UnoCSS 的 `@unocss/extractor-pug` 在本项目实测只能可靠提取属性形式 `div(class="ml-auto")`，点号链形式（如 `.flex.justify-between` 里的 `justify-between`）会被漏掉、产物 CSS 里没有对应选择器。新增样式类请一律写成属性形式。

## 安全边界

- 默认只绑定 `127.0.0.1`。改成 `0.0.0.0` 会让同网段任何人都能操作这个面板，请只在可信网络里这么做。
- 桥不校验调用方身份，但每个 `/api` 请求必须携带令牌（请求头 `X-ZT-Token`），否则直接返回业务错误；令牌不会被发往 `--url` 与基址覆盖值之外的地址。
- 令牌保存在浏览器 `sessionStorage`：关闭标签页即失效，新标签页需重新粘贴；主页「系统状态」标题右侧提供显式清除入口。
- `/api/raw` 只读透传仅接受本站相对路径，`//host`、含 scheme、含 `..` 的路径一律拒绝，避免被当作跳板。
- 请求头 `X-ZT-Base` 的基址覆盖必须是 `http(s)://` 绝对地址。

## 测试

```bash
go test ./...        # 契约文档、只读字段剥离、nwid 生成、数字保真、路径守卫、SPA 回落
pnpm -C web build    # 前端产物构建
```

## 发布

推送 `v` 前缀的 tag（如 `v2.0.0`）即触发 `.github/workflows/release.yml`：跑 `gofmt`/`go vet`/`go test` → 构建前端 → 交叉编译六个目标 → 连同 `checksums.txt` 上传到 GitHub Release。也可在 Actions 页手动触发以验证流水线（手动触发不创建 Release，产物标记为 prerelease 之外仅用于自检）。

版本口径：**2.x 起为 Go 单执行文件实现；1.x 是 Laravel 版本**，其代码与历史归档在 `laravel` 分支。tag 必须带 `v` 前缀——Actions 的 tag 过滤模式只保证 `*` 这类通配可用，不放裸版本号以免出现「打了 tag 但流水线没跑」的静默失败。

## 许可

MIT。
