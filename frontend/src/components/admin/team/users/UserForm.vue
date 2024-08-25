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

    <FormField name="teams" v-slot="{ componentField }">
      <FormItem>
        <FormLabel>Select teams</FormLabel>
        <FormControl>
          <SelectTag
            v-model="componentField.modelValue"
            :items="teamNames"
            placeHolder="Select teams"
          ></SelectTag>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField name="roles" v-slot="{ componentField }">
      <FormItem>
        <FormLabel>Select roles</FormLabel>
        <FormControl>
          <SelectTag
            v-model="componentField.modelValue"
            :items="roleNames"
            placeHolder="Select roles"
          ></SelectTag>
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

    <Button type="submit" size="sm"> {{ submitLabel }} </Button>
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
  initialValues: props.initialValues
})

const onSubmit = form.handleSubmit((values) => {
  props.submitForm(values)
})

// Watch for changes in initialValues and update the form.
watch(
  () => props.initialValues,
  (newValues) => {
    form.setValues(newValues)
  },
  { immediate: true, deep: true }
)
</script>
