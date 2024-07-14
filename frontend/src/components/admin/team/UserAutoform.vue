<template>
  <form @submit="onSubmit" class="w-2/3 space-y-6">
    <FormField v-slot="{ componentField }" name="first_name">
      <FormItem v-auto-animate>
        <FormLabel>First name</FormLabel>
        <FormControl>
          <Input type="text" placeholder="First name" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>
    <FormField v-slot="{ componentField }" name="last_name">
      <FormItem>
        <FormLabel>Last name</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Last name" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="email">
      <FormItem v-auto-animate>
        <FormLabel>Email</FormLabel>
        <FormControl>
          <Input type="email" placeholder="email" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField name="team_id" v-slot="{ componentField }">
      <FormItem>
        <FormLabel>Team</FormLabel>
        <FormControl>
          <Select v-bind="componentField" :modelValue="props.initialValues.team_id">
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

    <FormField name="send_welcome_email" v-slot="{ value, handleChange }" v-if="isNewForm">
      <FormItem>
        <FormControl>
          <div class="flex items-center space-x-2">
            <Checkbox id="send_welcome_email" :checked="value" @update:checked="handleChange" />
            <Label for="send_welcome_email">Send welcome email</Label>
          </div>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <Button type="submit"> Submit </Button>
  </form>
</template>

<script setup>
import { watch, onMounted, ref } from 'vue'
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
import { Input } from '@/components/ui/input'
import api from '@/api'

const teams = ref([])

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
  isNewForm: {
    type: Boolean,
    required: false,
    default: () => false
  }
})

onMounted(async () => {
  try {
    const resp = await api.getTeams()
    teams.value = resp.data.data
  } catch (err) {
    console.log(err)
  }
})

const form = useForm({
  validationSchema: toTypedSchema(getUserFormSchema(teams.value, props.isNewForm)),
  initialValues: props.initialValues
})

const onSubmit = form.handleSubmit((values) => {
  console.log('values -> ', values)
  props.submitForm(values)
})

// Watch for changes in initialValues and update the form.
watch(
  () => props.initialValues,
  (newValues) => {
    form.setValues(newValues)
  },
  { immediate: true }
)
</script>
