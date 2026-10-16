<script setup>
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import api from '@main/api'
import { createStyleHistory, styleTextLength } from '@main/utils/ai-style'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter
} from '@shared-ui/components/ui/dialog'
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem
} from '@shared-ui/components/ui/select'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { Alert, AlertDescription } from '@shared-ui/components/ui/alert'
import { Button } from '@shared-ui/components/ui/button'
const props = defineProps({
  open: Boolean,
  note: { type: Object, default: null },
  conversationUUID: { type: String, default: '' }
})
const emit = defineEmits(['update:open', 'adopt'])
const presets = [
  ['mine', 'Mein Stil'],
  ['friendly', 'Freundlich'],
  ['professional', 'Professionell'],
  ['empathetic', 'Empathisch'],
  ['short', 'Kurz & sachlich'],
  ['neutral', 'Neutral']
]
const style = ref('mine'),
  instructions = ref(''),
  busy = ref(false),
  error = ref(''),
  originalView = ref(false)
const history = reactive(createStyleHistory(''))
let generation = 0,
  controller
const cancel = () => {
  generation++
  controller?.abort()
  busy.value = false
}
watch(
  () => [props.open, props.note, props.conversationUUID],
  () => {
    cancel()
    if (props.open) {
      history.original = props.note?.suggestion || ''
      history.reset()
      style.value = 'mine'
      instructions.value = ''
      error.value = ''
      originalView.value = false
    }
  },
  { immediate: true }
)
onBeforeUnmount(cancel)
const tooLong = computed(() => styleTextLength(instructions.value) > 1000)
const visibleText = computed(() => (originalView.value ? history.original : history.current))
async function adjust() {
  if (busy.value || tooLong.value || !props.note) return
  const run = ++generation
  controller = new AbortController()
  busy.value = true
  error.value = ''
  try {
    const response = await api.adjustAIStyle(
      props.conversationUUID,
      {
        note_uuid: props.note.noteUUID,
        style: style.value,
        instructions: instructions.value,
        original_text: history.original,
        current_text: history.current
      },
      controller.signal
    )
    if (run !== generation || !props.open) return
    const result = response?.data
    if (
      !result?.success ||
      typeof result.request_id !== 'string' ||
      !result.adapted_text?.trim() ||
      styleTextLength(result.adapted_text) > 8000
    )
      throw new Error('Ungültige Antwort')
    history.push(result.adapted_text)
    originalView.value = false
  } catch (e) {
    if (run === generation && props.open)
      error.value =
        e.response?.data?.error?.message || 'Stilanpassung fehlgeschlagen. Bitte erneut versuchen.'
  } finally {
    if (run === generation) busy.value = false
  }
}
function undo() {
  history.undo()
  originalView.value = false
}
function reset() {
  history.reset()
  originalView.value = false
}
function adopt() {
  if (busy.value || originalView.value || history.versions.length < 2) return
  emit('adopt', { note: props.note, text: history.current })
  emit('update:open', false)
}
</script>
<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="max-h-[90dvh] overflow-y-auto sm:max-w-2xl">
      <DialogHeader>
        <DialogTitle>Stil anpassen</DialogTitle>
        <DialogDescription
          >Formuliere den KI-Vorschlag um. Prüfe Inhalt und Angaben vor der
          Übernahme.</DialogDescription
        >
      </DialogHeader>
      <label for="ai-style-preset" class="text-sm font-medium">Stil</label>
      <Select v-model="style" :disabled="busy">
        <SelectTrigger id="ai-style-preset"><SelectValue /></SelectTrigger>
        <SelectContent
          ><SelectItem v-for="[value, label] in presets" :key="value" :value="value">{{
            label
          }}</SelectItem></SelectContent
        >
      </Select>
      <label for="ai-style-instructions" class="text-sm font-medium"
        >Zusätzliche Anweisungen (optional)</label
      >
      <Textarea
        id="ai-style-instructions"
        v-model="instructions"
        :disabled="busy"
        rows="3"
        :aria-invalid="tooLong"
        aria-describedby="ai-style-limit"
        placeholder="Bitte in zwei Absätzen formulieren, freundlich aber professionell"
      />
      <p
        id="ai-style-limit"
        class="text-xs"
        :class="tooLong ? 'text-destructive' : 'text-muted-foreground'"
      >
        {{ styleTextLength(instructions) }} / 1000 Zeichen
      </p>
      <Button :disabled="busy || tooLong" @click="adjust">{{
        busy ? 'Wird angepasst …' : 'Anpassen'
      }}</Button>
      <div class="flex flex-wrap gap-2" aria-label="Vorschau auswählen">
        <Button
          variant="outline"
          size="sm"
          :aria-pressed="originalView"
          @click="originalView = true"
          >Original</Button
        >
        <Button
          variant="outline"
          size="sm"
          :aria-pressed="!originalView"
          @click="originalView = false"
          >Angepasst</Button
        >
      </div>
      <div
        class="max-h-64 overflow-auto whitespace-pre-wrap break-words rounded-md border bg-muted p-3 text-sm"
        aria-label="Antwortvorschau"
        aria-live="polite"
      >
        {{ visibleText }}
      </div>
      <div class="flex flex-wrap gap-2">
        <Button
          variant="ghost"
          size="sm"
          :disabled="busy || history.versions.length < 2"
          @click="undo"
          >Letzte Änderung rückgängig</Button
        >
        <Button
          variant="ghost"
          size="sm"
          :disabled="busy || history.versions.length < 2"
          @click="reset"
          >Original wiederherstellen</Button
        >
      </div>
      <Alert v-if="error" variant="destructive" role="alert"
        ><AlertDescription>{{ error }}</AlertDescription></Alert
      >
      <DialogFooter>
        <Button variant="outline" @click="emit('update:open', false)">Verwerfen</Button>
        <Button :disabled="busy || originalView || history.versions.length < 2" @click="adopt"
          >Übernehmen</Button
        >
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
