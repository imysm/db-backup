<template>
  <el-container class="layout-container">
    <!-- 侧边栏 -->
    <el-aside :width="isCollapse ? '64px' : '220px'" class="aside-container">
      <div class="logo-area">
        <el-icon :size="24" color="#409eff"><Coin /></el-icon>
        <span v-show="!isCollapse" class="logo-text">DB Backup</span>
      </div>
      <el-menu
        :default-active="route.path"
        router
        :collapse="isCollapse"
        background-color="#1a1a2e"
        text-color="#bfcbd9"
        active-text-color="#409eff"
        class="sidebar-menu"
      >
        <el-menu-item index="/dashboard">
          <el-icon><DataAnalysis /></el-icon>
          <template #title>仪表盘</template>
        </el-menu-item>
        <el-menu-item index="/jobs">
          <el-icon><FolderOpened /></el-icon>
          <template #title>备份任务</template>
        </el-menu-item>
        <el-menu-item index="/jobs/new">
          <el-icon><Plus /></el-icon>
          <template #title>创建任务</template>
        </el-menu-item>
        <el-menu-item index="/records">
          <el-icon><Document /></el-icon>
          <template #title>备份记录</template>
        </el-menu-item>
        <el-menu-item index="/verify">
          <el-icon><CircleCheck /></el-icon>
          <template #title>备份验证</template>
        </el-menu-item>
        <el-menu-item index="/restore">
          <el-icon><RefreshRight /></el-icon>
          <template #title>恢复管理</template>
        </el-menu-item>
        <el-sub-menu index="/storage">
          <template #title>
            <el-icon><Box /></el-icon>
            <span>存储管理</span>
          </template>
          <el-menu-item index="/storage">存储概览</el-menu-item>
          <el-menu-item index="/storage-forecast">容量预测</el-menu-item>
        </el-sub-menu>
        <el-sub-menu index="/system">
          <template #title>
            <el-icon><Tools /></el-icon>
            <span>系统工具</span>
          </template>
          <el-menu-item index="/health">系统体检</el-menu-item>
          <el-menu-item index="/backup-impact">影响分析</el-menu-item>
          <el-menu-item index="/alert-rules">告警规则</el-menu-item>
        </el-sub-menu>
        <el-menu-item index="/alerts">
          <el-icon><Warning /></el-icon>
          <template #title>告警中心</template>
        </el-menu-item>
        <el-menu-item index="/settings">
          <el-icon><Setting /></el-icon>
          <template #title>系统设置</template>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <el-container>
      <!-- 顶部栏 -->
      <el-header class="header-container">
        <div class="header-left">
          <el-icon class="collapse-btn" @click="isCollapse = !isCollapse" :size="20">
            <Fold v-if="!isCollapse" />
            <Expand v-else />
          </el-icon>
          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{ path: '/dashboard' }">首页</el-breadcrumb-item>
            <el-breadcrumb-item v-if="route.meta.title">{{ route.meta.title }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <div class="header-right">
          <el-button type="danger" size="small" text @click="handleLogout">
            <el-icon><SwitchButton /></el-icon>
            退出登录
          </el-button>
        </div>
      </el-header>

      <!-- 主内容区 -->
      <el-main class="main-container">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import {
  DataAnalysis, FolderOpened, Document, RefreshRight,
  Setting, CircleCheck, Coin, Fold, Expand, SwitchButton,
  Plus, Box, Tools, Warning
} from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const isCollapse = ref(false)

const handleLogout = () => {
  ElMessageBox.confirm('确定要退出登录吗？', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => {
    localStorage.removeItem('api_key')
    router.push('/login')
  }).catch(() => {})
}
</script>

<style scoped>
.layout-container {
  height: 100vh;
}

.aside-container {
  background-color: #1a1a2e;
  transition: width 0.3s;
  overflow: hidden;
}

.logo-area {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.logo-text {
  color: #fff;
  font-size: 18px;
  font-weight: bold;
  white-space: nowrap;
}

.sidebar-menu {
  border-right: none;
}

.sidebar-menu:not(.el-menu--collapse) {
  width: 220px;
}

.header-container {
  background-color: #fff;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.08);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  z-index: 10;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.collapse-btn {
  cursor: pointer;
  color: #606266;
  transition: color 0.2s;
}

.collapse-btn:hover {
  color: #409eff;
}

.main-container {
  background-color: #f0f2f5;
  padding: 20px;
}

@media (max-width: 768px) {
  .aside-container {
    position: fixed;
    z-index: 100;
    height: 100vh;
  }
}
</style>
