<template>
  <div class="storage-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>存储管理</span>
          <el-button circle @click="fetchStats" :icon="Refresh" />
        </div>
      </template>

      <!-- 存储统计卡片 -->
      <el-row :gutter="20" class="stats-row">
        <el-col :span="8">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-label">总备份记录</div>
              <div class="stat-value">{{ stats.total_records }}</div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="8">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-label">总存储大小</div>
              <div class="stat-value">{{ formatSize(stats.total_size) }}</div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="8">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <div class="stat-label">存储类型</div>
              <div class="stat-value">{{ stats.storages?.length || 0 }} 种</div>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <!-- 存储后端列表 -->
      <el-tabs v-model="activeStorage" @tab-change="handleStorageChange">
        <el-tab-pane 
          v-for="st in stats.storages" 
          :key="st.type" 
          :label="getStorageLabel(st.type)" 
          :name="st.type"
        >
          <el-card shadow="never">
            <template #header>
              <div class="storage-header">
                <span>{{ getStorageLabel(st.type) }}</span>
                <el-tag :type="st.enabled ? 'success' : 'info'">
                  {{ st.enabled ? '已启用' : '未启用' }}
                </el-tag>
              </div>
            </template>

            <el-descriptions :column="2" border>
              <el-descriptions-item label="存储类型">{{ st.type }}</el-descriptions-item>
              <el-descriptions-item label="对象数量">{{ st.object_count }}</el-descriptions-item>
              <el-descriptions-item label="总大小" :span="2">{{ formatSize(st.total_size) }}</el-descriptions-item>
            </el-descriptions>

            <!-- 对象列表 -->
            <el-divider>存储对象</el-divider>
            
            <div class="objects-toolbar">
              <el-input 
                v-model="prefixFilter" 
                placeholder="前缀过滤" 
                style="width: 200px"
                clearable
              />
              <el-button type="primary" @click="fetchObjects" :loading="objectsLoading">刷新</el-button>
            </div>

            <el-table 
              :data="filteredObjects" 
              v-loading="objectsLoading"
              style="width: 100%; margin-top: 12px"
              max-height="400"
            >
              <el-table-column prop="key" label="对象Key" min-width="200" show-overflow-tooltip />
              <el-table-column prop="size" label="大小" width="120">
                <template #default="{ row }">
                  {{ formatSize(row.size) }}
                </template>
              </el-table-column>
              <el-table-column prop="modTime" label="修改时间" width="180" />
              <el-table-column label="操作" width="180" fixed="right">
                <template #default="{ row }">
                  <el-button size="small" type="primary" link @click="handleDownload(row)">下载</el-button>
                  <el-button size="small" type="danger" link @click="handleDelete(row)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-card>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- 确认删除对话框 -->
    <el-dialog v-model="deleteDialogVisible" title="确认删除" width="400px">
      <p>确定要删除对象 <strong>{{ deleteTarget?.key }}</strong> 吗？</p>
      <p style="color: #f56c6c; margin-top: 10px">此操作不可恢复！</p>
      <template #footer>
        <el-button @click="deleteDialogVisible = false">取消</el-button>
        <el-button type="danger" @click="confirmDelete" :loading="deleteLoading">删除</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import api from '@/utils/api'

const activeStorage = ref('local')
const stats = ref<any>({
  storages: [],
  total_records: 0,
  total_size: 0
})
const objects = ref<any[]>([])
const objectsLoading = ref(false)
const prefixFilter = ref('')

const deleteDialogVisible = ref(false)
const deleteTarget = ref<any>(null)
const deleteLoading = ref(false)

const filteredObjects = computed(() => {
  if (!prefixFilter.value) return objects.value
  return objects.value.filter(obj => 
    obj.key.includes(prefixFilter.value)
  )
})

const getStorageLabel = (type: string) => {
  const labels: Record<string, string> = {
    local: '本地存储',
    s3: 'S3 兼容存储',
    oss: '阿里云 OSS',
    cos: '腾讯云 COS'
  }
  return labels[type] || type
}

const formatSize = (bytes: number) => {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const fetchStats = async () => {
  try {
    const res = await api.get('/storage/stats')
    stats.value = res.data || { storages: [], total_records: 0, total_size: 0 }
    if (stats.value.storages?.length > 0 && !activeStorage.value) {
      activeStorage.value = stats.value.storages[0].type
    }
    handleStorageChange()
  } catch (e: any) {
    ElMessage.error(e.message || '获取存储统计失败')
  }
}

const handleStorageChange = () => {
  if (activeStorage.value) {
    fetchObjects()
  }
}

const fetchObjects = async () => {
  if (!activeStorage.value) return
  
  objectsLoading.value = true
  try {
    const res = await api.get('/storage/objects', {
      params: { type: activeStorage.value, prefix: prefixFilter.value }
    })
    objects.value = res.data?.objects || []
  } catch (e: any) {
    ElMessage.error(e.message || '获取对象列表失败')
  } finally {
    objectsLoading.value = false
  }
}

const handleDownload = async (row: any) => {
  try {
    const res = await api.get('/storage/signed-url', {
      params: { type: activeStorage.value, key: row.key }
    })
    if (res.data?.url) {
      window.open(res.data.url, '_blank')
    } else {
      ElMessage.warning('无法获取下载链接')
    }
  } catch (e: any) {
    ElMessage.error(e.message || '获取下载链接失败')
  }
}

const handleDelete = (row: any) => {
  deleteTarget.value = row
  deleteDialogVisible.value = true
}

const confirmDelete = async () => {
  if (!deleteTarget.value) return
  
  deleteLoading.value = true
  try {
    await api.delete('/storage/objects', {
      params: { type: activeStorage.value, key: deleteTarget.value.key }
    })
    ElMessage.success('删除成功')
    deleteDialogVisible.value = false
    fetchObjects()
    fetchStats()
  } catch (e: any) {
    ElMessage.error(e.message || '删除失败')
  } finally {
    deleteLoading.value = false
  }
}

onMounted(() => {
  fetchStats()
})
</script>

<style scoped>
.storage-container {
  padding: 0;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
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

.stat-label {
  font-size: 14px;
  color: #909399;
  margin-bottom: 8px;
}

.stat-value {
  font-size: 24px;
  font-weight: bold;
  color: #303133;
}

.storage-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.objects-toolbar {
  display: flex;
  gap: 12px;
  margin-top: 12px;
}
</style>
