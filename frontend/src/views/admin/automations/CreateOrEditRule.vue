<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <Spinner v-if="isLoading"></Spinner>
  <div class="space-y-4">
    <p>{{ formTitle }}</p>
    <div :class="{ 'opacity-50 transition-opacity duration-300': isLoading }">
      <form @submit="onSubmit">
        <div class="space-y-5">
          <div class="space-y-5">
            <FormField
              v-slot="{ value, handleChange }"
              type="checkbox"
              name="enabled"
              v-if="!isNewForm"
            >
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

            <FormField v-slot="{ field }" name="name">
              <FormItem>
                <FormLabel>Name</FormLabel>
                <FormControl>
                  <Input type="text" placeholder="My new rule" v-bind="field" />
                </FormControl>
                <FormDescription>Name for the rule.</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ field }" name="description">
              <FormItem>
                <FormLabel>Description</FormLabel>
                <FormControl>
                  <Input type="text" placeholder="Description for new rule" v-bind="field" />
                </FormControl>
                <FormDescription>Description for the rule.</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>

            <FormField v-slot="{ componentField, handleInput }" name="type">
              <FormItem>
                <FormLabel>Type</FormLabel>
                <FormControl>
                  <Select v-bind="componentField" @update:modelValue="handleInput">
                    <SelectTrigger>
                      <SelectValue placeholder="Select a type" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectGroup>
                        <SelectItem value="new_conversation"> New conversation </SelectItem>
                        <SelectItem value="conversation_update"> Conversation update </SelectItem>
                        <SelectItem value="time_trigger"> Time trigger </SelectItem>
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                </FormControl>
                <FormDescription>Type of rule.</FormDescription>
                <FormMessage />
              </FormItem>
            </FormField>

            <div :class="{ hidden: form.values.type !== 'conversation_update' }">
              <FormField v-slot="{ componentField }" name="events">
                <FormItem>
                  <FormLabel>Events</FormLabel>
                  <FormControl>
                    <SelectTag
                      v-bind="componentField"
                      :items="conversationEvents || []"
                      placeholder="Select events"
                    >
                    </SelectTag>
                  </FormControl>
                  <FormDescription>Evaluate rule on these events.</FormDescription>
                  <FormMessage></FormMessage>
                </FormItem>
              </FormField>
            </div>
          </div>

          <p class="font-semibold">Match these rules</p>

          <RuleBox
            :ruleGroup="firstRuleGroup"
            @update-group="handleUpdateGroup"
            @add-condition="handleAddCondition"
            @remove-condition="handleRemoveCondition"
            :type="form.values.type"
            :groupIndex="0"
          />

          <div class="flex justify-center">
            <div class="flex items-center space-x-2">
              <Button
                :class="[groupOperator === 'AND' ? 'bg-black' : 'bg-gray-100 text-black']"
                @click.prevent="toggleGroupOperator('AND')"
              >
                AND
              </Button>
              <Button
                :class="[groupOperator === 'OR' ? 'bg-black' : 'bg-gray-100 text-black']"
                @click.prevent="toggleGroupOperator('OR')"
              >
                OR
              </Button>
            </div>
          </div>

          <RuleBox
            :ruleGroup="secondRuleGroup"
            @update-group="handleUpdateGroup"
            @add-condition="handleAddCondition"
            @remove-condition="handleRemoveCondition"
            :type="form.values.type"
            :groupIndex="1"
          />
          <p class="font-semibold">Perform these actions</p>

          <ActionBox
            :actions="getActions()"
            :update-actions="handleUpdateActions"
            @add-action="handleAddAction"
            @remove-action="handleRemoveAction"
          />
          <Button type="submit" :isLoading="isLoading">Save</Button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref, computed } from 'vue'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import RuleBox from '@/features/admin/automation/RuleBox.vue'
import ActionBox from '@/features/admin/automation/ActionBox.vue'
import api from '@/api'
import { Checkbox } from '@/components/ui/checkbox'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { formSchema } from '@/features/admin/automation/formSchema.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import { SelectTag } from '@/components/ui/select'
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
import { Spinner } from '@/components/ui/spinner'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { useRoute } from 'vue-router'
import { useRouter } from 'vue-router'

const isLoading = ref(false)
const route = useRoute()
const router = useRouter()
const emitter = useEmitter()
const rule = ref({
  id: 0,
  name: '',
  description: '',
  type: 'new_conversation',
  rules: [
    {
      groups: [
        {
          rules: [],
          logical_op: 'OR'
        },
        {
          rules: [],
          logical_op: 'OR'
        }
      ],
      actions: [],
      group_operator: 'OR'
    }
  ]
})

const conversationEvents = [
  'conversation.user.assigned',
  'conversation.team.assigned',
  'conversation.priority.change',
  'conversation.status.change',
  'conversation.message.outgoing',
  'conversation.message.incoming'
]

const props = defineProps({
  id: {
    type: [String, Number],
    required: false
  }
})

const breadcrumbPageLabel = () => {
  if (props.id > 0) return 'Edit rule'
  return 'New rule'
}

const formTitle = computed(() => {
  return ''
  // if (props.id > 0) return 'Edit existing rule'
  // return 'Create new rule'
})

const isNewForm = computed(() => {
  return props.id ? false : true
})

const breadcrumbLinks = [
  { path: 'automations', label: 'Automations' },
  { path: '', label: breadcrumbPageLabel() }
]

const firstRuleGroup = ref([])
const secondRuleGroup = ref([])
const groupOperator = ref('')

const getFirstGroup = () => {
  if (rule.value.rules?.[0]?.groups?.[0]) {
    return rule.value.rules[0].groups[0]
  }
  return []
}

const getSecondGroup = () => {
  if (rule.value.rules?.[0]?.groups?.[1]) {
    return rule.value.rules[0].groups[1]
  }
  return []
}

const getActions = () => {
  if (rule.value.rules?.[0]?.actions) {
    return rule.value.rules[0].actions
  }
  return []
}

const toggleGroupOperator = (value) => {
  if (rule.value.rules?.[0]) {
    rule.value.rules[0].group_operator = value
    groupOperator.value = value
  }
}

const getGroupOperator = () => {
  if (rule.value.rules?.[0]) {
    return rule.value.rules[0].group_operator
  }
  return ''
}

const handleUpdateGroup = (value, groupIndex) => {
  rule.value.rules[0].groups[groupIndex] = value.value
}

const handleAddCondition = (groupIndex) => {
  rule.value.rules[0].groups[groupIndex].rules.push({})
}

const handleRemoveCondition = (groupIndex, ruleIndex) => {
  rule.value.rules[0].groups[groupIndex].rules.splice(ruleIndex, 1)
}

const handleUpdateActions = (value, index) => {
  rule.value.rules[0].actions[index] = value
}

const handleAddAction = () => {
  rule.value.rules[0].actions.push({})
}

const handleRemoveAction = (index) => {
  rule.value.rules[0].actions.splice(index, 1)
}

const form = useForm({
  validationSchema: toTypedSchema(formSchema)
})

const onSubmit = form.handleSubmit(async (values) => {
  handleSave(values)
})

const handleSave = async (values) => {
  if (!areRulesValid()) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Invalid rules',
      variant: 'destructive',
      description: 'Make sure you have atleast one action and one rule.'
    })
    return
  }

  try {
    isLoading.value = true
    const updatedRule = { ...rule.value, ...values }
    // Delete fields not required.
    delete updatedRule.created_at
    delete updatedRule.updated_at
    if (props.id > 0) {
      await api.updateAutomationRule(props.id, updatedRule)
    } else {
      await api.createAutomationRule(updatedRule)
      router.push({ name: 'automations' })
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Success',
      description: 'Rule saved successfully'
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Could not save rule',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

// TODO: Add some vee-validate validations.
const areRulesValid = () => {
  // Must have atleast one action.
  if (rule.value.rules[0].actions.length == 0) {
    return false
  }

  // Must have atleast 1 group.
  if (rule.value.rules[0].groups.length == 0) {
    return false
  }

  // Group should have atleast one rule.
  if (rule.value.rules[0].groups[0].rules.length == 0) {
    return false
  }

  // Make sure each rule has all the required fields.
  for (const group of rule.value.rules[0].groups) {
    for (const rule of group.rules) {
      if (!rule.value || !rule.operator || !rule.field) {
        return false
      }
    }
  }
  return true
}

onMounted(async () => {
  if (props.id > 0) {
    try {
      isLoading.value = true
      let resp = await api.getAutomationRule(props.id)
      rule.value = resp.data.data
      if (resp.data.data.type === 'conversation_update') {
        rule.value.rules.events = []
      }
      form.setValues(resp.data.data)
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        title: 'Could not fetch rule',
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    } finally {
      isLoading.value = false
    }
  }
  if (route.query.type) {
    form.setFieldValue('type', route.query.type)
  }
  firstRuleGroup.value = getFirstGroup()
  secondRuleGroup.value = getSecondGroup()
  groupOperator.value = getGroupOperator()
})
</script>
