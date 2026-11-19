Interne Tickets: Einrichtung und F02
=================================

Der interne Modus verwendete bisher im Browser immer Posteingang-ID 3.
Bei Mitarbeiterzuweisung wurden dessen E-Mail und Name übernommen; ohne
Mitarbeiter diente eine feste Firmenadresse als Kontakt und Empfänger.
Diese Werte erfüllten die Anforderungen des gemeinsamen E-Mail-Create-Pfads:
Ein Ticket benötigt einen Kontakt, die erste Nachricht einen Empfänger und
einen E-Mail-Posteingang. Eine besondere technische Bedeutung der ID 3 besteht
im Backend nicht.

Die Zuordnung erfolgt jetzt ausschließlich serverseitig über zwei optionale
TOML-Parameter im vorhandenen Abschnitt `[conversation]`:

```toml
[conversation]
internal_inbox_uuid = ""
internal_contact_email = ""
```

`internal_inbox_uuid` identifiziert einen vorhandenen, aktiven, öffentlichen
E-Mail-Posteingang. Seine UUID ist bereits eindeutig in der Datenbank und bleibt
bei einer Umbenennung erhalten. Numerische IDs und nicht eindeutige Namen
werden nicht zur automatischen Auswahl verwendet. Es gibt keinen Fallback.

`internal_contact_email` ist der ausdrücklich gewählte Kontakt und tatsächliche
E-Mail-Empfänger für interne Tickets ohne direkt zugewiesenen Mitarbeiter,
insbesondere reine Teamtickets. Bei Mitarbeiterzuweisung verwendet der Server
wie bisher die gespeicherte E-Mail-Adresse und den Namen dieses aktiven
Mitarbeiters; dafür ist `internal_contact_email` nicht erforderlich. Bestehende
Kontakte werden wiederverwendet und nicht mit den angezeigten Stammdaten
überschrieben. Fehlende Kontakte entstehen über den vorhandenen Create-Pfad.
Die erste Nachricht bleibt eine normale ausgehende E-Mail; diese Änderung
wandelt sie nicht in eine private Notiz um.

Die entsprechenden Umgebungsvariablen sind
`LIBREDESK_CONVERSATION__INTERNAL_INBOX_UUID` und
`LIBREDESK_CONVERSATION__INTERNAL_CONTACT_EMAIL`.

Einrichtung:

1. Bei Bedarf über die bestehende Posteingangsverwaltung ein öffentliches
   E-Mail-Postfach mit gültiger Absender-/Transportkonfiguration einrichten.
2. Seine UUID aus der authentifizierten Antwort von `GET /api/v1/inboxes`
   übernehmen und `internal_inbox_uuid` setzen.
3. Für Teamtickets ohne direkten Mitarbeiter die gewünschte Empfängeradresse
   in `internal_contact_email` eintragen. Für bestehende Installationen kann
   damit die bisher verwendete Adresse ausdrücklich beibehalten werden.
4. Konfiguration beim nächsten freigegebenen Anwendungsneustart laden lassen.
   Dieses Implementierungsvorhaben ändert keine produktive Konfiguration und
   startet keine produktiven Dienste neu.

Nach einer Neuinstallation oder nach Löschen/Neuanlegen des Postfachs ist seine
neue UUID einzutragen. Eine leere Installation ohne E-Mail-Postfach bleibt
möglich: Die Parameter sind keine Start- oder Installationsvoraussetzung.
Fehler werden erst beim Verwenden der internen Erstellung gemeldet. Öffentliche,
persönliche und private nicht-interne Erstellung benötigen diese Konfiguration
nicht.

API und Schutz:

- Der bisherige Create-Endpunkt und die Attribute `private`/`internal` bleiben
  bestehen. Der interne Browsermodus sendet weder `inbox_id` noch Kontaktdaten.
- Bestehende API-Clients dürfen die tatsächliche konfigurierte Posteingang-ID
  weiterhin mitsenden. Eine widersprüchliche ID wird mit HTTP 400 abgelehnt;
  sie wird nicht stillschweigend auf ein anderes Postfach umgebogen.
- Bei internen Tickets bestimmt der Server den Kontakt. Mitgesendete externe
  Kundendaten und `external_user_id` wählen keinen anderen Empfänger aus.
- Existenz, Aktivierung, E-Mail-Kanal und öffentlicher Zugriffsmodus werden
  geprüft. Fehlende/ungültige Einrichtung führt zu verständlichen HTTP-400-
  Fehlern, bevor Kontakte oder Tickets geschrieben werden.
- Die F01-Validierung prüft weiterhin aktive Agenten, Create-Berechtigung,
  Zuweisungsrechte, existierende Teams und aktive Mitarbeiter. F01/F03,
  Creator/ACL, Mehrfachzuweisung und bestehende Sichtbarkeitsendpunkte bleiben
  maßgeblich. System-Tags und Benutzer-ID-Ausnahmen werden nicht geändert.
- Webhook-Ereignisse, Felder, Custom Attributes und Nachrichten-Metadaten
  behalten ihre Struktur. `inbox_id` enthält die tatsächliche konfigurierte ID,
  der Empfänger folgt der oben beschriebenen bisherigen Mitarbeiter-/Teamlogik.
  Externe n8n-Workflows sind hier nicht vollständig versioniert verfügbar.
  Falls sie selbst ID 3 oder eine Empfängeradresse fest voraussetzen, muss ihre
  Zuordnung vor einer späteren Aktivierung geprüft werden. Echte n8n-Durchläufe
  werden durch lokale Vertragstests nicht ersetzt.

Es werden keine Datensätze migriert, historischen Migrationen geändert oder
Postfächer automatisch angelegt. Bestehende Tickets und produktive Einstellungen
bleiben unverändert. Vor Aktivierung eines neuen Builds müssen Betreiber die
gewünschte UUID und gegebenenfalls die bisherige Team-Empfängeradresse setzen.

Gezielte Regressionstests liegen in `cmd/conversation_internal_test.go` sowie
`frontend/cypress/e2e/ui/createConversationForm.cy.js`. Die bestehenden
F01/F03-HTTP-Tests erhalten lediglich eine explizite interne Testkonfiguration;
ihre Schutzassertionen bleiben erhalten. Die tatsächlichen Prüfergebnisse und
verbleibenden Validierungsgrenzen werden im Abschlussbericht dokumentiert.

Validierung am 19.11.2026:

| Prüfung | Ergebnis |
| --- | --- |
| Vollständige betroffene Go-Pakete `./cmd ./internal/conversation ./internal/inbox ./internal/webhook ./internal/authz`, `-count=1` | 772 Tests/Untertests bestanden, keine Fehler, zwei bestehende Skips |
| F02-Fälle innerhalb dieses Laufs | 28 Tests/Untertests bestanden |
| Gezielter Race-Lauf: F02, F01/F03, Webhooks, Sichtbarkeits-/Spam-Parallelität | 57 Tests/Untertests bestanden, keine Fehler/Skips/Races |
| Vollständige Frontend-Unit-Suite, zwei Worker | 80 Dateien, 1.136 Tests bestanden |
| Cypress `createConversationForm.cy.js` | 17 Tests bestanden, keine Fehler/Skips; die beiden neuen UI-Fälle prüfen die Payload und Fehlerbehandlung mit einer simulierten Konfigurationsfehlermeldung |
| Go-Backend, Main, Widget, Stuffbin-Assets | Erfolgreich; alle 213 eingebetteten Dateien bytegleich geprüft |
| Frontend-Lint, `go vet` der betroffenen Pakete, `gofmt`, `git diff --check` | Erfolgreich |
| Installation und Upgrade auf leerer DB, anschließend Start ohne Inbox/Interne-Konfiguration | Erfolgreich, Inbox-Tabelle leer, HTTP 200 |

Die Go-HTTP-Tests verwenden echte Manager und PostgreSQL 17.12 mit einer
ausdrücklich isolierten DSN. Der Browser verwendet eine separate PostgreSQL-17.10-
Instanz, Redis und MailHog in einem internen Docker-Netz ohne veröffentlichte
Ports. Node 26.13.0 und pnpm 12.18.0, Frozen-Lockfile-Installation erfolgreich.
Go-Prüfungen mit `GOFLAGS=-p=2` und `GOMAXPROCS=2`, Frontend-Unit-Tests mit zwei Workern.

Der erste größere Go-Lauf erschöpfte den anfänglichen flüchtigen 2-GB-
Testdatenträger; die betroffenen Pakete wurden auf einer neuen 8-GB-Testinstanz
vollständig und erfolgreich wiederholt. Anfängliche Fehler im neuen Testfixture
und ein Cypress-Lintfehler wurden korrigiert. Keine Testregeln abgeschwächt oder
Skips ergänzt. Die zwei bestehenden Go-Skips betreffen leere Rollenlisten und
den optionalen lokalen Workflowexport-Vergleich.

Die gesamte Repository-Go-Suite und die gesamte Cypress-Suite wurden in diesem
F02-Schritt nicht wiederholt. Reale n8n-, SMTP-/IMAP-/OAuth-Durchläufe außerhalb
des Testnetzes sind nicht geprüft; der existierende Webhook-Vertragstest und
die Empfänger-/Custom-Attribute-Assertions sind erfolgreich. Sämtliche
temporären Testcontainer, Netze, Quellkopien und Buildartefakte wurden
entfernt; Testprotokolle liegen unter `/tmp/libredesk-f02-20261119`.

Geänderte Dateien:

- `cmd/conversation_internal.go`: optionale Konfiguration, UUID-Auflösung und serverseitiger Kontakt.
- `cmd/conversation.go`: internen Create-Request vor der bestehenden Validierung vorbereiten.
- `cmd/main.go`: optionale TOML-Werte einmalig laden.
- `frontend/apps/main/src/features/conversation/CreateConversation.vue`: keine feste ID oder Kontaktadresse und keine alten externen Kontaktdaten im internen Payload.
- `config.sample.toml`: beide optionalen Parameter dokumentieren.
- `cmd/conversation_internal_test.go`: neue F02-HTTP-Regressionen.
- `cmd/conversation_attributes_test.go`: explizite Testkonfiguration für die bestehenden F01/F03-Fälle.
- `frontend/cypress/e2e/ui/createConversationForm.cy.js`: zwei neue Payload-/Fehlerbehandlungsfälle und zugehörige Kontakt-Testdaten.
- `docs/INTERNAL_TICKETS.md`: Einrichtung, Kompatibilität und Ergebnisse.
- `.gitignore`: nur die neue F02-Dokumentation trotz der vorhandenen `docs/*`-Ignorierregel versionierbar machen.
