<template>
  <div class="approval-container">
    <!-- 统计卡片 -->
    <el-row :gutter="20" class="stats-row">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-value" style="color: #409eff">{{ stats.pending }}</div>
            <div class="stat-label">待审批</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-value" style="color: #67c23a">{{ stats.approved }}</div>
            <div class="stat-label">已通过</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-value" style="color: #f56c6c">{{ stats.rejected }}</div>
            <div class="stat-label">已拒绝</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-value" style="color: #909399">{{ stats.executed }}</div>
            <div class="stat-label">已执行</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 标签页 -->
    <el-tabs v-model="activeTab" class="approval-tabs">
      <el-tab-pane label="待我审批" name="pending">
        <el-table :data="pendingList" stripe class="approval-table">
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column label="类型" width="120">
            <template #default="{ row }">
              <el-tag>{{ getTypeLabel(row.type) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="title" label="标题" min-width="200" />
          <el-table-column prop="applicant_name" label="申请人" width="120" />
          <el-table-column prop="resource_name" label="关联资源" min-width="150" />
          <el-table-column prop="applied_at" label="申请时间" width="180">
            <template #default="{ row }">
              {{ formatTime(row.applied_at) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="160" fixed="right">
            <template #default="{ row }">
              <el-button type="primary" link @click="viewDetail(row)">详情</el-button>
              <el-button type="success" link @click="showApproveDialog(row)">通过</el-button>
              <el-button type="danger" link @click="showRejectDialog(row)">拒绝</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="pagination">
          <el-pagination
            v-model:current-page="pendingPage"
            :total="pendingTotal"
            layout="total, prev, pager, next"
            @current-change="loadPending"
          />
        </div>
      </el-tab-pane>

      <el-tab-pane label="我的申请" name="my">
        <el-form :inline="true" class="filter-form">
          <el-form-item label="状态">
            <el-select v-model="filter.status" placeholder="全部" clearable style="width: 120px">
              <el-option label="待审批" value="pending" />
              <el-option label="已通过" value="approved" />
              <el-option label="已拒绝" value="rejected" />
              <el-option label="已取消" value="cancelled" />
              <el-option label="已执行" value="executed" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="loadMy">查询</el-button>
            <el-button @click="filter.status = ''; loadMy()">重置</el-button>
          </el-form-item>
        </el-form>

        <el-table :data="myList" stripe class="approval-table">
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column label="类型" width="120">
            <template #default="{ row }">
              <el-tag>{{ getTypeLabel(row.type) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="getStatusType(row.status)">{{ getStatusLabel(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="title" label="标题" min-width="200" />
          <el-table-column prop="resource_name" label="关联资源" min-width="150" />
          <el-table-column prop="applied_at" label="申请时间" width="180">
            <template #default="{ row }">
              {{ formatTime(row.applied_at) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="120" fixed="right">
            <template #default="{ row }">
              <el-button type="primary" link @click="viewDetail(row)">详情</el-button>
              <el-button v-if="row.status === 'pending'" type="danger" link @click="cancelApproval(row)">取消</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="pagination">
          <el-pagination
            v-model:current-page="myPage"
            :total="myTotal"
            layout="total, prev, pager, next"
            @current-change="loadMy"
          />
        </div>
      </el-tab-pane>

      <el-tab-pane label="全部记录" name="all">
        <el-table :data="allList" stripe class="approval-table">
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column label="类型" width="120">
            <template #default="{ row }">
              <el-tag>{{ getTypeLabel(row.type) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="getStatusType(row.status)">{{ getStatusLabel(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="title" label="标题" min-width="200" />
          <el-table-column prop="applicant_name" label="申请人" width="120" />
          <el-table-column prop="approver_name" label="审批人" width="120">
            <template #default="{ row }">
              {{ row.approver_name || '-' }}
            </template>
          </el-table-column>
          <el-table-column prop="applied_at" label="申请时间" width="180">
            <template #default="{ row }">
              {{ formatTime(row.applied_at) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="80" fixed="right">
            <template #default="{ row }">
              <el-button type="primary" link @click="viewDetail(row)">详情</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="pagination">
          <el-pagination
            v-model:current-page="allPage"
            :total="allTotal"
            layout="total, prev, pager, next"
            @current-change="loadAll"
          />
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- 详情对话框 -->
    <el-dialog v-model="detailVisible" title="审批详情" width="600px">
      <div v-if="currentApproval" class="approval-detail">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="审批类型">
            <el-tag>{{ getTypeLabel(currentApproval.type) }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="审批状态">
            <el-tag :type="getStatusType(currentApproval.status)">
              {{ getStatusLabel(currentApproval.status) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="申请人">{{ currentApproval.applicant_name }}</el-descriptions-item>
          <el-descriptions-item label="申请时间">{{ formatTime(currentApproval.applied_at) }}</el-descriptions-item>
          <el-descriptions-item label="审批人">{{ currentApproval.approver_name || '待分配' }}</el-descriptions-item>
          <el-descriptions-item v-if="currentApproval.approved_at" label="审批时间">
            {{ formatTime(currentApproval.approved_at) }}
          </el-descriptions-item>
          <el-descriptions-item label="关联资源" :span="2">
            {{ currentApproval.resource_name }} ({{ currentApproval.resource_type }} #{{ currentApproval.resource_id }})
          </el-descriptions-item>
          <el-descriptions-item label="申请标题" :span="2">{{ currentApproval.title }}</el-descriptions-item>
          <el-descriptions-item label="申请说明" :span="2">
            <pre class="content-pre">{{ currentApproval.content || '-' }}</pre>
          </el-descriptions-item>
          <el-descriptions-item v-if="currentApproval.approve_note" label="审批意见" :span="2">
            {{ currentApproval.approve_note }}
          </el-descriptions-item>
          <el-descriptions-item v-if="currentApproval.reject_reason" label="拒绝原因" :span="2">
            {{ currentApproval.reject_reason }}
          </el-descriptions-item>
        </el-descriptions>

        <!-- 详情配置 -->
        <div v-if="currentApproval.details" class="details-section">
          <h4>详细配置</h4>
          <pre class="content-pre">{{ JSON.stringify(currentApproval.details, null, 2) }}</pre>
        </div>
      </div>
    </el-dialog>

    <!-- 通过对话框 -->
    <el-dialog v-model="approveDialogVisible" title="通过审批" width="400px">
      <el-form>
        <el-form-item label="审批意见">
          <el-input v-model="approveNote" type="textarea" :rows="3" placeholder="可选" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="approveDialogVisible = false">取消</el-button>
        <el-button type="success" @click="handleApprove">确认通过</el-button>
      </template>
    </el-dialog>

    <!-- 拒绝对话框 -->
    <el-dialog v-model="rejectDialogVisible" title="拒绝审批" width="400px">
      <el-form>
        <el-form-item label="拒绝原因" required>
          <el-input v-model="rejectReason" type="textarea" :rows="3" placeholder="请输入拒绝原因" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="rejectDialogVisible = false">取消</el-button>
        <el-button type="danger" @click="handleReject">确认拒绝</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '@/api'

const activeTab = ref('pending')

// 统计数据
const stats = reactive({
  pending: 0,
  approved: 0,
  rejected: 0,
  executed: 0
})

// 筛选
const filter = reactive({
  status: ''
})

// 待我审批
const pendingList = ref([])
const pendingPage = ref(1)
const pendingTotal = ref(0)

// 我的申请
const myList = ref([])
const myPage = ref(1)
const myTotal = ref(0)

// 全部记录
const allList = ref([])
const allPage = ref(1)
const allTotal = ref(0)

// 详情
const detailVisible = ref(false)
const currentApproval = ref<any>(null)

// 通过对话框
const approveDialogVisible = ref(false)
const approveNote = ref('')

// 拒绝对话框
const rejectDialogVisible = ref(false)
const rejectReason = ref('')

// 类型映射
const typeOptions = [
  { label: '恢复审批', value: 'restore' },
  { label: '任务删除', value: 'task_delete' },
  { label: '配置变更', value: 'config_change' }
]

// 状态映射
const statusOptions = [
  { label: '待审批', value: 'pending', type: 'warning' },
  { label: '已通过', value: 'approved', type: 'success' },
  { label: '已拒绝', value: 'rejected', type: 'danger' },
  { label: '已取消', value: 'cancelled', type: 'info' },
  { label: '已执行', value: 'executed', type: 'success' }
]

// 加载统计数据
const loadStats = async () => {
  try {
    const res = await api.get('/approvals/pending/count')
    if (res.code === 0) {
      stats.pending = res.data.pending_count
    }

    // 按状态统计
    const allRes = await api.get('/approvals', { params: { page_size: 1 } })
    if (allRes.code === 0) {
      // 获取各状态数量
      const pendingRes = await api.get('/approvals', { params: { status: 'pending', page_size: 1 } })
      const approvedRes = await api.get('/approvals', { params: { status: 'approved', page_size: 1 } })
      const rejectedRes = await api.get('/approvals', { params: { status: 'rejected', page_size: 1 } })
      const executedRes = await api.get('/approvals', { params: { status: 'executed', page_size: 1 } })

      if (pendingRes.code === 0) stats.pending = pendingRes.data.total
      if (approvedRes.code === 0) stats.approved = approvedRes.data.total
      if (rejectedRes.code === 0) stats.rejected = rejectedRes.data.total
      if (executedRes.code === 0) stats.executed = executedRes.data.total
    }
  } catch (e) {
    console.error('加载统计失败', e)
  }
}

// 加载待我审批
const loadPending = async () => {
  try {
    const res = await api.get('/approvals', {
      params: {
        role: 'approver',
        status: 'pending',
        page: pendingPage.value,
        page_size: 10
      }
    })
    if (res.code === 0) {
      pendingList.value = res.data.items
      pendingTotal.value = res.data.total
    }
  } catch (e) {
    console.error('加载待审批失败', e)
  }
}

// 加载我的申请
const loadMy = async () => {
  try {
    const res = await api.get('/approvals', {
      params: {
        role: 'applicant',
        status: filter.status || undefined,
        page: myPage.value,
        page_size: 10
      }
    })
    if (res.code === 0) {
      myList.value = res.data.items
      myTotal.value = res.data.total
    }
  } catch (e) {
    console.error('加载我的申请失败', e)
  }
}

// 加载全部
const loadAll = async () => {
  try {
    const res = await api.get('/approvals', {
      params: {
        page: allPage.value,
        page_size: 10
      }
    })
    if (res.code === 0) {
      allList.value = res.data.items
      allTotal.value = res.data.total
    }
  } catch (e) {
    console.error('加载全部失败', e)
  }
}

// 查看详情
const viewDetail = async (row: any) => {
  try {
    const res = await api.get(`/approvals/${row.id}`)
    if (res.code === 0) {
      currentApproval.value = res.data
      detailVisible.value = true
    }
  } catch (e) {
    ElMessage.error('加载详情失败')
  }
}

// 显示通过对话框
const showApproveDialog = (row: any) => {
  currentApproval.value = row
  approveNote.value = ''
  approveDialogVisible.value = true
}

// 处理通过
const handleApprove = async () => {
  if (!currentApproval.value) return
  try {
    const res = await api.post(`/approvals/${currentApproval.value.id}/approve`, {
      note: approveNote.value
    })
    if (res.code === 0) {
      ElMessage.success('已通过')
      approveDialogVisible.value = false
      loadPending()
      loadStats()
    }
  } catch (e) {
    ElMessage.error('操作失败')
  }
}

// 显示拒绝对话框
const showRejectDialog = (row: any) => {
  currentApproval.value = row
  rejectReason.value = ''
  rejectDialogVisible.value = true
}

// 处理拒绝
const handleReject = async () => {
  if (!currentApproval.value) return
  if (!rejectReason.value.trim()) {
    ElMessage.warning('请输入拒绝原因')
    return
  }
  try {
    const res = await api.post(`/approvals/${currentApproval.value.id}/reject`, {
      reason: rejectReason.value
    })
    if (res.code === 0) {
      ElMessage.success('已拒绝')
      rejectDialogVisible.value = false
      loadPending()
      loadStats()
    }
  } catch (e) {
    ElMessage.error('操作失败')
  }
}

// 取消申请
const cancelApproval = async (row: any) => {
  try {
    await ElMessageBox.confirm('确定取消此申请?', '取消申请')
    const res = await api.post(`/approvals/${row.id}/cancel`)
    if (res.code === 0) {
      ElMessage.success('已取消')
      loadMy()
      loadStats()
    }
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error('操作失败')
    }
  }
}

// 工具函数
const getTypeLabel = (type: string) => {
  const opt = typeOptions.find(t => t.value === type)
  return opt?.label || type
}

const getStatusLabel = (status: string) => {
  const opt = statusOptions.find(s => s.value === status)
  return opt?.label || status
}

const getStatusType = (status: string) => {
  const opt = statusOptions.find(s => s.value === status)
  return opt?.type || 'info'
}

const formatTime = (time: string) => {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

onMounted(() => {
  loadStats()
  loadPending()
  loadMy()
  loadAll()
})
</script>

<style scoped>
.approval-container {
  padding: 20px;
}

.stats-row {
  margin-bottom: 20px;
}

.stat-card {
  text-align: center;
}

.stat-content {
  padding: 10px 0;
}

.stat-value {
  font-size: 32px;
  font-weight: bold;
}

.stat-label {
  color: #909399;
  font-size: 14px;
  margin-top: 8px;
}

.approval-tabs {
  background: #fff;
  padding: 20px;
  border-radius: 4px;
}

.filter-form {
  margin-bottom: 16px;
}

.pagination {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

.content-pre {
  background: #f5f7fa;
  padding: 12px;
  border-radius: 4px;
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
}

.details-section {
  margin-top: 20px;
}

.details-section h4 {
  margin-bottom: 10px;
  font-size: 14px;
  color: #606266;
}
</style>
