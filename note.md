# 多人日程管理系统 (Schedule System) 设计与开发文档
## 1. 项目概述
### 1.1 项目背景
本项目为一个 **后端实习面试演示 Demo**，旨在实现一个支持多用户协作的日程管理系统。  
核心挑战不在于简单的 CRUD，而在于：
- 多用户在同一时间轴上的调度协调
- 会议邀请与状态流转
- 时间区间冲突检测
- 他人日程可见性控制
- 高频日程查询的缓存设计
### 1.2 支持的核心场景
- 两个账号登录并相互协作
- 查看彼此的日程忙闲状态
- 发起会议邀请
- 检测会议与已有日程是否冲突
- 接受或拒绝会议邀请
- 在日 / 周视图中查看自己的日程
### 1.3 技术栈
- **后端语言**：Golang
- **Web 框架**：Gin
- **ORM**：GORM
- **持久化**：MySQL
- **缓存**：Redis
- **认证**：JWT (JSON Web Token)
- **密码安全**：bcrypt
### 1.4 项目目标
1. **用户管理**：实现注册、登录、身份鉴权。
2. **日程管理**：支持课程、个人日程、会议三类事件。
3. **多维查看**：支持我的日 / 周视图、查看他人可见日程。
4. **协作系统**：支持发起 / 接受 / 拒绝会议。
5. **核心逻辑**：实现冲突检测、可见性过滤、Redis 缓存。
### 1.5 范围定义
#### 第一版 (MVP)
- 用户注册 / 登录
- JWT 鉴权
- 事件 CRUD
- 查看他人日程
- 会议邀请与状态流转
- 冲突检测
- 简单 Redis 缓存
- 自动建库与自动迁移
#### 后续计划
- 重复日程（RRULE）
- 邮件提醒
- WebSocket 实时通知
- 好友 / 通讯录系统
- 前端日历 UI
- 资源调度（会议室 / 教室）
- 空闲时间推荐
- 更细粒度权限控制
---
## 2. 系统整体功能分析
### 2.1 核心模块
#### 1）认证模块
负责用户身份管理与访问保护：
- 注册
- 登录
- JWT 鉴权
- 获取当前用户信息
#### 2）日程模块
负责用户事件的基础维护：
- 创建事件
- 修改事件
- 删除事件
- 查询我的事件列表
- 查询单个事件详情
#### 3）他人日程模块
负责跨用户查看日程：
- 用户关系权限判断
- 可见性过滤
- 忙碌状态展示
#### 4）会议模块
负责协作相关逻辑：
- 发起会议邀请
- 维护邀请列表
- 接受会议
- 拒绝会议
- 冲突检测
#### 5）缓存模块
负责性能优化：
- 我的日 / 周视图缓存
- 他人日程视图缓存
- 写操作触发缓存失效
---
### 2.2 第一版演示链路
推荐演示场景如下：
1. 用户 A 登录
2. 用户 A 创建自己的课程 / 个人日程
3. 用户 B 登录并创建自己的课程
4. 用户 A 查看用户 B 的可见日程
5. 用户 A 发起会议邀请
6. 如果时间冲突则提示失败
7. 如果不冲突则创建成功
8. 用户 B 接受或拒绝会议
9. 双方查看更新后的日程
---
### 2.3 非功能性目标
除功能外，本项目还需要具备一定工程质量：
- 启动流程稳定，可重复运行
- 支持自动建库、自动迁移
- 配置集中管理
- 项目分层清晰
- 返回结构统一
- 错误处理明确
- 便于继续扩展会议、关系、缓存等模块
---
## 3. 系统架构设计
### 3.1 分层结构
系统遵循标准 Golang 工程结构，实现职责分离：
- `cmd/server`：程序入口，负责资源初始化与服务启动
- `config`：配置文件加载逻辑
- `internal/api`：Handler 层，处理请求解析、调用 Service、返回 JSON
- `internal/service`：Service 层，处理核心业务逻辑
- `internal/dao`：DAO 层，封装数据库访问
- `internal/model`：Model 层，定义实体与常量
- `pkg`：通用工具库（JWT、密码加密、数据库初始化等）
---
### 3.2 每层职责
#### Handler 层
职责：
- 参数解析
- 请求校验
- 调用 Service
- 返回统一响应
- 不直接写复杂业务逻辑
#### Service 层
职责：
- 核心业务编排
- 冲突检测
- 权限判断
- 会议状态流转
- 缓存删除策略
#### DAO 层
职责：
- GORM / SQL 操作
- 数据库存取
- 基础查询封装
- 不承担业务判断
#### Model 层
职责：
- 定义表结构
- 定义枚举常量
- 提供统一实体模型
#### Package / Utility 层
职责：
- JWT 封装
- 密码加密
- 数据库初始化
- Redis 初始化
- 中间件
- 统一响应结构
---
### 3.3 启动流程
系统启动流程如下：
1. 加载配置
2. 初始化 MySQL（自动建库）
3. 执行 AutoMigrate（表结构同步）
4. 初始化 Redis
5. 初始化路由与依赖注入
6. 启动 HTTP 服务
可表示为：
```text
加载配置
   ↓
初始化 MySQL（自动建库）
   ↓
执行 AutoMigrate
   ↓
初始化 Redis
   ↓
初始化 Router / Handler / Service / DAO
   ↓
启动 HTTP 服务
```

### 3.4 系统架构图

可表示为：

```text
             +----------------------+
             |      Mobile App      |
             +----------------------+
                        |
                      HTTP
                        v
             +----------------------+
             |   API Gateway / Go   |
             +----------------------+
                 /              \
                v                v
      +----------------+  +----------------+
      |  Feed Service  |  |  User Service  |
      +----------------+  +----------------+
           /      \              /      \
          v        v            v        v
 +---------------+ +---------------+ +---------------+ +---------------+
 |  Redis Cache  | |  Redis Inbox  | |  MySQL User   | |  MySQL Posts  |
 +---------------+ +---------------+ +---------------+ +---------------+
```

### 3.5 当前架构目标
当前架构目标包括：

- 保持 Handler -> Service -> DAO 单向调用链
- 配置与业务逻辑分离
- 数据访问与业务判断分离
- 启动流程独立清晰
- 为后续 event / meeting / relation / cache 扩展留出空间

### 3.6 当前架构验收标准

- 服务可以正常启动
- MySQL 可以连接
- 数据库不存在时自动创建
- AutoMigrate 成功执行
- Redis 可以连接
- Gin 路由注册成功
- /healthz 可访问
- 项目结构分层清晰
- 无明显跨层耦合

## 4. 数据库设计

### 4.1 设计原则
数据库设计遵循以下原则：

- 统一抽象
- 将课程、个人日程、会议统一抽象为 event
- 最小必要模型
- 第一版只保留 MVP 必需表，避免过度设计
- 便于冲突检测
- 围绕时间区间查询建立统一模型
- 便于扩展
- 为参与者、权限、缓存留出演进空间

### 4.2 用户表 users
用于存储用户登录与展示信息。
字段设计

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| id | BIGINT | 用户唯一 ID |
| username | VARCHAR(64) | 登录账号（唯一） |
| password_hash | VARCHAR(255) | bcrypt 加密后的密码 |
| nickname | VARCHAR(64) | 昵称 |
| email | VARCHAR(128) | 邮箱，可选 |
| status | TINYINT | 账号状态 |
| created_at | DATETIME | 注册时间 |
| updated_at | DATETIME | 更新时间 |

设计说明
username 唯一，用于登录
password_hash 不保存明文密码
status 预留禁用 / 封禁能力

### 4.3 统一事件表 events
支持三种类型：
course（课程）
personal（个人）
meeting（会议）
字段设计

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| id | BIGINT | 事件 ID |
| owner_id | BIGINT | 所属用户 ID |
| title | VARCHAR(128) | 标题 |
| description | TEXT | 描述 |
| event_type | VARCHAR(32) | 事件类型 |
| start_time | DATETIME | 开始时间 |
| end_time | DATETIME | 结束时间 |
| visibility | VARCHAR(32) | 可见性 |
| status | VARCHAR(32) | 事件状态 |
| location | VARCHAR(128) | 地点 |
| source | VARCHAR(32) | 来源 |
| recurrence_rule | VARCHAR(255) | 重复规则，预留 |
| created_by | BIGINT | 创建人 |
| created_at | DATETIME | 创建时间 |
| updated_at | DATETIME | 更新时间 |

设计说明
课程、个人事项、会议统一抽象成 event
统一查询逻辑
统一冲突检测逻辑
统一缓存视图逻辑
当前第一版重要字段
第一版实际最关键字段：
id
owner_id
title
event_type
start_time
end_time
visibility
status
### 4.4 会议参与者表 event_attendees
该表仅用于会议类事件。
字段设计

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| id | BIGINT | 主键 ID |
| event_id | BIGINT | 关联的会议 ID |
| user_id | BIGINT | 参与者 ID |
| role | VARCHAR(32) | 角色 |
| response_status | VARCHAR(32) | 响应状态 |
| created_at | DATETIME | 创建时间 |
| updated_at | DATETIME | 更新时间 |

字段语义
role
organizer
attendee
response_status
pending
accepted
rejected
设计说明
一个会议可以有多个参与者
发起人和被邀请人统一存储
便于后续扩展多人会议

### 4.5 用户关系表 user_relations
用于控制用户之间是否允许查看日程。
字段设计

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| id | BIGINT | 主键 ID |
| user_id | BIGINT | 当前用户 |
| target_user_id | BIGINT | 目标用户 |
| can_view_calendar | TINYINT | 是否允许查看 |
| created_at | DATETIME | 创建时间 |

设计说明
表示 user_id 是否可以查看 target_user_id 的日程
第一版只实现简单布尔权限
后续可扩展为更多权限级别

### 4.6 表关系
users
 ├── 1:N events
 │        └── 1:N event_attendees
 └── 1:N user_relations

### 4.7 当前推荐建表 SQL（核心示例）
users
```sql
CREATE TABLE users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(64) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    nickname VARCHAR(64) NOT NULL,
    email VARCHAR(128) DEFAULT NULL,
    status TINYINT NOT NULL DEFAULT 1 COMMENT '1-active, 0-disabled',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

events
```sql
CREATE TABLE events (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    owner_id BIGINT NOT NULL COMMENT '事件所属用户',
    title VARCHAR(128) NOT NULL,
    description TEXT DEFAULT NULL,
    event_type VARCHAR(32) NOT NULL COMMENT 'course/personal/meeting',
    visibility VARCHAR(32) NOT NULL DEFAULT 'private' COMMENT 'private/busy_only/public',
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    location VARCHAR(128) DEFAULT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active' COMMENT 'active/cancelled',
    source VARCHAR(32) NOT NULL DEFAULT 'self_created' COMMENT 'self_created/meeting_invite',
    recurrence_rule VARCHAR(255) DEFAULT NULL COMMENT '先预留，第一版可不做',
    created_by BIGINT NOT NULL COMMENT '谁创建的',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT chk_event_time CHECK (end_time > start_time),
    INDEX idx_owner_start (owner_id, start_time),
    INDEX idx_owner_end (owner_id, end_time),
    INDEX idx_owner_status_start (owner_id, status, start_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

event_attendees
```sql
CREATE TABLE event_attendees (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    event_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    role VARCHAR(32) NOT NULL DEFAULT 'attendee' COMMENT 'organizer/attendee',
    response_status VARCHAR(32) NOT NULL DEFAULT 'pending' COMMENT 'pending/accepted/rejected',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_event_user (event_id, user_id),
    INDEX idx_user_status (user_id, response_status),
    INDEX idx_event_id (event_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

user_relations
```sql
CREATE TABLE user_relations (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    target_user_id BIGINT NOT NULL,
    can_view_calendar TINYINT NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_target (user_id, target_user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

## 5. 核心业务规则

### 5.1 冲突检测规则
判定两个时间区间 [S1, E1] 与 [S2, E2] 是否存在交集。
冲突判定条件：
NewStart < OldEnd AND NewEnd > OldStart
解释
只要满足上述条件，说明两个事件在时间上有重叠，即判定为冲突。

### 5.2 参与冲突检测的事件
以下事件都参与冲突检测：
课程（course）
个人日程（personal）
会议（meeting）
并且要求：
status = active
特别说明
会议在不同状态下可采用不同策略：
对组织者自己的会议：创建时即参与冲突检测
对被邀请人：只有接受时再次检查并确认是否冲突

### 5.3 会议邀请规则
发起时
组织者状态自动设为 accepted
被邀请人状态设为 pending
接受时
系统必须再次执行冲突检测：
若通过：更新为 accepted
若失败：拒绝接受
拒绝时
更新状态为 rejected
### 5.4 可见性规则

| 类型 | 表现行为 |
| :--- | :--- |
| private | 他人完全不可见该日程 |
| busy_only | 他人仅能看到该时段显示为 “Busy” |
| public | 他人可以看到完整信息 |

### 5.5 事件创建与删除规则
创建事件
必须携带起止时间
必须校验 end_time > start_time
必须校验所属用户身份
删除事件
第一版建议采用以下任一策略：
逻辑删除：更新 status = cancelled
物理删除：直接删除
推荐第一版优先采用逻辑删除，便于保留记录。

## 6. API 设计

### 6.0 接口总览

| 方法 | 路径 | 说明 |
| :--- | :--- | :--- |
| POST | /api/v1/auth/register | 注册 |
| POST | /api/v1/auth/login | 登录 |
| GET | /api/v1/me | 获取当前用户 |
| POST | /api/v1/events | 创建事件 |
| PUT | /api/v1/events/{id} | 修改事件 |
| DELETE | /api/v1/events/{id} | 删除事件 |
| GET | /api/v1/events | 查询我的事件 |
| GET | /api/v1/events/{id} | 查询事件详情 |
| GET | /api/v1/users/{id}/calendar | 查看他人日程 |
| POST | /api/v1/meetings | 发起会议邀请 |
| GET | /api/v1/meetings/invitations | 查看收到的邀请 |
| POST | /api/v1/meetings/{id}/accept | 接受会议邀请 |
| POST | /api/v1/meetings/{id}/reject | 拒绝会议邀请 |
| GET | /api/v1/meetings/{id} | 查看会议详情 |
| POST | /api/v1/relations | 设置关系权限 |
| GET | /api/v1/relations | 查看关系配置 |

### 6.1 认证模块

- POST /api/v1/auth/register：注册用户。

请求体示例：

```json
{
  "username": "alice",
  "password": "123456",
  "nickname": "Alice"
}
```

- POST /api/v1/auth/login：用户登录。

请求体示例：

```json
{
  "username": "alice",
  "password": "123456"
}
```

返回体示例：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "user_id": 1,
    "username": "alice",
    "token": "jwt-token"
  }
}
```

- GET /api/v1/me：获取当前用户信息。

### 6.2 事件 CRUD 模块

- POST /api/v1/events：创建事件。

请求体示例：

```json
{
  "title": "Data Structure",
  "description": "chapter 3",
  "event_type": "course",
  "visibility": "public",
  "start_time": "2026-03-24 08:00:00",
  "end_time": "2026-03-24 10:00:00",
  "location": "Room 101"
}
```

- PUT /api/v1/events/{id}：修改事件。

请求体示例：

```json
{
  "title": "Advanced Data Structure",
  "location": "Room 102"
}
```

- DELETE /api/v1/events/{id}：删除事件。
- GET /api/v1/events：获取我的日程列表。

参数建议：

```text
view=day|week
date=2026-03-24
```

- GET /api/v1/events/{id}：获取单个事件详情。

### 6.3 查看他人日程模块

- GET /api/v1/users/{id}/calendar：查看他人的公开 / 忙碌日程。
- 返回结果需按 visibility 做过滤。

### 6.4 会议模块

- POST /api/v1/meetings：发起会议邀请。

请求体示例：

```json
{
  "title": "Project Discussion",
  "description": "Discuss backend design",
  "start_time": "2026-03-24 11:30:00",
  "end_time": "2026-03-24 12:00:00",
  "location": "Online",
  "attendee_ids": [2],
  "visibility": "public"
}
```

- GET /api/v1/meetings/invitations：查看收到的会议邀请列表。
- POST /api/v1/meetings/{id}/accept：接受会议邀请。
- POST /api/v1/meetings/{id}/reject：拒绝会议邀请。
- GET /api/v1/meetings/{id}：查看会议详情。

### 6.5 用户关系模块

- POST /api/v1/relations：设置用户关系权限。

请求体示例：

```json
{
  "target_user_id": 2,
  "can_view_calendar": true
}
```

- GET /api/v1/relations：查看当前用户的关系配置。

### 6.6 统一响应格式

成功：

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

失败：

```json
{
  "code": 40001,
  "message": "time conflict",
  "data": null
}
```
## 7. Redis 设计
### 7.1 缓存目标
Redis 主要缓存：
我的日程日 / 周视图
他人可见日程视图
### 7.2 缓存 Key 设计
我的日程
schedule:user:{user_id}:day:{date}
schedule:user:{user_id}:week:{date}
例如：
schedule:user:1:day:2026-03-24
schedule:user:2:week:2026-03-24
他人视图
calendar:view:{viewer_id}:{target_user_id}:{view}:{date}
例如：
calendar:view:1:2:week:2026-03-24
### 7.3 缓存一致性策略
采用主动失效策略。
在以下写操作发生后，需要删除相关缓存：
创建事件
修改事件
删除事件
发起会议邀请
接受会议
拒绝会议
修改可见性
### 7.4 当前缓存策略说明
第一版只做：
简单缓存
写后删缓存
不做复杂更新同步
这样实现简单，便于 demo 展示与面试说明。
## 8. 阶段开发计划
第一阶段：基础工程（已完成）
目标：
项目骨架搭建
配置加载
MySQL 连接
自动建库
AutoMigrate
Redis 连接
路由注册
HTTP Server 启动
状态：
 已完成

第二阶段：认证系统
目标：
用户注册
用户登录
JWT 中间件
获取当前用户信息
统一未授权返回
状态：
 进行中 / 待完成

第三阶段：核心 CRUD
目标：
创建事件
修改事件
删除事件
查询我的日程
查询单个事件详情
状态：
 待完成

第四阶段：他人日程模块
目标：
用户关系权限
可见性过滤
查看他人日程接口
状态：
 待完成

第五阶段：会议系统
目标：
发起会议
邀请列表
接受会议
拒绝会议
冲突检测
状态：
 待完成

第六阶段：缓存优化
目标：
缓存我的日程
缓存他人日程
写操作删缓存
状态：
 待完成
## 9. 验收清单
### 9.1 启动检查
 配置加载正常
 MySQL 可以连接
 数据库不存在时自动创建
 AutoMigrate 正常执行
 Redis 可以连接
 服务能成功启动
 /healthz 可访问
### 9.2 接口检查
当前目标：
 注册接口可用
 登录接口可用
 /me 能正确返回当前用户
 JWT 未通过时返回 401
 接口响应结构统一
### 9.3 架构检查
 Handler -> Service -> DAO 调用链基本清晰
 配置、数据库、Redis 初始化职责已分开
 启动链路合理
 认证模块完整
 业务模块完整
 缓存模块完整
## 10. 每一阶段预计使用的 Prompt
### 10.1 第一阶段：基础工程
#### Prompt 1：生成基础骨架
Create a Go backend project named schedule-system.
Requirements:
- Use Gin framework
- Use MySQL
- Use Redis
- Use JWT authentication
- Use layered architecture: handler/service/dao
- Follow standard Go project layout
Create folders:
cmd/server
internal/api
internal/service
internal/dao
internal/model
pkg
config
Generate:
- main.go
- router.go
- basic server startup code
#### Prompt 2：生成数据库与 Redis 连接代码
Generate MySQL and Redis connection code.
Requirements:
- Use GORM
- MySQL connection
- Redis using go-redis v9
- Singleton pattern
- Files:
  internal/dao/db.go
  internal/dao/redis.go
#### Prompt 3：支持 .env 读取
Add .env support to config loading.
Use github.com/joho/godotenv.
Load .env at startup before os.Getenv.
Modify config.Load() only.
#### Prompt 4：自动建库与迁移
/plan
I want to add automatic database initialization to this Go backend project.
Expected behavior:
1. Read MYSQL_DSN from config
2. If the target database does not exist, create it automatically
3. Reconnect to the target database
4. Run GORM AutoMigrate for existing models
5. Start the HTTP server normally
Constraints:
- keep current project structure
- use minimal changes
- do not modify unrelated business logic
- do not change API routes or handler behavior
- add simple startup logs for each step
- if MYSQL_DSN contains a database name, handle bootstrap safely
Please first show:
- which files you will modify
- your implementation plan
- any risk or edge case you see
#### Prompt 5：基础工程 Review
Review the current project structure and architecture.
Please check:
1. whether the layered architecture is clean and consistent
2. whether startup flow is reasonable
3. whether config, database, redis, router, handler, service, dao separation is correct
4. whether there are any obvious structural issues before I continue implementing business features
Do not modify code yet.
Just give me a review summary with:
- what is good
- what should be improved now
- what can wait until later
### 10.2 第二阶段：认证系统
#### Prompt 1：认证模块计划
/plan
Implement authentication module for this project.
Requirements:
- POST /api/v1/auth/register
- POST /api/v1/auth/login
- GET /api/v1/me
- use bcrypt for password hashing
- use JWT for authentication
- keep current layered architecture
- keep changes minimal
- do not modify unrelated code
Please first show:
- which files you will modify
- implementation plan
- request and response design
#### Prompt 2：认证模块实现
Implement it now.
Requirements:
- keep changes minimal
- preserve current project structure
- return consistent JSON responses
- unauthorized requests should return 401 instead of panic
#### Prompt 3：认证模块验收
Review the authentication module implementation.
Please check:
1. whether register/login/me are complete
2. whether bcrypt is used correctly
3. whether JWT middleware is correct
4. whether unauthorized requests return proper JSON
5. whether the layering is still clean
Do not modify code yet.
Only give me a review summary.
### 10.3 第三阶段：事件 CRUD
#### Prompt 1：事件 CRUD 计划
/plan
Implement event CRUD for this project.
Requirements:
- create event
- update event
- delete event
- list my events by day or week
- get event detail
- use existing auth flow
- keep current layered architecture
- keep changes minimal
- do not modify unrelated code
Please first show:
- which files you will modify
- implementation plan
- request and response design
#### Prompt 2：补充冲突检测
Implement time conflict detection for event creation and update.
Rules:
- active events participate in conflict checking
- use overlap rule:
  new_start < old_end AND new_end > old_start
Requirements:
- keep changes minimal
- use current layered architecture
- return clear error when conflict occurs
#### Prompt 3：事件模块验收
Review the event CRUD implementation.
Please check:
1. whether create/update/delete/list/detail are complete
2. whether conflict detection is correct
3. whether status and visibility are handled properly
4. whether service and dao responsibilities are clean
Do not modify code yet.
Only provide a review summary.
### 10.4 第四阶段：他人日程模块
#### Prompt 1：查看他人日程计划
/plan
Implement shared calendar viewing for this project.
Requirements:
- GET /api/v1/users/{id}/calendar
- enforce user relationship permission
- apply visibility filtering
- support day/week query
- keep current layered architecture
- minimal changes
Please first show:
- files to modify
- implementation plan
- request and response design
#### Prompt 2：可见性过滤逻辑
Implement calendar visibility filtering.
Rules:
- private: not visible
- busy_only: only show busy time
- public: show full details
Requirements:
- keep business rules in service layer
- keep handler thin
- return filtered calendar view
#### Prompt 3：他人日程模块验收
Review the shared calendar implementation.
Please check:
1. whether permission check exists
2. whether visibility filtering is correct
3. whether private/busy_only/public are handled correctly
4. whether the layer separation remains clean
Do not modify code yet.
Only provide a review summary.
### 10.5 第五阶段：会议系统
#### Prompt 1：会议模块计划
/plan
Implement meeting invitation module.
Requirements:
- POST /api/v1/meetings
- GET /api/v1/meetings/invitations
- POST /api/v1/meetings/{id}/accept
- POST /api/v1/meetings/{id}/reject
- organizer accepted by default
- attendees pending by default
- run conflict detection when accepting invitation
- keep current layered architecture
- minimal changes
Please first show:
- files to modify
- implementation plan
- request and response design
#### Prompt 2：会议状态流转实现
Implement meeting invitation state transitions.
Requirements:
- organizer -> accepted
- attendee -> pending initially
- accept -> recheck time conflict, then accepted
- reject -> update to rejected
- clear JSON errors
- minimal changes
#### Prompt 3：会议模块验收
Review the meeting invitation module.
Please check:
1. whether invitation flow is complete
2. whether accept/reject logic is correct
3. whether conflict recheck on accept is implemented
4. whether organizer and attendee statuses are set correctly
5. whether layering remains clean
Do not modify code yet.
Only provide a review summary.
### 10.6 第六阶段：Redis 缓存
#### Prompt 1：缓存计划
/plan
Add Redis cache for calendar views.
Requirements:
- cache my day/week schedule
- cache shared calendar views
- define clear cache keys
- delete related cache on write operations
- keep current architecture
- minimal changes
Please first show:
- files to modify
- implementation plan
- cache key design
- invalidation strategy
#### Prompt 2：缓存实现
Implement Redis caching for calendar views.
Requirements:
- cache keys:
  schedule:user:{user_id}:day:{date}
  schedule:user:{user_id}:week:{date}
  calendar:view:{viewer_id}:{target_user_id}:{view}:{date}
- invalidate cache on:
  create event
  update event
  delete event
  create meeting
  accept meeting
  reject meeting
- keep changes minimal
#### Prompt 3：缓存模块验收
Review the Redis cache implementation.
Please check:
1. whether cache keys are reasonable
2. whether cache invalidation is correct
3. whether read/write paths are clean
4. whether cache logic is not leaking into unrelated layers
Do not modify code yet.
Only provide a review summary.
## 11. 面试讲解模板
我实现了一个基于 Go + MySQL + Redis 的多人日程管理系统。
在 建模 上，我采用了“统一事件模型”，将课程、会议、个人事项统一抽象为 event；
在 业务 上，我实现了一套基于时间区间重叠的冲突检测规则，并用于会议预约与接受时的校验；
在 协作 上，系统支持查看他人可见日程、发起会议邀请以及会议状态流转；
在 工程 上，我采用了标准的 Handler-Service-DAO 分层架构，并引入 Redis 对日程视图进行缓存，通过主动失效策略平衡了一致性与查询性能；
在 启动流程 上，系统支持自动建库与自动迁移，提升了项目初始化效率和工程完整性。
