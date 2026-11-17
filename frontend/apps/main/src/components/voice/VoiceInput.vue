<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { Mic, Square, X, LoaderCircle } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { useVoiceInput } from '@main/composables/useVoiceInput'
import { appendTranscript } from './voice-text'
const props = defineProps({
  modelValue: { type: String, default: '' },
  disabled: Boolean,
  active: { type: Boolean, default: true },
  contextKey: { type: [String, Number], default: '' },
  maxChars: { type: Number, default: Infinity },
  maxBytes: { type: Number, default: Infinity }
})
const emit = defineEmits(['update:modelValue', 'busy'])
const limitError = ref(false)
const { state, error, start, stop, cancel } = useVoiceInput((text) => {
  const next = appendTranscript(props.modelValue, text, props.maxChars, props.maxBytes)
  limitError.value = next === null
  if (next !== null) emit('update:modelValue', next)
})
const busy = computed(() => ['requesting', 'recording', 'processing'].includes(state.value))
watch(busy, (value) => emit('busy', value), { immediate: true })
onBeforeUnmount(() => emit('busy', false))
watch(
  () => [props.active, props.contextKey, props.disabled],
  () => {
    cancel()
    limitError.value = false
  }
)
function begin() {
  limitError.value = false
  start()
}
</script>
<template>
  <div class="flex min-w-0 flex-wrap items-center gap-2 text-xs" data-voice-input>
    <Button
      v-if="state === 'recording'"
      type="button"
      variant="destructive"
      size="sm"
      :aria-label="$t('voiceInput.stop')"
      @click="stop"
    >
      <Square class="h-4 w-4" /> {{ $t('voiceInput.recording') }}
    </Button>
    <Button
      v-else
      type="button"
      variant="ghost"
      size="icon"
      class="h-8 w-8 shrink-0"
      :disabled="disabled || !active || busy"
      :aria-label="$t('voiceInput.start')"
      @click="begin"
    >
      <LoaderCircle v-if="busy" class="h-4 w-4 animate-spin" />
      <Mic v-else class="h-4 w-4" />
    </Button>
    <Button
      v-if="busy"
      type="button"
      variant="ghost"
      size="icon"
      class="h-8 w-8"
      :aria-label="$t('voiceInput.cancel')"
      @click="cancel"
      ><X class="h-4 w-4"
    /></Button>
    <span
      v-if="state !== 'error' && !limitError"
      role="status"
      aria-live="polite"
      class="text-muted-foreground"
    >
      {{ state === 'ready' ? '' : $t(`voiceInput.${state}`) }}
    </span>
    <span v-if="error || limitError" role="alert" class="text-destructive">
      {{ $t(`voiceInput.errors.${limitError ? 'limit' : error}`) }}
    </span>
  </div>
</template>
