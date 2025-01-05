<template>
  <div class="flex flex-col space-y-1 border-b pb-3 mb-5 border-gray-200">
    <span class="font-semibold text-2xl">{{ title }}</span>
    <p class="text-muted-foreground text-lg">{{ description }}</p>
  </div>
</template>

<script setup>
defineProps({
  title: {
    type: String,
    required: true
  },
  description: {
    type: String,
    required: true
  }
})
</script>
