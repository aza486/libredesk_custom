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
          <Input type="text" placeholder="This role is for all support agents" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <p class="text-base">Set permissions for this role</p>

    <div v-for="entity in permissions" :key="entity.name" class="border box p-4 rounded-lg shadow-sm">
      <p class="text-lg mb-5">{{ entity.name }}</p>
      <div class="space-y-4">
        <FormField v-for="permission in entity.permissions" :key="permission.name" type="checkbox"
          :name="permission.name">
          <FormItem class="flex flex-col gap-y-5 space-y-0 rounded-lg">
            <div class="flex space-x-3">
              <FormControl>
                <Checkbox :checked="selectedPermissions.includes(permission.name)"
                  @update:checked="(newValue) => handleChange(newValue, permission.name)" />
                <FormLabel>{{ permission.label }}</FormLabel>
              </FormControl>
            </div>
          </FormItem>
        </FormField>
      </div>
    </div>
    <Button type="submit" size="sm" :isLoading="isLoading">{{ submitLabel }}</Button>
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
    required: false,
  }
})
const permissions = ref([
  {
    name: 'Conversation',
    permissions: [
      { name: 'conversations:read_all', label: 'View All' },
      { name: 'conversations:read_unassigned', label: 'View Unassigned' },
      { name: 'conversations:read_assigned', label: 'View Assigned' },
      { name: 'conversations:read', label: 'View' },
      { name: 'conversations:update_user_assignee', label: 'Edit User' },
      { name: 'conversations:update_team_assignee', label: 'Edit Team' },
      { name: 'conversations:update_priority', label: 'Edit Priority' },
      { name: 'conversations:update_status', label: 'Edit Status' },
      { name: 'conversations:update_tags', label: 'Edit Tags' },
      { name: 'messages:read', label: 'View Messages' },
      { name: 'messages:write', label: 'Reply' }
    ]
  },
  {
    name: 'Conversation status',
    permissions: [
      { name: 'status:read', label: 'View' },
      { name: 'status:write', label: 'Update' },
      { name: 'status:delete', label: 'Delete' }
    ]
  },
  {
    name: 'Admin',
    permissions: [
      { name: 'admin:read', label: 'Access' }
    ]
  },
  {
    name: 'Settings',
    permissions: [
      { name: 'settings_general:write', label: 'Update' },
      { name: 'settings_notifications:write', label: 'Update' },
      { name: 'settings_notifications:read', label: 'View' }
    ]
  },
  {
    name: 'OpenID Connect SSO',
    permissions: [
      { name: 'oidc:read', label: 'View' },
      { name: 'oidc:write', label: 'Update' },
      { name: 'oidc:delete', label: 'Delete' }
    ]
  },
  {
    name: 'Tags',
    permissions: [
      { name: 'tags:write', label: 'Create/Update' },
      { name: 'tags:delete', label: 'Delete' }
    ]
  },
  {
    name: 'Canned Responses',
    permissions: [
      { name: 'canned_responses:write', label: 'Create/Update' },
      { name: 'canned_responses:delete', label: 'Delete' }
    ]
  },
  {
    name: 'Dashboard',
    permissions: [
      { name: 'dashboard_global:read', label: 'Access' }
    ]
  },
  {
    name: 'Users',
    permissions: [
      { name: 'users:read', label: 'View' },
      { name: 'users:write', label: 'Create/Update' }
    ]
  },
  {
    name: 'Teams',
    permissions: [
      { name: 'teams:read', label: 'View' },
      { name: 'teams:write', label: 'Create/Update' }
    ]
  },
  {
    name: 'Automations',
    permissions: [
      { name: 'automations:read', label: 'View' },
      { name: 'automations:write', label: 'Create/Update' },
      { name: 'automations:delete', label: 'Delete' }
    ]
  },
  {
    name: 'Inboxes',
    permissions: [
      { name: 'inboxes:read', label: 'View' },
      { name: 'inboxes:write', label: 'Create/Update' },
      { name: 'inboxes:delete', label: 'Delete' }
    ]
  },
  {
    name: 'Roles',
    permissions: [
      { name: 'roles:read', label: 'View' },
      { name: 'roles:write', label: 'Create/Update' },
      { name: 'roles:delete', label: 'Delete' }
    ]
  },
  {
    name: 'Templates',
    permissions: [
      { name: 'templates:read', label: 'View' },
      { name: 'templates:write', label: 'Create/Update' },
      { name: 'templates:delete', label: 'Delete' }
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
