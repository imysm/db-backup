# 安全功能文档

本文档详细介绍数据库备份系统的安全功能和使用指南。

## 🔐 加密功能

### AES-256-GCM 加密

系统使用 AES-256-GCM 算法进行文件加密，这是目前业界标准的对称加密算法。

**特性：**
- 256 位密钥强度
- GCM 模式提供认证加密（AEAD）
- 每块使用独立随机 nonce
- 支持密钥哈希验证

### 密钥配置

加密密钥必须是 **32 字节**（64 个 hex 字符）的字符串。

**配置示例：**

```yaml
storage:
  encryption:
    enabled: true
    type: aes256
    key: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
```

**从环境变量读取密钥：**

```yaml
storage:
  encryption:
    enabled: true
    type: aes256
    key_env: "DB_BACKUP_ENCRYPTION_KEY"  # 从环境变量读取
```

**密钥生成示例：**

```bash
# 生成 32 字节随机密钥（hex 编码）
openssl rand -hex 32
```

### 加密备份流程

```
数据库 → mysqldump/pg_dump/mongodump → 压缩 → 加密 → 上传存储
```

### 解密恢复流程

```
下载存储 → 解密 → 解压 → mysql/pg_restore/mongorestore → 数据库
```

### 验证加密文件

```go
// 使用 VerifyEncryptedFile 验证密钥正确性
valid, err := executor.VerifyEncryptedFile("/path/to/backup.enc", keyHex, expectedKeyHash)
if err != nil {
    log.Printf("验证失败: %v", err)
}
```

## 🛡️ SQL 注入防护

### 数据库名消毒

所有数据库名在拼接到命令行前都会经过消毒处理：

```go
// sanitizeDBName 移除危险字符
func sanitizeDBName(name string) string {
    // 移除 / \ 空格 & | ; $ ` ' " ( ) { } < > 等危险字符
    name = strings.ReplaceAll(name, "'", "_")
    name = strings.ReplaceAll(name, ";", "_")
    // ... 更多字符
    return name
}
```

### 参数验证

使用正则表达式白名单验证：

```go
// ValidateDatabaseName 只允许字母、数字、下划线、连字符
re := regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)
if !re.MatchString(name) {
    return fmt.Errorf("数据库名包含非法字符")
}
```

## 🔑 密码安全

### MySQL 密码传递

**方式：** 使用 `--defaults-extra-file` 通过临时配置文件传递密码

```go
// 创建临时 CNF 文件（权限 0600）
tmpFile, err := os.CreateTemp("", "mysql_defaults_*.cnf")
os.Chmod(tmpFile.Name(), 0600)
fmt.Fprintf(tmpFile, "[client]\npassword=%s\n", password)
```

**为什么不用命令行参数？**
- `mysqldump -ppassword` 会暴露在 `ps aux` 输出中
- CNF 文件权限 0600 只允许当前用户读取

### SQL Server 密码传递

**方式：** 使用环境变量 `SQLCMDPASSWORD`

```go
env := []string{fmt.Sprintf("SQLCMDPASSWORD=%s", password)}
cmd := exec.Command("sqlcmd", args...)
cmd.Env = append(os.Environ(), env...)
```

## 📁 路径安全

### 路径遍历防护

```go
// ValidateBackupPath 防止路径遍历
func ValidateBackupPath(path string) error {
    if strings.Contains(path, "..") {
        return fmt.Errorf("备份路径不能包含路径遍历字符")
    }
    // 白名单字符验证
    re := regexp.MustCompile(`^[a-zA-Z0-9_\-/.\s]+$`)
    if !re.MatchString(path) {
        return fmt.Errorf("备份路径包含非法字符")
    }
    return nil
}
```

### 文件路径拼接

使用 `filepath.Join` 确保路径安全：

```go
// filepath.Join 会清理路径中的 .. 等字符
filePath := filepath.Join(storagePath, fileName)
```

## 🚫 登录安全

### 登录频率限制

防止暴力破解：

```go
// 5 次失败后锁定 15 分钟
limiter := auth.NewLoginRateLimiter(5, 15*time.Minute, 15*time.Minute)

// 检查是否锁定
if limiter.IsLocked(username) {
    return fmt.Errorf("账户已锁定，请 15 分钟后再试")
}

// 记录失败
locked := limiter.RecordFailure(username)
if locked {
    return fmt.Errorf("登录失败次数过多，账户已锁定")
}

// 登录成功
limiter.RecordSuccess(username)
```

### goroutine 泄漏修复

```go
func (l *LoginRateLimiter) Stop() {
    close(l.stopCleanup)    // 通知清理协程退出
    l.stopWg.Wait()         // 等待清理协程真正退出
}
```

## ☁️ 云存储安全

### COS 签名 URL

生成带时效的下载链接：

```go
// GetSignedURL 生成签名 URL
signedURL, err := cosStorage.GetSignedURL(ctx, key, 300) // 5 分钟有效期
if err != nil {
    return fmt.Errorf("生成签名URL失败: %w", err)
}
```

**安全特性：**
- 无密钥时返回明确错误（不会静默失败）
- 可设置过期时间
- 签名不可伪造

## 📝 审计日志

系统记录所有敏感操作：

```go
// 审计日志示例
audit.Log(authUser, "backup.create", map[string]interface{}{
    "job_id":   jobID,
    "database": dbName,
    "result":   "success",
})
```

## ⚠️ 安全最佳实践

### 1. 密钥管理

- ✅ 使用 32 字节随机密钥
- ✅ 密钥存储在环境变量或密钥管理服务
- ✅ 定期轮换密钥
- ❌ 不要把密钥写在配置文件里

### 2. 密码保护

- ✅ 使用强密码
- ✅ 定期更换密码
- ✅ 使用环境变量传递密码
- ❌ 不要在命令行中明文传递密码

### 3. 访问控制

- ✅ 限制数据库用户权限（只给必要权限）
- ✅ 限制备份文件目录权限（700 或 750）
- ✅ 使用防火墙限制访问来源

### 4. 监控告警

- ✅ 配置备份失败告警
- ✅ 监控异常登录行为
- ✅ 定期检查审计日志

## 🔒 安全配置检查清单

上线前请确认：

- [ ] 加密密钥已正确配置（32 字节 hex）
- [ ] 数据库密码使用环境变量
- [ ] 备份文件目录权限正确（700）
- [ ] 登录频率限制已启用
- [ ] 告警配置正确
- [ ] 审计日志已启用
