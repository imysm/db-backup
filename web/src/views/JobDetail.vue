<template>
  <div class="job-detail-container">
    <el-card v-loading="loading">
      <template #header>
        <div class="card-header">
          <el-button text @click="$router.push('/jobs')">
            <el-icon><Back /></el-icon>返回
          </el-button>
          <h3>{{ job.name || '任务详情' }}</h3>
          <div class="actions">
            <el-button type="primary" size="small" @click="handleRun" :loading="running">
              立即执行
            </el-button>
            <el-button size="small" @click="handleEdit">编辑</el-button>
            <el-button size="small" :type="job.enabled ? 'warning' : 'success'" @click="handleToggle">
              {{ job.enabled ? '禁用' : '启用' }}
            </el-button>
            <el-button type="danger" size="small" @click="handleDelete">删除</el-button>
          </div>
        </div>
      </template>

      <el-descriptions :column="2" border v-if="job.id">
        <el-descriptions-item label="任务名称" :span="2">{{ job.name }}</el-descriptions-item>
        <el-descriptions-item label="数据库类型">
          <el-tag size="small">{{ dbTypeLabel(job.database_type) }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="备份类型">
          <el-tag size="small" :type="job.backup_type === 'incremental' ? 'warning' : 'success'">
            {{ job.backup_type === 'incremental' ? '增量' : '全量' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="job.enabled ? 'success' : 'info'" size="small">
            {{ job.enabled ? '启用' : '禁用' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="执行周期">{{ job.schedule || '-' }}</el-descriptions-item>
        <el-descriptions-item label="主机地址">{{ job.host }}:{{ job.port }}</el-descriptions-item>
        <el-descriptions-item label="数据库名">{{ job.database }}</el-descriptions-item>
        <el-descriptions-item label="存储类型">{{ storageLabel(job.storage_type) }}</el-descriptions-item>
        <el-descriptions-item label="保留策略">{{ job.retention_days }} 天</el-descriptions-item>
        <el-descriptions-item label="压缩">
          <el-tag :type="job.compress ? 'success' : 'info'" size="small">
            {{ job.compress ? '启用' : '禁用' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="加密">
          <el-tag :type="job.encrypt ? 'warning' : 'info'" size="small">
            {{ job.encrypt ? '启用' : '禁用' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="创建时间">{{ job.created_at }}</el-descriptions-item>
        <el-descriptions-item label="最后执行">{{ job.updated_at }}</el-descriptions-item>
      </el-descriptions>

      <el-empty v-else description="任务不存在" />
    </el-card>

    <!-- 执行历史 -->
    <el-card style="margin-top: 16px" v-if="job.id">
      <template #header>
        <span>执行历史</span>
      </template>
      <el-table :data="records" v-loading="recordsLoading" style="width: 100%">
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="statusTag(row.status)" size="small">
              {{ statusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="file_size" label="文件大小" width="120">
          <template #default="{ row }">{{ formatSize(row.file_size) }}</template>
        </el-table-column>
        <el-table-column prop="duration" label="耗时(秒)" width="100" />
        <el-table-column prop="started_at" label="开始时间" width="180" />
        <el-table-column prop="finished_at" label="结束时间" width="180" />
        <el-table-column label="操作" width="120">
          <template #default="{ row }">
            <el-button size="small" type="primary" link @click="handleViewLog(row)">日志</el-button>
            <el-button size="small" type="success" link @click="handleRestore(row)" :disabled="row.status !== 'success'">恢复</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
        v-model:current-page="page"
        :page-size="20"
        :total="total"
        @current-change="fetchRecords"
        layout="total, prev, pager, next"
        style="margin-top: 16px; justify-content: flex-end"
      />
    </el-card>

    <!-- 日志对话框 -->
    <el-dialog v-model="logVisible" title="执行日志" width="800px">
      <el-input v-model="logContent" type="textarea" :rows="20" readonly />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Back } from '@element-plus/icons-vue'
import { jobAPI, recordAPI } from '@/api'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const running = ref(false)
const job = ref<any>({})
const records = ref<any[]>([])
const recordsLoading = ref(false)
const total = ref(0)
const page = ref(1)

const logVisible = ref(false)
const logContent = ref('')

const dbTypeLabel = (type: string) => {
  const map: Record<string, string> = {
    mysql: 'MySQL',
    postgres: 'PostgreSQL',
    mongodb: 'MongoDB',
    sqlserver: 'SQL Server',
    oracle: 'Oracle'
  }
  return map[type] || type
}

const storageLabel = (type: string) => {
  const map: Record<string, string> = {
    local: '本地存储',
    s3: 'S3',
    oss: '阿里云OSS',
    cos: '腾讯云COS'
  }
  return map[type] || type
}

const statusTag = (s: string) => {
  const map: Record<string, string> = {
    success: 'success',
    failed: 'danger',
    running: 'warning',
    pending: 'info'
  }
  return map[s] || 'info'
}

const statusLabel = (s: string) => {
  const map: Record<string, string> = {
    success: '成功',
    failed: '失败',
    running: '运行中',
    pending: '等待中'
  }
  return map[s] || s
}

const formatSize = (bytes: number) => {
  if (!bytes) return '-'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const fetchJob = async () => {
  const id = route.params.id
  if (!id) {
    ElMessage.error('无效的任务ID')
    return
  }
  loading.value = true
  try {
    const res = await jobAPI.get(id)
    job.value = res.data || {}
  } catch (e: any) {
    ElMessage.error(e.message || '获取任务详情失败')
  } finally {
    loading.value = false
  }
}

const fetchRecords = async () => {
  const id = route.params.id
  if (!id) return
  recordsLoading.value = true
  try {
    const res = await recordAPI.list({ job_id: id, page: page.value, page_size: 20 })
    records.value = res.data?.data || []
    total.value = res.data?.total || 0
  } catch (e: any) {
    ElMessage.error(e.message || '获取执行记录失败')
  } finally {
    recordsLoading.value = false
  }
}

const handleRun = async () => {
  try {
    await ElMessageBox.confirm('确定要立即执行此任务吗？', '确认', { type: 'info' })
    running.value = true
    await jobAPI.run(job.value.id)
    ElMessage.success('任务已触发执行')
    fetchRecords()
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e.message || '执行失败')
    }
  } finally {
    running.value = false
  }
}

const handleEdit = () => {
  router.push({ path: '/jobs', query: { editId: job.value.id } })
}

const handleToggle = async () => {
  try {
    await jobAPI.update(job.value.id, { enabled: !job.enabled })
    ElMessage.success(job.enabled ? '已禁用' : '已启用')
    fetchJob()
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  }
}

const handleDelete = async () => {
  try {
    await ElMessageBox.confirm('确定要删除此任务吗？此操作不可恢复！', '危险操作', { type: 'warning' })
    await jobAPI.delete(job.value.id)
    ElMessage.success('删除成功')
    router.push('/jobs')
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e.message || '删除失败')
    }
  }
}

const handleViewLog = async (row: any) => {
  logContent.value = row.log || row.error_message || '无日志'
  logVisible.value = true
}

const handleRestore = (row: any) => {
  router.push({ path: '/restore', query: { recordId: row.id } })
}

onMounted(() => {
  fetchJob()
  fetchRecords()
})
</script>

<style scoped>
.job-detail-container {
  padding: 0;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 16px;
}

.card-header h3 {
  flex: 1;
  margin: 0;
}

.actions {
  display: flex;
  gap: 8px;
}
</style>
