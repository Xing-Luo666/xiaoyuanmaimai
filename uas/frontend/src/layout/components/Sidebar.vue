<template>
  <div class="sidebar">
    <div class="logo">
      <img src="/logo.svg" alt="logo" v-if="!isCollapse" />
      <span v-if="!isCollapse" class="title">统一身份认证</span>
      <img src="/logo.svg" alt="logo" v-else class="logo-mini" />
    </div>
    <el-scrollbar>
      <el-menu
        :default-active="activeMenu"
        :collapse="isCollapse"
        :unique-opened="true"
        :collapse-transition="false"
        mode="vertical"
        background-color="#304156"
        text-color="#bfcbd9"
        active-text-color="#409eff"
        router
      >
        <template v-for="route in menuRoutes" :key="route.path">
          <sidebar-item :item="route" :base-path="route.path" />
        </template>
      </el-menu>
    </el-scrollbar>
  </div>
</template>

<script setup>
import { computed, inject } from 'vue'
import { useRoute } from 'vue-router'
import { routes as allRoutes } from '@/router'
import SidebarItem from './SidebarItem.vue'

const route = useRoute()
const isCollapse = inject('isCollapse')

const menuRoutes = computed(() => allRoutes.filter(r => !r.hidden && r.children && r.children.length))

const activeMenu = computed(() => route.path)
</script>

<style lang="scss" scoped>
@use '@/styles/variables.scss' as *;

.sidebar {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: $sidebarBg;
}

.logo {
  height: 50px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: #fff;
  background: #2b3a4d;
  overflow: hidden;
  white-space: nowrap;
  img { width: 28px; height: 28px; }
  .logo-mini { width: 28px; height: 28px; }
  .title { font-size: 16px; font-weight: 600; }
}

:deep(.el-menu) { border-right: none; }
:deep(.el-scrollbar) { flex: 1; }
</style>
