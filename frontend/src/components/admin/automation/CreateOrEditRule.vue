<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <Spinner v-if="isLoading"></Spinner>
  <span>{{ formTitle }}</span>
  <div :class="{ 'opacity-50 transition-opacity duration-300': isLoading }">
    <div class="space-y-5">
      <div class="space-y-5">
        <div class="grid w-full max-w-sm items-center gap-1.5">
          <Label for="rule_name">Name</Label>
          <Input id="rule_name" type="text" placeholder="Name for this rule" v-model="rule.name" />
        </div>
        <div class="grid w-full max-w-sm items-center gap-1.5">
          <Label for="rule_name">Description</Label>
          <Input id="rule_name" type="text" placeholder="Description for this rule" v-model="rule.description" />
        </div>
        <div class="grid w-full max-w-sm items-center gap-1.5">
          <Label for="rule_type">Type</Label>
          <Select id="rule_type" :modelValue="rule.type" @update:modelValue="handeTypeUpdate">
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
        </div>
      </div>
      <p class="font-semibold">Match these rules</p>
      <RuleBox :ruleGroup="firstRuleGroup" @update-group="handleUpdateGroup" @add-condition="handleAddCondition"
        @remove-condition="handleRemoveCondition" :groupIndex="0" />
      <div class="flex justify-center">
        <div class="flex items-center space-x-2">
          <Button :class="[groupOperator === 'AND' ? 'bg-black' : 'bg-gray-100 text-black']"
            @click="toggleGroupOperator('AND')">
            AND
          </Button>
          <Button :class="[groupOperator === 'OR' ? 'bg-black' : 'bg-gray-100 text-black']"
            @click="toggleGroupOperator('OR')">
            OR
          </Button>
        </div>
      </div>
      <RuleBox :ruleGroup="secondRuleGroup" @update-group="handleUpdateGroup" @add-condition="handleAddCondition"
        @remove-condition="handleRemoveCondition" :groupIndex="1" />
      <p class="font-semibold">Perform these actions</p>
      <ActionBox :actions="getActions()" :update-actions="handleUpdateActions" @add-action="handleAddAction"
        @remove-action="handleRemoveAction" />
      <Button @click="handleSave" :isLoading="isLoading">Save</Button>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref, computed } from 'vue'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import RuleBox from './RuleBox.vue'
import ActionBox from './ActionBox.vue'
import api from '@/api'
import { useRouter } from 'vue-router'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import { Spinner } from '@/components/ui/spinner'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'

const isLoading = ref(false)
const rule = ref({
  id: 0,
  name: '',
  description: '',
  type: 'new_conversation',
  rules: [
    {
      type: 'new_conversation',
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
  if (props.id > 0) return 'Edit existing rule'
  return 'Create new rule'
})

const breadcrumbLinks = [
  { path: '/admin/automations', label: 'Automations' },
  { path: '#', label: breadcrumbPageLabel() }
]

const router = useRouter()

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

const handeTypeUpdate = (value) => {
  rule.value.type = value
}

const handleSave = async () => {
  try {
    isLoading.value = true
    const updatedRule = { ...rule.value }
    // Delete fields not required.
    delete updatedRule.created_at
    delete updatedRule.updated_at
    if (props.id > 0) await api.updateAutomationRule(props.id, updatedRule)
    else await api.createAutomationRule(updatedRule)
  } finally {
    isLoading.value = false
  }
}

onMounted(async () => {
  if (props.id > 0) {
    try {
      isLoading.value = true
      let resp = await api.getAutomationRule(props.id)
      rule.value = resp.data.data
    } catch (error) {
      console.log(error)
      router.push({ path: `/admin/automations` })
    } finally {
      isLoading.value = false
    }
  }
  firstRuleGroup.value = getFirstGroup()
  secondRuleGroup.value = getSecondGroup()
  groupOperator.value = getGroupOperator()
})
</script>
