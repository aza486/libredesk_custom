<template>
  <TagsInput v-model="tags" class="px-0 gap-0">
     <div class="flex gap-2 flex-wrap items-center px-3">
       <TagsInputItem v-for="tag in tags" :key="tag" :value="tag">
         <TagsInputItemText>{{ tag }}</TagsInputItemText>
         <TagsInputItemDelete />
       </TagsInputItem>
     </div>
     <ComboboxRoot :model-value="tags" v-model:open="open" v-model:search-term="searchTerm" class="w-full">
       <ComboboxAnchor as-child>
         <ComboboxInput :placeholder="placeholder" as-child>
           <TagsInputInput class="w-full px-3" :class="tags.length > 0 ? 'mt-2' : ''" @keydown.enter.prevent />
         </ComboboxInput>
       </ComboboxAnchor>
       <ComboboxPortal>
         <ComboboxContent>
           <CommandList position="popper"
             class="w-[--radix-popper-anchor-width] rounded-md mt-2 border bg-popover text-popover-foreground shadow-md outline-none data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2">
             <CommandEmpty />
             <CommandGroup>
               <CommandItem v-for="item in filteredOptions" :key="item" :value="item" @select="handleSelect">
                 {{ item }}
               </CommandItem>
             </CommandGroup>
           </CommandList>
         </ComboboxContent>
       </ComboboxPortal>
     </ComboboxRoot>
   </TagsInput>
 </template>
 
 <script setup>
 import { CommandEmpty, CommandGroup, CommandItem, CommandList } from '@/components/ui/command'
 import { TagsInput, TagsInputInput, TagsInputItem, TagsInputItemDelete, TagsInputItemText } from '@/components/ui/tags-input'
 import { ComboboxAnchor, ComboboxContent, ComboboxInput, ComboboxPortal, ComboboxRoot } from 'radix-vue'
 import { computed, ref } from 'vue'
 
 const tags = defineModel({
   required: true,
   default: () => []
 })
 
 const props = defineProps({
   placeholder: {
     type: String,
     default: 'Select...'
   },
   items: {
     type: Array,
     required: true
   }
 })
 
 const open = ref(false)
 const searchTerm = ref('')
 
 const filteredOptions = computed(() =>
   props.items.filter(item => !tags.value.includes(item))
 )
 
 const handleSelect = (event) => {
   if (event.detail.value) {
     searchTerm.value = ''
     const newTags = Array.isArray(tags.value) ? [...tags.value] : []
     newTags.push(event.detail.value)
     tags.value = newTags
   }
   if (filteredOptions.value.length === 0) {
     open.value = false
   }
 }
 </script>