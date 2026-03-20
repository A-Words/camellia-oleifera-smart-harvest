<script setup lang="ts">
import type { TreeObservation } from '~/types/decision'
import type { Plot, TreeArchive } from '~/types/operations'
import { useCamera } from '~/composables/useCamera'
import { useDecisionSnapshot } from '~/composables/useDecisionSnapshot'
import { useGatewayBase } from '~/composables/useGatewayBase'
import { useHarvestContext } from '~/composables/useHarvestContext'
import { useInferenceStream } from '~/composables/useInferenceStream'
import { formatTreeStatus, formatDecisionTimestamp } from '~/utils/decision-presenter'

useSeoMeta({
  title: '识别',
  description: '绑定地块与树木后进行实时识别，并保存树级识别记录。'
})

const gatewayBase = useGatewayBase()
const videoElement = ref<HTMLVideoElement | null>(null)

const camera = useCamera(videoElement)
const stream = useInferenceStream({
  videoElement,
  frameIntervalMs: 300,
  jpegQuality: 0.8
})
const decisionSnapshot = useDecisionSnapshot()
const harvestContext = useHarvestContext()

const plots = ref<Plot[]>([])
const trees = ref<TreeArchive[]>([])
const latestObservation = ref<TreeObservation | null>(null)
const latestObservationError = ref('')
const assignmentError = ref('')
const isPlotLoading = ref(false)
const isTreeLoading = ref(false)
const isObservationSaving = ref(false)

const {
  startStream,
  stopStream,
  isStreaming,
  streamError,
  lastFrame,
  serverSummary,
  aggregateSummary
} = stream

const selectedPlotId = harvestContext.selectedPlotId
const selectedTreeId = harvestContext.selectedTreeId

const plotOptions = computed(() =>
  plots.value.map((plot) => ({
    label: `${plot.name} (${plot.code})`,
    value: plot.plot_id
  }))
)

const treeOptions = computed(() =>
  trees.value.map((tree) => ({
    label: `${tree.tree_code} · 第 ${tree.row_index} 行第 ${tree.col_index} 列`,
    value: tree.tree_id
  }))
)

const selectedTree = computed(() =>
  trees.value.find((tree) => tree.tree_id === selectedTreeId.value) || null
)

const canStartRecognition = computed(() =>
  Boolean(selectedPlotId.value) &&
  Boolean(selectedTreeId.value)
)

const startDisabledReason = computed(() => {
  if (!plots.value.length) {
    return '请先到作业页建立地块和树木档案，再返回识别页绑定当前作业对象。'
  }
  if (!selectedPlotId.value) {
    return '请先选择当前识别所属的地块。'
  }
  if (!selectedTreeId.value) {
    return '请先选择当前识别所属的树木。'
  }
  return ''
})

async function loadPlots() {
  isPlotLoading.value = true
  assignmentError.value = ''
  try {
    const response = await $fetch<{ items?: Plot[] }>(`${gatewayBase.value}/v1/operations/plots`)
    plots.value = response.items || []

    if (!plots.value.length) {
      harvestContext.setSelectedPlot('')
      harvestContext.setSelectedTree('')
      trees.value = []
      return
    }

    const stillExists = plots.value.some((plot) => plot.plot_id === selectedPlotId.value)
    if (!stillExists) {
      harvestContext.setSelectedPlot(plots.value[0]!.plot_id)
    }
  } catch (error) {
    assignmentError.value = error instanceof Error ? error.message : '读取地块失败。'
  } finally {
    isPlotLoading.value = false
  }
}

async function loadTrees(plotId: string) {
  if (!plotId) {
    trees.value = []
    harvestContext.setSelectedTree('')
    return
  }

  isTreeLoading.value = true
  latestObservation.value = null
  latestObservationError.value = ''
  try {
    const response = await $fetch<{ items?: TreeArchive[] }>(`${gatewayBase.value}/v1/operations/plots/${plotId}/trees`)
    trees.value = response.items || []

    const stillExists = trees.value.some((tree) => tree.tree_id === selectedTreeId.value)
    if (!stillExists) {
      harvestContext.setSelectedTree('')
    }
  } catch (error) {
    assignmentError.value = error instanceof Error ? error.message : '读取树木失败。'
  } finally {
    isTreeLoading.value = false
  }
}

async function loadLatestObservation(treeId: string) {
  if (!treeId) {
    latestObservation.value = null
    return
  }

  try {
    const response = await $fetch<{ items?: TreeObservation[] }>(`${gatewayBase.value}/v1/decision/observations`, {
      query: { tree_id: treeId }
    })
    latestObservation.value = response.items?.[0] || null
  } catch (error) {
    latestObservationError.value = error instanceof Error ? error.message : '读取最近识别记录失败。'
  }
}

async function handleStartRecognition() {
  assignmentError.value = ''
  if (!canStartRecognition.value) {
    assignmentError.value = startDisabledReason.value || '请先绑定当前地块和树木。'
    return
  }

  if (!camera.currentStream.value) {
    await camera.startCamera(camera.selectedDeviceId.value)
  }

  if (camera.cameraError.value) {
    return
  }

  await startStream()
}

async function handleStopRecognition() {
  await stopStream()
}

async function handleRefreshDevices() {
  await camera.refreshDevices()
}

async function handleSwitchDevice(deviceId: string) {
  await camera.switchCamera(deviceId)
}

function handleVideoElementChange(video: HTMLVideoElement | null) {
  videoElement.value = video
}

async function archiveCurrentObservation() {
  assignmentError.value = ''
  latestObservationError.value = ''
  if (!selectedTreeId.value || !decisionSnapshot.snapshot.value) {
    assignmentError.value = '需要先选择树木并获取一帧识别画面，才能保存当前树记录。'
    return
  }

  isObservationSaving.value = true
  try {
    latestObservation.value = await $fetch<TreeObservation>(`${gatewayBase.value}/v1/decision/observations`, {
      method: 'POST',
      body: {
        tree_id: selectedTreeId.value,
        captured_at: new Date().toISOString(),
        ...decisionSnapshot.snapshot.value
      }
    })
    decisionSnapshot.markArchived()
  } catch (error) {
    latestObservationError.value = error instanceof Error ? error.message : '保存树级识别记录失败。'
  } finally {
    isObservationSaving.value = false
  }
}

async function archiveAndOpenDecision() {
  if (decisionSnapshot.hasPendingObservation.value) {
    await archiveCurrentObservation()
    if (latestObservationError.value) {
      return
    }
  }
  await navigateTo('/decision')
}

watch(lastFrame, (frame) => {
  decisionSnapshot.updateSnapshotFromFrame(frame, {
    width: videoElement.value?.videoWidth || 0,
    height: videoElement.value?.videoHeight || 0
  })
})

watch(selectedPlotId, async (plotId) => {
  await loadTrees(plotId)
}, { immediate: true })

watch(selectedTreeId, async (treeId) => {
  await loadLatestObservation(treeId)
}, { immediate: true })

onMounted(async () => {
  await loadPlots()
})

onBeforeUnmount(() => {
  void stopStream()
  camera.stopCamera()
})
</script>

<template>
  <UContainer class="py-8 sm:py-12">
    <div class="space-y-6">
      <section class="space-y-2">
        <p class="text-xs uppercase tracking-widest text-muted">
          识别
        </p>
        <h1 class="text-2xl font-semibold text-highlighted sm:text-3xl">
          树级识别记录
        </h1>
        <p class="text-sm text-toned sm:text-base">
          先选定当前地块与树木，再进行实时识别，并把最新画面保存为该树的识别记录。
        </p>
      </section>

      <div class="grid grid-cols-1 gap-6 xl:grid-cols-[1.15fr_1fr]">
        <RecognitionCameraStage
          :devices="camera.options.value"
          :selected-device-id="camera.selectedDeviceId.value"
          :is-recognizing="isStreaming"
          :camera-loading="camera.isCameraLoading.value"
          :current-frame="lastFrame"
          :camera-error="camera.cameraError.value"
          :stream-error="streamError"
          :can-start-recognition="canStartRecognition"
          :start-disabled-reason="startDisabledReason"
          @update:video-element="handleVideoElementChange"
          @update:selected-device-id="handleSwitchDevice"
          @start="handleStartRecognition"
          @stop="handleStopRecognition"
          @refresh="handleRefreshDevices"
        >
          <div class="space-y-4">
            <UCard variant="subtle" :ui="{ body: 'p-4' }">
              <template #header>
                <div class="flex items-center justify-between gap-3">
                  <h3 class="text-sm font-semibold text-highlighted">
                    当前识别对象
                  </h3>
                  <UBadge :color="canStartRecognition ? 'success' : 'neutral'" variant="soft">
                    {{ canStartRecognition ? '已选定' : '待选择' }}
                  </UBadge>
                </div>
              </template>

              <div class="grid grid-cols-1 gap-3">
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
                  :model-value="selectedTreeId"
                  :items="treeOptions"
                  value-key="value"
                  label-key="label"
                  :loading="isTreeLoading"
                  :disabled="!selectedPlotId"
                  icon="i-lucide-tree-pine"
                  placeholder="选择树木"
                  @update:model-value="(value) => harvestContext.setSelectedTree((value as string) || '')"
                />
              </div>

              <UAlert
                v-if="assignmentError"
                class="mt-3"
                color="warning"
                variant="subtle"
                icon="i-lucide-triangle-alert"
                title="选择提示"
                :description="assignmentError"
              />
            </UCard>

            <div class="flex flex-wrap gap-2">
              <UButton
                color="neutral"
                variant="outline"
                icon="i-lucide-save"
                :disabled="!decisionSnapshot.hasSnapshot || !selectedTreeId"
                :loading="isObservationSaving"
                label="保存当前树记录"
                @click="archiveCurrentObservation"
              />
              <UButton
                color="primary"
                icon="i-lucide-arrow-right"
                :disabled="!selectedPlotId"
                :loading="isObservationSaving"
                label="进入地块决策"
                @click="archiveAndOpenDecision"
              />
              <UButton
                to="/operations"
                color="neutral"
                variant="outline"
                icon="i-lucide-briefcase-business"
                label="管理地块与树木"
              />
            </div>
          </div>
        </RecognitionCameraStage>

        <div class="space-y-6">
          <RecognitionLiveSummaryPanel
            :summary="aggregateSummary"
            :current-frame="lastFrame"
            :server-summary="serverSummary"
            :is-recognizing="isStreaming"
          />

          <UCard variant="outline" :ui="{ body: 'p-5 sm:p-6' }">
            <template #header>
              <div class="flex items-center justify-between gap-3">
                <div>
                  <h2 class="text-base font-semibold text-highlighted">
                    当前树识别记录
                  </h2>
                  <p class="mt-1 text-xs text-muted">
                    地块计划只读取已保存的树级识别记录，不直接使用未保存的临时画面。
                  </p>
                </div>
                <UBadge :color="latestObservation ? 'success' : 'neutral'" variant="soft">
                  {{ latestObservation ? '已保存' : '未保存' }}
                </UBadge>
              </div>
            </template>

            <div class="space-y-3">
              <UAlert
                v-if="!plots.length"
                color="warning"
                variant="subtle"
                icon="i-lucide-map-off"
                title="暂无地块档案"
                description="请先到作业页创建地块和树木，再回到识别页开始绑定。"
              />

              <UAlert
                v-if="latestObservationError"
                color="error"
                variant="subtle"
                icon="i-lucide-triangle-alert"
                title="记录异常"
                :description="latestObservationError"
              />

              <div v-if="selectedTree" class="rounded-lg border border-default bg-default px-4 py-4">
                <p class="text-sm font-medium text-highlighted">
                  {{ selectedTree.tree_code }}
                </p>
                <p class="mt-1 text-xs text-muted">
                  第 {{ selectedTree.row_index }} 行第 {{ selectedTree.col_index }} 列 · {{ formatTreeStatus(selectedTree.status) }}
                </p>
              </div>

              <div
                v-if="latestObservation"
                class="rounded-lg border border-default bg-default px-4 py-4 text-sm text-default"
              >
                <p>记录编号：{{ latestObservation.observation_id }}</p>
                <p class="mt-1">保存时间：{{ formatDecisionTimestamp(latestObservation.captured_at) }}</p>
                <p class="mt-1">检测目标：{{ latestObservation.detections.length }}</p>
                <p class="mt-1">画面尺寸：{{ latestObservation.frame_width }} × {{ latestObservation.frame_height }}</p>
              </div>

              <UAlert
                v-else
                color="neutral"
                variant="subtle"
                icon="i-lucide-scan-search"
                title="当前树还没有识别记录"
                description="开始识别后，点击“保存当前树记录”即可把当前画面提交到地块决策。"
              />
            </div>
          </UCard>
        </div>
      </div>
    </div>
  </UContainer>
</template>
