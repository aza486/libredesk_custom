<template>
  <form @submit.prevent="onSubmit" class="space-y-8">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem v-auto-animate>
        <FormLabel>{{ $t('form.field.name') }}</FormLabel>
        <FormControl>
          <Input type="text" :placeholder="t('globals.terms.agent')" v-bind="componentField" />
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

    <div>
      <div class="mb-5 text-lg">{{ $t('admin.role.setPermissionsForThisRole') }}</div>

      <div class="space-y-6">
        <div
          v-for="entity in permissions"
          :key="entity.name"
          class="rounded-lg border border-border bg-card"
        >
          <div class="border-b border-border bg-muted/30 px-5 py-3">
            <h4 class="font-medium text-card-foreground">{{ entity.name }}</h4>
          </div>

          <div class="p-5">
            <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              <FormField
                v-for="permission in entity.permissions"
                :key="permission.name"
                :name="permission.name"
              >
                <FormItem class="flex items-start space-x-3 space-y-0">
                  <FormControl>
                    <Checkbox
                      :checked="selectedPermissions.includes(permission.name)"
                      @update:checked="(newValue) => handleChange(newValue, permission.name)"
                    />
                  </FormControl>
                  <FormLabel class="font-normal text-sm">{{ permission.label }}</FormLabel>
                </FormItem>
              </FormField>
            </div>
          </div>
        </div>
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
    name: t('globals.terms.conversation'),
    permissions: [
      { name: 'conversations:read', label: t('admin.role.conversations.read') },
      { name: 'conversations:write', label: t('admin.role.conversations.write') },
      { name: 'conversations:read_assigned', label: t('admin.role.conversations.readAssigned') },
      { name: 'conversations:read_all', label: t('admin.role.conversations.readAll') },
      {
        name: 'conversations:read_unassigned',
        label: t('admin.role.conversations.readUnassigned')
      },
      { name: 'conversations:read_team_inbox', label: t('admin.role.conversations.readTeamInbox') },
      {
        name: 'conversations:update_user_assignee',
        label: t('admin.role.conversations.updateUserAssignee')
      },
      {
        name: 'conversations:update_team_assignee',
        label: t('admin.role.conversations.updateTeamAssignee')
      },
      {
        name: 'conversations:update_priority',
        label: t('admin.role.conversations.updatePriority')
      },
      { name: 'conversations:update_status', label: t('admin.role.conversations.updateStatus') },
      { name: 'conversations:update_tags', label: t('admin.role.conversations.updateTags') },
      { name: 'messages:read', label: t('admin.role.messages.read') },
      { name: 'messages:write', label: t('admin.role.messages.write') },
      { name: 'view:manage', label: t('admin.role.view.manage') }
    ]
  },
  {
    name: t('globals.terms.admin'),
    permissions: [
      { name: 'general_settings:manage', label: t('admin.role.generalSettings.manage') },
      { name: 'notification_settings:manage', label: t('admin.role.notificationSettings.manage') },
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
      { name: 'business_hours:manage', label: t('admin.role.businessHours.manage') },
      { name: 'sla:manage', label: t('admin.role.sla.manage') },
      { name: 'ai:manage', label: t('admin.role.ai.manage') },
      { name: 'custom_attributes:manage', label: t('admin.role.customAttributes.manage') }
    ]
  },
  {
    name: t('globals.terms.contact'),
    permissions: [{ name: 'contacts:manage', label: t('admin.role.contacts.manage') }]
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
