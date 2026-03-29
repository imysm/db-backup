<template>
  <div class="storage-forecast-container">
    <el-card>
      <template #header>
        <span>📅 存储容量预测</span>
      </template>

      <!-- 当前使用情况 -->
      <el-row :gutter="20">
        <el-col :span="8">
          <div class="stat-card">
            <div class="stat-label">当前使用</div>
            <div class="stat-value">{{ formatSize(currentUsage.used) }}</div>
          </div>
        </el-col>
        <el-col :span="8">
          <div class="stat-card">
            <div class="stat-label">可用容量</div>
            <div class="stat-value">{{ formatSize(currentUsage.available) }}</div>
          </div>
        </el-col>
        <el-col :span="8">
          <div class="stat-card">
            <div class="stat-label">预计耗尽</div>
            <div class="stat-value forecast-date">{{ forecastDate }}</div>
          </div>
        </el-col>
      </el-row>

      <!-- 容量趋势图 -->
      <div class="chart-section">
        <div class="chart-title">30天容量趋势</div>
        <div class="chart-container">
          <div class="chart-bars">
            <div 
              v-for="(day, index) in trendData" 
              :key="index"
              class="chart-bar-wrapper"
            >
              <div 
                class="chart-bar" 
                :style="{ height: day.percent + '%' }"
                :class="{ warning: day.percent > 70, danger: day.percent > 90 }"
              >
                <div class="bar-tooltip">{{ day.date }}: {{ day.percent }}%</div>
              </div>
              <div class="chart-label">{{ day.label }}</div>
            </div>
          </div>
        </div>
      </div>

      <!-- 预测结果 -->
      <el-alert
        v-if="forecast"
        :title="forecast.message"
        :type="forecast.type"
        :description="forecast.description"
        show-icon
        :closable="false"
        style="margin-top: 20px"
      >
        <template #default>
          <div class="forecast-detail">
            <span>预计剩余 {{ forecast.days }} 天</span>
            <el-button size="small" type="primary" @click="handleExpandStorage">
              扩展存储
            </el-button>
          </div>
        </template>
      </el-alert>
    </el-card>

    <!-- 建议 -->
    <el-card style="margin-top: 16px">
      <template #header>
        <span>💡 优化建议</span>
      </template>

      <el-table :data="suggestions" size="small">
        <el-table-column prop="action" label="操作" width="200" />
        <el-table-column prop="description" label="说明" />
        <el-table-column prop="potential" label="预计释放" width="120" />
        <el-table-column label="操作" width="120">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="handleAction(row)">
              {{ row.actionText }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import api from '@/utils/api'

const router = useRouter()

const currentUsage = ref({
  used: 0,
  available: 100 * 1024 * 1024 * 1024, // 100GB default
  total: 100 * 1024 * 1024 * 1024
})

const trendData = ref<any[]>([])

const forecastDate = computed(() => {
  if (!forecast.value || forecast.value.days <= 0) return 'N/A'
  const date = new Date()
  date.setDate(date.getDate() + forecast.value.days)
  return `${date.getMonth() + 1}/${date.getDate()}`
})

const forecast = computed(() => {
  const percent = (currentUsage.value.used / currentUsage.value.total) * 100
  const daysRemaining = Math.floor(currentUsage.value.available / (currentUsage.value.used / 30))
  
  if (percent >= 95) {
    return {
      type: 'error',
      message: '存储空间严重不足！',
      description: '建议立即清理过期备份或扩容存储',
      days: daysRemaining
    }
  }
  if (percent >= 90) {
    return {
      type: 'warning',
      message: '存储空间即将耗尽',
      description: '建议尽快清理过期备份或扩容存储',
      days: daysRemaining
    }
  }
  if (percent >= 70) {
    return {
      type: 'warning',
      message: '存储空间使用率较高',
      description: '建议关注并适时清理',
      days: daysRemaining
    }
  }
  return null
})

const suggestions = ref([
  {
    action: '清理30天前备份',
    description: '删除超过30天的备份记录和文件',
    potential: '~20GB',
    actionText: '立即清理'
  },
  {
    action: '调整保留策略',
    description: '将备份保留天数从30天调整为14天',
    potential: '~15GB',
    actionText: '修改策略'
  },
  {
    action: '启用压缩',
    description: '对未压缩的备份启用压缩',
    potential: '~10GB',
    actionText: '查看'
  }
])

const formatSize = (bytes: number) => {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const generateTrendData = () => {
  const data = []
  const today = new Date()
  
  for (let i = 29; i >= 0; i--) {
    const date = new Date(today)
    date.setDate(date.getDate() - i)
    const percent = 30 + Math.random() * 40 + (29 - i) * 0.5
    data.push({
      date: `${date.getMonth() + 1}/${date.getDate()}`,
      label: i % 7 === 0 ? `${date.getMonth() + 1}/${date.getDate()}` : '',
      percent: Math.min(percent, 100)
    })
  }
  
  trendData.value = data
}

const fetchStorageStats = async () => {
  try {
    const res = await api.get('/storage/stats')
    if (res.data) {
      currentUsage.value = {
        used: res.data.total_size || 0,
        available: Math.max(0, (res.data.storages?.[0]?.total_size || 100 * 1024 * 1024 * 1024) - (res.data.total_size || 0)),
        total: res.data.storages?.[0]?.total_size || 100 * 1024 * 1024 * 1024
      }
    }
  } catch (e) {
    console.error('获取存储统计失败', e)
  }
}

const handleExpandStorage = () => {
  ElMessage.info('请联系管理员扩展存储容量')
}

const handleAction = (row: any) => {
  if (row.action.includes('清理')) {
    router.push('/storage')
  } else if (row.action.includes('策略')) {
    router.push('/jobs')
  } else {
    router.push('/storage')
  }
}

onMounted(() => {
  fetchStorageStats()
  generateTrendData()
})
</script>

<style scoped>
.storage-forecast-container {
  padding: 0;
}

.stat-card {
  background: linear-gradient(135deg, #f5f7fa, #e4e8eb);
  padding: 20px;
  border-radius: 8px;
  text-align: center;
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

.forecast-date {
  color: #e6a23c;
}

.chart-section {
  margin-top: 30px;
}

.chart-title {
  font-size: 14px;
  color: #606266;
  margin-bottom: 15px;
}

.chart-container {
  height: 150px;
  padding: 10px 0;
}

.chart-bars {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  height: 100%;
}

.chart-bar-wrapper {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 3%;
}

.chart-bar {
  width: 100%;
  background: linear-gradient(135deg, #67c23a, #85ce61);
  border-radius: 2px 2px 0 0;
  min-height: 5px;
  transition: height 0.3s;
  position: relative;
}

.chart-bar.warning {
  background: linear-gradient(135deg, #e6a23c, #ebb563);
}

.chart-bar.danger {
  background: linear-gradient(135deg, #f56c6c, #f78989);
}

.bar-tooltip {
  display: none;
  position: absolute;
  bottom: 100%;
  left: 50%;
  transform: translateX(-50%);
  background: rgba(0, 0, 0, 0.8);
  color: #fff;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 11px;
  white-space: nowrap;
}

.chart-bar:hover .bar-tooltip {
  display: block;
}

.chart-label {
  font-size: 10px;
  color: #c0c4cc;
  margin-top: 4px;
  transform: rotate(-45deg);
  transform-origin: top left;
}

.forecast-detail {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 10px;
}
</style>
