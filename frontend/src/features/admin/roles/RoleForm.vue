<template>
  <form @submit.prevent="onSubmit" class="space-y-6">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem v-auto-animate>
        <FormLabel>{{ $t('form.field.name') }}</FormLabel>
        <FormControl>
          <Input type="text" :placeholder="t('globals.entities.agent')" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>
    <FormField v-slot="{ componentField }" name="description">
      <FormItem>
        <FormLabel>{{ $t('form.field.description') }}</FormLabel>
        <FormControl>
          <Input
            type="text"
            :placeholder="t('admin.role.roleForAllSupportAgents')"
            v-bind="componentField"
          />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <p class="text-base">{{ $t('admin.role.setPermissionsForThisRole') }}</p>

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
import { watch, ref, computed } from 'vue'
import { Button } from '@/components/ui/button'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from './formSchema.js'
import { vAutoAnimate } from '@formkit/auto-animate/vue'
import { Checkbox } from '@/components/ui/checkbox'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { useI18n } from 'vue-i18n'

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
    default: () => ''
  },
  isLoading: {
    type: Boolean,
    required: false
  }
})

const { t } = useI18n()

const submitLabel = computed(() => {
  return props.submitLabel || t('globals.buttons.save')
})

const permissions = ref([
  {
    name: 'Conversation',
    permissions: [
      { name: 'conversations:read', label: t('admin.role.conversations.read') },
      { name: 'conversations:write', label: t('admin.role.conversations.write') },
      { name: 'conversations:read_assigned', label: t('admin.role.conversations.read_assigned') },
      { name: 'conversations:read_all', label: t('admin.role.conversations.read_all') },
      { name: 'conversations:read_unassigned', label: t('admin.role.conversations.read_unassigned') },
      { name: 'conversations:read_team_inbox', label: t('admin.role.conversations.read_team_inbox') },
      { name: 'conversations:update_user_assignee', label: t('admin.role.conversations.update_user_assignee') },
      { name: 'conversations:update_team_assignee', label: t('admin.role.conversations.update_team_assignee') },
      { name: 'conversations:update_priority', label: t('admin.role.conversations.update_priority') },
      { name: 'conversations:update_status', label: t('admin.role.conversations.update_status') },
      { name: 'conversations:update_tags', label: t('admin.role.conversations.update_tags') },
      { name: 'messages:read', label: t('admin.role.messages.read') },
      { name: 'messages:write', label: t('admin.role.messages.write') },
      { name: 'view:manage', label: t('admin.role.view.manage') }
    ]
  },
  {
    name: 'Admin settings',
    permissions: [
      { name: 'general_settings:manage', label: t('admin.role.general_settings.manage') },
      { name: 'notification_settings:manage', label: t('admin.role.notification_settings.manage') },
      { name: 'status:manage', label: t('admin.role.status.manage') },
      { name: 'oidc:manage', label: t('admin.role.oidc.manage') },
      { name: 'tags:manage', label: t('admin.role.tags.manage') },
      { name: 'macros:manage', label: t('admin.role.macros.manage') },
      { name: 'users:manage', label: t('admin.role.users.manage') },
      { name: 'teams:manage', label: t('admin.role.teams.manage') },
      { name: 'automations:manage', label: t('admin.role.automations.manage') },
      { name: 'inboxes:manage', label: t('admin.role.inboxes.manage') },
      { name: 'roles:manage', label: t('admin.role.roles.manage') },
      { name: 'templates:manage', label: t('admin.role.templates.manage') },
      { name: 'reports:manage', label: t('admin.role.reports.manage') },
      { name: 'business_hours:manage', label: t('admin.role.business_hours.manage') },
      { name: 'sla:manage', label: t('admin.role.sla.manage') },
      { name: 'ai:manage', label: t('admin.role.ai.manage') }
    ]
  }
])

const selectedPermissions = ref([])

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t)),
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
