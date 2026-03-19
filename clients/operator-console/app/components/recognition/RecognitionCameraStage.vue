<script setup lang="ts">
import type { FrameResult } from '~/types/infer'
import type { CameraOption } from '~/composables/useCamera'
import { buildRecognitionOverlayBoxes } from '~/utils/recognition-overlay'

const props = defineProps<{
  devices: CameraOption[]
  selectedDeviceId: string
  isRecognizing: boolean
  cameraLoading: boolean
  currentFrame: FrameResult | null
  cameraError?: string
  streamError?: string
}>()

const emit = defineEmits<{
  (event: 'update:videoElement', value: HTMLVideoElement | null): void
  (event: 'update:selectedDeviceId', value: string): void
  (event: 'start'): void
  (event: 'stop'): void
  (event: 'refresh'): void
}>()

const localVideo = ref<HTMLVideoElement | null>(null)
const videoSize = ref({ width: 0, height: 0 })

watch(localVideo, (video) => {
  emit('update:videoElement', video)
}, { immediate: true })

watch(localVideo, (video, previousVideo) => {
  previousVideo?.removeEventListener('loadedmetadata', syncVideoSize)
  previousVideo?.removeEventListener('resize', syncVideoSize)

  if (!video) {
    videoSize.value = { width: 0, height: 0 }
    return
  }

  video.addEventListener('loadedmetadata', syncVideoSize)
  video.addEventListener('resize', syncVideoSize)
  syncVideoSize()
}, { immediate: true })

onBeforeUnmount(() => {
  localVideo.value?.removeEventListener('loadedmetadata', syncVideoSize)
  localVideo.value?.removeEventListener('resize', syncVideoSize)
  emit('update:videoElement', null)
})

const hasDevices = computed(() => props.devices.length > 0)
const stageAspectRatio = computed(() => {
  if (videoSize.value.width > 0 && videoSize.value.height > 0) {
    return `${videoSize.value.width} / ${videoSize.value.height}`
  }
  return '16 / 9'
})
const overlayBoxes = computed(() =>
  buildRecognitionOverlayBoxes(
    props.currentFrame?.detections || [],
    videoSize.value.width,
    videoSize.value.height
  )
)

function syncVideoSize() {
  const video = localVideo.value
  if (!video) {
    videoSize.value = { width: 0, height: 0 }
    return
  }

  videoSize.value = {
    width: video.videoWidth || 0,
    height: video.videoHeight || 0
  }
}
</script>

<template>
  <UCard variant="outline" :ui="{ body: 'p-4 sm:p-5' }">
    <template #header>
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 class="text-base font-semibold text-highlighted">
            实时识别
          </h2>
          <p class="mt-1 text-xs text-muted">
            通过网关 WebSocket 将油茶果画面发送到识别服务。
          </p>
        </div>
        <UBadge :color="isRecognizing ? 'success' : 'neutral'" variant="soft">
          {{ isRecognizing ? '识别中' : '空闲' }}
        </UBadge>
      </div>
    </template>

    <div class="space-y-4">
      <div class="grid grid-cols-1 gap-3 lg:grid-cols-[1fr_auto_auto]">
        <USelect
          :model-value="selectedDeviceId"
          :items="devices"
          value-key="value"
          label-key="label"
          icon="i-lucide-camera"
          :disabled="cameraLoading || !hasDevices"
          placeholder="选择摄像头"
          @update:model-value="(value) => emit('update:selectedDeviceId', value as string)"
        />

        <UButton
          color="neutral"
          variant="outline"
          icon="i-lucide-refresh-cw"
          :loading="cameraLoading"
          label="刷新设备"
          @click="emit('refresh')"
        />

        <UButton
          v-if="!isRecognizing"
          color="primary"
          icon="i-lucide-play"
          :loading="cameraLoading"
          :disabled="cameraLoading"
          label="开始识别"
          @click="emit('start')"
        />
        <UButton
          v-else
          color="error"
          variant="outline"
          icon="i-lucide-square"
          label="停止识别"
          @click="emit('stop')"
        />
      </div>

      <UAlert
        v-if="!hasDevices"
        color="warning"
        variant="subtle"
        icon="i-lucide-camera-off"
        title="未检测到摄像头"
        description="可先点击开始识别触发授权，再刷新设备列表。"
      />

      <UAlert
        v-if="cameraError"
        color="warning"
        variant="subtle"
        icon="i-lucide-triangle-alert"
        title="摄像头异常"
        :description="cameraError"
      />

      <UAlert
        v-if="streamError"
        color="error"
        variant="subtle"
        icon="i-lucide-wifi-off"
        title="识别流异常"
      >
        <template #description>
          <div class="space-y-2">
            <p class="text-sm">
              {{ streamError }}
            </p>
            <UButton
              v-if="hasDevices && !isRecognizing"
              size="xs"
              color="error"
              variant="outline"
              icon="i-lucide-rotate-cw"
              label="重试连接"
              @click="emit('start')"
            />
          </div>
        </template>
      </UAlert>

      <div
        class="relative overflow-hidden rounded-lg border border-accented bg-black/95"
        :style="{ aspectRatio: stageAspectRatio }"
      >
        <video
          ref="localVideo"
          autoplay
          muted
          playsinline
          class="absolute inset-0 h-full w-full object-contain"
        />

        <div
          v-if="overlayBoxes.length"
          class="pointer-events-none absolute inset-0"
          aria-label="识别结果框选叠层"
        >
          <div
            v-for="box in overlayBoxes"
            :key="box.key"
            class="absolute border-2 border-emerald-400 shadow-[0_0_0_1px_rgba(16,185,129,0.45)]"
            :style="{
              left: box.left,
              top: box.top,
              width: box.width,
              height: box.height
            }"
          >
            <div class="absolute left-0 top-0 -translate-y-full rounded-t-md bg-emerald-400/95 px-2 py-1 text-[11px] font-medium leading-none text-black">
              {{ box.label }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </UCard>
</template>
