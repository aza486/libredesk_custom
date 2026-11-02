<template>
  <AdminSplitLayout>
    <template #content>
      <p v-if="!review.allowed()" role="alert">Keine Berechtigung für KI-Lernverwaltung.</p>
      <template v-else>
        <div class="mb-4 flex gap-2">
          <Button
            v-for="entry in tabs"
            :key="entry.id"
            :variant="tab === entry.id ? 'default' : 'outline'"
            @click="tab = entry.id"
            >{{ entry.label }}</Button
          >
          <Button variant="outline" :disabled="loading || busy" @click="review.refresh"
            >Aktualisieren</Button
          >
        </div>
        <p v-if="error" role="alert" class="mb-3 text-destructive">{{ error }}</p>
        <LoadingOverlay :loading="loading" reserve-height>
          <div class="space-y-4">
            <p v-if="!items.length && !loading" class="text-muted-foreground">
              Keine passenden Einträge gefunden.
            </p>
            <article
              v-for="item in items"
              :key="item.id"
              class="space-y-3 rounded-lg border bg-card p-4"
            >
              <div class="flex flex-wrap items-center justify-between gap-2">
                <h2 class="font-semibold">
                  {{ item.content.topic || typeLabel(item.content.source_type) }}
                </h2>
                <span class="text-xs text-muted-foreground">{{
                  formatLearningDate(item.created_at)
                }}</span>
              </div>
              <p class="text-sm text-muted-foreground">
                {{ categoryLabel(item.content.category) }} · {{ statusSummary(item).review }} ·
                {{ statusSummary(item).publication }}
              </p>
              <p class="whitespace-pre-wrap break-words text-sm">
                {{ item.content.rule || item.content.human_answer }}
              </p>
              <Button variant="outline" :disabled="busy" @click="review.open(item)">{{
                tab === 'pending' ? 'Vorschlag prüfen' : 'Ansehen'
              }}</Button>
            </article>
          </div>
        </LoadingOverlay>
        <div class="mt-4 flex items-center gap-3">
          <Button variant="outline" :disabled="page <= 1 || loading || busy" @click="page--"
            >Zurück</Button
          >
          <span class="text-sm">Seite {{ page }}</span>
          <Button variant="outline" :disabled="!hasMore || loading || busy" @click="page++"
            >Weiter</Button
          >
        </div>
        <Dialog :open="!!selected" @update:open="closeDialog">
          <DialogContent class="max-h-[90vh] overflow-y-auto sm:max-w-3xl">
            <DialogHeader>
              <DialogTitle>Lernvorschlag prüfen</DialogTitle>
              <DialogDescription
                >Prüfe die vorgeschlagene allgemeine Erkenntnis anhand des Kontexts. Deine
                Entscheidung gilt für diesen gespeicherten Vorschlag.</DialogDescription
              >
            </DialogHeader>
            <template v-if="selected">
              <section class="space-y-2">
                <h3 class="font-semibold">Was hat die KI erkannt?</h3>
                <p class="whitespace-pre-wrap break-words text-sm">
                  {{
                    selected.rationale ||
                    'Die vorhandene Erkenntnis wurde aus Mitarbeiterfeedback abgeleitet.'
                  }}
                </p>
              </section>
              <section class="space-y-2 rounded-md border p-3">
                <h3 class="font-semibold">Was möchte die KI lernen?</h3>
                <p class="text-sm text-muted-foreground">
                  {{ typeLabel(selected.content.source_type) }} ·
                  {{ categoryLabel(selected.content.category) }}
                </p>
                <p class="whitespace-pre-wrap break-words">
                  {{ selected.content.rule || selected.content.human_answer }}
                </p>
                <dl class="space-y-2 text-sm">
                  <template v-for="field in proposalFields" :key="field.key">
                    <template v-if="selected.content[field.key]"
                      ><dt class="font-medium">{{ field.label }}</dt>
                      <dd class="whitespace-pre-wrap break-words">
                        {{
                          field.key === 'source'
                            ? sourceLabel(selected.content[field.key])
                            : selected.content[field.key]
                        }}
                      </dd></template
                    >
                  </template>
                </dl>
              </section>
              <section class="space-y-2">
                <h3 class="font-semibold">Warum muss das geprüft werden?</h3>
                <p class="whitespace-pre-wrap break-words text-sm">
                  {{
                    (selected.block_reason ? reasonLabel(selected.block_reason) : '') ||
                    'Neue allgemeine Firmeninformation benötigt eine unabhängige fachliche Prüfung.'
                  }}
                </p>
              </section>
              <details class="space-y-3 text-sm">
                <summary>Kontext</summary>
                <dl class="space-y-2">
                  <dt class="font-medium">Ursprüngliche Kundenanfrage</dt>
                  <dd class="whitespace-pre-wrap break-words">
                    {{ selected.context?.customer_question || 'Originalkontext nicht verfügbar.' }}
                  </dd>
                  <dt class="font-medium">Tatsächlich versendete Mitarbeiterantwort</dt>
                  <dd class="whitespace-pre-wrap break-words">
                    {{ selected.context?.human_answer || 'Antwort nicht verfügbar.' }}
                  </dd>
                  <dt class="font-medium">Ursprüngliche Kategorien</dt>
                  <dd>
                    {{
                      (selected.content.original_categories || []).map(categoryLabel).join(', ') ||
                      'Nicht zugeordnet'
                    }}
                  </dd>
                  <dt class="font-medium">Feedbackgrund</dt>
                  <dd class="whitespace-pre-wrap break-words">
                    {{ selected.feedback_reason || 'Kein zusätzlicher Feedbackgrund.' }}
                  </dd>
                  <dt class="font-medium">KI-Vorschlag</dt>
                  <dd class="whitespace-pre-wrap break-words">
                    {{ selected.ai_suggestion || 'Kein KI-Vorschlag vorhanden.' }}
                  </dd>
                </dl>
              </details>
              <details v-if="selected.original" class="space-y-2 text-sm">
                <summary>Ursprünglicher Lernvorschlag</summary>
                <p class="whitespace-pre-wrap break-words">
                  {{ selected.original.rule || selected.original.human_answer }}
                </p>
                <p class="whitespace-pre-wrap break-words">{{ selected.original.evidence }}</p>
                <p class="whitespace-pre-wrap break-words">{{ selected.original.limits }}</p>
              </details>
              <details v-if="selected.history?.length" class="space-y-2 text-sm">
                <summary>Prüfverlauf</summary>
                <article
                  v-for="entry in selected.history"
                  :key="entry.revision"
                  class="space-y-1 border-l pl-3"
                >
                  <p>
                    {{ auditActionLabel(entry.action) }} ·
                    {{ formatLearningDate(entry.created_at) }}
                  </p>
                  <p class="whitespace-pre-wrap break-words">{{ reasonLabel(entry.reason) }}</p>
                </article>
              </details>
              <p v-if="error" role="alert" class="text-destructive">{{ error }}</p>
              <div
                v-if="selected.review_ready && selected.decision === 'pending' && !selected.active"
                class="flex gap-2"
              >
                <Button
                  class="bg-green-600 text-white hover:bg-green-700"
                  :disabled="busy"
                  :is-loading="busy"
                  @click="perform('learn')"
                  >Lernen</Button
                >
                <Button variant="outline" :disabled="busy" @click="perform('discard')"
                  >Verwerfen</Button
                >
              </div>
              <p v-else class="text-sm text-muted-foreground">
                Keine offene Entscheidung. Unvollständige Altfälle werden nicht automatisch
                freigegeben.
              </p>
            </template>
          </DialogContent>
        </Dialog>
      </template>
    </template>
    <template #help
      ><p>
        Sichere Stil- und Kategoriekorrekturen werden automatisch verarbeitet. Neue allgemeine
        Fakten benötigen deine Prüfung. Lernen veröffentlicht erst nach erfolgreicher
        Datenschutzprüfung und gültigem Suchvektor.
      </p></template
    >
  </AdminSplitLayout>
</template>
<script setup>
import AdminSplitLayout from '@main/layouts/admin/AdminSplitLayout.vue'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import { Button } from '@shared-ui/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription
} from '@shared-ui/components/ui/dialog'
import { useUserStore } from '@main/stores/user'
import { useLearningReview } from '@main/features/admin/learning/useLearningReview'
import {
  categoryLabel,
  typeLabel,
  statusSummary,
  reasonLabel,
  formatLearningDate,
  auditActionLabel
} from '@main/features/admin/learning/learning-display'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
const review = useLearningReview(useUserStore())
const { tab, page, items, hasMore, loading, busy, error, selected } = review
const emitter = useEmitter()
const tabs = [
  { id: 'pending', label: 'Zur Prüfung' },
  { id: 'history', label: 'Verlauf' }
]
const sourceLabel = (value) =>
  ({ human_answer: 'Versendete Mitarbeiterantwort', feedback_reason: 'Feedbackgrund' })[value] ||
  value
const proposalFields = [
  { key: 'evidence', label: 'Beleg' },
  { key: 'source', label: 'Quelle' },
  { key: 'limits', label: 'Gültigkeit und Einschränkungen' }
]
const closeDialog = (value) => {
  if (!value && !busy.value) selected.value = null
}
const perform = async (action) => {
  if (await review.act(action))
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description:
        action === 'learn'
          ? 'Entscheidung gespeichert. Veröffentlichung folgt nach erfolgreicher technischer Prüfung.'
          : 'Vorschlag verworfen.'
    })
}
</script>
