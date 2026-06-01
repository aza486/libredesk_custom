<template>
  <ComboBox
    :model-value="normalizedValue"
    @update:model-value="$emit('update:modelValue', $event)"
    @select="$emit('select', $event)"
    :items="sortedItems"
    :placeholder="placeholder"
    :align="align"
  >
    <!-- Custom trigger passthrough -->
    <template v-if="$slots.trigger" #trigger="slotProps">
      <slot name="trigger" v-bind="slotProps" />
    </template>

    <!-- Items -->
    <template #item="{ item }">
      <div class="flex items-center gap-2">
        <!--USER -->
        <div v-if="type === 'user'" class="relative">
          <Avatar class="w-7 h-7">
            <AvatarImage :src="item.avatar_url || ''" :alt="item.label.slice(0, 2)" />
            <AvatarFallback>{{ item.label.slice(0, 2).toUpperCase() }}</AvatarFallback>
          </Avatar>
          <StatusDot
            v-if="item.availability_status"
            :status="item.availability_status"
            size="sm"
            class="absolute bottom-0 right-0 border border-background"
          />
        </div>

        <!-- Others -->
        <span v-else-if="item.emoji">{{ item.emoji }}</span>
        <div class="flex items-center justify-between w-full">

                <div class="flex items-center gap-2">
                  <span>{{ item.label }}</span>

                  <span
                    v-if="isCurrentUser(item)"
                    class="text-muted-foreground text-xs"
                  >
                    ({{ t('globals.terms.you') }})
                  </span>
                </div>

                <button
                  v-if="type === 'user' && item.value !== 'none'"
                  class="text-yellow-500 text-sm"
                  @click.stop="toggleFavorite(item.value)">
                  {{
                    favoriteAgents.includes(String(item.value))
                      ? '★'
                      : '☆'
                  }}
                </button>

              </div>
        <span v-if="isCurrentUser(item)" class="text-muted-foreground text-xs"
          >({{ t('globals.terms.you') }})</span>
      </div>
    </template>

    <!-- Selected -->
    <template #selected="{ selected }">
      <div class="flex items-center gap-2 min-w-0">
        <div v-if="selected" class="flex items-center gap-2 min-w-0">
          <!--USER -->
          <div v-if="type === 'user'" class="relative shrink-0">
            <Avatar class="w-7 h-7">
              <AvatarImage :src="selected.avatar_url || ''" :alt="selected.label.slice(0, 2)" />
              <AvatarFallback>{{ selected.label.slice(0, 2).toUpperCase() }}</AvatarFallback>
            </Avatar>
            <StatusDot
              v-if="selected.availability_status"
              :status="selected.availability_status"
              size="sm"
              class="absolute bottom-0 right-0 border border-background"
            />
          </div>

          <!-- Others -->
          <span v-else-if="selected.emoji" class="shrink-0">{{ selected.emoji }}</span>
          <span class="truncate">{{ selected.label }}</span>
          <span
            v-if="isCurrentUser(selected)"
            class="text-muted-foreground text-xs shrink-0"
            >({{ t('globals.terms.you') }})</span
          >
        </div>
        <span v-else class="truncate">{{ placeholder }}</span>
      </div>
    </template>
  </ComboBox>
</template>

<script setup>
import { computed } from 'vue'
import { useStorage } from '@vueuse/core'
import { Avatar, AvatarImage, AvatarFallback } from '@shared-ui/components/ui/avatar'
import ComboBox from '@shared-ui/components/ui/combobox/ComboBox.vue'
import StatusDot from '@shared-ui/components/StatusDot.vue'
import { useUserStore } from '@/stores/user'
import { useI18n } from 'vue-i18n'
const { t } = useI18n()
const userStore = useUserStore()
const favoriteAgents = useStorage(
  'favoriteAgents',
  []
)

const props = defineProps({
  modelValue: [String, Number, Object],
  placeholder: String,
  items: Array,
  type: {
    type: String
  },
  align: {
    type: String,
    default: 'center'
  }
})

const normalizedValue = computed(() => String(props.modelValue || ''))
const sortedItems = computed(() => {

  if (props.type !== 'user') {
    return props.items
  }

  return [...props.items].sort((a, b) => {

    // "none" immer unten
    if (a.value === 'none') return 1
    if (b.value === 'none') return -1

    const aFav =
      favoriteAgents.value.includes(String(a.value))

    const bFav =
      favoriteAgents.value.includes(String(b.value))

    if (aFav && !bFav) return -1
    if (!aFav && bFav) return 1

    return a.label.localeCompare(b.label)
  })
})

const isCurrentUser = (item) => {
  return (
    props.type === 'user' &&
    item.value !== 'none' &&
    String(item.value) === String(userStore.userID)
  )
}

const toggleFavorite = (userId) => {

  const id = String(userId)

  if (favoriteAgents.value.includes(id)) {
    favoriteAgents.value =
      favoriteAgents.value.filter(fav => fav !== id)
  } else {
    favoriteAgents.value.push(id)
  }
}

defineEmits(['update:modelValue', 'select'])
</script>
