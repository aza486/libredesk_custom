<template>
    <div class="box">
        <div class="flex flex-col">
            <h3>
                {{ title }}
            </h3>
            <p>
                {{ subTitle }}
            </p>
        </div>
    </div>
</template>

<script setup>
defineProps({
    title: {
        type: String,
        required: true,
    },
    subTitle: {
        type: String,
        required: true,
    }
})
</script>