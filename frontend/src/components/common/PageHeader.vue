<template>
  <div>
    <span class="text-2xl">
      {{ title }}
    </span>
    <p class="text-sm-muted">
      {{ subTitle }}
    </p>
  </div>
  <Separator class="my-3" />
</template>

<script setup>
import { Separator } from '@/components/ui/separator'
defineProps({
  title: {
    type: String,
    required: true
  },
  subTitle: {
    type: String,
    required: true
  }
})
</script>
