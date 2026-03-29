<template>
  <div class="dashboard">
    <!-- 统计卡片 -->
    <el-row :gutter="16">
      <el-col :xs="12" :sm="6">
        <el-card class="stat-card" shadow="hover">
          <div class="stat-icon" style="background: linear-gradient(135deg, #409eff, #66b1ff)">
            <el-icon :size="28"><Calendar /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ stats.today_total }}</div>
            <div class="stat-label">今日任务</div>
            <div class="stat-sub">
              成功 <span class="text-success">{{ stats.today_success }}</span>
              / 失败 <span class="text-danger">{{ stats.today_failed }}</span>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card class="stat-card" shadow="hover">
          <div class="stat-icon" style="background: linear-gradient(135deg, #67c23a, #85ce61)">
            <el-icon :size="28"><TrendCharts /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ stats.week_success_rate }}%</div>
            <div class="stat-label">本周成功率</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card class="stat-card" shadow="hover">
          <div class="stat-icon" style="background: linear-gradient(135deg, #e6a23c, #f0c78a)">
            <el-icon :size="28"><Coin /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ formatSize(stats.total_storage_bytes) }}</div>
            <div class="stat-label">总存储量</div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card class="stat-card" shadow="hover">
          <div class="stat-icon" style="background: linear-gradient(135deg, #f56c6c, #fab6b6)">
            <el-icon :size="28"><WarningFilled /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value" :class="{ 'text-danger': stats.last_24h_failed > 0 }">
              {{ stats.last_24h_failed }}
            </div>
            <div class="stat-label">24h 异常告警</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 图表和表格 -->
    <el-row :gutter="16" style="margin-top: 16px">
      <el-col :xs="24" :lg="16">
        <el-card shadow="hover">
          <template #header>
            <span>最近 7 天备份趋势</span>
          </template>
          <v-chart :option="chartOption" style="height: 320px" autoresize />
        </el-card>
      </el-col>
      <el-col :xs="24" :lg="8">
        <el-card shadow="hover">
          <template #header>
            <span>存储概览</span>
          </template>
          <div class="storage-overview">
            <v-chart :option="storageChartOption" style="height: 200px" autoresize />
            <div class="storage-detail">
              <p><span class="label">已启用任务</span> <strong>{{ stats.enabled_tasks }} / {{ stats.total_tasks }}</strong></p>
              <p><span class="label">今日执行中</span> <strong>{{ stats.today_running }}</strong></p>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 最近记录 -->
    <el-card shadow="hover" style="margin-top: 16px">
      <template #header>
        <div class="table-header">
          <span>最近备份记录</span>
          <el-button text type="primary" @click="$router.push('/records')">查看全部</el-button>
        </div>
      </template>
      <el-table :data="recentRecords" v-loading="loading" style="width: 100%">
        <el-table-column prop="job_name" label="任务名称" min-width="150" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)" size="small">
              {{ statusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="file_size" label="文件大小" width="120">
          <template #default="{ row }">{{ formatSize(row.file_size) }}</template>
        </el-table-column>
        <el-table-column prop="started_at" label="开始时间" width="180">
          <template #default="{ row }">{{ formatTime(row.started_at) }}</template>
        </el-table-column>
        <el-table-column prop="duration" label="耗时" width="100">
          <template #default="{ row }">{{ row.duration ? row.duration + 's' : '-' }}</template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 快捷操作 -->
    <el-card shadow="hover" style="margin-top: 16px">
      <template #header>
        <span>快捷操作</span>
      </template>
      <div class="quick-actions">
        <el-button type="primary" @click="$router.push('/jobs/new')">
          <el-icon><Plus /></el-icon>新建任务
        </el-button>
        <el-button @click="$router.push('/records')">
          <el-icon><Document /></el-icon>查看记录
        </el-button>
        <el-button @click="$router.push('/restore')">
          <el-icon><RefreshRight /></el-icon>恢复备份
        </el-button>
        <el-button @click="$router.push('/storage')">
          <el-icon><Box /></el-icon>存储管理
        </el-button>
        <el-button type="warning" @click="$router.push('/health')">
          <el-icon><Clock /></el-icon>系统体检
        </el-button>
        <el-button type="info" @click="$router.push('/storage-forecast')">
          <el-icon><TrendCharts /></el-icon>容量预测
        </el-button>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { BarChart, PieChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent, TitleComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { statsAPI, recordAPI } from '@/api'
import { Calendar, TrendCharts, Coin, WarningFilled, Plus, Document, RefreshRight, Box, Clock } from '@element-plus/icons-vue'

use([CanvasRenderer, BarChart, PieChart, GridComponent, TooltipComponent, LegendComponent, TitleComponent])

const loading = ref(false)

const stats = ref({
  total_tasks: 0,
  enabled_tasks: 0,
  today_total: 0,
  today_success: 0,
  today_failed: 0,
  today_running: 0,
  week_success_rate: 0,
  total_storage_bytes: 0,
  storage_limit_bytes: 0,
  last_24h_failed: 0,
  daily_stats: [] as { date: string; total: number; success: number; failed: number }[]
})

const recentRecords = ref<any[]>([])

// 趋势图配置
const chartOption = computed(() => ({
  tooltip: { trigger: 'axis' },
  legend: { data: ['成功', '失败'], bottom: 0 },
  grid: { left: 40, right: 20, top: 20, bottom: 40 },
  xAxis: {
    type: 'category',
    data: stats.value.daily_stats.map(d => d.date.slice(5)),
    axisLabel: { fontSize: 12 }
  },
  yAxis: {
    type: 'value',
    minInterval: 1,
    axisLabel: { fontSize: 12 }
  },
  series: [
    {
      name: '成功',
      type: 'bar',
      stack: 'total',
      data: stats.value.daily_stats.map(d => d.success),
      itemStyle: { color: '#67c23a', borderRadius: [0, 0, 0, 0] }
    },
    {
      name: '失败',
      type: 'bar',
      stack: 'total',
      data: stats.value.daily_stats.map(d => d.failed),
      itemStyle: { color: '#f56c6c', borderRadius: [4, 4, 0, 0] }
    }
  ]
}))

// 存储进度环图
const storageChartOption = computed(() => ({
  tooltip: { trigger: 'item', formatter: '{b}: {d}%' },
  series: [{
    type: 'pie',
    radius: ['55%', '75%'],
    avoidLabelOverlap: false,
    label: { show: true, position: 'center', formatter: formatSize(stats.value.total_storage_bytes), fontSize: 14, fontWeight: 'bold' },
    emphasis: { label: { show: true, fontSize: 16 } },
    data: [
      { value: stats.value.total_storage_bytes, name: '已使用', itemStyle: { color: '#409eff' } },
      { value: 1, name: '剩余', itemStyle: { color: '#e4e7ed' } }
    ]
  }]
}))

const formatSize = (bytes: number) => {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

const formatTime = (t: string) => t ? t.replace('T', ' ').slice(0, 19) : '-'

const statusTagType = (s: string) => {
  const map: Record<string, string> = { success: 'success', failed: 'danger', running: 'warning', pending: 'info' }
  return map[s] || 'info'
}

const statusLabel = (s: string) => {
  const map: Record<string, string> = { success: '成功', failed: '失败', running: '运行中', pending: '等待中' }
  return map[s] || s
}

onMounted(async () => {
  loading.value = true
  try {
    const [statsRes, recordsRes] = await Promise.all([
      statsAPI.get(),
      recordAPI.list({ page: 1, page_size: 10 })
    ])
    if (statsRes?.data) stats.value = { ...stats.value, ...statsRes.data }
    recentRecords.value = recordsRes?.data?.data || recordsRes?.data || []
  } catch (e) {
    console.error('加载数据失败', e)
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.stat-card {
  display: flex;
  align-items: center;
}

.stat-card :deep(.el-card__body) {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
}

.stat-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  flex-shrink: 0;
}

.stat-value {
  font-size: 24px;
  font-weight: bold;
  color: #303133;
}

.stat-label {
  font-size: 13px;
  color: #909399;
  margin-top: 4px;
}

.stat-sub {
  font-size: 12px;
  color: #c0c4cc;
  margin-top: 2px;
}

.text-success { color: #67c23a; }
.text-danger { color: #f56c6c; }

.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.storage-overview {
  text-align: center;
}

.storage-detail {
  margin-top: 10px;
  text-align: left;
  padding: 0 20px;
}

.storage-detail p {
  margin: 6px 0;
  font-size: 13px;
  color: #606266;
}

.storage-detail .label {
  color: #909399;
  margin-right: 12px;
}

@media (max-width: 768px) {
  .stat-value { font-size: 20px; }
}
</style>
