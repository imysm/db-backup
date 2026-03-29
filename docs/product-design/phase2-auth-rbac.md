# Phase 2: 用户认证 + RBAC — 产品设计文档

> 版本: v1.0 | 日期: 2026-03-27 | 状态: 设计中

## 1. 用户体系设计

### 1.1 用户模型

```go
type User struct {
    ID            uint      `gorm:"primaryKey" json:"id"`
    Username      string    `gorm:"size:50;uniqueIndex;notNull" json:"username"`
    PasswordHash  string    `gorm:"size:255;notNull" json:"-"`
    Email         string    `gorm:"size:128;uniqueIndex" json:"email"`
    Phone         string    `gorm:"size:20" json:"phone"`
    DisplayName   string    `gorm:"size:50" json:"display_name"`
    Avatar        string    `gorm:"size:255" json:"avatar"`
    Status        string    `gorm:"size:20;default:active" json:"status"` // active/disabled/locked
    LastLoginAt   *time.Time `json:"last_login_at"`
    LastLoginIP   string    `gorm:"size:45" json:"last_login_ip"`
    PasswordChangedAt *time.Time `json:"password_changed_at"`
    CreatedAt     time.Time `json:"created_at"`
    UpdatedAt     time.Time `json:"updated_at"`
    DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}
```

### 1.2 登录方式

**用户名密码 + JWT Token**：

1. 用户提交用户名 + 密码
2. 后端验证凭据，生成 JWT Token 对
3. 前端存储 Token，后续请求携带 Access Token
4. Access Token 过期 → 使用 Refresh Token 获取新 Token 对
5. Refresh Token 也过期 → 跳转登录页

### 1.3 Token 管理

**双 Token 方案**：

| Token 类型 | 有效期 | 用途 |
|-----------|--------|------|
| Access Token | 2 小时 | API 请求认证（Header: `Authorization: Bearer <token>`） |
| Refresh Token | 7 天 | 刷新 Access Token（存储在 HttpOnly Cookie 中） |

**Token 载荷（JWT Payload）**：

```json
// Access Token
{
  "sub": 1,
  "username": "admin",
  "roles": ["super_admin"],
  "type": "access",
  "iat": 1743000000,
  "exp": 1743007200,
  "jti": "unique-token-id"
}

// Refresh Token
{
  "sub": 1,
  "type": "refresh",
  "iat": 1743000000,
  "exp": 1743607200,
  "jti": "unique-refresh-id"
}
```

**过期策略**：
- Access Token 过期 → 前端自动调用 Refresh 接口（无感刷新）
- Refresh Token 过期 → 返回 401 → 前端跳转登录页
- 用户主动登出 → 服务端将 Refresh Token 加入黑名单（Redis 或内存）
- 密码修改后 → 所有现有 Token 立即失效

### 1.4 密码策略

| 规则 | 要求 |
|------|------|
| 最小长度 | 8 位 |
| 复杂度 | 必须包含大写、小写、数字，至少 3 种字符类型 |
| 常见密码 | 拒绝 Top 1000 常见密码 |
| 历史密码 | 不能与最近 3 次使用的密码相同 |
| 过期提醒 | 90 天提醒修改，强制修改周期 180 天（可配置） |
| 初始密码 | 首次登录强制修改密码 |

---

## 2. RBAC 权限模型

### 2.1 角色定义

| 角色 | 标识 | 说明 |
|------|------|------|
| 超级管理员 | `super_admin` | 系统最高权限，可管理用户和角色，不可删除 |
| 管理员 | `admin` | 管理所有备份任务和配置，不能管理用户 |
| 操作员 | `operator` | 执行备份/恢复操作，不能修改配置 |
| 只读用户 | `viewer` | 仅查看备份记录和统计，不能执行任何写操作 |

### 2.2 资源定义

| 资源 | 标识 | 说明 |
|------|------|------|
| 仪表盘 | `dashboard` | 统计数据查看 |
| 备份任务 | `job` | 任务 CRUD、启用/禁用、立即执行 |
| 备份记录 | `record` | 记录查看、下载、删除 |
| 数据恢复 | `restore` | 恢复操作 |
| 备份验证 | `verify` | 验证操作 |
| 系统设置 | `setting` | 存储配置、通知配置、全局参数 |
| 用户管理 | `user` | 用户 CRUD、角色分配 |
| 角色管理 | `role` | 角色 CRUD、权限分配 |
| 审计日志 | `audit` | 操作日志查看 |

### 2.3 权限矩阵

| 资源 | 操作 | super_admin | admin | operator | viewer |
|------|------|:-----------:|:-----:|:--------:|:------:|
| dashboard | 查看 | ✅ | ✅ | ✅ | ✅ |
| job | 查看 | ✅ | ✅ | ✅ | ✅ |
| job | 创建/编辑/删除 | ✅ | ✅ | ❌ | ❌ |
| job | 启用/禁用 | ✅ | ✅ | ❌ | ❌ |
| job | 立即执行 | ✅ | ✅ | ✅ | ❌ |
| record | 查看 | ✅ | ✅ | ✅ | ✅ |
| record | 下载 | ✅ | ✅ | ✅ | ❌ |
| record | 删除 | ✅ | ✅ | ❌ | ❌ |
| restore | 查看历史 | ✅ | ✅ | ✅ | ✅ |
| restore | 执行恢复 | ✅ | ✅ | ✅ | ❌ |
| verify | 查看 | ✅ | ✅ | ✅ | ✅ |
| verify | 执行验证 | ✅ | ✅ | ✅ | ❌ |
| setting | 查看 | ✅ | ✅ | ❌ | ❌ |
| setting | 修改 | ✅ | ✅ | ❌ | ❌ |
| user | 查看 | ✅ | ❌ | ❌ | ❌ |
| user | 创建/编辑/删除 | ✅ | ❌ | ❌ | ❌ |
| role | 查看 | ✅ | ❌ | ❌ | ❌ |
| role | 创建/编辑/删除 | ✅ | ❌ | ❌ | ❌ |
| audit | 查看 | ✅ | ✅ | ❌ | ❌ |

### 2.4 权限表示格式

权限采用 `resource:action` 格式：

```
job:read, job:create, job:update, job:delete, job:execute
record:read, record:download, record:delete
restore:read, restore:execute
verify:read, verify:execute
setting:read, setting:update
user:read, user:create, user:update, user:delete
role:read, role:create, role:update, role:delete
audit:read
```

---

## 3. 数据库设计

### 3.1 users 表

```sql
CREATE TABLE users (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    username        TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    email           TEXT UNIQUE,
    phone           TEXT,
    display_name    TEXT DEFAULT '',
    avatar          TEXT DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'active',  -- active, disabled, locked
    last_login_at   DATETIME,
    last_login_ip   TEXT,
    password_changed_at DATETIME,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at      DATETIME
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);
```

### 3.2 roles 表

```sql
CREATE TABLE roles (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,         -- super_admin, admin, operator, viewer
    display_name TEXT NOT NULL,                -- 超级管理员, 管理员, 操作员, 只读用户
    description TEXT DEFAULT '',
    is_builtin  BOOLEAN NOT NULL DEFAULT FALSE, -- 内置角色不可删除
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_roles_name ON roles(name);

-- 初始数据
INSERT INTO roles (name, display_name, description, is_builtin) VALUES
    ('super_admin', '超级管理员', '系统最高权限，可管理用户和角色', TRUE),
    ('admin', '管理员', '管理所有备份任务和配置', TRUE),
    ('operator', '操作员', '执行备份/恢复操作', TRUE),
    ('viewer', '只读用户', '仅查看备份记录和统计', TRUE);
```

### 3.3 permissions 表

```sql
CREATE TABLE permissions (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    resource    TEXT NOT NULL,   -- job, record, restore, setting, user, role, audit
    action      TEXT NOT NULL,   -- read, create, update, delete, execute, download
    name        TEXT NOT NULL UNIQUE, -- job:read, job:create ...
    description TEXT DEFAULT ''
);

CREATE INDEX idx_permissions_name ON permissions(name);

-- 初始数据
INSERT INTO permissions (resource, action, name, description) VALUES
    -- 仪表盘
    ('dashboard', 'read', 'dashboard:read', '查看仪表盘'),
    -- 备份任务
    ('job', 'read', 'job:read', '查看备份任务'),
    ('job', 'create', 'job:create', '创建备份任务'),
    ('job', 'update', 'job:update', '编辑备份任务'),
    ('job', 'delete', 'job:delete', '删除备份任务'),
    ('job', 'execute', 'job:execute', '执行/启用/禁用备份任务'),
    -- 备份记录
    ('record', 'read', 'record:read', '查看备份记录'),
    ('record', 'download', 'record:download', '下载备份文件'),
    ('record', 'delete', 'record:delete', '删除备份记录'),
    -- 数据恢复
    ('restore', 'read', 'restore:read', '查看恢复历史'),
    ('restore', 'execute', 'restore:execute', '执行恢复操作'),
    -- 备份验证
    ('verify', 'read', 'verify:read', '查看验证状态'),
    ('verify', 'execute', 'verify:execute', '执行验证操作'),
    -- 系统设置
    ('setting', 'read', 'setting:read', '查看系统设置'),
    ('setting', 'update', 'setting:update', '修改系统设置'),
    -- 用户管理
    ('user', 'read', 'user:read', '查看用户列表'),
    ('user', 'create', 'user:create', '创建用户'),
    ('user', 'update', 'user:update', '编辑用户'),
    ('user', 'delete', 'user:delete', '删除用户'),
    -- 角色管理
    ('role', 'read', 'role:read', '查看角色列表'),
    ('role', 'create', 'role:create', '创建角色'),
    ('role', 'update', 'role:update', '编辑角色'),
    ('role', 'delete', 'role:delete', '删除角色'),
    -- 审计日志
    ('audit', 'read', 'audit:read', '查看审计日志');
```

### 3.4 role_permissions 关联表

```sql
CREATE TABLE role_permissions (
    role_id       INTEGER NOT NULL,
    permission_id INTEGER NOT NULL,
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
);

CREATE INDEX idx_rp_role ON role_permissions(role_id);
CREATE INDEX idx_rp_permission ON role_permissions(permission_id);
```

### 3.5 user_roles 关联表

```sql
CREATE TABLE user_roles (
    user_id INTEGER NOT NULL,
    role_id INTEGER NOT NULL,
    PRIMARY KEY (user_id, role_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE
);

CREATE INDEX idx_ur_user ON user_roles(user_id);
CREATE INDEX idx_ur_role ON user_roles(role_id);
```

### 3.6 password_history 表

```sql
CREATE TABLE password_history (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL,
    password_hash TEXT NOT NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_ph_user ON password_history(user_id);
```

### 3.7 audit_logs 审计日志表

```sql
CREATE TABLE audit_logs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER,
    username    TEXT NOT NULL,          -- 冗余存储，防止用户删除后丢失
    action      TEXT NOT NULL,          -- login, logout, create_job, delete_record, ...
    resource    TEXT NOT NULL,          -- job, record, restore, setting, user, role
    resource_id TEXT,                   -- 操作的资源 ID
    detail      TEXT,                   -- JSON 格式的详细信息
    ip_address  TEXT,
    user_agent  TEXT,
    status      TEXT NOT NULL DEFAULT 'success', -- success, failure
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_user ON audit_logs(user_id);
CREATE INDEX idx_audit_action ON audit_logs(action);
CREATE INDEX idx_audit_resource ON audit_logs(resource);
CREATE INDEX idx_audit_created ON audit_logs(created_at);
```

### 3.8 ER 关系图

```
┌──────────┐     ┌────────────┐     ┌──────────────┐
│  users   │────<│ user_roles │>────│    roles     │
└──────────┘     └────────────┘     └──────────────┘
     │                                      │
     │                                      │
     ▼                                      ▼
┌───────────────────┐          ┌────────────────────┐
│ password_history  │          │ role_permissions   │
└───────────────────┘          └────────────────────┘
                                         │
                                         ▼
                                 ┌──────────────┐
                                 │ permissions  │
                                 └──────────────┘

┌──────────────────┐
│   audit_logs     │  (独立表，通过 user_id 关联)
└──────────────────┘
```

---

## 4. API 设计

### 4.1 认证 API

#### POST /api/v1/auth/login

用户登录。

**请求体**：

```json
{
  "username": "admin",
  "password": "Admin@123"
}
```

**响应**：

```json
{
  "code": 0,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 7200,
    "user": {
      "id": 1,
      "username": "admin",
      "display_name": "系统管理员",
      "email": "admin@example.com",
      "roles": ["super_admin"],
      "permissions": ["job:read", "job:create", "..."],
      "must_change_password": false
    }
  }
}
```

> 注：Refresh Token 通过 `Set-Cookie: refresh_token=xxx; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth; Max-Age=604800` 设置。

---

#### POST /api/v1/auth/logout

用户登出。

**请求头**：

```
Authorization: Bearer <access_token>
Cookie: refresh_token=<refresh_token>
```

**响应**：

```json
{
  "code": 0,
  "message": "登出成功"
}
```

> 服务端将 Refresh Token 加入黑名单。

---

#### POST /api/v1/auth/refresh

刷新 Access Token。

**请求体**（如果 Refresh Token 在 Cookie 中则无需请求体）：

```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**响应**：

```json
{
  "code": 0,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 7200
  }
}
```

---

#### GET /api/v1/auth/me

获取当前用户信息。

**请求头**：

```
Authorization: Bearer <access_token>
```

**响应**：

```json
{
  "code": 0,
  "data": {
    "id": 1,
    "username": "admin",
    "display_name": "系统管理员",
    "email": "admin@example.com",
    "phone": "13800138000",
    "avatar": "",
    "roles": ["super_admin"],
    "permissions": [
      "dashboard:read",
      "job:read", "job:create", "job:update", "job:delete", "job:execute",
      "record:read", "record:download", "record:delete",
      "restore:read", "restore:execute",
      "verify:read", "verify:execute",
      "setting:read", "setting:update",
      "user:read", "user:create", "user:update", "user:delete",
      "role:read", "role:create", "role:update", "role:delete",
      "audit:read"
    ],
    "last_login_at": "2026-03-27T10:00:00Z",
    "must_change_password": false
  }
}
```

---

#### PUT /api/v1/auth/password

修改密码。

**请求体**：

```json
{
  "old_password": "OldPass@123",
  "new_password": "NewPass@456"
}
```

**响应**：

```json
{
  "code": 0,
  "message": "密码修改成功"
}
```

> 修改密码后所有现有 Token 立即失效（通过 password_changed_at 实现）。

---

#### PUT /api/v1/auth/password/first-change

首次登录修改初始密码。

**请求体**：

```json
{
  "password": "MyNewPass@123",
  "confirm_password": "MyNewPass@123"
}
```

**响应**：

```json
{
  "code": 0,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 7200
  }
}
```

---

### 4.2 用户管理 API

#### GET /api/v1/users

获取用户列表。

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| `page` | int | 页码（默认 1） |
| `size` | int | 每页数量（默认 20） |
| `search` | string | 搜索用户名/邮箱 |
| `status` | string | 状态过滤：active/disabled/locked |
| `role` | string | 角色过滤 |

**响应**：

```json
{
  "code": 0,
  "data": {
    "total": 10,
    "page": 1,
    "size": 20,
    "items": [
      {
        "id": 1,
        "username": "admin",
        "display_name": "系统管理员",
        "email": "admin@example.com",
        "phone": "13800138000",
        "status": "active",
        "roles": ["super_admin"],
        "last_login_at": "2026-03-27T10:00:00Z",
        "created_at": "2026-01-01T00:00:00Z"
      }
    ]
  }
}
```

---

#### POST /api/v1/users

创建用户（仅 super_admin）。

**请求体**：

```json
{
  "username": "zhangsan",
  "password": "Temp@123456",
  "display_name": "张三",
  "email": "zhangsan@example.com",
  "phone": "13800138001",
  "role_ids": [3]
}
```

**响应**：

```json
{
  "code": 0,
  "data": {
    "id": 2,
    "username": "zhangsan",
    "message": "用户创建成功，初始密码已发送至邮箱"
  }
}
```

---

#### GET /api/v1/users/:id

获取用户详情。

**响应**：

```json
{
  "code": 0,
  "data": {
    "id": 2,
    "username": "zhangsan",
    "display_name": "张三",
    "email": "zhangsan@example.com",
    "phone": "13800138001",
    "status": "active",
    "roles": [
      {
        "id": 3,
        "name": "operator",
        "display_name": "操作员"
      }
    ],
    "last_login_at": "2026-03-25T08:00:00Z",
    "last_login_ip": "192.168.1.100",
    "password_changed_at": "2026-03-20T10:00:00Z",
    "created_at": "2026-01-15T00:00:00Z"
  }
}
```

---

#### PUT /api/v1/users/:id

更新用户信息。

**请求体**（所有字段可选）：

```json
{
  "display_name": "张三丰",
  "email": "zhangsanfeng@example.com",
  "phone": "13800138002",
  "status": "active",
  "role_ids": [2, 3]
}
```

**响应**：

```json
{
  "code": 0,
  "message": "用户更新成功"
}
```

---

#### DELETE /api/v1/users/:id

删除用户（软删除）。

**响应**：

```json
{
  "code": 0,
  "message": "用户已删除"
}
```

---

#### POST /api/v1/users/:id/reset-password

重置用户密码（管理员操作）。

**请求体**：

```json
{
  "new_password": "Reset@123456"
}
```

**响应**：

```json
{
  "code": 0,
  "message": "密码已重置，用户下次登录需修改密码"
}
```

---

#### PUT /api/v1/users/:id/status

启用/禁用/锁定用户。

**请求体**：

```json
{
  "status": "disabled"
}
```

**响应**：

```json
{
  "code": 0,
  "message": "用户状态已更新"
}
```

---

### 4.3 角色管理 API

#### GET /api/v1/roles

获取角色列表。

**响应**：

```json
{
  "code": 0,
  "data": [
    {
      "id": 1,
      "name": "super_admin",
      "display_name": "超级管理员",
      "description": "系统最高权限",
      "is_builtin": true,
      "permission_count": 24,
      "user_count": 1
    },
    {
      "id": 2,
      "name": "admin",
      "display_name": "管理员",
      "description": "管理所有备份任务和配置",
      "is_builtin": true,
      "permission_count": 16,
      "user_count": 3
    }
  ]
}
```

---

#### GET /api/v1/roles/:id

获取角色详情（含权限列表）。

**响应**：

```json
{
  "code": 0,
  "data": {
    "id": 2,
    "name": "admin",
    "display_name": "管理员",
    "description": "管理所有备份任务和配置",
    "is_builtin": true,
    "permissions": [
      { "id": 1, "name": "dashboard:read", "resource": "dashboard", "action": "read" },
      { "id": 2, "name": "job:read", "resource": "job", "action": "read" },
      { "id": 3, "name": "job:create", "resource": "job", "action": "create" }
    ]
  }
}
```

---

#### POST /api/v1/roles

创建自定义角色。

**请求体**：

```json
{
  "name": "db_operator",
  "display_name": "数据库操作员",
  "description": "可执行备份和恢复，不能修改配置",
  "permission_ids": [1, 2, 5, 6, 7, 8, 9, 10, 11]
}
```

**响应**：

```json
{
  "code": 0,
  "data": {
    "id": 5,
    "name": "db_operator",
    "message": "角色创建成功"
  }
}
```

---

#### PUT /api/v1/roles/:id

更新角色。

**请求体**：

```json
{
  "display_name": "高级操作员",
  "description": "可执行备份、恢复和验证",
  "permission_ids": [1, 2, 5, 6, 7, 8, 9, 10, 11, 12, 13]
}
```

---

#### DELETE /api/v1/roles/:id

删除自定义角色。

**约束**：内置角色（`is_builtin=true`）不可删除；有用户关联时不可删除。

**响应**：

```json
{
  "code": 0,
  "message": "角色已删除"
}
```

---

### 4.4 审计日志 API

#### GET /api/v1/audit-logs

获取审计日志列表。

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| `page` | int | 页码 |
| `size` | int | 每页数量 |
| `user_id` | int | 按用户过滤 |
| `action` | string | 按操作类型过滤 |
| `resource` | string | 按资源类型过滤 |
| `start_time` | string | 开始时间 |
| `end_time` | string | 结束时间 |

**响应**：

```json
{
  "code": 0,
  "data": {
    "total": 256,
    "page": 1,
    "size": 20,
    "items": [
      {
        "id": 256,
        "username": "admin",
        "action": "create_job",
        "resource": "job",
        "resource_id": "5",
        "detail": "{\"name\": \"mysql-prod-backup\", \"database_type\": \"mysql\"}",
        "ip_address": "192.168.1.100",
        "status": "success",
        "created_at": "2026-03-27T14:30:00Z"
      }
    ]
  }
}
```

---

## 5. 中间件设计

### 5.1 JWT 认证中间件

```go
func JWTAuthMiddleware(secret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 从 Header 提取 Access Token
        authHeader := c.GetHeader("Authorization")
        if strings.HasPrefix(authHeader, "Bearer ") {
            tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
            token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
                return []byte(secret), nil
            })
            if err == nil && token.Valid {
                claims := token.Claims.(jwt.MapClaims)
                c.Set("user_id", uint(claims["sub"].(float64)))
                c.Set("username", claims["username"].(string))
                c.Set("auth_type", "jwt")
                c.Next()
                return
            }
        }

        // 2. JWT 无效，继续尝试 API Key（向下兼容）
        // 如果 API Key 也无效，返回 401
        c.JSON(401, gin.H{"code": 401, "message": "未授权"})
        c.Abort()
    }
}
```

### 5.2 RBAC 权限校验中间件

```go
// RequirePermission 检查当前用户是否拥有指定权限
func RequirePermission(requiredPerms ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := c.GetUint("user_id")
        userPerms := getUserPermissions(userID) // 从缓存/DB 获取

        for _, perm := range requiredPerms {
            if !userPerms.Contains(perm) {
                c.JSON(403, gin.H{"code": 403, "message": "权限不足"})
                c.Abort()
                return
            }
        }
        c.Next()
    }
}

// 使用示例
r.DELETE("/api/v1/jobs/:id", RequirePermission("job:delete"), jobHandler.Delete)
r.POST("/api/v1/restore", RequirePermission("restore:execute"), restoreHandler.Restore)
r.GET("/api/v1/audit-logs", RequirePermission("audit:read"), auditHandler.List)
```

### 5.3 API Key 向下兼容

两种认证方式并存，优先级：JWT > API Key。

```
请求流程：
1. 检查 Authorization: Bearer <JWT>
   └─ 有效 → JWT 认证，设置 user_id、username、auth_type=jwt
   └─ 无效 → 继续步骤 2
2. 检查 X-API-Key / Authorization: Bearer <API Key>
   └─ 有效 → API Key 认证，auth_type=apikey，权限等同 super_admin
   └─ 无效 → 返回 401
```

**路由配置**：

```go
// 认证中间件（支持 JWT + API Key）
authMiddleware := AuthMiddleware(cfg.JWTSecret, cfg.APIKeys)

v1 := r.Group("/api/v1", authMiddleware)
{
    // 公开接口（不需要认证）
    v1.POST("/auth/login", authHandler.Login)
    v1.POST("/auth/refresh", authHandler.Refresh)

    // 需要认证的接口
    v1.GET("/auth/me", authHandler.Me)
    v1.PUT("/auth/password", authHandler.ChangePassword)

    // 用户管理（需要 user:* 权限）
    users := v1.Group("", RequirePermission("user:read"))
    {
        users.GET("/users", userHandler.List)
        users.POST("/users", RequirePermission("user:create"), userHandler.Create)
        users.GET("/users/:id", userHandler.Get)
        users.PUT("/users/:id", RequirePermission("user:update"), userHandler.Update)
        users.DELETE("/users/:id", RequirePermission("user:delete"), userHandler.Delete)
    }

    // ... 其他业务接口同理
}
```

---

## 6. 前端交互

### 6.1 登录/登出流程

```
登录流程:
┌─────────┐    POST /auth/login     ┌─────────┐
│  前端    │ ──────────────────────> │  后端    │
│         │ <────────────────────── │         │
│         │  access_token + user    │         │
│         │                          │         │
│  存储    │  localStorage: token   │         │
│  跳转    │  → /dashboard           │         │
└─────────┘                          └─────────┘

登出流程:
┌─────────┐    POST /auth/logout    ┌─────────┐
│  前端    │ ──────────────────────> │  后端    │
│         │ <────────────────────── │         │
│         │  200 OK                 │         │
│         │                          │         │
│  清除    │  localStorage + Cookie  │         │
│  跳转    │  → /login               │         │
└─────────┘                          └─────────┘
```

### 6.2 会话管理

```typescript
// Axios 拦截器配置
const api = axios.create({ baseURL: '/api/v1' });

// 请求拦截：自动附加 Token
api.interceptors.request.use(config => {
  const token = localStorage.getItem('access_token');
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

// 响应拦截：自动刷新 Token
api.interceptors.response.use(
  response => response,
  async error => {
    if (error.response?.status === 401) {
      try {
        const { data } = await axios.post('/api/v1/auth/refresh');
        localStorage.setItem('access_token', data.access_token);
        error.config.headers.Authorization = `Bearer ${data.access_token}`;
        return api(error.config); // 重试原请求
      } catch {
        localStorage.removeItem('access_token');
        window.location.href = '/login';
      }
    }
    return Promise.reject(error);
  }
);
```

### 6.3 权限控制

**菜单级别**：

```typescript
// 根据权限过滤菜单
const menuItems = [
  { path: '/dashboard', label: '仪表盘', perm: 'dashboard:read' },
  { path: '/jobs',      label: '任务管理', perm: 'job:read' },
  { path: '/restore',   label: '数据恢复', perm: 'restore:read' },
  { path: '/settings',  label: '系统设置', perm: 'setting:read' },
  { path: '/users',     label: '用户管理', perm: 'user:read' },
  { path: '/audit',     label: '审计日志', perm: 'audit:read' },
];

const visibleMenus = menuItems.filter(item =>
  user.permissions.includes(item.perm)
);
```

**按钮级别**：

```tsx
// 权限控制组件
function PermissionGuard({ permission, children, fallback = null }) {
  const { user } = useAuth();
  if (!user.permissions.includes(permission)) return fallback;
  return children;
}

// 使用
<PermissionGuard permission="job:create">
  <Button type="primary">新建任务</Button>
</PermissionGuard>

<PermissionGuard permission="job:execute">
  <Button>立即执行</Button>
</PermissionGuard>

<PermissionGuard permission="user:delete">
  <Popconfirm title="确定删除此用户？">
    <Button danger>删除</Button>
  </Popconfirm>
</PermissionGuard>
```

**路由级别**：

```typescript
// 路由守卫
function ProtectedRoute({ permission, children }) {
  const { user, isLoading } = useAuth();

  if (isLoading) return <Spin />;
  if (!user) return <Navigate to="/login" />;
  if (permission && !user.permissions.includes(permission)) {
    return <Result status="403" title="权限不足" />;
  }
  return children;
}

// 路由配置
<Route path="/users" element={
  <ProtectedRoute permission="user:read">
    <UserManagement />
  </ProtectedRoute>
} />
```

---

## 7. 安全要求

### 7.1 密码存储

**算法：bcrypt**

```go
import "golang.org/x/crypto/bcrypt"

// 哈希密码（cost=12）
hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)

// 验证密码
err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
```

| 配置项 | 值 | 说明 |
|--------|---|------|
| 算法 | bcrypt | Go 标准推荐，安全且经过广泛验证 |
| Cost | 12 | 约 250ms/次，平衡安全与性能 |
| Pepper | 从环境变量读取 | 额外加盐，防止彩虹表攻击 |

### 7.2 JWT 密钥管理

| 要求 | 实现 |
|------|------|
| 密钥来源 | 环境变量 `JWT_SECRET`，不硬编码 |
| 密钥长度 | ≥ 32 字节（256 位） |
| 密钥轮换 | 支持多密钥验证（旧密钥用于验证，新密钥用于签发） |
| 签名算法 | HS256（对称加密，单服务部署足够） |
| 密钥泄露 | 通过修改密钥使所有 Token 失效 |

```go
// 配置示例
type JWTConfig struct {
    Secret          string   `yaml:"secret" env:"JWT_SECRET"`
    AccessTTL       int      `yaml:"access_ttl" default:"7200"`      // 2 小时
    RefreshTTL      int      `yaml:"refresh_ttl" default:"604800"`    // 7 天
    OldSecrets      []string `yaml:"old_secrets"`                      // 轮换过渡期旧密钥
}
```

### 7.3 防暴力破解

**登录限流**：

| 策略 | 配置 |
|------|------|
| 同一 IP | 5 次/分钟 |
| 同一用户名 | 5 次/分钟 |
| 连续失败锁定 | 5 次失败 → 锁定 15 分钟 |
| 锁定通知 | 锁定时发送通知（如果配置了） |

```go
// 使用内存限流器（单实例）或 Redis（多实例）
var loginLimiter = rate.NewLimiter(5, time.Minute)

func LoginHandler(c *gin.Context) {
    ip := c.ClientIP()
    username := c.PostForm("username")
    key := fmt.Sprintf("login:%s:%s", ip, username)

    if !limiter.Allow(key) {
        c.JSON(429, gin.H{"code": 429, "message": "登录尝试过于频繁，请稍后再试"})
        return
    }
    // ... 正常登录逻辑
}
```

### 7.4 XSS/CSRF 防护

| 威胁 | 防护措施 |
|------|---------|
| XSS | React 自动转义；CSP Header `Content-Security-Policy: default-src 'self'` |
| CSRF | JWT 存储在 localStorage（非 Cookie），天然免疫 CSRF；Refresh Token Cookie 设置 `SameSite=Strict` |
| 点击劫持 | `X-Frame-Options: DENY` |
| 请求伪造 | 关键操作（删除用户、恢复数据）二次确认 |
| 敏感数据 | 密码字段永不返回前端（`json:"-"`）；API Key 部分遮掩显示 |
| 安全 Header | `X-Content-Type-Options: nosniff`、`Referrer-Policy: strict-origin-when-cross-origin` |

### 7.5 安全 Header 中间件

```go
func SecurityHeaders() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("X-Content-Type-Options", "nosniff")
        c.Header("X-Frame-Options", "DENY")
        c.Header("X-XSS-Protection", "1; mode=block")
        c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
        c.Header("Content-Security-Policy",
            "default-src 'self'; "+
            "script-src 'self'; "+
            "style-src 'self' 'unsafe-inline'; "+
            "img-src 'self' data:; "+
            "connect-src 'self' ws: wss:;")
        c.Next()
    }
}
```

---

## 附录 A：初始化超级管理员

系统首次启动时，自动创建超级管理员账号：

```go
func InitSuperAdmin(db *gorm.DB) {
    var count int64
    db.Model(&User{}).Count(&count)
    if count > 0 {
        return // 已有用户，跳过
    }

    hash, _ := bcrypt.GenerateFromPassword([]byte("admin"), 12)
    admin := User{
        Username:      "admin",
        PasswordHash:  string(hash),
        DisplayName:   "超级管理员",
        Email:         "admin@localhost",
        Status:        "active",
    }
    db.Create(&admin)

    // 分配 super_admin 角色
    db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, 1)", admin.ID)

    log.Println("⚠️ 超级管理员已创建: admin / admin（请立即修改密码）")
}
```

## 附录 B：API Key 兼容迁移

Phase 1 使用 API Key 认证，Phase 2 引入 JWT 后保持兼容：

1. **API Key 继续有效**：通过中间件同时支持两种认证
2. **API Key 权限**：等同于 `super_admin`（不区分角色）
3. **迁移建议**：逐步将 API Key 用户迁移到 JWT 用户
4. **未来废弃**：在 Phase 3 中可考虑废弃 API Key 认证
