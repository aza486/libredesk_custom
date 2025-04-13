<template>
  <div class="w-full space-y-6 pb-8 relative">
    <!-- Header -->
    <div class="flex items-center justify-between mb-4">
      <span class="text-xl font-semibold text-gray-900">{{ $t('globals.terms.note', 2) }}</span>
      <Button
        variant="outline"
        size="sm"
        @click="startAddingNote"
        v-if="!isAddingNote && !isLoading && notes.length !== 0"
        class="transition-all hover:bg-primary/10 hover:border-primary/30"
      >
        <PlusIcon class="mr-2" size="18" />
        {{ $t('globals.messages.new', { name: $t('globals.terms.note') }) }}
      </Button>
    </div>

    <div class="h-52" v-if="isLoading">
      <Spinner />
    </div>

    <!-- Note input -->
    <div v-if="isAddingNote">
      <form @submit.prevent="addOrUpdateNote" @keydown.ctrl.enter="addOrUpdateNote">
        <div class="space-y-4">
          <div class="box p-2 h-52 min-h-52">
            <Editor
              v-model:htmlContent="newNote"
              @update:htmlContent="(value) => (newNote = value)"
              :placeholder="t('editor.placeholder')"
            />
          </div>
          <div class="flex justify-end space-x-3 pt-2">
            <Button
              variant="outline"
              @click="cancelAddNote"
              class="transition-all hover:bg-gray-100"
            >
              Cancel
            </Button>
            <Button type="submit" :disabled="!newNote.trim()">
              {{
                editingNoteId
                  ? $t('globals.buttons.update') + ' ' + $t('globals.terms.note').toLowerCase()
                  : $t('globals.buttons.save') + ' ' + $t('globals.terms.note').toLowerCase()
              }}
            </Button>
          </div>
        </div>
      </form>
    </div>

    <!-- Notes card list -->
    <div class="space-y-4">
      <Card
        v-for="note in notes"
        :key="note.id"
        class="overflow-hidden border-gray-2 00 hover:border-gray-300 transition-all duration-200 box hover:shadow"
      >
        <!-- Header -->
        <CardHeader class="bg-gray-50/50 border-b p-2">
          <div class="flex items-center justify-between">
            <div class="flex items-center space-x-3">
              <Avatar class="border border-gray-200 shadow-sm">
                <AvatarImage :src="note.avatar_url" />
                <AvatarFallback>
                  {{ getInitials(note.first_name, note.last_name) }}
                </AvatarFallback>
              </Avatar>
              <div>
                <p class="text-sm font-medium text-gray-900">{{ note.first_name }}</p>
                <p class="text-xs text-muted-foreground flex items-center">
                  <ClockIcon class="h-3 w-3 mr-1 inline-block opacity-70" />
                  {{ formatDate(note.created_at) }}
                </p>
              </div>
            </div>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon" class="h-8 w-8 rounded-full">
                  <MoreVerticalIcon class="h-4 w-4" />
                  <span class="sr-only">Open menu</span>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" class="w-[180px]">
                <DropdownMenuItem @click="editNote(note)" class="cursor-pointer">
                  <PencilIcon class="mr-2" size="15" />
                  {{ $t('globals.buttons.edit', { name: $t('globals.terms.note').toLowerCase() }) }}
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  @click="deleteNote(note.id)"
                  class="text-destructive cursor-pointer"
                >
                  <TrashIcon class="mr-2" size="15" />
                  {{
                    $t('globals.buttons.delete', { name: $t('globals.terms.note').toLowerCase() })
                  }}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </CardHeader>

        <!-- Note content -->
        <CardContent class="pt-4 pb-5 text-gray-700">
          <p class="whitespace-pre-wrap text-sm leading-relaxed" v-dompurify-html="note.note"></p>
        </CardContent>
      </Card>
    </div>

    <!-- No notes message -->
    <div
      v-if="notes.length === 0 && !isAddingNote && !isLoading"
      class="box border-dashed p-10 text-center bg-gray-50/50 mt-6"
    >
      <div class="flex flex-col items-center">
        <div class="rounded-full bg-gray-100 p-4 mb-2">
          <MessageSquareIcon class="text-gray-400" size="25" />
        </div>
        <h3 class="mt-2 text-base font-medium text-gray-900">{{ $t('contact.noNotes') }}</h3>
        <p class="mt-1 text-sm text-muted-foreground max-w-sm mx-auto">
          {{ $t('contact.notes.help') }}
        </p>
        <Button variant="outline" class="mt-3 border-gray-300" @click="startAddingNote">
          <PlusIcon class="mr-2" size="15" />
          {{ $t('globals.messages.add', { name: $t('globals.terms.note').toLowerCase() }) }}
        </Button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { format } from 'date-fns'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardContent } from '@/components/ui/card'
import { Avatar, AvatarImage, AvatarFallback } from '@/components/ui/avatar'
import { Spinner } from '@/components/ui/spinner'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator
} from '@/components/ui/dropdown-menu'
import {
  PlusIcon,
  MoreVerticalIcon,
  PencilIcon,
  TrashIcon,
  ClockIcon,
  MessageSquareIcon
} from 'lucide-vue-next'
import Editor from '@/features/conversation/ConversationTextEditor.vue'
import { useI18n } from 'vue-i18n'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@/utils/http'
import { getInitials } from '@/utils/strings'
import api from '@/api'

const props = defineProps({ contactId: Number })
const { t } = useI18n()
const emitter = useEmitter()

const notes = ref([])
const isAddingNote = ref(false)
const newNote = ref('')
const editingNoteId = ref(null)
const isLoading = ref(false)

const fetchNotes = async () => {
  try {
    isLoading.value = true
    const { data } = await api.getContactNotes(props.contactId)
    notes.value = data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

onMounted(fetchNotes)

const formatDate = (date) => format(new Date(date), 'PPP p')

const startAddingNote = () => {
  isAddingNote.value = true
}

const cancelAddNote = () => {
  isAddingNote.value = false
  newNote.value = ''
  editingNoteId.value = null
}

const addOrUpdateNote = async () => {
  try {
    if (editingNoteId.value) {
      await api.updateContactNote(props.contactId, editingNoteId.value, { note: newNote.value })
    } else {
      await api.createContactNote(props.contactId, { note: newNote.value })
    }
    await fetchNotes()
    cancelAddNote()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const editNote = (note) => {
  editingNoteId.value = note.id
  newNote.value = note.note
  isAddingNote.value = true
}

const deleteNote = async (noteId) => {
  await api.deleteContactNote(props.contactId, noteId)
  await fetchNotes()
}
</script>
