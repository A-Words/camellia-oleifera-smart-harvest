<script setup lang="ts">
import type {
  CreatePlotRequest,
  CreateTreeRequest,
  Plot,
  PlotListResponse,
  TreeArchive,
  TreeListResponse,
  TreeStatus,
  UpdateWorkOrderRequest,
  WorkOrder,
  WorkOrderListResponse,
  WorkOrderStatus
} from '~/types/operations'
import { useGatewayBase } from '~/composables/useGatewayBase'
import { useHarvestContext } from '~/composables/useHarvestContext'
import {
  formatDecisionTimestamp,
  formatTreeStatus,
  formatWorkOrderStatus
} from '~/utils/decision-presenter'

useSeoMeta({
  title: '作业',
  description: '维护地块和树木档案，并推进树级作业单状态流转。'
})

const gatewayBase = useGatewayBase()
const harvestContext = useHarvestContext()

const selectedPlotId = harvestContext.selectedPlotId

const plots = ref<Plot[]>([])
const trees = ref<TreeArchive[]>([])
const workOrders = ref<WorkOrder[]>([])
const statusFilter = ref<'all' | WorkOrderStatus>('all')
const skipReasonDrafts = ref<Record<string, string>>({})

const plotForm = reactive<CreatePlotRequest>({
  name: '',
  code: '',
  row_count: 1,
  notes: ''
})

const treeForm = reactive<CreateTreeRequest>({
  plot_id: '',
  tree_code: '',
  row_index: 1,
  col_index: 1,
  x: 0,
  y: 0,
  status: 'active'
})

const isPlotLoading = ref(false)
const isTreeLoading = ref(false)
const isWorkOrderLoading = ref(false)
const isCreatingPlot = ref(false)
const isCreatingTree = ref(false)
const updatingTreeId = ref('')
const updatingWorkOrderId = ref('')

const pageError = ref('')
const pageMessage = ref('')

const plotOptions = computed(() =>
  plots.value.map((plot) => ({
    label: `${plot.name} (${plot.code})`,
    value: plot.plot_id
  }))
)

const workOrderSummary = computed(() =>
  workOrders.value.reduce<Record<WorkOrderStatus, number>>((accumulator, item) => {
    accumulator[item.status]++
    return accumulator
  }, {
    pending: 0,
    in_progress: 0,
    completed: 0,
    skipped: 0
  })
)

async function loadPlots() {
  isPlotLoading.value = true
  pageError.value = ''
  try {
    const response = await $fetch<PlotListResponse>(`${gatewayBase.value}/v1/operations/plots`)
    plots.value = response.items || []
    if (!plots.value.length) {
      harvestContext.setSelectedPlot('')
      return
    }

    const stillExists = plots.value.some((plot) => plot.plot_id === selectedPlotId.value)
    if (!stillExists) {
      harvestContext.setSelectedPlot(plots.value[0]!.plot_id)
    }
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : '读取地块失败。'
  } finally {
    isPlotLoading.value = false
  }
}

async function loadTrees(plotId: string) {
  if (!plotId) {
    trees.value = []
    return
  }

  isTreeLoading.value = true
  try {
    const response = await $fetch<TreeListResponse>(`${gatewayBase.value}/v1/operations/plots/${plotId}/trees`)
    trees.value = response.items || []
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : '读取树档案失败。'
  } finally {
    isTreeLoading.value = false
  }
}

async function loadWorkOrders(plotId: string) {
  if (!plotId) {
    workOrders.value = []
    return
  }

  isWorkOrderLoading.value = true
  try {
    const response = await $fetch<WorkOrderListResponse>(`${gatewayBase.value}/v1/operations/work-orders`, {
      query: {
        plot_id: plotId,
        status: statusFilter.value === 'all' ? undefined : statusFilter.value
      }
    })
    workOrders.value = response.items || []
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : '读取作业单失败。'
  } finally {
    isWorkOrderLoading.value = false
  }
}

async function handleCreatePlot() {
  isCreatingPlot.value = true
  pageError.value = ''
  pageMessage.value = ''
  try {
    const plot = await $fetch<Plot>(`${gatewayBase.value}/v1/operations/plots`, {
      method: 'POST',
      body: plotForm
    })
    plots.value = [...plots.value, plot]
    harvestContext.setSelectedPlot(plot.plot_id)
    plotForm.name = ''
    plotForm.code = ''
    plotForm.row_count = 1
    plotForm.notes = ''
    pageMessage.value = '已创建新的地块档案。'
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : '创建地块失败。'
  } finally {
    isCreatingPlot.value = false
  }
}

async function handleCreateTree() {
  if (!selectedPlotId.value) {
    pageError.value = '请先选择地块，再创建树档案。'
    return
  }

  isCreatingTree.value = true
  pageError.value = ''
  pageMessage.value = ''
  try {
    const tree = await $fetch<TreeArchive>(`${gatewayBase.value}/v1/operations/trees`, {
      method: 'POST',
      body: {
        ...treeForm,
        plot_id: selectedPlotId.value
      }
    })
    trees.value = [...trees.value, tree].sort((left, right) =>
      left.row_index - right.row_index ||
      left.col_index - right.col_index ||
      left.tree_code.localeCompare(right.tree_code)
    )
    treeForm.tree_code = ''
    treeForm.row_index = 1
    treeForm.col_index = 1
    treeForm.x = 0
    treeForm.y = 0
    treeForm.status = 'active'
    pageMessage.value = '已创建新的树档案。'
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : '创建树档案失败。'
  } finally {
    isCreatingTree.value = false
  }
}

async function handleToggleTreeStatus(tree: TreeArchive) {
  updatingTreeId.value = tree.tree_id
  pageError.value = ''
  pageMessage.value = ''
  try {
    const nextStatus: TreeStatus = tree.status === 'active' ? 'disabled' : 'active'
    const updated = await $fetch<TreeArchive>(`${gatewayBase.value}/v1/operations/trees/${tree.tree_id}`, {
      method: 'PATCH',
      body: { status: nextStatus }
    })
    trees.value = trees.value.map((item) => item.tree_id === updated.tree_id ? updated : item)
    pageMessage.value = `树档案 ${updated.tree_code} 已切换为${formatTreeStatus(updated.status)}。`
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : '更新树状态失败。'
  } finally {
    updatingTreeId.value = ''
  }
}

async function handleUpdateWorkOrder(workOrder: WorkOrder, status: WorkOrderStatus) {
  updatingWorkOrderId.value = workOrder.work_order_id
  pageError.value = ''
  pageMessage.value = ''
  try {
    const request: UpdateWorkOrderRequest = {
      status
    }
    if (status === 'skipped') {
      request.skip_reason_note = skipReasonDrafts.value[workOrder.work_order_id] || '现场人工判定跳过。'
    }

    const updated = await $fetch<WorkOrder>(`${gatewayBase.value}/v1/operations/work-orders/${workOrder.work_order_id}`, {
      method: 'PATCH',
      body: request
    })
    workOrders.value = workOrders.value.map((item) => item.work_order_id === updated.work_order_id ? updated : item)
    pageMessage.value = `作业单 ${updated.tree_code} 已更新为${formatWorkOrderStatus(updated.status)}。`
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : '更新作业单失败。'
  } finally {
    updatingWorkOrderId.value = ''
  }
}

watch(selectedPlotId, async (plotId) => {
  treeForm.plot_id = plotId
  await loadTrees(plotId)
  await loadWorkOrders(plotId)
}, { immediate: true })

watch(statusFilter, async () => {
  await loadWorkOrders(selectedPlotId.value)
})

onMounted(async () => {
  await loadPlots()
})
</script>

<template>
  <UContainer class="py-8 sm:py-12">
    <div class="space-y-6">
      <section class="space-y-2">
        <p class="text-xs uppercase tracking-widest text-muted">
          Operations
        </p>
        <h1 class="text-2xl font-semibold text-highlighted sm:text-3xl">
          地块建档与作业执行
        </h1>
        <p class="text-sm text-toned sm:text-base">
          在这里维护地块和树木档案，查看树级作业单，并推进 `pending -> in_progress -> completed / skipped` 状态流。
        </p>
      </section>

      <UAlert
        v-if="pageError"
        color="error"
        variant="subtle"
        icon="i-lucide-triangle-alert"
        title="作业异常"
        :description="pageError"
      />

      <UAlert
        v-else-if="pageMessage"
        color="success"
        variant="subtle"
        icon="i-lucide-badge-check"
        title="处理进展"
        :description="pageMessage"
      />

      <div class="grid grid-cols-1 gap-6 xl:grid-cols-[0.9fr_0.95fr_1.15fr]">
        <div class="space-y-6">
          <UCard variant="outline" :ui="{ body: 'p-5 sm:p-6' }">
            <template #header>
              <div>
                <h2 class="text-base font-semibold text-highlighted">
                  地块档案
                </h2>
                <p class="mt-1 text-xs text-muted">
                  建立整块地上下文后，识别页与决策页都会复用这里的地块选择。
                </p>
              </div>
            </template>

            <div class="space-y-3">
              <input v-model="plotForm.name" class="w-full rounded-lg border border-default bg-default px-3 py-2 text-sm" placeholder="地块名称" />
              <input v-model="plotForm.code" class="w-full rounded-lg border border-default bg-default px-3 py-2 text-sm" placeholder="地块编码" />
              <input
                :value="String(plotForm.row_count)"
                class="w-full rounded-lg border border-default bg-default px-3 py-2 text-sm"
                placeholder="行数"
                @input="plotForm.row_count = Number(($event.target as HTMLInputElement).value || 0)"
              />
              <textarea
                v-model="plotForm.notes"
                class="min-h-24 w-full rounded-lg border border-default bg-default px-3 py-2 text-sm"
                placeholder="地块备注"
              />
              <UButton
                color="primary"
                icon="i-lucide-plus"
                :loading="isCreatingPlot"
                label="创建地块"
                @click="handleCreatePlot"
              />

              <USelect
                :model-value="selectedPlotId"
                :items="plotOptions"
                value-key="value"
                label-key="label"
                :loading="isPlotLoading"
                icon="i-lucide-map"
                placeholder="选择当前地块"
                @update:model-value="(value) => harvestContext.setSelectedPlot((value as string) || '')"
              />
            </div>
          </UCard>

          <UCard variant="outline" :ui="{ body: 'p-5 sm:p-6' }">
            <template #header>
              <div>
                <h2 class="text-base font-semibold text-highlighted">
                  地块总览
                </h2>
                <p class="mt-1 text-xs text-muted">
                  当前会话只围绕一个选中地块展开识别、决策与作业执行。
                </p>
              </div>
            </template>

            <div v-if="plots.length" class="space-y-3">
              <div
                v-for="plot in plots"
                :key="plot.plot_id"
                class="rounded-lg border px-4 py-4 text-sm"
                :class="selectedPlotId === plot.plot_id ? 'border-primary bg-primary/5' : 'border-default bg-default'"
              >
                <p class="font-medium text-highlighted">
                  {{ plot.name }}
                </p>
                <p class="mt-1 text-xs text-muted">
                  {{ plot.code }} · {{ plot.row_count }} 行
                </p>
                <p class="mt-1 text-xs text-muted">
                  {{ plot.notes || '暂无备注' }}
                </p>
              </div>
            </div>

            <UAlert
              v-else
              color="neutral"
              variant="subtle"
              icon="i-lucide-map-off"
              title="暂无地块"
              description="先创建至少一个地块，才能继续建树和执行作业。"
            />
          </UCard>
        </div>

        <div class="space-y-6">
          <UCard variant="outline" :ui="{ body: 'p-5 sm:p-6' }">
            <template #header>
              <div>
                <h2 class="text-base font-semibold text-highlighted">
                  树木档案
                </h2>
                <p class="mt-1 text-xs text-muted">
                  每棵树记录行列和坐标，供地块级路线与树级作业单复用。
                </p>
              </div>
            </template>

            <div class="space-y-3">
              <input v-model="treeForm.tree_code" class="w-full rounded-lg border border-default bg-default px-3 py-2 text-sm" placeholder="树木编码" />
              <div class="grid grid-cols-2 gap-3">
                <input
                  :value="String(treeForm.row_index)"
                  class="w-full rounded-lg border border-default bg-default px-3 py-2 text-sm"
                  placeholder="行号"
                  @input="treeForm.row_index = Number(($event.target as HTMLInputElement).value || 0)"
                />
                <input
                  :value="String(treeForm.col_index)"
                  class="w-full rounded-lg border border-default bg-default px-3 py-2 text-sm"
                  placeholder="列号"
                  @input="treeForm.col_index = Number(($event.target as HTMLInputElement).value || 0)"
                />
              </div>
              <div class="grid grid-cols-2 gap-3">
                <input
                  :value="String(treeForm.x)"
                  class="w-full rounded-lg border border-default bg-default px-3 py-2 text-sm"
                  placeholder="X 坐标"
                  @input="treeForm.x = Number(($event.target as HTMLInputElement).value || 0)"
                />
                <input
                  :value="String(treeForm.y)"
                  class="w-full rounded-lg border border-default bg-default px-3 py-2 text-sm"
                  placeholder="Y 坐标"
                  @input="treeForm.y = Number(($event.target as HTMLInputElement).value || 0)"
                />
              </div>
              <select v-model="treeForm.status" class="w-full rounded-lg border border-default bg-default px-3 py-2 text-sm">
                <option value="active">active</option>
                <option value="disabled">disabled</option>
              </select>
              <UButton
                color="primary"
                icon="i-lucide-tree-pine"
                :disabled="!selectedPlotId"
                :loading="isCreatingTree"
                label="创建树档案"
                @click="handleCreateTree"
              />
            </div>
          </UCard>

          <UCard variant="outline" :ui="{ body: 'p-5 sm:p-6' }">
            <template #header>
              <div class="flex items-center justify-between gap-3">
                <div>
                  <h2 class="text-base font-semibold text-highlighted">
                    当前地块树清单
                  </h2>
                  <p class="mt-1 text-xs text-muted">
                    可随时停用树档案，停用后它不会进入新的地块级路线。
                  </p>
                </div>
                <UBadge :color="trees.length ? 'success' : 'neutral'" variant="soft">
                  {{ trees.length }} 棵
                </UBadge>
              </div>
            </template>

            <div v-if="trees.length" class="space-y-3">
              <div
                v-for="tree in trees"
                :key="tree.tree_id"
                class="rounded-lg border border-default bg-default px-4 py-4 text-sm text-default"
              >
                <div class="flex items-start justify-between gap-3">
                  <div>
                    <p class="font-medium text-highlighted">
                      {{ tree.tree_code }}
                    </p>
                    <p class="mt-1 text-xs text-muted">
                      第 {{ tree.row_index }} 行第 {{ tree.col_index }} 列 · 坐标 ({{ tree.x }}, {{ tree.y }})
                    </p>
                    <p class="mt-1 text-xs text-muted">
                      当前状态：{{ formatTreeStatus(tree.status) }}
                    </p>
                  </div>
                  <UButton
                    size="xs"
                    color="neutral"
                    variant="outline"
                    :loading="updatingTreeId === tree.tree_id"
                    :label="tree.status === 'active' ? '停用' : '启用'"
                    @click="handleToggleTreeStatus(tree)"
                  />
                </div>
              </div>
            </div>

            <UAlert
              v-else
              color="neutral"
              variant="subtle"
              icon="i-lucide-tree-pine"
              title="当前地块还没有树档案"
              description="先创建至少一棵树，再回到识别页绑定观测对象。"
            />
          </UCard>
        </div>

        <div class="space-y-6">
          <UCard variant="outline" :ui="{ body: 'p-5 sm:p-6' }">
            <template #header>
              <div class="flex items-center justify-between gap-3">
                <div>
                  <h2 class="text-base font-semibold text-highlighted">
                    树级作业单
                  </h2>
                  <p class="mt-1 text-xs text-muted">
                    `/decision` 生成的树级作业单会汇总到这里，可继续推进执行状态。
                  </p>
                </div>
                <UBadge :color="workOrders.length ? 'success' : 'neutral'" variant="soft">
                  {{ workOrders.length }} 张
                </UBadge>
              </div>
            </template>

            <div class="space-y-4">
              <select v-model="statusFilter" class="w-full rounded-lg border border-default bg-default px-3 py-2 text-sm">
                <option value="all">全部状态</option>
                <option value="pending">pending</option>
                <option value="in_progress">in_progress</option>
                <option value="completed">completed</option>
                <option value="skipped">skipped</option>
              </select>

              <div class="grid grid-cols-2 gap-3 text-sm">
                <div class="rounded-lg border border-default bg-default px-4 py-3">
                  <p class="text-xs text-muted">待执行</p>
                  <p class="mt-1 text-lg font-semibold text-highlighted">{{ workOrderSummary.pending }}</p>
                </div>
                <div class="rounded-lg border border-default bg-default px-4 py-3">
                  <p class="text-xs text-muted">执行中</p>
                  <p class="mt-1 text-lg font-semibold text-highlighted">{{ workOrderSummary.in_progress }}</p>
                </div>
                <div class="rounded-lg border border-default bg-default px-4 py-3">
                  <p class="text-xs text-muted">已完成</p>
                  <p class="mt-1 text-lg font-semibold text-highlighted">{{ workOrderSummary.completed }}</p>
                </div>
                <div class="rounded-lg border border-default bg-default px-4 py-3">
                  <p class="text-xs text-muted">已跳过</p>
                  <p class="mt-1 text-lg font-semibold text-highlighted">{{ workOrderSummary.skipped }}</p>
                </div>
              </div>

              <div v-if="workOrders.length" class="space-y-3">
                <div
                  v-for="item in workOrders"
                  :key="item.work_order_id"
                  class="rounded-lg border border-default bg-default px-4 py-4 text-sm text-default"
                >
                  <p class="font-medium text-highlighted">
                    {{ item.tree_code }}
                  </p>
                  <p class="mt-1 text-xs text-muted">
                    {{ formatWorkOrderStatus(item.status) }} · 作业单 {{ item.work_order_id }}
                  </p>
                  <p class="mt-1 text-xs text-muted">
                    创建于 {{ formatDecisionTimestamp(item.created_at) }}
                  </p>
                  <p class="mt-1 text-xs text-muted">
                    started_at: {{ item.started_at ? formatDecisionTimestamp(item.started_at) : '未开始' }}
                  </p>
                  <p class="mt-1 text-xs text-muted">
                    completed_at: {{ item.completed_at ? formatDecisionTimestamp(item.completed_at) : '未结束' }}
                  </p>

                  <div class="mt-3 flex flex-wrap gap-2">
                    <UButton
                      size="xs"
                      color="primary"
                      variant="outline"
                      :loading="updatingWorkOrderId === item.work_order_id"
                      label="进入执行"
                      @click="handleUpdateWorkOrder(item, 'in_progress')"
                    />
                    <UButton
                      size="xs"
                      color="success"
                      variant="outline"
                      :loading="updatingWorkOrderId === item.work_order_id"
                      label="标记完成"
                      @click="handleUpdateWorkOrder(item, 'completed')"
                    />
                    <UButton
                      size="xs"
                      color="warning"
                      variant="outline"
                      :loading="updatingWorkOrderId === item.work_order_id"
                      label="标记跳过"
                      @click="handleUpdateWorkOrder(item, 'skipped')"
                    />
                  </div>

                  <textarea
                    v-model="skipReasonDrafts[item.work_order_id]"
                    class="mt-3 min-h-20 w-full rounded-lg border border-default bg-default px-3 py-2 text-sm"
                    placeholder="若需要跳过，请先填写 skip_reason_note。"
                  />
                </div>
              </div>

              <UAlert
                v-else
                color="neutral"
                variant="subtle"
                icon="i-lucide-briefcase"
                title="当前地块暂无作业单"
                description="先到决策页生成地块计划，并下发树级作业单，当前列表才会出现执行任务。"
              />
            </div>
          </UCard>
        </div>
      </div>
    </div>
  </UContainer>
</template>
