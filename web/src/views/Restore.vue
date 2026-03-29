<template>
  <div class="restore-container">
    <!-- Tab 切换 -->
    <el-tabs v-model="activeTab" class="restore-tabs">
      <el-tab-pane label="恢复向导" name="wizard">
        <!-- 步骤条 -->
        <el-steps :active="currentStep" finish-status="success" align-center class="wizard-steps">
          <el-step title="选择备份" />
          <el-step title="恢复目标" />
          <el-step title="预检查" />
          <el-step title="执行恢复" />
        </el-steps>

        <div class="wizard-content">
          <!-- Step 1: 选择备份文件 -->
          <div v-show="currentStep === 0">
            <el-card>
              <template #header><span>选择备份文件</span></template>
              <el-form label-width="100px">
                <el-form-item label="选择任务">
                  <el-select v-model="wizard.selectedJobId" placeholder="请选择备份任务" style="width: 100%" filterable @change="onJobChange">
                    <el-option v-for="job in jobs" :key="job.id" :label="job.name" :value="job.id" />
                  </el-select>
                </el-form-item>
              </el-form>
              <el-table
                v-if="wizard.selectedJobId"
                :data="jobRecords"
                highlight-current-row
                @current-change="onRecordSelect"
                style="width: 100%"
              >
                <el-table-column width="55">
                  <template #default="{ row }">
                    <el-radio v-model="wizard.selectedRecordId" :value="row.id">&nbsp;</el-radio>
                  </template>
                </el-table-column>
                <el-table-column prop="job_name" label="任务名" width="140" />
                <el-table-column label="备份时间" min-width="180">
                  <template #default="{ row }">{{ formatTime(row.started_at) }}</template>
                </el-table-column>
                <el-table-column label="文件大小" width="120">
                  <template #default="{ row }">{{ formatSize(row.file_size) }}</template>
                </el-table-column>
                <el-table-column label="验证状态" width="100" align="center">
                  <template #default="{ row }">
                    <el-tag :type="row.verified ? 'success' : 'info'" size="small">
                      {{ row.verified ? '已验证' : '未验证' }}
                    </el-tag>
                  </template>
                </el-table-column>
              </el-table>
            </el-card>
          </div>

          <!-- Step 2: 配置恢复目标 -->
          <div v-show="currentStep === 1">
            <el-card>
              <template #header><span>配置恢复目标</span></template>
              <el-form :model="wizard.target" label-width="120px">
                <el-form-item label="数据库类型">
                  <el-select v-model="wizard.target.dbType" style="width: 100%">
                    <el-option label="PostgreSQL" value="postgres" />
                    <el-option label="MySQL" value="mysql" />
                    <el-option label="MongoDB" value="mongodb" />
                  </el-select>
                </el-form-item>
                <el-form-item label="主机">
                  <el-input v-model="wizard.target.host" placeholder="localhost" />
                </el-form-item>
                <el-row :gutter="20">
                  <el-col :span="12">
                    <el-form-item label="端口">
                      <el-input-number v-model="wizard.target.port" :min="1" :max="65535" style="width: 100%" />
                    </el-form-item>
                  </el-col>
                  <el-col :span="12">
                    <el-form-item label="数据库名">
                      <el-input v-model="wizard.target.db" placeholder="请输入目标数据库名" />
                    </el-form-item>
                  </el-col>
                </el-row>
                <el-row :gutter="20">
                  <el-col :span="12">
                    <el-form-item label="用户名">
                      <el-input v-model="wizard.target.user" placeholder="请输入用户名" />
                    </el-form-item>
                  </el-col>
                  <el-col :span="12">
                    <el-form-item label="密码">
                      <el-input v-model="wizard.target.pass" type="password" show-password placeholder="请输入密码" />
                    </el-form-item>
                  </el-col>
                </el-row>
              </el-form>
              <el-alert type="warning" show-icon :closable="false" style="margin-bottom: 16px">
                <template #title>⚠️ 恢复操作将覆盖目标数据库中的现有数据，此操作不可撤销！</template>
              </el-alert>
              <el-checkbox v-model="wizard.confirmedRisk">我已了解风险，确认执行恢复</el-checkbox>
            </el-card>
          </div>

          <!-- Step 3: 预检查 -->
          <div v-show="currentStep === 2">
            <el-card v-loading="prechecking">
              <template #header><span>预检查</span></template>
              <div v-if="!precheckResult && !prechecking" class="empty-tip">正在准备预检查...</div>
              <div v-if="precheckResult">
                <div v-for="(check, idx) in precheckResult.checks" :key="idx" class="check-item">
                  <el-icon :class="check.passed ? 'check-pass' : 'check-fail'" :size="20">
                    <CircleCheckFilled v-if="check.passed" />
                    <CircleCloseFilled v-else />
                  </el-icon>
                  <span class="check-name">{{ check.name }}</span>
                  <span class="check-msg">{{ check.message }}</span>
                </div>
                <el-divider />
                <div class="summary-section">
                  <h4>恢复摘要</h4>
                  <p><strong>源备份：</strong>{{ selectedRecord?.job_name }} - {{ formatTime(selectedRecord?.started_at) }}</p>
                  <p><strong>目标：</strong>{{ wizard.target.host }}:{{ wizard.target.port }}/{{ wizard.target.db }}</p>
                </div>
              </div>
            </el-card>
          </div>

          <!-- Step 4: 执行恢复 -->
          <div v-show="currentStep === 3">
            <el-card>
              <template #header><span>执行恢复</span></template>
              <div v-if="restoring" class="executing">
                <el-icon class="is-loading" :size="48" color="#409eff"><Loading /></el-icon>
                <p>正在执行恢复操作，请稍候...</p>
              </div>
              <div v-else-if="restoreResult" class="result-section">
                <el-result
                  :icon="restoreResult.success ? 'success' : 'error'"
                  :title="restoreResult.success ? '恢复成功' : '恢复失败'"
                >
                  <template #sub-title>
                    <p v-if="restoreResult.success">耗时：{{ restoreResult.duration }}</p>
                    <p v-if="!restoreResult.success" style="color: #f56c6c">{{ restoreResult.error }}</p>
                  </template>
                </el-result>
              </div>
            </el-card>
          </div>
        </div>

        <!-- 底部按钮 -->
        <div class="wizard-footer">
          <el-button v-if="currentStep > 0 && currentStep < 3" @click="currentStep--">上一步</el-button>
          <el-button v-if="currentStep < 2" type="primary" :disabled="!canNext" @click="handleNext">
            下一步
          </el-button>
          <el-button
            v-if="currentStep === 2 && precheckResult?.all_passed"
            type="primary"
            :disabled="!wizard.confirmedRisk"
            @click="currentStep = 3; doRestore()"
          >
            确认恢复
          </el-button>
          <el-button
            v-if="currentStep === 2 && precheckResult && !precheckResult.all_passed"
            type="primary"
            @click="currentStep--"
          >
            ← 上一步
          </el-button>
          <el-button v-if="currentStep === 3 && restoreResult" type="primary" @click="activeTab = 'history'">
            查看恢复历史
          </el-button>
          <el-button v-if="currentStep === 3" @click="resetWizard">重新开始</el-button>
        </div>
      </el-tab-pane>

      <el-tab-pane label="恢复历史" name="history">
        <el-table :data="historyList" v-loading="historyLoading" style="width: 100%">
          <el-table-column prop="id" label="ID" width="70" />
          <el-table-column prop="record_id" label="备份ID" width="90" />
          <el-table-column prop="job_name" label="任务名" width="140" />
          <el-table-column prop="details" label="详情" min-width="280" />
          <el-table-column prop="created_at" label="时间" width="180" />
        </el-table>
        <el-pagination
          v-if="historyTotal > 0"
          class="pagination"
          :current-page="historyPage"
          :page-size="historyPageSize"
          :total="historyTotal"
          layout="total, prev, pager, next"
          @current-change="onHistoryPageChange"
        />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { CircleCheckFilled, CircleCloseFilled, Loading } from '@element-plus/icons-vue'
import { restoreAPI, recordAPI } from '@/api'

const activeTab = ref('wizard')
const currentStep = ref(0)
const jobs = ref<any[]>([])
const jobRecords = ref<any[]>([])

// 向导状态
const wizard = ref({
  selectedJobId: null as number | null,
  selectedRecordId: null as number | null,
  target: { dbType: 'postgres', host: 'localhost', port: 5432, db: '', user: '', pass: '' },
  confirmedRisk: false,
})

const prechecking = ref(false)
const precheckResult = ref<any>(null)
const restoring = ref(false)
const restoreResult = ref<any>(null)

// 恢复历史
const historyList = ref<any[]>([])
const historyLoading = ref(false)
const historyPage = ref(1)
const historyPageSize = ref(20)
const historyTotal = ref(0)

const selectedRecord = computed(() => jobRecords.value.find(r => r.id === wizard.value.selectedRecordId))
const canNext = computed(() => {
  if (currentStep.value === 0) return !!wizard.value.selectedRecordId
  if (currentStep.value === 1) return !!wizard.value.target.db && !!wizard.value.target.user
  return false
})

const formatSize = (bytes: number) => {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const formatTime = (t: string) => {
  if (!t) return ''
  return t.replace('T', ' ').substring(0, 19)
}

const fetchJobs = async () => {
  try {
    const res = await recordAPI.list({ page_size: 100, status: 'success' })
    // 从记录中提取唯一任务
    const records: any[] = res.data?.data || []
    const jobMap = new Map<number, any>()
    records.forEach((r: any) => {
      if (!jobMap.has(r.job_id)) {
        jobMap.set(r.job_id, { id: r.job_id, name: r.job_name, dbType: r.database_type })
      }
    })
    jobs.value = Array.from(jobMap.values())
  } catch {
    // fallback: 使用 restore list
    const res = await restoreAPI.list({ page_size: 100 })
    const records: any[] = res.data?.data || []
    const jobMap = new Map<number, any>()
    records.forEach((r: any) => {
      if (!jobMap.has(r.job_id)) {
        jobMap.set(r.job_id, { id: r.job_id, name: r.job_name })
      }
    })
    jobs.value = Array.from(jobMap.values())
  }
}

const onJobChange = async (jobId: number) => {
  wizard.value.selectedRecordId = null
  try {
    const res = await restoreAPI.list({ page_size: 100 })
    jobRecords.value = (res.data?.data || []).filter((r: any) => r.job_id === jobId)
  } catch (e: any) {
    ElMessage.error('获取备份记录失败')
  }
}

const onRecordSelect = (row: any) => {
  if (row) wizard.value.selectedRecordId = row.id
}

const handleNext = () => {
  if (currentStep.value === 1) {
    // 进入预检查
    currentStep.value = 2
    doPrecheck()
  } else {
    currentStep.value++
  }
}

const doPrecheck = async () => {
  if (!wizard.value.selectedRecordId) return
  prechecking.value = true
  precheckResult.value = null
  try {
    const res = await restoreAPI.validatePOST(wizard.value.selectedRecordId, {
      target_host: wizard.value.target.host,
      target_port: wizard.value.target.port,
      target_db: wizard.value.target.db,
    })
    precheckResult.value = res.data
  } catch (e: any) {
    ElMessage.error('预检查失败: ' + (e.message || '未知错误'))
  } finally {
    prechecking.value = false
  }
}

const doRestore = async () => {
  if (!wizard.value.selectedRecordId) return
  restoring.value = true
  restoreResult.value = null
  try {
    const res = await restoreAPI.restore({
      record_id: wizard.value.selectedRecordId,
      target_host: wizard.value.target.host,
      target_port: wizard.value.target.port,
      target_db: wizard.value.target.db,
      target_user: wizard.value.target.user,
      target_pass: wizard.value.target.pass,
    })
    restoreResult.value = res.data
    if (res.data.success) {
      ElMessage.success('恢复成功')
    } else {
      ElMessage.error('恢复失败: ' + res.data.error)
    }
  } catch (e: any) {
    restoreResult.value = { success: false, error: e.message || '恢复失败' }
    ElMessage.error('恢复失败')
  } finally {
    restoring.value = false
  }
}

const resetWizard = () => {
  currentStep.value = 0
  wizard.value.selectedRecordId = null
  wizard.value.selectedJobId = null
  wizard.value.confirmedRisk = false
  precheckResult.value = null
  restoreResult.value = null
  jobRecords.value = []
}

// 恢复历史
const fetchHistory = async () => {
  historyLoading.value = true
  try {
    const res = await restoreAPI.list({ page: historyPage.value, page_size: historyPageSize.value })
    historyList.value = res.data?.data || []
    historyTotal.value = res.data?.total || 0
  } catch {
    ElMessage.error('获取恢复历史失败')
  } finally {
    historyLoading.value = false
  }
}

const onHistoryPageChange = (page: number) => {
  historyPage.value = page
  fetchHistory()
}

watch(activeTab, (tab) => {
  if (tab === 'history') fetchHistory()
})

onMounted(() => {
  fetchJobs()
})
</script>

<style scoped>
.restore-tabs { margin-top: 0; }
.wizard-steps { margin: 20px 0; }
.wizard-content { min-height: 350px; }
.wizard-footer { margin-top: 20px; text-align: center; }

.check-item { display: flex; align-items: center; gap: 10px; padding: 10px 0; border-bottom: 1px solid #f0f0f0; }
.check-pass { color: #67c23a; }
.check-fail { color: #f56c6c; }
.check-name { font-weight: 600; min-width: 100px; }
.check-msg { color: #666; }

.summary-section h4 { margin: 10px 0 8px; color: #303133; }
.summary-section p { margin: 4px 0; color: #606266; }

.executing { text-align: center; padding: 60px 0; }
.executing p { margin-top: 16px; color: #909399; }

.empty-tip { text-align: center; padding: 40px; color: #909399; }
.pagination { margin-top: 16px; justify-content: flex-end; }
</style>
