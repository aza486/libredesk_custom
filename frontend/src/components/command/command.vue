<template>
    <CommandDialog :open="open" @update:open="handleOpenChange">
        <CommandInput placeholder="Type a command or search..." @keydown="onInputKeydown" />
        <CommandList>
            <CommandEmpty>
                <p class="text-muted-foreground">No command available</p>
            </CommandEmpty>

            <!-- Commands requiring a conversation to be open -->
            <CommandGroup heading="Conversations" value="conversations"
                v-if="nestedCommand === null && conversationStore.current">
                <CommandItem value="conv-snooze" @select="setNestedCommand('snooze')">
                    Snooze
                </CommandItem>
                <CommandItem value="conv-resolve" @select="resolveConversation">
                    Resolve
                </CommandItem>
            </CommandGroup>
            <CommandGroup v-if="nestedCommand === 'snooze'" heading="Snooze for">
                <CommandItem value="snooze-1h" @select="handleSnooze(60)">1 hour</CommandItem>
                <CommandItem value="snooze-3h" @select="handleSnooze(180)">3 hours</CommandItem>
                <CommandItem value="snooze-6h" @select="handleSnooze(360)">6 hours</CommandItem>
                <CommandItem value="snooze-12h" @select="handleSnooze(720)">12 hours</CommandItem>
                <CommandItem value="snooze-1d" @select="handleSnooze(1440)">1 day</CommandItem>
                <CommandItem value="snooze-2d" @select="handleSnooze(2880)">2 days</CommandItem>
                <CommandItem value="snooze-custom" @select="showCustomDialog">Pick date & time</CommandItem>
            </CommandGroup>
        </CommandList>

        <!-- Navigation -->
        <div class="mt-2 px-4 py-2 text-xs text-gray-500 flex space-x-4">
            <span><kbd>Enter</kbd> select</span>
            <span><kbd>↑</kbd>/<kbd>↓</kbd> navigate</span>
            <span><kbd>Esc</kbd> close</span>
            <span><kbd>Backspace</kbd> parent</span>
        </div>
    </CommandDialog>

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
                            {{ selectedDate ? selectedDate : "Pick a date" }}
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
import { ref, watch, onMounted, onUnmounted } from 'vue'
import { useMagicKeys } from '@vueuse/core'
import { CalendarIcon } from 'lucide-vue-next'
import { useConversationStore } from '@/stores/conversation'
import { CONVERSATION_DEFAULT_STATUSES } from '@/constants/conversation'
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
    DialogTitle,
} from '@/components/ui/dialog'
import {
    Popover,
    PopoverContent,
    PopoverTrigger,
} from '@/components/ui/popover'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { Button } from '@/components/ui/button'
import { Calendar } from '@/components/ui/calendar'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

const conversationStore = useConversationStore()
const open = ref(false)
const emitter = useEmitter()
const nestedCommand = ref(null)
const showDatePicker = ref(false)
const selectedDate = ref(null)
const selectedTime = ref('12:00')

const keys = useMagicKeys()
const cmdK = keys['meta+k']
const ctrlK = keys['ctrl+k']

function handleOpenChange () {
    if (!open.value) nestedCommand.value = null
    open.value = !open.value
}

function setNestedCommand (command) {
    nestedCommand.value = command
}

function formatDuration (minutes) {
    return minutes < 60 ? `${minutes}m` : `${Math.floor(minutes / 60)}h`
}

async function handleSnooze (minutes) {
    await conversationStore.snoozeConversation(formatDuration(minutes))
    handleOpenChange()
}

async function resolveConversation () {
    await conversationStore.updateStatus(CONVERSATION_DEFAULT_STATUSES.RESOLVED)
    handleOpenChange()
}

function showCustomDialog () {
    handleOpenChange()
    showDatePicker.value = true
}

function closeDatePicker () {
    showDatePicker.value = false
}

function handleCustomSnooze () {
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

function onInputKeydown (e) {
    if (e.key === 'Backspace') {
        const inputVal = e.target.value || ''
        if (!inputVal && nestedCommand.value !== null) {
            e.preventDefault()
            nestedCommand.value = null
        }
    }
}

function preventDefaultKey (event) {
    if ((event.metaKey || event.ctrlKey) && event.key === 'k') {
        event.preventDefault()
        event.stopPropagation()
        return false
    }
}

onMounted(() => {
    emitter.on(EMITTER_EVENTS.SET_NESTED_COMMAND, (command) => {
        setNestedCommand(command)
        open.value = true
    })
    window.addEventListener('keydown', preventDefaultKey)
})

onUnmounted(() => {
    emitter.off(EMITTER_EVENTS.SET_NESTED_COMMAND)
    window.removeEventListener('keydown', preventDefaultKey)
})

watch([cmdK, ctrlK], ([mac, win]) => {
    if (mac || win) handleOpenChange()
})
</script>
