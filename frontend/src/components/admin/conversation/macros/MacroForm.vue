<template>
  <Spinner v-if="formLoading"></Spinner>
  <form @submit="onSubmit" class="space-y-6 w-full" :class="{ 'opacity-50': formLoading }">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>Name</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Macro name" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="message_content">
      <FormItem>
        <FormLabel>Response to be sent when macro is used</FormLabel>
        <FormControl>
          <QuillEditor
            v-model:content="componentField.modelValue"
            placeholder="Add a response (optional)"
            theme="snow"
            contentType="html"
            class="h-32 mb-12"
            @update:content="(value) => componentField.onChange(value)"
          />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="visibility">
      <FormItem>
        <FormLabel>Visibility</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue placeholder="Select visibility" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="all">All</SelectItem>
                <SelectItem value="team">Team</SelectItem>
                <SelectItem value="user">User</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-if="form.values.visibility === 'team'" v-slot="{ componentField }" name="team_id">
      <FormItem>
        <FormLabel>Team</FormLabel>
        <FormControl>
          <ComboBox v-bind="componentField" :items="tStore.options" placeholder="Select team">
            <template #item="{ item }">
              <div class="flex items-center gap-2 ml-2">
                <span>{{ item.emoji }}</span>
                <span>{{ item.label }}</span>
              </div>
            </template>
            <template #selected="{ selected }">
              <div class="flex items-center gap-2">
                <span v-if="selected">
                  {{ selected.emoji }}
                  <span>{{ selected.label }}</span>
                </span>
                <span v-else>Select team</span>
              </div>
            </template>
          </ComboBox>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-if="form.values.visibility === 'user'" v-slot="{ componentField }" name="user_id">
      <FormItem>
        <FormLabel>User</FormLabel>
        <FormControl>
          <ComboBox v-bind="componentField" :items="uStore.options" placeholder="Select user">
            <template #item="{ item }">
              <div class="flex items-center gap-2 ml-2">
                <Avatar class="w-7 h-7">
                  <AvatarImage :src="item.avatar_url" :alt="item.label.slice(0, 2)" />
                  <AvatarFallback>{{ item.label.slice(0, 2).toUpperCase() }}</AvatarFallback>
                </Avatar>
                <span>{{ item.label }}</span>
              </div>
            </template>
            <template #selected="{ selected }">
              <div class="flex items-center gap-2">
                <div v-if="selected" class="flex items-center gap-2">
                  <Avatar class="w-7 h-7">
                    <AvatarImage :src="selected.avatar_url" :alt="selected.label.slice(0, 2)" />
                    <AvatarFallback>{{ selected.label.slice(0, 2).toUpperCase() }}</AvatarFallback>
                  </Avatar>
                  <span>{{ selected.label }}</span>
                </div>
                <span v-else>Select user</span>
              </div>
            </template>
          </ComboBox>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="actions">
      <FormItem>
        <FormLabel> Actions </FormLabel>
        <FormControl>
          <ActionBuilder v-bind="componentField" :config="actionConfig" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>
    <Button type="submit" :isLoading="isLoading">{{ submitLabel }}</Button>
  </form>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'
import { Input } from '@/components/ui/input'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Avatar, AvatarImage, AvatarFallback } from '@/components/ui/avatar'
import ActionBuilder from '@/components/common/ActionBuilder.vue'
import { useConversationFilters } from '@/composables/useConversationFilters'
import { useUsersStore } from '@/stores/users'
import { useTeamStore } from '@/stores/team'
import { formSchema } from './formSchema.js'
import { QuillEditor } from '@vueup/vue-quill'
import '@vueup/vue-quill/dist/vue-quill.snow.css'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import ComboBox from '@/components/ui/combobox/ComboBox.vue'

const { macroActions } = useConversationFilters()
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
    default: 'Submit'
  },
  isLoading: {
    type: Boolean,
    default: false
  }
})

const form = useForm({
  validationSchema: toTypedSchema(formSchema)
})

const actionConfig = ref({
  actions: macroActions,
  typePlaceholder: 'Select action type',
  valuePlaceholder: 'Select value',
  addButtonText: 'Add new action'
})

const onSubmit = form.handleSubmit(async (values) => {
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
