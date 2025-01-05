<template>
  <div
    class="flex-1 rounded-xl px-6 py-4 border border-muted shadow-md hover:shadow-lg transition-transform duration-200 transform hover:scale-105 cursor-pointer bg-white max-w-80"
    @click="handleClick">
    <div class="flex items-center mb-3">
      <component :is="icon" size="24" class="mr-2 text-primary" />
      <p class="text-lg font-semibold text-gray-700">{{ title }}</p>
    </div>
    <p class="text-sm text-gray-500">{{ subTitle }}</p>
  </div>
</template>

<script setup>
import { defineProps, defineEmits } from 'vue'

const props = defineProps({
  title: String,
  subTitle: String,
  icon: Function,
  onClick: {
    type: Function,
    default: null
  }
})

const emit = defineEmits(['click'])

const handleClick = () => {
  if (props.onClick) props.onClick()
  emit('click')
}
</script>
