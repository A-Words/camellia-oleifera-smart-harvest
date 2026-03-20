<script setup lang="ts">
import type { DecisionRecommendationResponse } from '~/types/decision'
import { formatDecisionSkipReason, formatDecisionTimestamp, formatDecisionZoneLabel } from '~/utils/decision-presenter'

defineProps<{
  decision: DecisionRecommendationResponse
  generating: boolean
}>()
</script>

<template>
  <UCard variant="outline" :ui="{ body: 'p-5 sm:p-6' }">
    <template #header>
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p class="text-xs uppercase tracking-widest text-muted">
            Decision
          </p>
          <h1 class="text-2xl font-semibold text-highlighted sm:text-3xl">
            采摘路径与作业决策
          </h1>
          <p class="mt-2 text-sm text-toned">
            基于最近一帧识别结果，输出区域优先级、采摘顺序和跳过建议。
          </p>
        </div>
        <UBadge :color="generating ? 'warning' : 'success'" variant="soft">
          {{ generating ? '生成中' : '已生成' }}
        </UBadge>
      </div>
    </template>

    <div class="space-y-6">
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-5">
        <UCard variant="subtle" :ui="{ body: 'px-4 py-3' }">
          <p class="text-xs text-muted">
            总目标数
          </p>
          <p class="mt-2 text-2xl font-semibold text-highlighted">
            {{ decision.summary.total_detections }}
          </p>
        </UCard>

        <UCard variant="subtle" :ui="{ body: 'px-4 py-3' }">
          <p class="text-xs text-muted">
            可采目标
          </p>
          <p class="mt-2 text-2xl font-semibold text-emerald-600">
            {{ decision.summary.harvestable_count }}
          </p>
        </UCard>

        <UCard variant="subtle" :ui="{ body: 'px-4 py-3' }">
          <p class="text-xs text-muted">
            跳过目标
          </p>
          <p class="mt-2 text-2xl font-semibold text-amber-600">
            {{ decision.summary.skipped_count }}
          </p>
        </UCard>

        <UCard variant="subtle" :ui="{ body: 'px-4 py-3' }">
          <p class="text-xs text-muted">
            主优先区域
          </p>
          <p class="mt-2 text-lg font-semibold text-highlighted">
            {{ formatDecisionZoneLabel(decision.summary.main_priority_zone) }}
          </p>
        </UCard>

        <UCard variant="subtle" :ui="{ body: 'px-4 py-3' }">
          <p class="text-xs text-muted">
            预计路径长度
          </p>
          <p class="mt-2 text-lg font-semibold text-highlighted">
            {{ decision.summary.estimated_path_length.toFixed(2) }}
          </p>
        </UCard>
      </div>

      <UAlert
        color="neutral"
        variant="subtle"
        icon="i-lucide-info"
        :title="`决策编号 ${decision.decision_id}`"
        :description="`生成时间 ${formatDecisionTimestamp(decision.created_at)}`"
      />

      <div class="grid grid-cols-1 gap-4 xl:grid-cols-[0.95fr_1.05fr]">
        <UCard variant="subtle" :ui="{ body: 'p-4' }">
          <template #header>
            <div class="flex items-center justify-between gap-3">
              <h2 class="text-base font-semibold text-highlighted">
                区域优先级
              </h2>
              <UBadge color="neutral" variant="soft">
                {{ decision.zone_priorities.length }} 个区域
              </UBadge>
            </div>
          </template>

          <div v-if="decision.zone_priorities.length" class="space-y-3">
            <div
              v-for="zone in decision.zone_priorities"
              :key="zone.order"
              class="rounded-lg border border-default bg-default px-4 py-3"
            >
              <div class="flex items-center justify-between gap-3">
                <p class="font-medium text-highlighted">
                  {{ zone.order }}. {{ formatDecisionZoneLabel(zone.zone) }}
                </p>
                <UBadge color="success" variant="soft">
                  {{ zone.harvestable_count }} 枚可采
                </UBadge>
              </div>
              <p class="mt-2 text-sm text-toned">
                {{ zone.note }}
              </p>
              <p class="mt-1 text-xs text-muted">
                起始距离 {{ zone.distance_to_start.toFixed(2) }}
              </p>
            </div>
          </div>
          <UAlert
            v-else
            color="warning"
            variant="subtle"
            icon="i-lucide-map"
            title="暂无可采区域"
            description="当前识别结果中没有可采目标，建议继续观察或切换视角。"
          />
        </UCard>

        <div class="space-y-4">
          <UCard variant="subtle" :ui="{ body: 'p-4' }">
            <template #header>
              <div class="flex items-center justify-between gap-3">
                <h2 class="text-base font-semibold text-highlighted">
                  采摘顺序
                </h2>
                <UBadge color="primary" variant="soft">
                  {{ decision.pick_sequence.length }} 步
                </UBadge>
              </div>
            </template>

            <div v-if="decision.pick_sequence.length" class="space-y-3">
              <div
                v-for="item in decision.pick_sequence"
                :key="`${item.order}-${item.detection_index}`"
                class="rounded-lg border border-default bg-default px-4 py-3"
              >
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <p class="font-medium text-highlighted">
                    第 {{ item.order }} 步 · 检测 #{{ item.detection_index }}
                  </p>
                  <UBadge color="success" variant="soft">
                    {{ formatDecisionZoneLabel(item.zone) }}
                  </UBadge>
                </div>
                <p class="mt-2 text-sm text-toned">
                  {{ item.reason }}
                </p>
                <p class="mt-1 text-xs text-muted">
                  置信度 {{ Math.round(item.confidence * 100) }}%
                  <span v-if="item.track_id !== null"> · track {{ item.track_id }}</span>
                </p>
              </div>
            </div>
            <UAlert
              v-else
              color="warning"
              variant="subtle"
              icon="i-lucide-hand"
              title="当前无采摘顺序"
              description="当前快照没有可采目标，系统暂不建议进入采摘动作。"
            />
          </UCard>

          <UCard variant="subtle" :ui="{ body: 'p-4' }">
            <template #header>
              <div class="flex items-center justify-between gap-3">
                <h2 class="text-base font-semibold text-highlighted">
                  跳过建议
                </h2>
                <UBadge color="warning" variant="soft">
                  {{ decision.skip_items.length }} 项
                </UBadge>
              </div>
            </template>

            <div v-if="decision.skip_items.length" class="space-y-3">
              <div
                v-for="item in decision.skip_items"
                :key="`${item.detection_index}-${item.skip_reason}`"
                class="rounded-lg border border-default bg-default px-4 py-3"
              >
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <p class="font-medium text-highlighted">
                    检测 #{{ item.detection_index }}
                  </p>
                  <UBadge color="warning" variant="soft">
                    {{ formatDecisionSkipReason(item.skip_reason) }}
                  </UBadge>
                </div>
                <p class="mt-2 text-sm text-toned">
                  {{ item.reason }}
                </p>
                <p class="mt-1 text-xs text-muted">
                  {{ formatDecisionZoneLabel(item.zone) }} · 置信度 {{ Math.round(item.confidence * 100) }}%
                </p>
              </div>
            </div>
            <UAlert
              v-else
              color="success"
              variant="subtle"
              icon="i-lucide-check"
              title="当前无跳过项"
              description="当前快照中的目标都已被纳入采摘顺序。"
            />
          </UCard>
        </div>
      </div>
    </div>
  </UCard>
</template>
