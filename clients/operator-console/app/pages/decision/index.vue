<script setup lang="ts">
import type {
  DecisionPlan,
  DecisionPlanListResponse,
  DecisionZone,
  PickSequenceItem,
  PlanTreeSequenceItem,
  TreeObservation,
  TreeOverride,
  TreeRecommendation,
  ZonePriority
} from '~/types/decision'
import type {
  Plot,
  PlotListResponse,
  TreeArchive,
  TreeListResponse,
  WorkOrder,
  WorkOrderListResponse
} from '~/types/operations'
import { useDecisionSnapshot } from '~/composables/useDecisionSnapshot'
import { useGatewayBase } from '~/composables/useGatewayBase'
import { useHarvestContext } from '~/composables/useHarvestContext'
import {
  formatConfidencePercent,
  formatDecisionSkipReason,
  formatDecisionTimestamp,
  formatDecisionZoneLabel,
  formatTreeRecommendationStatus,
  formatWorkOrderStatus
} from '~/utils/decision-presenter'

useSeoMeta({
  title: '决策',
  description: '按地块生成树级路线、树内区域优先级与采摘顺序，并支持人工调整后下发作业单。'
})

type PickRow = PickSequenceItem & {
  manualSkipped: boolean
}

const gatewayBase = useGatewayBase()
const decisionSnapshot = useDecisionSnapshot()
const harvestContext = useHarvestContext()

const selectedPlotId = harvestContext.selectedPlotId
const selectedTreeId = harvestContext.selectedTreeId

const plots = ref<Plot[]>([])
const trees = ref<TreeArchive[]>([])
const plans = ref<DecisionPlan[]>([])
const currentPlanId = ref('')
const editableTreeSequence = ref<string[]>([])
const treeOverrideState = ref<Record<string, TreeOverride>>({})
const createdWorkOrders = ref<WorkOrder[]>([])

const isPlotLoading = ref(false)
const isTreeLoading = ref(false)
const isPlanLoading = ref(false)
const isArchivingObservation = ref(false)
const isGeneratingPlan = ref(false)
const isSavingOverride = ref(false)
const isCreatingWorkOrders = ref(false)

const pageError = ref('')
const pageMessage = ref('')

const plotOptions = computed(() =>
  plots.value.map((plot) => ({
    label: `${plot.name} (${plot.code})`,
    value: plot.plot_id
  }))
)

const planOptions = computed(() =>
  plans.value.map((plan) => ({
    label: `${formatDecisionTimestamp(plan.generated_at)} · ${plan.summary.ready_trees}/${plan.summary.total_trees} 棵已纳入计划`,
    value: plan.plan_id
  }))
)

const selectedPlot = computed(() =>
  plots.value.find((plot) => plot.plot_id === selectedPlotId.value) || null
)

const currentPlan = computed(() =>
  plans.value.find((plan) => plan.plan_id === currentPlanId.value) || plans.value[0] || null
)

const treeById = computed(() =>
  trees.value.reduce<Record<string, TreeArchive>>((accumulator, tree) => {
    accumulator[tree.tree_id] = tree
    return accumulator
  }, {})
)

const routeByTreeId = computed(() =>
  (currentPlan.value?.tree_sequence || []).reduce<Record<string, PlanTreeSequenceItem>>((accumulator, item) => {
    accumulator[item.tree_id] = item
    return accumulator
  }, {})
)

const effectiveTreeSequenceIds = computed(() => {
  const plan = currentPlan.value
  if (!plan) {
    return []
  }

  const knownTreeIds = new Set(plan.tree_sequence.map((item) => item.tree_id))
  const fromState = editableTreeSequence.value.filter((treeId) => knownTreeIds.has(treeId))
  const missing = plan.tree_sequence
    .map((item) => item.tree_id)
    .filter((treeId) => !fromState.includes(treeId))

  return fromState.length ? [...fromState, ...missing] : [...missing]
})

const orderedTreeSequence = computed(() =>
  effectiveTreeSequenceIds.value
    .map((treeId, index) => {
      const item = routeByTreeId.value[treeId]
      if (!item) {
        return null
      }
      return {
        ...item,
        priority_order: index + 1
      } satisfies PlanTreeSequenceItem
    })
    .filter((item): item is PlanTreeSequenceItem => Boolean(item))
)

const currentRecommendation = computed(() => {
  const plan = currentPlan.value
  if (!plan) {
    return null
  }
  return plan.tree_recommendations.find((item) => item.tree_id === selectedTreeId.value) || plan.tree_recommendations[0] || null
})

const pendingRecommendations = computed(() =>
  currentPlan.value?.tree_recommendations.filter((item) => item.status === 'pending_observation') || []
)

function cloneOverride(override: TreeOverride): TreeOverride {
  return {
    tree_id: override.tree_id,
    zone_order: [...override.zone_order],
    pick_sequence_detection_indices: [...override.pick_sequence_detection_indices],
    manual_skipped_detection_indices: [...override.manual_skipped_detection_indices]
  }
}

function findRecommendation(treeId: string) {
  return currentPlan.value?.tree_recommendations.find((item) => item.tree_id === treeId) || null
}

function buildDefaultOverride(recommendation: TreeRecommendation): TreeOverride {
  return {
    tree_id: recommendation.tree_id,
    zone_order: recommendation.zone_priorities.map((item) => item.zone),
    pick_sequence_detection_indices: recommendation.pick_sequence.map((item) => item.detection_index),
    manual_skipped_detection_indices: []
  }
}

function getEffectiveOverride(treeId: string): TreeOverride | null {
  const recommendation = findRecommendation(treeId)
  if (!recommendation) {
    return null
  }
  return treeOverrideState.value[treeId] ? cloneOverride(treeOverrideState.value[treeId]!) : buildDefaultOverride(recommendation)
}

function getEffectiveZoneOrder(recommendation: TreeRecommendation) {
  const override = getEffectiveOverride(recommendation.tree_id)
  return override?.zone_order.length
    ? override.zone_order
    : recommendation.zone_priorities.map((item) => item.zone)
}

function getEffectivePickOrder(recommendation: TreeRecommendation) {
  const override = getEffectiveOverride(recommendation.tree_id)
  return override?.pick_sequence_detection_indices.length
    ? override.pick_sequence_detection_indices
    : recommendation.pick_sequence.map((item) => item.detection_index)
}

function getManualSkippedDetectionIndices(treeId: string) {
  return getEffectiveOverride(treeId)?.manual_skipped_detection_indices || []
}

function mutateOverride(treeId: string, updater: (next: TreeOverride) => void) {
  const current = getEffectiveOverride(treeId)
  if (!current) {
    return
  }
  updater(current)
  treeOverrideState.value = {
    ...treeOverrideState.value,
    [treeId]: current
  }
}

function moveItem<T>(items: T[], index: number, delta: number) {
  const nextIndex = index + delta
  if (index < 0 || nextIndex < 0 || nextIndex >= items.length) {
    return items
  }
  const next = [...items]
  const [item] = next.splice(index, 1)
  if (item === undefined) {
    return items
  }
  next.splice(nextIndex, 0, item)
  return next
}

const currentDisplayedZonePriorities = computed(() => {
  const recommendation = currentRecommendation.value
  if (!recommendation) {
    return [] as ZonePriority[]
  }

  const zonePriorityByZone = recommendation.zone_priorities.reduce<Record<string, ZonePriority>>((accumulator, item) => {
    accumulator[item.zone] = item
    return accumulator
  }, {})

  return getEffectiveZoneOrder(recommendation)
    .map((zone, index) => {
      const item = zonePriorityByZone[zone]
      if (!item) {
        return null
      }
      return {
        ...item,
        order: index + 1
      } satisfies ZonePriority
    })
    .filter((item): item is ZonePriority => Boolean(item))
})

const currentDisplayedPickRows = computed(() => {
  const recommendation = currentRecommendation.value
  if (!recommendation) {
    return [] as PickRow[]
  }

  const itemByDetectionIndex = recommendation.pick_sequence.reduce<Record<number, PickSequenceItem>>((accumulator, item) => {
    accumulator[item.detection_index] = item
    return accumulator
  }, {})

  const orderedDetectionIndices = getEffectivePickOrder(recommendation)
  const manualSkipped = new Set(getManualSkippedDetectionIndices(recommendation.tree_id))
  const rows: PickRow[] = []
  const used = new Set<number>()

  for (const detectionIndex of orderedDetectionIndices) {
    const item = itemByDetectionIndex[detectionIndex]
    if (!item) {
      continue
    }
    rows.push({
      ...item,
      order: rows.length + 1,
      manualSkipped: manualSkipped.has(detectionIndex)
    })
    used.add(detectionIndex)
  }

  for (const item of recommendation.pick_sequence) {
    if (used.has(item.detection_index)) {
      continue
    }
    rows.push({
      ...item,
      order: rows.length + 1,
      manualSkipped: manualSkipped.has(item.detection_index)
    })
  }

  return rows
})

const currentPendingManualSkipCount = computed(() =>
  currentDisplayedPickRows.value.filter((item) => item.manualSkipped).length
)

function initializeEditableState(plan: DecisionPlan | null) {
  if (!plan) {
    editableTreeSequence.value = []
    treeOverrideState.value = {}
    return
  }

  editableTreeSequence.value = plan.manual_override_state.tree_sequence.length
    ? [...plan.manual_override_state.tree_sequence]
    : plan.tree_sequence.map((item) => item.tree_id)

  const nextOverrides: Record<string, TreeOverride> = {}
  for (const recommendation of plan.tree_recommendations) {
    const savedOverride = plan.manual_override_state.tree_overrides.find((item) => item.tree_id === recommendation.tree_id)
    nextOverrides[recommendation.tree_id] = savedOverride
      ? cloneOverride(savedOverride)
      : buildDefaultOverride(recommendation)
  }
  treeOverrideState.value = nextOverrides
}

function syncSelectedTree(plan: DecisionPlan | null) {
  if (!plan) {
    return
  }

  const exists = plan.tree_recommendations.some((item) => item.tree_id === selectedTreeId.value)
  if (exists) {
    return
  }

  const fallbackTreeId =
    plan.tree_sequence[0]?.tree_id ||
    plan.tree_recommendations[0]?.tree_id ||
    trees.value[0]?.tree_id ||
    ''

  if (fallbackTreeId) {
    harvestContext.setSelectedTree(fallbackTreeId)
  }
}

async function loadPlots() {
  isPlotLoading.value = true
  pageError.value = ''
  try {
    const response = await $fetch<PlotListResponse>(`${gatewayBase.value}/v1/operations/plots`)
    plots.value = response.items || []

    if (!plots.value.length) {
      harvestContext.setSelectedPlot('')
      harvestContext.setSelectedTree('')
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

    const stillExists = trees.value.some((tree) => tree.tree_id === selectedTreeId.value)
    if (!stillExists) {
      const fallbackTree = trees.value.find((tree) => tree.status === 'active') || trees.value[0] || null
      harvestContext.setSelectedTree(fallbackTree?.tree_id || '')
    }
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : '读取树档案失败。'
  } finally {
    isTreeLoading.value = false
  }
}

async function loadPlans(plotId: string) {
  if (!plotId) {
    plans.value = []
    currentPlanId.value = ''
    initializeEditableState(null)
    return
  }

  isPlanLoading.value = true
  try {
    const response = await $fetch<DecisionPlanListResponse>(`${gatewayBase.value}/v1/decision/plans`, {
      query: { plot_id: plotId }
    })
    plans.value = response.items || []
    if (!plans.value.length) {
      currentPlanId.value = ''
      initializeEditableState(null)
      return
    }

    if (!plans.value.some((plan) => plan.plan_id === currentPlanId.value)) {
      currentPlanId.value = plans.value[0]!.plan_id
    }
    initializeEditableState(currentPlan.value)
    syncSelectedTree(currentPlan.value)
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : '读取地块决策计划失败。'
  } finally {
    isPlanLoading.value = false
  }
}

async function maybeArchivePendingObservation() {
  if (!decisionSnapshot.hasPendingObservation.value || !decisionSnapshot.snapshot.value || !selectedTreeId.value) {
    return
  }

  isArchivingObservation.value = true
  pageError.value = ''
  try {
    await $fetch<TreeObservation>(`${gatewayBase.value}/v1/decision/observations`, {
      method: 'POST',
      body: {
        tree_id: selectedTreeId.value,
        captured_at: new Date().toISOString(),
        ...decisionSnapshot.snapshot.value
      }
    })
    decisionSnapshot.markArchived()
    pageMessage.value = '已自动保存当前树的最新识别记录，可直接生成地块计划。'
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : '保存当前树的识别记录失败。'
  } finally {
    isArchivingObservation.value = false
  }
}

async function refreshPage(plotId: string) {
  if (!plotId) {
    plans.value = []
    trees.value = []
    currentPlanId.value = ''
    initializeEditableState(null)
    return
  }

  await loadTrees(plotId)
  await maybeArchivePendingObservation()
  await loadPlans(plotId)
}

async function handleGeneratePlan() {
  if (!selectedPlotId.value) {
    pageError.value = '请先选择地块，再生成整块地决策计划。'
    return
  }

  pageError.value = ''
  pageMessage.value = ''
  await maybeArchivePendingObservation()
  if (pageError.value) {
    return
  }

  isGeneratingPlan.value = true
  try {
    const plan = await $fetch<DecisionPlan>(`${gatewayBase.value}/v1/decision/plans`, {
      method: 'POST',
      body: { plot_id: selectedPlotId.value }
    })
    plans.value = [plan, ...plans.value.filter((item) => item.plan_id !== plan.plan_id)]
    currentPlanId.value = plan.plan_id
    initializeEditableState(plan)
    syncSelectedTree(plan)
    pageMessage.value = '已生成新的地块决策计划。'
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : '生成地块计划失败。'
  } finally {
    isGeneratingPlan.value = false
  }
}

async function handleSaveOverride() {
  const plan = currentPlan.value
  if (!plan) {
    pageError.value = '当前没有可保存的决策计划。'
    return
  }

  isSavingOverride.value = true
  pageError.value = ''
  pageMessage.value = ''
  try {
    const request = {
      tree_sequence: [...effectiveTreeSequenceIds.value],
      tree_overrides: Object.values(treeOverrideState.value)
        .filter((item) => plan.tree_recommendations.some((recommendation) => recommendation.tree_id === item.tree_id))
        .map((item) => cloneOverride(item))
    }

    const nextPlan = await $fetch<DecisionPlan>(`${gatewayBase.value}/v1/decision/plans/${plan.plan_id}`, {
      method: 'PATCH',
      body: request
    })

    plans.value = plans.value.map((item) => item.plan_id === nextPlan.plan_id ? nextPlan : item)
    currentPlanId.value = nextPlan.plan_id
    initializeEditableState(nextPlan)
    syncSelectedTree(nextPlan)
    pageMessage.value = '人工调整已保存到当前计划。'
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : '保存人工调整失败。'
  } finally {
    isSavingOverride.value = false
  }
}

async function handleCreateWorkOrders() {
  const plan = currentPlan.value
  if (!plan) {
    pageError.value = '请先生成或选择一份决策计划。'
    return
  }

  isCreatingWorkOrders.value = true
  pageError.value = ''
  pageMessage.value = ''
  try {
    const response = await $fetch<WorkOrderListResponse>(`${gatewayBase.value}/v1/operations/work-orders`, {
      method: 'POST',
      body: { plan_id: plan.plan_id }
    })
    createdWorkOrders.value = response.items || []
    pageMessage.value = response.items.length
      ? `已为当前计划生成 ${response.items.length} 张树级作业单。`
      : '当前计划里还没有可下发的树级任务。'
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : '生成树级作业单失败。'
  } finally {
    isCreatingWorkOrders.value = false
  }
}

function handleSelectPlan(planId: string) {
  currentPlanId.value = planId
  initializeEditableState(currentPlan.value)
  syncSelectedTree(currentPlan.value)
}

function moveTree(treeId: string, delta: number) {
  const ids = [...effectiveTreeSequenceIds.value]
  const index = ids.indexOf(treeId)
  editableTreeSequence.value = moveItem(ids, index, delta)
}

function moveZone(zone: DecisionZone, delta: number) {
  const recommendation = currentRecommendation.value
  if (!recommendation) {
    return
  }

  mutateOverride(recommendation.tree_id, (next) => {
    if (!next.zone_order.length) {
      next.zone_order = recommendation.zone_priorities.map((item) => item.zone)
    }
    const index = next.zone_order.indexOf(zone)
    next.zone_order = moveItem(next.zone_order, index, delta)
  })
}

function movePick(detectionIndex: number, delta: number) {
  const recommendation = currentRecommendation.value
  if (!recommendation) {
    return
  }

  mutateOverride(recommendation.tree_id, (next) => {
    if (!next.pick_sequence_detection_indices.length) {
      next.pick_sequence_detection_indices = recommendation.pick_sequence.map((item) => item.detection_index)
    }
    const index = next.pick_sequence_detection_indices.indexOf(detectionIndex)
    next.pick_sequence_detection_indices = moveItem(next.pick_sequence_detection_indices, index, delta)
  })
}

function toggleManualSkip(detectionIndex: number) {
  const recommendation = currentRecommendation.value
  if (!recommendation) {
    return
  }

  mutateOverride(recommendation.tree_id, (next) => {
    const exists = next.manual_skipped_detection_indices.includes(detectionIndex)
    next.manual_skipped_detection_indices = exists
      ? next.manual_skipped_detection_indices.filter((item) => item !== detectionIndex)
      : [...next.manual_skipped_detection_indices, detectionIndex]
  })
}

watch(currentPlan, (plan) => {
  initializeEditableState(plan)
  syncSelectedTree(plan)
}, { immediate: true })

watch(selectedPlotId, async (plotId) => {
  await refreshPage(plotId)
}, { immediate: true })

onMounted(async () => {
  await loadPlots()
})
</script>

<template>
  <UContainer class="py-8 sm:py-12">
    <div class="space-y-6">
      <section class="space-y-2">
        <p class="text-xs uppercase tracking-widest text-muted">
          决策
        </p>
        <h1 class="text-2xl font-semibold text-highlighted sm:text-3xl">
          采摘路径与作业决策
        </h1>
        <p class="text-sm text-toned sm:text-base">
          以地块为上下文，整理树间路线、树内顺序、人工调整与树级任务下发。
        </p>
      </section>

      <UAlert
        v-if="pageError"
        color="error"
        variant="subtle"
        icon="i-lucide-triangle-alert"
        title="决策异常"
        :description="pageError"
      />

      <UAlert
        v-else-if="pageMessage"
        color="success"
        variant="subtle"
        icon="i-lucide-badge-check"
        title="已更新"
        :description="pageMessage"
      />

      <UAlert
        v-if="!plots.length"
        color="warning"
        variant="subtle"
        icon="i-lucide-map-off"
        title="暂无地块档案"
        description="请先前往作业页创建地块与树木档案，再返回当前页面生成整块地的采摘计划。"
      />

      <div v-else class="grid grid-cols-1 gap-6 xl:grid-cols-[0.9fr_1.1fr_0.95fr]">
        <div class="space-y-6">
          <UCard variant="outline" :ui="{ body: 'p-5 sm:p-6' }">
            <template #header>
              <div class="flex items-center justify-between gap-3">
                <div>
                  <h2 class="text-base font-semibold text-highlighted">
                    地块上下文
                  </h2>
                  <p class="mt-1 text-xs text-muted">
                    识别页保存的树级识别记录会自动汇总到这里，用来生成当前地块计划。
                  </p>
                </div>
                <UBadge :color="selectedPlot ? 'success' : 'neutral'" variant="soft">
                  {{ selectedPlot ? '已选地块' : '未选择' }}
                </UBadge>
              </div>
            </template>

            <div class="space-y-4">
              <USelect
                :model-value="selectedPlotId"
                :items="plotOptions"
                value-key="value"
                label-key="label"
                :loading="isPlotLoading"
                icon="i-lucide-map"
                placeholder="选择地块"
                @update:model-value="(value) => harvestContext.setSelectedPlot((value as string) || '')"
              />

              <USelect
                :model-value="currentPlanId"
                :items="planOptions"
                value-key="value"
                label-key="label"
                :loading="isPlanLoading"
                :disabled="!plans.length"
                icon="i-lucide-route"
                placeholder="选择历史计划"
                @update:model-value="(value) => handleSelectPlan((value as string) || '')"
              />

              <div class="flex flex-wrap gap-2">
                <UButton
                  color="primary"
                  icon="i-lucide-waypoints"
                  :loading="isGeneratingPlan || isArchivingObservation"
                  label="生成整块地计划"
                  @click="handleGeneratePlan"
                />
                <UButton
                  to="/recognition"
                  color="neutral"
                  variant="outline"
                  icon="i-lucide-camera"
                  label="返回识别页补充记录"
                />
              </div>

              <div
                v-if="currentPlan"
                class="rounded-lg border border-default bg-default px-4 py-4 text-sm text-default"
              >
                <p>地块：{{ selectedPlot?.name || currentPlan.plot_id }}</p>
                <p class="mt-1">计划生成：{{ formatDecisionTimestamp(currentPlan.generated_at) }}</p>
                <p class="mt-1">已纳入计划树木：{{ currentPlan.summary.ready_trees }} / {{ currentPlan.summary.total_trees }}</p>
                <p class="mt-1">待补记录树木：{{ currentPlan.summary.pending_observation_trees }}</p>
                <p class="mt-1">建议采摘目标：{{ currentPlan.summary.total_harvestable_count }}</p>
              </div>
            </div>
          </UCard>

          <UCard variant="outline" :ui="{ body: 'p-5 sm:p-6' }">
            <template #header>
              <div class="flex items-center justify-between gap-3">
                <div>
                  <h2 class="text-base font-semibold text-highlighted">
                    树优先级路线
                  </h2>
                  <p class="mt-1 text-xs text-muted">
                    系统会先结合成熟度与树间位置生成初始路线，你可以再按现场经验微调。
                  </p>
                </div>
                <UBadge :color="orderedTreeSequence.length ? 'success' : 'neutral'" variant="soft">
                  {{ orderedTreeSequence.length }} 棵
                </UBadge>
              </div>
            </template>

            <div v-if="orderedTreeSequence.length" class="space-y-3">
              <button
                v-for="item in orderedTreeSequence"
                :key="item.tree_id"
                type="button"
                class="w-full rounded-lg border px-4 py-4 text-left transition hover:border-primary"
                :class="selectedTreeId === item.tree_id ? 'border-primary bg-primary/5' : 'border-default bg-default'"
                @click="harvestContext.setSelectedTree(item.tree_id)"
              >
                <div class="flex items-start justify-between gap-3">
                  <div>
                    <p class="text-sm font-semibold text-highlighted">
                      {{ item.priority_order }}. {{ item.tree_code }}
                    </p>
                    <p class="mt-1 text-xs text-muted">
                      建议采摘 {{ item.harvestable_count }} 枚 · 距上一树 {{ item.distance_from_previous.toFixed(2) }}
                    </p>
                    <p class="mt-1 text-xs text-muted">
                      综合参考分 {{ item.harvestable_confidence_weighted_score.toFixed(2) }} · {{ formatTreeRecommendationStatus(item.status) }}
                    </p>
                  </div>
                  <div class="flex gap-1">
                    <UButton size="xs" color="neutral" variant="outline" icon="i-lucide-arrow-up" @click.stop="moveTree(item.tree_id, -1)" />
                    <UButton size="xs" color="neutral" variant="outline" icon="i-lucide-arrow-down" @click.stop="moveTree(item.tree_id, 1)" />
                  </div>
                </div>
              </button>
            </div>

            <UAlert
              v-else
              color="neutral"
              variant="subtle"
              icon="i-lucide-route-off"
              title="当前地块还没有主路线"
              description="生成计划后，已保存识别记录的树木会进入路线排序。"
            />

            <div v-if="pendingRecommendations.length" class="mt-4 space-y-2">
              <p class="text-xs font-medium uppercase tracking-widest text-muted">
                待补记录树木
              </p>
              <div class="flex flex-wrap gap-2">
                <UBadge
                  v-for="item in pendingRecommendations"
                  :key="item.tree_id"
                  color="warning"
                  variant="soft"
                >
                  {{ item.tree_code }}
                </UBadge>
              </div>
            </div>
          </UCard>
        </div>

        <div class="space-y-6">
          <UCard variant="outline" :ui="{ body: 'p-5 sm:p-6' }">
            <template #header>
              <div class="flex items-center justify-between gap-3">
                <div>
                  <h2 class="text-base font-semibold text-highlighted">
                    当前树推荐详情
                  </h2>
                  <p class="mt-1 text-xs text-muted">
                    这里聚焦当前树的优先区域、建议顺序与暂缓采摘提示。
                  </p>
                </div>
                <UBadge :color="currentRecommendation?.status === 'ready' ? 'success' : 'warning'" variant="soft">
                  {{ formatTreeRecommendationStatus(currentRecommendation?.status || 'pending_observation') }}
                </UBadge>
              </div>
            </template>

            <div v-if="currentRecommendation" class="space-y-4">
              <div class="rounded-lg border border-default bg-default px-4 py-4">
                <p class="text-sm font-semibold text-highlighted">
                  {{ currentRecommendation.tree_code }}
                </p>
                <p class="mt-1 text-xs text-muted">
                  识别记录：{{ currentRecommendation.observation_id || '待补记录' }}
                </p>
                <p class="mt-1 text-xs text-muted">
                  建议采摘 {{ currentRecommendation.summary.harvestable_count }} 枚，建议暂缓 {{ currentRecommendation.summary.skipped_count }} 枚
                </p>
                <p class="mt-1 text-xs text-muted">
                  主优先区域：{{ formatDecisionZoneLabel(currentRecommendation.summary.main_priority_zone) }}
                </p>
              </div>

              <UAlert
                v-if="currentRecommendation.status !== 'ready'"
                color="warning"
                variant="subtle"
                icon="i-lucide-scan-search"
                title="当前树还缺少识别记录"
                description="请回到识别页绑定该树并保存一帧识别结果，这棵树才会进入树内决策。"
              />

              <div v-else class="grid grid-cols-1 gap-4 lg:grid-cols-[0.95fr_1.05fr]">
                <div class="space-y-4">
                  <div class="rounded-lg border border-default bg-default px-4 py-4">
                    <div class="flex items-center justify-between gap-3">
                      <p class="text-sm font-semibold text-highlighted">
                        区域优先级
                      </p>
                      <UBadge color="neutral" variant="soft">
                        {{ currentDisplayedZonePriorities.length }} 个区域
                      </UBadge>
                    </div>
                    <div class="mt-3 space-y-2">
                      <div
                        v-for="item in currentDisplayedZonePriorities"
                        :key="item.zone"
                        class="rounded-md border border-default px-3 py-3"
                      >
                        <div class="flex items-start justify-between gap-3">
                          <div>
                            <p class="text-sm font-medium text-highlighted">
                              {{ item.order }}. {{ formatDecisionZoneLabel(item.zone) }}
                            </p>
                            <p class="mt-1 text-xs text-muted">
                              建议采摘 {{ item.harvestable_count }} 枚 · 距起始动作点 {{ item.distance_to_start.toFixed(2) }}
                            </p>
                            <p class="mt-1 text-xs text-muted">
                              {{ item.note }}
                            </p>
                          </div>
                          <div class="flex gap-1">
                            <UButton size="xs" color="neutral" variant="outline" icon="i-lucide-arrow-up" @click="moveZone(item.zone, -1)" />
                            <UButton size="xs" color="neutral" variant="outline" icon="i-lucide-arrow-down" @click="moveZone(item.zone, 1)" />
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div class="rounded-lg border border-default bg-default px-4 py-4">
                    <div class="flex items-center justify-between gap-3">
                      <p class="text-sm font-semibold text-highlighted">
                        跳过建议
                      </p>
                      <UBadge color="warning" variant="soft">
                        {{ currentRecommendation.skip_items.length }} 项
                      </UBadge>
                    </div>
                    <div v-if="currentRecommendation.skip_items.length" class="mt-3 space-y-2">
                      <div
                        v-for="item in currentRecommendation.skip_items"
                        :key="`${item.detection_index}-${item.skip_reason}`"
                        class="rounded-md border border-default px-3 py-3 text-sm text-default"
                      >
                        <p class="font-medium text-highlighted">
                          #{{ item.detection_index }} · {{ formatDecisionZoneLabel(item.zone) }}
                        </p>
                        <p class="mt-1 text-xs text-muted">
                          {{ formatDecisionSkipReason(item.skip_reason) }} · 置信度 {{ formatConfidencePercent(item.confidence) }}
                        </p>
                        <p class="mt-1 text-xs text-muted">
                          {{ item.reason }}
                        </p>
                      </div>
                    </div>
                    <UAlert
                      v-else
                      class="mt-3"
                      color="success"
                      variant="subtle"
                      icon="i-lucide-circle-check"
                      title="当前树没有系统暂缓项"
                      description="如果现场判断需要暂缓，可在右侧调整区把目标标记为跳过。"
                    />
                  </div>
                </div>

                <div class="rounded-lg border border-default bg-default px-4 py-4">
                  <div class="flex items-center justify-between gap-3">
                    <div>
                      <p class="text-sm font-semibold text-highlighted">
                        采摘顺序预览
                      </p>
                      <p class="mt-1 text-xs text-muted">
                        可按现场经验调整顺序，或先标记暂缓后再保存计划。
                      </p>
                    </div>
                    <UBadge :color="currentPendingManualSkipCount ? 'warning' : 'neutral'" variant="soft">
                      {{ currentPendingManualSkipCount }} 项待确认跳过
                    </UBadge>
                  </div>

                  <div v-if="currentDisplayedPickRows.length" class="mt-3 space-y-2">
                    <div
                      v-for="item in currentDisplayedPickRows"
                      :key="item.detection_index"
                      class="rounded-md border px-3 py-3"
                      :class="item.manualSkipped ? 'border-warning bg-warning/5' : 'border-default'"
                    >
                      <div class="flex items-start justify-between gap-3">
                        <div>
                          <div class="flex flex-wrap items-center gap-2">
                            <p class="text-sm font-medium text-highlighted">
                              {{ item.order }}. 目标 #{{ item.detection_index }}
                            </p>
                            <UBadge v-if="item.manualSkipped" color="warning" variant="soft">
                              已标记跳过
                            </UBadge>
                          </div>
                          <p class="mt-1 text-xs text-muted">
                            {{ formatDecisionZoneLabel(item.zone) }} · 置信度 {{ formatConfidencePercent(item.confidence) }}
                          </p>
                          <p class="mt-1 text-xs text-muted">
                            {{ item.reason }}
                          </p>
                        </div>
                        <div class="flex flex-wrap gap-1">
                          <UButton size="xs" color="neutral" variant="outline" icon="i-lucide-arrow-up" @click="movePick(item.detection_index, -1)" />
                          <UButton size="xs" color="neutral" variant="outline" icon="i-lucide-arrow-down" @click="movePick(item.detection_index, 1)" />
                          <UButton
                            size="xs"
                            :color="item.manualSkipped ? 'warning' : 'neutral'"
                            variant="outline"
                            icon="i-lucide-hand"
                            :label="item.manualSkipped ? '恢复顺序' : '标记跳过'"
                            @click="toggleManualSkip(item.detection_index)"
                          />
                        </div>
                      </div>
                    </div>
                  </div>

                  <UAlert
                    v-else
                    class="mt-3"
                    color="neutral"
                    variant="subtle"
                    icon="i-lucide-leaf"
                    title="当前树没有可采目标"
                    description="当前识别记录里没有可采目标，因此暂不生成树内采摘顺序。"
                  />
                </div>
              </div>
            </div>

            <UAlert
              v-else
              color="neutral"
              variant="subtle"
              icon="i-lucide-route"
              title="当前还没有地块计划"
              description="先选择地块并点击“生成整块地计划”，系统会根据各树最近一次识别记录整理路线。"
            />
          </UCard>
        </div>

        <div class="space-y-6">
          <UCard variant="outline" :ui="{ body: 'p-5 sm:p-6' }">
            <template #header>
              <div class="flex items-center justify-between gap-3">
                <div>
                  <h2 class="text-base font-semibold text-highlighted">
                    人工调整与下发
                  </h2>
                  <p class="mt-1 text-xs text-muted">
                    确认树顺序、区域顺序与采摘顺序后，再下发树级作业任务。
                  </p>
                </div>
                <UBadge :color="currentPlan?.manual_override ? 'warning' : 'neutral'" variant="soft">
                  {{ currentPlan?.manual_override ? '已保存本轮调整' : '沿用系统建议' }}
                </UBadge>
              </div>
            </template>

            <div class="space-y-4">
              <div
                v-if="currentRecommendation"
                class="rounded-lg border border-default bg-default px-4 py-4 text-sm text-default"
              >
                <p>当前树：{{ currentRecommendation.tree_code }}</p>
                <p class="mt-1">地块：{{ selectedPlot?.name || currentPlan?.plot_id || '未选择' }}</p>
                <p class="mt-1">树档案：{{ treeById[currentRecommendation.tree_id]?.tree_code || currentRecommendation.tree_code }}</p>
                <p class="mt-1">已标记跳过：{{ currentPendingManualSkipCount }} 项</p>
              </div>

              <div class="flex flex-wrap gap-2">
                <UButton
                  color="primary"
                  icon="i-lucide-save"
                  :disabled="!currentPlan"
                  :loading="isSavingOverride"
                  label="保存人工调整"
                  @click="handleSaveOverride"
                />
                <UButton
                  color="success"
                  icon="i-lucide-briefcase-business"
                  :disabled="!currentPlan"
                  :loading="isCreatingWorkOrders"
                  label="生成树级作业单"
                  @click="handleCreateWorkOrders"
                />
                <UButton
                  to="/operations"
                  color="neutral"
                  variant="outline"
                  icon="i-lucide-arrow-right"
                  label="进入作业执行页"
                />
              </div>

              <UAlert
                v-if="decisionSnapshot.hasPendingObservation && selectedTreeId"
                color="warning"
                variant="subtle"
                icon="i-lucide-save"
                title="检测到尚未保存的识别画面"
                description="点击生成计划前，系统会先自动把识别页留下的最新画面保存到当前树。"
              />
            </div>
          </UCard>

          <UCard variant="outline" :ui="{ body: 'p-5 sm:p-6' }">
            <template #header>
              <div class="flex items-center justify-between gap-3">
                <div>
                  <h2 class="text-base font-semibold text-highlighted">
                    最新下发结果
                  </h2>
                  <p class="mt-1 text-xs text-muted">
                    每棵可执行的树都会生成一张作业任务，之后可在作业页持续推进状态。
                  </p>
                </div>
                <UBadge :color="createdWorkOrders.length ? 'success' : 'neutral'" variant="soft">
                  {{ createdWorkOrders.length }} 张
                </UBadge>
              </div>
            </template>

            <div v-if="createdWorkOrders.length" class="space-y-3">
              <div
                v-for="item in createdWorkOrders"
                :key="item.work_order_id"
                class="rounded-lg border border-default bg-default px-4 py-4 text-sm text-default"
              >
                <p class="font-medium text-highlighted">
                  {{ item.tree_code }}
                </p>
                <p class="mt-1 text-xs text-muted">
                  作业单：{{ item.work_order_id }}
                </p>
                <p class="mt-1 text-xs text-muted">
                  当前状态：{{ formatWorkOrderStatus(item.status) }}
                </p>
                <p class="mt-1 text-xs text-muted">
                  覆盖区域 {{ item.zone_priorities.length }} 个 · 建议顺序 {{ item.pick_sequence.length }} 项
                </p>
              </div>
            </div>

            <UAlert
              v-else
              color="neutral"
              variant="subtle"
              icon="i-lucide-briefcase"
              title="尚未下发作业单"
              description="保存人工调整后点击“生成树级作业单”，就可以把当前计划转入执行闭环。"
            />
          </UCard>
        </div>
      </div>
    </div>
  </UContainer>
</template>
