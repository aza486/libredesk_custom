<template>
  <div class="flex">
    <div class="flex flex-col gap-x-5 box p-5 rounded-md space-y-5">
      <div class="flex items-center space-x-2">
        <p class="text-2xl">{{title}}</p>
        <div class="bg-green-100/70 flex items-center space-x-2 px-1 rounded">
          <span class="blinking-dot"></span>
          <strong class="uppercase tracking-wider">Live</strong>
        </div>
      </div>
      <div class="flex">
        <div v-for="(value, key) in counts" :key="key" class="flex flex-col items-center gap-y-2">
          <span class="text-muted-foreground">{{ labels[key] }}</span>
          <span class="text-2xl font-medium">{{ value }}</span>
        </div>
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
