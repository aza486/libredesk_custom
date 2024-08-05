<template>
  <div>
    <span class="sub-title">{{ title }}</span>
    <p class="text-xs-muted">{{ description }}</p>
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
});
</script>