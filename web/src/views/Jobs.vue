<template>
  <div class="jobs-container">
    <!-- 搜索栏 -->
    <el-card shadow="never" style="margin-bottom: 16px">
      <el-row :gutter="16" align="middle">
        <el-col :span="8">
          <el-input v-model="searchText" placeholder="搜索任务名称" clearable @clear="fetchJobs" @keyup.enter="fetchJobs">
            <template #prefix><el-icon><Search /></el-icon></template>
          </el-input>
        </el-col>
        <el-col :span="5">
          <el-select v-model="filterDbType" placeholder="数据库类型" clearable @change="fetchJobs" style="width: 100%">
            <el-option label="MySQL" value="mysql" />
            <el-option label="PostgreSQL" value="postgres" />
            <el-option label="MongoDB" value="mongodb" />
            <el-option label="SQL Server" value="sqlserver" />
            <el-option label="Oracle" value="oracle" />
          </el-select>
        </el-col>
        <el-col :span="5">
          <el-select v-model="filterEnabled" placeholder="启用状态" clearable @change="fetchJobs" style="width: 100%">
            <el-option label="已启用" value="true" />
            <el-option label="已禁用" value="false" />
          </el-select>
        </el-col>
        <el-col :span="6" style="text-align: right">
          <el-button type="primary" @click="handleCreate">
            <el-icon><Plus /></el-icon>新建任务
          </el-button>
        </el-col>
      </el-row>
    </el-card>

    <!-- 任务列表 -->
    <el-card shadow="never">
      <el-table
        ref="tableRef"
        :data="jobs"
        v-loading="loading"
        style="width: 100%"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="40" />
        <el-table-column prop="name" label="任务名称" min-width="140" show-overflow-tooltip />
        <el-table-column prop="database_type" label="数据库类型" width="110">
          <template #default="{ row }">
            <el-tag size="small" :type="dbTypeTag(row.database_type)">{{ dbTypeLabel(row.database_type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="主机:端口" width="180">
          <template #default="{ row }">{{ row.host }}:{{ row.port }}</template>
        </el-table-column>
        <el-table-column prop="schedule" label="Cron 表达式" width="130" />
        <el-table-column prop="storage_type" label="存储类型" width="90">
          <template #default="{ row }">{{ storageLabel(row.storage_type) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="80" align="center">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'" size="small">
              {{ row.enabled ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="最后执行" width="170">
          <template #default="{ row }">
            {{ row.last_run ? formatTime(row.last_run) : '-' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" link @click="handleViewDetail(row)">详情</el-button>
            <el-button size="small" type="primary" link @click="handleEdit(row)">编辑</el-button>
            <el-button size="small" type="success" link @click="handleRun(row)">执行</el-button>
            <el-dropdown @command="(cmd: string) => handleMoreCommand(cmd, row)" trigger="click">
              <el-button size="small" type="info" link>更多</el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="copy">复制任务</el-dropdown-item>
                  <el-dropdown-item :command="row.enabled ? 'disable' : 'enable'">
                    {{ row.enabled ? '禁用' : '启用' }}
                  </el-dropdown-item>
                  <el-dropdown-item command="delete" divided>
                    <span style="color: var(--el-color-danger)">删除</span>
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>

      <!-- 批量操作工具栏 -->
      <div v-if="selectedRows.length > 0" class="batch-bar">
        <span>已选 {{ selectedRows.length }} 项</span>
        <el-button size="small" type="success" @click="handleBatchToggle(true)">批量启用</el-button>
        <el-button size="small" type="warning" @click="handleBatchToggle(false)">批量禁用</el-button>
        <el-button size="small" type="danger" @click="handleBatchDelete">批量删除</el-button>
      </div>

      <!-- 分页 -->
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        :page-sizes="[10, 20, 50]"
        @current-change="fetchJobs"
        @size-change="fetchJobs"
        layout="total, sizes, prev, pager, next, jumper"
        style="margin-top: 16px; justify-content: flex-end"
      />
    </el-card>

    <!-- 创建/编辑抽屉 -->
    <el-drawer
      v-model="drawerVisible"
      :title="isEdit ? '编辑任务' : '新建任务'"
      size="520px"
      :close-on-click-modal="false"
    >
      <el-form ref="formRef" :model="form" :rules="formRules" label-width="100px" label-position="top">
        <!-- 基本信息 -->
        <el-divider content-position="left">基本信息</el-divider>
        <el-form-item label="任务名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入任务名称" />
        </el-form-item>
        <el-form-item label="数据库类型" prop="database_type">
          <el-select v-model="form.database_type" placeholder="请选择" style="width: 100%" @change="onDbTypeChange">
            <el-option label="MySQL" value="mysql" />
            <el-option label="PostgreSQL" value="postgres" />
            <el-option label="MongoDB" value="mongodb" />
            <el-option label="SQL Server" value="sqlserver" />
            <el-option label="Oracle" value="oracle" />
          </el-select>
        </el-form-item>

        <!-- 数据库连接 -->
        <el-divider content-position="left">数据库连接</el-divider>
        <el-row :gutter="16">
          <el-col :span="16">
            <el-form-item label="主机" prop="host">
              <el-input v-model="form.host" placeholder="localhost" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="端口" prop="port">
              <el-input-number v-model="form.port" :min="1" :max="65535" controls-position="right" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="用户名">
          <el-input v-model="form.username" placeholder="请输入用户名" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="form.password" type="password" show-password placeholder="请输入密码" />
        </el-form-item>
        <el-form-item label="数据库名">
          <el-input v-model="form.database" placeholder="请输入数据库名" />
        </el-form-item>
        <el-form-item>
          <el-button :loading="testingConn" @click="handleTestConnection">
            测试连接
          </el-button>
        </el-form-item>

        <!-- 备份策略 -->
        <el-divider content-position="left">备份策略</el-divider>
        <el-form-item label="Cron 表达式" prop="schedule">
          <el-input v-model="form.schedule" placeholder="0 2 * * *" @input="debouncedPreviewRuns">
            <template #append>
              <el-tooltip content="格式: 分 时 日 月 周" placement="top">
                <el-button>?</el-button>
              </el-tooltip>
            </template>
          </el-input>
        </el-form-item>
        <div v-if="nextRuns.length > 0" class="next-runs-preview">
          <span class="next-runs-label">下次执行：</span>
          <el-tag v-for="(r, i) in nextRuns" :key="i" size="small" style="margin: 2px">{{ r }}</el-tag>
        </div>

        <!-- 存储配置 -->
        <el-divider content-position="left">存储配置</el-divider>
        <el-form-item label="存储类型" prop="storage_type">
          <el-select v-model="form.storage_type" placeholder="请选择" style="width: 100%">
            <el-option label="本地存储" value="local" />
            <el-option label="S3 / MinIO" value="s3" />
            <el-option label="阿里云 OSS" value="oss" />
            <el-option label="腾讯云 COS" value="cos" />
          </el-select>
        </el-form-item>

        <!-- 本地存储配置 -->
        <template v-if="form.storage_type === 'local'">
          <el-form-item label="本地路径">
            <el-input v-model="storageConfig.base_path" placeholder="/data/backups" />
          </el-form-item>
        </template>

        <!-- S3 配置 -->
        <template v-if="form.storage_type === 's3'">
          <el-form-item label="Endpoint">
            <el-input v-model="storageConfig.endpoint" placeholder="https://s3.amazonaws.com" />
          </el-form-item>
          <el-row :gutter="16">
            <el-col :span="12">
              <el-form-item label="Region">
                <el-input v-model="storageConfig.region" placeholder="us-east-1" />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="Bucket">
                <el-input v-model="storageConfig.bucket" placeholder="my-bucket" />
              </el-form-item>
            </el-col>
          </el-row>
          <el-form-item label="Access Key">
            <el-input v-model="storageConfig.access_key" />
          </el-form-item>
          <el-form-item label="Secret Key">
            <el-input v-model="storageConfig.secret_key" type="password" show-password />
          </el-form-item>
        </template>

        <!-- OSS 配置 -->
        <template v-if="form.storage_type === 'oss'">
          <el-form-item label="Endpoint">
            <el-input v-model="storageConfig.oss_endpoint" placeholder="https://oss-cn-hangzhou.aliyuncs.com" />
          </el-form-item>
          <el-form-item label="Bucket">
            <el-input v-model="storageConfig.oss_bucket" placeholder="my-bucket" />
          </el-form-item>
        </template>

        <!-- COS 配置 -->
        <template v-if="form.storage_type === 'cos'">
          <el-form-item label="Endpoint">
            <el-input v-model="storageConfig.cos_endpoint" placeholder="https://cos.ap-beijing.myqcloud.com" />
          </el-form-item>
          <el-row :gutter="16">
            <el-col :span="12">
              <el-form-item label="Bucket">
                <el-input v-model="storageConfig.cos_bucket" placeholder="my-bucket" />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="Region">
                <el-input v-model="storageConfig.cos_region" placeholder="ap-beijing" />
              </el-form-item>
            </el-col>
          </el-row>
        </template>

        <!-- 高级设置 -->
        <el-collapse>
          <el-collapse-item title="高级设置" name="advanced">
            <el-row :gutter="16">
              <el-col :span="8">
                <el-form-item label="压缩">
                  <el-switch v-model="form.compress" />
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item label="加密">
                  <el-switch v-model="form.encrypt" />
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item label="保留天数">
                  <el-input-number v-model="form.retention_days" :min="1" :max="365" controls-position="right" style="width: 100%" />
                </el-form-item>
              </el-col>
            </el-row>
          </el-collapse-item>

          <!-- 通知设置 -->
          <el-collapse-item title="通知设置" name="notify">
            <el-form-item label="成功通知">
              <el-switch v-model="form.notify_on_success" />
            </el-form-item>
            <el-form-item label="失败通知">
              <el-switch v-model="form.notify_on_fail" />
            </el-form-item>
          </el-collapse-item>
        </el-collapse>
      </el-form>

      <template #footer>
        <el-button @click="drawerVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">确定</el-button>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Search, Plus } from '@element-plus/icons-vue'
import { jobAPI } from '@/api'

const router = useRouter()

// --- 列表状态 ---
const loading = ref(false)
const jobs = ref<any[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const searchText = ref('')
const filterDbType = ref('')
const filterEnabled = ref('')
const selectedRows = ref<any[]>([])
const tableRef = ref()

// --- 抽屉状态 ---
const drawerVisible = ref(false)
const isEdit = ref(false)
const submitting = ref(false)
const formRef = ref<FormInstance>()
const testingConn = ref(false)
const nextRuns = ref<string[]>([])
const storageConfig = reactive<Record<string, string>>({})

const defaultForm = () => ({
  name: '',
  database_type: 'postgres',
  host: 'localhost',
  port: 5432,
  username: '',
  password: '',
  database: '',
  schedule: '0 2 * * *',
  storage_type: 'local',
  retention_days: 7,
  compress: true,
  encrypt: false,
  notify_on_success: false,
  notify_on_fail: true,
  enabled: true,
})

const form = ref<any>(defaultForm())

const formRules: FormRules = {
  name: [{ required: true, message: '请输入任务名称', trigger: 'blur' }],
  database_type: [{ required: true, message: '请选择数据库类型', trigger: 'change' }],
  host: [{ required: true, message: '请输入主机地址', trigger: 'blur' }],
  port: [{ required: true, message: '请输入端口', trigger: 'blur' }],
  schedule: [{ required: true, message: '请输入 Cron 表达式', trigger: 'blur' }],
  storage_type: [{ required: true, message: '请选择存储类型', trigger: 'change' }],
}

// --- 默认端口映射 ---
const dbPortMap: Record<string, number> = {
  mysql: 3306, postgres: 5432, mongodb: 27017, sqlserver: 1433, oracle: 1521,
}

// --- 辅助函数 ---
const dbTypeLabel = (t: string) => ({ mysql: 'MySQL', postgres: 'PostgreSQL', mongodb: 'MongoDB', sqlserver: 'SQL Server', oracle: 'Oracle' }[t] || t)
const dbTypeTag = (t: string) => ({ mysql: '', postgres: 'success', mongodb: 'warning', sqlserver: 'info', oracle: 'danger' }[t] || 'info') as any
const storageLabel = (t: string) => ({ local: '本地', s3: 'S3', oss: 'OSS', cos: 'COS' }[t] || t)

const formatTime = (t: string) => {
  const d = new Date(t)
  return d.toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

// --- 数据获取 ---
const fetchJobs = async () => {
  loading.value = true
  try {
    const params: any = { page: page.value, page_size: pageSize.value }
    if (searchText.value) params.search = searchText.value
    if (filterDbType.value) params.database_type = filterDbType.value
    if (filterEnabled.value) params.enabled = filterEnabled.value
    const res: any = await jobAPI.list(params)
    jobs.value = res.data?.data || []
    total.value = res.data?.total || 0
  } catch (e: any) {
    ElMessage.error(e.message || '获取任务列表失败')
  } finally {
    loading.value = false
  }
}

// --- 表格选择 ---
const handleSelectionChange = (rows: any[]) => { selectedRows.value = rows }

// --- 数据库类型变更时自动设置端口 ---
const onDbTypeChange = (val: string) => {
  form.value.port = dbPortMap[val] || 3306
}

// --- 测试连接 ---
const handleTestConnection = async () => {
  testingConn.value = true
  try {
    // 先临时创建/更新以获取 ID，或者直接用 host:port 测试
    // 对于新建场景，直接用 net.Dial 思路，调用 API 不现实（没有 ID）
    // 改为前端简单提示：保存后可测试，或使用现有任务的 ID
    ElMessage.info('请先保存任务后再测试连接')
  } finally {
    testingConn.value = false
  }
}

// --- Cron 预览 ---
let previewTimer: ReturnType<typeof setTimeout> | null = null
const debouncedPreviewRuns = () => {
  if (previewTimer) clearTimeout(previewTimer)
  previewTimer = setTimeout(() => previewRuns(), 500)
}

const previewRuns = async () => {
  if (!form.value.schedule) { nextRuns.value = []; return }
  try {
    const res: any = await jobAPI.nextRuns(0, form.value.schedule)
    nextRuns.value = res.data?.next_runs || []
  } catch {
    nextRuns.value = []
  }
}

// --- 创建/编辑 ---
const handleCreate = () => {
  router.push('/jobs/new')
}

const handleViewDetail = (row: any) => {
  router.push(`/jobs/${row.id}`)
}

const handleEdit = (row: any) => {
  isEdit.value = true
  form.value = { ...row }
  // 解析 storage_config
  if (row.storage_config) {
    try {
      const cfg = typeof row.storage_config === 'string' ? JSON.parse(row.storage_config) : row.storage_config
      Object.keys(storageConfig).forEach(k => delete storageConfig[k])
      Object.assign(storageConfig, cfg)
    } catch { /* ignore */ }
  } else {
    Object.keys(storageConfig).forEach(k => delete storageConfig[k])
  }
  nextRuns.value = []
  if (form.value.schedule) previewRuns()
  drawerVisible.value = true
}

const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate()

  submitting.value = true
  try {
    const payload = { ...form.value }
    // 组装存储配置
    if (form.value.storage_type !== 'local') {
      payload.storage_config = JSON.stringify({ ...storageConfig })
    } else {
      payload.storage_config = JSON.stringify({ base_path: storageConfig.base_path || '/data/backups' })
    }

    if (isEdit.value) {
      await jobAPI.update(form.value.id, payload)
      ElMessage.success('更新成功')
    } else {
      await jobAPI.create(payload)
      ElMessage.success('创建成功')
    }
    drawerVisible.value = false
    fetchJobs()
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

// --- 执行 ---
const handleRun = async (row: any) => {
  try {
    await ElMessageBox.confirm(`确定要立即执行任务「${row.name}」吗？`, '提示', { type: 'info' })
    await jobAPI.run(row.id)
    ElMessage.success('任务已触发执行')
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e.message || '执行失败')
  }
}

// --- 更多操作 ---
const handleMoreCommand = async (cmd: string, row: any) => {
  if (cmd === 'copy') {
    const copy = { ...row, name: row.name + ' - 副本', id: undefined, created_at: undefined, updated_at: undefined }
    delete copy.id
    try {
      await jobAPI.create(copy)
      ElMessage.success('复制成功')
      fetchJobs()
    } catch (e: any) { ElMessage.error(e.message || '复制失败') }
  } else if (cmd === 'enable' || cmd === 'disable') {
    const enabled = cmd === 'enable'
    try {
      await jobAPI.update(row.id, { enabled })
      ElMessage.success(enabled ? '已启用' : '已禁用')
      fetchJobs()
    } catch (e: any) { ElMessage.error(e.message || '操作失败') }
  } else if (cmd === 'delete') {
    try {
      await ElMessageBox.confirm(`确定要删除任务「${row.name}」吗？`, '警告', { type: 'warning' })
      await jobAPI.delete(row.id)
      ElMessage.success('删除成功')
      fetchJobs()
    } catch (e: any) {
      if (e !== 'cancel') ElMessage.error(e.message || '删除失败')
    }
  }
}

// --- 批量操作 ---
const handleBatchToggle = async (enabled: boolean) => {
  try {
    await ElMessageBox.confirm(`确定要批量${enabled ? '启用' : '禁用'} ${selectedRows.value.length} 个任务吗？`, '提示', { type: 'info' })
    for (const row of selectedRows.value) {
      await jobAPI.update(row.id, { enabled })
    }
    ElMessage.success('操作成功')
    fetchJobs()
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e.message || '操作失败')
  }
}

const handleBatchDelete = async () => {
  try {
    await ElMessageBox.confirm(`确定要批量删除 ${selectedRows.value.length} 个任务吗？此操作不可恢复！`, '警告', { type: 'warning' })
    for (const row of selectedRows.value) {
      await jobAPI.delete(row.id)
    }
    ElMessage.success('删除成功')
    fetchJobs()
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e.message || '删除失败')
  }
}

onMounted(() => { fetchJobs() })
</script>

<style scoped>
.jobs-container {
  padding: 0;
}
.batch-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  padding: 8px 12px;
  background: var(--el-fill-color-light);
  border-radius: 4px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.next-runs-preview {
  margin: -12px 0 12px 100px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.next-runs-label {
  margin-right: 4px;
}
</style>
