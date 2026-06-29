<template>
  <template v-if="!item.hidden">
    <!-- 单一子菜单（且无更多子级） -->
    <template v-if="hasOneShowingChild(item.children, item) && (!onlyChild.children || !onlyChild.children.length) && !onlyChild.alwaysShow">
      <el-menu-item v-if="onlyChild" :index="resolvePath(onlyChild.path)">
        <el-icon v-if="onlyChild.meta?.icon"><component :is="onlyChild.meta.icon" /></el-icon>
        <template #title>{{ onlyChild.meta?.title }}</template>
      </el-menu-item>
    </template>
    <!-- 多级菜单 -->
    <el-sub-menu v-else :index="resolvePath(item.path)">
      <template #title>
        <el-icon v-if="item.meta?.icon"><component :is="item.meta.icon" /></el-icon>
        <span>{{ item.meta?.title }}</span>
      </template>
      <sidebar-item
        v-for="child in item.children"
        :key="child.path"
        :item="child"
        :base-path="resolvePath(child.path)"
      />
    </el-sub-menu>
  </template>
</template>

<script setup>
import { ref } from 'vue'
import path from 'path'

const props = defineProps({
  item: { type: Object, required: true },
  basePath: { type: String, default: '' }
})

const onlyChild = ref(null)

const hasOneShowingChild = (children = [], parent) => {
  const showingChildren = children.filter(item => {
    if (item.hidden) return false
    onlyChild.value = item
    return true
  })
  if (showingChildren.length === 1) return true
  if (showingChildren.length === 0) {
    onlyChild.value = { ...parent, path: '', noShowingChildren: true }
    return true
  }
  return false
}

const resolvePath = (routePath) => {
  if (/^(https?:|mailto:|tel:)/.test(routePath)) return routePath
  if (/^(https?:|mailto:|tel:)/.test(props.basePath)) return props.basePath
  // 处理 windows 路径
  const base = props.basePath.replace(/\\/g, '/')
  const p = routePath || ''
  if (base.endsWith('/')) return (base + p).replace(/\/+/g, '/')
  if (p.startsWith('/')) return base + p
  return base + '/' + p
}
</script>
