<script setup lang="ts">
import { useCamera } from '~/composables/useCamera'
import { useInferenceStream } from '~/composables/useInferenceStream'

useSeoMeta({
  title: '识别',
  description: '实时识别油茶果状态并输出采摘建议基线。'
})

const videoElement = ref<HTMLVideoElement | null>(null)

const camera = useCamera(videoElement)
const stream = useInferenceStream({
  videoElement,
  frameIntervalMs: 300,
  jpegQuality: 0.8
})
const {
  startStream,
  stopStream,
  isStreaming,
  streamError,
  lastFrame,
  serverSummary,
  aggregateSummary
} = stream

async function handleStartRecognition() {
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
          Recognition
        </p>
        <h1 class="text-2xl font-semibold text-highlighted sm:text-3xl">
          识别基线页
        </h1>
        <p class="text-sm text-toned sm:text-base">
          聚焦第一层识别能力，提供实时观察、成熟度汇总和采摘建议基线。
        </p>
      </section>

      <div class="grid grid-cols-1 gap-6 xl:grid-cols-[1.1fr_1fr]">
        <RecognitionCameraStage
          :devices="camera.options.value"
          :selected-device-id="camera.selectedDeviceId.value"
          :is-recognizing="isStreaming"
          :camera-loading="camera.isCameraLoading.value"
          :camera-error="camera.cameraError.value"
          :stream-error="streamError"
          @update:video-element="handleVideoElementChange"
          @update:selected-device-id="handleSwitchDevice"
          @start="handleStartRecognition"
          @stop="handleStopRecognition"
          @refresh="handleRefreshDevices"
        />

        <div class="space-y-6">
          <RecognitionLiveSummaryPanel
            :summary="aggregateSummary"
            :current-frame="lastFrame"
            :server-summary="serverSummary"
            :is-recognizing="isStreaming"
          />

          <UCard variant="outline" :ui="{ body: 'p-5 sm:p-6' }">
            <div class="space-y-3">
              <h2 class="text-base font-semibold text-highlighted">
                下一层预留
              </h2>
              <p class="text-sm text-toned">
                当前分支先固定识别基线。后续会把本页输出接入决策层，用于树优先级、区域优先级和采摘顺序建议。
              </p>
              <div class="flex flex-wrap gap-2">
                <UButton to="/decision" color="neutral" variant="outline" label="查看决策骨架" />
                <UButton to="/operations" color="neutral" variant="outline" label="查看作业骨架" />
              </div>
            </div>
          </UCard>
        </div>
      </div>
    </div>
  </UContainer>
</template>
