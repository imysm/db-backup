# Phase 3: 数据库连接测试 + 任务模板 + 恢复安全 — 产品设计文档

> 版本: v1.0 | 日期: 2026-03-27 | 状态: 设计中

## 1. 产品概述

Phase 3 围绕三个核心痛点展开：

| 模块 | 解决的问题 | 核心价值 |
|------|-----------|---------|
| **数据库连接测试** | 创建任务后才发现连接失败，反复试错 | 创建前验证，减少无效任务 |
| **备份任务模板** | 每次创建任务都要重复填写大量配置 | 一键复用，标准化配置 |
| **恢复安全加固** | 恢复操作高风险但缺乏保护机制 | 多重校验，防止误操作 |

---

## 2. 数据库连接测试

### 2.1 功能描述

在创建/编辑备份任务时，用户可以随时测试数据库连接是否可用，无需保存任务即可验证配置正确性。支持直连和 SSH 隧道两种方式。

### 2.2 API 设计

#### POST /api/v1/jobs/test-connection

测试数据库连接。

**请求体**：

```json
{
  "database_type": "mysql",
  "host": "192.168.1.100",
  "port": 3306,
  "username": "backup_user",
  "password": "secret123",
  "database": "mydb",
  "ssh_tunnel": {
    "enabled": false
  }
}
```

带 SSH 隧道的请求：

```json
{
  "database_type": "postgres",
  "host": "10.0.0.1",
  "port": 5432,
  "username": "readonly",
  "password": "secret123",
  "database": "production",
  "ssh_tunnel": {
    "enabled": true,
    "ssh_host": "jump.example.com",
    "ssh_port": 22,
    "ssh_user": "tunnel_user",
    "ssh_password": "ssh_secret",
    "ssh_private_key": "-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----"
  }
}
```

**成功响应（200）**：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "connected": true,
    "latency_ms": 23,
    "database_version": "MySQL 8.0.35",
    "database_list": ["mydb", "test", "information_schema"],
    "server_info": {
      "host": "192.168.1.100",
      "os": "Linux",
      "charset": "utf8mb4"
    }
  }
}
```

**失败响应（200，connected=false）**：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "connected": false,
    "error_code": "AUTH_FAILED",
    "error_message": "Access denied for user 'backup_user'@'192.168.1.50' (using password: YES)",
    "suggestion": "请检查用户名和密码是否正确，确认该用户有远程访问权限"
  }
}
```

**请求参数错误（400）**：

```json
{
  "code": 400,
  "message": "无效的数据库类型，支持: mysql, postgres, mongodb, sqlserver, oracle"
}
```

**错误码映射**：

| error_code | 含义 | suggestion |
|-----------|------|-----------|
| `AUTH_FAILED` | 认证失败 | 请检查用户名和密码是否正确 |
| `TIMEOUT` | 连接超时 | 请检查网络连通性和防火墙设置 |
| `NETWORK_UNREACHABLE` | 网络不可达 | 请检查目标主机是否在线 |
| `DB_NOT_FOUND` | 数据库不存在 | 请检查数据库名称是否正确 |
| `PERMISSION_DENIED` | 权限不足 | 请确认用户有访问该数据库的权限 |
| `SSH_TUNNEL_FAILED` | SSH 隧道建立失败 | 请检查 SSH 连接配置 |
| `SSL_ERROR` | SSL/TLS 错误 | 请检查 SSL 证书配置 |
| `UNKNOWN` | 未知错误 | 请查看系统日志获取详细信息 |

### 2.3 后端实现方案

**Handler 层**：

```go
// handler/job.go

type TestConnectionRequest struct {
    DatabaseType string         `json:"database_type" binding:"required,oneof=mysql postgres mongodb sqlserver oracle"`
    Host         string         `json:"host" binding:"required"`
    Port         int            `json:"port" binding:"required,min=1,max=65535"`
    Username     string         `json:"username" binding:"required"`
    Password     string         `json:"password"`
    Database     string         `json:"database"`
    SSHTunnel    *SSHTunnelConfig `json:"ssh_tunnel"`
}

type TestConnectionResult struct {
    Connected      bool           `json:"connected"`
    LatencyMs      int            `json:"latency_ms,omitempty"`
    DatabaseVersion string        `json:"database_version,omitempty"`
    DatabaseList   []string       `json:"database_list,omitempty"`
    ServerInfo     *ServerInfo    `json:"server_info,omitempty"`
    ErrorCode      string         `json:"error_code,omitempty"`
    ErrorMessage   string         `json:"error_message,omitempty"`
    Suggestion     string         `json:"suggestion,omitempty"`
}
```

**各数据库连接测试实现**：

```go
// internal/db/tester/tester.go

type ConnectionTester interface {
    TestConnection(ctx context.Context, cfg *TestConfig) (*TestResult, error)
}

// MySQL: 使用 go-sql-driver/mysql，执行 SELECT VERSION()
// PostgreSQL: 使用 pgx，执行 SELECT version()
// MongoDB: 使用 mongo-go-driver，执行 db.runCommand({ ping: 1 })
// SQL Server: 使用 go-mssqldb，执行 SELECT @@VERSION
// Oracle: 使用 godror，执行 SELECT * FROM v$version
```

**连接超时控制**：

- 连接超时：5 秒（可配置）
- 查询超时：3 秒
- 总超时：10 秒
- SSH 隧道建立超时：10 秒

**路由注册**：

```go
// router/router.go — jobs 组内新增
jobs.POST("/test-connection", jobHandler.TestConnection)
```

### 2.4 前端交互

**创建/编辑任务表单中的测试连接按钮**：

```
┌─────────────────────────────────────────────────────────┐
│  数据库连接配置                                          │
│                                                         │
│  数据库类型    [MySQL ▼]                                  │
│                                                         │
│  主机地址      [192.168.1.100        ]                   │
│  端口          [3306                 ]                   │
│  用户名        [backup_user          ]                   │
│  密码          [•••••••••            ]                   │
│  数据库名      [mydb                 ]                   │
│                                                         │
│  ┌─ SSH 隧道 ─────────────────────────────────────────┐  │
│  │  □ 启用 SSH 隧道                                  │  │
│  │    SSH 主机  [jump.example.com  ]                  │  │
│  │    SSH 端口  [22                  ]                  │  │
│  │    SSH 用户  [tunnel_user        ]                  │  │
│  │    SSH 密码  [•••••••            ]                  │  │
│  └────────────────────────────────────────────────────┘  │
│                                                         │
│  [🔄 测试连接]                              ✅ 连接成功  │
│                                                         │
│  ┌─ 连接信息 ─────────────────────────────────────────┐  │
│  │  延迟: 23ms  版本: MySQL 8.0.35                    │  │
│  │  可用数据库: mydb, test, information_schema        │  │
│  └────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

**按钮状态流转**：

| 状态 | 按钮文字 | 图标 | 可点击 |
|------|---------|------|-------|
| 默认 | 测试连接 | 🔌 | 是 |
| 测试中 | 测试中... | 🔄 (旋转) | 否 |
| 成功 | 测试连接 | ✅ 绿色勾 | 是 |
| 失败 | 测试连接 | ❌ 红色叉 | 是 |

**失败时的错误提示**：

```
┌─ 连接失败 ───────────────────────────────────────────┐
│  ❌ 认证失败                                          │
│  Access denied for user 'backup_user'@'192.168.1.50' │
│                                                      │
│  💡 建议：请检查用户名和密码是否正确，确认该用户       │
│     有远程访问权限                                    │
└──────────────────────────────────────────────────────┘
```

### 2.5 权限要求

| 操作 | 所需权限 |
|------|---------|
| 测试连接 | `backup_job:create` 或 `backup_job:update` |

---

## 3. 备份任务模板

### 3.1 功能描述

将常用的备份配置保存为模板，创建任务时可直接从模板填充，减少重复配置工作。系统内置各数据库类型的默认模板，用户也可创建自定义模板。

### 3.2 模板定义

**内置默认模板**：

| 模板名称 | 数据库类型 | Cron | 保留天数 | 备份类型 | 压缩 | 加密 |
|---------|-----------|------|---------|---------|------|------|
| MySQL 每日全量 | mysql | `0 2 * * *` | 7 | full | 是 | 否 |
| MySQL 每小时增量 | mysql | `0 * * * *` | 3 | incremental | 是 | 否 |
| PostgreSQL 每日全量 | postgres | `0 3 * * *` | 7 | full | 是 | 否 |
| MongoDB 每日全量 | mongodb | `0 2 * * *` | 7 | full | 是 | 否 |
| SQL Server 每日全量 | sqlserver | `0 2 * * *` | 7 | full | 是 | 否 |
| Oracle 每日全量 | oracle | `0 2 * * *` | 7 | full | 是 | 否 |

### 3.3 API 设计

#### GET /api/v1/templates

获取模板列表。

**请求参数（Query）**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| database_type | string | 否 | 按数据库类型筛选 |
| is_builtin | bool | 否 | 是否内置模板 |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页数量，默认 20 |

**响应（200）**：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "total": 6,
    "items": [
      {
        "id": 1,
        "name": "MySQL 每日全量",
        "database_type": "mysql",
        "description": "MySQL 数据库每日凌晨 2 点全量备份，保留 7 天",
        "schedule": "0 2 * * *",
        "retention_days": 7,
        "backup_type": "full",
        "compress": true,
        "encrypt": false,
        "storage_type": "local",
        "is_builtin": true,
        "created_at": "2026-03-27T00:00:00+08:00"
      }
    ]
  }
}
```

#### POST /api/v1/templates

创建自定义模板。

**请求体**：

```json
{
  "name": "生产 MySQL 全量备份",
  "database_type": "mysql",
  "description": "生产环境 MySQL 每日全量备份，保留 30 天，压缩加密",
  "schedule": "0 2 * * *",
  "retention_days": 30,
  "backup_type": "full",
  "compress": true,
  "encrypt": true,
  "storage_type": "s3",
  "storage_config": {
    "endpoint": "s3.example.com",
    "bucket": "db-backups"
  },
  "notify_on_success": false,
  "notify_on_fail": true,
  "notify_channels": ["email", "dingtalk"]
}
```

**响应（201）**：

```json
{
  "code": 0,
  "message": "创建成功",
  "data": {
    "id": 10,
    "name": "生产 MySQL 全量备份",
    "database_type": "mysql",
    "is_builtin": false,
    "created_at": "2026-03-27T12:00:00+08:00"
  }
}
```

#### GET /api/v1/templates/:id

获取模板详情。

**响应（200）**：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": 1,
    "name": "MySQL 每日全量",
    "database_type": "mysql",
    "description": "MySQL 数据库每日凌晨 2 点全量备份，保留 7 天",
    "schedule": "0 2 * * *",
    "retention_days": 7,
    "backup_type": "full",
    "compress": true,
    "encrypt": false,
    "storage_type": "local",
    "storage_config": null,
    "notify_on_success": false,
    "notify_on_fail": true,
    "notify_channels": null,
    "is_builtin": true,
    "created_by": null,
    "created_at": "2026-03-27T00:00:00+08:00",
    "updated_at": "2026-03-27T00:00:00+08:00"
  }
}
```

#### PUT /api/v1/templates/:id

更新模板。内置模板不可修改。

#### DELETE /api/v1/templates/:id

删除模板。内置模板不可删除。

**响应（403）**：

```json
{
  "code": 403,
  "message": "内置模板不允许删除"
}
```

#### POST /api/v1/jobs/from-template/:id

从模板创建任务。

**请求体**（模板未覆盖的字段）：

```json
{
  "name": "生产库 - app_db",
  "host": "10.0.0.5",
  "port": 3306,
  "username": "backup_user",
  "password": "secret123",
  "database": "app_db"
}
```

**响应（201）**：返回创建的 BackupJob 对象（同 POST /api/v1/jobs 响应格式）。

### 3.4 数据库设计

```sql
CREATE TABLE backup_templates (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name            VARCHAR(128)    NOT NULL COMMENT '模板名称',
    database_type   VARCHAR(20)     NOT NULL COMMENT '数据库类型',
    description     VARCHAR(512)    DEFAULT '' COMMENT '模板描述',

    -- 备份策略
    schedule        VARCHAR(64)     NOT NULL COMMENT 'Cron 表达式',
    retention_days  INT             DEFAULT 7 COMMENT '保留天数',
    backup_type     VARCHAR(20)     DEFAULT 'full' COMMENT '备份类型: full/incremental',

    -- 存储
    storage_type    VARCHAR(20)     DEFAULT 'local' COMMENT '存储类型',
    storage_config  JSON            NULL COMMENT '存储配置',

    -- 压缩加密
    compress        TINYINT(1)      DEFAULT 1 COMMENT '是否压缩',
    encrypt         TINYINT(1)      DEFAULT 0 COMMENT '是否加密',

    -- 通知
    notify_on_success TINYINT(1)    DEFAULT 0,
    notify_on_fail    TINYINT(1)    DEFAULT 1,
    notify_channels   JSON          NULL COMMENT '通知渠道',

    -- 元数据
    is_builtin      TINYINT(1)      DEFAULT 0 COMMENT '是否内置模板',
    created_by      BIGINT UNSIGNED NULL COMMENT '创建人 ID',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at      DATETIME        NULL,

    INDEX idx_database_type (database_type),
    INDEX idx_is_builtin (is_builtin)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='备份任务模板';
```

### 3.5 前端交互

**创建任务时的模板选择**：

```
┌─────────────────────────────────────────────────────────┐
│  创建备份任务                                           │
│                                                         │
│  ┌─ 选择模板 ─────────────────────────────────────────┐  │
│  │  [不使用模板 ▼]                                     │  │
│  │  ┌──────────────────────────────────────────────┐  │  │
│  │  │ 📋 MySQL 每日全量              MySQL  内置   │  │  │
│  │  │ 📋 MySQL 每小时增量            MySQL  内置   │  │  │
│  │  │ 📋 生产 MySQL 全量备份         MySQL  自定义 │  │  │
│  │  │ 📋 PostgreSQL 每日全量        PG     内置   │  │  │
│  │  │ 📋 MongoDB 每日全量           Mongo  内置   │  │  │
│  │  └──────────────────────────────────────────────┘  │  │
│  └────────────────────────────────────────────────────┘  │
│                                                         │
│  ── 以下字段已从模板自动填充（可修改） ──                 │
│                                                         │
│  任务名称      [生产库 - app_db                          ]
│  数据库类型    [MySQL ▼]         ← 模板锁定，不可修改     │
│  Cron 表达式   [0 2 * * *]                              │
│  保留天数      [7                                         ]
│  备份类型      [全量备份 ▼]                               │
│  ...                                                    │
│                                                         │
│  [取消]  [创建任务]                                      │
└─────────────────────────────────────────────────────────┘
```

**模板管理页面**：

```
┌─────────────────────────────────────────────────────────────┐
│  🗄️ DB Backup          [仪表盘] [任务] [模板] [设置]        │
│                                                             │
│  [+ 新建模板]                    筛选: [全部类型 ▼] [全部 ▼] │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ 模板名称          │ 类型    │ Cron      │ 保留 │ 类型  │  │
│  ├───────────────────┼─────────┼───────────┼──────┼──────┤  │
│  │ MySQL 每日全量     │ MySQL   │ 0 2 * * * │  7天 │ 内置  │  │
│  │ MySQL 每小时增量   │ MySQL   │ 0 * * * * │  3天 │ 内置  │  │
│  │ PG 每日全量        │ PG      │ 0 3 * * * │  7天 │ 内置  │  │
│  │ 生产 MySQL 全量    │ MySQL   │ 0 2 * * * │ 30天 │ 自定义│  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│  共 6 个模板                                    第 1/1 页   │
└─────────────────────────────────────────────────────────────┘
```

### 3.6 权限要求

| 操作 | 所需权限 |
|------|---------|
| 查看模板 | 登录即可 |
| 创建/编辑/删除自定义模板 | `template:manage` |
| 从模板创建任务 | `backup_job:create` |

---

## 4. 恢复安全加固

### 4.1 功能描述

恢复操作是备份系统中最危险的操作（可能覆盖生产数据），需要多重安全保护机制。

### 4.2 恢复流程设计

**完整恢复向导流程**：

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│  选择     │───▶│  预检     │───▶│  确认     │───▶│  执行     │───▶│  结果     │
│  备份文件  │    │  检查     │    │  验证     │    │  恢复     │    │  查看     │
└──────────┘    └──────────┘    └──────────┘    └──────────┘    └──────────┘
     ①               ②               ③               ④               ⑤
```

### 4.3 Step ① 选择备份文件

从备份记录列表中选择要恢复的备份文件，或手动指定文件路径。

### 4.4 Step ② 预检检查

预检是恢复前的自动检查，确保恢复可以安全执行。

**预检项目**：

| 检查项 | 说明 | 阻断条件 |
|-------|------|---------|
| 目标库连接 | 验证目标数据库可连接 | 连接失败 → 阻断 |
| 磁盘空间 | 检查目标库服务器磁盘空间是否充足 | 可用空间 < 备份文件 2 倍 → 警告 |
| 备份文件完整性 | 校验备份文件是否存在、可读、大小正常 | 文件不存在或损坏 → 阻断 |
| Checksum 校验 | 对比备份记录中的 checksum | 不匹配 → 阻断 |
| 权限检查 | 验证目标用户有写入权限 | 权限不足 → 阻断 |
| 目标库检测 | 检查目标库是否为空或是否已有数据 | 已有数据 → 强制警告 |
| 并发恢复检查 | 检查是否有其他恢复任务在执行 | 有 → 阻断 |

**预检 API**：

#### POST /api/v1/restore/pre-check

```json
// 请求
{
  "record_id": 123,
  "target_host": "10.0.0.5",
  "target_port": 3306,
  "target_database": "restore_db",
  "target_username": "root",
  "target_password": "secret123",
  "database_type": "mysql",
  "storage_type": "local"
}

// 响应
{
  "code": 0,
  "message": "ok",
  "data": {
    "passed": true,
    "warnings": [
      {
        "check": "disk_space",
        "level": "warning",
        "message": "目标服务器可用磁盘空间仅 15.2GB，建议至少 20GB",
        "blocking": false
      },
      {
        "check": "target_not_empty",
        "level": "warning",
        "message": "目标数据库 restore_db 已包含 42 张表，恢复将覆盖现有数据",
        "blocking": false
      }
    ],
    "errors": [],
    "target_info": {
      "version": "MySQL 8.0.35",
      "database_exists": true,
      "table_count": 42,
      "disk_free_gb": 15.2,
      "backup_size_gb": 7.8
    }
  }
}
```

### 4.5 Step ③ 二次确认

预检通过后，用户必须完成二次确认才能执行恢复。

**确认机制**：

1. **确认码**：前端生成 6 位随机确认码，用户需要手动输入
2. **恢复原因**：必填文本，说明为什么需要恢复
3. **风险提示**：明确展示将恢复到哪个库、备份文件信息

```
┌─────────────────────────────────────────────────────────────┐
│  ⚠️ 恢复操作确认                                            │
│                                                             │
│  ┌─ 恢复信息 ────────────────────────────────────────────┐  │
│  │  备份文件: mydb_20260327_020000.sql.gz                │  │
│  │  备份时间: 2026-03-27 02:00:00                        │  │
│  │  文件大小: 7.8 GB                                     │  │
│  │  目标库:   mysql://10.0.0.5:3306/restore_db          │  │
│  │  ⚠️ 目标库已包含 42 张表，数据将被覆盖                  │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│  恢复原因 *                                                 │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ 生产数据误删，需要从昨天的备份恢复                      │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│  请输入确认码: 8F3K2A                                        │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                                                       │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│  [取消]                              [⚠️ 确认执行恢复]       │
└─────────────────────────────────────────────────────────────┘
```

**生成确认码 API**：

#### POST /api/v1/restore/confirm-code

```json
// 响应
{
  "code": 0,
  "data": {
    "confirm_code": "8F3K2A",
    "expires_at": "2026-03-27T16:00:00+08:00"
  }
}
```

### 4.6 Step ④ 执行恢复

恢复 API 增加 confirm_code 和 reason 字段。

#### POST /api/v1/restore

```json
// 请求
{
  "record_id": 123,
  "target_host": "10.0.0.5",
  "target_port": 3306,
  "target_database": "restore_db",
  "target_username": "root",
  "target_password": "secret123",
  "database_type": "mysql",
  "storage_type": "local",
  "confirm_code": "8F3K2A",
  "reason": "生产数据误删，需要从昨天的备份恢复"
}
```

**恢复限流**：同一时间只允许一个恢复任务执行，使用分布式锁（内存锁或 Redis）控制。

**危险操作告警**：当目标库的 Host 匹配配置中标记为「生产环境」的主机时，发送 Critical 级别告警通知。

### 4.7 恢复记录审计

所有恢复操作（包括预检失败、确认取消、执行成功/失败）都记录审计日志。

#### GET /api/v1/restore/records

```json
{
  "code": 0,
  "data": {
    "total": 15,
    "items": [
      {
        "id": 1,
        "record_id": 123,
        "job_name": "生产 MySQL",
        "operator": "admin",
        "operator_id": 1,
        "target_host": "10.0.0.5:3306",
        "target_database": "restore_db",
        "backup_time": "2026-03-27T02:00:00+08:00",
        "backup_size": "7.8 GB",
        "status": "success",
        "duration": 342,
        "reason": "生产数据误删",
        "pre_check_passed": true,
        "ip_address": "192.168.1.50",
        "created_at": "2026-03-27T15:30:00+08:00"
      }
    ]
  }
}
```

### 4.8 数据库设计

```sql
-- 恢复记录表（替代原有简单的 restore 记录）
CREATE TABLE restore_records (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    record_id       BIGINT UNSIGNED NOT NULL COMMENT '关联的备份记录 ID',
    job_name        VARCHAR(128)    COMMENT '任务名称',

    -- 操作人
    operator_id     BIGINT UNSIGNED NOT NULL COMMENT '操作人用户 ID',
    operator_name   VARCHAR(50)     NOT NULL COMMENT '操作人用户名',

    -- 恢复目标
    target_host     VARCHAR(255)    NOT NULL COMMENT '目标主机',
    target_port     INT             NOT NULL COMMENT '目标端口',
    target_database VARCHAR(128)    NOT NULL COMMENT '目标数据库名',
    database_type   VARCHAR(20)     NOT NULL COMMENT '数据库类型',

    -- 备份信息快照
    backup_file     VARCHAR(512)    COMMENT '备份文件路径',
    backup_time     DATETIME        COMMENT '备份时间',
    backup_size     BIGINT          COMMENT '备份文件大小(bytes)',

    -- 恢复控制
    confirm_code    VARCHAR(10)     COMMENT '确认码',
    reason          TEXT            COMMENT '恢复原因',
    pre_check_result JSON           COMMENT '预检结果',

    -- 执行结果
    status          VARCHAR(20)     DEFAULT 'pending' COMMENT 'pending/pre_check/confirmed/running/success/failed/cancelled',
    started_at      DATETIME        NULL COMMENT '恢复开始时间',
    finished_at     DATETIME        NULL COMMENT '恢复结束时间',
    duration        INT             COMMENT '耗时(秒)',
    error_message   TEXT            COMMENT '错误信息',
    rows_affected   BIGINT          COMMENT '影响行数',

    -- 安全标记
    is_production   TINYINT(1)      DEFAULT 0 COMMENT '是否恢复到生产库',
    alerted         TINYINT(1)      DEFAULT 0 COMMENT '是否已发送告警',

    -- 请求信息
    ip_address      VARCHAR(45)     COMMENT '操作者 IP',
    user_agent      VARCHAR(512)    COMMENT '浏览器 User-Agent',

    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_record_id (record_id),
    INDEX idx_operator_id (operator_id),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='恢复记录审计表';
```

### 4.9 前端交互 — 恢复向导

**Step ① 选择备份文件**：

```
┌─────────────────────────────────────────────────────────────┐
│  恢复数据                                                    │
│                                                             │
│  ① 选择备份 ──── ② 预检 ──── ③ 确认 ──── ④ 执行            │
│  ━━━━━━━━━━━                                                   │
│                                                             │
│  任务: [生产 MySQL ▼]          搜索备份文件...               │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ ○ mydb_20260327_020000.sql.gz  7.8GB  2026-03-27 02:00│  │
│  │   ✅ 已验证  全量备份                                   │  │
│  ├───────────────────────────────────────────────────────┤  │
│  │ ○ mydb_20260326_020000.sql.gz  7.5GB  2026-03-26 02:00│  │
│  │   ✅ 已验证  全量备份                                   │  │
│  ├───────────────────────────────────────────────────────┤  │
│  │ ○ mydb_20260325_020000.sql.gz  7.2GB  2026-03-25 02:00│  │
│  │   ⚠️ 未验证  全量备份                                   │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│  [取消]                              [下一步：预检检查 →]     │
└─────────────────────────────────────────────────────────────┘
```

**Step ② 预检结果**：

```
┌─────────────────────────────────────────────────────────────┐
│  恢复数据                                                    │
│                                                             │
│  ① 选择备份 ──── ② 预检 ──── ③ 确认 ──── ④ 执行            │
│                  ━━━━━━━━━━━                                   │
│                                                             │
│  目标库配置                                                  │
│  主机: [10.0.0.5      ]  端口: [3306]                       │
│  用户: [root           ]  密码: [•••••]                     │
│  数据库: [restore_db    ]                                     │
│                                                             │
│  ┌─ 预检结果 ────────────────────────────────────────────┐  │
│  │  ✅ 目标库连接正常      MySQL 8.0.35                   │  │
│  │  ✅ 备份文件完整        7.8 GB, checksum 匹配           │  │
│  │  ✅ 权限检查通过        用户有 ALL PRIVILEGES           │  │
│  │  ✅ 无并发恢复任务                                     │  │
│  │  ⚠️ 磁盘空间偏低        可用 15.2GB，建议 20GB          │  │
│  │  ⚠️ 目标库非空          包含 42 张表，数据将被覆盖       │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│  [← 上一步]                            [下一步：确认 →]      │
└─────────────────────────────────────────────────────────────┘
```

### 4.10 权限要求

| 操作 | 所需权限 |
|------|---------|
| 恢复预检 | `restore:execute` |
| 执行恢复 | `restore:execute` |
| 查看恢复记录 | `restore:view` |
| 恢复到生产库 | `restore:production`（额外权限） |

---

## 5. 路由汇总

新增路由（基于现有 router.go 扩展）：

```go
// jobs 组内新增
jobs.POST("/test-connection", jobHandler.TestConnection)

// 模板管理
templates := v1.Group("/templates")
{
    templateHandler := handler.NewTemplateHandler(db)
    templates.GET("", templateHandler.List)
    templates.POST("", templateHandler.Create)
    templates.GET("/:id", templateHandler.Get)
    templates.PUT("/:id", templateHandler.Update)
    templates.DELETE("/:id", templateHandler.Delete)
}

// 从模板创建任务（jobs 组内新增）
jobs.POST("/from-template/:id", jobHandler.CreateFromTemplate)

// 恢复安全（restore 组内新增/改造）
restore.POST("/pre-check", restoreHandler.PreCheck)
restore.POST("/confirm-code", restoreHandler.GenerateConfirmCode)
restore.POST("", restoreHandler.Restore)  // 改造，增加 confirm_code 和 reason
restore.GET("/records", restoreHandler.Records)  // 新增：恢复审计记录
```

---

## 6. 实现优先级

| 优先级 | 模块 | 预估工时 |
|-------|------|---------|
| P0 | 数据库连接测试 | 2 天 |
| P0 | 恢复预检 + 二次确认 | 3 天 |
| P1 | 备份任务模板 | 2 天 |
| P1 | 恢复记录审计 | 1 天 |
| P2 | 恢复限流 + 生产库告警 | 1 天 |
