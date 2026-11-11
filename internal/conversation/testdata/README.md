# Service-Mail-Workflow: SQL-Testfixture

`service_workflow_registry_query.sql` enthält ausschließlich die Abfrage der
Node `Execute a SQL query2` aus dem zuvor in dieser Arbeitsfolge geprüften Export
`workflows/E-Mail Anonymisierung - Service Mail Guard.json`. Sie enthält keine
Credentials oder Workflow-Metadaten und muss zusammen mit
`internal/conversation/service_emails_test.go` versioniert und gepusht werden.

Der vollständige Export ist nicht versioniert und im aktuellen Checkout nicht
vorhanden. `workflows/` wird nicht durch `.gitignore` ausgeschlossen: Es wurde
lediglich kein Export ins Repository aufgenommen. Daher funktionierte der alte
Test nur mit der lokalen Datei. Die Go-CI (`.github/workflows/go.yml`) führt nach
`actions/checkout` mit PostgreSQL 17 `make test` aus; sie erzeugt keine n8n-Exporte.

Der erforderliche `TestServiceWorkflowRegistryQueryIntegration` liest jetzt die
SQL-Fixture vor dem Datenbankaufbau. Fehlende oder leere Fixtures führen zu einem
Fehler. Es gibt keinen Export-Fallback und keinen Skip wegen eines fehlenden
Exports. Der vorhandene PostgreSQL-Test prüft weiterhin Hinzufügen/Entfernen,
Public/Personal, Groß-/Kleinschreibung, Rand-Leerzeichen einschließlich Tabs/CR/LF,
ungültige UUIDs und Fehler beim Zugriff auf `service_email_addresses`.
Die bestehende Testdatenbank-Voraussetzung des Projekts bleibt unverändert.

## Optionaler lokaler Abgleich

`TestServiceWorkflowRegistryQueryMatchesLocalExport` vergleicht die Fixture mit
`Execute a SQL query2`, falls der übliche lokale Export vorhanden ist. Nur dessen
Abwesenheit wird ausdrücklich als optional protokolliert. Ungültiges JSON, eine
fehlende/doppelte Node oder eine abweichende Abfrage sind Fehler.

Ein anderer Export kann vom Repository-Root explizit gewählt werden:

```sh
LIBREDESK_SERVICE_WORKFLOW_EXPORT=/absoluter/pfad/export.json \
  go test ./internal/conversation \
  -run '^TestServiceWorkflowRegistryQueryMatchesLocalExport$' -count=1 -v
```

Bei expliziter Pfadangabe ist auch eine fehlende Datei ein Fehler. Der Vergleich
ignoriert ausschließlich Rand-Leerzeichen der gesamten SQL-Abfrage. Andere
Formatänderungen erfordern eine bewusste Prüfung von Export und Fixture.

Ohne echten lokalen Export kann dieser Vergleich keine Übereinstimmung mit einer
neueren oder produktiven n8n-Version bestätigen. Die SQL-Integration läuft dennoch
vollständig mit der festgehaltenen Abfrage. Die Abfrage wird nicht zur Laufzeit aus
produktiven Go-Konstanten generiert; Änderungen müssen bewusst übernommen werden.

## Umfang und bekannte separate Prüfhinweise

Produktive Go-Dateien, Workflow-/CI-Konfiguration und vollständige Exporte werden
für diesen Fix nicht geändert. Die ebenfalls vorhandene Frontenddatei
`serviceMailWorkflow.test.js` liest weiterhin lokale vollständige Exporte. Das ist
eine separate Abhängigkeit außerhalb dieses ausdrücklich auf die Go-SQL-Fixture
beschränkten Fixes; der vollständige `make test`-Lauf wird hierdurch nicht als
insgesamt grün behauptet.

## Lokale Validierung dieses Fixes

- Der angeforderte Befehl `go test ./internal/conversation -run
  '^TestServiceWorkflowRegistryQueryIntegration$' -count=1 -v` bestand mit einer
  eigens angelegten PostgreSQL-17-Testinstanz auf `127.0.0.1:5433` und einem
  schreibbaren Go-Cache unter `/tmp`. `workflows/` war dabei vollständig abwesend.
- Relevanter Go-Lauf über `./internal/conversation ./cmd` mit
  `-run '^(TestService|TestAIAnalysis|TestManualResolutionActualRenderedSend)'`:
  14 bestandene Tests/Subtests. Nur der getrennte optionale Exportvergleich
  wurde wegen des fehlenden lokalen Exports übersprungen.
- Der Vergleichsmechanismus wurde zusätzlich mit minimalen synthetischen
  Exportdateien unter `/tmp` geprüft: gleiche Abfrage besteht; abweichende Abfrage
  scheitert; ein explizit gewählter fehlender Export scheitert ebenfalls.
  Diese Proben bestätigen nicht den Inhalt einer aktuell fehlenden echten
  Workflow-Datei.
- `gofmt` und `git diff --check` bestanden. Fixture und diese Dokumentation werden
  ausdrücklich in den Git-Index aufgenommen; keine Commits oder Pushes.

Für den nächsten Commit/Push sind genau diese drei Dateien erforderlich:

1. `internal/conversation/service_emails_test.go`
2. `internal/conversation/testdata/service_workflow_registry_query.sql`
3. `internal/conversation/testdata/README.md`
