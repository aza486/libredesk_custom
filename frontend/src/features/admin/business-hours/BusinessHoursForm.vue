<template>
    <form @submit="onSubmit" class="space-y-8">
        <FormField v-slot="{ componentField }" name="name">
            <FormItem>
                <FormLabel>Name</FormLabel>
                <FormControl>
                    <Input type="text" placeholder="General working hours" v-bind="componentField" />
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="description">
            <FormItem>
                <FormLabel>Description</FormLabel>
                <FormControl>
                    <Input type="text" placeholder="General working hours for my company" v-bind="componentField" />
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="is_always_open">
            <FormItem>
            <FormLabel>
                Set business hours
            </FormLabel>
            <FormControl>
                    <RadioGroup v-bind="componentField">
                        <div class="flex flex-col space-y-2">
                            <div class="flex items-center space-x-3">
                                <RadioGroupItem id="r1" value="true" />
                                <Label for="r1">Always open (24x7)</Label>
                            </div>
                            <div class="flex items-center space-x-3">
                                <RadioGroupItem id="r2" value="false" />
                                <Label for="r2">Custom business hours</Label>
                            </div>
                        </div>
                    </RadioGroup>
                </FormControl>
                <FormMessage />
            </FormItem>
        </FormField>

        <div v-if="form.values.is_always_open === 'false'">
            <div>
                <div v-for="day in WEEKDAYS" :key="day" class="flex items-center justify-between space-y-2">
                    <div class="flex items-center space-x-3">
                        <Checkbox :id="day" :checked="!!selectedDays[day]"
                            @update:checked="handleDayToggle(day, $event)" />
                        <Label :for="day" class="font-medium text-gray-800">{{ day }}</Label>
                    </div>
                    <div class="flex space-x-2 items-center">
                        <div class="flex flex-col items-start">
                            <Input type="time" :defaultValue="hours[day]?.open || '09:00'"
                                @update:modelValue="(val) => updateHours(day, 'open', val)"
                                :disabled="!selectedDays[day]" />
                        </div>
                        <span class="text-gray-500">to</span>
                        <div class="flex flex-col items-start">
                            <Input type="time" :defaultValue="hours[day]?.close || '17:00'"
                                @update:modelValue="(val) => updateHours(day, 'close', val)"
                                :disabled="!selectedDays[day]" />
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <Dialog >
            <div>
                <div class="flex justify-between items-center mb-4">
                    <div></div>
                    <DialogTrigger as-child>
                        <Button>New holiday</Button>
                    </DialogTrigger>
                </div>
            </div>
            <SimpleTable :headers="['Name', 'Date']" :keys="['name', 'date']" :data="holidays" @deleteItem="deleteHoliday" />
            <DialogContent class="sm:max-w-[425px]">
                <DialogHeader>
                    <DialogTitle>New holiday</DialogTitle>
                    <DialogDescription>
                    </DialogDescription>
                </DialogHeader>
                <div class="grid gap-4 py-4">
                    <div class="grid grid-cols-4 items-center gap-4">
                        <Label for="holiday_name" class="text-right">
                            Name
                        </Label>
                        <Input id="holiday_name" v-model="holidayName" class="col-span-3" />
                    </div>
                    <div class="grid grid-cols-4 items-center gap-4">
                        <Label for="date" class="text-right">
                            Date
                        </Label>
                        <Popover>
                            <PopoverTrigger as-child>
                                <Button variant="outline" :class="cn(
                                    'w-[280px] justify-start text-left font-normal',
                                    !holidayDate && 'text-muted-foreground',
                                )">
                                    <CalendarIcon class="mr-2 h-4 w-4" />
                                    {{ holidayDate && !isNaN(new Date(holidayDate).getTime()) ? format(new
                                        Date(holidayDate), 'MMMM dd, yyyy') : "Pick a date" }}
                                </Button>
                            </PopoverTrigger>
                            <PopoverContent class="w-auto p-0">
                                <Calendar v-model="holidayDate" />
                            </PopoverContent>
                        </Popover>
                    </div>
                </div>
                <DialogFooter>
                    <Button  :disabled="!holidayName || !holidayDate"
                        @click="saveHoliday">
                        Save changes
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
        <Button type="submit" :disabled="isLoading">{{ submitLabel }}</Button>
    </form>
</template>

<script setup>
import { ref, watch, reactive } from 'vue'
import { Button } from '@/components/ui/button'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { formSchema } from './formSchema.js'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Calendar } from '@/components/ui/calendar'
import { Input } from '@/components/ui/input'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { cn } from '@/lib/utils'
import { format } from 'date-fns'
import { WEEKDAYS } from '@/constants/date'
import { Calendar as CalendarIcon } from 'lucide-vue-next'
import SimpleTable from '@/components/table/SimpleTable.vue'
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from '@/components/ui/dialog'

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
        default: () => 'Save'
    },
    isNewForm: {
        type: Boolean
    },
    isLoading: {
        type: Boolean,
        required: false
    },
})

let holidays = reactive([])
const holidayName = ref('')
const holidayDate = ref(null)
const selectedDays = ref({})
const hours = ref({})


const form = useForm({
    validationSchema: toTypedSchema(formSchema),
    initialValues: props.initialValues
})


const saveHoliday = () => {
    holidays.push({
        name: holidayName.value,
        date: new Date(holidayDate.value).toISOString().split('T')[0]
    })
    holidayName.value = ''
    holidayDate.value = null
}

const deleteHoliday = (item) => {
    holidays.splice(holidays.findIndex(h => h.name === item.name), 1)
}

const handleDayToggle = (day, checked) => {
    selectedDays.value = {
        ...selectedDays.value,
        [day]: checked
    }

    if (checked && !hours.value[day]) {
        hours.value[day] = { open: '09:00', close: '17:00' }
    } else if (!checked) {
        const newHours = { ...hours.value }
        delete newHours[day]
        hours.value = newHours
    }
}

const updateHours = (day, type, value) => {
    if (!hours.value[day]) {
        hours.value[day] = { open: '09:00', close: '17:00' }
    }
    hours.value[day][type] = value
}


const onSubmit = form.handleSubmit((values) => {
    values.is_always_open = values.is_always_open === 'true'
    const businessHours = values.is_always_open === true
        ? {}
        :
        Object.keys(selectedDays.value)
            .filter(day => selectedDays.value[day])
            .reduce((acc, day) => {
                acc[day] = hours.value[day]
                return acc
            }, {})
    const finalValues = {
        ...values,
        hours: businessHours,
        holidays: holidays
    }
    props.submitForm(finalValues)
})

// Watch for initial values
watch(
    () => props.initialValues,
    (newValues) => {
        if (!newValues || Object.keys(newValues).length === 0) {
            return
        }
        // Set business hours if provided
        newValues.is_always_open = newValues.is_always_open.toString()
        if (newValues.is_always_open === 'false') {
            hours.value = newValues.hours || {}
            selectedDays.value = Object.keys(hours.value).reduce((acc, day) => {
                acc[day] = true
                return acc
            }, {})
        }
        // Set other form values
        form.setValues(newValues)
        holidays.length = 0
        holidays.push(...(newValues.holidays || []))
    },
    { deep: true }
)

</script>