# ShareLink

ShareLink 是一个自用短链管理和反向代理网关。它可以把公开的两段式路径，例如 `/export/backups`，映射到真实目标地址；管理员可以选择反向代理或 302 跳转模式，并为链接配置有效期、访问日志、User-Agent 策略、下载文件名和内存缓存。

适合用来发布临时文件、隐藏上游域名、统一管理外链入口，或为内部资源提供带统计和访问控制的公开访问地址。

## 主要功能

- **短链管理**：通过后台创建、搜索、启用、禁用和删除短链，公开访问路径固定为 `/{prefix}/{slug}`。
- **反向代理模式**：由 ShareLink 代请求上游地址，访问者看不到真实目标域名。
- **302 跳转模式**：直接跳转到目标地址，同时保留访问日志。
- **SSRF 防护**：仅允许 HTTP/HTTPS 目标，解析目标域名并拦截不安全地址，降低内网探测和 DNS Rebinding 风险。
- **访问有效期**：支持开始时间、过期时间和手动启停。
- **User-Agent 策略**：支持白名单、黑名单、混合模式、关键词匹配、正则匹配和空 UA 控制。
- **内存缓存**：代理模式下可对 GET 响应启用 LRU 内存缓存，并限制缓存容量、TTL 和单对象大小。
- **下载文件名控制**：支持继承上游文件名、自定义文件名或从目标 URL 自动提取文件名。
- **访问统计**：记录 PV/UV、状态码、缓存命中、来源、User-Agent、IP 哈希和 GeoIP 区域信息。
- **管理后台**：React + TypeScript + Vite 构建的深色后台界面，后端生产模式可直接托管前端产物。

## 技术栈

- 后端：Go 1.26、Gin、GORM、SQLite
- 前端：React 19、TypeScript、Vite
- 部署：Docker 多阶段构建、Docker Compose、Nginx 反向代理示例
- GeoIP：`ip2region.xdb` 离线数据库

## 快速启动

推荐使用 Docker Compose 部署。

1. 复制环境变量模板：

   ```bash
   cp .env.example .env
   ```

2. 修改 `.env` 中的关键配置：

   ```env
   INITIAL_ADMIN_PASSWORD=change-this-password
   JWT_SECRET=replace-with-a-random-32-character-secret
   ```

3. 启动服务：

   ```bash
   docker compose up -d
   ```

4. 打开管理后台：

   ```text
   http://localhost:8080/admin
   ```

默认登录用户名是 `url`，密码是 `.env` 中的 `INITIAL_ADMIN_PASSWORD`。该密码只在首次初始化空数据库时写入，后续请在后台设置页修改密码。

## 本地开发

### 环境要求

- Go 1.26.x
- Node.js 22.x 推荐
- npm

### 目录结构

```text
backend/              Go 后端：API、鉴权、反向代理、缓存、日志、设置
backend/cmd/server/   后端服务入口
backend/data/         本地 GeoIP 数据库位置
frontend/             React / TypeScript / Vite 管理后台
docker-compose.yml    Docker Compose 部署配置
nginx.example.conf    Nginx 反向代理示例
Dockerfile            前后端一体化多阶段构建
.env.example          环境变量模板
run-test.cmd          Windows 本地测试启动脚本
```

### 本地一键测试

Windows 环境可以直接运行：

```cmd
run-test.cmd check
run-test.cmd
```

`check` 会检查 Go、npm、GeoIP 数据库和前端依赖是否可用；不带参数会分别启动后端和前端测试窗口。

### 启动后端

在本地开发时建议从 `backend` 目录启动，这样相对路径会正确指向 `sharelink.db`、`data/ip2region.xdb` 和 `../frontend/dist`。

```powershell
cd backend
$env:INITIAL_ADMIN_PASSWORD="adminpassword"
$env:DB_DSN="sharelink.db"
$env:IP_DB_PATH="data/ip2region.xdb"
go run cmd/server/main.go
```

后端默认监听：

```text
http://localhost:8080
```

常用接口路径：

```text
GET  /health
POST /api/auth/login
POST /api/auth/logout
GET  /api/admin/*
```

公开短链不走 `/api`，而是两段式路径，例如：

```text
/export/backups
/download/report.zip
```

### 启动前端

```bash
cd frontend
npm install
npm run dev
```

开发后台地址：

```text
http://localhost:5173/admin
```

生产构建：

```bash
npm run build
```

如果后端不是 `http://localhost:8080`，可以在前端开发环境设置：

```bash
VITE_API_BASE_URL=http://localhost:18080
```

同时需要把前端开发域名加入后端 `CORS_ALLOWED_ORIGINS`。

## 环境变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `PORT` | `8080` | 后端监听端口 |
| `DB_TYPE` | `sqlite` | 数据库类型，目前按 SQLite 使用 |
| `DB_DSN` | `/data/sharelink.db` | SQLite 数据库文件路径 |
| `DATA_DIR` | `/data` | 数据目录，容器内会挂载为持久卷 |
| `IP_DB_PATH` | `/app/data/ip2region.xdb` | GeoIP 数据库路径；Docker 默认使用镜像内文件 |
| `LOG_LEVEL` | `info` | 日志级别，`debug` 会启用 Gin 调试模式 |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:5173,http://127.0.0.1:5173` | 前端开发跨域白名单 |
| `APP_TIMEZONE` | `Asia/Shanghai` | 访问统计、今日 PV/UV 和访客日粒度 UV 的业务时区；数据库时间统一按 UTC 存储 |
| `INITIAL_ADMIN_PASSWORD` | 无 | 空数据库首次启动时的管理员初始密码，未设置会拒绝启动 |
| `JWT_SECRET` | 示例值 | JWT 签名密钥，生产环境必须替换为随机强密钥 |
| `VITE_API_BASE_URL` | `http://localhost:8080` | 仅前端开发服务器使用，生产构建走同源 API |

## 链接规则

- 每条公开链接由 `prefix` 和 `slug` 组成，最终路径是 `/{prefix}/{slug}`。
- `prefix` 必须以 `/` 开头，只能是单层路径，例如 `/export`。
- `slug` 不能包含 `/`，为空时后端会自动生成 10 位随机值。
- `prefix` 和 `slug` 仅允许字母、数字、`.`、`_`、`-`。
- `/admin`、`/api`、`/assets`、`/health` 等系统路径不能作为短链前缀。
- 创建后不允许修改 `prefix` 和 `slug`，避免已发布链接失效。

## 部署说明

Docker Compose 会构建前端和后端，并把运行数据保存到项目根目录的 `./data`：

```bash
docker compose up -d --build
```

如果放在 Nginx 后面，可以参考：

```text
nginx.example.conf
```

反向代理时建议保留这些头：

```nginx
proxy_set_header Host $host;
proxy_set_header X-Real-IP $remote_addr;
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
proxy_set_header X-Forwarded-Proto $scheme;
```

如果要让后端信任代理传入的客户端 IP，需要在后台设置中开启 `trust_proxy_headers`，并确保只有可信 Nginx 能访问后端端口。

## 验证命令

后端测试：

```bash
cd backend
go test -v ./...
```

前端检查和构建：

```bash
cd frontend
npm run lint
npm run build
```

健康检查：

```bash
curl http://localhost:8080/health
```

## 注意事项

- `INITIAL_ADMIN_PASSWORD` 只用于首次初始化密码；数据库已有密码后，修改该环境变量不会重置登录密码。
- 生产环境必须替换 `JWT_SECRET`，不要使用示例值。
- 反向代理模式会拦截上游 3xx 跳转，避免跳转链绕过 SSRF 检查。
- 默认最大代理响应大小为 5 MB，可在后台设置中调整。
- GeoIP 依赖 `ip2region.xdb`，当前版本启动时必须能读取该文件；文件缺失或损坏会导致服务启动失败。
