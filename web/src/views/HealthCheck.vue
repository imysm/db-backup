<template>
  <div class="health-check-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>🔧 系统体检报告</span>
          <el-button type="primary" @click="runHealthCheck" :loading="loading">
            {{ loading ? '检查中...' : '立即体检' }}
          </el-button>
        </div>
      </template>

      <!-- 健康分数 -->
      <div v-if="report" class="score-section">
        <div class="score-circle" :class="scoreClass">
          <div class="score-value">{{ report.score }}</div>
          <div class="score-label">健康分数</div>
        </div>
        <div class="score-info">
          <div class="level-badge" :class="report.level">
            {{ levelText }}
          </div>
          <div class="timestamp">检查时间: {{ report.timestamp }}</div>
        </div>
      </div>

      <!-- 检查项列表 -->
      <div v-if="report" class="items-section">
        <el-collapse>
          <el-collapse-item 
            v-for="category in categories" 
            :key="category.name"
            :title="category.title"
            :name="category.name"
          >
            <el-table :data="getItemsByCategory(category.name)" size="small">
              <el-table-column prop="name" label="检查项" width="150" />
              <el-table-column prop="status" label="状态" width="100">
                <template #default="{ row }">
                  <el-tag :type="getStatusType(row.status)" size="small">
                    {{ getStatusText(row.status) }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="message" label="说明" />
            </el-table>
          </el-collapse-item>
        </el-collapse>
      </div>

      <!-- 空状态 -->
      <el-empty v-if="!report && !loading" description="点击「立即体检」开始健康检查" />
    </el-card>

    <!-- 建议操作 -->
    <el-card v-if="report && suggestions.length > 0" style="margin-top: 16px">
      <template #header>
        <span>📋 建议操作</span>
      </template>
      <el-timeline>
        <el-timeline-item 
          v-for="(suggestion, index) in suggestions" 
          :key="index"
          :color="getSuggestionColor(suggestion.priority)"
        >
          <div class="suggestion-item">
            <div class="suggestion-title">{{ suggestion.title }}</div>
            <div class="suggestion-desc">{{ suggestion.description }}</div>
            <el-button 
              v-if="suggestion.action" 
              size="small" 
              type="primary" 
              @click="handleSuggestionAction(suggestion)"
            >
              {{ suggestion.actionText || '立即处理' }}
            </el-button>
          </div>
        </el-timeline-item>
      </el-timeline>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import api from '@/utils/api'

const router = useRouter()

const loading = ref(false)
const report = ref<any>(null)

interface HealthItem {
  category: string
  name: string
  status: string
  message: string
  detail?: string
}

interface HealthReport {
  score: number
  level: string
  timestamp: string
  items: HealthItem[]
}

const categories = [
  { name: 'database', title: '🗄️ 数据库' },
  { name: 'backup', title: '💾 备份任务' },
  { name: 'records', title: '📋 备份记录' },
  { name: 'storage', title: '💿 存储' },
  { name: 'alert', title: '🔔 告警' }
]

const scoreClass = computed(() => {
  if (!report.value) return ''
  const score = report.value.score
  if (score >= 80) return 'healthy'
  if (score >= 50) return 'warning'
  return 'critical'
})

const levelText = computed(() => {
  if (!report.value) return ''
  const map: Record<string, string> = {
    healthy: '✅ 健康',
    warning: '⚠️ 警告',
    critical: '❌ 危急'
  }
  return map[report.value.level] || report.value.level
})

const getItemsByCategory = (category: string) => {
  if (!report.value) return []
  const categoryMap: Record<string, string[]> = {
    database: ['数据库'],
    backup: ['备份任务'],
    records: ['备份记录'],
    storage: ['存储'],
    alert: ['告警']
  }
  const keywords = categoryMap[category] || [category]
  return report.value.items.filter((item: HealthItem) => 
    keywords.some(k => item.category.includes(k) || item.name.includes(k))
  )
}

const getStatusType = (status: string) => {
  const map: Record<string, string> = {
    pass: 'success',
    warning: 'warning',
    fail: 'danger'
  }
  return map[status] || 'info'
}

const getStatusText = (status: string) => {
  const map: Record<string, string> = {
    pass: '通过',
    warning: '警告',
    fail: '失败'
  }
  return map[status] || status
}

interface Suggestion {
  title: string
  description: string
  action?: string
  actionText?: string
  priority: number
}

const suggestions = computed(() => {
  if (!report.value) return []
  
  const result: Suggestion[] = []
  
  report.value.items.forEach((item: HealthItem) => {
    if (item.status === 'fail') {
      if (item.name.includes('长期无备份')) {
        result.push({
          title: '立即执行全量备份',
          description: `任务"${item.detail || item.message}"超过30天无成功备份，请立即执行一次全量备份验证系统正常`,
          action: '/jobs/new',
          actionText: '创建备份任务',
          priority: 1
        })
      }
      if (item.name.includes('存储')) {
        result.push({
          title: '清理过期备份',
          description: '存储空间即将用尽，请清理过期备份或扩容存储',
          action: '/storage',
          actionText: '查看存储',
          priority: 1
        })
      }
      if (item.name.includes('任务配置') && item.message.includes('禁用')) {
        result.push({
          title: '启用备份任务',
          description: '所有备份任务已禁用，数据将不会自动备份',
          action: '/jobs',
          actionText: '查看任务',
          priority: 1
        })
      }
    } else if (item.status === 'warning') {
      if (item.name.includes('配置不完整')) {
        result.push({
          title: '完善任务配置',
          description: `${item.detail || item.message}，请检查并完善任务配置`,
          action: '/jobs',
          actionText: '查看任务',
          priority: 2
        })
      }
      if (item.name.includes('失败率')) {
        result.push({
          title: '检查备份失败原因',
          description: item.message,
          action: '/records',
          actionText: '查看记录',
          priority: 2
        })
      }
      if (item.name.includes('存储')) {
        result.push({
          title: '关注存储使用',
          description: item.message,
          action: '/storage',
          actionText: '查看存储',
          priority: 2
        })
      }
    }
  })
  
  return result.sort((a, b) => a.priority - b.priority)
})

const getSuggestionColor = (priority: number) => {
  if (priority === 1) return '#f56c6c'
  if (priority === 2) return '#e6a23c'
  return '#67c23a'
}

const runHealthCheck = async () => {
  loading.value = true
  try {
    const res = await api.get('/health/report')
    report.value = res.data
    ElMessage.success('健康检查完成')
  } catch (e: any) {
    ElMessage.error(e.message || '健康检查失败')
  } finally {
    loading.value = false
  }
}

const handleSuggestionAction = (suggestion: Suggestion) => {
  if (suggestion.action) {
    router.push(suggestion.action)
  }
}

// 自动执行一次检查
runHealthCheck()
</script>

<style scoped>
.health-check-container {
  padding: 0;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.score-section {
  display: flex;
  align-items: center;
  gap: 30px;
  padding: 30px;
  background: linear-gradient(135deg, #f5f7fa, #e4e8eb);
  border-radius: 8px;
  margin-bottom: 20px;
}

.score-circle {
  width: 120px;
  height: 120px;
  border-radius: 50%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #fff;
}

.score-circle.healthy {
  background: linear-gradient(135deg, #67c23a, #85ce61);
}

.score-circle.warning {
  background: linear-gradient(135deg, #e6a23c, #ebb563);
}

.score-circle.critical {
  background: linear-gradient(135deg, #f56c6c, #f78989);
}

.score-value {
  font-size: 36px;
  font-weight: bold;
  line-height: 1;
}

.score-label {
  font-size: 14px;
  margin-top: 8px;
  opacity: 0.9;
}

.score-info {
  flex: 1;
}

.level-badge {
  display: inline-block;
  padding: 8px 16px;
  border-radius: 4px;
  font-size: 16px;
  font-weight: bold;
  margin-bottom: 10px;
}

.level-badge.healthy {
  background: #f0f9eb;
  color: #67c23a;
}

.level-badge.warning {
  background: #fdf6ec;
  color: #e6a23c;
}

.level-badge.critical {
  background: #fef0f0;
  color: #f56c6c;
}

.timestamp {
  color: #909399;
  font-size: 13px;
}

.items-section {
  margin-top: 20px;
}

.suggestion-item {
  padding: 8px 0;
}

.suggestion-title {
  font-weight: bold;
  font-size: 14px;
  margin-bottom: 4px;
}

.suggestion-desc {
  color: #606266;
  font-size: 13px;
  margin-bottom: 8px;
}
</style>
