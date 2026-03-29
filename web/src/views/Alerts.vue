<template>
  <div class="alerts-container">
    <!-- 统计卡片 -->
    <el-row :gutter="20" class="stats-row">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-value" style="color: #f56c6c">{{ stats.total }}</div>
            <div class="stat-label">活跃告警</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-value" style="color: #f56c6c">{{ stats.p0 }}</div>
            <div class="stat-label">P0 紧急</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-value" style="color: #e6a23c">{{ stats.p1 }}</div>
            <div class="stat-label">P1 重要</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-value" style="color: #67c23a">{{ stats.resolved }}</div>
            <div class="stat-label">已解决</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 标签页 -->
    <el-tabs v-model="activeTab" class="alert-tabs">
      <el-tab-pane label="告警列表" name="list">
        <!-- 筛选条件 -->
        <el-form :inline="true" class="filter-form">
          <el-form-item label="级别">
            <el-select v-model="filter.level" placeholder="全部" clearable style="width: 120px">
              <el-option v-for="opt in levelOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="filter.status" placeholder="全部" clearable style="width: 120px">
              <el-option v-for="opt in statusOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="loadAlerts">查询</el-button>
            <el-button @click="resetFilter">重置</el-button>
          </el-form-item>
        </el-form>

        <!-- 告警列表 -->
        <el-table :data="alerts" stripe class="alert-table">
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column label="级别" width="100">
            <template #default="{ row }">
              <el-tag :type="getLevelType(row.level)" effect="dark">{{ row.level }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="getStatusType(row.status)">{{ getStatusLabel(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="title" label="告警标题" min-width="200" />
          <el-table-column prop="task_name" label="任务名称" width="150" />
          <el-table-column prop="event_type" label="事件类型" width="120" />
          <el-table-column prop="triggered_at" label="触发时间" width="180">
            <template #default="{ row }">
              {{ formatTime(row.triggered_at) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="200" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="viewDetail(row)">详情</el-button>
              <el-button link type="success" v-if="row.status === 'active'" @click="acknowledge(row)">确认</el-button>
              <el-button link type="warning" v-if="row.status !== 'resolved'" @click="resolve(row)">解决</el-button>
            </template>
          </el-table-column>
        </el-table>

        <!-- 分页 -->
        <div class="pagination">
          <el-pagination
            v-model:current-page="pagination.page"
            v-model:page-size="pagination.pageSize"
            :total="pagination.total"
            :page-sizes="[10, 20, 50, 100]"
            layout="total, sizes, prev, pager, next"
            @size-change="loadAlerts"
            @current-change="loadAlerts"
          />
        </div>
      </el-tab-pane>

      <el-tab-pane label="规则管理" name="rules">
        <div class="toolbar">
          <el-button type="primary" @click="showRuleDialog()">创建规则</el-button>
        </div>

        <el-table :data="rules" stripe class="rule-table">
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="name" label="规则名称" min-width="150" />
          <el-table-column label="级别" width="100">
            <template #default="{ row }">
              <el-tag :type="getLevelType(row.level)" effect="dark">{{ row.level }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="启用" width="80">
            <template #default="{ row }">
              <el-switch v-model="row.enabled" @change="toggleRule(row)" />
            </template>
          </el-table-column>
          <el-table-column prop="priority" label="优先级" width="80" />
          <el-table-column label="条件" min-width="200">
            <template #default="{ row }">
              <span v-if="row.conditions && row.conditions.length">
                {{ row.conditions.length }} 个条件 ({{ row.condition_op }})
              </span>
            </template>
          </el-table-column>
          <el-table-column prop="matched_count" label="匹配次数" width="100" />
          <el-table-column prop="cooldown" label="冷却(秒)" width="100" />
          <el-table-column label="操作" width="200" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="showRuleDialog(row)">编辑</el-button>
              <el-button link type="info" @click="copyRule(row)">复制</el-button>
              <el-button link type="danger" @click="deleteRule(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>

        <div class="pagination">
          <el-pagination
            v-model:current-page="rulesPage"
            v-model:page-size="rulesPageSize"
            :total="rulesTotal"
            layout="total, prev, pager, next"
            @current-change="loadRules"
          />
        </div>
      </el-tab-pane>

      <el-tab-pane label="渠道管理" name="channels">
        <div class="toolbar">
          <el-button type="primary" @click="showChannelDialog()">创建渠道</el-button>
        </div>

        <el-table :data="channels" stripe class="channel-table">
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="name" label="渠道名称" min-width="150" />
          <el-table-column label="类型" width="100">
            <template #default="{ row }">
              {{ getChannelTypeLabel(row.type) }}
            </template>
          </el-table-column>
          <el-table-column label="启用" width="80">
            <template #default="{ row }">
              <el-switch v-model="row.enabled" @change="toggleChannel(row)" />
            </template>
          </el-table-column>
          <el-table-column prop="priority" label="优先级" width="80" />
          <el-table-column label="健康状态" width="120">
            <template #default="{ row }">
              <el-tag :type="getHealthType(row.health_status)">
                {{ getHealthLabel(row.health_status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="send_count" label="发送次数" width="100" />
          <el-table-column prop="failed_count" label="失败次数" width="100" />
          <el-table-column prop="last_sent_at" label="最后发送" width="180">
            <template #default="{ row }">
              {{ row.last_sent_at ? formatTime(row.last_sent_at) : '-' }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="200" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="showChannelDialog(row)">编辑</el-button>
              <el-button link type="success" @click="testChannel(row)">测试</el-button>
              <el-button link type="danger" @click="deleteChannel(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <!-- 告警详情对话框 -->
    <el-dialog v-model="detailVisible" title="告警详情" width="700px">
      <div v-if="currentAlert" class="alert-detail">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="告警级别">
            <el-tag :type="getLevelType(currentAlert.level)" effect="dark">
              {{ currentAlert.level }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="告警状态">
            <el-tag :type="getStatusType(currentAlert.status)">
              {{ getStatusLabel(currentAlert.status) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="告警标题" :span="2">
            {{ currentAlert.title }}
          </el-descriptions-item>
          <el-descriptions-item label="事件类型">{{ currentAlert.event_type }}</el-descriptions-item>
          <el-descriptions-item label="触发时间">
            {{ formatTime(currentAlert.triggered_at) }}
          </el-descriptions-item>
          <el-descriptions-item label="任务名称">{{ currentAlert.task_name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="数据库类型">{{ currentAlert.db_type || '-' }}</el-descriptions-item>
          <el-descriptions-item label="触发时间" :span="2">
            {{ formatTime(currentAlert.triggered_at) }}
          </el-descriptions-item>
          <el-descriptions-item label="告警内容" :span="2">
            <pre class="content-pre">{{ currentAlert.content }}</pre>
          </el-descriptions-item>
          <el-descriptions-item v-if="currentAlert.acknowledged_at" label="确认时间" :span="2">
            {{ formatTime(currentAlert.acknowledged_at) }} by {{ currentAlert.acknowledged_by }}
          </el-descriptions-item>
          <el-descriptions-item v-if="currentAlert.resolved_at" label="解决时间" :span="2">
            {{ formatTime(currentAlert.resolved_at) }} by {{ currentAlert.resolved_by }}
          </el-descriptions-item>
        </el-descriptions>

        <h4 style="margin-top: 20px">通知记录</h4>
        <el-table :data="currentAlert.notification_records" size="small">
          <el-table-column prop="channel_name" label="渠道" />
          <el-table-column prop="channel_type" label="类型" />
          <el-table-column label="状态">
            <template #default="{ row }">
              <el-tag :type="row.status === 'sent' ? 'success' : 'danger'" size="small">
                {{ row.status }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="sent_at" label="发送时间">
            <template #default="{ row }">
              {{ row.sent_at ? formatTime(row.sent_at) : '-' }}
            </template>
          </el-table-column>
        </el-table>

        <h4 style="margin-top: 20px">处理备注</h4>
        <el-table :data="currentAlert.notes" size="small">
          <el-table-column prop="content" label="备注内容" />
          <el-table-column prop="created_by" label="添加人" width="120" />
          <el-table-column prop="created_at" label="时间" width="180">
            <template #default="{ row }">
              {{ formatTime(row.created_at) }}
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-dialog>

    <!-- 规则对话框 -->
    <el-dialog v-model="ruleDialogVisible" :title="isEditRule ? '编辑规则' : '创建规则'" width="800px">
      <el-form ref="ruleFormRef" :model="ruleForm" :rules="ruleRules" label-width="100px">
        <el-form-item label="规则名称" prop="name">
          <el-input v-model="ruleForm.name" placeholder="请输入规则名称" />
        </el-form-item>
        <el-form-item label="规则描述">
          <el-input v-model="ruleForm.description" type="textarea" :rows="2" placeholder="可选" />
        </el-form-item>
        <el-form-item label="告警级别" prop="level">
          <el-select v-model="ruleForm.level" placeholder="请选择">
            <el-option v-for="opt in levelOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="优先级">
          <el-input-number v-model="ruleForm.priority" :min="1" :max="100" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="ruleForm.enabled" />
        </el-form-item>
        <el-form-item label="条件组合">
          <el-radio-group v-model="ruleForm.condition_op">
            <el-radio label="AND">全部满足 (AND)</el-radio>
            <el-radio label="OR">任一满足 (OR)</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="触发条件" prop="conditions">
          <div v-for="(cond, index) in ruleForm.conditions" :key="index" class="condition-row">
            <el-select v-model="cond.field" style="width: 120px">
              <el-option label="数据库类型" value="db_type" />
              <el-option label="事件类型" value="event_type" />
              <el-option label="任务名称" value="task_name" />
              <el-option label="告警内容" value="content" />
            </el-select>
            <el-select v-model="cond.operator" style="width: 120px">
              <el-option v-for="opt in operatorOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
            </el-select>
            <el-input v-model="cond.value" placeholder="值" style="width: 200px" />
            <el-button type="danger" link @click="removeCondition(index)">删除</el-button>
          </div>
          <el-button type="primary" link @click="addCondition">添加条件</el-button>
        </el-form-item>
        <el-form-item label="通知渠道" prop="channels">
          <el-select v-model="ruleForm.channels" multiple placeholder="选择通知渠道" style="width: 100%">
            <el-option v-for="ch in channels" :key="ch.id" :label="ch.name" :value="ch.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="冷却时间">
          <el-input-number v-model="ruleForm.cooldown" :min="0" :max="86400" />
          <span class="form-tip">秒，0 表示不冷却</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="ruleDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveRule">确定</el-button>
      </template>
    </el-dialog>

    <!-- 渠道对话框 -->
    <el-dialog v-model="channelDialogVisible" :title="isEditChannel ? '编辑渠道' : '创建渠道'" width="600px">
      <el-form ref="channelFormRef" :model="channelForm" :rules="channelRules" label-width="100px">
        <el-form-item label="渠道名称" prop="name">
          <el-input v-model="channelForm.name" placeholder="请输入渠道名称" />
        </el-form-item>
        <el-form-item label="渠道类型" prop="type">
          <el-select v-model="channelForm.type" placeholder="选择类型">
            <el-option v-for="opt in channelTypeOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="Webhook URL" prop="config.webhook_url">
          <el-input v-model="channelForm.config.webhook_url" placeholder="请输入 Webhook URL" />
        </el-form-item>
        <el-form-item label="关键词">
          <el-input v-model="channelForm.config.keyword" placeholder="可选，用于验证消息" />
        </el-form-item>
        <el-form-item v-if="channelForm.type === 'dingtalk'" label="签名密钥">
          <el-input v-model="channelForm.config.secret" placeholder="钉钉签名密钥" />
        </el-form-item>
        <el-form-item v-if="channelForm.type === 'email'" label="SMTP 服务器">
          <el-input v-model="channelForm.config.smtp_host" placeholder="smtp.example.com" />
        </el-form-item>
        <el-form-item v-if="channelForm.type === 'email'" label="SMTP 端口">
          <el-input-number v-model="channelForm.config.smtp_port" :min="1" :max="65535" />
        </el-form-item>
        <el-form-item v-if="channelForm.type === 'email'" label="用户名">
          <el-input v-model="channelForm.config.username" placeholder="邮箱地址" />
        </el-form-item>
        <el-form-item v-if="channelForm.type === 'email'" label="密码">
          <el-input v-model="channelForm.config.password" type="password" placeholder="邮箱密码或授权码" show-password />
        </el-form-item>
        <el-form-item v-if="channelForm.type === 'email'" label="发件人">
          <el-input v-model="channelForm.config.from" placeholder="显示的发件人地址" />
        </el-form-item>
        <el-form-item v-if="channelForm.type === 'email'" label="收件人">
          <el-select v-model="channelForm.config.to" multiple placeholder="选择收件人" style="width: 100%">
            <el-option label="ops@example.com" value="ops@example.com" />
            <el-option label="dba@example.com" value="dba@example.com" />
          </el-select>
        </el-form-item>
        <el-form-item label="优先级">
          <el-input-number v-model="channelForm.priority" :min="1" :max="100" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="channelForm.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="channelDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveChannel">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { alertAPI, channelAPI, alertRuleAPI, alertDashboardAPI, AlertLevelOptions, AlertStatusOptions, ChannelTypeOptions, ConditionOperatorOptions } from '@/api/alert'

const activeTab = ref('list')

// 统计数据
const stats = reactive({
  total: 0,
  p0: 0,
  p1: 0,
  resolved: 0
})

// 筛选条件
const filter = reactive({
  level: '',
  status: ''
})

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

// 告警列表
const alerts = ref([])

// 规则列表
const rules = ref([])
const rulesPage = ref(1)
const rulesPageSize = ref(10)
const rulesTotal = ref(0)

// 渠道列表
const channels = ref([])

// 详情对话框
const detailVisible = ref(false)
const currentAlert = ref<any>(null)

// 规则对话框
const ruleDialogVisible = ref(false)
const isEditRule = ref(false)
const ruleFormRef = ref()
const ruleForm = reactive({
  id: 0,
  name: '',
  description: '',
  enabled: true,
  priority: 50,
  level: 'P2',
  condition_op: 'AND',
  conditions: [] as any[],
  channels: [] as number[],
  cooldown: 300
})
const ruleRules = {
  name: [{ required: true, message: '请输入规则名称', trigger: 'blur' }],
  level: [{ required: true, message: '请选择告警级别', trigger: 'change' }],
  conditions: [{ required: true, message: '请添加触发条件', trigger: 'change' }]
}

// 渠道对话框
const channelDialogVisible = ref(false)
const isEditChannel = ref(false)
const channelFormRef = ref()
const channelForm = reactive({
  id: 0,
  name: '',
  type: 'feishu',
  enabled: true,
  priority: 1,
  config: {
    webhook_url: '',
    keyword: '',
    secret: '',
    smtp_host: '',
    smtp_port: 465,
    username: '',
    password: '',
    from: '',
    to: [] as string[]
  }
})
const channelRules = {
  name: [{ required: true, message: '请输入渠道名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择渠道类型', trigger: 'change' }],
  'config.webhook_url': [{ required: true, message: '请输入 Webhook URL', trigger: 'blur' }]
}

// 选项
const levelOptions = AlertLevelOptions
const statusOptions = AlertStatusOptions
const channelTypeOptions = ChannelTypeOptions
const operatorOptions = ConditionOperatorOptions

// 加载统计数据
const loadStats = async () => {
  try {
    const res = await alertDashboardAPI.overview()
    if (res.code === 0) {
      stats.total = res.data.active_alerts.total
      stats.p0 = res.data.active_alerts.P0
      stats.p1 = res.data.active_alerts.P1
    }
  } catch (e) {
    console.error('加载统计失败', e)
  }
}

// 加载告警列表
const loadAlerts = async () => {
  try {
    const res = await alertAPI.list({
      level: filter.level || undefined,
      status: filter.status || undefined,
      page: pagination.page,
      page_size: pagination.pageSize
    })
    if (res.code === 0) {
      alerts.value = res.data.items
      pagination.total = res.data.total
    }
  } catch (e) {
    console.error('加载告警失败', e)
  }
}

// 加载规则列表
const loadRules = async () => {
  try {
    const res = await alertRuleAPI.list({
      page: rulesPage.value,
      page_size: rulesPageSize.value
    })
    if (res.code === 0) {
      rules.value = res.data.items
      rulesTotal.value = res.data.total
    }
  } catch (e) {
    console.error('加载规则失败', e)
  }
}

// 加载渠道列表
const loadChannels = async () => {
  try {
    const res = await channelAPI.list({ page_size: 100 })
    if (res.code === 0) {
      channels.value = res.data.items
    }
  } catch (e) {
    console.error('加载渠道失败', e)
  }
}

// 重置筛选
const resetFilter = () => {
  filter.level = ''
  filter.status = ''
  loadAlerts()
}

// 查看详情
const viewDetail = async (row: any) => {
  try {
    const res = await alertAPI.get(row.id)
    if (res.code === 0) {
      currentAlert.value = res.data
      detailVisible.value = true
    }
  } catch (e) {
    ElMessage.error('加载告警详情失败')
  }
}

// 确认告警
const acknowledge = async (row: any) => {
  try {
    await ElMessageBox.confirm('确认此告警?', '确认告警')
    const res = await alertAPI.acknowledge(row.id)
    if (res.code === 0) {
      ElMessage.success('已确认')
      loadAlerts()
      loadStats()
    }
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error('操作失败')
    }
  }
}

// 解决告警
const resolve = async (row: any) => {
  try {
    await ElMessageBox.confirm('标记此告警为已解决?', '解决告警')
    const res = await alertAPI.resolve(row.id)
    if (res.code === 0) {
      ElMessage.success('已解决')
      loadAlerts()
      loadStats()
    }
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error('操作失败')
    }
  }
}

// 规则操作
const showRuleDialog = (row?: any) => {
  if (row) {
    isEditRule.value = true
    Object.assign(ruleForm, {
      id: row.id,
      name: row.name,
      description: row.description,
      enabled: row.enabled,
      priority: row.priority,
      level: row.level,
      condition_op: row.condition_op,
      conditions: [...(row.conditions || [])],
      channels: [...(row.channels || [])],
      cooldown: row.cooldown
    })
  } else {
    isEditRule.value = false
    Object.assign(ruleForm, {
      id: 0,
      name: '',
      description: '',
      enabled: true,
      priority: 50,
      level: 'P2',
      condition_op: 'AND',
      conditions: [],
      channels: [],
      cooldown: 300
    })
  }
  ruleDialogVisible.value = true
}

const addCondition = () => {
  ruleForm.conditions.push({ field: 'db_type', operator: 'eq', value: '' })
}

const removeCondition = (index: number) => {
  ruleForm.conditions.splice(index, 1)
}

const saveRule = async () => {
  if (!ruleFormRef.value) return
  await ruleFormRef.value.validate(async (valid) => {
    if (!valid) return
    try {
      const data = { ...ruleForm }
      const res = isEditRule.value
        ? await alertRuleAPI.update(ruleForm.id, data)
        : await alertRuleAPI.create(data)
      if (res.code === 0) {
        ElMessage.success('保存成功')
        ruleDialogVisible.value = false
        loadRules()
      }
    } catch (e) {
      ElMessage.error('保存失败')
    }
  })
}

const toggleRule = async (row: any) => {
  try {
    await alertRuleAPI.update(row.id, { enabled: row.enabled })
    ElMessage.success('更新成功')
  } catch (e) {
    ElMessage.error('更新失败')
    row.enabled = !row.enabled
  }
}

const copyRule = async (row: any) => {
  try {
    const res = await alertRuleAPI.copy(row.id)
    if (res.code === 0) {
      ElMessage.success('复制成功')
      loadRules()
    }
  } catch (e) {
    ElMessage.error('复制失败')
  }
}

const deleteRule = async (row: any) => {
  try {
    await ElMessageBox.confirm('确定删除此规则?', '删除规则', { type: 'warning' })
    const res = await alertRuleAPI.delete(row.id)
    if (res.code === 0) {
      ElMessage.success('删除成功')
      loadRules()
    }
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

// 渠道操作
const showChannelDialog = (row?: any) => {
  if (row) {
    isEditChannel.value = true
    channelForm.id = row.id
    channelForm.name = row.name
    channelForm.type = row.type
    channelForm.enabled = row.enabled
    channelForm.priority = row.priority
    // config 需要解析
    if (row.config) {
      Object.assign(channelForm.config, row.config)
    }
  } else {
    isEditChannel.value = false
    Object.assign(channelForm, {
      id: 0,
      name: '',
      type: 'feishu',
      enabled: true,
      priority: 1,
      config: {
        webhook_url: '',
        keyword: '',
        secret: '',
        smtp_host: '',
        smtp_port: 465,
        username: '',
        password: '',
        from: '',
        to: []
      }
    })
  }
  channelDialogVisible.value = true
}

const saveChannel = async () => {
  if (!channelFormRef.value) return
  await channelFormRef.value.validate(async (valid) => {
    if (!valid) return
    try {
      const data = {
        name: channelForm.name,
        type: channelForm.type,
        enabled: channelForm.enabled,
        priority: channelForm.priority,
        config: { ...channelForm.config }
      }
      const res = isEditChannel.value
        ? await channelAPI.update(channelForm.id, data)
        : await channelAPI.create(data)
      if (res.code === 0) {
        ElMessage.success('保存成功')
        channelDialogVisible.value = false
        loadChannels()
      }
    } catch (e) {
      ElMessage.error('保存失败')
    }
  })
}

const toggleChannel = async (row: any) => {
  try {
    await channelAPI.update(row.id, { enabled: row.enabled })
    ElMessage.success('更新成功')
  } catch (e) {
    ElMessage.error('更新失败')
    row.enabled = !row.enabled
  }
}

const testChannel = async (row: any) => {
  try {
    ElMessage.info('正在发送测试消息...')
    const res = await channelAPI.test(row.id)
    if (res.code === 0) {
      ElMessage.success('测试消息发送成功')
    } else {
      ElMessage.error('发送失败')
    }
  } catch (e) {
    ElMessage.error('发送失败')
  }
}

const deleteChannel = async (row: any) => {
  try {
    await ElMessageBox.confirm('确定删除此渠道?', '删除渠道', { type: 'warning' })
    const res = await channelAPI.delete(row.id)
    if (res.code === 0) {
      ElMessage.success('删除成功')
      loadChannels()
    }
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

// 工具函数
const getLevelType = (level: string) => {
  const map: any = { P0: 'danger', P1: 'warning', P2: 'primary', P3: 'info' }
  return map[level] || 'info'
}

const getStatusType = (status: string) => {
  const map: any = { active: 'danger', acknowledged: 'warning', resolved: 'success', escalated: 'danger' }
  return map[status] || 'info'
}

const getStatusLabel = (status: string) => {
  const opt = statusOptions.find((s: any) => s.value === status)
  return opt?.label || status
}

const getHealthType = (status: string) => {
  const map: any = { healthy: 'success', unhealthy: 'danger', unknown: 'info' }
  return map[status] || 'info'
}

const getHealthLabel = (status: string) => {
  const map: any = { healthy: '健康', unhealthy: '异常', unknown: '未知' }
  return map[status] || status
}

const getChannelTypeLabel = (type: string) => {
  const opt = channelTypeOptions.find((c: any) => c.value === type)
  return opt?.label || type
}

const formatTime = (time: string) => {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

onMounted(() => {
  loadStats()
  loadAlerts()
  loadRules()
  loadChannels()
})
</script>

<style scoped>
.alerts-container {
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

.alert-tabs {
  background: #fff;
  padding: 20px;
  border-radius: 4px;
}

.filter-form {
  margin-bottom: 16px;
}

.toolbar {
  margin-bottom: 16px;
}

.pagination {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

.condition-row {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
  align-items: center;
}

.form-tip {
  margin-left: 8px;
  color: #909399;
  font-size: 12px;
}

.content-pre {
  background: #f5f7fa;
  padding: 12px;
  border-radius: 4px;
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
}
</style>
