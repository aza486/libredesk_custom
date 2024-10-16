<template>
    <div class="flex justify-between mb-5">
      <PageHeader title="Tags" description="Manage conversation tags" />
      <div class="flex justify-end mb-4">
        <Dialog v-model:open="dialogOpen">
          <DialogTrigger as-child>
            <Button size="sm">New Tag</Button>
          </DialogTrigger>
          <DialogContent class="sm:max-w-[425px]">
            <DialogHeader>
              <DialogTitle>Add a Tag</DialogTitle>
              <DialogDescription> Set tag name. Click save when you're done. </DialogDescription>
            </DialogHeader>
            <form @submit.prevent="onSubmit">
              <FormField v-slot="{ field }" name="name">
                <FormItem>
                  <FormLabel>Name</FormLabel>
                  <FormControl>
                    <Input type="text" placeholder="billing, tech" v-bind="field" />
                  </FormControl>
                  <FormDescription></FormDescription>
                  <FormMessage />
                </FormItem>
              </FormField>
              <DialogFooter class="mt-7">
                <Button type="submit" size="sm">Save Changes</Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </div>
    </div>
  
    <Spinner v-if="isLoading"></Spinner>
    <div v-else>
      <DataTable :columns="columns" :data="tags" />
    </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import DataTable from '@/components/admin/DataTable.vue'
import { Spinner } from '@/components/ui/spinner'
import { columns } from '@/components/admin/conversation/tags/dataTableColumns.js'
import { Button } from '@/components/ui/button'
import PageHeader from '@/components/admin/common/PageHeader.vue'
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
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import api from '@/api'

const isLoading = ref(false)
const tags = ref([])
const emit = useEmitter()
const dialogOpen = ref(false)

onMounted(() => {
  getTags()
  emit.on(EMITTER_EVENTS.REFRESH_LIST, (data) => {
    if (data?.model === 'tags') getTags()
  })
})

const form = useForm({
  validationSchema: toTypedSchema(formSchema)
})

const getTags = async () => {
  isLoading.value = true
  const resp = await api.getTags()
  tags.value = resp.data.data
  isLoading.value = false
}

const onSubmit = form.handleSubmit(async (values) => {
  try {
    await api.createTag(values)
    dialogOpen.value = false
    getTags()
  } catch (error) {
    console.error('Failed to create tag:', error)
  }
})
</script>
