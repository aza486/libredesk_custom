<template>
    <form>
        <FormField v-slot="{ componentField }" name="name">
            <FormItem>
                <FormLabel>{{$t('form.field.name')}}</FormLabel>
                <FormControl>
                    <Input type="text" placeholder="billing" v-bind="componentField" />
                </FormControl>
                <FormDescription></FormDescription>
                <FormMessage />
            </FormItem>
        </FormField>
        <!-- Form submit button slot -->
        <slot name="footer" ></slot>
    </form>
</template>

<script setup>
import {
    FormControl,
    FormDescription,
    FormField,
    FormItem,
    FormLabel,
    FormMessage
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
</script>