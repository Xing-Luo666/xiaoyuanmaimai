<template>
  <div>
    <el-card class="search-card" shadow="never">
      <el-form :inline="true" :model="query" class="search-form">
        <el-form-item label="菜单名称">
          <el-input v-model="query.menuName" placeholder="请输入菜单名称" clearable @keyup.enter="loadList" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="loadList">查询</el-button>
          <el-button :icon="Refresh" @click="resetQuery">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>
    <el-card shadow="never">
      <div class="mb10"><el-button type="primary" :icon="Plus" @click="handleAdd(null)">新增</el-button></div>
      <el-table v-loading="loading" :data="list" border row-key="id" default-expand-all :tree-props="{ children: 'children' }">
        <el-table-column prop="menuName" label="菜单名称" width="200" />
        <el-table-column prop="icon" label="图标" width="80" align="center">
          <template #default="{row}"><el-icon v-if="row.icon"><component :is="row.icon" /></el-icon></template>
        </el-table-column>
        <el-table-column prop="menuSort" label="排序" width="80" />
        <el-table-column prop="path" label="路径" width="160" />
        <el-table-column prop="component" label="组件" width="200" show-overflow-tooltip />
        <el-table-column prop="menuType" label="类型" width="80">
          <template #default="{row}"><el-tag size="small">{{ { M: '目录', C: '菜单', F: '按钮' }[row.menuType] }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="perms" label="权限标识" width="160" />
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{row}">
            <el-button link type="primary" @click="handleAdd(row)">新增子菜单</el-button>
            <el-button link type="primary" @click="handleEdit(row)">编辑</el-button>
            <el-button link type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialog.visible" :title="dialog.title" width="640px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="上级菜单">
          <el-tree-select v-model="form.parentId" :data="treeData" :props="{ label: 'menuName', value: 'id', children: 'children' }" check-strictly clearable placeholder="不选则为顶级菜单" />
        </el-form-item>
        <el-form-item label="菜单类型">
          <el-radio-group v-model="form.menuType">
            <el-radio value="M">目录</el-radio>
            <el-radio value="C">菜单</el-radio>
            <el-radio value="F">按钮</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="菜单名称" prop="menuName"><el-input v-model="form.menuName" /></el-form-item>
        <el-row>
          <el-col :span="12"><el-form-item label="图标"><el-input v-model="form.icon" placeholder="Element图标名" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="排序"><el-input-number v-model="form.menuSort" :min="0" /></el-form-item></el-col>
        </el-row>
        <el-row v-if="form.menuType !== 'F'">
          <el-col :span="12"><el-form-item label="路由路径"><el-input v-model="form.path" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="组件路径"><el-input v-model="form.component" /></el-form-item></el-col>
        </el-row>
        <el-form-item label="权限标识"><el-input v-model="form.perms" placeholder="如 system:user:list" /></el-form-item>
        <el-form-item label="是否显示" v-if="form.menuType !== 'F'">
          <el-radio-group v-model="form.visible">
            <el-radio :value="1">显示</el-radio>
            <el-radio :value="0">隐藏</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialog.visible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Refresh, Plus } from '@element-plus/icons-vue'
import { listMenu, addMenu, updateMenu, deleteMenu } from '@/api/system'

const loading = ref(false)
const list = ref([])
const query = reactive({ menuName: '' })
const formRef = ref(null)
const dialog = reactive({ visible: false, title: '', type: 'add' })
const form = reactive({ id: null, parentId: 0, menuName: '', menuSort: 0, path: '', component: '', menuType: 'C', visible: 1, perms: '', icon: '' })
const rules = { menuName: [{ required: true, message: '请输入菜单名称', trigger: 'blur' }] }

const treeData = computed(() => [{ id: 0, menuName: '主目录', children: list.value }])

const loadList = async () => {
  loading.value = true
  try { const res = await listMenu(query); list.value = res.data || [] } finally { loading.value = false }
}
const resetQuery = () => { query.menuName = ''; loadList() }
const resetForm = () => { Object.assign(form, { id: null, parentId: 0, menuName: '', menuSort: 0, path: '', component: '', menuType: 'C', visible: 1, perms: '', icon: '' }) }

const handleAdd = (row) => { resetForm(); if (row) form.parentId = row.id; dialog.title = '新增菜单'; dialog.type = 'add'; dialog.visible = true }
const handleEdit = (row) => { resetForm(); Object.assign(form, row); dialog.title = '编辑菜单'; dialog.type = 'edit'; dialog.visible = true }
const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除「${row.menuName}」？`, '提示', { type: 'warning' })
    await deleteMenu(row.id); ElMessage.success('删除成功'); loadList()
  } catch (e) {}
}

const submitForm = async () => {
  try {
    await formRef.value.validate()
    if (dialog.type === 'add') { await addMenu(form); ElMessage.success('新增成功') }
    else { await updateMenu(form); ElMessage.success('修改成功') }
    dialog.visible = false; loadList()
  } catch (e) {}
}

loadList()
</script>
