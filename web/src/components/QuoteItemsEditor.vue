<script setup lang="ts">
import { FullScreen } from '@element-plus/icons-vue'
import { ref } from 'vue'
import type { QuoteItem } from '../api/quote'
import { isQuoteStoredUrl, uploadImage, uploadImageFromUrl } from '../api/upload'
import { isSamePartSpec, isSameProductSpec } from '../utils/quotePrice'
import { ElMessage } from 'element-plus'

const props = withDefaults(
  defineProps<{
    items: QuoteItem[]
    large?: boolean
    showZoomBtn?: boolean
    /** 组车骨架模式：产品/配件固定，填名称规格价图 */
    skeleton?: boolean
    /** 顾客二次编辑：隐藏成本/拿货价，可用自定义上传 */
    publicMode?: boolean
    uploadFn?: (file: File, subdir?: string) => Promise<string>
    uploadFromUrlFn?: (url: string, subdir?: string) => Promise<string>
  }>(),
  { large: false, showZoomBtn: false, skeleton: false, publicMode: false },
)

const emit = defineEmits<{
  zoom: []
  previewImage: [url: string]
}>()

function sameCategory(a: QuoteItem, b: QuoteItem): boolean {
  const ca = (a.category || '').trim()
  const cb = (b.category || '').trim()
  return ca !== '' && ca === cb
}

function isProductHead(idx: number): boolean {
  if (idx <= 0) return true
  const cur = props.items[idx]
  const prev = props.items[idx - 1]
  if (props.skeleton) {
    if ((cur.category || '').trim()) return !sameCategory(cur, prev)
    return true
  }
  return !isSameProductSpec(cur, prev)
}

function isPartHead(idx: number): boolean {
  if (!props.skeleton || idx <= 0) return true
  return !isSamePartSpec(props.items[idx], props.items[idx - 1])
}

function isCategoryHead(idx: number): boolean {
  if (!props.skeleton) return isProductHead(idx)
  if (idx <= 0) return true
  const cur = props.items[idx]
  const prev = props.items[idx - 1]
  if (!(cur.category || '').trim()) return true
  return !sameCategory(cur, prev)
}

/** 当前行所属产品组的 [首行, 末行] */
function findGroupRange(idx: number): [number, number] {
  let head = idx
  while (head > 0 && !isCategoryHead(head)) head -= 1
  const cat = (props.items[head]?.category || '').trim()
  let end = head
  if (!cat) return [head, head]
  for (let i = head + 1; i < props.items.length; i++) {
    if ((props.items[i].category || '').trim() === cat) end = i
    else break
  }
  return [head, end]
}

function renumberSort() {
  props.items.forEach((it, i) => {
    it.sort = (i + 1) * 10
  })
}

function emptyItem(partial?: Partial<QuoteItem>): QuoteItem {
  return {
    sort: (props.items.length + 1) * 10,
    source: 'manual',
    category: '',
    partName: '',
    name: '',
    specLabel: '',
    imageUrl: '',
    qty: 1,
    unit: '件',
    retailPrice: 0,
    costPrice: 0,
    quotePrice: 0,
    upgradeNote: '',
    paramsText: '',
    remark: '',
    ...partial,
  }
}

function addSpecRow(idx: number) {
  const base = props.items[idx]
  if (!base) return
  props.items.splice(
    idx + 1,
    0,
    emptyItem({
      source: base.source || 'manual',
      productId: base.productId ?? null,
      skuId: null,
      category: base.category || '',
      partName: base.partName || '',
      name: base.name,
      imageUrl: '',
      unit: base.unit || '件',
      paramsText: base.paramsText || '',
    }),
  )
  renumberSort()
}

/** 在当前配件后追加一行同配件规格 */
function addPartSpecRow(idx: number) {
  const base = props.items[idx]
  if (!base) return
  if (!(base.partName || '').trim()) {
    ElMessage.warning('请先填写配件名称，再加规格')
    return
  }
  props.items.splice(
    idx + 1,
    0,
    emptyItem({
      source: base.source || 'template',
      category: base.category || '',
      partName: base.partName,
      name: '',
      unit: base.unit || '件',
    }),
  )
  renumberSort()
}

/** 在当前产品组末尾追加一行配件 */
function addPartRow(idx: number) {
  if (!props.items[idx]) return
  const [head, end] = findGroupRange(idx)
  const cat = props.items[head]?.category || ''
  props.items.splice(
    end + 1,
    0,
    emptyItem({
      source: 'template',
      category: cat,
      partName: '',
      name: '',
    }),
  )
  renumberSort()
}

/** 在当前产品组后面新建一个产品组（首行，可填产品组名+名称+配件） */
function addProductGroup(idx: number) {
  const [, end] = findGroupRange(idx)
  props.items.splice(
    end + 1,
    0,
    emptyItem({
      source: 'manual',
      category: '',
      partName: '',
      name: '',
    }),
  )
  renumberSort()
}

function onCategoryInput(idx: number, val: string) {
  if (!isCategoryHead(idx)) return
  const [head, end] = findGroupRange(idx)
  for (let i = head; i <= end; i++) {
    props.items[i].category = val
  }
}

function onPartNameInput(idx: number, val: string) {
  if (!isPartHead(idx)) return
  const old = (props.items[idx].partName || '').trim()
  props.items[idx].partName = val
  if (!old) return
  const cat = (props.items[idx].category || '').trim()
  for (let i = idx + 1; i < props.items.length; i++) {
    const it = props.items[i]
    if ((it.category || '').trim() !== cat) break
    if ((it.partName || '').trim() !== old) break
    it.partName = val
  }
}

function onProductNameInput(idx: number, val: string) {
  const old = props.items[idx].name
  props.items[idx].name = val
  const head = props.items[idx]
  for (let i = idx + 1; i < props.items.length; i++) {
    const it = props.items[i]
    if (props.skeleton) {
      if (sameCategory(head, it)) continue
      break
    }
    if (head.productId && it.productId && Number(it.productId) === Number(head.productId)) {
      it.name = val
      continue
    }
    if (!head.productId && (it.name || '') === (old || '')) {
      it.name = val
      continue
    }
    break
  }
}

function removeRow(idx: number) {
  props.items.splice(idx, 1)
  renumberSort()
}

function moveRow(idx: number, dir: -1 | 1) {
  const j = idx + dir
  if (j < 0 || j >= props.items.length) return
  const tmp = props.items[idx]
  props.items[idx] = props.items[j]
  props.items[j] = tmp
  renumberSort()
}

async function uploadRowImage(idx: number, opt: { file: File }) {
  try {
    props.items[idx].imageUrl = props.uploadFn
      ? await props.uploadFn(opt.file, 'items')
      : await uploadImage(opt.file, 'items')
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

const linkDraft = ref('')
const linkIdx = ref(-1)
const linkUploading = ref(false)

function openLinkPopover(idx: number) {
  linkIdx.value = idx
  linkDraft.value = props.items[idx]?.imageUrl || ''
}

async function applyImageLink() {
  const idx = linkIdx.value
  const raw = linkDraft.value.trim()
  if (idx < 0 || !props.items[idx]) return
  if (!raw) {
    props.items[idx].imageUrl = ''
    return
  }
  if (isQuoteStoredUrl(raw)) {
    props.items[idx].imageUrl = raw
    ElMessage.success('已使用本站图片')
    return
  }
  if (!/^https?:\/\//i.test(raw)) {
    ElMessage.error('请粘贴以 http(s):// 开头的图片链接')
    return
  }
  linkUploading.value = true
  try {
    const url = props.uploadFromUrlFn
      ? await props.uploadFromUrlFn(raw, 'items')
      : await uploadImageFromUrl(raw, 'items')
    props.items[idx].imageUrl = url
    linkDraft.value = url
    ElMessage.success('已上传到报价中心')
  } catch (e) {
    ElMessage.error((e as Error).message || '上传失败')
  } finally {
    linkUploading.value = false
  }
}

function openImage(url?: string) {
  const u = (url || '').trim()
  if (!u) return
  emit('previewImage', u)
}

function formatSupplyAt(raw?: string | null) {
  if (!raw) return ''
  const d = new Date(raw)
  if (Number.isNaN(d.getTime())) return String(raw)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}
</script>

<template>
  <div class="items-editor" :class="{ large }">
    <div v-if="showZoomBtn" class="editor-toolbar">
      <el-button type="primary" plain :icon="FullScreen" @click="emit('zoom')">放大编辑</el-button>
    </div>
    <el-table :data="items" border :size="large ? 'default' : 'small'" row-key="sort" class="items-table">
      <template v-if="skeleton">
        <el-table-column label="产品（组）" :width="large ? 120 : 100">
          <template #default="{ row, $index }">
            <el-input
              v-if="isCategoryHead($index)"
              :model-value="row.category"
              placeholder="如 车架组"
              @update:model-value="(v: string) => onCategoryInput($index, v)"
            />
            <span v-else class="muted-cell" />
          </template>
        </el-table-column>
        <el-table-column label="名称" :min-width="large ? 180 : 140">
          <template #default="{ row, $index }">
            <el-input
              v-if="isCategoryHead($index)"
              :model-value="row.name"
              placeholder="具体型号/套件名"
              @update:model-value="(v: string) => onProductNameInput($index, v)"
            />
            <span v-else class="muted-cell" />
          </template>
        </el-table-column>
        <el-table-column label="配件" :width="large ? 150 : 120">
          <template #default="{ row, $index }">
            <el-input
              v-if="isPartHead($index)"
              :model-value="row.partName"
              placeholder="配件"
              @update:model-value="(v: string) => onPartNameInput($index, v)"
            />
            <div v-else class="spec-cont">└ 同配件规格</div>
          </template>
        </el-table-column>
      </template>
      <template v-else>
        <el-table-column label="产品" :min-width="large ? 220 : 160">
          <template #default="{ row, $index }">
            <el-input
              v-if="isProductHead($index)"
              :model-value="row.name"
              placeholder="产品名称"
              @update:model-value="(v: string) => onProductNameInput($index, v)"
            />
            <div v-else class="spec-cont">└ 同产品规格</div>
          </template>
        </el-table-column>
      </template>

      <el-table-column label="规格" :min-width="large ? 200 : 130">
        <template #default="{ row }"><el-input v-model="row.specLabel" placeholder="规格" type="textarea" :rows="large ? 2 : 1" /></template>
      </el-table-column>
      <el-table-column label="图" :width="large ? 140 : 112">
        <template #default="{ row, $index }">
          <div class="img-cell">
            <button
              v-if="row.imageUrl"
              type="button"
              class="thumb-btn"
              title="预览图片"
              @click="openImage(row.imageUrl)"
            >
              <img :src="row.imageUrl" alt="" class="thumb" />
            </button>
            <div class="img-actions">
              <el-upload :show-file-list="false" :http-request="(o: any) => uploadRowImage($index, o)" accept="image/*">
                <el-button link type="primary">上传</el-button>
              </el-upload>
              <el-popover placement="bottom" :width="340" trigger="click" @show="openLinkPopover($index)">
                <template #reference>
                  <el-button link type="primary">链接</el-button>
                </template>
                <div class="img-url-box">
                  <el-input
                    v-model="linkDraft"
                    type="textarea"
                    :rows="2"
                    clearable
                    placeholder="粘贴图片链接，将自动上传到报价中心"
                    @keyup.enter.exact="applyImageLink"
                  />
                  <div class="img-url-actions">
                    <el-button type="primary" size="small" :loading="linkUploading" @click="applyImageLink">
                      上传到报价中心
                    </el-button>
                  </div>
                  <div class="img-url-tip">外链会转存到报价中心，避免失效</div>
                </div>
              </el-popover>
            </div>
          </div>
        </template>
      </el-table-column>
      <el-table-column v-if="!skeleton" label="数量" :width="large ? 120 : 90">
        <template #default="{ row }">
          <el-input-number v-model="row.qty" :min="0.01" :step="1" controls-position="right" style="width:100%" />
        </template>
      </el-table-column>
      <el-table-column :label="skeleton ? '优惠价' : '报价'" :width="large ? 130 : 110">
        <template #default="{ row }">
          <el-input v-model.number="row.quotePrice" inputmode="decimal" placeholder="0.00" />
        </template>
      </el-table-column>
      <el-table-column label="零售价" :width="large ? 130 : 110">
        <template #default="{ row }">
          <el-input v-model.number="row.retailPrice" inputmode="decimal" placeholder="0.00" />
        </template>
      </el-table-column>
      <el-table-column v-if="!publicMode" label="成本价" :width="large ? 130 : 110">
        <template #default="{ row }">
          <el-input v-model.number="row.costPrice" inputmode="decimal" placeholder="0.00" />
        </template>
      </el-table-column>
      <el-table-column v-if="!publicMode" label="拿货价" :width="large ? 150 : 128">
        <template #default="{ row }">
          <div v-if="row.supplyPriceAt" class="supply-cell">
            <div class="supply-price">¥{{ Number(row.supplyPrice || 0).toFixed(2) }}</div>
            <div class="supply-at">{{ formatSupplyAt(row.supplyPriceAt) }}</div>
          </div>
          <span v-else class="muted-cell">待供货商填</span>
        </template>
      </el-table-column>
      <el-table-column v-if="!publicMode" label="供货商备注" :min-width="large ? 160 : 120">
        <template #default="{ row }">
          <span v-if="(row.supplyRemark || '').trim()" class="supply-remark">{{ row.supplyRemark }}</span>
          <span v-else class="muted-cell">—</span>
        </template>
      </el-table-column>
      <el-table-column v-if="!skeleton" label="小计" :width="large ? 110 : 90" align="right">
        <template #default="{ row }">{{ (Number(row.qty || 0) * Number(row.quotePrice || 0)).toFixed(2) }}</template>
      </el-table-column>
      <el-table-column v-if="!skeleton" label="升级款" :min-width="large ? 160 : 120">
        <template #default="{ row }"><el-input v-model="row.upgradeNote" /></template>
      </el-table-column>
      <el-table-column v-if="!skeleton" label="参数" :min-width="large ? 180 : 140">
        <template #default="{ row }"><el-input v-model="row.paramsText" type="textarea" :rows="large ? 2 : 1" /></template>
      </el-table-column>
      <el-table-column label="备注" :min-width="large ? 160 : 120">
        <template #default="{ row }"><el-input v-model="row.remark" type="textarea" :rows="large ? 2 : 1" /></template>
      </el-table-column>
      <el-table-column label="操作" :width="large ? (skeleton ? 320 : 200) : skeleton ? 268 : 168" fixed="right">
        <template #default="{ $index }">
          <template v-if="skeleton">
            <el-button link type="primary" @click="addPartSpecRow($index)">加规格</el-button>
            <el-button link type="primary" @click="addPartRow($index)">加配件</el-button>
            <el-button v-if="isCategoryHead($index)" link type="warning" @click="addProductGroup($index)">加产品</el-button>
          </template>
          <el-button v-else link type="primary" @click="addSpecRow($index)">加规格</el-button>
          <el-button link @click="moveRow($index, -1)">上</el-button>
          <el-button link @click="moveRow($index, 1)">下</el-button>
          <el-button link type="danger" @click="removeRow($index)">删</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<style scoped>
.editor-toolbar { display: flex; justify-content: flex-end; margin-bottom: 8px; }
.items-editor.large .items-table { --el-font-size-base: 14px; }
.spec-cont { color: #8f959e; font-size: 12px; padding: 0 4px; }
.fixed-cell { font-weight: 600; color: #303133; }
.muted-cell { color: #c0c4cc; display: inline-block; min-height: 20px; }
.img-cell { display: flex; flex-direction: column; align-items: center; gap: 2px; }
.img-actions { display: flex; gap: 2px; flex-wrap: wrap; justify-content: center; }
.img-url-box { display: flex; flex-direction: column; gap: 6px; }
.img-url-actions { display: flex; justify-content: flex-end; }
.img-url-tip { font-size: 12px; color: #8f959e; }
.thumb-btn {
  padding: 0;
  border: 0;
  background: transparent;
  cursor: zoom-in;
  line-height: 0;
  border-radius: 4px;
  overflow: hidden;
  width: 36px;
  height: 36px;
}
.large .thumb-btn { width: 56px; height: 56px; }
.thumb { width: 100%; height: 100%; object-fit: cover; display: block; }
.supply-cell { line-height: 1.25; }
.supply-price { font-weight: 600; color: #2f5d43; }
.supply-at { font-size: 11px; color: #8f959e; margin-top: 2px; }
.supply-remark { white-space: pre-wrap; font-size: 13px; color: #303133; }
</style>
