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


    <FormField name="teams">
      <FormItem v-auto-animate>
        <FormLabel>Teams</FormLabel>
        <FormControl>
          <SelectTag v-model="selectedTeams" :items="teamNames" :initialValue="initialTeamNames"
            placeHolder="Select teams"></SelectTag>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField name="roles">
      <FormItem v-auto-animate>
        <FormLabel>Roles</FormLabel>
        <FormControl>
          <SelectTag v-model="selectedRoles" :items="roleNames" :initialValue="initialValues.roles"
            placeHolder="Select roles"></SelectTag>
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

    <Button type="submit" size="sm" :isLoading="isLoading"> {{ submitLabel }} </Button>
  </form>
</template>

<script setup>
import { watch, onMounted, ref, computed } from 'vue'
import { Button } from '@/components/ui/button'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { userFormSchema } from './userFormSchema.js'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { vAutoAnimate } from '@formkit/auto-animate/vue'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { SelectTag } from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import api from '@/api'

const teams = ref([])
const roles = ref([])
const selectedRoles = ref([])
const selectedTeams = ref([])
const initialTeamNames = computed(() => props.initialValues.teams?.map(team => team.name) || [])

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

onMounted(async () => {
  try {
    const [teamsResp, rolesResp] = await Promise.all([api.getTeams(), api.getRoles()])
    teams.value = teamsResp.data.data
    roles.value = rolesResp.data.data
  } catch (err) {
    console.log(err)
  }
})

const teamNames = computed(() => teams.value.map((team) => team.name))
const roleNames = computed(() => roles.value.map((role) => role.name))

const form = useForm({
  validationSchema: toTypedSchema(userFormSchema),
})

const onSubmit = form.handleSubmit((values) => {
  values.teams = selectedTeams.value.map(team => ({ name: team }))
  values.roles = selectedRoles.value
  props.submitForm(values)
})

watch(
  () => props.initialValues,
  (newValues) => {
    // Hack.
    if (Object.keys(newValues).length)
      setTimeout(() => form.setValues(newValues), 0)
  },
  { deep: true, immediate: true }
)

</script>
