<template>
  <div
    class="box flex-1 rounded-lg px-8 py-4 transition-shadow duration-170 cursor-pointer hover:bg-muted"
    @click="handleClick"
  >
    <div class="flex items-center mb-4">
      <component :is="icon" size="16" class="mr-2" />
      <p class="text-lg">{{ title }}</p>
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
    type: Function,
    required: true
  },
  onClick: {
    type: Function,
    default: null
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
