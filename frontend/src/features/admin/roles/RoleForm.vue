<template>
  <form @submit.prevent="onSubmit" class="space-y-6">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem v-auto-animate>
        <FormLabel>Name</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Agent" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>
    <FormField v-slot="{ componentField }" name="description">
      <FormItem>
        <FormLabel>Description</FormLabel>
        <FormControl>
          <Input
            type="text"
            placeholder="This role is for all support agents"
            v-bind="componentField"
          />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <p class="text-base">Set permissions for this role</p>

    <div v-for="entity in permissions" :key="entity.name" class="box p-4">
      <p class="text-lg mb-5">{{ entity.name }}</p>
      <div class="space-y-4">
        <FormField
          v-for="permission in entity.permissions"
          :key="permission.name"
          type="checkbox"
          :name="permission.name"
        >
          <FormItem class="flex flex-col gap-y-5 space-y-0 rounded-lg">
            <div class="flex space-x-3">
              <FormControl>
                <Checkbox
                  :checked="selectedPermissions.includes(permission.name)"
                  @update:checked="(newValue) => handleChange(newValue, permission.name)"
                />
                <FormLabel>{{ permission.label }}</FormLabel>
              </FormControl>
            </div>
          </FormItem>
        </FormField>
      </div>
    </div>
    <Button type="submit" :isLoading="isLoading">{{ submitLabel }}</Button>
  </form>
</template>

<script setup>
import { watch, ref } from 'vue'
import { Button } from '@/components/ui/button'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { formSchema } from './formSchema.js'
import { vAutoAnimate } from '@formkit/auto-animate/vue'
import { Checkbox } from '@/components/ui/checkbox'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'

const props = defineProps({
  initialValues: {
    type: Object,
    required: false
  },
  submitForm: {
    type: Function,
    required: true
  },
  submitLabel: {
    type: String,
    required: false,
    default: () => 'Submit'
  },
  isLoading: {
    type: Boolean,
    required: false
  }
})

const permissions = ref([
  {
    name: 'Conversation',
    permissions: [
      { name: 'conversations:read', label: 'View conversation' },
      { name: 'conversations:read_assigned', label: 'View conversations assigned to me' },
      { name: 'conversations:read_all', label: 'View all conversations' },
      { name: 'conversations:read_unassigned', label: 'View all unassigned conversations' },
      { name: 'conversations:read_team_inbox', label: 'View conversations in team inbox' },
      { name: 'conversations:update_user_assignee', label: 'Assign conversations to users' },
      { name: 'conversations:update_team_assignee', label: 'Assign conversations to teams' },
      { name: 'conversations:update_priority', label: 'Change conversation priority' },
      { name: 'conversations:update_status', label: 'Change conversation status' },
      { name: 'conversations:update_tags', label: 'Add or remove conversation tags' },
      { name: 'messages:read', label: 'View conversation messages' },
      { name: 'messages:write', label: 'Send messages in conversations' },
      { name: 'view:manage', label: 'Create and manage conversation views' }
    ]
  },
  {
    name: 'Admin settings',
    permissions: [
      { name: 'general_settings:manage', label: 'Manage General Settings' },
      { name: 'notification_settings:manage', label: 'Manage Notification Settings' },
      { name: 'status:manage', label: 'Manage Conversation Statuses' },
      { name: 'oidc:manage', label: 'Manage SSO Configuration' },
      { name: 'tags:manage', label: 'Manage Tags' },
      { name: 'macros:manage', label: 'Manage Macros' },
      { name: 'users:manage', label: 'Manage Users' },
      { name: 'teams:manage', label: 'Manage Teams' },
      { name: 'automations:manage', label: 'Manage Automations' },
      { name: 'inboxes:manage', label: 'Manage Inboxes' },
      { name: 'roles:manage', label: 'Manage Roles' },
      { name: 'templates:manage', label: 'Manage Templates' },
      { name: 'reports:manage', label: 'Manage Reports' },
      { name: 'business_hours:manage', label: 'Manage Business Hours' },
      { name: 'sla:manage', label: 'Manage SLA Policies' },
      { name: 'ai:manage', label: 'Manage AI Features' }
    ]
  }
])

const selectedPermissions = ref([])

const form = useForm({
  validationSchema: toTypedSchema(formSchema),
  initialValues: props.initialValues
})

const onSubmit = form.handleSubmit((values) => {
  values.permissions = selectedPermissions.value
  props.submitForm(values)
})

const handleChange = (value, perm) => {
  if (value) {
    selectedPermissions.value.push(perm)
  } else {
    const index = selectedPermissions.value.indexOf(perm)
    if (index > -1) {
      selectedPermissions.value.splice(index, 1)
    }
  }
}

// Watch for changes in initialValues and update the form.
watch(
  () => props.initialValues,
  (newValues) => {
    form.setValues(newValues)
    selectedPermissions.value = newValues.permissions || []
  },
  { deep: true, immediate: true }
)
</script>
