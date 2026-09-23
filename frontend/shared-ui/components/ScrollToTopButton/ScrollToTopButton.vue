<template>
  <Transition
    enter-active-class="transition ease-out duration-200"
    enter-from-class="opacity-0 translate-y-1"
    enter-to-class="opacity-100 translate-y-0"
    leave-active-class="transition ease-in duration-150"
    leave-from-class="opacity-100 translate-y-0"
    leave-to-class="opacity-0 translate-y-1"
  >
    <div
      v-show="!isAtTop"
      class="absolute bottom-[4.25rem] right-6 z-10"
    >
      <button
        @click="$emit('scroll-to-top')"
        class="w-10 h-10 rounded-full flex items-center justify-center shadow-lg border bg-background text-primary transition-colors duration-200 hover:bg-accent"
      >
        <ChevronUp class="w-4 h-4" />
      </button>
    </div>
  </Transition>
</template>

<script setup>
import { ChevronUp } from 'lucide-vue-next'

defineProps({
  isAtTop: {
    type: Boolean,
    required: true
  }
})

defineEmits(['scroll-to-top'])
</script>