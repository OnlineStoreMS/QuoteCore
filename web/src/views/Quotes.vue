<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { copyQuote, deleteQuote, listQuotes, voidQuote, type Quote } from '../api/quote'

const router = useRouter()
const loading = ref(false)
const list = ref<Quote[]>([])
const total = ref(0)
const query = reactive({ keyword: '', status: '' as string | number | '', page: 1, pageSize: 20 })

const statusMap: Record<number, { label: string; type: '' | 'info' | 'success' | 'warning' | 'danger' }> = {
  1: { label: '草稿', type: 'info' },
  2: { label: '已发送', type: 'warning' },
  3: { label: '已成交', type: 'success' },
  4: { label: '作废', type: 'danger' },
}

async function load() {
  loading.value = true
  try {
    const data = await listQuotes({
      keyword: query.keyword || undefined,
      status: query.status === '' ? undefined : query.status,
      page: query.page,
      pageSize: query.pageSize,
    })
    list.value = data.list || []
    total.value = data.total || 0
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function onCopy(row: Quote) {
  try {
    const q = await copyQuote(row.id)
    ElMessage.success('已复制')
    router.push(`/quotes/${q.id}`)
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function onVoid(row: Quote) {
  await ElMessageBox.confirm(`确认作废报价单 ${row.quoteNo}？`, '提示', { type: 'warning' })
  try {
    await voidQuote(row.id)
    ElMessage.success('已作废')
    load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function onDelete(row: Quote) {
  await ElMessageBox.confirm(`确认删除报价单 ${row.quoteNo}？`, '提示', { type: 'warning' })
  try {
    await deleteQuote(row.id)
    ElMessage.success('已删除')
    load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(load)
</script>

<template>
  <div>
    <div class="toolbar">
      <el-input v-model="query.keyword" clearable placeholder="单号/标题/客户/电话" style="width: 240px" @keyup.enter="query.page=1; load()" />
      <el-select v-model="query.status" clearable placeholder="状态" style="width: 120px" @change="query.page=1; load()">
        <el-option label="草稿" :value="1" />
        <el-option label="已发送" :value="2" />
        <el-option label="已成交" :value="3" />
        <el-option label="作废" :value="4" />
      </el-select>
      <el-button type="primary" @click="query.page=1; load()">查询</el-button>
      <el-button type="success" @click="router.push('/quotes/new')">新建报价</el-button>
    </div>

    <el-table v-loading="loading" :data="list" border stripe>
      <el-table-column prop="quoteNo" label="单号" width="150" />
      <el-table-column prop="title" label="标题" min-width="140" />
      <el-table-column prop="customerName" label="客户" width="120" />
      <el-table-column prop="contactPhone" label="电话" width="120" />
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="statusMap[row.status]?.type || 'info'" size="small">{{ statusMap[row.status]?.label || row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="合计" width="110" align="right">
        <template #default="{ row }">¥{{ Number(row.totalAmt || 0).toFixed(2) }}</template>
      </el-table-column>
      <el-table-column prop="updatedAt" label="更新时间" width="170" />
      <el-table-column label="操作" width="260" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="router.push(`/quotes/${row.id}`)">编辑</el-button>
          <el-button link type="primary" @click="onCopy(row)">复制</el-button>
          <el-button link type="warning" :disabled="row.status === 4" @click="onVoid(row)">作废</el-button>
          <el-button link type="danger" @click="onDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pager">
      <el-pagination
        v-model:current-page="query.page"
        v-model:page-size="query.pageSize"
        :total="total"
        layout="total, prev, pager, next"
        @current-change="load"
      />
    </div>
  </div>
</template>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 12px; flex-wrap: wrap; }
.pager { margin-top: 12px; display: flex; justify-content: flex-end; }
</style>
