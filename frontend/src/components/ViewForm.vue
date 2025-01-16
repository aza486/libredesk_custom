<template>
  <Dialog :open="openDialog" @update:open="openDialog = false">
    <DialogContent>
      <DialogHeader class="space-y-1">
        <DialogTitle>{{ view?.id ? 'Edit' : 'Create' }} view</DialogTitle>
        <DialogDescription>
          Views let you create custom filters and save them.
        </DialogDescription>
      </DialogHeader>
      <form @submit.prevent="onSubmit">
        <div class="grid gap-4 py-4">
          <FormField v-slot="{ componentField }" name="name">
            <FormItem>
              <FormLabel>Name</FormLabel>
              <FormControl>
                <Input id="name" class="col-span-3" placeholder="Name" v-bind="componentField" />
              </FormControl>
              <FormDescription>Enter a unique name for your view.</FormDescription>
              <FormMessage />
            </FormItem>
          </FormField>
          <FormField v-slot="{ componentField }" name="inbox_type">
            <FormItem>
              <FormLabel>Inbox</FormLabel>
              <FormControl>
                <Select class="w-full" v-bind="componentField">
                  <SelectTrigger>
                    <SelectValue placeholder="Select inbox" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      <SelectItem
                        v-for="(value, key) in CONVERSATION_VIEWS_INBOXES"
                        :key="key"
                        :value="key"
                      >
                        {{ value }}
                      </SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </FormControl>
              <FormDescription>Select inbox to filter conversations from.</FormDescription>
              <FormMessage />
            </FormItem>
          </FormField>
          <FormField v-slot="{ componentField }" name="filters">
            <FormItem>
              <FormLabel>Filters</FormLabel>
              <FormControl>
                <Filter :fields="filterFields" :showButtons="false" v-bind="componentField" />
              </FormControl>
              <FormDescription>Add multiple filters to customize view.</FormDescription>
              <FormMessage />
            </FormItem>
          </FormField>
        </div>
        <DialogFooter>
          <Button type="submit" :disabled="isSubmitting" :isLoading="isSubmitting">
            {{ isSubmitting ? 'Saving...' : 'Save changes' }}
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useForm } from 'vee-validate'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@/components/ui/form'
import { CONVERSATION_VIEWS_INBOXES } from '@/constants/conversation'
import { Input } from '@/components/ui/input'
import Filter from '@/components/common/FilterBuilder.vue'
import { useConversationFilters } from '@/composables/useConversationFilters'
import { toTypedSchema } from '@vee-validate/zod'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import { z } from 'zod'
import api from '@/api'

const emitter = useEmitter()
const openDialog = defineModel('openDialog', { required: false, default: false })
const view = defineModel('view', { required: false, default: {} })
const isSubmitting = ref(false)

const { conversationsListFilters } = useConversationFilters()

const filterFields = computed(() =>
  Object.entries(conversationsListFilters.value).map(([field, value]) => ({
    model: 'conversations',
    label: value.label,
    field,
    type: value.type,
    operators: value.operators,
    options: value.options ?? []
  }))
)
const formSchema = toTypedSchema(
  z.object({
    id: z.number().optional(),
    name: z
      .string()
      .min(2, { message: 'Name must be at least 2 characters.' })
      .max(250, { message: 'Name cannot exceed 250 characters.' }),
    inbox_type: z.enum(Object.keys(CONVERSATION_VIEWS_INBOXES)),
    filters: z
      .array(
        z.object({
          model: z.string({ required_error: 'Filter required' }),
          field: z.string({ required_error: 'Filter required' }),
          operator: z.string({ required_error: 'Filter required' }),
          value: z.union([z.string(), z.number(), z.boolean()])
        })
      )
      .default([])
  })
)

const form = useForm({ validationSchema: formSchema })

const onSubmit = form.handleSubmit(async (values) => {
  if (isSubmitting.value) return

  isSubmitting.value = true
  try {
    if (values.id) {
      await api.updateView(values.id, values)
    } else {
      await api.createView(values)
    }
    emitter.emit(EMITTER_EVENTS.REFRESH_LIST, { model: 'view' })
    openDialog.value = false
    form.resetForm()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isSubmitting.value = false
  }
})

watch(
  () => view.value,
  (newVal) => {
    if (newVal && Object.keys(newVal).length) {
      form.setValues(newVal)
    }
  },
  { immediate: true }
)
</script>
