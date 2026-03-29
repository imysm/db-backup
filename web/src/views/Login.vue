<template>
  <div class="login-container">
    <el-card class="login-card" shadow="hover">
      <div class="login-header">
        <el-icon :size="40" color="#409eff"><Coin /></el-icon>
        <h2>数据库备份系统</h2>
        <p class="subtitle">DB Backup System</p>
      </div>
      <el-form :model="form" @submit.prevent="handleLogin">
        <el-form-item>
          <el-input
            v-model="form.apiKey"
            placeholder="请输入 API Key"
            size="large"
            :prefix-icon="Key"
            show-password
            @keyup.enter="handleLogin"
          />
        </el-form-item>
        <el-form-item>
          <el-button
            type="primary"
            size="large"
            style="width: 100%"
            :loading="loading"
            @click="handleLogin"
          >
            登 录
          </el-button>
        </el-form-item>
      </el-form>
      <div class="login-footer">
        <span>API Key 请联系管理员获取</span>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Key, Coin } from '@element-plus/icons-vue'
import { loginAPI } from '@/api'

const router = useRouter()
const loading = ref(false)
const form = ref({ apiKey: '' })

const handleLogin = async () => {
  if (!form.value.apiKey.trim()) {
    ElMessage.warning('请输入 API Key')
    return
  }
  loading.value = true
  try {
    await loginAPI.verify(form.value.apiKey.trim())
    localStorage.setItem('api_key', form.value.apiKey.trim())
    ElMessage.success('登录成功')
    router.push('/dashboard')
  } catch (e: any) {
    if (e.response?.status === 401) {
      ElMessage.error('API Key 无效')
    } else {
      ElMessage.error('登录失败，请检查网络连接')
    }
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  width: 100%;
  height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%);
}

.login-card {
  width: 420px;
  border-radius: 12px;
}

.login-card :deep(.el-card__body) {
  padding: 30px 40px 20px;
}

.login-header {
  text-align: center;
  margin-bottom: 30px;
}

.login-header h2 {
  margin: 12px 0 4px;
  color: #303133;
  font-size: 22px;
}

.subtitle {
  color: #909399;
  font-size: 13px;
  margin: 0;
}

.login-footer {
  text-align: center;
  color: #c0c4cc;
  font-size: 12px;
  margin-top: 10px;
}
</style>
