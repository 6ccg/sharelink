# ShareLink 产品需求与实现状态

本文档记录 ShareLink 当前版本的产品边界、已实现能力和未完成事项。它不再作为早期设计草案维护；已完成能力只保留必要结论，未完成能力明确边界和后续处理方式。

## 1. 产品定位

ShareLink 是一个面向个人使用场景的短链接管理与反向代理网关。

它的核心价值不是单纯缩短 URL，而是提供一个可管理、可统计、可限制、可过期、可缓存的公开访问入口。管理员在后台创建两段式公开路径，例如 `/export/abc123`，并把它映射到真实目标 URL。访问者访问公开路径时，系统按配置执行反向代理或 302 跳转。

当前定位：

- 单管理员使用。
- 单实例部署。
- SQLite 持久化。
- Docker Compose 部署。
- 后台 Web 管理。
- 公开链接访问不需要登录。
- 后台页面和 `/api/admin/*` 需要登录。

非目标：

- 不做完整网站镜像。
- 不做浏览器级通用代理。
- 不做开放代理接口，例如 `/proxy?url=...`。
- 不做多用户、RBAC、组织管理。
- 不做高可用、分布式缓存、多节点同步。
- 不替代 CDN，也不实现完整 HTTP 缓存协议。

## 2. 当前技术栈

| 模块 | 当前实现 |
| --- | --- |
| 后端 | Go 1.26、Gin、GORM、SQLite |
| 前端 | React 19、TypeScript、Vite |
| 缓存 | 进程内 RAM LRU 缓存 |
| GeoIP | `ip2region.xdb` 离线库 |
| 部署 | Dockerfile、根目录 Docker Compose、Nginx 示例 |
| 鉴权 | 单管理员密码 + JWT |

## 3. 已实现能力

### 3.1 管理员登录

已实现。

- 默认用户名固定为 `url`。
- 初始密码来自 `INITIAL_ADMIN_PASSWORD`，仅在空数据库首次启动时写入。
- 密码使用 bcrypt 哈希存储。
- 登录成功后签发 24 小时有效的 JWT。
- `/api/admin/*` 使用 Bearer Token 鉴权。
- 后台前端会在无 Token 或 Token 失效时跳转到 `/login`。
- 登录失败按客户端 IP 做简单内存限流：10 分钟内 5 次失败后返回 429。
- 支持后台修改管理员密码，修改时需要验证旧密码。

边界：

- 只有一个管理员账号。
- 没有注册、找回密码、多因素认证。
- JWT 失效时间目前固定为 24 小时，不支持后台配置。
- 登录失败计数保存在进程内，服务重启后清空。

### 3.2 链接管理

已实现。

链接字段包括：

- `prefix`
- `slug`
- `public_path`
- `target_url`
- `mode`
- `enabled`
- `start_time`
- `expire_time`
- `cache_enabled`
- `cache_ttl`
- `cache_max_object_size_mb`
- `filename_mode`
- `custom_filename`
- `ua_policy_id`
- `note`

支持能力：

- 创建、列表、详情、编辑、删除链接。
- 启用、禁用链接。
- 自动生成 10 位随机 slug。
- 按关键词、模式、启用状态查询链接。
- 创建后不允许修改 `prefix` 和 `slug`，避免公开路径失效。

路径规则：

- 公开路径固定为 `/{prefix}/{slug}`。
- `prefix` 必须以 `/` 开头，且只能是单级路径。
- `slug` 不能包含 `/`。
- `prefix` 和 `slug` 只允许字母、数字、`.`、`_`、`-`。
- 同一 `prefix + slug` 组合必须唯一。
- `/admin`、`/api`、`/assets`、`/static`、`/login`、`/logout`、`/favicon.ico`、`/favicon.svg`、`/health`、`/icons.svg` 是保留路径。

边界：

- 没有批量导入导出。
- 没有链接分组、标签、搜索高级语法。
- 没有链接访问密码。
- 没有按 IP、地区或 Referer 做访问控制。

### 3.3 公开访问与有效期

已实现。

访问公开路径时系统会：

1. 按 `prefix + slug` 查询链接。
2. 检查链接是否存在。
3. 检查是否启用。
4. 检查开始时间和过期时间。
5. 检查 User-Agent 策略。
6. 记录访问日志。
7. 根据 `mode` 执行反向代理或 302 跳转。

返回边界：

- 链接不存在：404。
- 链接禁用：403。
- 未到开始时间：403。
- 已过期：410。
- UA 被拦截：403。
- 目标连接失败：502。
- 代理响应超过限制：413 或连接中断。

### 3.4 反向代理模式

已实现。

`mode = proxy` 时，ShareLink 请求后台配置的 `target_url`，并把响应返回给访问者。访问者浏览器地址栏保持在 ShareLink 公开路径。

当前代理策略：

- 只允许 `http://` 和 `https://` 目标。
- 代理请求前会做 SSRF 检查。
- 后端自定义 `DialContext`，先解析并校验目标域名，再连接校验通过的 IP，降低 DNS Rebinding 风险。
- 请求目标站时重写 Host 为目标 URL 的 Host。
- 删除客户端请求中的 Cookie。
- 删除响应中的 `Set-Cookie`。
- 删除 hop-by-hop headers。
- 上游 3xx 响应会被拦截，返回 502，不向浏览器透传 `Location`。
- HTTPS 目标使用严格 TLS 校验，不提供忽略证书错误开关。
- 支持配置连接超时、响应头超时、整体代理超时。
- 支持最大代理响应大小限制，默认 5 MB。

已明确边界：

- 这是“一次性目标 URL 代理”，不是完整网站镜像。
- 不改写 HTML 内部链接。
- 不递归代理页面中的图片、CSS、JS。
- 不自动改写 JavaScript 里的接口地址。
- 不解析和替换 HTML 中的绝对路径。
- 不保证复杂网页、SPA 页面完整可用。
- 不保证目标响应内容里不会出现源站域名。
- 不支持代理 WebSocket。
- 不支持上游 3xx 跳转。

适合场景：

- 文件下载。
- 图片、文本、JSON。
- 简单 HTML。
- 单次接口响应。

不适合场景：

- 需要完整隐藏源站的复杂网页。
- 页面内依赖大量相对资源或跨域接口的前端应用。
- 需要登录态 Cookie 的目标站。

### 3.5 302 跳转模式

已实现。

`mode = redirect` 时，公开路径返回 302 到 `target_url`。

- 跳转前仍会检查启用状态、有效期和 UA 策略。
- 跳转访问会记录日志。
- 访问者会看到真实目标 URL，这是该模式的预期行为。
- 跳转前也会对目标 Host 做 SSRF 检查。

边界：

- 当前只支持 302，不支持配置 301。
- 跳转模式不隐藏真实目标地址。
- 跳转模式不使用 RAM 缓存。

### 3.6 RAM 缓存

已实现。

缓存范围：

- 仅对 `mode = proxy` 的 GET 请求生效。
- 链接必须开启 `cache_enabled`。
- 全局缓存必须开启。
- 仅缓存 2xx 响应。
- 缓存状态会写入访问日志。

缓存配置：

- 全局缓存开关：`global_cache_enabled`，默认 true。
- 全局 RAM 上限：`global_cache_max_memory_mb`，默认 64 MB。
- 链接缓存 TTL：`cache_ttl`，默认 600 秒。
- 链接单对象上限：`cache_max_object_size_mb`，默认 5 MB。

缓存管理：

- 查看缓存对象数、占用内存、总上限、命中数、未命中数和命中率。
- 清空全部缓存。
- 清除单个链接的缓存。
- 修改、禁用、删除链接时会清理该链接缓存。

边界：

- 缓存仅在进程内，服务重启后全部丢失。
- 不实现磁盘缓存。
- 不实现 ETag、Last-Modified、304 条件请求。
- 不按请求头区分缓存，例如 User-Agent、Accept-Language、Authorization。
- 当前缓存 Key 包含 `prefix`、`slug`、请求方法和原始 Query String。
- HEAD、POST、PUT、PATCH、DELETE 不缓存。

### 3.7 User-Agent 策略

已实现。

支持策略：

- disabled
- whitelist
- blacklist
- mixed

匹配能力：

- 关键词包含匹配。
- 正则匹配。
- 大小写敏感配置。
- 空 User-Agent 是否允许。
- 全局 UA 策略。
- 链接级 UA 策略。
- 策略测试接口。

优先级：

- 链接配置了 UA 策略时，使用链接级策略。
- 链接没有配置时，使用全局 UA 策略。
- 没有全局策略时默认允许。

边界：

- 不支持 IP 黑白名单。
- 不支持 Referer 黑白名单。
- 不支持多条件组合规则。
- 正则表达式错误时该条关键词不会命中，但创建/更新时目前只校验 JSON 数组格式，不预编译全部正则。

### 3.8 访问日志与统计

已实现。

访问日志字段包括：

- link_id
- prefix
- slug
- public_path
- IP 明文
- IP hash
- visitor_hash
- User-Agent
- Referer
- Country / Region / City
- Access Time
- Mode
- Status
- Blocked Reason
- Response Status Code
- Upstream Status Code
- Response Size
- Cache Status

后台支持：

- 日志分页查询。
- 按链接、路径、IP、国家、状态、模式、缓存状态、时间范围和关键词筛选。
- 概览统计：链接总数、启用链接数、总 PV、总 UV、今日 PV、今日 UV、缓存状态、GeoIP 状态、运行时长。
- 趋势统计：最近 15 天 PV/UV。
- Geo 分布统计。
- User-Agent 分布统计。

边界：

- 日志自动清理配置项已存在，但当前代码未看到定时清理 worker 的实际执行逻辑。
- 没有日志导出。
- 没有按链接维度的完整详情报表。
- UV 当前按 `IP + User-Agent + 日期` 生成 hash，不能等同于真实独立用户。

### 3.9 GeoIP

已实现。

- 使用 `ip2region.xdb` 离线库。
- 启动时把数据库读入内存。
- 查询结果使用进程内 `sync.Map` 缓存。
- 内网或本地 IP 显示为 `内网/本地`。
- 查询失败显示 `未知`。
- 后台设置可通过 `geoip_enabled` 关闭解析。

重要边界：

- `IP_DB_PATH` 指向的文件缺失或损坏时，当前服务会启动失败。
- 关闭 `geoip_enabled` 只影响运行期解析，不会跳过启动时加载文件。
- GeoIP 缓存没有容量上限和过期策略。
- 解析精度取决于 `ip2region.xdb` 数据质量。

### 3.10 部署

已实现。

- 提供 Dockerfile。
- 提供根目录 `docker-compose.yml`。
- 提供 `.env.example`。
- 提供根目录 `nginx.example.conf`。
- Docker 镜像内同时包含后端二进制、前端 `dist` 和 GeoIP 数据库。
- Compose 使用 `./data:/data` 绑定挂载，把 SQLite 数据库保存到项目根目录的 `data` 目录。
- GeoIP 数据库随镜像放在 `/app/data/ip2region.xdb`，避免 `./data:/data` 首次挂载时遮住 GeoIP 文件。
- 后端生产模式会托管前端构建产物。

边界：

- Compose 当前是本地构建，不是拉取已发布镜像。
- HTTPS 由前置 Nginx/Caddy/Traefik 处理，ShareLink 自身只监听 HTTP。
- 没有数据库备份、恢复脚本。
- 没有健康检查配置写入 Compose。

## 4. 未完成与后续增强

### 4.1 P0 需要先厘清或修正

这些不是新功能，但会影响文档和使用预期。

| 项目 | 当前状态 | 边界或处理建议 |
| --- | --- | --- |
| GeoIP 文件缺失行为 | 当前会启动失败 | README 和部署文档必须明确 `ip2region.xdb` 是启动依赖，或后续改为缺失时降级运行 |
| `DB_TYPE` | 配置存在，但实际只打开 SQLite | 文档中不要承诺 PostgreSQL；如保留字段，应注明当前忽略 |
| 日志自动清理 | 有设置项，未确认有定时清理实现 | 若未实现，应从 UI 文案中降低承诺，或补 worker |
| `trust_proxy_headers` | 已支持开关，但没有受信任代理 IP 列表 | 只能在后端只被可信反代访问时开启 |
| API 文档 | PRD 不再展开所有接口 | 如需要对外 API，应单独生成 `docs/API.md` |

### 4.2 P1 可选增强

- PostgreSQL 实际支持。
- 日志导出。
- 日志自动清理 worker。
- 备份与恢复脚本。
- 链接批量导入导出。
- 单链接详情统计页。
- 自定义公开错误页。
- IP 黑白名单。
- Referer 策略。
- 可配置 JWT 过期时间。
- Compose healthcheck。

### 4.3 P2 后续增强

- 完整 HTTP 缓存协议。
- ETag / Last-Modified 再验证。
- 304 条件请求。
- SingleFlight 请求合并。
- 磁盘缓存。
- 多域名绑定。
- Webhook 通知。
- 多管理员账户。
- API Token 管理。

## 5. 验收标准

### 5.1 核心链路

| ID | 场景 | 预期 |
| --- | --- | --- |
| UAT-01 | 首次空数据库启动且设置 `INITIAL_ADMIN_PASSWORD` | 服务启动，管理员密码写入数据库 |
| UAT-02 | 首次空数据库启动但未设置 `INITIAL_ADMIN_PASSWORD` | 服务拒绝启动并输出明确错误 |
| UAT-03 | 使用用户名 `url` 和正确密码登录 | 返回 JWT，进入后台 |
| UAT-04 | 未登录请求 `/api/admin/links` | 返回 401 |
| UAT-05 | 未登录访问 `/admin` | 前端跳转 `/login` |

### 5.2 链接管理

| ID | 场景 | 预期 |
| --- | --- | --- |
| UAT-06 | 创建 `/export/test1` | 创建成功，列表可见 |
| UAT-07 | 重复创建 `/export/test1` | 创建失败，提示冲突 |
| UAT-08 | 创建 `/go/test1` | 可与 `/export/test1` 共存 |
| UAT-09 | 创建 prefix 为 `/admin` 的链接 | 创建失败 |
| UAT-10 | 创建 slug 含 `/` 的链接 | 创建失败 |
| UAT-11 | 禁用链接后访问 | 返回 403 |
| UAT-12 | 过期链接访问 | 返回 410 |
| UAT-13 | 未到开始时间访问 | 返回 403 |

### 5.3 代理与跳转

| ID | 场景 | 预期 |
| --- | --- | --- |
| UAT-14 | proxy 模式访问文件 | 地址栏保持 ShareLink 路径，返回目标内容 |
| UAT-15 | proxy 模式访问 JSON | 返回目标 JSON |
| UAT-16 | 目标 URL 不可访问 | 返回 502 并记录日志 |
| UAT-17 | 上游 3xx 跳转 | 返回 502，不透传 Location |
| UAT-18 | 响应体超过限制 | 返回 413 或中断并记录 `response_size_exceeded` |
| UAT-19 | redirect 模式访问 | 返回 302，并记录日志 |
| UAT-20 | redirect 链接过期 | 不跳转，返回 410 |

### 5.4 缓存

| ID | 场景 | 预期 |
| --- | --- | --- |
| UAT-21 | 开启缓存后首次 GET | `cache_status = miss` |
| UAT-22 | 第二次相同 GET | `cache_status = hit` |
| UAT-23 | 缓存 TTL 到期后访问 | 重新请求上游 |
| UAT-24 | 响应超过单对象缓存上限 | 正常返回但不写入缓存 |
| UAT-25 | 清除单链接缓存 | 该链接缓存失效 |
| UAT-26 | 清空全部缓存 | 所有缓存失效 |
| UAT-27 | POST 请求 | 不写入缓存 |

### 5.5 UA、日志和统计

| ID | 场景 | 预期 |
| --- | --- | --- |
| UAT-28 | 黑名单命中 | 返回 403，记录 blocked reason |
| UAT-29 | 白名单未命中 | 返回 403 |
| UAT-30 | 链接级 UA 策略存在 | 优先于全局策略 |
| UAT-31 | 成功访问链接 | PV 增加 |
| UAT-32 | 同 IP 同 UA 当日多次访问 | UV 按当前 hash 规则只算一次 |
| UAT-33 | GeoIP 解析成功 | 日志显示国家和地区 |
| UAT-34 | 内网 IP | 日志显示 `内网/本地` |

### 5.6 安全

| ID | 场景 | 预期 |
| --- | --- | --- |
| UAT-35 | 创建 `file://` target_url | 拒绝 |
| UAT-36 | 访问 `http://127.0.0.1` 目标 | 创建或访问阶段被拒绝 |
| UAT-37 | 目标域名解析到私网 IP | 被 SSRF 检查拦截 |
| UAT-38 | 公开错误页 | 不暴露 `target_url` 和堆栈 |
| UAT-39 | 访问者传入 `?url=` | 不会被当作代理目标 |

### 5.7 部署

| ID | 场景 | 预期 |
| --- | --- | --- |
| UAT-40 | Docker Compose 启动 | 服务监听 8080 |
| UAT-41 | 重启容器 | 数据库内容不丢失 |
| UAT-42 | 缺失 `ip2region.xdb` | 当前预期为启动失败 |
| UAT-43 | Nginx 反代访问 | 后台和公开链接可正常访问 |
| UAT-44 | `/health` | 返回 `{"status":"ok"}` |

## 6. 当前版本结论

ShareLink 当前已经达到个人使用 MVP：可以管理短链、通过代理或跳转发布链接、记录访问统计、做 UA 控制、启用 RAM 缓存，并用 Docker Compose 部署。

后续工作不应继续扩大第一版范围。优先级应放在：

1. 修正文档与实现不一致的地方。
2. 明确 GeoIP 启动依赖。
3. 处理日志清理设置与实际 worker 的差异。
4. 补充部署健康检查和数据备份说明。
5. 按真实使用需求再选择 P1 增强。
