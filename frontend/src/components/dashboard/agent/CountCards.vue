<template>
    <div class="flex mt-5 gap-x-7">
        <Card class="w-1/6" v-for="(value, key) in counts" :key="key">
            <CardHeader>
                <CardTitle class="text-4xl">
                    {{ value }}
                </CardTitle>
                <CardDescription>
                    {{ labels[key] }}
                </CardDescription>
            </CardHeader>
        </Card>
    </div>
</template>
<script setup>
import {
    Card,
    CardDescription,
    CardHeader,
    CardTitle,
} from '@/components/ui/card'


defineProps({
    counts: {
        type: Object,
        required: true,
    },
    labels: {
        type: Object,
        required: true,
    }
})
</script>