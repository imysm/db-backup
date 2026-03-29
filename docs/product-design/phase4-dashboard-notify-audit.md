# Phase 4: 备份健康仪表盘 + 通知升级 + 审计日志 — 产品设计文档

> 版本: v1.0 | 日期: 2026-03-27 | 状态: 设计中

## 1. 产品概述

Phase 4 从「可观测性」和「安全合规」两个维度提升系统能力：

| 模块 | 解决的问题 | 核心价值 |
|------|-----------|---------|
| **健康仪表盘** | 缺乏全局视角，问题发现滞后 | 一眼看清备份系统健康度 |
| **通知升级** | 通知渠道单一、无分级、无升级 | 及时精准的告警触达 |
| **审计日志** | 操作不可追溯，无法满足合规要求 | 完整的操作记录，可审计可追溯 |

---

## 2. 备份健康仪表盘

### 2.1 健康评分算法

**评分维度与权重**：

| 维度 | 权重 | 指标 | 满分条件 |
|------|------|------|---------|
| **成功率** | 40% | 过去 7 天备份成功率 | ≥ 99% |
| **RPO 达标** | 20% | 备份间隔是否满足 RPO 要求 | 全部任务达标 |
| **验证覆盖率** | 20% | 最近 7 天备份的验证比例 | ≥ 80% |
| **存储健康** | 20% | 磁盘使用率、备份文件完整性 | 使用率 < 80%，全部文件完整 |

**评分计算**：

```
健康总分 = 成功率分 × 0.4 + RPO分 × 0.2 + 验证分 × 0.2 + 存储分 × 0.2

其中：
- 成功率分 = min(成功率 / 99%, 1.0) × 100
- RPO分 = (达标任务数 / 总任务数) × 100
- 验证分 = min(已验证数 / 总备份数 / 0.8, 1.0) × 100
- 存储分 = min((1 - 使用率) / 0.2, 1.0) × 100  （使用率 < 80% 满分）
```

**评分等级**：

| 等级 | 分数范围 | 颜色 | 含义 |
|------|---------|------|------|
| 优秀 | 90-100 | 🟢 绿色 | 系统健康 |
| 良好 | 70-89 | 🟡 黄色 | 需要关注 |
| 警告 | 50-69 | 🟠 橙色 | 存在风险 |
| 严重 | 0-49 | 🔴 红色 | 需要立即处理 |

### 2.2 仪表盘布局设计

```
┌─────────────────────────────────────────────────────────────────────┐
│  🗄️ DB Backup          [仪表盘] [任务] [记录] [恢复] [设置]         │
│                                                                     │
│  ┌─────────────┐  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────┐│
│  │  健康评分    │  │ 今日任务  │ │ 成功率    │ │ 平均耗时  │ │ 存储用  ││
│  │             │  │          │ │          │ │          │ │  量     ││
│  │    87       │  │   23     │ │  95.7%   │ │  2m 35s  │ │  68.3% ││
│  │   /100      │  │          │ │          │ │          │ │        ││
│  │   🟢 良好   │  │ 成功 22  │ │ ↑ 1.2%   │ │ ↓ 12s   │ │ 52/76G ││
│  │             │  │ 失败  1  │ │          │ │          │ │        ││
│  └─────────────┘  └──────────┘ └──────────┘ └──────────┘ └────────┘│
│                                                                     │
│  ┌────────────────────────────────┐  ┌────────────────────────────┐ │
│  │  备份趋势 (近7天)              │  │  存储趋势 (近30天)          │ │
│  │  [7天 ▼] [30天]               │  │  [7天] [30天 ▼]            │ │
│  │                                │  │                            │ │
│  │  100% ┤    ╭──╮                │  │  80G ┤          ╱──        │ │
│  │   80% ┤ ╭──╯  ╰──╮            │  │  60G ┤     ╱──╯            │ │
│  │   60% ┤╯         ╰──╮         │  │  40G ┤  ╱──╯                │ │
│  │   40% ┤             ╰──╮       │  │  20G ┤╱                     │ │
│  │   20% ┤                ╰──     │  │   0G ┤                      │ │
│  │    0% ┤  Mo Tu We Th Fr Sa Su │  │      ┤  W1  W2  W3  W4     │ │
│  │        成功率  ── 任务数 ──    │  │      ── 已用  ── 可用 ──    │ │
│  └────────────────────────────────┘  └────────────────────────────┘ │
│                                                                     │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  最近备份                                [查看全部 →]        │   │
│  │  任务名          │ 时间       │ 耗时  │ 大小   │ 状态        │   │
│  │  ───────────────┼───────────┼──────┼───────┼──────          │   │
│  │  生产 MySQL      │ 10:02:35  │ 2m3s │ 7.8GB │ ✅ 成功      │   │
│  │  生产 PostgreSQL │ 10:05:12  │ 1m8s │ 3.2GB │ ✅ 成功      │   │
│  │  测试 MongoDB    │ 10:00:05  │ 45s  │ 1.1GB │ ❌ 失败      │   │
│  │  ...                                                       │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ┌────────────────────────────┐  ┌──────────────────────────────┐   │
│  │  🚨 告警 (3)               │  │  ⏰ 即将过期的备份            │   │
│  │  ┌────────────────────────┐│  │  ┌────────────────────────┐ │   │
│  │  │ 🔴 MongoDB 连续失败3次 ││  │  │ 📋 old_backup.sql.gz   │ │   │
│  │  │    3小时前             ││  │  │    明天过期 (保留7天)   │ │   │
│  │  │ ⚠️ 存储使用率 > 80%   ││  │  │ 📋 test_dump.tar.gz    │ │   │
│  │  │    5小时前             ││  │  │    后天过期             │ │   │
│  │  └────────────────────────┘│  │  └────────────────────────┘ │   │
│  └────────────────────────────┘  └──────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.3 数据采集方案

**指标定义**：

| 指标 | 计算方式 | 采集频率 |
|------|---------|---------|
| 健康评分 | 4 维度加权计算 | 每 5 分钟 |
| 今日任务数 | COUNT(backup_records WHERE DATE(created_at) = TODAY) | 实时 |
| 成功率 | 成功数 / 总数 × 100%（近 7 天） | 每 5 分钟 |
| 平均耗时 | AVG(duration)（近 7 天成功任务） | 每 5 分钟 |
| 存储使用 | SUM(file_size) / 总空间 × 100% | 每 10 分钟 |
| 备份趋势 | 按天聚合成功/失败数 | 每 1 小时 |

**缓存策略**：

- 健康评分：Redis 缓存 5 分钟，key: `stats:health:{date}`
- 趋势数据：Redis 缓存 1 小时，key: `stats:trends:{range}`
- 存储数据：Redis 缓存 10 分钟，key: `stats:storage`
- 告警列表：Redis 缓存 1 分钟，key: `stats:alerts`

### 2.4 API 设计

#### GET /api/v1/stats/health

获取健康评分。

**响应（200）**：

```json
{
  "code": 0,
  "data": {
    "score": 87,
    "level": "good",
    "details": {
      "success_rate": {
        "score": 97,
        "weight": 0.4,
        "value": 0.957,
        "label": "成功率"
      },
      "rpo_compliance": {
        "score": 100,
        "weight": 0.2,
        "value": 1.0,
        "label": "RPO 达标"
      },
      "verification_coverage": {
        "score": 75,
        "weight": 0.2,
        "value": 0.6,
        "label": "验证覆盖率"
      },
      "storage_health": {
        "score": 84,
        "weight": 0.2,
        "value": 0.683,
        "label": "存储健康"
      }
    },
    "calculated_at": "2026-03-27T23:45:00+08:00"
  }
}
```

#### GET /api/v1/stats/trends

获取趋势数据。

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| range | string | 否 | `7d`（默认）/ `30d` |
| metric | string | 否 | `backup_count` / `success_rate` / `duration`（默认全部） |

**响应（200）**：

```json
{
  "code": 0,
  "data": {
    "range": "7d",
    "backup_count": [
      { "date": "2026-03-21", "total": 22, "success": 21, "failed": 1 },
      { "date": "2026-03-22", "total": 22, "success": 22, "failed": 0 },
      { "date": "2026-03-23", "total": 22, "success": 21, "failed": 1 },
      { "date": "2026-03-24", "total": 22, "success": 22, "failed": 0 },
      { "date": "2026-03-25", "total": 22, "success": 22, "failed": 0 },
      { "date": "2026-03-26", "total": 23, "success": 22, "failed": 1 },
      { "date": "2026-03-27", "total": 23, "success": 22, "failed": 1 }
    ],
    "success_rate": [
      { "date": "2026-03-21", "rate": 95.5 },
      { "date": "2026-03-22", "rate": 100.0 }
    ],
    "avg_duration": [
      { "date": "2026-03-21", "seconds": 178 },
      { "date": "2026-03-22", "seconds": 165 }
    ]
  }
}
```

#### GET /api/v1/stats/storage

获取存储分析。

**响应（200）**：

```json
{
  "code": 0,
  "data": {
    "total_space_gb": 76.0,
    "used_space_gb": 52.0,
    "free_space_gb": 24.0,
    "usage_percent": 68.4,
    "by_database_type": [
      { "type": "mysql", "size_gb": 28.5, "count": 42, "percent": 54.8 },
      { "type": "postgres", "size_gb": 15.2, "count": 35, "percent": 29.2 },
      { "type": "mongodb", "size_gb": 8.3, "count": 21, "percent": 16.0 }
    ],
    "expiring_soon": [
      { "record_id": 100, "job_name": "生产 MySQL", "file": "backup.sql.gz", "expires_at": "2026-03-28T02:00:00+08:00", "size_mb": 7800 },
      { "record_id": 101, "job_name": "测试 PG", "file": "pg_dump.sql.gz", "expires_at": "2026-03-29T03:00:00+08:00", "size_mb": 3200 }
    ]
  }
}
```

#### GET /api/v1/stats/alerts

获取告警列表。

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| level | string | 否 | `info` / `warning` / `critical` |
| status | string | 否 | `active` / `resolved` |
| page | int | 否 | 页码 |
| page_size | int | 否 | 每页数量 |

**响应（200）**：

```json
{
  "code": 0,
  "data": {
    "total": 3,
    "items": [
      {
        "id": 1,
        "level": "critical",
        "title": "MongoDB 备份连续失败 3 次",
        "message": "测试 MongoDB 任务在过去 3 次执行均失败，最近错误: connection refused",
        "job_id": 5,
        "status": "active",
        "created_at": "2026-03-27T20:00:00+08:00",
        "resolved_at": null
      }
    ]
  }
}
```

### 2.5 前端图表选型

**推荐 Recharts**（React 生态，轻量、声明式、TypeScript 友好）：

| 图表 | 组件 | 用途 |
|------|------|------|
| 备份趋势 | `AreaChart` | 7/30 天备份数量趋势（堆叠面积图） |
| 成功率趋势 | `LineChart` | 成功率变化曲线 |
| 存储趋势 | `BarChart` | 按周/月存储使用柱状图 |
| 数据库占比 | `PieChart` | 各数据库类型存储占比 |
| 健康评分 | 自定义组件 | 环形进度条 + 数字 |

---

## 3. 通知系统升级

### 3.1 功能描述

在现有 Webhook/钉钉/飞书通知基础上，扩展更多渠道，增加分级和升级策略，支持定期报告和静默期。

### 3.2 通知渠道扩展

| 渠道 | 配置项 | 优先级 |
|------|-------|--------|
| 钉钉 Webhook | webhook_url、secret | 已有 |
| 飞书 Webhook | webhook_url、secret | 已有 |
| Slack Webhook | webhook_url | 已有 |
| **邮件 SMTP** | smtp_host、smtp_port、username、password、from、to | 新增 |
| **企业微信** | webhook_url、corp_id、corp_secret、agent_id | 新增 |

### 3.3 通知分级

| 级别 | 含义 | 示例 | 默认渠道 |
|------|------|------|---------|
| **Info** | 正常信息通知 | 备份成功、定期报告 | 邮件 |
| **Warning** | 需要关注 | 备份失败 1 次、存储空间不足 | 钉钉 + 邮件 |
| **Critical** | 需要立即处理 | 连续失败 3 次、恢复到生产库 | 全部渠道 |

### 3.4 告警升级策略

```
失败 1 次  → Warning  → 通知: 任务负责人
失败 2 次  → Warning  → 通知: 任务负责人
失败 3 次  → Critical → 通知: 任务负责人 + 管理员
失败 5 次  → Critical → 通知: 任务负责人 + 管理员 + 上级领导
恢复成功   → Info     → 通知: 任务负责人（告知问题已恢复）
```

**升级策略配置**：

```json
{
  "escalation_policy": [
    {
      "failure_count": 1,
      "level": "warning",
      "notify_roles": ["operator"],
      "channels": ["email"]
    },
    {
      "failure_count": 3,
      "level": "critical",
      "notify_roles": ["operator", "admin"],
      "channels": ["email", "dingtalk", "feishu"]
    },
    {
      "failure_count": 5,
      "level": "critical",
      "notify_roles": ["operator", "admin", "manager"],
      "channels": ["email", "dingtalk", "feishu"]
    }
  ]
}
```

### 3.5 定期备份报告

| 报告类型 | 发送时间 | 内容 |
|---------|---------|------|
| **每日摘要** | 每天 08:00 | 昨日备份概况（总数/成功/失败/存储变化） |
| **每周汇总** | 周一 09:00 | 本周备份统计、成功率趋势、异常事件 |
| **月度报告** | 每月 1 日 10:00 | 全月统计、健康评分趋势、存储分析、改进建议 |

### 3.6 通知模板

内置通知模板，支持自定义修改：

| 模板 | 变量 | 示例 |
|------|------|------|
| backup_success | `{task_name}`, `{file_size}`, `{duration}`, `{time}` | ✅ 备份成功: 生产 MySQL, 7.8GB, 2m35s |
| backup_failed | `{task_name}`, `{error}`, `{time}`, `{failure_count}` | ❌ 备份失败: 测试 MongoDB, connection refused (第3次) |
| restore_executed | `{task_name}`, `{target}`, `{operator}`, `{reason}` | ⚠️ 恢复执行: 生产 MySQL → restore_db, 操作人: admin |
| daily_report | `{date}`, `{total}`, `{success}`, `{failed}`, `{storage}` | 📊 每日备份摘要 (03-27)... |

### 3.7 通知静默期

配置维护窗口，静默期内不发送 Info 和 Warning 级别通知（Critical 不受影响）。

```json
{
  "silence_periods": [
    {
      "name": "周末维护窗口",
      "cron": "0 2 * * 6,0",
      "duration_minutes": 240,
      "skip_levels": ["info", "warning"]
    }
  ]
}
```

### 3.8 API 设计

#### 通知渠道

##### GET /api/v1/notify/channels

```json
{
  "code": 0,
  "data": {
    "channels": [
      { "type": "email", "enabled": true, "configured": true, "name": "邮件 SMTP" },
      { "type": "dingtalk", "enabled": true, "configured": true, "name": "钉钉" },
      { "type": "feishu", "enabled": false, "configured": false, "name": "飞书" },
      { "type": "wecom", "enabled": false, "configured": false, "name": "企业微信" }
    ]
  }
}
```

##### PUT /api/v1/notify/channels/:type

```json
// 请求 - 配置邮件 SMTP
{
  "enabled": true,
  "config": {
    "smtp_host": "smtp.example.com",
    "smtp_port": 465,
    "username": "backup@example.com",
    "password": "smtp_secret",
    "from": "backup@example.com",
    "to": ["dba@example.com", "ops@example.com"],
    "use_ssl": true
  }
}
```

#### 通知规则

##### GET /api/v1/notify/rules

##### POST /api/v1/notify/rules

```json
{
  "name": "生产任务失败告警",
  "event": "backup_failed",
  "conditions": {
    "job_tags": ["production"],
    "failure_count_min": 1
  },
  "level": "warning",
  "channels": ["email", "dingtalk"],
  "recipients": ["admin@example.com"],
  "escalation_enabled": true
}
```

##### PUT /api/v1/notify/rules/:id

##### DELETE /api/v1/notify/rules/:id

#### 通知历史

##### GET /api/v1/notify/history

```json
{
  "code": 0,
  "data": {
    "total": 128,
    "items": [
      {
        "id": 1,
        "level": "warning",
        "title": "备份失败: 测试 MongoDB",
        "channel": "dingtalk",
        "recipient": "admin@example.com",
        "status": "sent",
        "error": null,
        "created_at": "2026-03-27T20:00:00+08:00"
      }
    ]
  }
}
```

#### 通知模板

##### GET /api/v1/notify/templates

##### PUT /api/v1/notify/templates/:id

```json
{
  "subject": "❌ 备份失败: {{task_name}}",
  "body": "任务 {{task_name}} 备份失败\n错误: {{error}}\n时间: {{time}}\n连续失败: {{failure_count}} 次",
  "enabled": true
}
```

### 3.9 数据库设计

```sql
-- 通知渠道配置
CREATE TABLE notify_channels (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    type            VARCHAR(30)     NOT NULL COMMENT '渠道类型: email/dingtalk/feishu/wecom/slack',
    name            VARCHAR(50)     NOT NULL COMMENT '渠道名称',
    enabled         TINYINT(1)      DEFAULT 0 COMMENT '是否启用',
    config          JSON            COMMENT '渠道配置(加密存储)',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE KEY uk_type (type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='通知渠道配置';

-- 通知规则
CREATE TABLE notification_rules (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name            VARCHAR(128)    NOT NULL COMMENT '规则名称',
    event           VARCHAR(50)     NOT NULL COMMENT '触发事件: backup_failed/backup_success/restore_executed/daily_report',
    conditions      JSON            COMMENT '触发条件',
    level           VARCHAR(20)     DEFAULT 'info' COMMENT '通知级别: info/warning/critical',
    channels        JSON            NOT NULL COMMENT '通知渠道列表',
    recipients      JSON            COMMENT '接收人列表',
    escalation_enabled TINYINT(1)   DEFAULT 0 COMMENT '是否启用升级策略',
    enabled         TINYINT(1)      DEFAULT 1 COMMENT '是否启用',
    created_by      BIGINT UNSIGNED NULL,
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_event (event)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='通知规则';

-- 通知历史
CREATE TABLE notification_history (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    rule_id         BIGINT UNSIGNED NULL COMMENT '关联的规则 ID',
    level           VARCHAR(20)     NOT NULL COMMENT '通知级别',
    title           VARCHAR(256)    NOT NULL COMMENT '通知标题',
    content         TEXT            COMMENT '通知内容',
    channel         VARCHAR(30)     NOT NULL COMMENT '发送渠道',
    recipient       VARCHAR(256)    COMMENT '接收人',
    status          VARCHAR(20)     DEFAULT 'pending' COMMENT 'pending/sent/failed',
    error           VARCHAR(512)    COMMENT '发送错误信息',
    related_type    VARCHAR(50)     COMMENT '关联资源类型: job/record/restore',
    related_id      BIGINT UNSIGNED COMMENT '关联资源 ID',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_rule_id (rule_id),
    INDEX idx_level (level),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='通知历史记录';

-- 通知模板
CREATE TABLE notify_templates (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    type            VARCHAR(50)     NOT NULL COMMENT '模板类型: backup_success/backup_failed/restore_executed/daily_report',
    name            VARCHAR(128)    NOT NULL COMMENT '模板名称',
    subject         VARCHAR(256)    COMMENT '标题模板(支持变量)',
    body            TEXT            NOT NULL COMMENT '内容模板(支持变量)',
    enabled         TINYINT(1)      DEFAULT 1,
    is_builtin      TINYINT(1)      DEFAULT 1 COMMENT '是否内置模板',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE KEY uk_type (type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='通知模板';

-- 通知静默期
CREATE TABLE notify_silence_periods (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name            VARCHAR(128)    NOT NULL COMMENT '静默期名称',
    cron            VARCHAR(64)     NOT NULL COMMENT 'Cron 表达式(静默开始时间)',
    duration_minutes INT            DEFAULT 120 COMMENT '持续分钟数',
    skip_levels     JSON            COMMENT '跳过的通知级别',
    enabled         TINYINT(1)      DEFAULT 1,
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='通知静默期配置';
```

### 3.10 前端交互

**通知渠道配置页面**：

```
┌─────────────────────────────────────────────────────────────┐
│  通知渠道                                                    │
│                                                             │
│  ┌─ 邮件 SMTP ──────────────── [已启用 ✅] ──────────────┐  │
│  │  SMTP 服务器: [smtp.example.com]  端口: [465]         │  │
│  │  用户名: [backup@example.com]     SSL: [✅]           │  │
│  │  发件人: [backup@example.com]                         │  │
│  │  收件人: [dba@example.com, ops@example.com]           │  │
│  │  [测试发送]                        [保存配置]          │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌─ 钉钉 ──────────────────── [已启用 ✅] ──────────────┐  │
│  │  Webhook URL: [https://oapi.dingtalk.com/robot/...]   │  │
│  │  Secret: [••••••••]                                   │  │
│  │  [测试发送]                        [保存配置]          │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌─ 企业微信 ───────────────── [未配置 ⚪] ─────────────┐  │
│  │  [配置渠道 →]                                         │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### 3.11 权限要求

| 操作 | 所需权限 |
|------|---------|
| 查看通知配置 | 登录即可 |
| 修改通知渠道/规则 | `notify:manage` |
| 查看通知历史 | 登录即可 |
| 自定义通知模板 | `notify:manage` |

---

## 4. 操作审计日志

### 4.1 功能描述

记录系统中所有关键操作的审计日志，支持查询和筛选，满足安全合规要求。

### 4.2 审计范围

| 操作类型 | 资源 | 记录内容 |
|---------|------|---------|
| 任务创建 | backup_job | 任务名称、数据库类型、主机 |
| 任务修改 | backup_job | 修改前后的差异 |
| 任务删除 | backup_job | 被删除的任务信息快照 |
| 任务启停 | backup_job | 启用/禁用状态 |
| 手动执行 | backup_job | 触发来源 |
| 恢复执行 | restore_record | 恢复详情、确认码 |
| 用户创建 | user | 新用户信息 |
| 用户修改 | user | 修改字段 |
| 角色权限变更 | role/permission | 变更详情 |
| 登录 | auth | 登录方式、IP、UA |
| 登出 | auth | - |
| 配置变更 | system_config | 配置项、修改前后值 |
| 通知规则变更 | notification_rule | 变更详情 |

### 4.3 日志字段

```go
type AuditLog struct {
    ID           uint      `gorm:"primaryKey" json:"id"`
    OperatorID   uint      `json:"operator_id"`       // 操作人 ID
    OperatorName string    `json:"operator_name"`     // 操作人用户名
    Action       string    `json:"action"`            // 操作类型: create/update/delete/login/logout/execute
    ResourceType string    `json:"resource_type"`     // 资源类型: backup_job/user/role/restore/config
    ResourceID   string    `json:"resource_id"`       // 资源 ID
    ResourceName string    `json:"resource_name"`     // 资源名称（便于阅读）
    Detail       string    `gorm:"type:text" json:"detail"` // 操作详情（JSON 格式）
    IPAddress    string    `json:"ip_address"`
    UserAgent    string    `json:"user_agent"`
    CreatedAt    time.Time `json:"created_at"`
}
```

### 4.4 API 设计

#### GET /api/v1/audit-logs

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| action | string | 否 | 操作类型筛选 |
| resource_type | string | 否 | 资源类型筛选 |
| operator_id | int | 否 | 操作人 ID |
| start_time | string | 否 | 开始时间 (ISO 8601) |
| end_time | string | 否 | 结束时间 |
| keyword | string | 否 | 关键词搜索（匹配 resource_name、detail） |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页数量，默认 20，最大 100 |

**响应（200）**：

```json
{
  "code": 0,
  "data": {
    "total": 256,
    "items": [
      {
        "id": 256,
        "operator_id": 1,
        "operator_name": "admin",
        "action": "execute",
        "resource_type": "restore",
        "resource_id": "1",
        "resource_name": "恢复: 生产 MySQL → restore_db",
        "detail": "{\"confirm_code\":\"8F3K2A\",\"reason\":\"数据误删\",\"backup_time\":\"2026-03-27T02:00:00+08:00\"}",
        "ip_address": "192.168.1.50",
        "user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
        "created_at": "2026-03-27T15:30:00+08:00"
      },
      {
        "id": 255,
        "operator_id": 2,
        "operator_name": "dba",
        "action": "update",
        "resource_type": "backup_job",
        "resource_id": "5",
        "resource_name": "生产 MySQL",
        "detail": "{\"changes\":{\"schedule\":{\"from\":\"0 2 * * *\",\"to\":\"0 3 * * *\"},\"retention_days\":{\"from\":7,\"to\":14}}}",
        "ip_address": "192.168.1.60",
        "user_agent": "Mozilla/5.0 ...",
        "created_at": "2026-03-27T14:20:00+08:00"
      },
      {
        "id": 254,
        "operator_id": 1,
        "operator_name": "admin",
        "action": "login",
        "resource_type": "auth",
        "resource_id": "1",
        "resource_name": "用户登录",
        "detail": "{\"method\":\"password\"}",
        "ip_address": "192.168.1.50",
        "user_agent": "Mozilla/5.0 ...",
        "created_at": "2026-03-27T09:00:00+08:00"
      }
    ]
  }
}
```

### 4.5 数据库设计

```sql
-- 在 Phase 2 的 audit_logs 基础上增强
CREATE TABLE audit_logs (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    operator_id     BIGINT UNSIGNED NOT NULL COMMENT '操作人 ID',
    operator_name   VARCHAR(50)     NOT NULL COMMENT '操作人用户名',
    action          VARCHAR(30)     NOT NULL COMMENT '操作类型: create/update/delete/enable/disable/execute/login/logout/config_change',
    resource_type   VARCHAR(30)     NOT NULL COMMENT '资源类型: backup_job/backup_record/restore/user/role/notification_rule/config/auth',
    resource_id     VARCHAR(64)     NOT NULL DEFAULT '' COMMENT '资源 ID',
    resource_name   VARCHAR(256)    DEFAULT '' COMMENT '资源名称(便于阅读)',
    detail          TEXT            COMMENT '操作详情(JSON)',
    ip_address      VARCHAR(45)     DEFAULT '' COMMENT '客户端 IP',
    user_agent      VARCHAR(512)    DEFAULT '' COMMENT 'User-Agent',
    created_at      DATETIME        DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_operator_id (operator_id),
    INDEX idx_action (action),
    INDEX idx_resource_type (resource_type),
    INDEX idx_created_at (created_at),
    INDEX idx_resource (resource_type, resource_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='操作审计日志';
```

### 4.6 日志保留策略

| 策略 | 值 |
|------|-----|
| 默认保留天数 | 90 天 |
| 可配置范围 | 30-365 天 |
| 清理方式 | 每日凌晨定时任务清理过期日志 |
| Critical 级别操作 | 永久保留（不自动清理） |

### 4.7 前端交互

**审计日志查询页面**：

```
┌─────────────────────────────────────────────────────────────┐
│  🗄️ DB Backup          [仪表盘] [任务] [审计日志] [设置]     │
│                                                             │
│  筛选条件                                                    │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ 操作类型: [全部 ▼]  资源类型: [全部 ▼]  操作人: [全部 ▼]│  │
│  │ 时间范围: [2026-03-20] ~ [2026-03-27]                  │  │
│  │ 关键词: [________________]           [搜索] [重置]     │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ 时间              │ 操作人  │ 操作   │ 资源     │ 详情 │  │
│  ├───────────────────┼────────┼────────┼─────────┼──────┤  │
│  │ 03-27 15:30:00   │ admin  │ ⚡执行  │ 恢复     │ 展开 │  │
│  │ 03-27 14:20:00   │ dba    │ ✏️修改  │ 备份任务 │ 展开 │  │
│  │ 03-27 09:00:00   │ admin  │ 🔑登录  │ 认证     │ 展开 │  │
│  │ 03-26 18:00:00   │ dba    │ 🗑️删除  │ 备份任务 │ 展开 │  │
│  │ 03-26 10:00:00   │ admin  │ ➕创建  │ 用户     │ 展开 │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│  共 256 条记录                                 第 1/13 页   │
│  [← 上一页]                               [下一页 →]        │
└─────────────────────────────────────────────────────────────┘
```

**展开详情**：

```
┌─ 操作详情 ─────────────────────────────────────────────────┐
│  操作人: admin (ID: 1)                                     │
│  操作: 修改 (update)                                       │
│  资源: 备份任务 #5 "生产 MySQL"                             │
│  IP: 192.168.1.50                                         │
│  UA: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)      │
│                                                            │
│  变更内容:                                                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  字段            │ 修改前          │ 修改后          │  │
│  │  schedule        │ 0 2 * * *      │ 0 3 * * *      │  │
│  │  retention_days  │ 7              │ 14             │  │
│  └──────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────┘
```

### 4.8 权限要求

| 操作 | 所需权限 |
|------|---------|
| 查看审计日志 | 登录即可（仅可查看自己的操作） |
| 查看所有审计日志 | `audit:view_all` |
| 导出审计日志 | `audit:export` |
| 删除审计日志 | 不允许（仅系统自动清理） |

---

## 5. 路由汇总

```go
// 仪表盘统计
stats := v1.Group("/stats")
{
    statsHandler := handler.NewStatsHandler(db)
    stats.GET("/health", statsHandler.Health)
    stats.GET("/trends", statsHandler.Trends)
    stats.GET("/storage", statsHandler.Storage)
    stats.GET("/alerts", statsHandler.Alerts)
}

// 通知管理
notify := v1.Group("/notify")
{
    notifyHandler := handler.NewNotifyHandler(db)
    notify.GET("/channels", notifyHandler.ListChannels)
    notify.PUT("/channels/:type", notifyHandler.UpdateChannel)
    notify.GET("/rules", notifyHandler.ListRules)
    notify.POST("/rules", notifyHandler.CreateRule)
    notify.PUT("/rules/:id", notifyHandler.UpdateRule)
    notify.DELETE("/rules/:id", notifyHandler.DeleteRule)
    notify.GET("/history", notifyHandler.ListHistory)
    notify.GET("/templates", notifyHandler.ListTemplates)
    notify.PUT("/templates/:id", notifyHandler.UpdateTemplate)
}

// 审计日志
audit := v1.Group("/audit-logs")
{
    auditHandler := handler.NewAuditHandler(db)
    audit.GET("", auditHandler.List)
}
```

---

## 6. 实现优先级

| 优先级 | 模块 | 预估工时 |
|-------|------|---------|
| P0 | 健康评分 + 关键指标 API | 2 天 |
| P0 | 审计日志（核心记录 + 查询） | 2 天 |
| P1 | 仪表盘前端（趋势图 + 布局） | 3 天 |
| P1 | 通知分级 + 告警升级策略 | 2 天 |
| P1 | 通知渠道扩展（邮件 + 企业微信） | 2 天 |
| P2 | 通知模板自定义 + 静默期 | 1 天 |
| P2 | 定期备份报告 | 1 天 |
| P2 | 审计日志前端（查询 + 导出） | 1 天 |
