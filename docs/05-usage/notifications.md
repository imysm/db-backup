# 通知配置

本文档介绍如何配置钉钉、飞书、企业微信、邮件等通知渠道。

## 概述

db-backup 支持多种通知方式，在备份完成或失败时发送通知。

## 通知类型

| 类型 | 触发时机 | 说明 |
|------|----------|------|
| `success` | 备份成功 | 发送成功通知 |
| `failed` | 备份失败 | 发送失败告警 |
| `both` | 成功或失败 | 都发送通知 |

---

## 钉钉

### 创建机器人

1. 打开钉钉群 → 群设置 → 智能群助手
2. 添加机器人 → 自定义
3. 设置机器人名称和头像
4. 安全设置选择"自定义关键词"，添加"备份"
5. 复制 Webhook URL

### 配置

```yaml
notify:
  dingtalk:
    enabled: true
    webhook: https://oapi.dingtalk.com/robot/send?access_token=xxx
    secret: SECxxx  # 可选，加签密钥
    at_mobiles:     # @ 手机号
      - "13800138000"
    at_all: false   # 是否 @ 所有人
```

### 消息模板

```
【备份成功】
任务: mysql-prod-backup
数据库: mydb
文件: mydb_20260320.sql.gz
大小: 1024 MB
耗时: 120 秒
```

---

## 飞书

### 创建机器人

1. 打开飞书群 → 设置 → 群机器人
2. 添加机器人 → 自定义机器人
3. 复制 Webhook URL

### 配置

```yaml
notify:
  feishu:
    enabled: true
    webhook: https://open.feishu.cn/open-apis/bot/v2/hook/xxx
    secret: xxx  # 可选，签名密钥
```

### 消息卡片

飞书支持富文本消息卡片：

```yaml
notify:
  feishu:
    enabled: true
    webhook: https://open.feishu.cn/open-apis/bot/v2/hook/xxx
    card:
      title: "备份通知"
      color: "blue"  # blue/green/red/orange
```

---

## 企业微信

### 创建机器人

1. 打开企业微信群 → 右键 → 添加群机器人
2. 新建机器人
3. 复制 Webhook URL

### 配置

```yaml
notify:
  wechat:
    enabled: true
    webhook: https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx
    mentioned_list:   # @ 用户
      - "user1"
      - "user2"
    mentioned_mobile_list:  # @ 手机号
      - "13800138000"
```

---

## 邮件

### SMTP 配置

```yaml
notify:
  email:
    enabled: true
    smtp_host: smtp.example.com
    smtp_port: 465
    smtp_user: noreply@example.com
    smtp_password: ${SMTP_PASSWORD}
    smtp_ssl: true
    from: noreply@example.com
    to:
      - ops@example.com
      - dba@example.com
```

### 邮件模板

```
主题: [db-backup] 备份任务执行结果

任务 ID: mysql-prod-backup
状态: 成功
数据库: mydb
文件: mydb_20260320.sql.gz
大小: 1024 MB
耗时: 120 秒
开始时间: 2026-03-20 02:00:00
结束时间: 2026-03-20 02:02:00
```

---

## Webhook

自定义 HTTP 回调：

```yaml
notify:
  webhook:
    enabled: true
    url: https://api.example.com/webhook/backup
    method: POST
    headers:
      Authorization: "Bearer xxx"
      Content-Type: "application/json"
    timeout: 10s
```

### 请求格式

```json
{
  "event": "backup_completed",
  "job_id": "mysql-prod-backup",
  "status": "success",
  "timestamp": "2026-03-20T02:02:00Z",
  "data": {
    "file_path": "/data/backups/mysql/mydb_20260320.sql.gz",
    "file_size": 1073741824,
    "duration": 120
  }
}
```

---

## 任务级配置

可以为每个任务单独配置通知：

```yaml
tasks:
  - id: mysql-prod-backup
    database:
      type: mysql
      host: localhost
      port: 3306
      username: root
      password: ${MYSQL_PASSWORD}
      database: mydb
    schedule: "0 2 * * *"
    notify:
      on_success:
        - feishu
      on_failure:
        - dingtalk
        - email
```

### 通知规则

| 配置 | 说明 |
|------|------|
| `on_success` | 备份成功时发送通知 |
| `on_failure` | 备份失败时发送通知 |

---

## 通知模板

### 全局模板

```yaml
notify:
  templates:
    success:
      title: "备份成功"
      body: |
        任务: {{.JobID}}
        数据库: {{.Database}}
        文件: {{.FilePath}}
        大小: {{.FileSize}}
        耗时: {{.Duration}}秒
    failure:
      title: "备份失败"
      body: |
        任务: {{.JobID}}
        错误: {{.Error}}
        时间: {{.Timestamp}}
```

### 模板变量

| 变量 | 说明 |
|------|------|
| `{{.JobID}}` | 任务 ID |
| `{{.Database}}` | 数据库名 |
| `{{.Status}}` | 状态 |
| `{{.FilePath}}` | 备份文件路径 |
| `{{.FileSize}}` | 文件大小 |
| `{{.Duration}}` | 执行时长 |
| `{{.Error}}` | 错误信息 |
| `{{.Timestamp}}` | 时间戳 |

---

## 环境变量

推荐使用环境变量存储敏感信息：

```yaml
notify:
  dingtalk:
    webhook: ${DINGTALK_WEBHOOK}
    secret: ${DINGTALK_SECRET}
  email:
    smtp_password: ${SMTP_PASSWORD}
```

---

## 测试通知

### CLI 测试

```bash
./db-backup -config configs/config.yaml -test-notify
```

### API 测试

```bash
curl -X POST http://localhost:8080/api/notify/test
```

---

## 最佳实践

### 1. 分级通知

- **测试环境**: 仅飞书通知
- **生产环境**: 失败时钉钉 + 邮件 + 短信

### 2. 值班轮换

配合值班系统，@ 当值人员：

```yaml
notify:
  dingtalk:
    at_mobiles:
      - "{{.OnDuty}}"
```

### 3. 告警聚合

避免频繁告警：

- 相同任务 1 小时内只告警一次
- 聚合多个失败任务

---

*最后更新: 2026-03-20*
