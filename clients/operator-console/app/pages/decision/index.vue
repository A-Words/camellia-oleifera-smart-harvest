<script setup lang="ts">
import type { DecisionHistoryItem, DecisionRecommendationResponse } from '~/types/decision'
import { useDecisionSnapshot } from '~/composables/useDecisionSnapshot'
import { useGatewayBase } from '~/composables/useGatewayBase'

useSeoMeta({
  title: '决策',
  description: '采摘路径与作业决策能力页。'
})

const gatewayBase = useGatewayBase()
const {
  snapshot,
  hasSnapshot,
  shouldAutoSubmit,
  markSubmitted
} = useDecisionSnapshot()

const currentDecision = ref<DecisionRecommendationResponse | null>(null)
const historyItems = ref<DecisionHistoryItem[]>([])
const isGenerating = ref(false)
const isHistoryLoading = ref(false)
const pageError = ref('')

async function loadHistory() {
  isHistoryLoading.value = true
  try {
    const response = await $fetch(`${gatewayBase.value}/v1/decision/history`)
    historyItems.value = (response as { items?: DecisionHistoryItem[] }).items || []
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : '读取决策历史失败。'
  } finally {
    isHistoryLoading.value = false
  }
}

async function generateDecision() {
  if (!snapshot.value) {
    return
  }

  isGenerating.value = true
  pageError.value = ''

  try {
    currentDecision.value = await $fetch<DecisionRecommendationResponse>(
      `${gatewayBase.value}/v1/decision/recommendation`,
      {
        method: 'POST',
        body: snapshot.value
      }
    )
    markSubmitted()
    await loadHistory()
  } catch (error) {
    pageError.value = error instanceof Error ? error.message : '生成采摘决策失败。'
  } finally {
    isGenerating.value = false
  }
}

onMounted(async () => {
  await loadHistory()
  if (shouldAutoSubmit.value) {
    await generateDecision()
  }
})
</script>

<template>
  <UContainer class="py-8 sm:py-12">
    <div class="space-y-6">
      <section class="flex flex-wrap items-start justify-between gap-3">
        <div class="space-y-2">
          <p class="text-xs uppercase tracking-widest text-muted">
            Decision
          </p>
          <h1 class="text-2xl font-semibold text-highlighted sm:text-3xl">
            采摘路径与作业决策
          </h1>
          <p class="text-sm text-toned sm:text-base">
            基于识别层最近一帧快照，生成可采区域优先级、就近采摘顺序和跳过建议。
          </p>
        </div>

        <div class="flex flex-wrap gap-2">
          <UButton
            v-if="hasSnapshot"
            color="primary"
            icon="i-lucide-sparkles"
            :loading="isGenerating"
            label="基于最新识别结果生成"
            @click="generateDecision"
          />
          <UButton
            color="neutral"
            variant="outline"
            icon="i-lucide-refresh-cw"
            :loading="isHistoryLoading"
            label="刷新历史"
            @click="loadHistory"
          />
        </div>
      </section>

      <UAlert
        v-if="pageError"
        color="error"
        variant="subtle"
        icon="i-lucide-triangle-alert"
        title="决策服务异常"
        :description="pageError"
      />

      <DecisionResultPanel
        v-if="currentDecision"
        :decision="currentDecision"
        :generating="isGenerating"
      />
      <DecisionPlaceholderPanel
        v-else
        :has-history="historyItems.length > 0"
      />

      <div id="decision-history">
        <DecisionHistoryPanel
          :items="historyItems"
          :loading="isHistoryLoading"
        />
      </div>
    </div>
  </UContainer>
</template>
