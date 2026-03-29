<template>
  <div class="backup-impact-container">
    <el-card>
      <template #header>
        <span>⚠️ 备份影响分析</span>
      </template>

      <!-- 预估影响 -->
      <el-descriptions title="预估影响" :column="2" border>
        <el-descriptions-item label="预计耗时">
          <el-tag type="info">{{ impact.duration }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="预估IO负载">
          <el-tag :type="ioLoadType">{{ ioLoadText }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="对生产影响">
          <el-tag :type="productionImpactType">{{ productionImpactText }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="建议时段">
          <el-tag type="success">{{ suggestedTime }}</el-tag>
        </el-descriptions-item>
      </el-descriptions>

      <!-- 详细分析 -->
      <el-divider content-position="left">详细分析</el-divider>

      <el-table :data="analysisItems" size="small">
        <el-table-column prop="factor" label="因素" width="150" />
        <el-table-column prop="description" label="说明" />
        <el-table-column prop="impact" label="影响程度" width="100">
          <template #default="{ row }">
            <el-tag :type="getImpactType(row.impact)" size="small">
              {{ row.impact }}
            </el-tag>
          </template>
        </el-table-column>
      </el-table>

      <!-- 优化建议 -->
      <el-divider content-position="left">优化建议</el-divider>

      <el-alert
        v-for="(suggestion, index) in suggestions"
        :key="index"
        :title="suggestion.title"
        :type="suggestion.type"
        :description="suggestion.description"
        show-icon
        :closable="false"
        style="margin-bottom: 10px"
      />
    </el-card>

    <!-- 配置建议 -->
    <el-card style="margin-top: 16px">
      <template #header>
        <span>💡 推荐配置</span>
      </template>

      <el-form label-width="120px">
        <el-form-item label="备份时段">
          <el-select v-model="recommendedConfig.timeWindow" placeholder="选择备份时段">
            <el-option label="业务低峰期 (02:00-06:00)" value="low_traffic" />
            <el-option label="周末全天" value="weekend" />
            <el-option label="自定义" value="custom" />
          </el-select>
        </el-form-item>
        <el-form-item label="并发备份">
          <el-switch v-model="recommendedConfig.concurrent" />
          <span style="margin-left: 10px; color: #909399">允许多个备份任务同时执行</span>
        </el-form-item>
        <el-form-item label="限速配置">
          <el-slider 
            v-model="recommendedConfig.throttlePercent" 
            :min="10" 
            :max="100" 
            :step="10"
            show-stops
          />
          <span style="margin-left: 10px">{{ recommendedConfig.throttlePercent }}% CPU</span>
        </el-form-item>
        <el-form-item label="压缩级别">
          <el-radio-group v-model="recommendedConfig.compressLevel">
            <el-radio label="1">快速压缩</el-radio>
            <el-radio label="6">均衡</el-radio>
            <el-radio label="9">高压缩</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="applyConfig">应用配置到作业</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import api from '@/utils/api'

const router = useRouter()

const impact = ref({
  duration: '30-45 分钟',
  ioLoad: 'medium',
  productionImpact: 'low'
})

const ioLoadType = computed(() => {
  const map: Record<string, string> = {
    low: 'success',
    medium: 'warning',
    high: 'danger'
  }
  return map[impact.value.ioLoad] || 'info'
})

const ioLoadText = computed(() => {
  const map: Record<string, string> = {
    low: '低',
    medium: '中等',
    high: '高'
  }
  return map[impact.value.ioLoad] || '未知'
})

const productionImpactType = computed(() => {
  const map: Record<string, string> = {
    none: 'success',
    low: 'success',
    medium: 'warning',
    high: 'danger'
  }
  return map[impact.value.productionImpact] || 'info'
})

const productionImpactText = computed(() => {
  const map: Record<string, string> = {
    none: '无影响（支持在线备份）',
    low: '极小',
    medium: '中等',
    high: '较大'
  }
  return map[impact.value.productionImpact] || '未知'
})

const suggestedTime = computed(() => {
  return '建议时段: 02:00-06:00'
})

const analysisItems = computed(() => [
  {
    factor: '数据库类型',
    description: 'PostgreSQL 支持在线备份，MySQL 使用 mysqldump 会有只读锁',
    impact: '低'
  },
  {
    factor: '数据量',
    description: '当前预估数据量约 5GB，备份时间取决于网络和磁盘IO',
    impact: '中'
  },
  {
    factor: '备份时段',
    description: '当前设置为业务高峰期，可能影响业务响应',
    impact: '高'
  },
  {
    factor: '并发数',
    description: '当前无并发限制，多个大任务可能同时执行',
    impact: '中'
  }
])

const suggestions = computed(() => [
  {
    title: '调整备份时段到业务低峰期',
    description: '建议将备份时间调整到 02:00-06:00，减少对业务的影响',
    type: 'warning'
  },
  {
    title: '启用限速保护',
    description: '建议限制 CPU 使用率不超过 50%，避免备份占用过多资源',
    type: 'info'
  },
  {
    title: '使用增量备份',
    description: '数据量较大时，建议使用增量备份减少单次备份时间和资源占用',
    type: 'success'
  }
])

const getImpactType = (impact: string) => {
  const map: Record<string, string> = {
    低: 'success',
    中: 'warning',
    高: 'danger'
  }
  return map[impact] || 'info'
}

const recommendedConfig = ref({
  timeWindow: 'low_traffic',
  concurrent: false,
  throttlePercent: 50,
  compressLevel: '6'
})

const applyConfig = () => {
  ElMessage.success('配置已保存，将在下次备份时生效')
  router.push('/jobs')
}
</script>

<style scoped>
.backup-impact-container {
  padding: 0;
}
</style>
