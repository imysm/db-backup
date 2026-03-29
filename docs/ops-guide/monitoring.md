# 监控告警

本文档介绍如何配置 Prometheus + Grafana 监控和告警。

## 概述

db-backup 内置 Prometheus 指标，可通过 `/metrics` 端点暴露。

---

## Prometheus 指标

### 访问指标

```bash
curl http://localhost:8080/metrics
```

### 可用指标

| 指标名称 | 类型 | 标签 | 说明 |
|----------|------|------|------|
| `db_backup_tasks_total` | Gauge | - | 任务总数 |
| `db_backup_runs_total` | Counter | status, job_id | 备份执行次数 |
| `db_backup_duration_seconds` | Histogram | job_id, status | 备份耗时 |
| `db_backup_size_bytes` | Histogram | job_id | 备份文件大小 |
| `db_backup_errors_total` | Counter | job_id, error_type | 错误次数 |
| `db_backup_last_success_timestamp` | Gauge | job_id | 最后成功时间 |
| `db_backup_storage_bytes` | Gauge | storage_type | 存储使用量 |

### 指标示例

```
# HELP db_backup_runs_total Total number of backup runs
# TYPE db_backup_runs_total counter
db_backup_runs_total{job_id="mysql-prod-backup",status="success"} 95
db_backup_runs_total{job_id="mysql-prod-backup",status="failed"} 5

# HELP db_backup_duration_seconds Duration of backup in seconds
# TYPE db_backup_duration_seconds histogram
db_backup_duration_seconds_bucket{job_id="mysql-prod-backup",le="60"} 20
db_backup_duration_seconds_bucket{job_id="mysql-prod-backup",le="120"} 50
db_backup_duration_seconds_bucket{job_id="mysql-prod-backup",le="300"} 90
db_backup_duration_seconds_bucket{job_id="mysql-prod-backup",le="+Inf"} 100
db_backup_duration_seconds_sum{job_id="mysql-prod-backup"} 12000
db_backup_duration_seconds_count{job_id="mysql-prod-backup"} 100

# HELP db_backup_size_bytes Size of backup file in bytes
# TYPE db_backup_size_bytes histogram
db_backup_size_bytes_bucket{job_id="mysql-prod-backup",le="1.048576e+06"} 10
db_backup_size_bytes_bucket{job_id="mysql-prod-backup",le="1.073741824e+09"} 50
db_backup_size_bytes_bucket{job_id="mysql-prod-backup",le="+Inf"} 100
```

---

## Prometheus 配置

### prometheus.yml

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'db-backup'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: /metrics

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093

rule_files:
  - /etc/prometheus/rules/*.yml
```

### Docker Compose

```yaml
version: '3.8'

services:
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - ./rules:/etc/prometheus/rules
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'

  alertmanager:
    image: prom/alertmanager:latest
    ports:
      - "9093:9093"
    volumes:
      - ./alertmanager.yml:/etc/alertmanager/alertmanager.yml

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    volumes:
      - grafana-data:/var/lib/grafana
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin

volumes:
  grafana-data:
```

---

## 告警规则

### rules/backup.yml

```yaml
groups:
  - name: db-backup
    rules:
      # 备份失败告警
      - alert: BackupFailed
        expr: increase(db_backup_runs_total{status="failed"}[1h]) > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "备份任务失败"
          description: "任务 {{ $labels.job_id }} 备份失败"

      # 备份超时告警
      - alert: BackupTimeout
        expr: db_backup_duration_seconds_bucket{le="+Inf"} - db_backup_duration_seconds_bucket{le="3600"} > 0
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "备份任务超时"
          description: "任务 {{ $labels.job_id }} 备份时间超过 1 小时"

      # 备份文件过大告警
      - alert: BackupSizeLarge
        expr: db_backup_size_bytes_bucket{le="+Inf"} > 10737418240
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "备份文件过大"
          description: "任务 {{ $labels.job_id }} 备份文件超过 10GB"

      # 长时间无成功备份告警
      - alert: NoBackupSuccess
        expr: time() - db_backup_last_success_timestamp > 86400
        for: 1h
        labels:
          severity: critical
        annotations:
          summary: "长时间无成功备份"
          description: "任务 {{ $labels.job_id }} 超过 24 小时无成功备份"

      # 存储空间不足告警
      - alert: StorageSpaceLow
        expr: db_backup_storage_bytes / 107374182400 > 0.9
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "备份存储空间不足"
          description: "备份存储使用率超过 90%"
```

---

## Alertmanager 配置

### alertmanager.yml

```yaml
global:
  resolve_timeout: 5m
  smtp_smarthost: 'smtp.example.com:465'
  smtp_from: 'alert@example.com'
  smtp_auth_username: 'alert@example.com'
  smtp_auth_password: '${SMTP_PASSWORD}'

route:
  group_by: ['alertname', 'job_id']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  receiver: 'default'
  routes:
    - match:
        severity: critical
      receiver: 'critical'
    - match:
        severity: warning
      receiver: 'warning'

receivers:
  - name: 'default'
    email_configs:
      - to: 'ops@example.com'

  - name: 'critical'
    email_configs:
      - to: 'ops@example.com,dba@example.com'
    webhook_configs:
      - url: 'https://oapi.dingtalk.com/robot/send?access_token=xxx'
        send_resolved: true

  - name: 'warning'
    email_configs:
      - to: 'ops@example.com'
```

---

## Grafana 仪表盘

### 导入仪表盘

1. 打开 Grafana → Dashboards → Import
2. 输入仪表盘 ID 或上传 JSON
3. 选择 Prometheus 数据源
4. 点击 Import

### 关键面板

| 面板 | 说明 |
|------|------|
| 备份成功率 | 成功/失败比例 |
| 备份耗时趋势 | 平均/最大/最小耗时 |
| 备份文件大小 | 文件大小趋势 |
| 任务执行频率 | 每日执行次数 |
| 存储使用量 | 存储空间使用趋势 |

---

## 健康检查

### HTTP 端点

```bash
# 健康检查
curl http://localhost:8080/health

# 就绪检查
curl http://localhost:8080/ready
```

### Kubernetes 配置

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

---

*最后更新: 2026-03-20*
