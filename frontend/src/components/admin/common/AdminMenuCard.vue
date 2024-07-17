<template>
  <div
    class="max-w-sm rounded overflow-hidden box border p-6 hover:shadow-lg transition-shadow duration-200 cursor-pointer hover:bg-muted"
    @click="handleClick"
  >
    <div class="flex items-center mb-4">
      <component :is="icon" size="24" class="mr-2" />
      <span class="text-base">{{ title }}</span>
    </div>
    <p class="text-gray-600">{{ subtitle }}</p>
  </div>
</template>

<script setup>
import { defineProps, defineEmits } from 'vue'

const props = defineProps({
  title: {
    type: String,
    required: true
  },
  subtitle: {
    type: String,
    required: true
  },
  icon: {
    type: String,
    required: true
  },
  onClick: {
    type: Function,
    required: false
  }
})

const emit = defineEmits(['click'])

const handleClick = () => {
  if (props.onClick) {
    props.onClick()
  }
  emit('click')
}
</script>
