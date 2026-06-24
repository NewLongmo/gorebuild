# DW0RDWK

> 面向网课代理、货源对接与订单运营的一体化管理系统。<br>
> Go + Vue 重构旧式 PHP/Layui 业务，保留成熟流程，同时补齐现代后台、队列、统计、同步和插件化运行能力。

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white)
![Vue](https://img.shields.io/badge/Vue-3-42B883?style=flat-square&logo=vue.js&logoColor=white)
![Vite](https://img.shields.io/badge/Vite-6-646CFF?style=flat-square&logo=vite&logoColor=white)
![MySQL](https://img.shields.io/badge/MySQL-8-4479A1?style=flat-square&logo=mysql&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=flat-square&logo=redis&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)

> [!IMPORTANT]
> **作者闲言**：综合了 24 年和 25 年的几套 29 模板进行的重构，有旧代码也有较新的代码，从中借鉴很多思路。大部分由 AI 辅助完成（貌似全是）。如果有用就点点 Star，谢谢你！

> [!WARNING]
> 本项目仅供学习交流使用，请勿用于商业用途，否则后果自负。
>
> 欢迎 Star 喵，欢迎 PR 喵。

## 项目亮点

DW0RDWK 不是单纯的后台模板，而是一套围绕“课程商品、代理用户、上游货源、订单执行、售后处理”的完整业务系统。

| 能力 | 说明 |
| --- | --- |
| 后台管理 | 管理员首页、数据统计、菜单管理、用户/代理、分类、课程、订单、工单、充值卡、系统配置 |
| 订单提交 | 用户端订单提交工作台，支持分类筛选、平台搜索、查课、批量账号整理、收藏课程和推荐下单 |
| 货源管理 | 支持 29wk 货源配置、余额查询、拉取商品、价格规则、词替换、分类独立价格和同步配置 |
| 订单队列 | Redis 队列执行普通/极速订单，支持失败重试、队列恢复、补刷、改密重刷和退款 |
| 售后与日志 | 用户端、管理端和公开自助查单页可查看订单进度与执行日志，支持自助补刷和改密重刷 |
| 统计运营 | 近 7/30 日趋势、订单排行、平台排行、充值排行、邀请排行、公告和运营配置 |
| 运行监控 | 记录 29wk 订单同步、价格同步、队列运行状态，支持管理员手动触发同步任务 |
| 直跑插件 | 平台插件、Worker 节点、Worker 指令和代理池管理，为后续平台直跑能力预留统一框架 |
| 旧版兼容 | 保留 `api.php` 风格接口，便于旧系统或第三方继续以 `uid + key` 对接 |

## 技术架构

```text
DW0RDWK
├─ backend/              Go + Fiber API、GORM、队列 worker、29wk 对接、运行监控
├─ frontend/             Vue 3 + Vite + Ant Design Vue 管理端/用户端
├─ backend/migrations/   MySQL 初始化和兼容迁移 SQL
├─ scripts/              本地验证和部署后冒烟检查脚本
└─ docker-compose.yml    MySQL、Redis、Backend、Frontend 一键编排
```

运行时由四个服务组成：

- `frontend`：Nginx 托管前端静态资源，并将 `/api/` 代理到后端。
- `backend`：提供 API、认证、业务服务、队列消费、定时同步和迁移种子。
- `mysql`：保存用户、课程、订单、货源、日志、菜单和运行状态。
- `redis`：作为订单队列和缓存依赖。

## 快速启动

本地需要安装 Docker 和 Docker Compose。

```bash
git clone https://github.com/NewLongmo/DW0RDWK.git
cd DW0RDWK
docker compose up --build
```

启动后访问：

- 前端入口：`http://localhost:8081`
- 自助查单：`http://localhost:8081/support`
- 后端健康检查：`http://localhost:8080/healthz`
- 后端依赖检查：`http://localhost:8080/readyz`
- API 前缀：`http://localhost:8080/api/v1`

开发环境没有 `.env` 文件时，会使用 `docker-compose.yml` 中的默认值。默认管理员仅用于本地初始化，公开部署前必须立即修改：

```text
账号：admin
密码：admin123
```

## 生产部署

建议复制模板后再启动，不要把真实 `.env` 提交到仓库。

```bash
cp .env.deploy.example .env
docker compose up --build -d
```

上线前至少修改这些配置：

```env
APP_ENV=production
AUTH_SECRET=替换为至少32位随机字符串
MYSQL_ROOT_PASSWORD=替换为数据库root密码
MYSQL_PASSWORD=替换为应用数据库密码
BOOTSTRAP_ADMIN_PASSWORD=替换为初始管理员密码
CORS_ALLOW_ORIGINS=http://你的域名或服务器IP
```

宝塔面板部署时，可以把源码上传并解压到站点目录，例如 `/www/wwwroot/dw0rdwk`，再在该目录执行 `docker compose up --build -d`。如果宝塔使用 Nginx 反向代理，将域名代理到前端容器端口 `8081` 即可。

## 常用命令

后端测试：

```bash
cd backend
go test ./...
```

前端构建：

```bash
cd frontend
npm run build
```

完整验证：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts\verify.ps1
```

Linux/macOS：

```bash
sh scripts/verify.sh
sh scripts/smoke.sh
```

## 29wk 货源对接

后台进入“货源管理”后新增货源：

- 接入方式选择 `29通用`。
- 接口地址填写上游站点根地址或 `api.php` 地址。
- UID 和 Key 填写上游提供的对接信息。
- 根据业务需要配置订单同步、货源同步、价格模式、取整规则、词替换和分类价格策略。

配置完成后，可以拉取上游商品并选择上架。本地课程会保存上游分类、商品 ID、对接编码和价格策略，后续查课、下单、补刷和同步会复用该货源配置。

## 旧版 API 兼容

代理登录后可在“账号安全”页面开通或更换 API Key。旧系统对接方可继续按 `uid + key` 调用：

```text
POST /api.php?act=getmoney
POST /api.php?act=getclass
POST /api.php?act=get
POST /api.php?act=add
POST /api.php?act=getadd
POST /api.php?act=uporder
POST /api.php?act=chadan
POST /api.php?act=budan
```

普通 JSON 对接通道仍支持 `/courses/query`、`/orders`、`/orders/refresh` 等接口，方便接入自定义平台。

## 安全说明

- `.env` 已被 `.gitignore` 忽略，不要提交真实数据库密码、JWT 密钥、上游 UID/Key 或服务器密码。
- `.env.deploy.example` 和 `backend/.env.example` 只保留占位配置，部署时请复制后自行填写。
- 默认管理员只用于首次启动演示，生产环境必须修改账号密码和 `AUTH_SECRET`。
- 公开仓库不包含真实上游货源凭据、线上数据库备份、服务器 SSH 信息或构建产物。

## 贡献

欢迎提交 Issue 和 Pull Request。建议在提交前先运行：

```bash
cd backend && go test ./...
cd ../frontend && npm run build
```

如果改动涉及数据库结构，请同时确认现有数据可无损迁移；如果改动涉及订单、余额、退款、上游下单或同步任务，请补充对应的回归说明。

## License

DW0RDWK 使用 [MIT License](./LICENSE) 开源。
