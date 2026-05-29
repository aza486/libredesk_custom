<template>
  <span
    v-if="count > 0"
      class="ml-2 bg-primary text-primary-foreground text-xs rounded-full min-w-[1.3rem] h-5 px-1 flex items-center justify-center shrink-0"  >
    {{ count > 9 ? '9+' : count }}
  </span>
</template>
 
<script setup>
defineProps({
  count: {
    type: Number,
    required: true
  }
})
</script>