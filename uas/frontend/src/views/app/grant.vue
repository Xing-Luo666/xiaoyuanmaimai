<template>
  <div>
    <el-card class="search-card" shadow="never">
      <el-form :inline="true" :model="query" class="search-form">
        <el-form-item label="AppID">
          <el-input v-model="query.appId" placeholder="请输入AppID" clearable @keyup.enter="loadList" />
        </el-form-item>
        <el-form-item label="用户类型">
          <el-select v-model="query.userType" placeholder="全部" clearable style="width:120px">
            <el-option label="个体用户" value="personal" />
            <el-option label="企业用户" value="corp" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="loadList">查询</el-button>
          <el-button :icon="Refresh" @click="resetQuery">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never">
      <el-table v-loading="loading" :data="list" border>
        <el-table-column type="index" label="序号" width="60" />
        <el-table-column prop="userType" label="用户类型" width="100">
          <template #default="{row}">
            <el-tag :type="row.userType === 'personal' ? '' : 'success'">
              {{ row.userType === 'personal' ? '个体用户' : '企业用户' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="userName" label="用户姓名" width="140" show-overflow-tooltip />
        <el-table-column prop="appId" label="AppID" width="180" show-overflow-tooltip />
        <el-table-column prop="appName" label="应用名称" width="180" show-overflow-tooltip />
        <el-table-column prop="grantTime" label="授权时间" width="170" :formatter="formatTimeCol" />
        <el-table-column prop="expireTime" label="过期时间" width="170">
          <template #default="{row}">
            {{ row.expireTime ? formatTime(row.expireTime) : '永久' }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{row}">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">
              {{ row.status === 1 ? '已授权' : '已撤销' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{row}">
            <el-button link type="danger" :disabled="row.status !== 1" @click="handleRevoke(row)">撤销</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination-wrap">
        <el-pagination
          v-model:current-page="query.pageNum"
          v-model:page-size="query.pageSize"
          :total="total"
          layout="total, prev, pager, next, jumper"
          @current-change="loadList"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Refresh } from '@element-plus/icons-vue'
import { listGrant, deleteGrant } from '@/api/uas'
import { formatTime, formatTimeCol } from '@/utils/format'

const loading = ref(false)
const list = ref([])
const total = ref(0)
const query = reactive({ pageNum: 1, pageSize: 10, appId: '', userType: '' })

const loadList = async () => {
  loading.value = true
  try {
    const res = await listGrant(query)
    list.value = res.rows || []
    total.value = res.total || 0
  } finally {
    loading.value = false
  }
}

const resetQuery = () => {
  query.appId = ''
  query.userType = ''
  query.pageNum = 1
  loadList()
}

const handleRevoke = async (row) => {
  try {
    await ElMessageBox.confirm(`确认撤销「${row.userName}」对「${row.appName}」的授权？`, '提示', { type: 'warning' })
    await deleteGrant(row.id)
    ElMessage.success('撤销成功')
    loadList()
  } catch (e) {}
}

loadList()
</script>
