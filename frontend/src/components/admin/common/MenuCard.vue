<template>
  <div
    class="max-w-sm overflow-hidden box border px-6 py-6 hover:shadow-md transition-shadow duration-170 cursor-pointer hover:bg-muted"
    @click="handleClick"
  >
    <div class="flex items-center mb-4">
      <component :is="icon" size="20" class="mr-2" />
      <span class="text-base">{{ title }}</span>
    </div>
    <p class="text-gray-600 text-sm">{{ subtitle }}</p>
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
