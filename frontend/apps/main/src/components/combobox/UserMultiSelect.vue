<template>
  <div class="space-y-2">
    <div v-if="selectedUsers.length" class="flex flex-wrap gap-2">
      <Badge v-for="user in selectedUsers" :key="user.value" variant="secondary" class="gap-1 pr-1">
        <span>{{ user.label }}</span>
        <span
          v-if="isLocked(user.value)"
          class="text-muted-foreground"
          title="Kann nicht entfernt werden"
        >
          🔒
        </span>
        <button
          v-else
          type="button"
          class="rounded-sm px-1 text-muted-foreground hover:bg-muted-foreground/15 hover:text-foreground"
          :aria-label="`${user.label} entfernen`"
          @click="removeUser(user.value)"
        >
          ×
        </button>
      </Badge>
    </div>

    <SelectComboBox
      v-if="!readonly"
      :items="availableUsers"
      :placeholder="placeholder"
      type="user"
      :keep-open="true"
      @select="addUser"
    />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import SelectComboBox from './SelectCombobox.vue'
import { Badge } from '@shared-ui/components/ui/badge'

const props = defineProps({
  modelValue: {
    type: Array,
    default: () => []
  },
  items: {
    type: Array,
    default: () => []
  },
  lockedValues: {
    type: Array,
    default: () => []
  },
  readonly: Boolean,
  placeholder: {
    type: String,
    default: 'Mitarbeiter hinzufügen'
  }
})

const emit = defineEmits(['update:modelValue'])

const selectedUsers = computed(() => props.modelValue)
const lockedValues = computed(() => new Set(props.lockedValues.map((value) => String(value))))
const isSelected = (userId) =>
  props.modelValue.some((user) => String(user.value) === String(userId))
const isLocked = (userId) => props.readonly || lockedValues.value.has(String(userId))
const availableUsers = computed(() => props.items.filter((user) => !isSelected(user.value)))

const addUser = (user) => {
  if (isSelected(user.value)) return
  emit('update:modelValue', [...props.modelValue, user])
}

const removeUser = (userId) => {
  if (isLocked(userId)) return
  emit(
    'update:modelValue',
    props.modelValue.filter((user) => String(user.value) !== String(userId))
  )
}
</script>
