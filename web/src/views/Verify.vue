<template>
  <div class="verify-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>备份验证</span>
          <el-button
            type="warning"
            :disabled="selectedIds.length === 0"
            @click="handleBatchVerify"
          >批量验证 ({{ selectedIds.length }})</el-button>
        </div>
      </template>

      <!-- 筛选栏 -->
      <div class="filter-bar">
        <el-select v-model="filters.job_id" placeholder="全部任务" clearable style="width: 180px">
          <el-option v-for="job in jobs" :key="job.id" :label="job.name" :value="String(job.id)" />
        </el-select>
        <el-select v-model="filters.verifyStatus" placeholder="验证状态" clearable style="width: 140px">
          <el-option label="已验证" value="verified" />
          <el-option label="未验证" value="unverified" />
          <el-option label="验证失败" value="failed" />
        </el-select>
        <el-button type="primary" @click="handleSearch">搜索</el-button>
        <el-button @click="handleReset">重置</el-button>
      </div>

      <!-- 表格 -->
      <el-table
        :data="records"
        v-loading="loading"
        style="width: 100%"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="50" />
        <el-table-column label="备份记录" min-width="160">
          <template #default="{ row }">
            {{ row.job_name }} #{{ row.id }}
          </template>
        </el-table-column>
        <el-table-column label="备份状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getBackupStatusType(row.status)" size="small">
              {{ getBackupStatusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="验证状态" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.verified" type="success" size="small">已验证</el-tag>
            <el-tag v-else-if="row.verify_error" type="danger" size="small">验证失败</el-tag>
            <el-tag v-else type="info" size="small">未验证</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="验证结果" width="160">
          <template #default="{ row }">
            <span v-if="row.verified">校验和匹配</span>
            <span v-else-if="row.verify_error" style="color: #f56c6c">{{ row.verify_error }}</span>
            <span v-else style="color: #909399">-</span>
          </template>
        </el-table-column>
        <el-table-column label="验证时间" width="170">
          <template #default="{ row }">
            {{ row.verified_at ? formatTime(row.verified_at) : '-' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button
              size="small" link type="primary"
              :disabled="row.status !== 'success'"
              @click="handleVerify(row)"
            >手动验证</el-button>
            <el-button
              size="small" link type="warning"
              :disabled="row.status !== 'success'"
              @click="handleTestRestore(row)"
            >测试恢复</el-button>
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
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { recordAPI, verifyAPI, jobAPI } from '@/api'

const loading = ref(false)
const records = ref<any[]>([])
const jobs = ref<any[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const selectedIds = ref<number[]>([])

const filters = reactive({
  job_id: '',
  verifyStatus: ''
})

const getBackupStatusType = (status: string) => {
  const map: any = { success: 'success', running: 'warning', failed: 'danger', pending: 'info' }
  return map[status] || 'info'
}

const getBackupStatusText = (status: string) => {
  const map: any = { success: '成功', running: '运行中', failed: '失败', pending: '等待中' }
  return map[status] || status
}

const formatTime = (t: string) => {
  if (!t) return '-'
  return new Date(t).toLocaleString('zh-CN')
}

const fetchRecords = async () => {
  loading.value = true
  try {
    const params: any = { page: page.value, page_size: pageSize.value, sort_by: 'started_at', sort_order: 'desc' }
    if (filters.job_id) params.job_id = filters.job_id
    // 只获取成功的记录用于验证
    params.status = 'success'
    const res = await recordAPI.list(params)
    let data = res.data?.data || []
    // 客户端过滤验证状态
    if (filters.verifyStatus === 'verified') {
      data = data.filter((r: any) => r.verified)
    } else if (filters.verifyStatus === 'unverified') {
      data = data.filter((r: any) => !r.verified && !r.verify_error)
    } else if (filters.verifyStatus === 'failed') {
      data = data.filter((r: any) => r.verify_error)
    }
    records.value = data
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
  filters.verifyStatus = ''
  page.value = 1
  fetchRecords()
}

const handleSelectionChange = (rows: any[]) => {
  selectedIds.value = rows.map(r => r.id)
}

const handleVerify = async (row: any) => {
  try {
    const res = await verifyAPI.verify(row.id)
    if (res.data.passed) {
      ElMessage.success(`#${row.id} 验证通过`)
    } else {
      ElMessage.warning(`#${row.id} 验证失败: ${res.data.error}`)
    }
    fetchRecords()
  } catch (e: any) {
    ElMessage.error(e.message || '验证失败')
  }
}

const handleTestRestore = async (row: any) => {
  try {
    const res = await verifyAPI.testRestore(row.id)
    if (res.data.passed) {
      ElMessage.success(`#${row.id} 恢复测试通过`)
    } else {
      ElMessage.warning(`#${row.id} 恢复测试失败: ${res.data.error}`)
    }
  } catch (e: any) {
    ElMessage.error(e.message || '测试恢复失败')
  }
}

const handleBatchVerify = async () => {
  try {
    await ElMessageBox.confirm(`确定要批量验证 ${selectedIds.value.length} 条记录吗？`, '提示')
    loading.value = true
    const res = await verifyAPI.batch(selectedIds.value)
    const { results, passed, total: count } = res.data
    ElMessage.success(`批量验证完成：${passed}/${count} 通过`)
    if (passed < count) {
      const failed = results.filter((r: any) => !r.passed)
      console.warn('批量验证失败项:', failed)
    }
    fetchRecords()
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e.message || '批量验证失败')
  } finally {
    loading.value = false
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
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
