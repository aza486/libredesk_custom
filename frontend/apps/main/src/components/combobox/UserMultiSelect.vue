<template>
  <div class="space-y-2">

    <!-- Ausgewählte Benutzer -->
    <div
      v-if="selectedUsers.length"
      class="flex flex-wrap gap-2"
    >
      <Badge
        v-for="user in selectedUsers"
        :key="user.value"
        variant="secondary"
        class="cursor-pointer"
        @click="removeUser(user.value)"
      >
        {{ user.label }} ✕
      </Badge>
    </div>

    <!-- Auswahl -->
     <SelectComboBox
      :items="items"
      :placeholder="placeholder"
      @select="addUser"
      type="user"
      :keep-open="true"
      >

      <template #item-right="{ item }">
        <span
          v-if="isSelected(item.value)"
          class="font-bold text-green-500"
        >
          ✓
        </span>
      </template>

    </SelectComboBox>

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
  placeholder: {
    type: String,
    default: 'Benutzer auswählen'
  }
})

const emit = defineEmits([
  'update:modelValue'
])

const selectedUsers = computed(() => {
  return props.modelValue
})

const isSelected = (userId) => {

  return props.modelValue.some(
    user =>
      Number(user.value) === Number(userId)
  )
}

const addUser = (user) => {

  if (isSelected(user.value)) {

    removeUser(user.value)
    return
  }

  emit(
    'update:modelValue',
    [
      ...props.modelValue,
      user
    ]
  )
}

const removeUser = (userId) => {

  emit(
    'update:modelValue',
    props.modelValue.filter(
      user =>
        Number(user.value) !== Number(userId)
    )
  )
}
</script>