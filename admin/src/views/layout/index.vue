<!--
  管理后台主布局组件
  
  设计系统：design-system/echochat/MASTER.md（管理端适配）
  
  结构：
  - 左侧固定侧边栏（可折叠）：logo + 导航菜单
  - 右侧主区域：顶部栏（面包屑 + 管理员信息）+ 内容区
  
  使用 Element Plus Container 布局组件
-->
<template>
  <el-container class="layout-container">
    <!-- 左侧边栏 -->
    <el-aside :width="isCollapse ? '64px' : '220px'" class="layout-aside">
      <div class="logo-section">
        <div class="logo-icon">E</div>
        <span v-show="!isCollapse" class="logo-text">EchoChat</span>
      </div>

      <el-menu
        :default-active="activeMenu"
        :collapse="isCollapse"
        router
        background-color="#1E293B"
        text-color="#94A3B8"
        active-text-color="#FFFFFF"
        class="aside-menu"
      >
        <el-menu-item index="/dashboard">
          <el-icon><DataAnalysis /></el-icon>
          <template #title>仪表盘</template>
        </el-menu-item>

        <el-sub-menu index="user-manage">
          <template #title>
            <el-icon><User /></el-icon>
            <span>用户管理</span>
          </template>
          <el-menu-item index="/user/list">用户列表</el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="contact-manage">
          <template #title>
            <el-icon><Connection /></el-icon>
            <span>联系人管理</span>
          </template>
          <el-menu-item index="/contact/list">好友关系</el-menu-item>
        </el-sub-menu>

        <el-menu-item index="/monitor/online">
          <el-icon><Monitor /></el-icon>
          <template #title>在线监控</template>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <!-- 右侧主区域 -->
    <el-container class="layout-main">
      <!-- 顶部栏 -->
      <el-header class="layout-header">
        <div class="header-left">
          <el-icon class="collapse-btn" @click="isCollapse = !isCollapse">
            <Expand v-if="isCollapse" />
            <Fold v-else />
          </el-icon>
        </div>

        <div class="header-right">
          <el-dropdown trigger="click" @command="handleCommand">
            <span class="admin-info">
              <el-avatar :size="32" class="admin-avatar">
                {{ avatarLetter }}
              </el-avatar>
              <span class="admin-name">{{ username }}</span>
              <el-icon><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="logout">退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <!-- 内容区 -->
      <el-main class="layout-content">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
/**
 * 管理后台布局逻辑
 *
 * 侧边栏折叠状态
 * 管理员信息从 Store 获取
 * 退出登录后清除状态并跳转
 */
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/store/user'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

/** 侧边栏折叠状态 */
const isCollapse = ref(false)

/** 当前激活的菜单项（跟随路由） */
const activeMenu = computed(() => route.path)

/** 管理员用户名 */
const username = computed(() => userStore.username || '管理员')

/** 头像占位字母 */
const avatarLetter = computed(() => {
  const name = userStore.userInfo?.nickname || userStore.username || '?'
  return name.charAt(0).toUpperCase()
})

/**
 * 下拉菜单命令处理
 * @param {string} command
 */
const handleCommand = async (command) => {
  if (command === 'logout') {
    await userStore.logout()
    router.push('/login')
  }
}
</script>

<style scoped>
/*
 * 管理端布局样式
 * 左侧导航：深色 #1E293B
 * 顶部栏：白色 + 底部阴影
 * 内容区：灰色背景 #F0F2F5
 */
.layout-container {
  min-height: 100vh;
}

/* ---- 侧边栏 ---- */
.layout-aside {
  background-color: #1E293B;
  transition: width 0.3s ease;
  overflow: hidden;
}

.logo-section {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 60px;
  padding: 0 16px;
  border-bottom: 1px solid #334155;
}

.logo-icon {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  background-color: #2563EB;
  color: #FFFFFF;
  font-size: 18px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.logo-text {
  font-size: 16px;
  font-weight: 600;
  color: #FFFFFF;
  margin-left: 10px;
  white-space: nowrap;
}

.aside-menu {
  border-right: none !important;
}

/* ---- 顶部栏 ---- */
.layout-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background-color: #FFFFFF;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
  padding: 0 20px;
  height: 60px;
}

.header-left {
  display: flex;
  align-items: center;
}

.collapse-btn {
  font-size: 20px;
  cursor: pointer;
  color: #64748B;
  transition: color 200ms;
}

.collapse-btn:hover {
  color: #2563EB;
}

.header-right {
  display: flex;
  align-items: center;
}

.admin-info {
  display: flex;
  align-items: center;
  cursor: pointer;
  gap: 8px;
}

.admin-avatar {
  background-color: #2563EB;
  color: #FFFFFF;
  font-weight: 600;
}

.admin-name {
  font-size: 14px;
  color: #475569;
}

/* ---- 内容区 ---- */
.layout-content {
  background-color: #F0F2F5;
  padding: 20px;
  min-height: calc(100vh - 60px);
}
</style>
