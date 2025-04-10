<template>
  <ContactDetail>
    <div class="flex flex-col mx-auto items-start">
      <div class="mb-6">
        <CustomBreadcrumb :links="breadcrumbLinks" />
      </div>

      <div v-if="contact" class="flex justify-center space-y-4 w-full">
        <!-- Card -->
        <div class="flex flex-col w-full">
          <div class="h-16"></div>

          <div class="flex flex-col space-y-2">
            <!-- Avatar with upload-->
            <AvatarUpload
              @upload="onUpload"
              @remove="onRemove"
              :src="contact.avatar_url"
              :initials="getInitials"
              :label="t('globals.buttons.upload')"
            />

            <div>
              <h2 class="text-2xl font-bold text-gray-900">
                {{ contact.first_name }} {{ contact.last_name }}
              </h2>
            </div>

            <div class="text-xs text-gray-500">
              {{ $t('form.field.createdOn') }}
              {{ contact.created_at ? format(new Date(contact.created_at), 'PPP') : 'N/A' }}
            </div>

            <div class="w-30 pt-3">
              <Button
                :variant="contact.enabled ? 'destructive' : 'outline'"
                @click="showBlockConfirmation = true"
                size="sm"
              >
                <ShieldOffIcon v-if="contact.enabled" size="18" class="mr-2" />
                <ShieldCheckIcon v-else size="18" class="mr-2" />
                {{ t(contact.enabled ? 'globals.buttons.block' : 'globals.buttons.unblock') }}
              </Button>
            </div>
          </div>

          <div class="mt-12">
            <form @submit.prevent="onSubmit" class="space-y-8">
              <div class="flex flex-wrap gap-6">
                <div class="flex-1">
                  <FormField v-slot="{ componentField }" name="first_name">
                    <FormItem class="flex flex-col">
                      <FormLabel class="flex items-center">
                        {{ t('form.field.firstName') }}
                      </FormLabel>
                      <FormControl>
                        <Input v-bind="componentField" type="text" />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  </FormField>
                </div>

                <div class="flex-1">
                  <FormField v-slot="{ componentField }" name="last_name" class="flex-1">
                    <FormItem class="flex flex-col">
                      <FormLabel class="flex items-center">
                        {{ t('form.field.lastName') }}
                      </FormLabel>
                      <FormControl>
                        <Input v-bind="componentField" type="text" />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  </FormField>
                </div>
              </div>

              <FormField v-slot="{ componentField }" name="avatar_url">
                <FormItem>
                  <FormControl>
                    <Input v-bind="componentField" type="hidden" />
                  </FormControl>
                </FormItem>
              </FormField>

              <div class="flex flex-wrap gap-6">
                <div class="flex-1">
                  <FormField v-slot="{ componentField }" name="email">
                    <FormItem class="flex flex-col">
                      <FormLabel class="flex items-center">
                        {{ t('form.field.email') }}
                      </FormLabel>
                      <FormControl>
                        <Input v-bind="componentField" type="email" />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  </FormField>
                </div>

                <div class="flex flex-col flex-1">
                  <div class="flex items-end">
                    <FormField v-slot="{ componentField }" name="phone_number_calling_code">
                      <FormItem class="w-20">
                        <FormLabel class="flex items-center whitespace-nowrap">
                          {{ t('form.field.phoneNumber') }}
                        </FormLabel>
                        <FormControl>
                          <ComboBox
                            v-bind="componentField"
                            :items="allCountries"
                            :placeholder="t('form.field.select')"
                            :buttonClass="'rounded-r-none border-r-0'"
                          >
                            <template #item="{ item }">
                              <div class="flex items-center gap-2">
                                <div class="w-7 h-7 flex items-center justify-center">
                                  <span v-if="item.emoji">{{ item.emoji }}</span>
                                </div>
                                <span class="text-sm">{{ item.label }} ( {{ item.value }})</span>
                              </div>
                            </template>

                            <template #selected="{ selected }">
                              <div class="flex items-center mb-1">
                                <span v-if="selected" class="text-xl leading-none">
                                  {{ selected.emoji }}
                                </span>
                              </div>
                            </template>
                          </ComboBox>
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    </FormField>

                    <div class="flex-1">
                      <FormField v-slot="{ componentField }" name="phone_number">
                        <FormItem class="relative">
                          <FormControl>
                            <!-- Input field -->
                            <Input
                              type="tel"
                              v-bind="componentField"
                              class="rounded-l-none"
                              inputmode="numeric"
                            />
                            <FormMessage class="absolute top-full mt-1 text-sm" />
                          </FormControl>
                        </FormItem>
                      </FormField>
                    </div>
                  </div>
                </div>
              </div>

              <div>
                <Button type="submit" :isLoading="formLoading" :disabled="formLoading">
                  {{
                    $t('globals.buttons.update', {
                      name: $t('globals.terms.contact').toLowerCase()
                    })
                  }}
                </Button>
              </div>
            </form>
          </div>
        </div>
      </div>

      <!-- Loading state -->
      <Spinner v-else />

      <!-- Block/Unblock confirmation dialog -->
      <Dialog :open="showBlockConfirmation" @update:open="showBlockConfirmation = $event">
        <DialogContent class="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{{
              contact?.enabled
                ? t('globals.buttons.block', {
                    name: t('globals.terms.contact')
                  })
                : t('globals.buttons.unblock', {
                    name: t('globals.terms.contact')
                  })
            }}</DialogTitle>
            <DialogDescription>
              {{ contact?.enabled ? t('contact.blockConfirm') : t('contact.unblockConfirm') }}
            </DialogDescription>
          </DialogHeader>
          <div class="flex justify-end space-x-2 pt-4">
            <Button variant="outline" @click="showBlockConfirmation = false">
              {{ t('globals.buttons.cancel') }}
            </Button>
            <Button
              :variant="contact?.enabled ? 'destructive' : 'default'"
              @click="confirmToggleBlock"
            >
              {{ contact?.enabled ? t('globals.buttons.block') : t('globals.buttons.unblock') }}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  </ContactDetail>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { format } from 'date-fns'
import { useI18n } from 'vue-i18n'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { AvatarUpload } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { FormField, FormItem, FormLabel, FormControl, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription
} from '@/components/ui/dialog'
import { ShieldOffIcon, ShieldCheckIcon } from 'lucide-vue-next'
import ContactDetail from '@/layouts/contact/ContactDetail.vue'
import api from '@/api'
import { createFormSchema } from './formSchema.js'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { handleHTTPError } from '@/utils/http'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { Spinner } from '@/components/ui/spinner'
import ComboBox from '@/components/ui/combobox/ComboBox.vue'
import countries from '@/constants/countries.js'

const { t } = useI18n()
const emitter = useEmitter()
const route = useRoute()
const formLoading = ref(false)
const contact = ref(null)
const showBlockConfirmation = ref(false)
const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t))
})

const allCountries = countries.map((country) => ({
  label: country.name,
  value: country.calling_code,
  emoji: country.emoji
}))

const breadcrumbLinks = [
  { path: 'contacts', label: t('globals.terms.contact', 2) },
  {
    path: '',
    label: t('globals.messages.edit', {
      name: t('globals.terms.contact')
    })
  }
]

onMounted(() => {
  fetchContact()
})

async function fetchContact() {
  try {
    const { data } = await api.getContact(route.params.id)
    contact.value = data.data
    form.setValues(data.data)
  } catch (err) {
    showError(err)
  }
}

const getInitials = computed(() => {
  if (!contact.value) return ''
  const firstName = contact.value.first_name || ''
  const lastName = contact.value.last_name || ''
  return `${firstName.charAt(0).toUpperCase()}${lastName.charAt(0).toUpperCase()}`
})

async function confirmToggleBlock() {
  showBlockConfirmation.value = false
  await toggleBlock()
}

async function toggleBlock() {
  try {
    form.setFieldValue('enabled', !contact.value.enabled)
    await onSubmit(form.values)
    await fetchContact()
    const messageKey = contact.value.enabled
      ? 'globals.messages.unblockedSuccessfully'
      : 'globals.messages.blockedSuccessfully'
    emitToast(t(messageKey, { name: t('globals.terms.contact') }))
  } catch (err) {
    showError(err)
  }
}

const onSubmit = form.handleSubmit(async (values) => {
  try {
    formLoading.value = true
    await api.updateContact(contact.value.id, {
      ...values
    })
    await fetchContact()
    emitToast(t('globals.messages.updatedSuccessfully', { name: t('globals.terms.contact') }))
  } catch (err) {
    showError(err)
  } finally {
    formLoading.value = false
  }
})

async function onUpload(file) {
  try {
    formLoading.value = true
    const formData = new FormData()
    formData.append('files', file)
    formData.append('first_name', form.values.first_name)
    formData.append('last_name', form.values.last_name)
    formData.append('email', form.values.email)
    formData.append('phone_number', form.values.phone_number)
    formData.append('phone_number_calling_code', form.values.phone_number_calling_code)
    formData.append('enabled', form.values.enabled)
    const { data } = await api.updateContact(contact.value.id, formData)
    contact.value.avatar_url = data.avatar_url
    form.setFieldValue('avatar_url', data.avatar_url)
    emitToast(t('globals.messages.updatedSuccessfully', { name: t('globals.terms.avatar') }))
    fetchContact()
  } catch (err) {
    showError(err)
  } finally {
    formLoading.value = false
  }
}

async function onRemove() {
  contact.value.avatar_url = null
  form.setFieldValue('avatar_url', null)
  await onUpload(null)
}

function emitToast(description) {
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description })
}

function showError(err) {
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
    variant: 'destructive',
    description: handleHTTPError(err).message
  })
}
</script>
