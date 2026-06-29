<template>
  <div>
    <el-card class="search-card" shadow="never">
      <el-form :inline="true" :model="query" class="search-form">
        <el-form-item label="操作人"><el-input v-model="query.operName" placeholder="请输入" clearable @keyup.enter="loadList" /></el-form-item>
        <el-form-item label="操作模块"><el-select v-model="query.module" placeholder="全部" clearable style="width:160px"><el-option v-for="m in modules" :key="m.value" :label="m.label" :value="m.value" /></el-select></el-form-item>
        <el-form-item label="操作类型"><el-select v-model="query.operType" placeholder="全部" clearable style="width:120px"><el-option label="新增" value="add" /><el-option label="修改" value="update" /><el-option label="删除" value="delete" /><el-option label="其他" value="other" /></el-select></el-form-item>
        <el-form-item label="时间范围"><el-date-picker v-model="query.dateRange" type="daterange" range-separator="至" start-placeholder="开始" end-placeholder="结束" value-format="YYYY-MM-DD" /></el-form-item>
        <el-form-item><el-button type="primary" :icon="Search" @click="loadList">查询</el-button><el-button :icon="Refresh" @click="resetQuery">重置</el-button></el-form-item>
      </el-form>
    </el-card>
    <el-card shadow="never">
      <el-table v-loading="loading" :data="list" border>
        <el-table-column type="index" label="序号" width="60" />
        <el-table-column prop="operName" label="操作人" width="120" />
        <el-table-column prop="module" label="模块" width="140" />
        <el-table-column prop="operType" label="类型" width="80"><template #default="{row}"><el-tag size="small">{{ row.operType }}</el-tag></template></el-table-column>
        <el-table-column prop="description" label="描述" show-overflow-tooltip />
        <el-table-column prop="requestMethod" label="方法" width="80" />
        <el-table-column prop="operIp" label="IP" width="140" />
        <el-table-column label="耗时" width="80"><template #default="{row}">{{ row.costTime }}ms</template></el-table-column>
        <el-table-column prop="operTime" label="操作时间" width="170" :formatter="formatTimeCol" />
        <el-table-column label="操作" width="80" fixed="right"><template #default="{row}"><el-button link type="primary" @click="handleView(row)">详情</el-button></template></el-table-column>
      </el-table>
      <div class="pagination-wrap"><el-pagination v-model:current-page="query.pageNum" v-model:page-size="query.pageSize" :total="total" layout="total, sizes, prev, pager, next, jumper" @size-change="loadList" @current-change="loadList" /></div>
    </el-card>
    <el-dialog v-model="detailVisible" title="操作详情" width="720px">
      <el-descriptions :column="2" border>
        <el-descriptions-item label="操作人">{{ detail.operName }}</el-descriptions-item>
        <el-descriptions-item label="模块">{{ detail.module }}</el-descriptions-item>
        <el-descriptions-item label="类型">{{ detail.operType }}</el-descriptions-item>
        <el-descriptions-item label="方法">{{ detail.requestMethod }}</el-descriptions-item>
        <el-descriptions-item label="URL" :span="2">{{ detail.operUrl }}</el-descriptions-item>
        <el-descriptions-item label="IP">{{ detail.operIp }}</el-descriptions-item>
        <el-descriptions-item label="耗时">{{ detail.costTime }}ms</el-descriptions-item>
        <el-descriptions-item label="时间" :span="2">{{ detail.operTime }}</el-descriptions-item>
        <el-descriptions-item label="描述" :span="2">{{ detail.description }}</el-descriptions-item>
        <el-descriptions-item label="请求参数" :span="2"><pre>{{ detail.operParam }}</pre></el-descriptions-item>
        <el-descriptions-item label="返回结果" :span="2"><pre>{{ detail.jsonResult }}</pre></el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { Search, Refresh } from '@element-plus/icons-vue'
import { listOperLog } from '@/api/system'
import { formatTimeCol } from '@/utils/format'

const loading = ref(false)
const list = ref([])
const total = ref(0)
const query = reactive({ pageNum: 1, pageSize: 10, operName: '', module: '', operType: '', dateRange: [] })
const detailVisible = ref(false)
const detail = ref({})
const modules = [
  { label: '用户管理', value: 'user' }, { label: '系统管理', value: 'system' },
  { label: '应用管理', value: 'app' }, { label: '授权管理', value: 'grant' },
  { label: '登录日志', value: 'login' }
]

const loadList = async () => { loading.value = true; try { const res = await listOperLog(query); list.value = res.rows || []; total.value = res.total || 0 } finally { loading.value = false } }
const resetQuery = () => { query.operName = ''; query.module = ''; query.operType = ''; query.dateRange = []; query.pageNum = 1; loadList() }
const handleView = (row) => { detail.value = row; detailVisible.value = true }
loadList()
</script>

<style scoped>
pre { white-space: pre-wrap; word-wrap: break-word; max-height: 200px; overflow: auto; margin: 0; }
</style>
