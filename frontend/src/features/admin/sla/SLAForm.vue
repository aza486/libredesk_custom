<template>
  <form @submit="onSubmit" class="space-y-8">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>Name</FormLabel>
        <FormControl>
          <Input type="text" placeholder="SLA Name" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="description">
      <FormItem>
        <FormLabel>Description</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Describe the SLA" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="first_response_time">
      <FormItem>
        <FormLabel>First response time</FormLabel>
        <FormControl>
          <Input type="text" placeholder="6h" v-bind="componentField" />
        </FormControl>
        <FormDescription>
          Duration in hours or minutes to respond to a conversation.
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="resolution_time">
      <FormItem>
        <FormLabel>Resolution time</FormLabel>
        <FormControl>
          <Input type="text" placeholder="24h" v-bind="componentField" />
        </FormControl>
        <FormDescription> Duration in hours or minutes to resolve a conversation. </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Notifications Section -->
    <div class="space-y-6">
      <div class="flex items-center justify-between pb-3 border-b">
        <div class="space-y-1">
          <h3 class="text-lg font-semibold text-foreground">Alert Configuration</h3>
          <p class="text-sm text-muted-foreground">Set up notification triggers and recipients</p>
        </div>
        <div class="flex gap-2">
          <Button type="button" variant="outline" size="sm" @click="addNotification('breach')">
            <Plus class="w-4 h-4 mr-2" />
            Add Breach
          </Button>
          <Button type="button" variant="outline" size="sm" @click="addNotification('warning')">
            <Plus class="w-4 h-4 mr-2" />
            Add Warning
          </Button>
        </div>
      </div>

      <!-- Notifications List -->
      <div v-if="form.values.notifications?.length > 0" class="space-y-3">
        <div
          v-for="(notification, index) in form.values.notifications"
          :key="index"
          class="group relative p-5 box bg-background transition-all hover:border-foreground/20"
        >
          <FormField :name="`notifications.${index}.type`" v-slot="{ componentField }">
            <Input v-bind="componentField" type="hidden" />
          </FormField>

          <!-- Card Header -->
          <div class="flex items-center justify-between mb-5">
            <div class="flex items-center gap-3">
              <span
                class="flex items-center justify-center w-8 h-8 rounded-lg"
                :class="{
                  'bg-red-100/80 text-red-600': notification.type === 'breach',
                  'bg-amber-100/80 text-amber-600': notification.type === 'warning'
                }"
              >
                <CircleAlert size="18" v-if="notification.type === 'warning'" />
                <Timer size="18" v-else />
              </span>
              <div>
                <div class="font-medium text-foreground">
                  {{ notification.type === 'warning' ? 'Warning' : 'Breach' }} Notification
                </div>
                <p class="text-xs text-muted-foreground">
                  {{ notification.type === 'warning' ? 'Pre-breach alert' : 'Post-breach action' }}
                </p>
              </div>
            </div>
            <Button
              variant="ghost"
              size="sm"
              @click.prevent="removeNotification(index)"
              class="opacity-70 hover:opacity-100 text-muted-foreground hover:text-foreground"
            >
              <X class="w-4 h-4" />
            </Button>
          </div>

          <!-- Configuration Fields -->
          <div class="grid gap-5 md:grid-cols-2">
            <!-- Timing Section -->
            <div class="space-y-3">
              <div class="space-y-6">
                <FormField
                  :name="`notifications.${index}.time_delay_type`"
                  v-slot="{ componentField }"
                  v-if="notification.type === 'breach'"
                >
                  <FormItem>
                    <FormLabel class="flex items-center gap-1.5 text-sm font-medium">
                      <Clock class="w-4 h-4 text-muted-foreground" />
                      Trigger Timing
                    </FormLabel>
                    <FormControl>
                      <Select v-bind="componentField" class="hover:border-foreground/30">
                        <SelectTrigger class="w-full">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectGroup>
                            <SelectItem value="immediately" class="focus:bg-accent">
                              Immediately on breach
                            </SelectItem>
                            <SelectItem value="after" class="focus:bg-accent">
                              After specific duration
                            </SelectItem>
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                    </FormControl>
                  </FormItem>
                </FormField>

                <FormField :name="`notifications.${index}.time_delay`" v-slot="{ componentField }">
                  <FormItem v-if="shouldShowTimeDelay(index)">
                    <FormLabel class="flex items-center gap-1.5 text-sm font-medium">
                      <Hourglass class="w-4 h-4 text-muted-foreground" />
                      {{ notification.type === 'warning' ? 'Advance Warning' : 'Follow-up Delay' }}
                    </FormLabel>
                    <FormControl>
                      <Select v-bind="componentField" class="hover:border-foreground/30">
                        <SelectTrigger class="w-full">
                          <SelectValue placeholder="Select duration..." />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectGroup>
                            <SelectItem
                              v-for="duration in delayDurations"
                              :key="duration"
                              :value="duration"
                              class="focus:bg-accent"
                            >
                              {{ duration }}
                            </SelectItem>
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                </FormField>
              </div>
            </div>

            <!-- Recipients Section -->
            <div class="space-y-3">
              <FormField
                :name="`notifications.${index}.recipients`"
                v-slot="{ componentField, handleChange }"
              >
                <FormItem>
                  <FormLabel class="flex items-center gap-1.5 text-sm font-medium">
                    <Users class="w-4 h-4 text-muted-foreground" />
                    Notification Recipients
                  </FormLabel>
                  <FormControl>
                    <SelectTag
                      :items="
                        usersStore.options.concat({
                          label: 'Assigned user',
                          value: 'assigned_user'
                        })
                      "
                      placeholder="Start typing to search..."
                      v-model="componentField.modelValue"
                      @update:modelValue="handleChange"
                      class="w-full hover:border-foreground/30"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              </FormField>
            </div>
          </div>
        </div>
      </div>

      <!-- Empty State -->
      <div
        v-else
        class="flex flex-col items-center justify-center p-8 space-y-3 rounded-xl bg-muted/30 border border-dashed"
      >
        <Bell class="w-8 h-8 text-muted-foreground" />
        <p class="text-sm text-muted-foreground">No active notifications configured</p>
      </div>
    </div>

    <Button type="submit" :disabled="isLoading" :isLoading="isLoading" class="mt-6">
      {{ submitLabel }}
    </Button>
  </form>
</template>

<script setup>
import { watch } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { formSchema } from './formSchema'
import { Button } from '@/components/ui/button'
import { X, Plus, Timer, CircleAlert, Users, Clock, Hourglass, Bell } from 'lucide-vue-next'
import { useUsersStore } from '@/stores/users'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@/components/ui/form'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { SelectTag } from '@/components/ui/select'
import { Input } from '@/components/ui/input'

const props = defineProps({
  initialValues: {
    type: Object,
    default: () => ({})
  },
  submitForm: {
    type: Function,
    required: true
  },
  submitLabel: {
    type: String,
    default: 'Save'
  },
  isLoading: {
    type: Boolean,
    default: false
  }
})

const usersStore = useUsersStore()

const delayDurations = [
  '5m',
  '10m',
  '15m',
  '30m',
  '45m',
  '1h',
  '2h',
  '3h',
  '4h',
  '5h',
  '6h',
  '7h',
  '8h',
  '9h',
  '10h',
  '11h',
  '12h'
]

const form = useForm({
  validationSchema: toTypedSchema(formSchema),
  initialValues: {
    name: '',
    description: '',
    first_response_time: '',
    resolution_time: '',
    notifications: []
  }
})

const shouldShowTimeDelay = (index) => {
  const notification = form.values.notifications?.[index]
  if (!notification) return false
  return notification.type === 'warning' || notification.time_delay_type === 'after'
}

const addNotification = (type) => {
  const notifications = [...form.values.notifications || []]
  notifications.push({
    type: type,
    time_delay_type: type === 'warning' ? 'before' : 'immediately',
    time_delay: type === 'warning' ? '10m' : '',
    recipients: []
  })
  form.setFieldValue('notifications', notifications)
}

const removeNotification = (index) => {
  const notifications = [...form.values.notifications]
  notifications.splice(index, 1)
  console.log("Notifications", notifications)
  form.setFieldValue('notifications', notifications)
}

watch(
  () => props.initialValues,
  (newValues) => {
    if (!newValues || Object.keys(newValues).length === 0) {
      form.resetForm()
      return
    }

    const transformedNotifications = (newValues.notifications || []).map((notification) => ({
      ...notification,
      time_delay_type:
        notification.type === 'warning'
          ? 'before'
          : notification.time_delay
            ? 'after'
            : 'immediately'
    }))

    form.setValues({
      ...newValues,
      notifications: transformedNotifications
    })
  },
  { immediate: true, deep: true }
)

const onSubmit = form.handleSubmit((values) => {
  const payload = {
    ...values,
    notifications: values.notifications.map((notification) => ({
      ...notification,
      time_delay: notification.time_delay_type === 'immediately' ? '' : notification.time_delay
    }))
  }
  props.submitForm(payload)
})

// watch(
//   () => form.errors,
//   (errors) => {
//     if (Object.keys(errors).length > 0) {
//       console.log('Form has errors', errors)
//     }
//   },
//   { deep: true }
// )
</script>
