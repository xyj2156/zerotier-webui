# zerotier-webui

面向 **zerotier-one 本地自建控制器** 的极简 Web 管理面板。
只在你想改网络的时候 `php artisan serve` 起来，改完 `Ctrl+C` 走人——没有守护进程、没有 Docker、系统里不留任何常驻服务。

## 特性

- **按需启动**：单进程 `php artisan serve` 就能跑完前后端；不用的时候完全关掉，不留后台。
- **零外部依赖**：默认 SQLite（`database/database.sqlite` 一个文件），账号、会话、缓存都在里面，删掉即归零。
- **不接管 ZeroTier Central**：直接对接 zerotier-one 内置的本地控制器 REST API（`http://127.0.0.1:9993` + `X-ZT1-Auth` 头），网络拓扑、密钥、成员全在你自己机器上。
- **配置字段整体透传**：network config 与 member 不裁剪；前端表单覆盖不到的字段留了 JSON 全量编辑兜底，zerotier-one 后续新增字段不用改代码即可维护。
- **前端命名路由**：接入 [`@route-forge/vue`](https://github.com/xyj2156/route-forge) 与后端 [`route-forge/laravel`](https://github.com/xyj2156/route-forge-laravel)，接口调用完全按名拼装，无手写 URL。

## 运行环境

| 组件 | 版本 | 备注 |
| ---- | ---- | ---- |
| PHP | 8.3+ | 扩展需 `pdo_sqlite`、`mbstring`、`openssl`、`curl` |
| Composer | 2.x | 后端依赖 |
| Node.js | 20+ | 前端构建 |
| pnpm | 9+ | 前端包管理（`corepack enable` 即可） |
| zerotier-one | 1.12+ | 需开启本地控制器 API（默认监听 `127.0.0.1:9993`） |

## 快速开始

### 1. 装依赖

```bash
git clone https://github.com/xyj2156/zerotier-webui.git
cd zerotier-webui

composer install
pnpm install
```

### 2. 首次初始化（一次性）

```bash
cp .env.example .env
php artisan key:generate

touch database/database.sqlite
php artisan migrate --seed
```

`--seed` 会建一个默认管理员：`admin@example.com` / `password`。想换邮箱或密码，先在 `.env` 里配 `ADMIN_EMAIL` / `ADMIN_PASSWORD` 再 migrate。

### 3. 配 zerotier-one 控制器

编辑 `.env`：

```dotenv
# 控制器地址，本机默认 9993
ZEROTIER_API_URL=http://127.0.0.1:9993

# 令牌：控制器所在机器上 authtoken_secret 文件的内容
#   Linux:   cat ~/.zerotier/authtoken_secret
#   Windows: type C:\ProgramData\ZeroTier\One\authtoken_secret
ZEROTIER_API_TOKEN=<粘贴上面的字符串>

# 控制器 10 位节点地址：留空则自动 GET /status 拿；仅多控制器场景手填
ZEROTIER_NODE_ADDRESS=
```

### 4. 要用的时候启动

**推荐（日常使用，只留一个进程）**：先 build 一次，之后每次只跑 `php artisan serve`。

```bash
pnpm build              # 产物落到 public/build，一次即可
php artisan serve       # 浏览器打开 http://127.0.0.1:8000
```

**开发模式（前端热更新）**：两个终端。

```bash
# 终端 A
php artisan serve

# 终端 B
pnpm dev                # http://127.0.0.1:5173，代理到 artisan serve
```

### 5. 用完关掉

终端按 `Ctrl+C` 即可。进程不留痕、系统不留后台。想彻底重置删掉 `database/database.sqlite` 再重新 `migrate --seed`。

## 与 ZeroTier Central 的差异

本项目对接的是 **zerotier-one 本地控制器**，不是 `my.zerotier.com` 的 Central API。上手前值得知道：

- 列表 `GET /controller/network` 只返回 `nwid` 字符串数组，需要逐个 `GET /controller/network/<nwid>` 拉配置——服务层已经聚合好了。
- 新建网络时 `nwid` **必须** = 控制器自身 10 位节点地址 + 6 位随机 hex，且 body 要带 `active: true`；否则 403。
- 没有 Central 的独立 `/route`、`/node`、`/peer` drop 端点——路由、成员授权、别名全部通过 `POST /controller/network/<nwid>` 的整体 config 完成。
- `member.clientVersion` 本地是字符串（如 `"1.12.2"`），Central 是对象；前端已做兼容。

## 开发

```bash
php artisan test        # 认证 + zerotier API mock 共 12 例
```

代码结构：

```
app/
├── Http/Controllers/       # LoginController / ViewController / ZerotierController
├── Http/Middleware/Jwt.php
└── Services/
    ├── JwtService.php      # 自写 HS256，无外部 JWT 包
    └── ZerotierService.php # zerotier-one 本地控制器 HTTP 客户端
resources/js/
├── routes/index.js         # 前端路由（@route-forge/vue 命名调用）
├── utils/auth.js           # localStorage 令牌 / 用户
└── views/
    ├── login.vue
    ├── dashboard.vue
    ├── network.vue
    ├── network-detail.vue  # 成员管理 / 网络配置 / 原始数据 三 tab
    └── user.vue
```

## 许可

[MIT](LICENSE) © 2026 xyj2156
