<template>
  <form @submit.prevent="onSubmit" class="w-2/3 space-y-6">
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
    <Button type="submit" size="sm">{{ submitLabel }}</Button>
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
      { name: 'conversations:read', label: 'View conversation' },
      { name: 'conversations:read_all', label: 'View all conversations' },
      { name: 'conversations:read_team', label: 'View team conversations' },
      { name: 'conversations:read_assigned', label: 'View assigned conversations' },
      { name: 'conversations:update_user_assignee', label: 'Edit assigned user' },
      { name: 'conversations:update_team_assignee', label: 'Edit assigned team' },
      { name: 'conversations:update_priority', label: 'Edit priority' },
      { name: 'conversations:update_status', label: 'Edit status' },
      { name: 'conversations:update_tags', label: 'Edit tags' },
      { name: 'messages:read', label: 'View conversation messages' },
      { name: 'messages:write', label: 'Reply to conversation' }
    ]
  },
  {
    name: 'Admin',
    permissions: [
      { name: 'settings_general:write', label: 'Update general settings' },
      { name: 'settings_notifications:write', label: 'Update notification settings' },
      { name: 'settings_notifications:read', label: 'View notification settings' },

      { name: 'oidc:read', label: 'View OpenID connect configurations' },
      { name: 'oidc:write', label: 'Update OpenID connect providers' },
      { name: 'oidc:delete', label: 'Delete OpenID connect providers' },

      { name: 'status:read', label: 'View statuses available for conversations' },
      { name: 'status:write', label: 'Update statuses available for conversations' },
      { name: 'status:delete', label: 'Delete statuses available for conversations' },

      { name: 'tags:write', label: 'Create and update tags' },
      { name: 'tags:delete', label: 'Delete tags' },

      { name: 'canned_responses:write', label: 'Create and update canned responses' },
      { name: 'canned_responses:delete', label: 'Delete canned responses' },

      { name: 'dashboard_global:read', label: 'Access global dashboard' },

      { name: 'users:read', label: 'View users' },
      { name: 'users:write', label: 'Create and update users' },

      { name: 'teams:read', label: 'View teams' },
      { name: 'teams:write', label: 'Create and update teams' },

      { name: 'automations:read', label: 'View automation rules' },
      { name: 'automations:write', label: 'Create and update automation rules' },
      { name: 'automations:delete', label: 'Delete automation rules' },

      { name: 'inboxes:read', label: 'View inboxes' },
      { name: 'inboxes:write', label: 'Create and update inboxes' },
      { name: 'inboxes:delete', label: 'Delete inboxes' },

      { name: 'roles:read', label: 'View roles' },
      { name: 'roles:write', label: 'Create and update roles' },
      { name: 'roles:delete', label: 'Delete roles' },

      { name: 'templates:read', label: 'View templates' },
      { name: 'templates:write', label: 'Create and update templates' }
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
    if (newValues) {
      form.setValues(newValues)
      selectedPermissions.value = newValues.permissions || []
    }
  },
  { deep: true }
)
</script>
