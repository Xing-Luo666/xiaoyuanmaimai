<template>
  <el-breadcrumb class="breadcrumb" separator="/">
    <transition-group name="breadcrumb">
      <el-breadcrumb-item v-for="(item, idx) in levelList" :key="item.path + idx">
        <span v-if="idx === levelList.length - 1" class="no-redirect">{{ item.meta?.title }}</span>
        <a v-else @click.prevent="handleLink(item)">{{ item.meta?.title }}</a>
      </el-breadcrumb-item>
    </transition-group>
  </el-breadcrumb>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const levelList = ref([])

const getBreadcrumb = () => {
  const matched = route.matched.filter(item => item.meta && item.meta.title)
  levelList.value = matched
}

watch(() => route.path, getBreadcrumb, { immediate: true })

const handleLink = (item) => {
  router.push(item.path)
}
</script>

<style lang="scss" scoped>
.breadcrumb {
  display: inline-flex;
  align-items: center;
  font-size: 14px;
  .no-redirect { color: #97a8be; cursor: text; }
  a { color: #606266; font-weight: normal; cursor: pointer; }
  a:hover { color: #409eff; }
}
.breadcrumb-enter-active, .breadcrumb-leave-active { transition: all 0.3s; }
.breadcrumb-enter-from, .breadcrumb-leave-to { opacity: 0; transform: translateX(20px); }
</style>
