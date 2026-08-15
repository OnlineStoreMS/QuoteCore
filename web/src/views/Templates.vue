<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ASSEMBLE_SKELETON_PRESET,
  deleteTemplate,
  isSkeletonTemplate,
  listTemplates,
  saveTemplate,
  type QuoteTemplate,
  type QuoteTemplateLine,
} from '../api/quote'
import { uploadImage, uploadImageFromUrl, isQuoteStoredUrl } from '../api/upload'

const route = useRoute()
const router = useRouter()
const pageKind = computed<'layout' | 'skeleton'>(() =>
  route.meta.templateKind === 'skeleton' ? 'skeleton' : 'layout',
)
const isBizPage = computed(() => pageKind.value === 'skeleton')
const pageTitle = computed(() => (isBizPage.value ? '业务模板' : '版式模板'))

function createFromBiz(row: QuoteTemplate) {
  router.push({ path: '/quotes/new', query: { bizTemplateId: String(row.id) } })
}

const loading = ref(false)
const allList = ref<QuoteTemplate[]>([])
const dialog = ref(false)
const editingId = ref<number | undefined>()
const form = reactive({
  name: '',
  kind: 'layout' as 'layout' | 'skeleton',
  isDefault: false,
  logoUrl: '',
  shopName: '',
  shopPhone: '',
  shopAddress: '',
  headerSubtitle: '',
  footerText: '',
  showLogo: true,
  showRetailPrice: true,
  showSpecImage: true,
  showUpgrade: true,
  showParams: true,
  showTotals: true,
  stylePreset: 'compare',
  lines: [] as QuoteTemplateLine[],
})

const list = computed(() =>
  allList.value.filter((t) => {
    const sk = isSkeletonTemplate(t)
    return isBizPage.value ? sk : !sk
  }),
)

async function load() {
  loading.value = true
  try {
    allList.value = await listTemplates()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

function resetForm(partial?: Partial<typeof form>) {
  Object.assign(form, {
    name: isBizPage.value ? '新业务模板' : '新版式模板',
    kind: pageKind.value,
    isDefault: false,
    logoUrl: '',
    shopName: '',
    shopPhone: '',
    shopAddress: '',
    headerSubtitle: isBizPage.value ? '组车配件清单' : '',
    footerText: '本报价单有效期内价格有效。',
    showLogo: true,
    showRetailPrice: true,
    showSpecImage: true,
    showUpgrade: !isBizPage.value,
    showParams: !isBizPage.value,
    showTotals: true,
    stylePreset: 'compare',
    lines: [] as QuoteTemplateLine[],
    ...partial,
  })
}

function openCreate() {
  editingId.value = undefined
  if (isBizPage.value) {
    resetForm({
      name: '组装车业务模板',
      kind: 'skeleton',
      showUpgrade: false,
      showParams: false,
      lines: ASSEMBLE_SKELETON_PRESET.map((x) => ({ ...x })),
    })
  } else {
    resetForm({ kind: 'layout' })
  }
  logoLinkDraft.value = ''
  dialog.value = true
}

function openEdit(row: QuoteTemplate) {
  editingId.value = row.id
  resetForm({
    name: row.name,
    kind: pageKind.value,
    isDefault: row.isDefault,
    logoUrl: row.logoUrl || '',
    shopName: row.shopName || '',
    shopPhone: row.shopPhone || '',
    shopAddress: row.shopAddress || '',
    headerSubtitle: row.headerSubtitle || '',
    footerText: row.footerText || '',
    showLogo: row.showLogo,
    showRetailPrice: row.showRetailPrice,
    showSpecImage: row.showSpecImage,
    showUpgrade: row.showUpgrade,
    showParams: row.showParams,
    showTotals: row.showTotals,
    stylePreset: row.stylePreset || 'compare',
    lines: (row.lines || []).map((ln) => ({
      sort: ln.sort,
      category: ln.category || '',
      partName: ln.partName || '',
      hint: ln.hint || '',
    })),
  })
  logoLinkDraft.value = row.logoUrl || ''
  dialog.value = true
}

const logoLinkDraft = ref('')
const logoUploading = ref(false)

async function onUpload(opt: { file: File }) {
  try {
    form.logoUrl = await uploadImage(opt.file, 'logo')
    logoLinkDraft.value = form.logoUrl
    ElMessage.success('Logo 已上传')
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function applyLogoLink() {
  const raw = logoLinkDraft.value.trim()
  if (!raw) {
    form.logoUrl = ''
    return
  }
  if (isQuoteStoredUrl(raw)) {
    form.logoUrl = raw
    ElMessage.success('已使用本站图片')
    return
  }
  if (!/^https?:\/\//i.test(raw)) {
    ElMessage.error('请粘贴以 http(s):// 开头的图片链接')
    return
  }
  logoUploading.value = true
  try {
    form.logoUrl = await uploadImageFromUrl(raw, 'logo')
    logoLinkDraft.value = form.logoUrl
    ElMessage.success('已上传到报价中心')
  } catch (e) {
    ElMessage.error((e as Error).message || '上传失败')
  } finally {
    logoUploading.value = false
  }
}

function addPartLine(idx: number) {
  const head = idx
  let start = head
  while (start > 0 && (form.lines[start].category || '') === (form.lines[start - 1].category || '') && (form.lines[start].category || '').trim()) {
    start -= 1
  }
  // find end of group containing idx
  const cat = (form.lines[idx]?.category || '').trim()
  let end = idx
  if (cat) {
    for (let i = idx + 1; i < form.lines.length; i++) {
      if ((form.lines[i].category || '').trim() === cat) end = i
      else break
    }
  }
  form.lines.splice(end + 1, 0, {
    sort: 0,
    category: cat || form.lines[idx]?.category || '',
    partName: '',
    hint: '',
  })
  form.lines.forEach((ln, i) => {
    ln.sort = (i + 1) * 10
  })
}

function addProductGroupLine(idx?: number) {
  const at =
    typeof idx === 'number'
      ? (() => {
          const cat = (form.lines[idx]?.category || '').trim()
          let end = idx
          if (cat) {
            for (let i = idx + 1; i < form.lines.length; i++) {
              if ((form.lines[i].category || '').trim() === cat) end = i
              else break
            }
          }
          return end + 1
        })()
      : form.lines.length
  form.lines.splice(at, 0, {
    sort: 0,
    category: '',
    partName: '',
    hint: '',
  })
  form.lines.forEach((ln, i) => {
    ln.sort = (i + 1) * 10
  })
}

function isLineCategoryHead(idx: number): boolean {
  if (idx <= 0) return true
  const cur = (form.lines[idx].category || '').trim()
  if (!cur) return true
  return cur !== (form.lines[idx - 1].category || '').trim()
}

function onLineCategoryInput(idx: number, val: string) {
  if (!isLineCategoryHead(idx)) return
  const old = form.lines[idx].category || ''
  const catOld = old.trim()
  let end = idx
  if (catOld) {
    for (let i = idx + 1; i < form.lines.length; i++) {
      if ((form.lines[i].category || '').trim() === catOld) end = i
      else break
    }
  }
  for (let i = idx; i <= end; i++) form.lines[i].category = val
}

function removeLine(idx: number) {
  form.lines.splice(idx, 1)
  form.lines.forEach((ln, i) => {
    ln.sort = (i + 1) * 10
  })
}

function loadPreset() {
  form.kind = 'skeleton'
  form.lines = ASSEMBLE_SKELETON_PRESET.map((x) => ({ ...x }))
  ElMessage.success('已载入组装车预设')
}

async function onSave() {
  form.kind = pageKind.value
  if (form.kind === 'skeleton') {
    const bad = form.lines.findIndex((ln) => !(ln.partName || '').trim())
    if (bad >= 0) {
      ElMessage.warning(`第 ${bad + 1} 行配件名称必填`)
      return
    }
    if (!form.lines.length) {
      ElMessage.warning('业务模板至少需要一行配件')
      return
    }
  }
  try {
    const payload =
      pageKind.value === 'skeleton'
        ? {
            name: form.name,
            kind: 'skeleton' as const,
            isDefault: false,
            lines: form.lines,
            showLogo: true,
            showRetailPrice: true,
            showSpecImage: true,
            showUpgrade: false,
            showParams: false,
            showTotals: true,
            stylePreset: 'compare',
          }
        : {
            ...form,
            kind: 'layout' as const,
            lines: [],
          }
    await saveTemplate(payload, editingId.value)
    ElMessage.success('已保存')
    dialog.value = false
    load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function onDelete(row: QuoteTemplate) {
  await ElMessageBox.confirm(`删除模板「${row.name}」？`, '提示', { type: 'warning' })
  try {
    await deleteTemplate(row.id)
    ElMessage.success('已删除')
    load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

watch(
  () => route.path,
  () => {
    dialog.value = false
  },
)

onMounted(load)
</script>

<template>
  <div>
    <div class="toolbar">
      <el-button type="primary" @click="openCreate">
        {{ isBizPage ? '新建业务模板' : '新建版式模板' }}
      </el-button>
      <span class="hint">
        <template v-if="isBizPage">
          只设计产品组 / 配件明细骨架；报价单外观统一由「版式模板」控制。
        </template>
        <template v-else>
          统一配置 Logo、店名、页脚与显示项，对所有报价单生效。
        </template>
      </span>
    </div>

    <el-table v-loading="loading" :data="list" border>
      <el-table-column prop="name" label="名称" min-width="160" />
      <el-table-column v-if="isBizPage" label="明细行" width="90">
        <template #default="{ row }">{{ row.lines?.length || 0 }}</template>
      </el-table-column>
      <el-table-column v-if="!isBizPage" label="默认" width="80">
        <template #default="{ row }">
          <el-tag v-if="row.isDefault" type="success" size="small">默认</el-tag>
        </template>
      </el-table-column>
      <el-table-column v-if="!isBizPage" prop="shopName" label="店名" width="140" />
      <el-table-column v-if="!isBizPage" prop="shopPhone" label="电话" width="120" />
      <el-table-column v-if="!isBizPage" label="样式" width="100">
        <template #default="{ row }">{{ row.stylePreset === 'simple' ? '简洁' : '对比价' }}</template>
      </el-table-column>
      <el-table-column label="操作" :width="isBizPage ? 220 : 160">
        <template #default="{ row }">
          <el-button v-if="isBizPage" link type="warning" @click="createFromBiz(row)">用此创建报价</el-button>
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="danger" @click="onDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog
      v-model="dialog"
      :title="editingId ? `编辑${pageTitle}` : `新建${pageTitle}`"
      :width="isBizPage ? '780px' : '640px'"
      destroy-on-close
    >
      <el-form label-width="100px">
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>

        <template v-if="!isBizPage">
          <el-form-item label="默认版式"><el-switch v-model="form.isDefault" /></el-form-item>
          <el-form-item label="样式">
            <el-radio-group v-model="form.stylePreset">
              <el-radio-button value="compare">对比价</el-radio-button>
              <el-radio-button value="simple">简洁</el-radio-button>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="Logo">
            <div class="logo-row">
              <el-image v-if="form.logoUrl" :src="form.logoUrl" style="width: 64px; height: 64px" fit="contain" />
              <el-upload :show-file-list="false" :http-request="onUpload as any" accept="image/*">
                <el-button>上传</el-button>
              </el-upload>
            </div>
            <div class="logo-link">
              <el-input v-model="logoLinkDraft" clearable placeholder="粘贴 Logo 图片链接，将上传到报价中心" />
              <el-button type="primary" :loading="logoUploading" @click="applyLogoLink">上传到报价中心</el-button>
            </div>
          </el-form-item>
          <el-form-item label="店名"><el-input v-model="form.shopName" /></el-form-item>
          <el-form-item label="电话"><el-input v-model="form.shopPhone" /></el-form-item>
          <el-form-item label="地址"><el-input v-model="form.shopAddress" /></el-form-item>
          <el-form-item label="副标题"><el-input v-model="form.headerSubtitle" /></el-form-item>
          <el-form-item label="页脚条款"><el-input v-model="form.footerText" type="textarea" :rows="3" /></el-form-item>
          <el-form-item label="显示项">
            <el-checkbox v-model="form.showLogo">Logo</el-checkbox>
            <el-checkbox v-model="form.showRetailPrice">零售价</el-checkbox>
            <el-checkbox v-model="form.showSpecImage">规格图</el-checkbox>
            <el-checkbox v-model="form.showUpgrade">升级款</el-checkbox>
            <el-checkbox v-model="form.showParams">参数</el-checkbox>
            <el-checkbox v-model="form.showTotals">合计</el-checkbox>
          </el-form-item>
        </template>

        <template v-else>
          <el-form-item label="业务明细">
            <div class="lines-toolbar">
              <el-button size="small" type="primary" plain @click="addProductGroupLine()">加产品组</el-button>
              <el-button
                size="small"
                :disabled="!form.lines.length"
                @click="addPartLine(form.lines.length - 1)"
              >
                加配件（末组）
              </el-button>
              <el-button size="small" type="primary" plain @click="loadPreset">载入组装车预设</el-button>
              <span class="hint">加配件插到当前产品组末尾；加产品组在组后新建。</span>
            </div>
            <el-table :data="form.lines" border size="small" class="lines-table">
              <el-table-column label="产品（组）" min-width="120">
                <template #default="{ row, $index }">
                  <el-input
                    v-if="isLineCategoryHead($index)"
                    :model-value="row.category"
                    placeholder="如 车架组"
                    @update:model-value="(v: string) => onLineCategoryInput($index, v)"
                  />
                  <span v-else class="line-muted" />
                </template>
              </el-table-column>
              <el-table-column label="配件" min-width="140">
                <template #default="{ row }"><el-input v-model="row.partName" placeholder="如 车架" /></template>
              </el-table-column>
              <el-table-column label="填写提示" min-width="140">
                <template #default="{ row }"><el-input v-model="row.hint" placeholder="可选" /></template>
              </el-table-column>
              <el-table-column label="操作" width="160">
                <template #default="{ $index }">
                  <el-button link type="primary" @click="addPartLine($index)">加配件</el-button>
                  <el-button v-if="isLineCategoryHead($index)" link type="warning" @click="addProductGroupLine($index)">加产品</el-button>
                  <el-button link type="danger" @click="removeLine($index)">删</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="dialog = false">取消</el-button>
        <el-button type="primary" @click="onSave">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.toolbar { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; flex-wrap: wrap; }
.hint { color: #909399; font-size: 13px; }
.logo-row { display: flex; align-items: center; gap: 12px; }
.logo-link { display: flex; gap: 8px; margin-top: 8px; align-items: center; }
.logo-link .el-input { flex: 1; }
.lines-toolbar { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; flex-wrap: wrap; width: 100%; }
.lines-table { width: 100%; }
.line-muted { display: inline-block; min-height: 20px; color: #c0c4cc; }
</style>
