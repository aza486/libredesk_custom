<template>
  <div>
    <div class="flex justify-between mb-5">
      <PageHeader title="Canned responses" description="Manage canned responses" />
      <div class="flex justify-end mb-4">
        <Dialog v-model:open="dialogOpen">
          <DialogTrigger as-child>
            <Button size="sm">New canned response</Button>
          </DialogTrigger>
          <DialogContent class="sm:max-w-[425px]">
            <DialogHeader>
              <DialogTitle>Add a canned response</DialogTitle>
              <DialogDescription> Set canned response name. Click save when you're done. </DialogDescription>
            </DialogHeader>
            <form @submit.prevent="onSubmit">
              <FormField v-slot="{ field }" name="title">
                <FormItem>
                  <FormLabel>Title</FormLabel>
                  <FormControl>
                    <Input type="text" placeholder="" v-bind="field" />
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
                <Button type="submit" size="sm">Save Changes</Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </div>
    </div>
    <div class="w-full">
      <DataTable :columns="columns" :data="cannedResponses" />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import DataTable from '@/components/admin/DataTable.vue'
import { columns } from './dataTableColumns.js'
import { Button } from '@/components/ui/button'
import PageHeader from '@/components/admin/common/PageHeader.vue'
import { Textarea } from '@/components/ui/textarea'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@/components/ui/dialog'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { formSchema } from './formSchema.js'
import { useEmitter } from '@/composables/useEmitter'
import api from '@/api'

const cannedResponses = ref([])
const emit = useEmitter()
const dialogOpen = ref(false)

onMounted(() => {
  getCannedResponses()
  emit.on('refresh-list', refreshList)
})

onUnmounted(() => {
  emit.off('refresh-list', refreshList)
})

const refreshList = (data) => {
  if (data?.name === 'canned_responses') getCannedResponses()
}

const form = useForm({
  validationSchema: toTypedSchema(formSchema)
})

const getCannedResponses = async () => {
  const resp = await api.getCannedResponses()
  cannedResponses.value = resp.data.data
}

const onSubmit = form.handleSubmit(async (values) => {
  try {
    await api.createCannedResponse(values)
    dialogOpen.value = false
    getCannedResponses()
  } catch (error) {
    console.error('Failed to create canned response:', error)
  }
})
</script>
