<template>
  <div v-if="state !== 'idle'" class="flex items-center gap-3 text-xs">
    <span v-if="state === 'saving'" class="text-muted-foreground"
      >⟳ Änderungen werden gespeichert…</span
    >
    <span v-else-if="state === 'saved'" class="text-green-600 dark:text-green-400">
      ✓ Änderungen gespeichert
    </span>
    <span v-else class="text-destructive">⚠ Änderungen konnten nicht gespeichert werden</span>
    <button
      v-if="canUndo"
      type="button"
      class="text-muted-foreground underline underline-offset-2 hover:text-foreground"
      @click="$emit('undo')"
    >
      Rückgängig
    </button>
  </div>
</template>

<script setup>
defineProps({
  state: {
    type: String,
    default: 'idle'
  },
  canUndo: Boolean
})

defineEmits(['undo'])
</script>
