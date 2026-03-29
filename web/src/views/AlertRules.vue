<template>
  <div class="alert-rules-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>告警规则</span>
          <el-button type="primary" @click="handleCreate">新建规则</el-button>
        </div>
      </template>

      <!-- 筛选 -->
      <div class="filter-section">
        <el-select v-model="levelFilter" placeholder="级别筛选" clearable style="width: 120px; margin-right: 10px" @change="fetchRules">
          <el-option label="P0" value="P0" />
          <el-option label="P1" value="P1" />
          <el-option label="P2" value="P2" />
          <el-option label="P3" value="P3" />
        </el-select>
        <el-select v-model="enabledFilter" placeholder="状态筛选" clearable style="width: 120px" @change="fetchRules">
          <el-option label="已启用" :value="true" />
          <el-option label="已禁用" :value="false" />
        </el-select>
      </div>

      <!-- 规则列表 -->
      <el-table :data="rules" v-loading="loading" style="width: 100%">
        <el-table-column prop="name" label="规则名称" min-width="150" />
        <el-table-column prop="level" label="级别" width="80">
          <template #default="{ row }">
            <el-tag :type="getLevelType(row.level)" size="small">{{ row.level }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="priority" label="优先级" width="80" />
        <el-table-column prop="enabled" label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'" size="small">
              {{ row.enabled ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="condition_op" label="条件" width="80">
          <template #default="{ row }">
            <span>{{ row.condition_op }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="cooldown" label="冷却时间" width="100">
          <template #default="{ row }">
            {{ row.cooldown }}s
          </template>
        </el-table-column>
        <el-table-column prop="matched_count" label="触发次数" width="100" />
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" link @click="handleEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" link @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        @current-change="fetchRules"
        layout="total, prev, pager, next"
        style="margin-top: 20px; justify-content: flex-end"
      />
    </el-card>

    <!-- 创建/编辑规则对话框 -->
    <el-dialog 
      v-model="dialogVisible" 
      :title="isEdit ? '编辑告警规则' : '新建告警规则'" 
      width="700px"
      @close="resetForm"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="规则名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入规则名称" />
        </el-form-item>

        <el-form-item label="描述" prop="description">
          <el-input v-model="form.description" type="textarea" :rows="2" placeholder="可选描述" />
        </el-form-item>

        <el-row :gutter="20">
          <el-col :span="8">
            <el-form-item label="级别" prop="level">
              <el-select v-model="form.level" placeholder="选择级别">
                <el-option label="P0 - 紧急" value="P0" />
                <el-option label="P1 - 重要" value="P1" />
                <el-option label="P2 - 一般" value="P2" />
                <el-option label="P3 - 提示" value="P3" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="优先级" prop="priority">
              <el-input-number v-model="form.priority" :min="1" :max="100" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="启用">
              <el-switch v-model="form.enabled" />
            </el-form-item>
          </el-col>
        </el-row>

        <!-- 触发条件 -->
        <el-divider>触发条件</el-divider>
        
        <el-form-item label="条件组合">
          <el-radio-group v-model="form.condition_op">
            <el-radio label="AND">满足所有条件</el-radio>
            <el-radio label="OR">满足任一条件</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="条件列表">
          <div v-for="(cond, index) in form.conditions" :key="index" class="condition-row">
            <el-select v-model="cond.field" placeholder="字段" style="width: 120px">
              <el-option label="status" value="status" />
              <el-option label="duration" value="duration" />
              <el-option label="file_size" value="file_size" />
              <el-option label="job_name" value="job_name" />
              <el-option label="database_type" value="database_type" />
              <el-option label="error_message" value="error_message" />
            </el-select>
            <el-select v-model="cond.operator" placeholder="操作符" style="width: 100px">
              <el-option label="等于" value="eq" />
              <el-option label="不等于" value="ne" />
              <el-option label="大于" value="gt" />
              <el-option label="大于等于" value="gte" />
              <el-option label="小于" value="lt" />
              <el-option label="小于等于" value="lte" />
              <el-option label="包含" value="contains" />
              <el-option label="正则" value="regex" />
            </el-select>
            <el-input v-model="cond.value" placeholder="值" style="width: 150px" />
            <el-button type="danger" link @click="removeCondition(index)">删除</el-button>
          </div>
          <el-button type="primary" link @click="addCondition">+ 添加条件</el-button>
        </el-form-item>

        <!-- 通知渠道 -->
        <el-divider>通知渠道</el-divider>

        <el-form-item label="冷却时间">
          <el-input-number v-model="form.cooldown" :min="0" :max="86400" /> 秒
        </el-form-item>

        <el-form-item label="通知渠道">
          <el-checkbox-group v-model="form.channels">
            <el-checkbox v-for="ch in channels" :key="ch.id" :label="ch.id">
              {{ ch.name }} ({{ ch.type }})
            </el-checkbox>
          </el-checkbox-group>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm" :loading="submitLoading">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '@/utils/api'

const loading = ref(false)
const rules = ref<any[]>([])
const channels = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const levelFilter = ref('')
const enabledFilter = ref<boolean | null>(null)

const dialogVisible = ref(false)
const isEdit = ref(false)
const submitLoading = ref(false)
const formRef = ref()

const form = reactive({
  id: null as number | null,
  name: '',
  description: '',
  enabled: true,
  priority: 50,
  level: 'P2',
  condition_op: 'AND',
  conditions: [] as { field: string; operator: string; value: string }[],
  channels: [] as number[],
  cooldown: 300
})

const formRules = {
  name: [{ required: true, message: '请输入规则名称', trigger: 'blur' }],
  level: [{ required: true, message: '请选择级别', trigger: 'change' }]
}

const getLevelType = (level: string) => {
  const map: Record<string, string> = {
    P0: 'danger',
    P1: 'warning',
    P2: 'info',
    P3: 'success'
  }
  return map[level] || 'info'
}

const fetchRules = async () => {
  loading.value = true
  try {
    const params: any = { page: page.value, page_size: pageSize.value }
    if (levelFilter.value) params.level = levelFilter.value
    if (enabledFilter.value !== null) params.enabled = enabledFilter.value

    const res = await api.get('/alert/rules', { params })
    rules.value = res.data || []
    total.value = res.total || 0
  } catch (e: any) {
    ElMessage.error(e.message || '获取规则列表失败')
  } finally {
    loading.value = false
  }
}

const fetchChannels = async () => {
  try {
    const res = await api.get('/alert/channels')
    channels.value = res.data || []
  } catch (e: any) {
    console.error('获取渠道失败', e)
  }
}

const handleCreate = () => {
  isEdit.value = false
  dialogVisible.value = true
}

const handleEdit = (row: any) => {
  isEdit.value = true
  form.id = row.id
  form.name = row.name
  form.description = row.description || ''
  form.enabled = row.enabled
  form.priority = row.priority
  form.level = row.level
  form.condition_op = row.condition_op
  form.conditions = row.conditions || []
  form.channels = row.channels || []
  form.cooldown = row.cooldown
  dialogVisible.value = true
}

const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除规则 "${row.name}" 吗？`,
      '删除规则',
      { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }
    )
    await api.delete(`/alert/rules/${row.id}`)
    ElMessage.success('删除成功')
    fetchRules()
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e.message || '删除失败')
    }
  }
}

const addCondition = () => {
  form.conditions.push({ field: 'status', operator: 'eq', value: '' })
}

const removeCondition = (index: number) => {
  form.conditions.splice(index, 1)
}

const resetForm = () => {
  form.id = null
  form.name = ''
  form.description = ''
  form.enabled = true
  form.priority = 50
  form.level = 'P2'
  form.condition_op = 'AND'
  form.conditions = []
  form.channels = []
  form.cooldown = 300
}

const submitForm = async () => {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  submitLoading.value = true
  try {
    if (isEdit.value) {
      await api.put(`/alert/rules/${form.id}`, form)
      ElMessage.success('更新成功')
    } else {
      await api.post('/alert/rules', form)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    fetchRules()
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  } finally {
    submitLoading.value = false
  }
}

onMounted(() => {
  fetchRules()
  fetchChannels()
})
</script>

<style scoped>
.alert-rules-container {
  padding: 0;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.filter-section {
  margin-bottom: 16px;
  display: flex;
}

.condition-row {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 8px;
}
</style>
