<template>
  <div class="flex flex-1 flex-col gap-x-5 box p-5 space-y-5 bg-white">
    <div class="flex items-center space-x-2">
      <p class="text-2xl">{{ title }}</p>
      <div class="bg-green-100/70 flex items-center space-x-2 px-1 rounded">
        <span class="blinking-dot"></span>
        <p class="uppercase text-xs">Live</p>
      </div>
    </div>
    <div class="flex justify-between pr-32">
      <div v-for="(value, key) in counts" :key="key" class="flex flex-col items-center gap-y-2">
        <span class="text-muted-foreground">{{ labels[key] }}</span>
        <span class="text-2xl font-medium">{{ value }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  counts: { type: Object, required: true },
  labels: { type: Object, required: true },
  title: { type: String, required: true }
})
</script>
