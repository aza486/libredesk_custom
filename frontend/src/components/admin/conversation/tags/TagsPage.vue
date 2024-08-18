<template>
  <div>
    <div class="flex justify-between mb-5">
      <PageHeader title="Tags" description="Manage conversation tags" />
      <div class="flex justify-end mb-4">
        <Dialog v-model:open="dialogOpen">
          <DialogTrigger as-child>
            <Button @click="newTag" size="sm">New Tag</Button>
          </DialogTrigger>
          <DialogContent class="sm:max-w-[425px]">
            <DialogHeader>
              <DialogTitle>Add a Tag</DialogTitle>
              <DialogDescription>
                Set tag name. Click save when you're done.
              </DialogDescription>
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
              <DialogFooter>
                <Button type="submit" size="sm">Save Changes</Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </div>
    </div>
    <div class="w-full">
      <DataTable :columns="columns" :data="tags" />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import DataTable from '@/components/admin/DataTable.vue'
import { columns } from '@/components/admin/conversation/tags/dataTableColumns.js'
import { Button } from '@/components/ui/button'
import PageHeader from '@/components/admin/common/PageHeader.vue'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { formSchema } from './formSchema.js'
import { useEmitter } from '@/composables/useEmitter'
import api from '@/api'

const tags = ref([])
const emit = useEmitter()
const dialogOpen = ref(false)

onMounted(() => {
  getTags()
  emit.on('refresh-list', (data) => {
    if (data?.name === "tags")
      getTags()
  })
})

const form = useForm({
  validationSchema: toTypedSchema(formSchema),
})

const getTags = async () => {
  const resp = await api.getTags()
  tags.value = resp.data.data
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
