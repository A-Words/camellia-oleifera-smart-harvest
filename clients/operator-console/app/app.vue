<script setup lang="ts">
import AppTopNav from '~/components/navigation/AppTopNav.vue'
import { shouldShowTopNav } from '~/utils/top-nav-visibility'
</script>

<template>
  <UApp>
    <AppTopNav v-if="shouldShowTopNav()" />
    <NuxtPage />
  </UApp>
</template>
