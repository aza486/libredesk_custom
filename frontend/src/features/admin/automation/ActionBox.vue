<template>
  <div class="space-y-5 rounded-lg" :class="{ 'box p-5': actions.length > 0 }">
    <div class="space-y-5">
      <div v-for="(action, index) in actions" :key="index" class="space-y-5">
        <div v-if="index > 0">
          <hr class="border-t-2 border-dotted border-gray-300" />
        </div>

        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <div class="flex gap-5">
              <div class="w-48">
                <!-- Type -->
                <Select
                  v-model="action.type"
                  @update:modelValue="(value) => handleFieldChange(value, index)"
                >
                  <SelectTrigger class="m-auto">
                    <SelectValue placeholder="Select action" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      <SelectItem
                        v-for="(actionConfig, key) in conversationActions"
                        :key="key"
                        :value="key"
                      >
                        {{ actionConfig.label }}
                      </SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </div>

              <!-- Value -->
              <div
                v-if="action.type && conversationActions[action.type]?.type === 'tag'"
                class="w-full"
              >
                <SelectTag
                  v-model="action.value"
                  :items="tagsStore.tagNames"
                  placeholder="Select tag"
                />
              </div>

              <div
                class="w-48"
                v-if="action.type && conversationActions[action.type]?.type === 'select'"
              >
                <ComboBox
                  v-model="action.value[0]"
                  :items="conversationActions[action.type]?.options"
                  placeholder="Select"
                  @select="handleValueChange($event, index)"
                >
                  <template #item="{ item }">
                    <div class="flex items-center gap-2 ml-2">
                      <Avatar v-if="action.type === 'assign_user'" class="w-7 h-7">
                        <AvatarImage :src="item.avatar_url ?? ''" :alt="item.label.slice(0, 2)" />
                        <AvatarFallback>
                          {{ item.label.slice(0, 2).toUpperCase() }}
                        </AvatarFallback>
                      </Avatar>
                      <span v-if="action.type === 'assign_team'">
                        {{ item.emoji }}
                      </span>
                      <span>{{ item.label }}</span>
                    </div>
                  </template>

                  <template #selected="{ selected }">
                    <div v-if="action.type === 'assign_team'">
                      <div v-if="selected" class="flex items-center gap-2">
                        {{ selected.emoji }}
                        <span>{{ selected.label }}</span>
                      </div>
                      <span v-else>Select team</span>
                    </div>

                    <div v-else-if="action.type === 'assign_user'" class="flex items-center gap-2">
                      <div v-if="selected" class="flex items-center gap-2">
                        <Avatar class="w-7 h-7">
                          <AvatarImage
                            :src="selected.avatar_url ?? ''"
                            :alt="selected.label.slice(0, 2)"
                          />
                          <AvatarFallback>
                            {{ selected.label.slice(0, 2).toUpperCase() }}
                          </AvatarFallback>
                        </Avatar>
                        <span>{{ selected.label }}</span>
                      </div>
                      <span v-else>Select user</span>
                    </div>
                    <span v-else>
                      <span v-if="!selected"> Select</span>
                      <span v-else>{{ selected.label }} </span>
                    </span>
                  </template>
                </ComboBox>
              </div>
            </div>

            <div class="cursor-pointer" @click.prevent="removeAction(index)">
              <X size="16" />
            </div>
          </div>

          <div
            class="box p-2 h-96 min-h-96"
            v-if="action.type && conversationActions[action.type]?.type === 'richtext'"
          >
            <Editor
              v-model:htmlContent="action.value[0]"
              @update:htmlContent="(value) => handleEditorChange(value, index)"
              :placeholder="'Shift + Enter to add new line'"
            />
          </div>
        </div>
      </div>
    </div>
    <div>
      <Button variant="outline" @click.prevent="addAction">Add action</Button>
    </div>
  </div>
</template>

<script setup>
import { toRefs } from 'vue'
import { Button } from '@/components/ui/button'
import { X } from 'lucide-vue-next'
import { useTagStore } from '@/stores/tag'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import ComboBox from '@/components/ui/combobox/ComboBox.vue'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { SelectTag } from '@/components/ui/select'
import { useConversationFilters } from '@/composables/useConversationFilters'
import { getTextFromHTML } from '@/utils/strings.js'
import Editor from '@/features/conversation/ConversationTextEditor.vue'

const props = defineProps({
  actions: {
    type: Array,
    required: true
  }
})

const { actions } = toRefs(props)
const emit = defineEmits(['update-actions', 'add-action', 'remove-action'])
const tagsStore = useTagStore()
const { conversationActions } = useConversationFilters()

const handleFieldChange = (value, index) => {
  actions.value[index].value = []
  actions.value[index].type = value
  emitUpdate(index)
}

const handleValueChange = (value, index) => {
  if (typeof value === 'object') {
    value = value.value
  }
  actions.value[index].value = [value]
  emitUpdate(index)
}

const handleEditorChange = (value, index) => {
  // If text is empty, set HTML to empty string
  const textContent = getTextFromHTML(value)
  if (textContent.length === 0) {
    value = ''
  }
  actions.value[index].value = [value]
  emitUpdate(index)
}

const removeAction = (index) => {
  emit('remove-action', index)
}

const addAction = () => {
  emit('add-action')
}

const emitUpdate = (index) => {
  emit('update-actions', actions, index)
}
</script>
