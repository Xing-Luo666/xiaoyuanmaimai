<template>
  <div>
    <el-card class="search-card" shadow="never">
      <el-form :inline="true" :model="query" class="search-form">
        <el-form-item label="用户名"><el-input v-model="query.username" placeholder="请输入" clearable @keyup.enter="loadList" /></el-form-item>
        <el-form-item label="IP"><el-input v-model="query.ipaddr" placeholder="请输入" clearable @keyup.enter="loadList" /></el-form-item>
        <el-form-item label="状态"><el-select v-model="query.status" placeholder="全部" clearable style="width:120px"><el-option label="成功" value="1" /><el-option label="失败" value="0" /></el-select></el-form-item>
        <el-form-item><el-button type="primary" :icon="Search" @click="loadList">查询</el-button><el-button :icon="Refresh" @click="resetQuery">重置</el-button></el-form-item>
      </el-form>
    </el-card>
    <el-card shadow="never">
      <el-table v-loading="loading" :data="list" border>
        <el-table-column type="index" label="序号" width="60" />
        <el-table-column prop="username" label="用户名" width="140" />
        <el-table-column prop="ipaddr" label="IP" width="140" />
        <el-table-column prop="loginLocation" label="登录地点" width="140" />
        <el-table-column prop="browser" label="浏览器" width="120" />
        <el-table-column prop="os" label="操作系统" width="160" />
        <el-table-column label="状态" width="80"><template #default="{row}"><el-tag :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '成功' : '失败' }}</el-tag></template></el-table-column>
        <el-table-column prop="msg" label="消息" show-overflow-tooltip />
        <el-table-column prop="loginTime" label="登录时间" width="170" :formatter="formatTimeCol" />
      </el-table>
      <div class="pagination-wrap"><el-pagination v-model:current-page="query.pageNum" v-model:page-size="query.pageSize" :total="total" layout="total, prev, pager, next, jumper" @current-change="loadList" /></div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { Search, Refresh } from '@element-plus/icons-vue'
import { listLoginLog } from '@/api/system'
import { formatTimeCol } from '@/utils/format'

const loading = ref(false)
const list = ref([])
const total = ref(0)
const query = reactive({ pageNum: 1, pageSize: 10, username: '', ipaddr: '', status: '' })

const loadList = async () => { loading.value = true; try { const res = await listLoginLog(query); list.value = res.rows || []; total.value = res.total || 0 } finally { loading.value = false } }
const resetQuery = () => { query.username = ''; query.ipaddr = ''; query.status = ''; query.pageNum = 1; loadList() }
loadList()
</script>
