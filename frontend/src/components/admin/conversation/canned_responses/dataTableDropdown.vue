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
    <DialogContent class="sm:max-w-[425px]">
      <DialogHeader>
        <DialogTitle>Edit canned response</DialogTitle>
        <DialogDescription>Click save when you're done. </DialogDescription>
      </DialogHeader>
      <form @submit.prevent="onSubmit">
        <FormField v-slot="{ componentField }" name="title">
          <FormItem>
            <FormLabel>Title</FormLabel>
            <FormControl>
              <Input type="text" placeholder="" v-bind="componentField" />
            </FormControl>
            <FormDescription></FormDescription>
            <FormMessage />
          </FormItem>
        </FormField>
        <FormField v-slot="{ componentField }" name="content">
          <FormItem>
            <FormLabel>Content</FormLabel>
            <FormControl>
              <Textarea v-bind="componentField" />
            </FormControl>
            <FormDescription></FormDescription>
            <FormMessage />
          </FormItem>
        </FormField>
        <DialogFooter>
          <Button type="submit" size="sm"> Save changes </Button>
        </DialogFooter>
      </form>
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
import { formSchema } from './formSchema.js'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@/components/ui/form'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { useEmitter } from '@/composables/useEmitter'
import api from '@/api/index.js'

const dialogOpen = ref(false)
const emit = useEmitter()

const props = defineProps({
  canned_response: {
    type: Object,
    required: true,
    default: () => ({
      id: '',
      name: ''
    })
  }
})

const form = useForm({
  validationSchema: toTypedSchema(formSchema)
})

const onSubmit = form.handleSubmit(async (values) => {
  await api.updateCannedResponse(props.canned_response.id, values)
  dialogOpen.value = false
  emitRefreshCannedResponseList()
})

const deleteCannedResponse = async () => {
  await api.deleteCannedResponse(props.canned_response.id)
  dialogOpen.value = false
  emitRefreshCannedResponseList()
}

const emitRefreshCannedResponseList = () => {
  emit.emit('refresh-list', {
    name: 'canned_responses'
  })
}

// Watch for changes in initialValues and update the form.
watch(
  () => props.canned_response,
  (newValues) => {
    form.setValues(newValues)
  },
  { immediate: true, deep: true }
)
</script>
