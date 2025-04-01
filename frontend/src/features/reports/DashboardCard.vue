<template>
  <div class="flex flex-1 flex-col gap-x-5 box p-5 space-y-5 bg-white">
    <div class="flex items-center space-x-2">
      <p class="text-2xl flex items-center">{{ title }}</p>
      <div class="bg-green-100/70 flex items-center space-x-2 px-1 rounded">
        <span class="blinking-dot"></span>
        <p class="uppercase text-xs">{{ $t('globals.terms.live') }}</p>
      </div>
    </div>
    <div class="flex justify-between pr-32">
      <div
        v-for="(item, key) in filteredCounts"
        :key="key"
        class="flex flex-col items-center gap-y-2"
      >
        <span class="text-muted-foreground">{{ labels[key] }}</span>
        <span class="text-2xl font-medium">{{ item }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  counts: { type: Object, required: true },
  labels: { type: Object, required: true },
  title: { type: String, required: true }
})

// Filter out counts that don't have a label
const filteredCounts = computed(() => {
  return Object.fromEntries(Object.entries(props.counts).filter(([key]) => props.labels[key]))
})
</script>
