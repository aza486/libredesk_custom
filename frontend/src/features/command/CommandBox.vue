<template>
  <CommandDialog :open="open" @update:open="handleOpenChange" class="z-[51] !min-w-[50vw] !min-h-[60vh]">
    <CommandInput placeholder="Type a command or search..." @keydown="onInputKeydown" />
    <CommandList class="!min-h-[60vh] !min-w-[50vw]">
      <CommandEmpty>
        <p class="text-muted-foreground">No command available</p>
      </CommandEmpty>

      <!-- Commands requiring a conversation to be open -->
      <CommandGroup
        heading="Conversations"
        value="conversations"
        v-if="nestedCommand === null && conversationStore.hasConversationOpen"
      >
        <CommandItem value="conv-snooze" @select="setNestedCommand('snooze')"> Snooze </CommandItem>
        <CommandItem value="conv-resolve" @select="resolveConversation"> Resolve </CommandItem>
        <CommandItem value="apply-macro" @select="setNestedCommand('apply-macro')">
          Apply macro
        </CommandItem>
      </CommandGroup>

      <CommandGroup v-if="nestedCommand === 'snooze'" heading="Snooze for">
        <CommandItem value="1 hour" @select="handleSnooze(60)">1 hour</CommandItem>
        <CommandItem value="3 hours" @select="handleSnooze(180)">3 hours</CommandItem>
        <CommandItem value="6 hours" @select="handleSnooze(360)">6 hours</CommandItem>
        <CommandItem value="12 hours" @select="handleSnooze(720)">12 hours</CommandItem>
        <CommandItem value="1 day" @select="handleSnooze(1440)">1 day</CommandItem>
        <CommandItem value="2 days" @select="handleSnooze(2880)">2 days</CommandItem>
        <CommandItem value="pick date & time" @select="showCustomDialog">
          Pick date & time
        </CommandItem>
      </CommandGroup>

      <!-- Macros -->
      <!-- TODO move to a separate component -->
      <div v-if="nestedCommand === 'apply-macro'" class="bg-background">
        <CommandGroup heading="Apply macro" class="pb-2">
          <div class="min-h-[400px] overflow-auto">
            <div class="grid grid-cols-12 gap-3">
              <div class="col-span-4 pr-2">
                <CommandItem
                  v-for="(macro, index) in macroStore.macroOptions"
                  :key="macro.value"
                  :value="macro.label"
                  :data-index="index"
                  @select="handleApplyMacro(macro)"
                  class="px-3 py-2 rounded-md cursor-pointer transition-all duration-200 hover:bg-primary/10 hover:text-primary"
                >
                  <div class="flex items-center gap-2">
                    <Zap size="14" class="text-primary shrink-0" />
                    <span class="text-sm truncate w-full break-words whitespace-normal">{{
                      macro.label
                    }}</span>
                  </div>
                </CommandItem>
              </div>

              <div class="col-span-8 pl-2 border-l">
                <div class="space-y-3 text-xs">
                  <div v-if="replyContent" class="space-y-1">
                    <p class="text-xs font-semibold text-primary">Reply Preview</p>
                    <div
                      class="w-full min-h-200 p-2 bg-muted/50 rounded-md overflow-auto shadow-sm native-html"
                      v-dompurify-html="replyContent"
                    />
                  </div>

                  <div v-if="otherActions.length > 0" class="space-y-1">
                    <p class="text-xs font-semibold text-primary">Actions</p>
                    <div class="space-y-1.5 max-w-sm">
                      <div
                        v-for="action in otherActions"
                        :key="action.type"
                        class="flex items-center gap-2 px-2 py-1.5 bg-muted/30 hover:bg-accent hover:text-accent-foreground rounded-md text-xs transition-all duration-200 group"
                      >
                        <div
                          class="p-1 bg-primary/10 rounded-full group-hover:bg-primary/20 transition-colors duration-200"
                        >
                          <User
                            v-if="action.type === 'assign_user'"
                            :size="10"
                            class="shrink-0 text-primary"
                          />
                          <Users
                            v-else-if="action.type === 'assign_team'"
                            :size="10"
                            class="shrink-0 text-primary"
                          />
                          <Pin
                            v-else-if="action.type === 'set_status'"
                            :size="10"
                            class="shrink-0 text-primary"
                          />
                          <Rocket
                            v-else-if="action.type === 'set_priority'"
                            :size="10"
                            class="shrink-0 text-primary"
                          />
                          <Tags
                            v-else-if="action.type === 'set_tags'"
                            :size="10"
                            class="shrink-0 text-primary"
                          />
                        </div>
                        <span class="truncate">{{ getActionLabel(action) }}</span>
                      </div>
                    </div>
                  </div>
                  <div
                    v-if="!replyContent && otherActions.length === 0"
                    class="flex items-center justify-center h-20"
                  >
                    <p class="text-xs text-muted-foreground italic">
                      Select a macro to view details
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </CommandGroup>
      </div>
    </CommandList>

    <!-- Navigation -->
    <!-- TODO: Move to a separate component -->
    <div class="mt-2 px-4 py-2 text-xs text-gray-500 flex space-x-4">
      <span><kbd>Enter</kbd> select</span>
      <span><kbd>↑</kbd>/<kbd>↓</kbd> navigate</span>
      <span><kbd>Esc</kbd> close</span>
      <span><kbd>Backspace</kbd> parent</span>
    </div>
  </CommandDialog>

  <!-- Date Picker for Custom Snooze -->
  <!-- TODO: Move to a separate component -->
  <Dialog :open="showDatePicker" @update:open="closeDatePicker">
    <DialogContent class="sm:max-w-[425px]">
      <DialogHeader>
        <DialogTitle>Pick Snooze Time</DialogTitle>
      </DialogHeader>
      <div class="grid gap-4 py-4">
        <Popover>
          <PopoverTrigger as-child>
            <Button variant="outline" class="w-full justify-start text-left font-normal">
              <CalendarIcon class="mr-2 h-4 w-4" />
              {{ selectedDate ? selectedDate : 'Pick a date' }}
            </Button>
          </PopoverTrigger>
          <PopoverContent class="w-auto p-0">
            <Calendar mode="single" v-model="selectedDate" />
          </PopoverContent>
        </Popover>
        <div class="grid gap-2">
          <Label>Time</Label>
          <Input type="time" v-model="selectedTime" />
        </div>
      </div>
      <DialogFooter>
        <Button @click="handleCustomSnooze">Snooze</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { ref, watch, onMounted, onUnmounted, computed } from 'vue'
import { useMagicKeys } from '@vueuse/core'
import { CalendarIcon } from 'lucide-vue-next'
import { useConversationStore } from '@/stores/conversation'
import { useMacroStore } from '@/stores/macro'
import { CONVERSATION_DEFAULT_STATUSES } from '@/constants/conversation'
import { Users, User, Pin, Rocket, Tags, Zap } from 'lucide-vue-next'
import {
  CommandDialog,
  CommandInput,
  CommandList,
  CommandEmpty,
  CommandGroup,
  CommandItem
} from '@/components/ui/command'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@/components/ui/dialog'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { Button } from '@/components/ui/button'
import { Calendar } from '@/components/ui/calendar'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

const conversationStore = useConversationStore()
const macroStore = useMacroStore()

const open = ref(false)
const emitter = useEmitter()
const nestedCommand = ref(null)
const showDatePicker = ref(false)
const selectedDate = ref(null)
const selectedTime = ref('12:00')

const { Meta_K, Ctrl_K } = useMagicKeys({
  passive: false,
  onEventFired(e) {
    if (e.key === 'k' && (e.metaKey || e.ctrlKey)) {
      e.preventDefault()
    }
  }
})

watch([Meta_K, Ctrl_K], ([mac, win]) => {
  if (mac || win) handleOpenChange()
})

const highlightedMacro = ref(null)

function handleApplyMacro(macro) {
  // Create a deep copy.
  const plainMacro = JSON.parse(JSON.stringify(macro))
  conversationStore.setMacro(plainMacro)
  handleOpenChange()
}

const getActionLabel = computed(() => (action) => {
  const prefixes = {
    assign_user: 'Assign to user',
    assign_team: 'Assign to team',
    set_status: 'Set status',
    set_priority: 'Set priority',
    set_tags: 'Set tags'
  }
  return `${prefixes[action.type]}: ${action.display_value.length > 0 ? action.display_value.join(', ') : action.value.join(', ')}`
})

const replyContent = computed(() => highlightedMacro.value?.message_content || '')

const otherActions = computed(
  () =>
    highlightedMacro.value?.actions?.filter(
      (a) => a.type !== 'send_private_note' && a.type !== 'send_reply'
    ) || []
)

function handleOpenChange() {
  if (!open.value) nestedCommand.value = null
  open.value = !open.value
}

function setNestedCommand(command) {
  nestedCommand.value = command
}

function formatDuration(minutes) {
  return minutes < 60 ? `${minutes}m` : `${Math.floor(minutes / 60)}h`
}

async function handleSnooze(minutes) {
  await conversationStore.snoozeConversation(formatDuration(minutes))
  handleOpenChange()
}

async function resolveConversation() {
  await conversationStore.updateStatus(CONVERSATION_DEFAULT_STATUSES.RESOLVED)
  handleOpenChange()
}

function showCustomDialog() {
  handleOpenChange()
  showDatePicker.value = true
}

function closeDatePicker() {
  showDatePicker.value = false
}

function handleCustomSnooze() {
  const [hours, minutes] = selectedTime.value.split(':')
  const snoozeDate = new Date(selectedDate.value)
  snoozeDate.setHours(parseInt(hours), parseInt(minutes))
  const diffMinutes = Math.floor((snoozeDate - new Date()) / (1000 * 60))

  if (diffMinutes <= 0) {
    alert('Select a future time')
    return
  }
  handleSnooze(diffMinutes)
  closeDatePicker()
  handleOpenChange()
}

function onInputKeydown(e) {
  if (e.key === 'Backspace') {
    const inputVal = e.target.value || ''
    if (!inputVal && nestedCommand.value !== null) {
      e.preventDefault()
      nestedCommand.value = null
    }
  }
}

onMounted(() => {
  emitter.on(EMITTER_EVENTS.SET_NESTED_COMMAND, (command) => {
    setNestedCommand(command)
    open.value = true
  })
  watchHighlightedMacro()
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.SET_NESTED_COMMAND)
})

const watchHighlightedMacro = () => {
  const observer = new MutationObserver(() => {
    const highlightedEl = document.querySelector('[data-highlighted]')?.getAttribute('data-index')
    highlightedMacro.value = highlightedEl ? macroStore.macroOptions[highlightedEl] : null
  })

  observer.observe(document.body, {
    attributes: true,
    subtree: true
  })
}
</script>
