<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import { useEventListener } from '@vueuse/core'
import { ChevronDown, PanelsTopLeft, Settings2 } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import { useUserStore } from '@main/stores/user'
import api from '@main/api'
import { useMessageTemplates, templateError } from './useMessageTemplates'
import MessageTemplatesManager from './MessageTemplatesManager.vue'
const props = defineProps({ conversationUUID: String, isFullscreen: Boolean, disabled: Boolean })
const emit = defineEmits(['insert', 'captureSelection'])
const user = useUserStore()
const identity = computed(() => JSON.stringify([user.userID, user.roles, user.permissions, user.teams, user.user?.enabled]))
const context = () => props.conversationUUID
const { templates, canSupport, loading, error, load, clear } = useMessageTemplates(
  context,
  () => identity.value
)
const open = ref(false)
const anchor = ref(null)
const placement = ref(props.isFullscreen ? 'bottom' : 'top')
const updatePlacement = () => {
  const rect = anchor.value?.getBoundingClientRect()
  const preferred = props.isFullscreen ? 'bottom' : 'top'
  placement.value = preferred
  if (!rect) return
  const above = rect.top - 12
  const below = window.innerHeight - rect.bottom - 12
  const wanted = Math.min(
    400,
    72 +
      (canSupport.value ? 24 : 0) +
      templates.value.length * 32 +
      (templates.value.length >= 10 ? 44 : 0)
  )
  const preferredSpace = preferred === 'top' ? above : below
  const alternativeSpace = preferred === 'top' ? below : above
  if (preferredSpace < wanted && alternativeSpace > preferredSpace)
    placement.value = preferred === 'top' ? 'bottom' : 'top'
}
useEventListener('resize', () => {
  if (open.value) updatePlacement()
})
const managing = ref(false)
const search = ref('')
const inserting = ref(false)
const focusEditor = ref(false)
const groups = computed(() =>
  ['personal', ...(canSupport.value ? ['customer_support'] : [])].map((scope) => ({
    scope,
    label: scope === 'personal' ? 'Meine Vorlagen' : 'Kundensupport',
    items: templates.value
      .filter(
        (t) =>
          t.scope === scope && t.name.toLocaleLowerCase().includes(search.value.toLocaleLowerCase())
      )
      .sort((a, b) => a.name.localeCompare(b.name, 'de', { sensitivity: 'base' }))
  }))
)
watch(open, async (value) => {
  if (!value) {
    clear()
    return
  }
  emit('captureSelection')
  updatePlacement()
  search.value = ''
  await load()
  if (open.value) updatePlacement()
})
watch([() => props.conversationUUID, identity], () => {
  open.value = false
  managing.value = false
})
const select = async (item) => {
  if (props.disabled || inserting.value) return
  const originalIdentity = identity.value
  const originalContext = context()
  inserting.value = true
  try {
    const { data } = await api.getMessageTemplate(item.uuid, originalContext)
    if (originalIdentity !== identity.value || originalContext !== context() || props.disabled)
      return
    focusEditor.value = true
    open.value = false
    await nextTick()
    emit('insert', data.data.content)
  } catch (err) {
    clear()
    error.value = templateError(err)
  } finally {
    inserting.value = false
  }
}
const afterClose = (event) => {
  if (focusEditor.value || managing.value) event.preventDefault()
  focusEditor.value = false
}
const onSearchKeydown = (event) => {
  event.stopPropagation()
  if (event.key === 'Escape') {
    event.preventDefault()
    open.value = false
  }
  if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
    event.preventDefault()
    const items = [...event.target.closest('[role="menu"]').querySelectorAll('[role="menuitem"]')]
    ;(event.key === 'ArrowDown' ? items[0] : items.at(-1))?.focus()
  }
}
const edit = async () => {
  open.value = false
  await nextTick()
  managing.value = true
}
</script>
<template>
  <div ref="anchor" class="ml-2 border-l pl-2 shrink-0">
    <DropdownMenu v-model:open="open" :modal="false">
      <DropdownMenuTrigger as-child>
        <Button
          type="button"
          variant="ghost"
          class="gap-2 px-2 max-md:min-h-11"
          :disabled="disabled"
          @pointerdown="emit('captureSelection')"
        >
          <PanelsTopLeft class="size-4" /><span>Vorlagen</span><ChevronDown class="size-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        :side="placement"
        align="start"
        :avoid-collisions="true"
        :prioritize-position="true"
        :collision-padding="12"
        class="flex flex-col w-80 max-w-[calc(100vw-24px)] max-h-[min(400px,var(--radix-dropdown-menu-content-available-height))]"
        @close-auto-focus="afterClose"
      >
        <div v-if="templates.length >= 10" class="p-1 shrink-0">
          <Input
            v-model="search"
            placeholder="Vorlagen suchen …"
            aria-label="Vorlagen suchen"
            @keydown="onSearchKeydown"
            @pointermove.stop
          />
        </div>
        <div class="min-h-0 overflow-y-auto overscroll-contain" :aria-busy="loading">
          <p v-if="loading" class="p-3 text-sm text-muted-foreground" role="status">
            Vorlagen werden geladen …
          </p>
          <p v-else-if="error" role="alert" class="p-3 text-sm text-destructive">{{ error }}</p>
          <template v-else v-for="group in groups" :key="group.scope">
            <DropdownMenuLabel class="text-xs text-muted-foreground">{{
              group.label
            }}</DropdownMenuLabel>
            <DropdownMenuItem
              v-for="item in group.items"
              :key="item.uuid"
              :disabled="inserting"
              :title="item.name"
              @select.prevent="select(item)"
            >
              <span class="truncate">{{ item.name }}</span>
            </DropdownMenuItem>
            <p v-if="!group.items.length" class="px-2 pb-2 text-xs text-muted-foreground">
              {{ search ? 'Keine Treffer' : 'Noch keine Vorlagen' }}
            </p>
          </template>
        </div>
        <div class="shrink-0">
          <DropdownMenuSeparator />
          <DropdownMenuItem @select.prevent="edit"
            ><Settings2 class="mr-2 size-4" />Vorlagen bearbeiten</DropdownMenuItem
          >
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
    <MessageTemplatesManager
      v-if="managing"
      v-model:open="managing"
      :conversationUUID="conversationUUID"
      :identity="identity"
      @changed="clear"
    />
  </div>
</template>
