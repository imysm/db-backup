<template>
  <div class="records-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>备份记录</span>
        </div>
      </template>

      <!-- 筛选栏 -->
      <div class="filter-bar">
        <el-select v-model="filters.job_id" placeholder="全部任务" clearable style="width: 180px">
          <el-option v-for="job in jobs" :key="job.id" :label="job.name" :value="String(job.id)" />
        </el-select>
        <el-select v-model="filters.status" placeholder="全部状态" clearable style="width: 120px">
          <el-option label="成功" value="success" />
          <el-option label="失败" value="failed" />
          <el-option label="运行中" value="running" />
          <el-option label="等待中" value="pending" />
        </el-select>
        <el-date-picker
          v-model="filters.dateRange"
          type="daterange"
          range-separator="至"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          value-format="YYYY-MM-DD"
          style="width: 260px"
        />
        <el-button type="primary" @click="handleSearch">搜索</el-button>
        <el-button @click="handleReset">重置</el-button>
      </div>

      <!-- 表格 -->
      <el-table :data="records" v-loading="loading" style="width: 100%">
        <el-table-column prop="job_name" label="任务名称" min-width="140" />
        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)" size="small">
              {{ getStatusText(row.status) }}
            </el-tag>
            <el-tooltip v-if="row.verified" content="已验证" placement="top">
              <el-icon style="margin-left: 4px; color: #67c23a"><CircleCheckFilled /></el-icon>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="文件大小" width="110">
          <template #default="{ row }">
            {{ formatSize(row.file_size) }}
          </template>
        </el-table-column>
        <el-table-column label="耗时" width="90">
          <template #default="{ row }">
            {{ formatDuration(row.duration) }}
          </template>
        </el-table-column>
        <el-table-column label="开始时间" width="170">
          <template #default="{ row }">
            {{ formatTime(row.started_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button size="small" link type="primary" @click="showDetail(row)">详情</el-button>
            <el-button
              v-if="row.status === 'success'"
              size="small" link type="success"
              @click="handleDownload(row)"
            >下载</el-button>
            <el-button
              v-if="row.status === 'success' && !row.verified"
              size="small" link type="warning"
              @click="handleVerify(row)"
            >验证</el-button>
            <el-button size="small" link type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next"
          @current-change="fetchRecords"
          @size-change="fetchRecords"
        />
      </div>
    </el-card>

    <!-- 详情抽屉 -->
    <el-drawer v-model="drawerVisible" title="备份记录详情" size="450px">
      <template v-if="currentRecord">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="任务名称">{{ currentRecord.job_name }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="getStatusType(currentRecord.status)" size="small">{{ getStatusText(currentRecord.status) }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="开始时间">{{ formatTime(currentRecord.started_at) }}</el-descriptions-item>
          <el-descriptions-item label="结束时间">{{ formatTime(currentRecord.finished_at) }}</el-descriptions-item>
          <el-descriptions-item label="耗时">{{ formatDuration(currentRecord.duration) }}</el-descriptions-item>
          <el-descriptions-item label="文件路径">{{ currentRecord.file_path || '-' }}</el-descriptions-item>
          <el-descriptions-item label="文件大小">{{ formatSize(currentRecord.file_size) }}</el-descriptions-item>
          <el-descriptions-item label="SHA256">
            <el-text truncated style="max-width: 280px; font-family: monospace; font-size: 12px">
              {{ currentRecord.checksum || '-' }}
            </el-text>
          </el-descriptions-item>
          <el-descriptions-item label="验证状态">
            <el-tag v-if="currentRecord.verified" type="success" size="small">已验证</el-tag>
            <el-tag v-else type="info" size="small">未验证</el-tag>
          </el-descriptions-item>
        </el-descriptions>

        <!-- 错误信息 -->
        <div v-if="currentRecord.status === 'failed' && currentRecord.error_message" class="error-box">
          <el-text type="danger">{{ currentRecord.error_message }}</el-text>
        </div>

        <!-- 操作按钮 -->
        <div class="drawer-actions">
          <el-button
            v-if="currentRecord.status === 'success'"
            type="success"
            @click="handleDownload(currentRecord)"
          >下载备份文件</el-button>
          <el-button
            v-if="currentRecord.status === 'success'"
            type="warning"
            @click="handleRestore(currentRecord)"
          >恢复此备份</el-button>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CircleCheckFilled } from '@element-plus/icons-vue'
import { recordAPI, verifyAPI, jobAPI } from '@/api'

const router = useRouter()
const loading = ref(false)
const records = ref<any[]>([])
const jobs = ref<any[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const drawerVisible = ref(false)
const currentRecord = ref<any>(null)

const filters = reactive({
  job_id: '',
  status: '',
  dateRange: null as string[] | null
})

const getStatusType = (status: string) => {
  const map: any = { success: 'success', running: 'warning', failed: 'danger', pending: 'info' }
  return map[status] || 'info'
}

const getStatusText = (status: string) => {
  const map: any = { success: '成功', running: '运行中', failed: '失败', pending: '等待中' }
  return map[status] || status
}

const formatSize = (bytes: number) => {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const formatDuration = (seconds: number) => {
  if (!seconds) return '-'
  const m = Math.floor(seconds / 60)
  const s = Math.floor(seconds % 60)
  if (m > 0) return `${m}m ${s}s`
  return `${s}s`
}

const formatTime = (t: string) => {
  if (!t) return '-'
  return new Date(t).toLocaleString('zh-CN')
}

const fetchRecords = async () => {
  loading.value = true
  try {
    const params: any = { page: page.value, page_size: pageSize.value }
    if (filters.job_id) params.job_id = filters.job_id
    if (filters.status) params.status = filters.status
    if (filters.dateRange?.length === 2) {
      params.start_date = filters.dateRange[0]
      params.end_date = filters.dateRange[1]
    }
    const res = await recordAPI.list(params)
    records.value = res.data?.data || []
    total.value = res.data?.total || 0
  } catch (e: any) {
    ElMessage.error(e.message || '获取记录列表失败')
  } finally {
    loading.value = false
  }
}

const fetchJobs = async () => {
  try {
    const res = await jobAPI.list({ page_size: 100 })
    jobs.value = res.data?.data || []
  } catch { /* ignore */ }
}

const handleSearch = () => {
  page.value = 1
  fetchRecords()
}

const handleReset = () => {
  filters.job_id = ''
  filters.status = ''
  filters.dateRange = null
  page.value = 1
  fetchRecords()
}

const showDetail = (row: any) => {
  currentRecord.value = row
  drawerVisible.value = true
}

const handleDownload = async (row: any) => {
  try {
    const res: any = await recordAPI.download(row.id)
    const blob = new Blob([res])
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `backup_${row.id}.sql`
    a.click()
    window.URL.revokeObjectURL(url)
    ElMessage.success('下载成功')
  } catch (e: any) {
    ElMessage.error(e.message || '下载失败')
  }
}

const handleVerify = async (row: any) => {
  try {
    await ElMessageBox.confirm(`确定要验证备份 "${row.job_name}" 吗？`, '提示')
    const res = await verifyAPI.verify(row.id)
    if (res.data.passed) {
      ElMessage.success('验证通过')
    } else {
      ElMessage.warning('验证失败: ' + res.data.error)
    }
    fetchRecords()
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e.message || '验证失败')
  }
}

const handleRestore = async (row: any) => {
  try {
    await ElMessageBox.confirm(
      `确定要恢复备份 "${row.name || row.job_name}" 吗？\n\n备份时间: ${row.started_at}\n文件大小: ${formatSize(row.file_size)}\n\n⚠️ 此操作将覆盖现有数据！`,
      '危险操作：恢复数据库',
      {
        type: 'warning',
        confirmButtonText: '确认恢复',
        cancelButtonText: '取消',
        dangerouslyUseHTMLString: true
      }
    )
    router.push({ path: '/restore', query: { recordId: row.id } })
  } catch {
    // 用户取消
  }
}

const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm('确定要删除这条记录吗？', '警告', { type: 'warning' })
    await recordAPI.delete(row.id)
    ElMessage.success('删除成功')
    fetchRecords()
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e.message || '删除失败')
  }
}

onMounted(() => {
  fetchJobs()
  fetchRecords()
})
</script>

<style scoped>
.filter-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
  align-items: center;
}
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
.error-box {
  margin-top: 16px;
  padding: 12px;
  background: #fef0f0;
  border-radius: 4px;
}
.drawer-actions {
  margin-top: 24px;
  display: flex;
  gap: 12px;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
