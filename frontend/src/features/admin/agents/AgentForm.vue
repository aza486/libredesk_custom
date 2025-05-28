<template>
  <form @submit.prevent="onSubmit" class="space-y-8">
    <!-- Summary Section -->
    <div class="bg-muted/30 box py-6 px-3" v-if="!isNewForm">
      <div class="flex items-start gap-6">
        <Avatar class="w-20 h-20">
          <AvatarImage :src="props.initialValues.avatar_url || ''" :alt="Avatar" />
          <AvatarFallback>
            {{ getInitials(props.initialValues.first_name, props.initialValues.last_name) }}
          </AvatarFallback>
        </Avatar>

        <div class="space-y-4 flex-2">
          <div class="flex items-center gap-3">
            <h3 class="text-lg font-semibold text-gray-900 dark:text-foreground">
              {{ props.initialValues.first_name }} {{ props.initialValues.last_name }}
            </h3>
            <Badge :class="['px-2 rounded-full text-xs font-medium', availabilityStatus.color]">
              {{ availabilityStatus.text }}
            </Badge>
          </div>

          <div class="flex flex-wrap items-center gap-6">
            <div class="flex items-center gap-2">
              <Clock class="w-5 h-5 text-gray-400" />
              <div>
                <p class="text-sm text-gray-500">{{ $t('form.field.lastActive') }}</p>
                <p class="text-sm font-medium text-gray-700 dark:text-foreground">
                  {{
                    props.initialValues.last_active_at
                      ? format(new Date(props.initialValues.last_active_at), 'PPpp')
                      : 'N/A'
                  }}
                </p>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <LogIn class="w-5 h-5 text-gray-400" />
              <div>
                <p class="text-sm text-gray-500">{{ $t('form.field.lastLogin') }}</p>
                <p class="text-sm font-medium text-gray-700 dark:text-foreground">
                  {{
                    props.initialValues.last_login_at
                      ? format(new Date(props.initialValues.last_login_at), 'PPpp')
                      : 'N/A'
                  }}
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Form Fields -->
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

    <FormField v-slot="{ componentField }" name="availability_status" v-if="!isNewForm">
      <FormItem>
        <FormLabel>{{ t('form.field.availabilityStatus') }}</FormLabel>
        <FormControl>
          <Select v-bind="componentField" v-model="componentField.modelValue">
            <SelectTrigger>
              <SelectValue
                :placeholder="
                  t('form.field.select', {
                    name: t('form.field.availabilityStatus')
                  })
                "
              />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="active_group">{{ t('globals.terms.active') }}</SelectItem>
                <SelectItem value="away_manual">{{ t('globals.terms.away') }}</SelectItem>
                <SelectItem value="away_and_reassigning">
                  {{ t('form.field.awayReassigning') }}
                </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
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
import { Badge } from '@/components/ui/badge'
import { Clock, LogIn } from 'lucide-vue-next'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { SelectTag } from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { useI18n } from 'vue-i18n'
import { format } from 'date-fns'
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

const availabilityStatus = computed(() => {
  const status = form.values.availability_status
  if (status === 'active_group') return { text: t('globals.terms.active'), color: 'bg-green-500' }
  if (status === 'away_manual') return { text: t('globals.terms.away'), color: 'bg-yellow-500' }
  if (status === 'away_and_reassigning')
    return { text: t('form.field.awayReassigning'), color: 'bg-orange-500' }
  return { text: t('globals.terms.offline'), color: 'bg-gray-400' }
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
  if (values.availability_status === 'active_group') {
    values.availability_status = 'online'
  }
  values.teams = values.teams.map((team) => ({ name: team }))
  props.submitForm(values)
})

const getInitials = (firstName, lastName) => {
  if (!firstName && !lastName) return ''
  if (!firstName) return lastName.charAt(0).toUpperCase()
  if (!lastName) return firstName.charAt(0).toUpperCase()
  return `${firstName.charAt(0).toUpperCase()}${lastName.charAt(0).toUpperCase()}`
}

watch(
  () => props.initialValues,
  (newValues) => {
    // Hack.
    if (Object.keys(newValues).length > 0) {
      setTimeout(() => {
        if (
          newValues.availability_status === 'away' ||
          newValues.availability_status === 'offline' ||
          newValues.availability_status === 'online'
        ) {
          newValues.availability_status = 'active_group'
        }
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
