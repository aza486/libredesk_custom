<template>
  <Dialog :open="dialogOpen" @update:open="dialogOpen = false">
    <DialogContent class="max-w-5xl w-full h-[90vh] flex flex-col">
      <DialogHeader>
        <DialogTitle>
          {{
            $t('globals.messages.new', {
              name: $t('globals.terms.conversation')
            })
          }}
        </DialogTitle>
      </DialogHeader>
      <form @submit="createConversation" class="flex flex-col flex-1 overflow-hidden">
        <div class="flex-1 space-y-4 pr-1 overflow-y-auto pb-2">
          <FormField name="contact_email">
            <FormItem class="relative">
              <FormLabel>{{ $t('form.field.email') }}</FormLabel>
              <FormControl>
                <Input
                  type="email"
                  :placeholder="t('conversation.searchContact')"
                  v-model="emailQuery"
                  @input="handleSearchContacts"
                  autocomplete="off"
                />
              </FormControl>
              <FormMessage />

              <ul
                v-if="searchResults.length"
                class="border rounded p-2 max-h-60 overflow-y-auto absolute bg-white w-full z-50 shadow-lg"
              >
                <li
                  v-for="contact in searchResults"
                  :key="contact.email"
                  @click="selectContact(contact)"
                  class="cursor-pointer p-2 hover:bg-gray-100 rounded"
                >
                  {{ contact.first_name }} {{ contact.last_name }} ({{ contact.email }})
                </li>
              </ul>
            </FormItem>
          </FormField>

          <FormField v-slot="{ componentField }" name="first_name">
            <FormItem>
              <FormLabel>{{ $t('form.field.firstName') }}</FormLabel>
              <FormControl>
                <Input type="text" placeholder="" v-bind="componentField" required />
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>

          <FormField v-slot="{ componentField }" name="last_name">
            <FormItem>
              <FormLabel>{{ $t('form.field.lastName') }}</FormLabel>
              <FormControl>
                <Input type="text" placeholder="" v-bind="componentField" required />
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>

          <FormField v-slot="{ componentField }" name="subject">
            <FormItem>
              <FormLabel>{{ $t('form.field.subject') }}</FormLabel>
              <FormControl>
                <Input type="text" placeholder="" v-bind="componentField" required />
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>

          <FormField v-slot="{ componentField }" name="inbox_id">
            <FormItem>
              <FormLabel>{{ $t('form.field.inbox') }}</FormLabel>
              <FormControl>
                <Select v-bind="componentField">
                  <SelectTrigger>
                    <SelectValue :placeholder="t('form.field.selectInbox')" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      <SelectItem
                        v-for="option in inboxStore.options"
                        :key="option.value"
                        :value="option.value"
                      >
                        {{ option.label }}
                      </SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>

          <!-- Set assigned team -->
          <FormField v-slot="{ componentField }" name="team_id">
            <FormItem>
              <FormLabel>{{ $t('form.field.assignTeamOptional') }}</FormLabel>
              <FormControl>
                <ComboBox
                  v-bind="componentField"
                  :items="[{ value: 'none', label: 'None' }, ...teamStore.options]"
                  :placeholder="t('form.field.selectTeam')"
                >
                  <template #item="{ item }">
                    <div class="flex items-center gap-3 py-2">
                      <div class="w-7 h-7 flex items-center justify-center">
                        <span v-if="item.emoji">{{ item.emoji }}</span>
                        <div
                          v-else
                          class="text-primary bg-muted rounded-full w-7 h-7 flex items-center justify-center"
                        >
                          <Users size="14" />
                        </div>
                      </div>
                      <span class="text-sm">{{ item.label }}</span>
                    </div>
                  </template>

                  <template #selected="{ selected }">
                    <div class="flex items-center gap-3">
                      <div class="w-7 h-7 flex items-center justify-center" v-if="selected">
                        {{ selected?.emoji }}
                      </div>
                      <span class="text-sm">{{
                        selected?.label || t('form.field.selectTeam')
                      }}</span>
                    </div>
                  </template>
                </ComboBox>
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>

          <!-- Set assigned agent -->
          <FormField v-slot="{ componentField }" name="agent_id">
            <FormItem>
              <FormLabel>{{ $t('form.field.assignAgentOptional') }}</FormLabel>
              <FormControl>
                <ComboBox
                  v-bind="componentField"
                  :items="[{ value: 'none', label: 'None' }, ...uStore.options]"
                  :placeholder="t('form.field.selectAgent')"
                >
                  <template #item="{ item }">
                    <div class="flex items-center gap-3 py-2">
                      <Avatar class="w-8 h-8">
                        <AvatarImage
                          :src="item.value === 'none' ? '' : item.avatar_url || ''"
                          :alt="item.value === 'none' ? 'N' : item.label.slice(0, 2)"
                        />
                        <AvatarFallback>
                          {{ item.value === 'none' ? 'N' : item.label.slice(0, 2).toUpperCase() }}
                        </AvatarFallback>
                      </Avatar>
                      <span class="text-sm">{{ item.label }}</span>
                    </div>
                  </template>

                  <template #selected="{ selected }">
                    <div class="flex items-center gap-3">
                      <Avatar class="w-7 h-7" v-if="selected">
                        <AvatarImage
                          :src="
                            selected?.value === 'none'
                              ? ''
                              : selected?.avatar_url || ''
                          "
                          :alt="selected?.value === 'none' ? 'N' : selected?.label?.slice(0, 2)"
                        />
                        <AvatarFallback>
                          {{
                            selected?.value === 'none'
                              ? 'N'
                              : selected?.label?.slice(0, 2)?.toUpperCase()
                          }}
                        </AvatarFallback>
                      </Avatar>
                      <span class="text-sm">{{
                        selected?.label || t('form.field.selectAgent')
                      }}</span>
                    </div>
                  </template>
                </ComboBox>
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>

          <FormField
            v-slot="{ componentField }"
            name="content"
            class="flex-1 min-h-0 flex flex-col"
          >
            <FormItem class="flex flex-col flex-1">
              <FormLabel>{{ $t('form.field.message') }}</FormLabel>
              <FormControl class="flex-1 min-h-0 flex flex-col">
                <div class="flex-1 min-h-0 flex flex-col">
                  <Editor
                    v-model:htmlContent="componentField.modelValue"
                    @update:htmlContent="(value) => componentField.onChange(value)"
                    :placeholder="t('editor.placeholder')"
                    class="w-full flex-1 overflow-y-auto p-2 min-h-[200px] box"
                  />
                </div>
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>
        </div>

        <DialogFooter class="mt-4 pt-2 border-t shrink-0">
          <Button type="submit" :disabled="loading" :isLoading="loading">
            {{ $t('globals.buttons.submit') }}
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { z } from 'zod'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { ref, watch } from 'vue'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import ComboBox from '@/components/ui/combobox/ComboBox.vue'
import { Users } from 'lucide-vue-next'
import { handleHTTPError } from '@/utils/http'
import { useInboxStore } from '@/stores/inbox'
import { useUsersStore } from '@/stores/users'
import { useTeamStore } from '@/stores/team'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { useI18n } from 'vue-i18n'
import Editor from '@/features/conversation/ConversationTextEditor.vue'
import api from '@/api'

const dialogOpen = defineModel({
  required: false,
  default: () => false
})

const inboxStore = useInboxStore()
const { t } = useI18n()
const uStore = useUsersStore()
const teamStore = useTeamStore()
const emitter = useEmitter()
const loading = ref(false)
const searchResults = ref([])
const emailQuery = ref('')
let timeoutId = null

const formSchema = z.object({
  subject: z.string().min(
    3,
    t('form.error.min', {
      min: 3
    })
  ),
  content: z.string().min(
    1,
    t('globals.messages.cannotBeEmpty', {
      name: t('globals.terms.message')
    })
  ),
  inbox_id: z.any().refine((val) => inboxStore.options.some((option) => option.value === val), {
    message: t('globals.messages.required')
  }),
  team_id: z.any().optional(),
  agent_id: z.any().optional(),
  contact_email: z.string().email(t('globals.messages.invalidEmailAddress')),
  first_name: z.string().min(1, t('globals.messages.required')),
  last_name: z.string().min(1, t('globals.messages.required'))
})

const form = useForm({
  validationSchema: toTypedSchema(formSchema),
  initialValues: {
    inbox_id: null,
    team_id: null,
    agent_id: null,
    subject: '',
    content: '',
    contact_email: '',
    first_name: '',
    last_name: ''
  }
})

watch(emailQuery, (newVal) => {
  form.setFieldValue('contact_email', newVal)
})

const handleSearchContacts = async () => {
  clearTimeout(timeoutId)
  timeoutId = setTimeout(async () => {
    const query = emailQuery.value.trim()

    if (query.length < 3) {
      searchResults.value.splice(0)
      return
    }

    try {
      const resp = await api.searchContacts({ query })
      searchResults.value = [...resp.data.data]
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
      searchResults.value.splice(0)
    }
  }, 300)
}

const selectContact = (contact) => {
  emailQuery.value = contact.email
  form.setFieldValue('first_name', contact.first_name)
  form.setFieldValue('last_name', contact.last_name || '')
  searchResults.value.splice(0)
}

const createConversation = form.handleSubmit(async (values) => {
  loading.value = true
  try {
    await api.createConversation(values)
    dialogOpen.value = false
    form.resetForm()
    emailQuery.value = ''
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    loading.value = false
  }
})
</script>
