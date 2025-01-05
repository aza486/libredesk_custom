<template>
  <Dialog v-model:open="dialogOpen">
    <DropdownMenu>
      <DropdownMenuTrigger as-child>
        <Button variant="ghost" class="w-8 h-8 p-0">
          <span class="sr-only">Open menu</span>
          <MoreHorizontal class="w-4 h-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        <DialogTrigger as-child>
          <DropdownMenuItem> Edit </DropdownMenuItem>
        </DialogTrigger>
        <DropdownMenuItem @click="deleteCannedResponse"> Delete </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
    <DialogContent class="sm:max-w-[625px]">
      <DialogHeader>
        <DialogTitle>Edit canned response</DialogTitle>
        <DialogDescription>Edit title and content, click save when you're done. </DialogDescription>
      </DialogHeader>
      <CannedResponsesForm @submit="onSubmit">
        <template #footer>
          <DialogFooter class="mt-7">
            <Button type="submit">Save Changes</Button>
          </DialogFooter>
        </template>
      </CannedResponsesForm>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { watch, ref } from 'vue'
import { MoreHorizontal } from 'lucide-vue-next'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import { Button } from '@/components/ui/button'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import CannedResponsesForm from './CannedResponsesForm.vue'
import '@vueup/vue-quill/dist/vue-quill.snow.css';
import { formSchema } from './formSchema.js'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@/components/ui/dialog'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import api from '@/api/index.js'

const dialogOpen = ref(false)
const emit = useEmitter()

const props = defineProps({
  cannedResponse: {
    type: Object,
    required: true,
  }
})

const form = useForm({
  validationSchema: toTypedSchema(formSchema)
})

const onSubmit = form.handleSubmit(async (values) => {
  await api.updateCannedResponse(props.cannedResponse.id, values)
  dialogOpen.value = false
  emitRefreshCannedResponseList()
})

const deleteCannedResponse = async () => {
  await api.deleteCannedResponse(props.cannedResponse.id)
  dialogOpen.value = false
  emitRefreshCannedResponseList()
}

const emitRefreshCannedResponseList = () => {
  emit.emit(EMITTER_EVENTS.REFRESH_LIST, {
    model: 'canned_responses'
  })
}

// Watch for changes in initialValues and update the form.
watch(
  () => props.cannedResponse,
  (newValues) => {
    form.setValues(newValues)
  },
  { immediate: true, deep: true }
)
</script>
