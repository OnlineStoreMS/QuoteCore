<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  approveSecondEdit,
  copyQuote,
  deleteQuote,
  fetchDashboardStats,
  isSkeletonTemplate,
  listQuotes,
  listTemplates,
  voidQuote,
  type Quote,
  type QuoteTemplate,
} from '../api/quote'

const router = useRouter()
const route = useRoute()
const loading = ref(false)
const list = ref<Quote[]>([])
const total = ref(0)
const query = reactive({ keyword: '', status: '' as string | number | '', secondEdit: '' as string, page: 1, pageSize: 20 })
const applyCount = ref(0)

const bizDialog = ref(false)
const bizLoading = ref(false)
const bizTemplates = ref<QuoteTemplate[]>([])
const selectedBizId = ref<number | null>(null)

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
      secondEdit: query.secondEdit || undefined,
      page: query.page,
      pageSize: query.pageSize,
    })
    list.value = data.list || []
    total.value = data.total || 0
    try {
      const st = await fetchDashboardStats()
      applyCount.value = st.secondEditApplyCount || 0
    } catch {
      applyCount.value = list.value.filter((q) => q.isSecondEdit).length
    }
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function openBizCreate() {
  bizDialog.value = true
  bizLoading.value = true
  selectedBizId.value = null
  try {
    const all = await listTemplates()
    bizTemplates.value = all.filter((t) => isSkeletonTemplate(t))
    if (bizTemplates.value.length === 1) {
      selectedBizId.value = bizTemplates.value[0].id
    }
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    bizLoading.value = false
  }
}

function confirmBizCreate() {
  if (!selectedBizId.value) {
    ElMessage.warning('请选择业务模板')
    return
  }
  bizDialog.value = false
  router.push({ path: '/quotes/new', query: { bizTemplateId: String(selectedBizId.value) } })
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

async function onApprove(row: Quote) {
  try {
    await approveSecondEdit(row.id)
    ElMessage.success('已通过二次编辑申请')
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

onMounted(async () => {
  if (route.query.secondEdit) query.secondEdit = '1'
  await load()
  if (route.query.fromBiz === '1') {
    await openBizCreate()
    router.replace({ path: '/quotes' })
  }
})
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
      <el-select v-model="query.secondEdit" clearable placeholder="二次编辑" style="width: 140px" @change="query.page=1; load()">
        <el-option label="二次编辑申请" value="1" />
        <el-option label="待审核" value="pending" />
      </el-select>
      <el-button type="primary" @click="query.page=1; load()">查询</el-button>
      <el-button type="success" @click="router.push('/quotes/new')">新建报价</el-button>
      <el-button type="warning" plain @click="openBizCreate">从业务模板创建</el-button>
      <el-tag
        v-if="applyCount"
        type="warning"
        class="apply-tip"
        style="cursor:pointer"
        @click="query.secondEdit='pending'; query.page=1; load()"
      >
        {{ applyCount }} 条二次编辑申请
      </el-tag>
    </div>

    <el-table v-loading="loading" :data="list" border stripe>
      <el-table-column prop="quoteNo" label="单号" width="150" />
      <el-table-column prop="title" label="标题" min-width="140">
        <template #default="{ row }">
          <div>{{ row.title }}</div>
          <el-tag v-if="row.isSecondEdit" size="small" :type="row.secondEditApproved ? 'success' : 'warning'">
            {{ row.secondEditApproved ? '二次编辑已通过' : '二次编辑待审核' }} · 原单 {{ row.originQuoteNo }}
          </el-tag>
          <el-tag v-else-if="row.secondEditUsed" size="small" type="danger">二次编辑申请</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="客户" width="140">
        <template #default="{ row }">
          <div>{{ row.customerName || '—' }}</div>
          <div v-if="row.isSecondEdit && row.secondEditApplicant" class="applicant">申请人 {{ row.secondEditApplicant }} {{ row.secondEditApplicantPhone }}</div>
        </template>
      </el-table-column>
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
      <el-table-column label="操作" width="300" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="router.push(`/quotes/${row.id}`)">编辑</el-button>
          <el-button
            v-if="row.isSecondEdit && !row.secondEditApproved"
            link
            type="success"
            @click="onApprove(row)"
          >
            通过审核
          </el-button>
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

    <el-dialog v-model="bizDialog" title="从业务模板创建报价" width="480px" destroy-on-close>
      <div v-loading="bizLoading">
        <p class="biz-hint">选择业务模板后，将带入产品/配件固定行，再填写名称、规格与价格。</p>
        <el-empty v-if="!bizLoading && !bizTemplates.length" description="暂无业务模板，请先到「业务模板」创建" />
        <el-radio-group v-else v-model="selectedBizId" class="biz-list">
          <el-radio
            v-for="t in bizTemplates"
            :key="t.id"
            :value="t.id"
            border
            class="biz-item"
          >
            <div class="biz-item-body">
              <strong>{{ t.name }}</strong>
              <span>{{ t.lines?.length || 0 }} 行明细</span>
            </div>
          </el-radio>
        </el-radio-group>
      </div>
      <template #footer>
        <el-button @click="bizDialog = false">取消</el-button>
        <el-button type="primary" :disabled="!selectedBizId" @click="confirmBizCreate">创建并填写</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 12px; flex-wrap: wrap; align-items: center; }
.applicant { font-size: 12px; color: #e6a23c; margin-top: 2px; }
.pager { margin-top: 12px; display: flex; justify-content: flex-end; }
.biz-hint { margin: 0 0 12px; color: #909399; font-size: 13px; }
.biz-list { display: flex; flex-direction: column; gap: 8px; width: 100%; align-items: stretch; }
.biz-item { width: 100%; height: auto; margin: 0 !important; padding: 10px 12px; }
.biz-item-body { display: flex; flex-direction: column; gap: 2px; text-align: left; }
.biz-item-body span { color: #909399; font-size: 12px; font-weight: 400; }
</style>
