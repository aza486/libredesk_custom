<template>
  <Spinner v-if="formLoading"></Spinner>
  <form @submit="onSubmit" class="space-y-6 w-full" :class="{ 'opacity-50': formLoading }">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ t('form.field.name') }} </FormLabel>
        <FormControl>
          <Input type="text" placeholder="" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="message_content">
      <FormItem>
        <FormLabel>{{ t('admin.macro.messageContent') }}</FormLabel>
        <FormControl>
          <div class="box p-2 h-96 min-h-96">
            <Editor
              v-model:htmlContent="componentField.modelValue"
              @update:htmlContent="(value) => componentField.onChange(value)"
              :placeholder="t('editor.placeholder')"
            />
          </div>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField
      v-slot="{ componentField }"
      name="actions"
      :validate-on-blur="false"
      :validate-on-change="false"
    >
      <FormItem>
        <FormLabel>
          {{ t('globals.terms.action', 2) }} ({{ t('globals.terms.optional', 1).toLowerCase() }})
        </FormLabel>
        <FormControl>
          <ActionBuilder
            v-model:actions="componentField.modelValue"
            :config="actionConfig"
            @update:actions="(value) => componentField.onChange(value)"
          />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField, handleChange }" name="visible_when">
      <FormItem>
        <FormLabel>{{ t('globals.messages.visibleWhen') }}</FormLabel>
        <FormControl>
          <SelectTag
            :items="[
              { label: t('globals.messages.replying'), value: 'replying' },
              {
                label: t('globals.messages.starting', {
                  name: t('globals.terms.conversation').toLowerCase()
                }),
                value: 'starting_conversation'
              },
              {
                label: t('globals.messages.adding', {
                  name: t('globals.terms.privateNote', 2).toLowerCase()
                }),
                value: 'adding_private_note'
              }
            ]"
            v-model="componentField.modelValue"
            @update:modelValue="handleChange"
          />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField
      v-slot="{ componentField }"
      name="visibility"
      :validate-on-blur="false"
      :validate-on-change="false"
      :validate-on-input="false"
      :validate-on-mount="false"
      :validate-on-model-update="false"
    >
      <FormItem>
        <FormLabel>{{ t('globals.terms.visibility') }}</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="all">{{
                  t('globals.messages.all', {
                    name: t('globals.terms.user', 2).toLowerCase()
                  })
                }}</SelectItem>
                <SelectItem value="team">{{ t('globals.terms.team') }}</SelectItem>
                <SelectItem value="user">{{ t('globals.terms.user') }}</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-if="form.values.visibility === 'team'" v-slot="{ componentField }" name="team_id">
      <FormItem>
        <FormLabel>{{ t('globals.terms.team') }}</FormLabel>
        <FormControl>
          <SelectComboBox
            v-bind="componentField"
            :items="tStore.options"
            :placeholder="t('form.field.selectTeam')"
            type="team"
          />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-if="form.values.visibility === 'user'" v-slot="{ componentField }" name="user_id">
      <FormItem>
        <FormLabel>{{ t('globals.terms.user') }}</FormLabel>
        <FormControl>
          <SelectComboBox
            v-bind="componentField"
            :items="uStore.options"
            :placeholder="t('form.field.selectUser')"
            type="user"
          />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>
    <Button type="submit" :isLoading="isLoading">{{ submitLabel }}</Button>
  </form>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'
import { Input } from '@/components/ui/input'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import ActionBuilder from '@/features/admin/macros/ActionBuilder.vue'
import { useConversationFilters } from '@/composables/useConversationFilters'
import { useUsersStore } from '@/stores/users'
import { useTeamStore } from '@/stores/team'
import { getTextFromHTML } from '@/utils/strings.js'
import { createFormSchema } from './formSchema.js'
import SelectComboBox from '@/components/combobox/SelectCombobox.vue'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
  SelectTag
} from '@/components/ui/select'
import { useI18n } from 'vue-i18n'
import Editor from '@/features/conversation/ConversationTextEditor.vue'

const { macroActions } = useConversationFilters()
const { t } = useI18n()
const formLoading = ref(false)
const uStore = useUsersStore()
const tStore = useTeamStore()
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
    default: ''
  },
  isLoading: {
    type: Boolean,
    default: false
  }
})

const submitLabel = computed(() => {
  return (
    props.submitLabel ||
    (props.initialValues.id ? t('globals.buttons.update') : t('globals.buttons.create'))
  )
})
const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t)),
  initialValues: {
    visible_when: props.initialValues.visible_when || [
      'replying',
      'starting_conversation',
      'adding_private_note'
    ],
    visibility: props.initialValues.visibility || 'all'
  }
})

const actionConfig = ref({
  actions: macroActions,
  typePlaceholder: t('form.field.selectActionType'),
  valuePlaceholder: t('form.field.selectValue'),
  addButtonText: t('form.field.addNewAction')
})

const onSubmit = form.handleSubmit(async (values) => {
  // If the text of HTML is empty then set the HTML to empty string
  const textContent = getTextFromHTML(values.message_content)
  if (textContent.length === 0) {
    values.message_content = ''
  }
  props.submitForm(values)
})

watch(
  () => props.initialValues,
  (newValues) => {
    if (Object.keys(newValues).length === 0) return
    form.setValues(newValues)
  },
  { immediate: true }
)
</script>
