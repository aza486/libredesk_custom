<template>
  <form @submit="onSubmit" class="space-y-6">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem v-auto-animate>
        <FormLabel>Name</FormLabel>
        <FormControl>
          <Input type="text" placeholder="Name" v-bind="componentField" />
        </FormControl>
        <FormDescription>Select an unique name.</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField name="conversation_assignment_type" v-slot="{ componentField }">
      <FormItem>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue placeholder="Select a assignment type" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem v-for="at in assignmentTypes" :key="at" :value="at">
                  {{ at }}
                </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormDescription>
          Round robin: Conversations are assigned to team members in a round-robin fashion. <br>
          Manual: Conversations are manually assigned to team members.
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="timezone">
      <FormItem>
        <FormLabel>Timezone</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue placeholder="Select a timezone" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem v-for="timezone in timezones" :key="timezone" :value="timezone">
                  {{ timezone }}
                </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormDescription>Team's timezone will be used to calculate SLA.</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="business_hours_id">
      <FormItem>
        <FormLabel>Business hours</FormLabel>
        <FormControl>
          <Select v-bind="componentField">
            <SelectTrigger>
              <SelectValue placeholder="Select business hours" />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem v-for="bh in businessHours" :key="bh.id" :value="bh.id">
                  {{ bh.name }}
                </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </FormControl>
        <FormDescription>Default business hours.</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <Button type="submit" :isLoading="isLoading"> {{ submitLabel }} </Button>
  </form>
</template>

<script setup>
import { watch, computed, ref, onMounted } from 'vue'
import { Button } from '@/components/ui/button'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { teamFormSchema } from './teamFormSchema.js'
import { vAutoAnimate } from '@formkit/auto-animate/vue'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@/components/ui/form'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { Input } from '@/components/ui/input'
import { handleHTTPError } from '@/utils/http'
import api from '@/api'

const emitter = useEmitter()
const timezones = computed(() => {
  return Intl.supportedValuesOf('timeZone')
})
const assignmentTypes = ['Round robin', 'Manual']
const businessHours = ref([])
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
  isLoading: {
    type: Boolean,
    required: false,
  }
})

const form = useForm({
  validationSchema: toTypedSchema(teamFormSchema)
})

onMounted(() => {
  fetchBusinessHours()
})

const fetchBusinessHours = async () => {
  try {
    const response = await api.getAllBusinessHours()
    businessHours.value = response.data.data
  } catch (error) {
    // If unauthorized (no permission), show a toast message.
    if (error.response.status === 403) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Unauthorized',
        variant: 'destructive',
        description: 'You do not have permission to view business hours.'
      })
    } else {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Could not fetch business hours',
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    }
  }
}

const onSubmit = form.handleSubmit((values) => {
  props.submitForm(values)
})

// Watch for changes in initialValues and update the form.
watch(
  () => props.initialValues,
  (newValues) => {
    if (Object.keys(newValues).length === 0) return
    form.setValues(newValues)
  },
  { immediate: true }
)
</script>
