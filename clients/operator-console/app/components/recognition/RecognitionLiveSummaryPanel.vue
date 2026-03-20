<script setup lang="ts">
import type { FrameResult, SessionSummary } from '~/types/infer'
import type { SessionAggregateSummary } from '~/utils/session-aggregator'

const props = defineProps<{
  summary: SessionAggregateSummary
  currentFrame: FrameResult | null
  serverSummary: SessionSummary | null
  isRecognizing: boolean
}>()
</script>

<template>
  <UCard variant="outline" :ui="{ body: 'p-4 sm:p-5' }">
    <template #header>
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 class="text-base font-semibold text-highlighted">
            会话汇总
          </h2>
          <p class="mt-1 text-xs text-muted">
            当前会话累计识别 {{ summary.total }} 枚油茶果目标
          </p>
        </div>
        <UBadge :color="isRecognizing ? 'success' : 'neutral'" variant="soft">
          {{ isRecognizing ? '实时更新中' : '已暂停更新' }}
        </UBadge>
      </div>
    </template>

    <div class="space-y-4">
      <UAlert
        color="neutral"
        variant="subtle"
        icon="i-lucide-leaf"
        title="检测基线说明"
      >
        <template #description>
          <p class="text-sm">
            当前主线已支持油茶果检测与成熟度判断，识别框会优先展示可采状态，后续再继续补强采摘建议。
          </p>
        </template>
      </UAlert>

      <UCard
        v-if="currentFrame"
        variant="subtle"
        :ui="{ body: 'px-4 py-3' }"
      >
        <p class="text-xs text-muted">
          当前帧汇总（frame #{{ currentFrame.frame_index }})
        </p>
        <div class="mt-2 grid grid-cols-1 gap-x-4 gap-y-1 text-sm sm:grid-cols-2">
          <p>总数：{{ currentFrame.frame_summary.total }}</p>
          <p>当前帧目标框：{{ currentFrame.detections.length }}</p>
        </div>
      </UCard>

      <UCard
        v-if="serverSummary"
        variant="subtle"
        :ui="{ body: 'px-4 py-3' }"
      >
        <p class="text-xs text-muted">
          服务端会话总量（summary 事件）
        </p>
        <p class="mt-1 text-sm text-default">
          total_detected = {{ serverSummary.total_detected }}
        </p>
      </UCard>
    </div>
  </UCard>
</template>
