<template>
  <div class="tags-view">
    <el-scrollbar>
      <div class="tags-wrap">
        <router-link
          v-for="tag in visitedTags"
          :key="tag.path"
          :to="tag.path"
          class="tag-item"
          :class="{ active: isActive(tag) }"
          @contextmenu.prevent="openMenu($event, tag)"
        >
          {{ tag.title }}
          <el-icon v-if="!tag.affix" class="close-icon" @click.prevent.stop="closeTag(tag)"><Close /></el-icon>
        </router-link>
      </div>
    </el-scrollbar>
    <ul v-show="menu.visible" class="contextmenu" :style="{ top: menu.top + 'px', left: menu.left + 'px' }">
      <li @click="refreshSelectedTag(menu.tag)">刷新</li>
      <li v-if="!menu.tag?.affix" @click="closeTag(menu.tag)">关闭</li>
      <li @click="closeOthers(menu.tag)">关闭其他</li>
      <li @click="closeAll()">关闭全部</li>
    </ul>
  </div>
</template>

<script setup>
import { ref, watch, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()

const visitedTags = ref([])
const menu = ref({ visible: false, top: 0, left: 0, tag: null })

const isActive = (tag) => tag.path === route.path

const addTag = () => {
  if (route.name && route.meta?.title) {
    const exists = visitedTags.value.find(t => t.path === route.path)
    if (!exists) {
      visitedTags.value.push({
        path: route.path,
        title: route.meta.title,
        affix: route.path === '/dashboard'
      })
    }
  }
}

const closeTag = (tag) => {
  if (!tag) return
  const idx = visitedTags.value.findIndex(t => t.path === tag.path)
  if (idx > -1) {
    visitedTags.value.splice(idx, 1)
    if (isActive(tag)) {
      const next = visitedTags.value[idx] || visitedTags.value[idx - 1]
      router.push(next ? next.path : '/dashboard')
    }
  }
  menu.value.visible = false
}

const closeOthers = (tag) => {
  visitedTags.value = visitedTags.value.filter(t => t.affix || t.path === tag.path)
  if (!visitedTags.value.find(t => t.path === tag.path)) {
    visitedTags.value.push({ path: tag.path, title: tag.title })
  }
  router.push(tag.path)
  menu.value.visible = false
}

const closeAll = () => {
  visitedTags.value = visitedTags.value.filter(t => t.affix)
  router.push('/dashboard')
  menu.value.visible = false
}

const refreshSelectedTag = (tag) => {
  router.replace({ path: '/redirect' + tag.path })
  menu.value.visible = false
}

const openMenu = (e, tag) => {
  menu.value = {
    visible: true,
    top: e.clientY,
    left: e.clientX,
    tag
  }
}

const closeMenu = () => { menu.value.visible = false }

onMounted(() => {
  addTag()
  document.addEventListener('click', closeMenu)
})
onUnmounted(() => {
  document.removeEventListener('click', closeMenu)
})
watch(() => route.path, addTag)
</script>

<style lang="scss" scoped>
@use '@/styles/variables.scss' as *;

.tags-view {
  height: $tagsViewHeight;
  background: #fff;
  border-bottom: 1px solid #ebeef5;
  display: flex;
  align-items: center;
  padding: 0 8px;
  position: relative;
}

.tags-wrap { display: flex; align-items: center; gap: 4px; white-space: nowrap; }

.tag-item {
  display: inline-flex;
  align-items: center;
  height: 26px;
  padding: 0 8px;
  font-size: 12px;
  border: 1px solid #d9d9d9;
  border-radius: 3px;
  color: #495060;
  background: #fff;
  cursor: pointer;
  text-decoration: none;
  &:hover { color: $primaryColor; }
  &.active {
    color: #fff;
    background: $primaryColor;
    border-color: $primaryColor;
  }
  .close-icon {
    margin-left: 4px;
    font-size: 12px;
    border-radius: 50%;
    padding: 2px;
    &:hover { background: rgba(255,255,255,0.3); }
  }
}

.contextmenu {
  position: fixed;
  list-style: none;
  margin: 0;
  padding: 4px 0;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  box-shadow: 0 2px 12px rgba(0,0,0,0.1);
  z-index: 3000;
  min-width: 100px;
  li {
    padding: 6px 16px;
    font-size: 13px;
    cursor: pointer;
    &:hover { background: #f5f7fa; color: $primaryColor; }
  }
}
</style>
