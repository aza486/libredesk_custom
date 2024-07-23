<template>
  <form @submit.prevent="onSubmit" class="w-2/3 space-y-6">
    <FormField v-slot="{ field }" name="first_name">
      <FormItem v-auto-animate>
        <FormLabel>First name</FormLabel>
        <FormControl>
          <Input type="text" placeholder="First name" v-bind="field" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>
    <FormField v-slot="{ field }" name="last_name">
      <FormItem>
        <FormLabel>Last name</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Last name" v-bind="field" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ field }" name="email">
      <FormItem v-auto-animate>
        <FormLabel>Email</FormLabel>
        <FormControl>
          <Input type="email" placeholder="Email" v-bind="field" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField name="team_id" v-slot="{ field }">
      <FormItem>
        <FormLabel>Team</FormLabel>
        <FormControl>
          <Select v-bind="field" :modelValue="props.initialValues.team_id">
            <SelectTrigger>
              <SelectValue placeholder="Select a team" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem v-for="team in teams" :key="team.id" :value="team.id">
                  {{ team.name }}
                </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField name="roles" v-slot="{ field }">
      <FormItem>
        <FormLabel>Select role</FormLabel>
        <FormControl>
          <TagsInput v-model="roles" class="px-0 gap-0 shadow-sm">
            <div class="flex gap-2 flex-wrap items-center px-3">
              <TagsInputItem v-for="item in roles" :key="item" :value="item">
                <TagsInputItemText />
                <TagsInputItemDelete />
              </TagsInputItem>
            </div>

            <ComboboxRoot v-model="roles" v-model:open="open" v-model:searchTerm="searchTerm" class="w-full" v-bind="field">
              <ComboboxAnchor as-child>
                <ComboboxInput placeholder="Select role" as-child>
                  <TagsInputInput class="w-full px-3" :class="roles.length > 0 ? 'mt-2' : ''" @keydown.enter.prevent />
                </ComboboxInput>
              </ComboboxAnchor>

              <ComboboxPortal>
                <CommandList position="popper" class="w-[--radix-popper-anchor-width] rounded-md mt-2 border bg-popover text-popover-foreground shadow-md outline-none data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2">
                  <CommandEmpty />
                  <CommandGroup>
                    <CommandItem v-for="user in filteredUsers" :key="user.value" :value="user.label" @select.prevent="(ev) => {
                      if (typeof ev.detail.value === 'string') {
                        searchTerm = ''
                        roles.push(ev.detail.value)
                      }

                      if (filteredUsers.length === 0) {
                        open = false
                      }
                    }">
                      {{ user.label }}
                    </CommandItem>
                  </CommandGroup>
                </CommandList>
              </ComboboxPortal>
            </ComboboxRoot>
          </TagsInput>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField name="send_welcome_email" v-slot="{ field }" v-if="isNewForm">
      <FormItem>
        <FormControl>
          <div class="flex items-center space-x-2">
            <Checkbox id="send_welcome_email" :checked="field.value" @update:checked="field.handleChange" />
            <Label for="send_welcome_email">Send welcome email</Label>
          </div>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <Button type="submit" size="sm"> {{ submitLabel }} </Button>
  </form>
</template>

<script setup>
import { watch, onMounted, ref, computed } from 'vue'
import { Button } from '@/components/ui/button'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { getUserFormSchema } from './userFormSchema.js'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { vAutoAnimate } from '@formkit/auto-animate/vue'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { ComboboxAnchor, ComboboxInput, ComboboxPortal, ComboboxRoot } from 'radix-vue'
import { CommandEmpty, CommandGroup, CommandItem, CommandList } from '@/components/ui/command'
import { TagsInput, TagsInputInput, TagsInputItem, TagsInputItemDelete, TagsInputItemText } from '@/components/ui/tags-input'
import { Input } from '@/components/ui/input'
import api from '@/api'

const teams = ref([])

const frameworks = [
  { value: 'Agent', label: 'Agent' },
  { value: 'Admin', label: 'Admin' },
]

const roles = ref([])
const open = ref(false)
const searchTerm = ref('')

const filteredUsers = computed(() => frameworks.filter(i => !roles.value.includes(i.label)))

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
    default: 'Submit'
  },
  isNewForm: {
    type: Boolean,
    required: false,
    default: false
  }
})

onMounted(async () => {
  try {
    const resp = await api.getTeams()
    teams.value = resp.data.data
  } catch (err) {
    console.log(err)
  }

  if (props.initialValues && props.initialValues.roles) {
    roles.value = props.initialValues.roles
  }
})

const form = useForm({
  validationSchema: toTypedSchema(getUserFormSchema(teams.value, props.isNewForm)),
  initialValues: props.initialValues
})

const onSubmit = form.handleSubmit((values) => {
  values.roles = roles.value
  props.submitForm(values)
})

// Watch for changes in initialValues and update the form.
watch(
  () => props.initialValues,
  (newValues) => {
    form.setValues(newValues)
    if (newValues && newValues.roles) {
      roles.value = newValues.roles
    }
  },
  { immediate: true }
)
</script>
