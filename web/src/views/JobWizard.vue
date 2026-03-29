<template>
  <div class="job-wizard">
    <el-card>
      <template #header>
        <div class="wizard-header">
          <el-page-header @back="handleBack" title="返回">
            <template #content>
              <span class="header-title">{{ isEdit ? '编辑任务' : '新建备份任务' }}</span>
            </template>
          </el-page-header>
        </div>
      </template>

      <!-- 步骤条 -->
      <el-steps :active="currentStep" finish-status="success" class="wizard-steps">
        <el-step title="选择数据库" />
        <el-step title="数据库连接" />
        <el-step title="备份策略" />
        <el-step title="存储和通知" />
        <el-step title="确认创建" />
      </el-steps>

      <div class="step-content">
        <!-- Step 1: 选择数据库类型 -->
        <div v-if="currentStep === 0" class="step step-1">
          <h3 class="step-title">选择数据库类型</h3>
          <p class="step-desc">请选择要备份的数据库类型</p>
          <div class="db-type-grid">
            <div
              v-for="db in dbTypes"
              :key="db.value"
              class="db-type-card"
              :class="{ active: wizardData.database_type === db.value }"
              @click="selectDbType(db.value)"
            >
              <div class="db-icon">
                <component :is="db.icon" />
              </div>
              <div class="db-name">{{ db.label }}</div>
              <div class="db-tag" v-if="wizardData.database_type === db.value">
                <el-icon><Check /></el-icon>
              </div>
            </div>
          </div>
        </div>

        <!-- Step 2: 数据库连接配置 -->
        <div v-if="currentStep === 1" class="step step-2">
          <h3 class="step-title">配置数据库连接</h3>
          <p class="step-desc">填写数据库连接信息</p>
          <el-form :model="wizardData" label-width="100px" class="connect-form">
            <el-form-item label="主机地址" required>
              <el-input v-model="wizardData.host" placeholder="localhost 或 IP 地址" />
            </el-form-item>
            <el-form-item label="端口" required>
              <el-input-number v-model="wizardData.port" :min="1" :max="65535" style="width: 100%" />
            </el-form-item>
            <el-form-item label="数据库名" required>
              <el-input v-model="wizardData.database" :placeholder="`请输入${getDbLabel()}数据库名`" />
            </el-form-item>
            <el-form-item label="用户名">
              <el-input v-model="wizardData.username" placeholder="请输入用户名" />
            </el-form-item>
            <el-form-item label="密码">
              <el-input v-model="wizardData.password" type="password" show-password placeholder="请输入密码" />
            </el-form-item>
            <el-form-item label="任务名称" required>
              <el-input v-model="wizardData.name" placeholder="给这个备份任务起个名字" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" plain @click="handleTestConnection" :loading="testing">
                <el-icon v-if="!testing"><Connection /></el-icon>
                测试连接
              </el-button>
              <span v-if="testResult !== null" class="test-result" :class="testResult ? 'success' : 'error'">
                {{ testResult ? '✓ 连接成功' : '✗ 连接失败' }}
              </span>
            </el-form-item>
          </el-form>
        </div>

        <!-- Step 3: 备份策略 -->
        <div v-if="currentStep === 2" class="step step-3">
          <h3 class="step-title">配置备份策略</h3>
          <p class="step-desc">选择备份类型和执行计划</p>
          <el-form :model="wizardData" label-width="100px" class="strategy-form">
            <el-form-item label="备份类型" required>
              <el-radio-group v-model="wizardData.backup_type">
                <el-radio label="full">全量备份</el-radio>
                <el-radio label="incremental">增量备份</el-radio>
              </el-radio-group>
              <div class="form-tip" v-if="wizardData.backup_type === 'incremental'">
                <el-icon><Warning /></el-icon>
                增量备份仅记录自上次备份以来的变化数据
              </div>
            </el-form-item>
            <el-form-item label="执行计划" required>
              <CronExpression v-model="wizardData.schedule" />
            </el-form-item>
            <el-form-item label="保留策略">
              <div class="retention-input">
                <el-input-number v-model="wizardData.retention_days" :min="1" :max="365" />
                <span class="retention-unit">天</span>
              </div>
              <div class="form-tip">备份文件保留天数，过期自动清理</div>
            </el-form-item>
          </el-form>
        </div>

        <!-- Step 4: 存储和通知 -->
        <div v-if="currentStep === 3" class="step step-4">
          <h3 class="step-title">配置存储和通知</h3>
          <p class="step-desc">选择备份文件存储位置和通知方式</p>
          <el-form :model="wizardData" label-width="100px" class="storage-form">
            <el-form-item label="存储类型" required>
              <el-radio-group v-model="wizardData.storage_type" class="storage-group">
                <el-radio label="local">
                  <span class="storage-label">
                    <el-icon><FolderOpened /></el-icon>
                    本地存储
                  </span>
                </el-radio>
                <el-radio label="s3">
                  <span class="storage-label">
                    S3 / MinIO
                  </span>
                </el-radio>
                <el-radio label="oss">
                  <span class="storage-label">
                    阿里云 OSS
                  </span>
                </el-radio>
                <el-radio label="cos">
                  <span class="storage-label">
                    腾讯云 COS
                  </span>
                </el-radio>
              </el-radio-group>
            </el-form-item>

            <!-- 高级选项 -->
            <el-divider content-position="left">高级选项</el-divider>
            <el-row :gutter="20">
              <el-col :span="8">
                <el-form-item label="压缩">
                  <el-switch v-model="wizardData.compress" />
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item label="加密">
                  <el-switch v-model="wizardData.encrypt" />
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item label="成功通知">
                  <el-switch v-model="wizardData.notify_on_success" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="20">
              <el-col :span="8">
                <el-form-item label="失败通知">
                  <el-switch v-model="wizardData.notify_on_failure" />
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item label="启用任务">
                  <el-switch v-model="wizardData.enabled" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row v-if="wizardData.encrypt">
              <el-col :span="12">
                <el-form-item label="加密密钥">
                  <el-input v-model="wizardData.encrypt_key" type="password" show-password placeholder="32位密钥" />
                </el-form-item>
              </el-col>
            </el-row>
          </el-form>
        </div>

        <!-- Step 5: 确认预览 -->
        <div v-if="currentStep === 4" class="step step-5">
          <h3 class="step-title">确认配置</h3>
          <p class="step-desc">请确认以下配置信息无误后点击创建</p>
          <div class="preview-card">
            <el-descriptions :column="2" border>
              <el-descriptions-item label="任务名称" :span="2">{{ wizardData.name }}</el-descriptions-item>
              <el-descriptions-item label="数据库类型">
                <el-tag>{{ getDbLabel() }}</el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="备份类型">
                <el-tag :type="wizardData.backup_type === 'incremental' ? 'warning' : 'success'">
                  {{ wizardData.backup_type === 'incremental' ? '增量' : '全量' }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="主机地址">{{ wizardData.host }}:{{ wizardData.port }}</el-descriptions-item>
              <el-descriptions-item label="数据库">{{ wizardData.database }}</el-descriptions-item>
              <el-descriptions-item label="执行计划">
                <code>{{ wizardData.schedule }}</code>
              </el-descriptions-item>
              <el-descriptions-item label="保留策略">{{ wizardData.retention_days }} 天</el-descriptions-item>
              <el-descriptions-item label="存储类型">
                <el-tag>{{ getStorageLabel() }}</el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="压缩">
                <el-tag :type="wizardData.compress ? 'success' : 'info'">
                  {{ wizardData.compress ? '启用' : '禁用' }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="加密">
                <el-tag :type="wizardData.encrypt ? 'warning' : 'info'">
                  {{ wizardData.encrypt ? '启用' : '禁用' }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="成功通知">
                <el-tag :type="wizardData.notify_on_success ? 'success' : 'info'">
                  {{ wizardData.notify_on_success ? '启用' : '禁用' }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="失败通知">
                <el-tag :type="wizardData.notify_on_failure ? 'danger' : 'info'">
                  {{ wizardData.notify_on_failure ? '启用' : '禁用' }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="任务状态">
                <el-tag :type="wizardData.enabled ? 'success' : 'info'">
                  {{ wizardData.enabled ? '启用' : '禁用' }}
                </el-tag>
              </el-descriptions-item>
            </el-descriptions>
          </div>
        </div>
      </div>

      <!-- 底部导航 -->
      <div class="step-footer">
        <el-button v-if="currentStep > 0" @click="prevStep">上一步</el-button>
        <el-button v-if="currentStep < 4" type="primary" @click="nextStep">下一步</el-button>
        <el-button v-if="currentStep === 4" type="primary" @click="handleSubmit" :loading="submitting">
          {{ isEdit ? '保存修改' : '创建任务' }}
        </el-button>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, h } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Check, Connection, Warning, FolderOpened } from '@element-plus/icons-vue'
import { jobAPI } from '@/api'
import CronExpression from '@/components/CronExpression.vue'

// Database type icons as simple SVG components
const MySQLIcon = () => h('svg', { viewBox: '0 0 24 24', fill: '#00758f', class: 'db-svg' }, [
  h('path', { d: 'M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 14.5c-2.49 0-4.5-2.01-4.5-4.5S9.51 7.5 12 7.5s4.5 2.01 4.5 4.5-2.01 4.5-4.5 4.5z' })
])

const PostgreSQLIcon = () => h('svg', { viewBox: '0 0 24 24', fill: '#336791', class: 'db-svg' }, [
  h('path', { d: 'M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 14.5c-2.49 0-4.5-2.01-4.5-4.5S9.51 7.5 12 7.5s4.5 2.01 4.5 4.5-2.01 4.5-4.5 4.5z' })
])

const SQLServerIcon = () => h('svg', { viewBox: '0 0 24 24', fill: '#CC2927', class: 'db-svg' }, [
  h('path', { d: 'M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 14.5c-2.49 0-4.5-2.01-4.5-4.5S9.51 7.5 12 7.5s4.5 2.01 4.5 4.5-2.01 4.5-4.5 4.5z' })
])

const MongoDBIcon = () => h('svg', { viewBox: '0 0 24 24', fill: '#47A248', class: 'db-svg' }, [
  h('path', { d: 'M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 14.5c-2.49 0-4.5-2.01-4.5-4.5S9.51 7.5 12 7.5s4.5 2.01 4.5 4.5-2.01 4.5-4.5 4.5z' })
])

const router = useRouter()
const route = useRoute()

const currentStep = ref(0)
const submitting = ref(false)
const testing = ref(false)
const testResult = ref<boolean | null>(null)
const isEdit = ref(false)

// Database type definitions
const dbTypes = [
  { value: 'mysql', label: 'MySQL', icon: MySQLIcon, defaultPort: 3306 },
  { value: 'postgres', label: 'PostgreSQL', icon: PostgreSQLIcon, defaultPort: 5432 },
  { value: 'sqlserver', label: 'SQL Server', icon: SQLServerIcon, defaultPort: 1433 },
  { value: 'mongodb', label: 'MongoDB', icon: MongoDBIcon, defaultPort: 27017 }
]

const dbPortMap: Record<string, number> = {
  mysql: 3306,
  postgres: 5432,
  sqlserver: 1433,
  mongodb: 27017
}

const wizardData = reactive({
  name: '',
  database_type: '',
  host: 'localhost',
  port: 5432,
  database: '',
  username: '',
  password: '',
  schedule: '0 0 * * *',
  storage_type: 'local',
  retention_days: 7,
  backup_type: 'full',
  compress: true,
  encrypt: false,
  encrypt_key: '',
  notify_on_success: false,
  notify_on_failure: true,
  enabled: true
})

const getDbLabel = () => {
  const db = dbTypes.find(d => d.value === wizardData.database_type)
  return db?.label || wizardData.database_type
}

const getStorageLabel = () => {
  const map: Record<string, string> = {
    local: '本地存储',
    s3: 'S3 / MinIO',
    oss: '阿里云 OSS',
    cos: '腾讯云 COS'
  }
  return map[wizardData.storage_type] || wizardData.storage_type
}

const selectDbType = (type: string) => {
  wizardData.database_type = type
  wizardData.port = dbPortMap[type] || 5432
  testResult.value = null
}

const handleTestConnection = async () => {
  if (!wizardData.host || !wizardData.port || !wizardData.database) {
    ElMessage.warning('请先填写主机、端口和数据库名')
    return
  }
  testing.value = true
  testResult.value = null
  try {
    await jobAPI.testConnection({
      database_type: wizardData.database_type,
      host: wizardData.host,
      port: wizardData.port,
      database: wizardData.database,
      username: wizardData.username,
      password: wizardData.password
    })
    testResult.value = true
    ElMessage.success('连接测试成功')
  } catch (e: any) {
    testResult.value = false
    ElMessage.error(e.message || '连接测试失败')
  } finally {
    testing.value = false
  }
}

const validateCurrentStep = (): boolean => {
  switch (currentStep.value) {
    case 0:
      if (!wizardData.database_type) {
        ElMessage.warning('请选择数据库类型')
        return false
      }
      return true
    case 1:
      if (!wizardData.host) {
        ElMessage.warning('请输入主机地址')
        return false
      }
      if (!wizardData.database) {
        ElMessage.warning('请输入数据库名')
        return false
      }
      if (!wizardData.name) {
        ElMessage.warning('请输入任务名称')
        return false
      }
      return true
    case 2:
      if (!wizardData.schedule) {
        ElMessage.warning('请选择执行计划')
        return false
      }
      return true
    case 3:
      if (!wizardData.storage_type) {
        ElMessage.warning('请选择存储类型')
        return false
      }
      return true
    default:
      return true
  }
}

const nextStep = () => {
  if (validateCurrentStep()) {
    currentStep.value++
  }
}

const prevStep = () => {
  if (currentStep.value > 0) {
    currentStep.value--
  }
}

const handleSubmit = async () => {
  submitting.value = true
  try {
    const submitData = { ...wizardData }
    if (!submitData.password) {
      delete submitData.password
    }
    await jobAPI.create(submitData)
    ElMessage.success('任务创建成功')
    router.push('/jobs')
  } catch (e: any) {
    ElMessage.error(e.message || '创建失败')
  } finally {
    submitting.value = false
  }
}

const handleBack = () => {
  router.push('/jobs')
}

onMounted(async () => {
  // Check if editing existing job
  const editId = route.query.editId
  if (editId) {
    isEdit.value = true
    try {
      const res = await jobAPI.get(Number(editId))
      const job = res.data
      Object.assign(wizardData, {
        ...job,
        password: '',
        encrypt_key: job.encrypt_key || ''
      })
    } catch {
      ElMessage.error('加载任务信息失败')
    }
  }
})
</script>

<style scoped>
.job-wizard {
  max-width: 900px;
  margin: 0 auto;
}

.wizard-header {
  display: flex;
  align-items: center;
}

.header-title {
  font-size: 16px;
  font-weight: 600;
}

.wizard-steps {
  margin: 24px 0 32px;
}

.step-content {
  min-height: 380px;
}

.step-title {
  font-size: 18px;
  font-weight: 600;
  margin: 0 0 8px;
  color: #303133;
}

.step-desc {
  font-size: 14px;
  color: #909399;
  margin: 0 0 24px;
}

/* Step 1: DB Type Cards */
.db-type-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.db-type-card {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 24px 16px;
  border: 2px solid #dcdfe6;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s;
  background: #fff;
}

.db-type-card:hover {
  border-color: #409eff;
  box-shadow: 0 2px 12px rgba(64, 158, 255, 0.15);
}

.db-type-card.active {
  border-color: #409eff;
  background: #ecf5ff;
}

.db-icon {
  font-size: 48px;
  margin-bottom: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

:deep(.db-svg) {
  width: 48px;
  height: 48px;
}

.db-name {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
}

.db-tag {
  position: absolute;
  top: -8px;
  right: -8px;
  width: 24px;
  height: 24px;
  background: #409eff;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 12px;
}

/* Step 2: Connection Form */
.connect-form {
  max-width: 500px;
}

.test-result {
  margin-left: 12px;
  font-size: 14px;
}

.test-result.success {
  color: #67c23a;
}

.test-result.error {
  color: #f56c6c;
}

/* Step 3: Strategy Form */
.strategy-form {
  max-width: 600px;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 6px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.retention-input {
  display: flex;
  align-items: center;
  gap: 8px;
}

.retention-unit {
  color: #606266;
  font-size: 14px;
}

/* Step 4: Storage Form */
.storage-group {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.storage-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
}

/* Step 5: Preview */
.preview-card {
  max-width: 700px;
}

.step-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 32px;
  padding-top: 20px;
  border-top: 1px solid #f0f0f0;
}

@media (max-width: 768px) {
  .db-type-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
