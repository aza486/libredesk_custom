<template>
  <div class="min-h-screen flex flex-col w-full">
    <div class="flex flex-wrap gap-4 pb-4">
      <div class="flex items-center gap-4 mb-4">
        <!-- Filter Popover -->
        <Popover>
          <PopoverTrigger>
            <ListFilter size="18" class="text-muted-foreground cursor-pointer" />
          </PopoverTrigger>
          <PopoverContent class="w-full p-4 flex flex-col gap-4">
            <FilterBuilder
              :fields="filterFields"
              :showButtons="true"
              v-model="filters"
              @apply="fetchActivityLogs"
            />
          </PopoverContent>
        </Popover>

        <!-- Order By Popover -->
        <Popover>
          <PopoverTrigger>
            <ArrowDownUp size="18" class="text-muted-foreground cursor-pointer" />
          </PopoverTrigger>
          <PopoverContent class="w-[200px] p-4 flex flex-col gap-4">
            <!-- order by field -->
            <Select v-model="orderByField" @update:model-value="fetchActivityLogs">
              <SelectTrigger class="h-8 w-full">
                <SelectValue :placeholder="orderByField" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem :value="'activity_logs.created_at'">
                  {{ t('form.field.createdAt') }}
                </SelectItem>
              </SelectContent>
            </Select>

            <!-- order by direction -->
            <Select v-model="orderByDirection" @update:model-value="fetchActivityLogs">
              <SelectTrigger class="h-8 w-full">
                <SelectValue :placeholder="orderByDirection" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem :value="'asc'">Ascending</SelectItem>
                <SelectItem :value="'desc'">Descending</SelectItem>
              </SelectContent>
            </Select>
          </PopoverContent>
        </Popover>
      </div>

      <div v-if="loading" class="flex flex-col gap-4 w-full">
        <Card v-for="i in perPage" :key="i" class="p-4 flex-shrink-0">
          <div class="flex items-center gap-4">
            <Skeleton class="h-10 w-10 rounded-full" />
            <div class="space-y-2 w-full">
              <Skeleton class="h-3 w-[160px]" />
              <Skeleton class="h-3 w-[140px]" />
            </div>
          </div>
        </Card>
      </div>

      <template v-else>
        <div class="w-full overflow-x-auto">
          <SimpleTable
            :headers="[t('form.field.name'), t('form.field.date'), t('globals.terms.ipAddress')]"
            :keys="['activity_description', 'created_at', 'ip']"
            :data="activityLogs"
            :showDelete="false"
          />
        </div>
        <div v-if="activityLogs.length === 0" class="flex items-center justify-center w-full h-32">
          <p class="text-lg text-muted-foreground">{{ t('globals.states.noResults') }}</p>
        </div>
      </template>
    </div>

    <div class="sticky bottom-0 bg-background p-4 mt-auto">
      <div class="flex flex-col sm:flex-row items-center justify-between gap-4">
        <div class="flex items-center gap-2">
          <span class="text-sm text-muted-foreground">
            {{ t('globals.terms.page') }} {{ page }} of {{ totalPages }}
          </span>
          <Select v-model="perPage" @update:model-value="handlePerPageChange">
            <SelectTrigger class="h-8 w-[70px]">
              <SelectValue :placeholder="perPage" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem :value="10">10</SelectItem>
              <SelectItem :value="15">15</SelectItem>
              <SelectItem :value="30">30</SelectItem>
              <SelectItem :value="50">50</SelectItem>
              <SelectItem :value="100">100</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <Pagination>
          <PaginationList class="flex items-center gap-1">
            <PaginationListItem>
              <PaginationFirst
                :class="{ 'cursor-not-allowed opacity-50': page === 1 }"
                @click.prevent="page > 1 ? goToPage(1) : null"
              />
            </PaginationListItem>
            <PaginationListItem>
              <PaginationPrev
                :class="{ 'cursor-not-allowed opacity-50': page === 1 }"
                @click.prevent="page > 1 ? goToPage(page - 1) : null"
              />
            </PaginationListItem>
            <template v-for="pageNumber in visiblePages" :key="pageNumber">
              <PaginationListItem v-if="pageNumber === '...'">
                <PaginationEllipsis />
              </PaginationListItem>
              <PaginationListItem v-else>
                <Button
                  :is-active="pageNumber === page"
                  @click.prevent="goToPage(pageNumber)"
                  :variant="pageNumber === page ? 'default' : 'outline'"
                >
                  {{ pageNumber }}
                </Button>
              </PaginationListItem>
            </template>
            <PaginationListItem>
              <PaginationNext
                :class="{ 'cursor-not-allowed opacity-50': page === totalPages }"
                @click.prevent="page < totalPages ? goToPage(page + 1) : null"
              />
            </PaginationListItem>
            <PaginationListItem>
              <PaginationLast
                :class="{ 'cursor-not-allowed opacity-50': page === totalPages }"
                @click.prevent="page < totalPages ? goToPage(totalPages) : null"
              />
            </PaginationListItem>
          </PaginationList>
        </Pagination>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, h, watch } from 'vue'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import SimpleTable from '@/components/table/SimpleTable.vue'
import {
  Pagination,
  PaginationEllipsis,
  PaginationFirst,
  PaginationLast,
  PaginationList,
  PaginationListItem,
  PaginationNext,
  PaginationPrev
} from '@/components/ui/pagination'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select'
import FilterBuilder from '@/features/filter/FilterBuilder.vue'
import { Button } from '@/components/ui/button'
import { ArrowDownUp, ListFilter } from 'lucide-vue-next'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import api from '@/api'
import { useActivityLogFilters } from '@/composables/useActivityLogFilters'
import { useI18n } from 'vue-i18n'
import { format } from 'date-fns'
import { getVisiblePages } from '@/utils/pagination'


const activityLogs = ref([])
const { t } = useI18n()
const loading = ref(true)
const page = ref(1)
const perPage = ref(10)
const orderByField = ref('activity_logs.created_at')
const orderByDirection = ref('desc')
const totalCount = ref(0)
const totalPages = ref(0)
const filters = ref([])
const { activityLogListFilters } = useActivityLogFilters()

const filterFields = computed(() =>
  Object.entries(activityLogListFilters.value).map(([field, value]) => ({
    model: 'activity_logs',
    label: value.label,
    field,
    type: value.type,
    operators: value.operators,
    options: value.options ?? []
  }))
)

const visiblePages = computed(() => getVisiblePages(page.value, totalPages.value))

async function fetchActivityLogs() {
  loading.value = true
  try {
    const resp = await api.getActivityLogs({
      page: page.value,
      page_size: perPage.value,
      filters: JSON.stringify(filters.value),
      order: orderByDirection.value,
      order_by: orderByField.value
    })
    activityLogs.value = resp.data.data.results
    totalCount.value = resp.data.data.count
    totalPages.value = resp.data.data.total_pages

    // Format the created_at field
    activityLogs.value = activityLogs.value.map((log) => ({
      ...log,
      created_at: format(new Date(log.created_at), 'PPpp')
    }))
  } catch (err) {
    console.error('Error fetching activity logs:', err)
    activityLogs.value = []
    totalCount.value = 0
  } finally {
    loading.value = false
  }
}

function goToPage(p) {
  if (p >= 1 && p <= totalPages.value && p !== page.value) {
    page.value = p
  }
}

function handlePerPageChange() {
  page.value = 1
  fetchActivityLogs()
}

watch([page, perPage, orderByField, orderByDirection], fetchActivityLogs)

onMounted(fetchActivityLogs)
</script>
