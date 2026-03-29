<template>
  <div class="cron-expression">
    <!-- 预设快捷选项 -->
    <div class="preset-section">
      <span class="preset-label">预设:</span>
      <el-radio-group v-model="selectedPreset" @change="handlePresetChange" size="small">
        <el-radio-button label="hourly">每小时</el-radio-button>
        <el-radio-button label="daily">每天</el-radio-button>
        <el-radio-button label="weekly">每周</el-radio-button>
        <el-radio-button label="monthly">每月</el-radio-button>
        <el-radio-button label="custom">自定义</el-radio-button>
      </el-radio-group>
    </div>

    <!-- 自定义表达式 -->
    <div v-if="selectedPreset === 'custom'" class="custom-section">
      <el-form label-width="80px" size="small">
        <el-row :gutter="10">
          <el-col :span="6">
            <el-form-item label="分钟">
              <el-select v-model="custom.minute" placeholder="选择" @change="updateExpression">
                <el-option label="每分钟" value="*" />
                <el-option label="每5分钟" value="*/5" />
                <el-option label="每10分钟" value="*/10" />
                <el-option label="每15分钟" value="*/15" />
                <el-option label="每30分钟" value="*/30" />
                <el-option v-for="m in minutes" :key="m" :label="m" :value="m" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="小时">
              <el-select v-model="custom.hour" placeholder="选择" @change="updateExpression">
                <el-option label="每小时" value="*" />
                <el-option label="每2小时" value="*/2" />
                <el-option label="每6小时" value="*/6" />
                <el-option label="每12小时" value="*/12" />
                <el-option v-for="h in hours" :key="h" :label="h" :value="h" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="日期">
              <el-select v-model="custom.day" placeholder="选择" @change="updateExpression">
                <el-option label="每天" value="*" />
                <el-option label="每月1号" value="1" />
                <el-option label="每月15号" value="15" />
                <el-option label="每月最后一天" value="L" />
                <el-option v-for="d in days" :key="d" :label="d" :value="d" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item label="月份">
              <el-select v-model="custom.month" placeholder="选择" @change="updateExpression">
                <el-option label="每月" value="*" />
                <el-option v-for="m in months" :key="m" :label="m" :value="m" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="24">
            <el-form-item label="星期">
              <el-checkbox-group v-model="custom.weekday" @change="updateExpression">
                <el-checkbox-button label="0">周日</el-checkbox-button>
                <el-checkbox-button label="1">周一</el-checkbox-button>
                <el-checkbox-button label="2">周二</el-checkbox-button>
                <el-checkbox-button label="3">周三</el-checkbox-button>
                <el-checkbox-button label="4">周四</el-checkbox-button>
                <el-checkbox-button label="5">周五</el-checkbox-button>
                <el-checkbox-button label="6">周六</el-checkbox-button>
              </el-checkbox-group>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </div>

    <!-- 表达式显示 -->
    <div class="expression-display">
      <code class="expression-value">{{ expression }}</code>
      <span class="expression-desc">{{ description }}</span>
    </div>

    <!-- 下次执行预览 -->
    <div v-if="nextRuns.length > 0" class="next-runs">
      <span class="next-runs-label">下次执行:</span>
      <ul class="next-runs-list">
        <li v-for="(run, index) in nextRuns" :key="index">{{ run }}</li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'

const props = defineProps<{
  modelValue: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

// 辅助数组
const minutes = Array.from({ length: 60 }, (_, i) => i.toString())
const hours = Array.from({ length: 24 }, (_, i) => i.toString())
const days = Array.from({ length: 31 }, (_, i) => (i + 1).toString())
const months = Array.from({ length: 12 }, (_, i) => (i + 1).toString())

// 预设配置
const presets: Record<string, string> = {
  hourly: '0 * * * *',
  daily: '0 0 * * *',
  weekly: '0 0 * * 0',
  monthly: '0 0 1 * *'
}

// 预设描述
const presetDescriptions: Record<string, string> = {
  hourly: '每小时整点执行',
  daily: '每天凌晨执行',
  weekly: '每周日凌晨执行',
  monthly: '每月1日凌晨执行'
}

const selectedPreset = ref<string>('daily')
const custom = ref({
  minute: '0',
  hour: '0',
  day: '*',
  month: '*',
  weekday: ['0'] as string[]
})

const expression = computed(() => {
  if (selectedPreset.value !== 'custom') {
    return presets[selectedPreset.value]
  }
  
  const weekdayStr = custom.value.weekday.length > 0 
    ? custom.value.weekday.sort().join(',')
    : '*'
  
  return `${custom.value.minute} ${custom.value.hour} ${custom.value.day} ${custom.value.month} ${weekdayStr}`
})

const description = computed(() => {
  if (selectedPreset.value !== 'custom') {
    return presetDescriptions[selectedPreset.value]
  }
  return parseCronDescription(expression.value)
})

// 计算下次执行时间
const nextRuns = computed(() => {
  if (!expression.value) return []
  
  try {
    const runs = getNextRuns(expression.value, 5)
    return runs
  } catch {
    return []
  }
})

const handlePresetChange = (preset: string) => {
  if (preset !== 'custom') {
    emit('update:modelValue', presets[preset])
  } else {
    emit('update:modelValue', expression.value)
  }
}

const updateExpression = () => {
  emit('update:modelValue', expression.value)
}

// 解析 cron 描述
const parseCronDescription = (cron: string): string => {
  const parts = cron.split(' ')
  if (parts.length !== 5) return '无效表达式'
  
  const [minute, hour, day, month, weekday] = parts
  
  // 简化描述
  if (minute === '*' && hour === '*') return '每分钟执行'
  if (minute === '0' && hour === '*') return '每小时整点执行'
  if (minute === '0' && hour === '0' && day === '*' && month === '*' && weekday === '*') return '每天凌晨执行'
  if (minute === '0' && hour === '0' && day === '*' && month === '*' && weekday !== '*') return '每周执行'
  if (minute === '0' && hour === '0' && day === '1' && month === '*' && weekday === '*') return '每月1日凌晨执行'
  
  // 自定义描述
  const parts_desc: string[] = []
  
  if (minute !== '*') parts_desc.push(`${minute}分`)
  if (hour !== '*') parts_desc.push(`${hour}时`)
  if (day !== '*') parts_desc.push(`${day}号`)
  if (month !== '*') parts_desc.push(`${month}月`)
  if (weekday !== '*' && weekday !== '0,1,2,3,4,5,6') {
    const weekdayNames: Record<string, string> = {
      '0': '周日', '1': '周一', '2': '周二', '3': '周三', '4': '周四', '5': '周五', '6': '周六'
    }
    parts_desc.push(weekday.split(',').map(w => weekdayNames[w] || w).join('、'))
  }
  
  return parts_desc.join(' ') || '每分钟执行'
}

// 计算下次执行时间（简化版）
const getNextRuns = (cron: string, count: number): string[] => {
  const parts = cron.split(' ')
  if (parts.length !== 5) return []
  
  const [minute, hour, day, month, weekday] = parts
  const runs: string[] = []
  const now = new Date()
  
  for (let i = 1; i <= 30 && runs.length < count; i++) {
    const next = new Date(now.getTime() + i * 60 * 1000)
    
    // 简化检查：每分钟检查一次
    const nextMinute = next.getMinutes()
    const nextHour = next.getHours()
    const nextDay = next.getDate()
    const nextMonth = next.getMonth() + 1
    const nextWeekday = next.getDay()
    
    // 基础检查
    if (minute !== '*' && minute !== nextMinute.toString() && !minute.includes(nextMinute.toString())) continue
    if (hour !== '*' && hour !== nextHour.toString() && !hour.includes(nextHour.toString())) continue
    if (day !== '*' && day !== nextDay.toString()) continue
    if (month !== '*' && month !== nextMonth.toString()) continue
    if (weekday !== '*' && !weekday.split(',').includes(nextWeekday.toString())) continue
    
    runs.push(formatDate(next))
  }
  
  return runs
}

const formatDate = (date: Date): string => {
  const pad = (n: number) => n.toString().padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

// 初始化：从外部值解析
watch(() => props.modelValue, (val) => {
  if (!val) return
  
  // 检查是否是预设
  const presetKey = Object.entries(presets).find(([_, v]) => v === val)?.[0]
  if (presetKey) {
    selectedPreset.value = presetKey
  } else {
    selectedPreset.value = 'custom'
    // 解析自定义表达式
    const parts = val.split(' ')
    if (parts.length === 5) {
      custom.value.minute = parts[0]
      custom.value.hour = parts[1]
      custom.value.day = parts[2]
      custom.value.month = parts[3]
      custom.value.weekday = parts[4].split(',')
    }
  }
}, { immediate: true })
</script>

<style scoped>
.cron-expression {
  padding: 8px 0;
}

.preset-section {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.preset-label {
  font-size: 14px;
  color: #606266;
  flex-shrink: 0;
}

.custom-section {
  background: #f5f7fa;
  padding: 12px;
  border-radius: 4px;
  margin-bottom: 16px;
}

.expression-display {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  background: #ecf5ff;
  border-radius: 4px;
  margin-bottom: 12px;
}

.expression-value {
  font-family: 'Courier New', monospace;
  font-size: 14px;
  font-weight: bold;
  color: #409eff;
  background: transparent;
}

.expression-desc {
  font-size: 13px;
  color: #606266;
}

.next-runs {
  font-size: 13px;
}

.next-runs-label {
  color: #909399;
  margin-right: 8px;
}

.next-runs-list {
  display: inline;
  list-style: none;
  padding: 0;
  margin: 0;
}

.next-runs-list li {
  display: inline;
  color: #67c23a;
}

.next-runs-list li:not(:last-child)::after {
  content: '、';
  color: #909399;
}
</style>
