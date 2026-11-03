<script setup>
import { ArrowLeft } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter
} from '@shared-ui/components/ui/dialog'

defineProps({
  open: Boolean,
  embedded: Boolean,
  title: { type: String, required: true },
  description: { type: String, default: '' },
  closeDisabled: Boolean,
  contentClass: { type: String, default: '' }
})
const emit = defineEmits(['update:open', 'closeAutoFocus'])
</script>

<template>
  <component
    :is="embedded ? 'section' : Dialog"
    v-if="!embedded || open"
    v-bind="embedded ? { 'aria-label': title, 'data-ai-context': '' } : { open }"
    :class="embedded ? 'h-full min-h-0 min-w-0 overflow-y-auto overscroll-contain' : undefined"
    @update:open="emit('update:open', $event)"
  >
    <component
      :is="embedded ? 'div' : DialogContent"
      :class="embedded ? 'grid min-w-0 gap-4 p-3 md:p-4 [overflow-wrap:anywhere]' : contentClass"
      @close-auto-focus="emit('closeAutoFocus', $event)"
    >
      <Button
        v-if="embedded"
        variant="ghost"
        class="w-fit max-w-full justify-start"
        :disabled="closeDisabled"
        @click="emit('update:open', false)"
      >
        <ArrowLeft class="h-4 w-4 shrink-0" />
        Zurück zum Verlauf
      </Button>
      <DialogHeader :class="{ 'min-w-0 text-left': embedded }">
        <component
          :is="embedded ? 'h2' : DialogTitle"
          :class="{ 'text-lg font-semibold': embedded }"
          >{{ title }}</component
        >
        <component
          :is="embedded ? 'p' : DialogDescription"
          :class="{ 'text-sm text-muted-foreground': embedded }"
          >{{ description }}</component
        >
      </DialogHeader>
      <slot />
      <DialogFooter :class="{ 'flex-row flex-wrap justify-end gap-2': embedded }">
        <slot name="footer" />
      </DialogFooter>
    </component>
  </component>
</template>
