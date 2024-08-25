<script setup>
import { MoreHorizontal } from 'lucide-vue-next'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import { Button } from '@/components/ui/button'

const props = defineProps({
  inbox: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['editInbox', 'deleteInbox', 'toggleInbox'])

function editInbox(id) {
  emit('editInbox', id)
}

function deleteInbox(id) {
  emit('deleteInbox', id)
}

function toggleInbox(id) {
  emit('toggleInbox', id)
}
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" class="w-8 h-8 p-0">
        <span class="sr-only">Open menu</span>
        <MoreHorizontal class="w-4 h-4" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent>
      <DropdownMenuItem @click="editInbox(props.inbox.id)"> Edit </DropdownMenuItem>
      <DropdownMenuItem @click="deleteInbox(props.inbox.id)"> Delete </DropdownMenuItem>
      <DropdownMenuItem @click="toggleInbox(props.inbox.id)" v-if="props.inbox.disabled">
        Enable
      </DropdownMenuItem>
      <DropdownMenuItem @click="toggleInbox(props.inbox.id)" v-else> Disable </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
