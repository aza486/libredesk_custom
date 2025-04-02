<template>
  <form @submit.prevent="onSubmit" class="space-y-6">
    <FormField v-slot="{ field }" name="first_name">
      <FormItem v-auto-animate>
        <FormLabel>{{ $t('form.field.firstName') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="" v-bind="field" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ field }" name="last_name">
      <FormItem>
        <FormLabel>{{ $t('form.field.lastName') }}</FormLabel>
        <FormControl>
          <Input type="text" placeholder="" v-bind="field" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ field }" name="email">
      <FormItem v-auto-animate>
        <FormLabel>{{ $t('form.field.email') }}</FormLabel>
        <FormControl>
          <Input type="email" placeholder="" v-bind="field" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField, handleChange }" name="teams">
      <FormItem v-auto-animate>
        <FormLabel>{{ $t('form.field.teams') }}</FormLabel>
        <FormControl>
          <SelectTag
            :items="teamOptions"
            :placeholder="t('form.field.selectTeams')"
            v-model="componentField.modelValue"
            @update:modelValue="handleChange"
          />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField, handleChange }" name="roles">
      <FormItem v-auto-animate>
        <FormLabel>{{ $t('form.field.roles') }}</FormLabel>
        <FormControl>
          <SelectTag
            :items="roleOptions"
            :placeholder="t('form.field.selectRoles')"
            v-model="componentField.modelValue"
            @update:modelValue="handleChange"
          />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ field }" name="new_password" v-if="!isNewForm">
      <FormItem v-auto-animate>
        <FormLabel>{{ t('form.field.setPassword') }}</FormLabel>
        <FormControl>
          <Input type="password" placeholder="" v-bind="field" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField name="send_welcome_email" v-slot="{ value, handleChange }" v-if="isNewForm">
      <FormItem>
        <FormControl>
          <div class="flex items-center space-x-2">
            <Checkbox :checked="value" @update:checked="handleChange" />
            <Label>{{ $t('form.field.sendWelcomeEmail') }}</Label>
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
          <FormLabel> {{ $t('form.field.enabled') }} </FormLabel>
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
import { createFormSchema } from './formSchema.js'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { vAutoAnimate } from '@formkit/auto-animate/vue'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { SelectTag } from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { useI18n } from 'vue-i18n'
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
const { t } = useI18n()
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

const teamOptions = computed(() =>
  teams.value.map((team) => ({ label: team.name, value: team.name }))
)
const roleOptions = computed(() =>
  roles.value.map((role) => ({ label: role.name, value: role.name }))
)

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t))
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
