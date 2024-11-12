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
import { Roles } from '@/constants/user'
import api from '@/api'

const emit = useEmitter()
const router = useRouter()
const props = defineProps({
  role: {
    type: Object,
    required: true,
    default: () => ({
      id: ''
    })
  }
})

function editRole (id) {
  router.push({ path: `/admin/teams/roles/${id}/edit` })
}

async function deleteRole (id) {
  try {
    await api.deleteRole(id)
    emitRefreshTeamList()
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const emitRefreshTeamList = () => {
  emit.emit(EMITTER_EVENTS.REFRESH_LIST, {
    model: 'team'
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
      <DropdownMenuItem @click="editRole(props.role.id)"> Edit </DropdownMenuItem>
      <DropdownMenuItem @click="deleteRole(props.role.id)" v-if="Roles.includes(props.role.name) === false">
        Delete </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
