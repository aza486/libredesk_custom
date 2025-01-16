<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" class="w-8 h-8 p-0">
        <span class="sr-only">Open menu</span>
        <MoreHorizontal class="w-4 h-4" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent>
      <DropdownMenuItem @click="editMacro">Edit</DropdownMenuItem>
      <DropdownMenuItem @click="deleteMacro">Delete</DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>

<script setup>
import { MoreHorizontal } from 'lucide-vue-next'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import { Button } from '@/components/ui/button'
import '@vueup/vue-quill/dist/vue-quill.snow.css'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useRouter } from 'vue-router'
import api from '@/api/index.js'

const router = useRouter()
const emit = useEmitter()
const props = defineProps({
  macro: {
    type: Object,
    required: true
  }
})

const deleteMacro = async () => {
  await api.deleteMacro(props.macro.id)
  emit.emit(EMITTER_EVENTS.REFRESH_LIST, {
    model: 'macros'
  })
}

const editMacro = () => {
  router.push({ path: `/admin/conversations/macros/${props.macro.id}/edit` })
}
</script>
