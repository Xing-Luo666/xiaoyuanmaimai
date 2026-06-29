<template>
  <div>
    <el-card class="search-card" shadow="never">
      <el-form :inline="true" :model="query" class="search-form">
        <el-form-item label="用户类型">
          <el-select v-model="query.userType" placeholder="全部" clearable style="width:140px">
            <el-option label="个体用户" value="personal" />
            <el-option label="企业用户" value="corp" />
          </el-select>
        </el-form-item>
        <el-form-item label="审核状态">
          <el-select v-model="query.auditStatus" placeholder="全部" clearable style="width:140px">
            <el-option label="待审核" value="1" />
            <el-option label="已通过" value="2" />
            <el-option label="已驳回" value="3" />
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
        <el-table-column prop="username" label="账号" width="140" />
        <el-table-column prop="realName" label="姓名/名称" width="160" show-overflow-tooltip />
        <el-table-column prop="phone" label="联系电话" width="140" />
        <el-table-column label="审核状态" width="100">
          <template #default="{row}">
            <el-tag :type="auditTag(row.auditStatus)">{{ auditText(row.auditStatus) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="auditRemark" label="审核备注" show-overflow-tooltip />
        <el-table-column prop="createTime" label="提交时间" width="170" :formatter="formatTimeCol" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{row}">
            <el-button link type="success" :disabled="row.auditStatus !== 1" @click="handleAudit(row, 2)">通过</el-button>
            <el-button link type="danger" :disabled="row.auditStatus !== 1" @click="handleAudit(row, 3)">驳回</el-button>
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

    <el-dialog v-model="dialog.visible" :title="dialog.title" width="500px">
      <el-form label-width="100px">
        <el-form-item label="审核结果">
          <el-tag :type="dialog.status === 2 ? 'success' : 'danger'">
            {{ dialog.status === 2 ? '通过' : '驳回' }}
          </el-tag>
        </el-form-item>
        <el-form-item label="审核备注">
          <el-input v-model="dialog.remark" type="textarea" :rows="3" placeholder="请输入审核备注" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialog.visible = false">取消</el-button>
        <el-button type="primary" @click="submitAudit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, Refresh } from '@element-plus/icons-vue'
import { listAudit, auditUser, auditCorp } from '@/api/uas'
import { formatTimeCol } from '@/utils/format'

const loading = ref(false)
const list = ref([])
const total = ref(0)
const query = reactive({ pageNum: 1, pageSize: 10, userType: '', auditStatus: '' })
const dialog = reactive({ visible: false, title: '', id: null, userType: '', status: 2, remark: '' })

const auditText = (s) => ({ 0: '未提交', 1: '待审核', 2: '已通过', 3: '已驳回' }[s] || '未提交')
const auditTag = (s) => ({ 0: 'info', 1: 'warning', 2: 'success', 3: 'danger' }[s] || 'info')

const loadList = async () => {
  loading.value = true
  try {
    const res = await listAudit(query)
    list.value = res.rows || []
    total.value = res.total || 0
  } finally {
    loading.value = false
  }
}

const resetQuery = () => {
  query.userType = ''
  query.auditStatus = ''
  query.pageNum = 1
  loadList()
}

const handleAudit = (row, status) => {
  dialog.id = row.id
  dialog.userType = row.userType
  dialog.status = status
  dialog.remark = ''
  dialog.title = status === 2 ? '审核通过' : '审核驳回'
  dialog.visible = true
}

const submitAudit = async () => {
  try {
    if (dialog.userType === 'personal') {
      await auditUser(dialog.id, { auditStatus: dialog.status, auditRemark: dialog.remark })
    } else {
      await auditCorp(dialog.id, { auditStatus: dialog.status, auditRemark: dialog.remark })
    }
    ElMessage.success('审核成功')
    dialog.visible = false
    loadList()
  } catch (e) {}
}

loadList()
</script>
