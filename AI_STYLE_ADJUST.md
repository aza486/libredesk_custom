# KI-Notizen: Stil anpassen und Versandherkunft

Implementierung im bestehenden Arbeitsbaum auf `main`, Ausgangsstand `283794e75`.
Kein Branch, Worktree, Commit, Deployment oder n8n-Workflow wurde angelegt/geändert.
Diese Datei liegt bewusst außerhalb von `docs/`, da `docs/` hier ignoriert wird.

## Bedienung

Zwischen „KI-Antwort übernehmen“ und „Eigene Antwort“ erscheint „Stil anpassen“
für validierte versionierte KI-Notizen in E-Mail-Konversationen mit `messages:write`.
Der Vorschlag muss für diesen Endpunkt höchstens 8000 Unicode-Codepoints lang sein.
Bei einer bereits geladenen öffentlichen, eingereihten/gesendeten Antwort mit derselben
Quellzuordnung wird der Button ausgeblendet. Der Server prüft die Zuordnung erneut.
Fehlgeschlagene Antworten sperren eine erneute Anpassung nicht.

Der Dialog verwendet die bestehenden Dialog-, Select-, Textarea-, Alert- und
Button-Komponenten, Theme-Farben und responsive Scrollbereiche. ESC/Schließen/
Verwerfen brechen laufende Browser-Anfragen ab; verspätete Ergebnisse werden durch
eine Generationsprüfung verworfen. n8n kann eine bereits laufende Textberechnung
noch abschließen, ohne dass daraus ein Versand oder eine Speicherung entsteht.

Presets: `mine`, `friendly`, `professional`, `empathetic`, `short`, `neutral`.
Zusätzliche Anweisungen: höchstens 1000 Unicode-Codepoints. Zähler und Sperre prüfen
auch eingefügten Text. Fehler behalten Eingabe und bisherigen Verlauf bei.
Jede weitere Anpassung verwendet den aktuellen Text; das Original bleibt unverändert.
Original/Angepasst zeigen Plaintext mit Zeilenumbrüchen. Rückgängig entfernt genau
eine Version; Original wiederherstellen leert den Verlauf. Nur eine angepasste
Version kann übernommen werden. Die anschließende vorhandene Übernahmebestätigung
zeigt den tatsächlich einzufügenden Text; erst Bestätigen ersetzt den öffentlichen
Composer. Kein automatischer Versand.

## Style-API und Konfiguration

`POST /api/v1/conversations/{cuuid}/ai-style-adjust`, Mitarbeiter-Session und
`messages:write`. Zusätzlich: aktiver Agent, Konversationszugriff, E-Mail-Inbox,
private ausgehende nicht gelöschte versionierte KI-Notiz derselben Konversation,
exakte Übereinstimmung von `original_text` mit deren `suggested_message` sowie
öffentliche eingehende Kontakt-Quellnachricht. Allgemeine Nachrichtenmetadaten
werden weiterhin nicht schreibbar gemacht. Fremde Requestfelder werden abgelehnt.

Browser-Beispiel (bestehende Session/CSRF-Verarbeitung des API-Clients):

```http
POST /api/v1/conversations/<conversation_uuid>/ai-style-adjust
Content-Type: application/json

{
  "note_uuid": "<private_note_uuid>",
  "style": "professional",
  "instructions": "Bitte zwei Absätze.",
  "original_text": "Hallo, hier ist der ursprüngliche Vorschlag.",
  "current_text": "Hallo, hier ist der zuletzt angepasste Vorschlag."
}
```

Go sendet ausschließlich `request_id` (neu erzeugte UUID), `agent_id` (Session-ID
als String), `style`, `instructions`, `original_text`, `current_text` an n8n.
Original und aktueller Text: jeweils nicht leer und maximal 8000 Unicode-Codepoints.
Originale längerer KI-Notizen bleiben über die bisherigen Aktionen nutzbar.
Es werden keine Profilbeispiele oder Profildaten aus Libredesk übermittelt.

n8n-Antwort, HTTP 200, direkt ohne Libredesk-Envelope an Vue:

```json
{"success":true,"request_id":"<identische UUID>","adapted_text":"Guten Tag, …","profile_source":"agent"}
```

`profile_source`: `agent|fallback|preset`. Go prüft Request-ID, Erfolg,
Profilquelle und nichtleeren Ausgabetext bis 8000 Zeichen. Fehler nach erfolgreicher
Authentifizierung/Autorisierung verwenden `success=false`, `request_id` und
`error:{code,message}`. Bestehende Auth-/Berechtigungsfehler behalten das
Libredesk-Envelope. HTTP 400 für ungültige Eingabe, 409 für verwendete Vorschläge,
422 für von n8n verworfene Ausgabe, 503 für fehlende Konfiguration, 502 für ungültige
Antwort/Netzwerkfehler, 504 für Timeout. Keine ungeprüften n8n-Fehlertexte werden
an den Browser gegeben. Gesamtdauer maximal 45 Sekunden; Browsertimeout 50 Sekunden.

```toml
[ai_style]
endpoint = "http://n8n:5678/webhook/libredesk-style"
token = "" # aus Secret-Umgebung setzen
internal_http_endpoint = "http://n8n:5678/webhook/libredesk-style"
internal_http_ip = "172.20.0.5" # Beispiel: tatsächliche feste private n8n-IP einsetzen
```

Environment: `LIBREDESK_AI_STYLE__ENDPOINT`, `LIBREDESK_AI_STYLE__TOKEN`,
`LIBREDESK_AI_STYLE__INTERNAL_HTTP_ENDPOINT`, `LIBREDESK_AI_STYLE__INTERNAL_HTTP_IP`.
In n8n Header-Auth-Credential `X-Libredesk-Style-Token` mit demselben Secret verwenden.
Keine Browser-Credentials. Alle vier leeren Werte deaktivieren die Verbindung.
HTTPS ist der Standard; dann bleiben die beiden HTTP-Ausnahmefelder leer.
Für internes Docker-HTTP müssen Endpoint und Ausnahme-URL exakt übereinstimmen,
mit explizitem Port und einem einzelnen fest gepinnten privaten Ziel-IP.
Libredesk und n8n benötigen ein gemeinsames internes Docker-Netz. Die IP muss
stabil konfiguriert werden. Keine globalen SSRF-Ausnahmen und keine veröffentlichten
n8n-Ports erforderlich. Redirects und HTTP-Proxys sind deaktiviert.
Konfigurationsänderungen benötigen einen späteren regulären Anwendungsneustart.

## Provenienz getrennt vom Lernfeedback

Der Composer hält `ai_response` mit `response_source`, `source_message_uuid` und
`note_uuid`. Auswahl bestimmt die Herkunft:

- Original übernehmen: `ai`.
- Style-Ergebnis übernehmen: `ai_adapted`.
- Bisherige Aktion „Eigene Antwort“: `human` mit bestehender Feedback-Vormerkung.

Diese drei Metadaten bleiben im bestehenden mitarbeitergebundenen Server-Entwurf
erhalten. Keine Varianten, Originalkopien oder Style-Anweisungen werden persistiert;
der übernommene Text ist lediglich der normale bearbeitbare Antwortentwurf.
Normales Tippen ändert die explizit gewählte Herkunft nicht automatisch. Bei einer
vollständig eigenen Neufassung ist „Eigene Antwort“ zu wählen. Es gibt keine
inhaltliche Ähnlichkeitsheuristik und kein Lernfeedback aus bloßer Textbearbeitung.
Abbrechen einer menschlichen Feedback-Vormerkung verhindert deren Übermittlung;
ohne Vormerkung wird beim normalen Versand auch kein `human`-Provenienzereignis
mitgesendet. Bereits bestehende Entwürfe ohne Herkunft bleiben unverändert nutzbar.

Beim bestehenden Nachrichten-POST kommt ausschließlich das gezielte Feld hinzu:

```json
{"ai_response":{"response_source":"ai_adapted","source_message_uuid":"<customer_message_uuid>","note_uuid":"<private_note_uuid>"}}
```

Go prüft Session, öffentlichen Agentenversand per E-Mail, Quellnachricht und
KI-Notiz-Zuordnung serverseitig. `ai|ai_adapted` gemeinsam mit einem
`ai_feedback_original_message_uuid` wird abgelehnt; `human` muss dazu passen.
Die Herkunft ist eine Mitarbeiterangabe, kein kryptographischer Nachweis einer
LLM-Generierung. Der originale `suggested_message`-Wert wird nie geändert.

`QueueReply` und der bestehende Worker behalten ihren Versandpfad. Der gerenderte
Transporttext wird vor dem Versand als `ai_response_sent_body` in den vorhandenen
Nachrichtenmetadaten gesichert. Nach gespeichertem Status `sent` wird zusätzlich
`ai_response_sent_at` gesetzt. Erst dann ist das Ereignis zustellbar. Ein Fehler
beim notwendigen Snapshot stoppt vor dem Transport; eine fehlende oder defekte
Provenienz-Webhook-Anbindung blockiert dagegen weder Queue noch E-Mail-Versand.

## Separater Provenienz-Webhook

```toml
[ai_provenance]
endpoint = "http://n8n:5678/webhook/libredesk-provenance"
token = "" # aus Secret-Umgebung setzen
internal_http_endpoint = "http://n8n:5678/webhook/libredesk-provenance"
internal_http_ip = "172.20.0.5"
```

Environment analog: `LIBREDESK_AI_PROVENANCE__ENDPOINT`, `__TOKEN`,
`__INTERNAL_HTTP_ENDPOINT`, `__INTERNAL_HTTP_IP`. Header-Auth-Credential:
`X-Libredesk-Provenance-Token`. HTTPS-/Docker-Regeln wie oben.

Der unabhängige Backend-Prozess liest nur gespeicherte öffentliche Agentennachrichten
mit Status `sent` und Provenienz-Snapshot. Keine neue Tabelle oder Migration.
Er startet beim Backendstart und prüft nach jeder Runde alle 15 Sekunden erneut,
bis zu zehn Ereignisse pro Runde, höchstens zehn Sekunden je Webhook-Aufruf.
Ein PostgreSQL-Advisory-Lock verhindert parallele Runden mehrerer Instanzen.
Fehlerhafte Ereignisse werden nach Versuchszeit sortiert, damit neue Ereignisse
weiter verarbeitet werden. Bei großen Nachrichtenbeständen ist vor Aktivierung
die Abfrageleistung zu prüfen; es wird kein zusätzlicher Index automatisch angelegt.

```json
{
  "event_type": "outgoing_response_sent",
  "conversation_uuid": "<conversation_uuid>",
  "source_message_uuid": "<customer_message_uuid>",
  "outgoing_message_uuid": "<outgoing_uuid>",
  "response_source": "ai_adapted",
  "sent_message": "<tatsächlich gerenderter Transporttext, gegebenenfalls HTML>",
  "sent_at": "<persistierter Zeitstempel mit Zeitzone>"
}
```

`Idempotency-Key` ist immer die ausgehende UUID. n8n muss das Ereignis transaktional
anhand dieser UUID deduplizieren, die Quellnachricht eindeutig zuordnen und nur
`mail_events.response_source`/`sent_message` aktualisieren. Bei mehreren Antworten
auf dieselbe Quelle muss der Empfänger die gewünschte Reihenfolge anhand von
`sent_at`/ausgehender UUID festlegen. Erfolg erst nach Commit bestätigen:

```json
{"committed":true,"outgoing_message_uuid":"<identische outgoing_uuid>"}
```

HTTP 200 mit exakt passender Bestätigung setzt `ai_response_committed=true` in der
Libredesk-Nachricht. Sonst bleibt das Ereignis zur Wiederholung erhalten. Verlorene
Commit-Antworten oder Backendabbrüche können dasselbe Ereignis erneut ausliefern:
Empfänger-Idempotenz ist zwingend. Es gibt keinerlei Aufruf des E-Mail-Versands
oder Nachrichten-Retry aus diesem Hintergrundprozess. Nicht konfigurierte
Anbindung lässt Ereignisse bis zur späteren Aktivierung liegen; Aufbewahrung und
Löschen der zugrundeliegenden Nachrichten begrenzen diese Wiederaufnahme.
`sent` bedeutet Transportannahme; die vorhandene Lücke zwischen SMTP-Annahme und
lokalem Status-Commit bleibt bestehen.

`/ai-feedback` und `internal/aifeedback/feedback.go` bleiben unverändert. Stil-
Aktionen erzeugen keine Feedback-/RAG-/Lerndatensätze. Die n8n-Gegenstelle darf für
`ai` und `ai_adapted` ebenfalls keine solchen Datensätze erzeugen. Kein Zugriff auf
`pg-n8n`, keine Anpassung von Learning-Workflows durch diese Implementierung.

## Prüfung und manuelle Abnahme

- Frontend: 36 Vitest-Dateien, 736 Tests erfolgreich, einschließlich neuem Dialog,
  Verlauf, Fehler/Retry, spätem Ergebnis, Übernahmebestätigung und Versandherkunft.
- Gezieltes ESLint aller zehn betroffenen Frontend-Dateien erfolgreich.
- Go: `go test ./cmd ./internal/aiassist ./internal/conversation ./internal/aifeedback ./internal/authz/... -count=1` erfolgreich.
- `go vet` für dieselben Pakete erfolgreich.
- Main- und Widget-Vite-Build erfolgreich, Ausgaben ausschließlich unter
  `/tmp/libredesk-ai-style-main` und `/tmp/libredesk-ai-style-widget`.
- `git diff --check` erfolgreich. Keine externen Dependencies ergänzt.
- PostgreSQL-Integrationstest `TestAIProvenanceOnlyAfterSentAndIndependentRetry`
  ist vorhanden, aber wegen fehlender Test-DB auf `127.0.0.1:5433` übersprungen.
  Vor Aktivierung mit isoliertem `LIBREDESK_TEST_DB_DSN` ausführen, niemals Produktion.
  Er prüft pending/failed/sent, Snapshot, Retry, Commit-Markierung, Quellzuordnung
  und unveränderte Anzahl ausgehender Kundennachrichten.
- Kein produktiver End-to-End-Test. n8n ist noch einzurichten; ein tatsächliches
  Update von `pg-n8n.mail_events` wurde nicht ausgeführt oder bestätigt.

Manuell in einer Testumgebung:

1. Beide n8n-Testwebhooks mit Header-Auth und den oben dokumentierten Antworten
   einrichten. Style-Workflow muss Bedeutungsinhalt, Platzhalter, Nummern, Preise,
   Daten und URLs erhalten und unsichere Ergebnisse mit 422 verwerfen. Profilpflege
   und Anonymisierung eventueller Profilbeispiele bleiben Sache des separaten Workflows.
2. Versionierte KI-Notiz öffnen: alle sechs Presets, mehrfache Anpassung, Vergleich,
   Rückgängig, Reset und 1000-Zeichen-Grenze prüfen. HTML-artigen Text und Emoji testen.
3. Bestehenden Composer füllen. Übernehmen bis zur Bestätigung unverändert lassen,
   Bestätigung abbrechen beziehungsweise bestätigen. Vor Versand darf kein
   Feedback-/Provenienzereignis entstehen. Verwerfen/ESC und mobile/dunkle Ansicht prüfen.
4. Verzögerten/fehlgeschlagenen Style-Webhook testen, Dialog schließen und Ticket
   wechseln. Keine späten Ergebnisse in anderen Dialogen; Retry behält Eingaben.
5. Original beziehungsweise Anpassung übernehmen und öffentlich senden. Erst nach
   `sent` muss n8n `ai` beziehungsweise `ai_adapted` mit beiden Nachrichten-UUIDs erhalten.
6. Provenienzempfänger auf Fehler setzen beziehungsweise Commit-Antwort verlieren.
   Versand bleibt einmalig; Wiederholungen besitzen dieselbe ausgehende UUID.
   Nach passender Commit-Bestätigung dürfen weitere Runden kein Ereignis senden.
7. „Eigene Antwort“ samt Begründung und bestehendem Feedback-Retry prüfen. Private
   Notizen und normale Antworten ohne Herkunft bleiben beim bisherigen Verhalten.

## Dateien

Backend: `cmd/ai_style.go`, `cmd/ai_style_test.go`, `cmd/ai_provenance.go`,
`cmd/messages.go`, `cmd/handlers.go`, `cmd/main.go`, `config.sample.toml`,
`internal/aiassist/client.go`, `internal/aiassist/client_test.go`,
`internal/conversation/ai_provenance.go`, `internal/conversation/ai_provenance_test.go`,
`internal/conversation/message.go`, `internal/conversation/queries.sql`.

Frontend: `frontend/apps/main/src/api/index.js`,
`composables/useDraftManager.js`, `composables/useDraftManager.ai-feedback.test.js`,
`features/conversation/AIStyleDialog.vue`, `features/conversation/AIStyleDialog.test.js`,
`features/conversation/ReplyBox.vue`, `features/conversation/ReplyBox.ai-feedback.test.js`,
`features/conversation/message/MessageBubble.vue`, `utils/ai-style.js`,
`utils/ai-style.test.js` (alle relativen Frontend-Pfade unter
`frontend/apps/main/src/`). Dokumentation: diese Datei.
