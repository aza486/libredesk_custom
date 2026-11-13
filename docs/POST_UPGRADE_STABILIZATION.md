# Post-Upgrade-Stabilisierung

Stand: 12.11.2026. Grundlage ist `DEPENDENCY_UPGRADE_REPORT.md`; dessen Dependency-Upgrade wurde erhalten. Einzige nachträgliche Versionskorrektur: `github.com/knadh/smtppool` 1.3.0 → 1.2.0 wegen eines real nachgewiesenen SMTP-Kompatibilitätsbruchs. Die npm-Versionen und das pnpm-Lockfile bleiben unverändert. Kein Commit, Push, Produktivdeployment, Produktionsdatenbankzugriff, Neustart produktiver Container oder Eingriff in n8n. Der Checkout enthielt bereits das uncommittete Upgrade. Die unten aufgeführten Stabilisierungsergebnisse beziehen sich auf diesen Stand.

## A. ESLint

Ausgangslauf: **287 Fehler**, vorher laut Upgradebericht 285. Nach der Bereinigung: **0 Fehler** im vollständigen `pnpm lint`.

| Regel                             | Vorher | Ursache und Behandlung                                                                                                                                                                                                                                                                                                                           |
| --------------------------------- | -----: | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `no-unused-vars`                  |    157 | Überwiegend absichtlich entfernte Objektfelder beim Rest-Destructuring in Formularfixtures und weitergereichten Props: `ignoreRestSiblings: true`. Tatsächlich ungenutzte Catch-/Callback-Bindungen entfernt; Catch-Behandlung, `useI18n()` und `defineEmits()` bleiben erhalten. Unbenutzte Arraypositionen durch Destructuring-Lücken ersetzt. |
| `cypress/unsafe-to-chain-command` |     75 | Alte Cypress-Konventionen: nach `clear`, `click`, `focus`, `scrollIntoView` das Ziel neu abfragen. Keine Assertions entfernt, keine Regel abgeschaltet.                                                                                                                                                                                          |
| `vue/multi-word-component-names`  |     43 | Etablierte Root-, Routen- und UI-Wrapper-Identitäten. Ausnahme nur für die 43 konkret aufgeführten Dateien in `eslint.config.js`, keine Verzeichnisfreigabe oder globale Abschaltung. Imports, Routen und öffentliche Namen unverändert.                                                                                                         |
| `no-useless-escape`               |      4 | Livechat-Snippets: gemeinsames `'</' + 'script>'` vermeidet das SFC-Endtag im Quelltext; generiertes HTML bleibt gleich.                                                                                                                                                                                                                         |
| `no-undef`                        |      3 | Echter bestehender Widgetfehler: `syncMissedMessages(attempt = 0)` initialisiert den bisher undefinierten Wiederholungszähler. Zwei neue Tests prüfen Wiederholungsgrenze und Zeitabstand; vorhandene Tests bleiben erhalten.                                                                                                                    |
| `vue/no-ref-as-operand`           |      2 | `ActionBox`/`RuleBox` geben absichtlich den Ref an den vorhandenen Empfänger weiter; dieser liest `.value`. Je eine kommentierte Ausnahme direkt am Emit, kein geänderter Eventvertrag.                                                                                                                                                          |
| `no-useless-assignment`           |      2 | Überflüssige Initialwerte in ReplyBox und Widget-Geschäftszeiten entfernt. Alle Pfade setzen die Variable vor Verwendung.                                                                                                                                                                                                                        |
| `cypress/no-unnecessary-waiting`  |      1 | Eng begrenzte Ausnahme für begrenztes MailHog-Polling: externer SMTP-Eingang besitzt kein DOM-/Browserereignis. Dokumentation direkt am `cy.wait(1000)`.                                                                                                                                                                                         |

Die Vue-Ausnahmeliste ist selbst die vollständige Liste der 43 Ausnahmen: bestehende Root-/Admin-Views und die importierten shared-ui-Wrapper. Die Regel bleibt für neue Dateien aktiv. Browser-, Node-, Vitest- und Cypress-Globals bleiben dateibezogen; Fehler- und Sicherheitsregeln wurden nicht pauschal abgeschaltet. Die Altbestände stammen überwiegend aus alten Testkonventionen und kosmetischen Namensregeln; der Widget-Zähler ist ein echter Programmierfehler. Nicht jede Altmeldung wurde durch das Dependency-Upgrade verursacht.

Zusätzlich bei realen Browserprüfungen gefunden: `App.vue` leitete einen Admin bei einem vollständigen `/inboxes/all/...`-Aufruf bereits vor dem asynchron geladenen Benutzerprofil um. `useAllInboxRouteGuard` wartet jetzt auf die authentifizierte Benutzer-ID und prüft Route und Admin-Rolle erneut, auch bei Rollenentzug. Vier Regressionstests sichern Admin-Deep-Link, Agent-Weiterleitung, Rollenentzug und andere Inbox-Routen. Die serverseitige Autorisierung bleibt maßgeblich.

## B. CI

Zentrale Versionen bleiben Node **26.13.0** aus `.node-version`, pnpm **12.18.0** aus `frontend/package.json` und Go **1.27.4** aus `go.mod`. `make build` installiert frozen, baut Main und Widget und packt die Assets in das Go-Binary. Der normale Lintbefehl ist lokal und in CI identisch und mutiert keine Dateien.

In dieser Runde geändert:

- `.github/workflows/frontend-ci.yml`: Ubuntu 24.04 explizit statt beweglichem `ubuntu-latest`, passende GTK-/ALSA-t64-Pakete, MailHog auf den verwendeten 1.0.1-Tag gepinnt, `pnpm exec cypress verify` vor Tests, Prüfung auf unveränderte Dependency-Dateien nach dem Build. Paketliste nach [offiziellen Cypress-Voraussetzungen](https://docs.cypress.io/app/get-started/install-cypress#linux-prerequisites).
- `.github/workflows/go.yml`: `go mod tidy -diff` und vollständiges `go vet ./...` als verbindliche Schritte. Zwei bestehende Copylock-Warnungen im My-Inbox-Test durch separat initialisierte Manager behoben; keine Analyzer deaktiviert.

Die weiteren Workflows (Build-Cache, Release, Crowdin, Issues, i18n-Guard, PR-Welcome) wurden gegen die vorhandenen Toolchain-Pins geprüft; deren vorausgehende Upgradeänderungen bleiben erhalten. Keine automatischen Dependency-Updates, `continue-on-error` oder neuen Skips ergänzt. Die GitHub-hosted Workflows selbst wurden hier nicht remote gestartet; lokale Prüfungen ersetzen keine tatsächlich ausgeführte Actions-Pipeline.

### Workflow-Verträge

`TestServiceWorkflowRegistryQueryIntegration` verwendet die bereits versionierte kleine SQL-Fixture `internal/conversation/testdata/service_workflow_registry_query.sql`. Deren echter PostgreSQL-Lauf besteht. Die bewusst entfernte Frontenddatei `serviceMailWorkflow.test.js` bleibt entfernt. Keine produktiven n8n-Exporte hinzugefügt.

Die optionale lokale Exportvergleichsprüfung bleibt optional nur beim fehlenden Standardpfad; ein ausdrücklich gesetzter ungültiger Pfad oder ein Query-Mismatch bleibt ein Fehler. Verpflichtende CI-Abdeckung hängt damit nicht vom lokalen Service-Mail-Export ab.

**Learning-Blocker:** `AI_Learning_Processor_v9.json` und `AI_Learning_Processor_v10.json` fehlen auch in der verfügbaren Git-Historie. Die SQL-Dateien unter `internal/learning/artifacts/` und die Dokumentation enthalten nicht die vollständigen originalen Claim-/Finalize-/SaveDraft-Abfragen oder Workflow-JavaScript-Schritte. Die Tests prüfen neben Go-Verhalten tatsächlich diese Workflow-Verträge: Feedback-Zeilenlocks, atomare Claim-/Publish-Übergänge, Idempotenz, konkurrierende Verarbeitung, manuellen Faktenschutz, begrenzte Rollenrechte und Embedding-/Staging-Schutz. Aus Assertions allein lässt sich keine unabhängige, vertrauenswürdige Implementierungsfixture rekonstruieren.

Deshalb wurden weder erfundene Exporte noch abgeschwächte Tests erstellt. Mit ausdrücklich aktiviertem isoliertem Learning-PostgreSQL bestehen **63 Tests**, genau **2 scheitern** beim Öffnen der beiden Dateien. Der separate Node-Vertragstest `internal/learning/artifacts/workflow.test.mjs` wurde ebenfalls ausgeführt und scheitert mit ENOENT bereits beim Import von v9; er benötigt dieselben Artefakte und gehört nicht zum normalen Vitest-Lauf. Für einen vollständigen Learning-Vertragslauf werden die verifizierten, credentialfreien SQL-/JS-Vertragsquellen benötigt; vollständige produktive Exporte müssen dafür nicht ins Repository.

## C. Cypress

Auf dem Host fehlen neben `libatk-1.0.so.0` weitere Bibliotheken (AtkBridge, CUPS, Cairo, GTK3, Pango, Xcomposite, Xdamage, Xfixes, Xrandr, GBM, ALSA, AT-SPI). Keine Hostbibliotheken installiert.

Verwendet wurde das offizielle **`cypress/base:26.13.0`**, Manifest-Digest `sha256:5f5138fff456c4250cb327421f6a43e71c3feb50e46f95b3261c5bdc6f4b0531`, mit verifiziertem Node 26.13.0 und dem bereits installierten Cypress **15.21.1**. Eine eigene beschreibbare Kopie des Cypress-Binary-Caches liegt im ignorierten Arbeitsverzeichnis; der ursprüngliche Cache bleibt erhalten. `cypress verify` besteht. Browser und App liegen im eigenen internen Docker-Netz ohne Verbindung zu echten Kunden-Maildiensten. SMTP und Mailprüfung laufen ausschließlich über eigenes MailHog.

Erster vollständiger Lauf über 52 vorhandene Specs: **606 Tests, 522 bestanden, 27 fehlgeschlagen, 55 bereits vorhandene pending, 2 nach einem Renderer-Absturz nicht ausgeführt**. Laufzeit 15:34. Der Renderer-Absturz trat in der AI-Tool-Suite unter gleichzeitigem Build-/Test-Speicherdruck auf; ein konkurrierender Main-Build wurde ebenfalls mit Exit 137 beendet. Beide Frontends wurden anschließend ohne diese Konkurrenz erfolgreich gebaut (Main 45,64 s, Widget 13,82 s).

Die Widget-JWT-Fehler im ersten Lauf beruhten auf `crypto.subtle` unter `http://app:9000`, einem unsicheren Nicht-Loopback-Kontext. Die Wiederholung verwendet den Netzwerk-Namespace ausschließlich der eigenen Staging-App und `http://localhost:9000`; kein unsicherer Browser-Schalter und keine Abschaltung von Kryptografie oder Fehlerbehandlung. Wiederholung der acht betroffenen Specs plus neuer Custom-Spec mit korrigierter Admin-Profilprüfung und localhost: **74 Tests, 55 bestanden, 18 fehlgeschlagen, 1 bestehender pending, keine Skips/Renderer-Abstürze** (5:19). AI-Tool 6/6, Widget-Embed 6/6, Widget-Session/Auth 8/8 und neue Custom-UI-Prüfungen 3/3 erfolgreich. Diese Wiederholung lief noch vor dem SMTP-Pin; MailHog-Zustellung wurde dadurch als eigener Upgradefehler sichtbar.

Die Browserbefehle liefen aus einem eigenen Testcontainer (die Umgebungsvariable `CYPRESS_SYSTEM_PASSWORD` enthält nur das Passwort des synthetischen Staging-Systembenutzers):

```sh
docker run --rm --name libredesk-stabilization-cypress \
  --network container:libredesk-stabilization-app --shm-size=1g \
  -e CYPRESS_RUN_BINARY=/cypress/15.21.1/Cypress/Cypress \
  -e CYPRESS_CACHE_FOLDER=/cypress -e XDG_CONFIG_HOME=/tmp/cypress-config \
  -e CYPRESS_BASE_URL=http://localhost:9000 -e CYPRESS_SYSTEM_PASSWORD \
  -e CYPRESS_MAILHOG_URL=http://mailhog:8025 \
  -e CYPRESS_SMTP_HOST=mailhog -e CYPRESS_SMTP_PORT=1025 \
  -v "$WORK/cypress-cache:/cypress" \
  -v /opt/libredesk-custom/frontend:/e2e -w /e2e \
  cypress/base:26.13.0 node node_modules/cypress/bin/cypress run --e2e --headless \
  --spec 'cypress/e2e/adminFlows.cy.js,cypress/e2e/api/contacts.cy.js,cypress/e2e/ui/aiToolForm.cy.js,cypress/e2e/ui/conversationLifecycle.cy.js,cypress/e2e/ui/createConversationForm.cy.js,cypress/e2e/ui/incomingConversation.cy.js,cypress/e2e/ui/postUpgrade.cy.js,cypress/e2e/integration/livechat/embed.cy.js,cypress/e2e/integration/livechat/session.cy.js'
```

Der vollständige erste Lauf nutzte denselben Containerbefehl mit `--network libredesk-stabilization-net`, Base-URL `http://app:9000` und ohne `--spec`. Die abschließende SMTP-/Custom-Wiederholung begrenzt `--spec` auf `adminFlows.cy.js` und `ui/postUpgrade.cy.js`.

Die drei neuen Browsertests bestätigen einen Admin-Deep-Link bei künstlich verzögerter echter Profilantwort, den Erhalt eines ungesendeten Tiptap-Entwurfs beim Expandieren/Wiederherstellen sowie Addieren/Bestätigen/Entfernen einer Service-Adresse über Sidebar, API und Administration. Keine echten Mails oder KI-Dienste verwendet.

Verbleibende unabhängige Browserblocker:

- `api/contacts.cy.js`: vier Erwartungen 404/400 treffen auf die bestehende `personalContactAccess`-Autorisierung, die vor dem Handler einheitlich 403 liefert. Rechteprüfung nicht umgangen.
- `ui/createConversationForm.cy.js`: fünf alte Erwartungen an lokale E-Mail-/Vorname-/Inbox-Pflichtfehler. Das bestehende Custom-Schema setzt diese Felder optional; der Server validiert anschließend. Kein Rückbau der internen/externalen Ticketlogik, keine abgeschwächten Assertions.
- `ui/incomingConversation.cy.js`: fünf Fälle. Nicht-Admin-Agent verwendet Admin-Route `/inboxes/all`, außerdem passt das Seed-/Sichtbarkeitsmodell nicht zu den aktuellen ACLs. Kein pauschales `read_all`-/Admin-Freischalten oder Umgehen privater Inbox-Grenzen.
- `ui/conversationLifecycle.cy.js`: drei alte UI-Annahmen (`Select agent`, `.bg-private` und `/\d+\s*Open/` als Buttonselektor). Teamzuweisung, Priorität, Tags, Thread-Öffnen und Darstellung einer Antwort bestehen. Die noch fehlgeschlagene Agentzuweisung und Notiz-/Statusdarstellung sind nicht als browserseitig abgenommen gewertet.

Abschließend mit smtppool 1.2: **8/8 Browserfälle bestanden** (adminFlows und postUpgrade), einschließlich echter SMTP-Zustellung. Damit verbleiben **17** Fehler in den vier oben beschriebenen bestehenden Suites. Dies ist eine Zusammenführung der getrennten Läufe; es gab keinen anschließend komplett grünen Gesamtlauf aller nun 53 Specs. Die vorhandenen 55 pending-Fälle bleiben offen.

Ein erfolgreicher Start allein ist keine Browserfreigabe. Bereits vorher vorhandene `it.skip` bleiben sichtbar; es wurden keine neuen Skips eingeführt.

## D. Isoliertes Staging und Regressionen

Für die Prüfung erstellte eigene Container mit Präfix `libredesk-stabilization-`: App (Alpine 3.24.4, selbst gebautes Go-Binary und beide Frontends), PostgreSQL 17 mit pgvector, Redis 7 ohne Persistenz, MailHog, offizieller Cypress-Container. Eigene Netzwerke, Datenbank- und Upload-Testdaten in tmpfs, keine produktiven Mounts oder Credentials. Kein vorhandenes Docker-Volume gelöscht. Das interne App-Netz veröffentlicht auf diesem Docker-Host keine Hostports. Der erste Browserlauf verwendete `http://app:9000`; die Wiederholung verwendet `http://localhost:9000` im gemeinsamen Netzwerk-Namespace der eigenen Test-App. MailHog bleibt ausschließlich `http://mailhog:8025`.

Für lokale Go-Integrationstests lief eine separate eigene PostgreSQL-Instanz ausschließlich auf **127.0.0.1:55437**, ebenfalls mit synthetischen Daten und eigener tmpfs-Datenablage. Learning benutzt dort ausschließlich `learning_test`. Die DSNs und der zufällige Staging-Schlüssel werden nicht in Projektdateien übernommen.

Nach Abschluss wurden alle fünf eigens angelegten Testcontainer und beide Testnetzwerke entfernt. Keine Docker-Volumes gelöscht; lokale Image-/Werkzeugcaches und die ignorierten Rohlogs bleiben für die Nachvollziehbarkeit erhalten.

### Migrationen

- Neuinstallation mit `--install --idempotent-install --yes` erfolgreich, Systembenutzer erzeugt.
- Tatsächlicher `--upgrade --yes` durch die komplette registrierte Migrationsfolge bis v2.23.0 erfolgreich.
- Wiederholtes Upgrade: erfolgreich, keine weiteren Migrationen erforderlich.
- Neuer Integrationstest `TestPostUpgradeMigrationChainPreservesExistingRecords`: älteres Schema nach v2.19 mit vorhandenen Benutzer-, Inbox- und Ordnerdaten; v2.20, v2.21, v2.22 und v2.23 jeweils zweimal ausgeführt. Bestandsdaten bleiben erhalten; nullable Telefon-/Signaturfelder, einmalige Berechtigungen und bestehende Service-Mail-Ordner geprüft.
- Vorhandene Integrationstests prüfen zusätzlich die einzelnen Migrationen und Konsistenz von Queries und Modellen. Kein produktiver Migrationslauf und keine Migrationsimplementierung geändert.

### Abdeckung der Custom-Funktionen

| Bereich                                                                                     | Tatsächliche Prüfung                                                                                                                  | Grenze                                                                                                                                      |
| ------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| Kundentickets, Umverteilt, Rückgabe, Wiederöffnen, Mehrfachteams, Creator-/Herrscher-Rechte | PostgreSQL-Tests `TestCustomerRouting*`, einschließlich Listen, Counts, Filter, Rollback, konkurrierende Änderungen und WebSocket-ACL | Kein echter externer Bearbeiterdienst angesprochen                                                                                          |
| Sidebar, Teams, eigene/private/sichtbare Postfächer, Custom Folders, Filter und Pagination  | `TestSidebar*`, `TestMyInbox*`, `TestPersonal*`, `TestRedistributed*`; Vue-Komponenten- und Listenpositions-Tests                     | Kein vollständiger manueller Browserdurchlauf jeder Ordnerkombination                                                                       |
| Service-Mail-Verwaltung und gemeinsame Registry, Ordner, Filter, Zähler, KI-Ausschluss      | `TestServiceEmail*`, `TestServiceMailFoldersIntegration`, SQL-Vertragsfixture sowie ServiceEmailManager-/ServiceMailSection-Vitest    | Nachgelagerte produktive n8n-Konsumenten bleiben ungetestet                                                                                 |
| Spam-Priorität und Antwortbehandlung                                                        | `TestSpam*`: Transaktionen, Wiederherstellung, exklusiver Filter vor Pagination/Count, eingehende Nachrichten und Mailbox-ACL         | Kein produktiver Spam-Workflow ausgeführt                                                                                                   |
| UUID, private Postfächer, Medien, WebSocket, Visibility-Entzug                              | Go-Integration in authz/conversation/inbox/media/ws/httputil; Race Detector                                                           | Tests verwenden synthetische Identitäten                                                                                                    |
| Composer, Fullscreen, KI-Übernahme/Stil/Feedback, Signaturen und Geschäftstelefon           | Vorhandene Vitest- und Go-Tests einschließlich Sender-Kontext und Template-/Inbox-Signaturgrenzen                                     | Keine echten KI-Dienste; Formatierung, Links, Tabellen, Mentions und jeder Entwurfs-/Signaturablauf nicht vollständig im Browser abgenommen |
| Widget                                                                                      | Build und Widget-Vitest einschließlich WebSocket-Reconnect-Retry; vorhandene Cypress-Livechat-Suite                                   | Browserresultate separat, kein echter Kundeneinbau                                                                                          |

### Nachgewiesene SMTP-Regression und Korrektur

Der reale Antwort-/MailHog-Test entdeckte `SMTP STARTTLS extension not found`: smtppool 1.3 verlangt STARTTLS auch dann, wenn LibreDesk für das explizite `tls_type=none` keinen TLSConfig übergibt. Lokaler Vergleich der vollständigen Poolquellen von 1.2 und 1.3: ausschließlich zwei Bedingungen bei implizitem TLS/STARTTLS unterscheiden sich. Die Pool-/Verbindungszählerkorrekturen sind schon in 1.2 enthalten. Deshalb gezielter Pin auf **1.2.0**, keine Änderung an den SMTP-Produktfunktionen oder TLS-Einstellungen und kein neuer Mailversandmechanismus.

Der neue `TestSMTPConfiguredTransport` prüft mit eigenen Loopback-SMTP-Servern sechs reale Transportfälle: explizites Plaintext, erforderliches STARTTLS, sicherer Default ohne expliziten Typ, funktionierendes STARTTLS, implizites TLS und weiterhin abgewiesenes unbekanntes Zertifikat. Alle sechs bestehen mit 1.2. Derselbe Plaintext-Test scheitert kontrolliert mit 1.3 und genau der beobachteten STARTTLS-Meldung. Der Test ist damit ein unabhängiger Regressionsnachweis; es wurde keine Produktions-TLS-Prüfung abgeschaltet. Die abschließende Browserprüfung mit dem korrigierten Binary besteht: **adminFlows 5/5 einschließlich tatsächlicher Zustellung an MailHog**, Custom-UI 3/3; zusammen **8/8**, keine Skips (50 Sekunden).

## E. Verbleibende Sicherheitsbefunde

Außer dem oben erläuterten SMTP-Kompatibilitätspin keine weiteren Paketversionen in dieser Runde geändert. Die sechs bekannten npm-Befunde bleiben offen; zwei Meldungen betreffen denselben Decoder. Das ist keine Entwarnung. Installierte Versionen und Verwendung wurden lokal geprüft; öffentliche Advisory-Seiten wurden einzeln gelesen. Ein neuer globaler `pnpm audit`-Lauf wurde von der automatischen Freigabeprüfung wegen Übertragung des Dependency-Graphen an einen externen Dienst abgewiesen. Er wurde nicht umgangen. Die frühere Zahl sechs ist deshalb keine neu erhobene vollständige Audit-Summe.

| Befund                                                                                                                                                                                | Installiert / tatsächlicher Pfad                                                     | Bewertung und Updategrenze                                                                                                                                                                                                                                                                                                                             |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| [Zod GHSA-mrhx-c7xc-m3w5](https://github.com/advisories/GHSA-mrhx-c7xc-m3w5), hoch                                                                                                    | 3.25.76, Browser-Formularvalidierung über `@vee-validate/zod` 4.15.1                 | Große ungültige Arrays können den Speicher durch unbeschränkte Fehlerobjekte erschöpfen. Kein Node-Produktionsserver; dennoch potenzieller UI-DoS. Der gepatchte Zod-4-Zweig passt nicht zum Peer-Vertrag `^3.24.0`; kein Override auf Zod 4.                                                                                                          |
| [Tiptap GHSA-cp6q-959q-f8rh](https://github.com/advisories/GHSA-cp6q-959q-f8rh), mittel                                                                                               | Core 2.27.3, tatsächlicher Browsereditor und eigene Extensions mit `mergeAttributes` | Präparierte JSON-Attribute können über Prototype-Manipulation in HTML-Attribute und XSS gelangen. Gefundene Editor-Eingänge verwenden HTML-Strings bzw. fest erzeugte JSON-Strukturen; kein freier Import fremder Editor-JSON gefunden. Das beweist keine vollständige Unerreichbarkeit. Fix erst 3.30.4; vollständige Editor-Migration separat nötig. |
| [decode-uri-component GHSA-vcc3-ghjq-m6fr](https://github.com/advisories/GHSA-vcc3-ghjq-m6fr), mittel; [GHSA-qr28-fpjm-wq7f](https://github.com/advisories/GHSA-qr28-fpjm-wq7f), hoch | 0.2.2 über optionalen Stylus-/CSS-/Source-Map-Buildpfad                              | Aufwendiges Decodieren fehlerhafter Prozentfolgen; hier Buildwerkzeug, kein öffentlicher Go-HTTP-Decoder. 0.5.1 ist ESM, der alte Consumer erwartet eine CommonJS-Funktion. Ein erzwungenes Override wäre kein nachgewiesen kompatibler Patch. Nur vertrauenswürdige Buildquellen verarbeiten.                                                         |
| [glob GHSA-3g3m-2fjr-6pw4](https://github.com/advisories/GHSA-3g3m-2fjr-6pw4), mittel                                                                                                 | 10.5.0 über Sucrase/Tailwind-Werkzeuge; daneben 7.2.3                                | Abgebrochene Dateibaumsuche läuft weiter. Sucrase verarbeitet Projektpfade, kein Request-Handler und kein dortiger AbortSignal-Pfad gefunden. Patch 14.0.2 ist ein weiterer Major; kein ungeprüftes Override.                                                                                                                                          |
| [minimatch GHSA-g3hv-75wg-m249](https://github.com/advisories/GHSA-g3hv-75wg-m249), mittel                                                                                            | 9.0.9 und 10.2.6 in Build-/Lint-Werkzeugen; daneben 3.1.5                            | Expansion bei `optimizationLevel >= 2` und fremden Mustern; hier Repository-Konfiguration statt öffentlicher Pattern-Eingaben. Patch 11.0.1 ist ein weiterer Major. Kein öffentlicher Node-Dienst ausgeliefert.                                                                                                                                        |
| [Go GO-2026-5932](https://pkg.go.dev/vuln/GO-2026-5932)                                                                                                                               | `golang.org/x/crypto` v0.58.0; tatsächlicher `./cmd`-Importpfad verwendet bcrypt     | Hinweis betrifft die nicht mehr gepflegte OpenPGP-Paketfamilie. `go list -deps ./cmd` enthält kein `x/crypto/openpgp`; dieser konkrete Pfad ist im Server nicht importiert. Das ist keine pauschale Aussage über das gesamte Modul. Kein unnötiger Wechsel auf eine Pseudoversion.                                                                     |

Buildwerkzeug-Befunde bleiben für Entwickler/CI relevant, wenn fremde Quelltexte oder Konfigurationen gebaut werden. Ein statisches Frontend macht clientseitige Zod-/Tiptap-Befunde nicht automatisch harmlos. Die bereits dokumentierten Entscheidungen gegen Tiptap 3, Zod 4, Tailwind 4 und Vite 8 (SIGILL) bleiben erhalten.

## F. Dateien und exakte Validierung

Lokale Rohlogs, Ausgangssnapshot und Test-Image-Kontext liegen ausschließlich unter `frontend/node_modules/.cache/post-upgrade-stabilization/` (ignoriert). `before/` und `baseline.patch` unterscheiden diese Runde von den bereits vorhandenen Upgradeänderungen. Die synthetische Staging-Konfiguration ist keine Commitdatei.

Die finalen Prüfungen nutzten die temporär bereitgestellte Node-/pnpm-Toolchain. `$WORK` bezeichnet das oben genannte ignorierte Arbeitsverzeichnis; `$TEST_DSN` ausschließlich die eigene lokale Testdatenbank und `$LEARNING_DSN` dort `learning_test` (keine Produktionsverbindung).

```sh
# aus frontend/
pnpm install --frozen-lockfile
pnpm lint
pnpm test:run --maxWorkers=1
NODE_OPTIONS=--max-old-space-size=2560 pnpm build:main
NODE_OPTIONS=--max-old-space-size=2560 pnpm build:widget
sha256sum -c "$WORK/lock-before.sha256"

# aus dem Repository; GOMAXPROCS=2, GOMEMLIMIT=512MiB
LIBREDESK_TEST_DB_DSN="$TEST_DSN" go test -p 1 -count=1 -json ./...
LIBREDESK_TEST_DB_DSN="$TEST_DSN" go test -race -p 1 -count=1 -json ./internal/authz ./internal/conversation ./internal/inbox ./internal/media ./internal/ws ./internal/httputil ./internal/aiagent ./internal/inbox/channel/email
LIBREDESK_TEST_DB_DSN="$TEST_DSN" LIBREDESK_LEARNING_TEST_DSN="$LEARNING_DSN" go test -p 1 -count=1 -json ./internal/learning
LIBREDESK_TEST_DB_DSN="$TEST_DSN" go test -p 1 -count=1 -run '^TestPostUpgradeMigrationChainPreservesExistingRecords$' -v ./internal/migrations
node --test internal/learning/artifacts/workflow.test.mjs
go vet -p 1 ./...
go mod verify
go mod tidy -diff
CGO_ENABLED=0 go build -p 1 -ldflags='-X main.versionString=v2.23.0 -X github.com/abhinavxd/libredesk/internal/version.Version=v2.23.0 -X main.buildString=post-upgrade-staging' -o "$WORK/image-context/libredesk" ./cmd
stuffbin -a stuff -in "$WORK/image-context/libredesk" -out "$WORK/image-context/libredesk" frontend/dist i18n schema.sql static
# Nur mit eigener synthetischer Konfiguration in diesem Image-Kontext:
docker build -t libredesk-stabilization:current "$WORK/image-context"
git diff --check
```

| Prüfung                                        | Finales Ergebnis                                                                                                                         |
| ---------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| Frozen-Install                                 | Erfolg; SHA256 von Manifest und Lockfile unverändert, pnpm 12.18.0                                                                       |
| ESLint                                         | Gesamtes Frontend, 0 Fehler                                                                                                              |
| Vitest                                         | 74 Dateien, **1.093 Tests bestanden**, keine fehlgeschlagen/übersprungen                                                                 |
| Cypress                                        | Vollständiger Erstlauf und gezielte Wiederholungen siehe C; finale SMTP-/Custom-Prüfung 8/8, insgesamt weiter 17 offene Browserfälle     |
| Main / Widget                                  | Beide Produktionsbuilds erfolgreich; bestehende Fonts-/PURE-/Chunkgrößenhinweise sind keine Buildfehler                                  |
| Go, vollständiger Lauf mit PostgreSQL          | **1.558 bestanden, 28 bestehende Skips, 0 Fehler**                                                                                       |
| Learning, gesondert mit aktiviertem PostgreSQL | **63 bestanden, 2 Fehler** wegen fehlender Vertragsdateien, keine Skips                                                                  |
| Learning Node-Vertragsdatei                    | 1 Datei fehlgeschlagen mit ENOENT beim Import; keine Aussage über ausgeführte Einzelverträge                                             |
| Race Detector, acht kritische Pakete           | **682 bestanden, 1 bestehender optionaler Export-Skip, 0 Fehler**                                                                        |
| Go Vet                                         | Erfolg, einschließlich behobener Copylocks                                                                                               |
| Modulprüfung / tidy                            | `all modules verified`; `go mod tidy -diff` ohne Diff                                                                                    |
| Go-Build / Stuffbin / eigenes Alpine-Image     | Erfolg                                                                                                                                   |
| GitHub-Actions-YAML                            | Alle acht Workflow-Dateien erfolgreich mit YAML BaseLoader geparst; keine Remoteausführung                                               |
| Testdaten                                      | Service-Mail-SQL und README sowie vier Learning-SQL-Dateien bereits Git-getrackt; keine neuen produktiven Exporte                        |
| Testintegrität                                 | AST-Vergleich aller 25 bereinigten Cypress-Dateien: Anzahl Assertions, Tests, `.skip` und `.only` gegenüber Eingangssnapshot unverändert |
| Diff-/Secret-Sichtung                          | `git diff --check` erfolgreich; neue Daten ausschließlich synthetisch, keine Staging-Konfiguration im Projekt-Diff                       |

Die 28 Go-Skips bestehen aus 26 Learning-Einträgen (im normalen Gesamtlauf ist dessen gesonderter DSN nicht gesetzt), dem optionalen lokalen Service-Mail-Exportvergleich und dem bestehenden `TestValidateAgentRequestEmptyRolesSlice` (bekannte Behandlung einer leeren Rollenliste). Der separate Learning-Lauf macht die dortigen Fehler ausdrücklich sichtbar. Ein erster Go-Lauf bei noch nicht erreichbarem Testport wurde verworfen; die obigen 1.558 sind der tatsächlich datenbankgestützte Abschlusslauf.

### Vollständige Dateiliste dieser Stabilisierung

- `.github/workflows/frontend-ci.yml`
- `.github/workflows/go.yml`
- `.gitignore`
- `docs/POST_UPGRADE_STABILIZATION.md`
- `frontend/apps/main/src/App.vue`
- `frontend/apps/main/src/components/combobox/SelectCombobox.vue`
- `frontend/apps/main/src/components/editor/extensions/ResizableImage.js`
- `frontend/apps/main/src/components/importer/Importer.vue`
- `frontend/apps/main/src/composables/useAllInboxRouteGuard.js`
- `frontend/apps/main/src/composables/useAllInboxRouteGuard.test.js`
- `frontend/apps/main/src/features/admin/agents/AgentForm.vue`
- `frontend/apps/main/src/features/admin/automation/ActionBox.vue`
- `frontend/apps/main/src/features/admin/automation/RuleBox.vue`
- `frontend/apps/main/src/features/admin/inbox/EmailInboxForm.vue`
- `frontend/apps/main/src/features/admin/inbox/LivechatInboxForm.vue`
- `frontend/apps/main/src/features/conversation/ReplyBox.vue`
- `frontend/apps/main/src/features/conversation/ReplyBoxContent.vue`
- `frontend/apps/main/src/main.js`
- `frontend/apps/main/src/stores/appSettings.js`
- `frontend/apps/widget/src/composables/useBusinessHours.js`
- `frontend/apps/widget/src/websocket.js`
- `frontend/apps/widget/src/websocket.test.js`
- `frontend/cypress/e2e/adminFlows.cy.js`
- `frontend/cypress/e2e/ui/agentForm.cy.js`
- `frontend/cypress/e2e/ui/aiSnippetForm.cy.js`
- `frontend/cypress/e2e/ui/aiToolForm.cy.js`
- `frontend/cypress/e2e/ui/automationForm.cy.js`
- `frontend/cypress/e2e/ui/businessHoursForm.cy.js`
- `frontend/cypress/e2e/ui/contextLinkForm.cy.js`
- `frontend/cypress/e2e/ui/conversationLifecycle.cy.js`
- `frontend/cypress/e2e/ui/createConversationForm.cy.js`
- `frontend/cypress/e2e/ui/customAttributeForm.cy.js`
- `frontend/cypress/e2e/ui/emailInboxForm.cy.js`
- `frontend/cypress/e2e/ui/generalSettingsForm.cy.js`
- `frontend/cypress/e2e/ui/helpCenterForm.cy.js`
- `frontend/cypress/e2e/ui/incomingConversation.cy.js`
- `frontend/cypress/e2e/ui/livechatInboxForm.cy.js`
- `frontend/cypress/e2e/ui/macroForm.cy.js`
- `frontend/cypress/e2e/ui/notificationSettingsForm.cy.js`
- `frontend/cypress/e2e/ui/postUpgrade.cy.js`
- `frontend/cypress/e2e/ui/roleForm.cy.js`
- `frontend/cypress/e2e/ui/slaForm.cy.js`
- `frontend/cypress/e2e/ui/statusForm.cy.js`
- `frontend/cypress/e2e/ui/tagForm.cy.js`
- `frontend/cypress/e2e/ui/teamForm.cy.js`
- `frontend/cypress/e2e/ui/templateForm.cy.js`
- `frontend/cypress/e2e/ui/webhookForm.cy.js`
- `frontend/cypress/support/commands.js`
- `frontend/eslint.config.js`
- `frontend/shared-ui/components/ui/auto-form/dependencies.js`
- `frontend/shared-ui/components/ui/combobox/ComboBox.vue`
- `go.mod`
- `go.sum`
- `internal/conversation/my_inbox_folders_test.go`
- `internal/inbox/channel/email/smtp_transport_test.go`
- `internal/migrations/post_upgrade_test.go`

Der abschließende Pfadabgleich findet keine unerwarteten Änderungen; die 15 unverändert gebliebenen, vorhandenen Upgrade-Dateien stimmen bytegenau mit dem Eingangssnapshot überein.

Die weiteren bereits vorhandenen Upgradeänderungen bleiben erhalten: `.node-version`, `.goreleaser.yaml`, `Dockerfile`, `Makefile`, `frontend/package.json`, `frontend/pnpm-lock.yaml`, `frontend/pnpm-workspace.yaml`, `frontend/vite.config.js`, die Entfernung von `.eslintrc.cjs`, die beiden I18n-Testkommentare, der Upgradebericht sowie die Upgrade-Pins in Crowdin/Issues/Build-Cache/Release. Diese Runde führt keine erneute Dependency-Migration durch; nur der ausdrücklich dokumentierte SMTP-Pin überschreibt das frühere 1.3.0-Ziel des Upgradeberichts.

## G. Deployment-Freigabe

| Frage                               | Bewertung                                                                                                                                                                                                                                                             |
| ----------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| CI bereit?                          | Lint, Unit-/Go-/PostgreSQL-Prüfungen und Builds lokal erfolgreich; **gesamte CI noch nicht grün**, weil verbindliche Cypress-Tests weiterhin scheitern. Remote-Actions nicht ausgeführt; zusätzlicher Learning-Vertragslauf durch fehlende Originalquellen blockiert. |
| Staging bereit?                     | Eigene Instanz startet, Migrationen, Anmeldung, API, Frontend/Widget und WebSocket-Abläufe geprüft. Vollständige funktionale Abnahme weiterhin offen.                                                                                                                 |
| Browserregressionen bestanden?      | **Teilweise**: finale SMTP-Zustellung, Custom-UI, Widget und zahlreiche vorhandene Specs bestehen; die oben einzeln beschriebenen Altvertrags-/Selektor-/Sichtbarkeitsfälle bleiben rot. Keine neuen Skips oder gelöschten Assertions.                                |
| Datenbankmigrationen kompatibel?    | **Ja, im geprüften isolierten Umfang**: Neuinstallation, vollständige Upgradefolge, älteres Testschema, Wiederholung und Datenbestandserhalt erfolgreich.                                                                                                             |
| Produktivdeployment empfehlenswert? | **Derzeit nein.** Erst die verbleibenden Browserverträge klären, Learning-SQL-/JS-Vertragsquellen verlässlich bereitstellen und die offenen Security-Risiken bewerten.                                                                                                |

Es wurde kein Produktivdeployment ausgeführt. Die Ergebnisse gelten für synthetische lokale Testsysteme; produktive n8n-Konsumenten, echte E-Mail-Postfächer, KI-Dienste und vollständige manuelle Browserkombinationen bleiben außerhalb dieser Abnahme.
