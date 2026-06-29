<template>
  <div>
    <el-card class="search-card" shadow="never">
      <el-form :inline="true" :model="query" class="search-form">
        <el-form-item label="角色名称">
          <el-input v-model="query.roleName" placeholder="请输入角色名称" clearable @keyup.enter="loadList" />
        </el-form-item>
        <el-form-item label="权限字符">
          <el-input v-model="query.roleKey" placeholder="请输入权限字符" clearable @keyup.enter="loadList" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="loadList">查询</el-button>
          <el-button :icon="Refresh" @click="resetQuery">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never">
      <div class="mb10"><el-button type="primary" :icon="Plus" @click="handleAdd">新增</el-button></div>
      <el-table v-loading="loading" :data="list" border>
        <el-table-column type="index" label="序号" width="60" />
        <el-table-column prop="roleName" label="角色名称" width="180" />
        <el-table-column prop="roleKey" label="权限字符" width="180" />
        <el-table-column prop="roleSort" label="显示顺序" width="100" />
        <el-table-column label="状态" width="100">
          <template #default="{row}"><el-tag :type="row.status === 1 ? 'success' : 'danger'">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" show-overflow-tooltip />
        <el-table-column prop="createTime" label="创建时间" width="170" :formatter="formatTimeCol" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{row}">
            <el-button link type="primary" @click="handleEdit(row)">编辑</el-button>
            <el-button link type="danger" :disabled="row.roleKey === 'admin'" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination-wrap">
        <el-pagination v-model:current-page="query.pageNum" v-model:page-size="query.pageSize" :total="total" layout="total, prev, pager, next, jumper" @current-change="loadList" />
      </div>
    </el-card>

    <el-dialog v-model="dialog.visible" :title="dialog.title" width="640px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="角色名称" prop="roleName"><el-input v-model="form.roleName" /></el-form-item>
        <el-form-item label="权限字符" prop="roleKey"><el-input v-model="form.roleKey" /></el-form-item>
        <el-form-item label="显示顺序"><el-input-number v-model="form.roleSort" :min="0" /></el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status">
            <el-radio :value="1">启用</el-radio>
            <el-radio :value="0">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="菜单权限">
          <el-tree
            ref="menuTreeRef"
            :data="menuTree"
            show-checkbox
            node-key="id"
            :default-checked-keys="form.menuIds"
            :props="{ label: 'label', children: 'children' }"
            check-strictly
          />
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="form.remark" type="textarea" :rows="2" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialog.visible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Refresh, Plus } from '@element-plus/icons-vue'
import { listRole, addRole, updateRole, deleteRole, menuTreeSelect } from '@/api/system'
import { formatTimeCol } from '@/utils/format'

const loading = ref(false)
const list = ref([])
const total = ref(0)
const query = reactive({ pageNum: 1, pageSize: 10, roleName: '', roleKey: '' })
const formRef = ref(null)
const menuTreeRef = ref(null)
const menuTree = ref([])
const dialog = reactive({ visible: false, title: '', type: 'add' })
const form = reactive({ id: null, roleName: '', roleKey: '', roleSort: 0, status: 1, remark: '', menuIds: [] })
const rules = {
  roleName: [{ required: true, message: '请输入角色名称', trigger: 'blur' }],
  roleKey: [{ required: true, message: '请输入权限字符', trigger: 'blur' }]
}

const loadList = async () => {
  loading.value = true
  try { const res = await listRole(query); list.value = res.rows || []; total.value = res.total || 0 } finally { loading.value = false }
}
const resetQuery = () => { query.roleName = ''; query.roleKey = ''; query.pageNum = 1; loadList() }
const resetForm = () => { Object.assign(form, { id: null, roleName: '', roleKey: '', roleSort: 0, status: 1, remark: '', menuIds: [] }) }

const handleAdd = () => { resetForm(); dialog.title = '新增角色'; dialog.type = 'add'; dialog.visible = true }
const handleEdit = (row) => {
  resetForm()
  Object.assign(form, row)
  form.menuIds = row.menuIds || []
  dialog.title = '编辑角色'; dialog.type = 'edit'; dialog.visible = true
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除「${row.roleName}」？`, '提示', { type: 'warning' })
    await deleteRole(row.id); ElMessage.success('删除成功'); loadList()
  } catch (e) {}
}

const submitForm = async () => {
  try {
    await formRef.value.validate()
    form.menuIds = menuTreeRef.value?.getCheckedKeys() || []
    if (dialog.type === 'add') { await addRole(form); ElMessage.success('新增成功') }
    else { await updateRole(form); ElMessage.success('修改成功') }
    dialog.visible = false; loadList()
  } catch (e) {}
}

const loadMenuTree = async () => {
  try { const res = await menuTreeSelect(); menuTree.value = res.data || [] } catch (e) {}
}

loadMenuTree()
loadList()
</script>
