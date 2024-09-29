<template>
  <div class="flex flex-col items-center justify-center h-64 space-y-2">
    <component :is="icon" :stroke-width="1.4" :size="90" />
    <h1 class="text-lg font-semibold text-gray-800">
      {{ title }}
    </h1>
    <p class="text-gray-600 text-center">
      {{ message }}
    </p>
  </div>
</template>

<script setup>
defineProps({
  icon: {
    type: Object,
    required: true
  },
  title: {
    type: String,
    required: true
  },
  message: {
    type: String,
    required: true
  }
})
</script>
