<script setup lang="ts">
import type { DecisionHistoryItem } from '~/types/decision'
import { formatDecisionSkipReason, formatDecisionTimestamp, formatDecisionZoneLabel } from '~/utils/decision-presenter'

defineProps<{
  items: DecisionHistoryItem[]
  loading: boolean
}>()
</script>

<template>
  <UCard variant="outline" :ui="{ body: 'p-5 sm:p-6' }">
    <template #header>
      <div class="flex items-center justify-between gap-3">
        <div>
          <h2 class="text-base font-semibold text-highlighted">
            最近决策历史
          </h2>
          <p class="mt-1 text-xs text-muted">
            最近 10 条识别快照决策记录
          </p>
        </div>
        <UBadge :color="loading ? 'warning' : 'neutral'" variant="soft">
          {{ loading ? '读取中' : `${items.length} 条` }}
        </UBadge>
      </div>
    </template>

    <div v-if="items.length" class="space-y-3">
      <div
        v-for="item in items"
        :key="item.decision_id"
        class="rounded-lg border border-default bg-default px-4 py-4"
      >
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div>
            <p class="font-medium text-highlighted">
              {{ item.decision_id }}
            </p>
            <p class="mt-1 text-xs text-muted">
              {{ formatDecisionTimestamp(item.created_at) }}
            </p>
          </div>
          <UBadge color="neutral" variant="soft">
            {{ formatDecisionZoneLabel(item.summary.main_priority_zone) }}
          </UBadge>
        </div>

        <div class="mt-3 grid grid-cols-1 gap-2 text-sm text-default sm:grid-cols-2">
          <p>检测总数：{{ item.summary.total_detections }}</p>
          <p>可采目标：{{ item.summary.harvestable_count }}</p>
          <p>跳过目标：{{ item.summary.skipped_count }}</p>
          <p>路径长度：{{ item.summary.estimated_path_length.toFixed(2) }}</p>
        </div>

        <div class="mt-3 flex flex-wrap gap-2">
          <UBadge
            v-for="zone in item.zone_priorities.slice(0, 3)"
            :key="`${item.decision_id}-${zone.zone}`"
            color="primary"
            variant="subtle"
          >
            {{ zone.order }}. {{ formatDecisionZoneLabel(zone.zone) }}
          </UBadge>
          <UBadge
            v-for="skip in item.skip_items.slice(0, 2)"
            :key="`${item.decision_id}-${skip.detection_index}-${skip.skip_reason}`"
            color="warning"
            variant="subtle"
          >
            {{ formatDecisionSkipReason(skip.skip_reason) }}
          </UBadge>
        </div>
      </div>
    </div>

    <UAlert
      v-else
      color="neutral"
      variant="subtle"
      icon="i-lucide-history"
      title="暂无历史记录"
      description="生成一次采摘决策后，最近记录会显示在这里。"
    />
  </UCard>
</template>
