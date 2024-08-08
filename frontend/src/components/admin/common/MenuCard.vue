<template>
  <div
    class="max-w-sm overflow-hidden box rounded-lg px-6 py-4 transition-shadow duration-170 cursor-pointer hover:bg-muted"
    @click="handleClick">
    <div class="flex items-center mb-4">
      <component :is="icon" size="17" class="mr-2" />
      <span>{{ title }}</span>
    </div>
    <p class="text-xs-muted">{{ subTitle }}</p>
  </div>
</template>

<script setup>
import { defineProps, defineEmits } from 'vue'

const props = defineProps({
  title: {
    type: String,
    required: true
  },
  subTitle: {
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
