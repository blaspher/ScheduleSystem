# Schedule System

## 1. 项目简介
这是一个基于 Go 的日程系统后端（MVP），聚焦个人日程管理、共享日历可见性控制、会议邀请流转与基础缓存优化。  
项目采用分层架构（handler/service/dao），并接入 MySQL、Redis、JWT，适合作为后端工程化与业务拆层的面试展示项目。

## 2. 功能概览
- 用户认证
  - 注册：`POST /api/v1/auth/register`
  - 登录：`POST /api/v1/auth/login`
  - 当前用户信息：`GET /api/v1/me`
  - 密码使用 `bcrypt` 哈希，鉴权使用 JWT Bearer Token
- 事件管理（Event CRUD）
  - 创建/更新/逻辑删除/列表/详情
  - 时间冲突校验（同 owner 下不可重叠）
  - 支持 `day/week` 视图和时间区间过滤
- 共享日历
  - 设置权限：`POST /api/v1/relations`
  - 查看他人日历：`GET /api/v1/users/{id}/calendar`
  - 可见性规则：`private` 不可见，`busy_only` 脱敏展示，`public` 完整展示
- 会议邀请
  - 创建会议：`POST /api/v1/meetings`
  - 我的待处理邀请：`GET /api/v1/meetings/invitations`
  - 接受/拒绝：`POST /api/v1/meetings/{id}/accept|reject`
  - 接受前冲突检测，冲突返回 `409`
- 缓存（Redis）
  - 缓存事件列表、事件详情（owner scoped 200）、共享日历（权限检查后）
  - 事件变更与关系变更触发失效

## 3. 技术栈
- 语言与框架：Go 1.25.x、Gin
- 数据库：MySQL + GORM
- 缓存：Redis（`go-redis/v9`）
- 认证：JWT（`github.com/golang-jwt/jwt/v5`）
- 配置：`godotenv` 自动加载根目录 `.env`
- 脚本：PowerShell（Windows 优先）

## 4. 系统架构
```text
Client (Apifox / Frontend)
        |
        v
Gin Router + JWT Middleware
        |
        v
Handler (参数校验/HTTP响应)
        |
        v
Service (业务规则/冲突校验/权限判断/缓存读写)
        |
        +---------------------> Redis (缓存)
        |
        v
DAO (GORM 数据访问)
        |
        v
MySQL
```

主分层说明：
- `internal/api`：路由与 handler，负责请求绑定、参数校验、HTTP 状态码映射。
- `internal/service`：核心业务逻辑（鉴权、日程冲突、可见性过滤、会议状态流转、缓存策略）。
- `internal/dao`：数据访问封装，隔离 SQL/GORM 细节。
- `internal/model`：领域模型定义。
- `pkg`：通用组件（JWT、数据库初始化、Redis/cache、middleware）。
- `cmd/server`：程序入口与依赖装配。

## 5. 核心数据模型
- `users`
  - `id`, `username`, `password`, `created_at`, `updated_at`
- `events`
  - `id`, `owner_id`, `title`, `description`, `event_type`, `visibility`
  - `start_time`, `end_time`, `location`, `status`, `created_at`, `updated_at`
  - 约束语义：`owner_id` 是事件所有者；会议场景中等于 organizer
- `user_relations`（表名固定为 `user_relations`）
  - `id`, `user_id`, `target_user_id`, `can_view_calendar`, `created_at`
  - 权限语义：`user_id=viewer`, `target_user_id=calendar owner`
- `meeting_attendees`
  - `id`, `meeting_id`, `user_id`, `role`, `status`, `created_at`, `updated_at`
  - `role`：`organizer|attendee`
  - `status`：`pending|accepted|rejected`

## 6. API 概览
- 健康检查
  - `GET /healthz`
- 认证
  - `POST /api/v1/auth/register`
  - `POST /api/v1/auth/login`
  - `GET /api/v1/me`（需 JWT）
- 关系与共享日历（需 JWT）
  - `POST /api/v1/relations`
  - `GET /api/v1/users/{id}/calendar?view=day|week&date=YYYY-MM-DD`
- 事件（需 JWT）
  - `POST /api/v1/events`
  - `PUT /api/v1/events/{id}`
  - `DELETE /api/v1/events/{id}`（逻辑删除：`status=cancelled`）
  - `GET /api/v1/events`
    - 支持：`include_cancelled=true|false`
    - 支持：`start_time_from`/`start_time_to`（RFC3339）
    - 支持：`view=day|week&date=YYYY-MM-DD`
  - `GET /api/v1/events/{id}`
- 会议（需 JWT）
  - `POST /api/v1/meetings`
  - `GET /api/v1/meetings/invitations`（当前用户仅 attendee 且默认 pending）
  - `POST /api/v1/meetings/{id}/accept`
  - `POST /api/v1/meetings/{id}/reject`

说明：
- 业务接口统一返回风格为 `{"message":"ok","data":...}`；错误返回 `{"message":"..."}`。
- JWT 中间件失败返回 `401`，字段为 `{"error":"..."}`（当前实现如此）。

## 7. 本地运行与测试
前置依赖：
- 已安装 Go（可执行 `go` 命令）
- MySQL 已启动（账号对 `MYSQL_DSN` 可连通）
- Redis 已启动（默认 `127.0.0.1:6379`）

环境变量：
- 项目启动会自动加载根目录 `.env`
- 可参考 `config/.env.example` 填写
- 关键项：`SERVER_PORT`、`MYSQL_DSN`、`REDIS_ADDR`、`REDIS_PASSWORD`、`REDIS_DB`、`JWT_SECRET`

Windows 推荐命令：
```powershell
# 1) 启动服务（含 /healthz 就绪探测，默认 15 秒）
powershell -ExecutionPolicy Bypass -File .\scripts\run-local.ps1

# 2) 一键冒烟（默认 KeepData）
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-test.ps1

# 3) 冒烟后清理（best-effort）
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-test.ps1 -Cleanup
```

可选直启：
```powershell
go run .\cmd\server\main.go
```

## 8. 示例演示流程
建议用于面试演示（约 3~5 分钟）：
1. 启动 MySQL 与 Redis，执行 `run-local.ps1`，展示 `healthz` 就绪。
2. 执行 `smoke-test.ps1`，展示 `PASS/FAIL` 汇总。
3. 说明关键业务链路（脚本已覆盖）：
   - 注册/登录 userA、userB
   - userA 创建事件并查询列表
   - 配置关系 `userB -> userA` 后，userB 查看 userA 共享日历
   - userA 发起会议邀请 userB
   - userB 查询邀请并接受
4. 如果要演示缓存，可在 Redis `MONITOR` 下重复请求 `GET /api/v1/events` 或共享日历接口观察读写行为与失效。

