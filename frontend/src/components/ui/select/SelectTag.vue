<template>
    <div ref="dropdownRef">
        <TagsInput v-model="selectedItems" class="px-0 gap-0 shadow-sm">
            <div class="flex gap-2 flex-wrap items-center px-3">
                <TagsInputItem v-for="item in selectedItems" :key="item" :value="item">
                    <TagsInputItemText>{{ item }}</TagsInputItemText>
                    <TagsInputItemDelete />
                </TagsInputItem>
            </div>

            <ComboboxRoot v-model:open="isOpen" class="w-full">
                <ComboboxAnchor as-child>
                    <ComboboxInput :placeholder="placeHolder" as-child>
                        <TagsInputInput class="w-full px-3" :class="selectedItems.length > 0 ? 'mt-2' : ''"
                            @keydown.enter.prevent />
                    </ComboboxInput>
                </ComboboxAnchor>

                <ComboboxPortal>
                    <CommandList position="popper"
                        class="w-[--radix-popper-anchor-width] rounded-md mt-2 border bg-popover text-popover-foreground shadow-md outline-none data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2">
                        <CommandEmpty />
                        <CommandGroup>
                            <CommandItem v-for="item in filteredItems" :key="item" :value="item"
                                @select.prevent="selectItem(item)">
                                {{ item }}
                            </CommandItem>
                        </CommandGroup>
                    </CommandList>
                </ComboboxPortal>
            </ComboboxRoot>
        </TagsInput>
    </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { onClickOutside } from '@vueuse/core';
import { ComboboxAnchor, ComboboxInput, ComboboxPortal, ComboboxRoot } from 'radix-vue';
import { CommandEmpty, CommandGroup, CommandItem, CommandList } from '@/components/ui/command';
import { TagsInput, TagsInputInput, TagsInputItem, TagsInputItemDelete, TagsInputItemText } from '@/components/ui/tags-input';

const props = defineProps({
    items: {
        type: Array,
        required: true,
        default: () => []
    },
    placeHolder: {
        type: String,
        required: false,
        default: () => ''
    },
    initialValue: {
        type: Array,
        default: () => []
    }
});

const selectedItems = defineModel({ default: [] });
const isOpen = ref(false);
const searchTerm = ref('');
const dropdownRef = ref(null);

onClickOutside(dropdownRef, () => {
    isOpen.value = false;
});

onMounted(() => {
    selectedItems.value = props.initialValue
});

const filteredItems = computed(() => {
    if (searchTerm.value) {
        return props.items.filter(
            item =>
                item.toLowerCase().includes(searchTerm.value.toLowerCase()) &&
                !selectedItems.value.includes(item)
        );
    }
    return props.items.filter(item => !selectedItems.value.includes(item));
});

function selectItem (item) {
    if (!selectedItems.value.includes(item)) {
        selectedItems.value.push(item);
    }
    if (filteredItems.value.length === 0) {
        isOpen.value = false;
    }
}
</script>
