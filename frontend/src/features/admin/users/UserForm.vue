<template>
  <form @submit.prevent="onSubmit" class="space-y-6">
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

    <FormField v-slot="{ componentField }" name="teams">
      <FormItem v-auto-animate>
        <FormLabel>Teams</FormLabel>
        <FormControl>
          <SelectTag :items="teamNames" placeholder="Select teams" v-bind="componentField">
          </SelectTag>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="roles">
      <FormItem v-auto-animate>
        <FormLabel>Roles</FormLabel>
        <FormControl>
          <SelectTag :items="roleNames" placeholder="Select roles" v-bind="componentField">
          </SelectTag>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ field }" name="new_password" v-if="!isNewForm">
      <FormItem v-auto-animate>
        <FormLabel>Set password</FormLabel>
        <FormControl>
          <Input type="password" placeholder="Password" v-bind="field" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField name="send_welcome_email" v-slot="{ value, handleChange }" v-if="isNewForm">
      <FormItem>
        <FormControl>
          <div class="flex items-center space-x-2">
            <Checkbox :checked="value" @update:checked="handleChange" />
            <Label>Send welcome email?</Label>
          </div>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ value, handleChange }" type="checkbox" name="enabled" v-if="!isNewForm">
      <FormItem class="flex flex-row items-start gap-x-3 space-y-0">
        <FormControl>
          <Checkbox :checked="value" @update:checked="handleChange" />
        </FormControl>
        <div class="space-y-1 leading-none">
          <FormLabel> Enabled </FormLabel>
          <FormMessage />
        </div>
      </FormItem>
    </FormField>

    <Button type="submit" :isLoading="isLoading"> {{ submitLabel }} </Button>
  </form>
</template>

<script setup>
import { watch, onMounted, ref, computed } from 'vue'
import { Button } from '@/components/ui/button'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { userFormSchema } from './formSchema.js'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { vAutoAnimate } from '@formkit/auto-animate/vue'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { SelectTag } from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import api from '@/api'

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
  },
  isLoading: {
    Type: Boolean,
    required: false
  }
})

const teams = ref([])
const roles = ref([])

onMounted(async () => {
  try {
    const [teamsResp, rolesResp] = await Promise.allSettled([api.getTeams(), api.getRoles()])
    teams.value = teamsResp.value.data.data
    roles.value = rolesResp.value.data.data
  } catch (err) {
    console.log(err)
  }
})

const teamNames = computed(() => teams.value.map((team) => team.name))
const roleNames = computed(() => roles.value.map((role) => role.name))

const form = useForm({
  validationSchema: toTypedSchema(userFormSchema)
})

const onSubmit = form.handleSubmit((values) => {
  values.teams = values.teams.map((team) => ({ name: team }))
  props.submitForm(values)
})

watch(
  () => props.initialValues,
  (newValues) => {
    // Hack.
    if (Object.keys(newValues).length) {
      setTimeout(() => {
        form.setValues(newValues)
        form.setFieldValue(
          'teams',
          newValues.teams.map((team) => team.name)
        )
      }, 0)
    }
  },
  { deep: true, immediate: true }
)
</script>
