<script setup lang="ts">
import { CollectionTag, Document, Grid, HomeFilled, Ticket } from '@element-plus/icons-vue'
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const collapsed = defineModel<boolean>('collapsed', { default: false })

const activeMenu = computed(() => {
  if (route.path.startsWith('/quotes')) return '/quotes'
  if (route.path.startsWith('/templates/layout')) return '/templates/layout'
  if (route.path.startsWith('/templates/business')) return '/templates/business'
  if (route.path.startsWith('/templates')) return '/templates/layout'
  return route.path
})

const openMenus = computed(() => (route.path.startsWith('/templates') ? ['templates'] : []))

const logoText = computed(() => (collapsed.value ? 'QC' : '报价中心'))

function navigate(path: string) {
  router.push(path)
}
</script>

<template>
  <aside class="sidebar" :class="{ collapsed }">
    <div class="logo">{{ logoText }}</div>
    <el-menu
      :default-active="activeMenu"
      :default-openeds="openMenus"
      :collapse="collapsed"
      background-color="#001529"
      text-color="#ffffffa6"
      active-text-color="#fff"
    >
      <el-menu-item index="/dashboard" @click="navigate('/dashboard')">
        <el-icon><HomeFilled /></el-icon>
        <span>工作台</span>
      </el-menu-item>
      <el-menu-item index="/quotes" @click="navigate('/quotes')">
        <el-icon><Document /></el-icon>
        <span>报价单</span>
      </el-menu-item>
      <el-sub-menu index="templates">
        <template #title>
          <el-icon><Ticket /></el-icon>
          <span>报价模板</span>
        </template>
        <el-menu-item index="/templates/layout" @click="navigate('/templates/layout')">
          <el-icon><Grid /></el-icon>
          <span>版式模板</span>
        </el-menu-item>
        <el-menu-item index="/templates/business" @click="navigate('/templates/business')">
          <el-icon><CollectionTag /></el-icon>
          <span>业务模板</span>
        </el-menu-item>
      </el-sub-menu>
    </el-menu>
  </aside>
</template>

<style scoped>
.sidebar {
  width: 220px;
  background: #001529;
  transition: width 0.2s;
  flex-shrink: 0;
}
.sidebar.collapsed {
  width: 64px;
}
.logo {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-weight: 600;
  font-size: 16px;
  border-bottom: 1px solid #ffffff14;
}
.sidebar :deep(.el-menu) {
  border-right: none;
}
</style>
