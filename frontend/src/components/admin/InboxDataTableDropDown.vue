<script setup>
import { MoreHorizontal } from 'lucide-vue-next'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import { Button } from '@/components/ui/button'
import { useRouter } from 'vue-router'
import api from '@/api'

const router = useRouter()

const props = defineProps({
  inbox: {
    type: Object,
    required: true,
    default: () => ({
      id: ''
    })
  }
})

function editInbox(id) {
  router.push({ path: `/admin/inboxes/${id}/edit` })
}

async function deleteInbox(id) {
  await api.deleteInbox(id)
  router.push({ path: '/admin/inboxes' })
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
    </DropdownMenuContent>
  </DropdownMenu>
</template>
