<script setup lang="ts">
defineProps<{
  hasHistory: boolean
}>()
</script>

<template>
  <UCard variant="outline" :ui="{ body: 'p-6 sm:p-8' }">
    <div class="space-y-4">
      <p class="text-xs uppercase tracking-widest text-muted">
        Decision
      </p>
      <h1 class="text-2xl font-semibold text-highlighted sm:text-3xl">
        等待识别快照
      </h1>
      <p class="text-sm text-toned sm:text-base">
        决策层会直接消费最近一帧识别快照，生成区域优先级、采摘顺序和跳过建议。
      </p>
      <div class="flex flex-wrap gap-2">
        <UButton to="/recognition" icon="i-lucide-camera" label="前往识别页获取快照" />
        <UBadge
          v-if="hasHistory"
          color="neutral"
          variant="soft"
        >
          已有最近历史可供查看
        </UBadge>
      </div>
    </div>
  </UCard>
</template>
