<script setup>
import { MoreHorizontal } from 'lucide-vue-next'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import { Button } from '@/components/ui/button'
import { useRouter } from 'vue-router'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import api from '@/api'

const emit = useEmitter()
const router = useRouter()

const props = defineProps({
  user: {
    type: Object,
    required: true,
    default: () => ({
      id: ''
    })
  }
})

function editUser (id) {
  router.push({ path: `/admin/teams/users/${id}/edit` })
}

async function deleteUser (id) {
  try {
    await api.deleteUser(id)
    emitRefreshUserList()
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const emitRefreshUserList = () => {
  emit.emit(EMITTER_EVENTS.REFRESH_LIST, {
    model: 'user'
  })
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
      <DropdownMenuItem @click="editUser(props.user.id)"> Edit </DropdownMenuItem>
      <DropdownMenuItem @click="deleteUser(props.user.id)"> Delete </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
