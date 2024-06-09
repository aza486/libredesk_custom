<script setup lang="ts">
import { LoaderCircle } from 'lucide-vue-next'


defineProps({
  loading: {
    type: Boolean,
    default: false
  },
  text: {
    type: String,
    default: 'Loading'
  }
});
</script>

<template>
  <div v-if="loading" class="flex">
    <LoaderCircle class="animate-spin size-6 mr-2" /> {{ text }}
  </div>
</template>