# Neuinstallation mit leeren Geschäftsdaten

Nur auf einem **neuen isolierten Linux-Server bzw. eigenem Docker-Daemon**
ausführen. Die festen Container-/Netz-/Volumen-Namen und Hostports kollidieren mit
Raptor. Diese Anleitung wurde dort nicht ausgeführt. Keine historischen Daten,
Produktionsvolumes, Workflows, Schlüssel oder Zugangsdaten übernehmen.

## Voraussetzungen und Dateien

- Git, Docker Engine mit Compose v2, GNU make/Standardwerkzeuge, Go 1.27.4,
  Node 26.13.x und pnpm 12.18.0; Python 3 und PostgreSQL-Client `psql` für F15-01.
  Netzwerkzugriff auf Image-Registries, Go-/npm-/PyPI-Pakete und GitHub-Releases.
- LibreDesk-Quellen einschließlich der abgeschlossenen Installationsfixes;
  geprüfter lokaler Stand `4730f0011457e338855c5a7a294616b1285a7239`.
  Dessen Verfügbarkeit im Remote zuerst klären; der letzte Inventarcheck fand
  vier lokale Commits vor der gespeicherten `origin/main`-Referenz.
- Dieser neue `local-install/n8n/`-Bestand und diese Anleitung sind noch
  **uncommittet** und fehlen daher derzeit ebenfalls in einem frischen Clone.
- `local-install/n8n-postgres/` komplett separat bereitstellen: 19 vorhandene,
  getestete Dateien, weiterhin absichtlich gitignoriert. Nicht neu entwickeln und
  nicht durch alte SQL-Dumps ersetzen. `docs/`-Berichte sind nur lokal verfügbar.
- Bestehendes privates Workflow-Repository separat klonen/bereitstellen; dessen
  tägliches Backup bleibt unverändert. Keine neuen Workflow-Backups erforderlich.

## 1. LibreDesk vorbereiten und bauen

```sh
# Auf dem neuen Server: URL und freigegebenen Commit bewusst einsetzen.
git clone <PRIVATE_LIBREDESK_REPOSITORY_URL> /opt/libredesk-custom
cd /opt/libredesk-custom
git checkout <FREIGEGEBENER_COMMIT>
cp config.sample.toml config.toml
mkdir -p uploads
```

`config.toml` neu einrichten: eigener `app.encryption_key` mit genau 32 Zeichen,
DB `db:5432` und Redis `redis:6379`. Neue Root-`.env` mit `POSTGRES_USER`,
`POSTGRES_DB`, `POSTGRES_PASSWORD`, `LIBREDESK_SYSTEM_USER_PASSWORD` anlegen.
DB-Namen/Zugang in TOML und ENV müssen übereinstimmen. Das Systempasswort muss
die vorhandenen Stärkeanforderungen erfüllen. Secrets im geschützten Editor oder
vorhandenen Secret-Werkzeug setzen, nicht in diese Anleitung oder Shell-Historie.
`config.toml` und beide späteren `.env`-Dateien nur für den Betreiber lesbar halten.
`OPENAI_API_KEY`/`TRANSCRIPTION_PROVIDER` erst für verwendete LibreDesk-Funktionen
einrichten. Ohne E-Mail-Postfach muss der Basisstart funktionieren.

```sh
make build
docker compose -p libredesk-custom config --quiet
docker compose -p libredesk-custom up -d --build
docker compose -p libredesk-custom ps
docker compose -p libredesk-custom exec -T app ./libredesk --check-system-tags --config /libredesk/config.toml
```

`make build` baut Main, Widget, Backend und bettet Schema/i18n/static ein.
Das bestehende Dockerfile benötigt dieses neue Linux-Binary für die Zielarchitektur.
Compose wartet auf PostgreSQL-Health und gestartetes Redis; App-Installation/
Upgrade laufen bereits im vorhandenen Startkommando. Bei fehlender Bereitschaft
erst Status prüfen, nicht blind erneut initialisieren. Öffentliche LibreDesk-URL
anschließend unter den bestehenden allgemeinen Einstellungen (`app.root_url`)
setzen. Für erste Funktionstests Postfächer/SMTP/IMAP erst nach Bedarf einrichten.

## 2. n8n-Konfiguration, CA, Netze und leere Volumes

Den neuen Installationsbestand nach `local-install/n8n/` legen, dann:

```sh
cd /opt/libredesk-custom/local-install/n8n
cp .env.example .env
chmod 600 .env
```

`N8N_POSTGRES_PASSWORD` und `N8N_ENCRYPTION_KEY` mit **neuen unabhängigen Secrets**
füllen; leere Werte verhindern die Compose-Auflösung. Die Datenbankvariablen
betreffen pg-n8n, nicht die LibreDesk-Datenbank. Der initiale DB-Owner ist kein
späteres Workflow-Laufzeitcredential. `N8N_HOST`, `WEBHOOK_URL`, Protokoll und
Cookie-/Proxy-Einstellungen an den Testserver anpassen. Die Beispiele verwenden
direktes HTTP; bei vorgeschaltetem TLS-Proxy URL und Secure-Cookie entsprechend
setzen. Der Encryption-Key bleibt für dieses neue n8n-Volume stabil.

Die Compose-Datei benötigt eine freigegebene **öffentliche CA-Zertifikatsdatei in
PEM-Format**, standardmäßig `certs/picturesoftRoot.crt` neben dem Compose.
Diese Datei wird absichtlich nicht mitgeliefert. Für den Zielserver passende CA
von der zuständigen Stelle beziehen und Identität/Gültigkeit prüfen; keine
privaten Schlüssel und kein ungeprüftes Produktionszertifikat kopieren.
Alternativen Pfad über `N8N_CA_FILE` setzen. Der schreibgeschützte Mount verweigert
das automatische Anlegen eines Verzeichnisses bei fehlender Quelldatei.

```sh
# Nur auf dem neuen Ziel: vorhandene Namen dürfen keine fremden Daten enthalten.
docker network inspect libredesk-custom_libredesk libredesk_integrations
docker volume ls
# Erst wenn beide Namen noch nicht existieren:
docker volume create n8n_new
docker volume create pg_data
docker compose -p n8n config --quiet
docker compose -p n8n up -d --build
docker compose -p n8n ps
```

`volume create` leert vorhandene Volumes **nicht**; deshalb vorher Namen prüfen.
LibreDesk muss zuerst die zwei extern referenzierten Netze erzeugen. Projektnamen
**`libredesk-custom`** und **`n8n`** beibehalten; der neue Compose setzt zusätzlich
`name: n8n`, damit der Unterordner keine abweichende Netzidentität verursacht.
n8n erzeugt `n8n_n8n_network` selbst. Kein manuelles Netzwerk-Anhängen nötig.
`libredesk_integrations` benötigt das freie Subnetz `172.30.240.0/24`:
n8n `172.30.240.5`, Anonymizer `172.30.240.7`. Bestehende LibreDesk-IP-Pins bleiben
damit passend; bei anderem Subnetz alle Netz-/Pin-Werte gemeinsam bewerten.

Images: n8n **2.25.7**, vorhandenes PG16-/pgvector-0.8.6-Image mit gleichem Digest,
Anonymizer Python 3.12. Dessen Implementierung ist bytegleich übernommen;
FastAPI 0.139.2, Uvicorn 0.51.0, spaCy 3.8.14 und Modell 3.8.0 sind festgelegt.
Das Modell wird als festes Wheel mit SHA-256 aus dem
[offiziellen spaCy-Modellrelease](https://github.com/explosion/spacy-models/releases/tag/de_core_news_lg-3.8.0)
installiert; Build prüft Paketkonsistenz und tatsächliches Laden des Modells.
Basisimage und transitive Python-Abhängigkeiten sind damit noch kein vollständiger
Digest-/Hash-Lock. Ein realer Neubuild ist Teil des folgenden isolierten Tests.

## 3. Leeres Workflow-Schema installieren

n8n erzeugt im neuen `n8n_new`-Volume seine eigene SQLite-Instanz und benötigt ein
neues Owner-Konto. **pg-n8n enthält die Workflow-/Learning-Daten, nicht die
n8n-Credentials.** PostgreSQL-Bereitschaft abwarten; dieses Compose hat dafür
keine projektübergreifende Startabhängigkeit.

Den unveränderten F15-01-Bestand separat bereitstellen. Explizite libpq-Verbindung
auf die **neue** pg-n8n-DB einstellen (`PGHOST`, `PGPORT`, `PGDATABASE`, `PGUSER`,
geschützte Passwortbereitstellung). Vom Testhost ist der Port **5434**, intern
`pg-n8n:5432`. Neues Extension-Installationskonto und getrennten vertrauenswürdigen
Schema-Owner einrichten; nicht gegen LibreDesk oder eine Produktions-DB ausführen.

```sh
cd /opt/libredesk-custom/local-install/n8n-postgres
# Mit Extension-Installationskonto und explizit gewählter neuer Ziel-DB:
psql -X -v ON_ERROR_STOP=1 -f sql/000_extensions.sql
# Jetzt libpq-Verbindung auf den getrennten Schema-Owner umstellen:
python3 install.py
python3 install.py --check
# Separate eingeschränkte Principals vorher administrativ anlegen:
# n8n_workflow, libredesk_learning, n8n_case_publisher, n8n_vector
python3 configure_roles.py --workflow-role n8n_workflow \
  --case-publisher-role n8n_case_publisher --vector-role n8n_vector
```

Rollenpasswörter, LOGIN/CONNECT und `pg_hba` sind bewusste neue Einrichtung;
der getrennte Schema-Owner benötigt CREATE im Ziel-Schema `public`.
Der Rechtehelfer erstellt keine Zugangsdaten. Das Vector-Profil ist für die
bestehenden nativen PGVector-Nodes nötig und bleibt vom allgemeinen Workflow-
Credential getrennt; Review/Publisher ebenfalls getrennt. Keine produktiven
Rollen/IDs übernehmen. Optionales Knowledge-Modul für den ersten Test weglassen.
Schema-/Rechtevertrag und weitere Details stehen lokal in
[`../docs/F15_SQL_BASELINE_VALIDATION.md`](../docs/F15_SQL_BASELINE_VALIDATION.md).

## 4. Workflows und neue Integrationszugänge

Bestehendes privates Workflow-Repository auf dem neuen Server klonen. Gewünschte
JSONs über den vorhandenen n8n-Import zunächst **inaktiv** importieren und Zustand
kontrollieren. Keine Aktivierungszustände, alte Pin-Daten oder Ausführungszustände
blind übernehmen; historische Alternativflows nicht parallel einschalten.
Bestehende Exporte werden hier nicht verändert. Alte Credential-IDs ersetzen
keine neuen Zugangsdaten oder Projekt-/Owner-Zuordnungen.

- PostgreSQL-Credentials auf neue Rollen/DBs zuordnen: Workflow, Vector und
  LibreDesk-Registry-Lesen separat; LibreDesk-Learning-/Publisher-DSNs gemäß Vertrag.
- Neuen aktiven LibreDesk-Integrationsagenten mit beliebiger ID und API-Key/Secret
  erstellen. Nur benötigte Ticket-/Inbox-Rechte und `messages:read`, `messages:write`,
  `conversations:update_tags`, `conversations:add_system_tags`, ggf. Status/
  Priorität und `messages:write_spam_detection_note` vergeben. Neue ID separat in
  `ai_analysis.allowed_agent_ids` eintragen; ID 41 ist keine Voraussetzung.
- Neue OpenAI- und externe Bestell-/Produktsuch-Credentials zuordnen; LAN-Ziele
  und passende CA prüfen. Eingangs-, Feedback-, Style- und Versandabschluss-
  Webhooks samt notwendigen Sender-/Empfänger-Secrets bewusst einrichten;
  LibreDesk-Abonnements nicht aus Datenbankkopien ableiten.
- Vor Funktionsaktivierung bekannte F15-Vertragslücken berücksichtigen:
  Spam-Notizzweck/-Erfolgskette und kontaktlose interne Tickets. Eingangsauthentisierung
  bleibt die gesonderte F15-02-Abnahme; dieser Bestand implementiert sie nicht.

## 5. Erreichbarkeit und Abnahme

Zuerst `docker compose ... ps` und DB-/App-Bereitschaft prüfen. Auf dem neuen Ziel
DNS/TCP aus dem jeweiligen Container und anschließend fachliche HTTP-Tests mit
synthetischen Daten durchführen:

| Verbraucher | Interne Ziele |
|---|---|
| LibreDesk | `db:5432`, `redis:6379`, `n8n:5678` |
| n8n | `pg-n8n:5432`, `libredesk_db:5432`, `libredesk_app:9000`, `anonymizer:8001/analyze` |

Hostports bleiben wie im Bestand: LibreDesk 9000, n8n 5678, pg-n8n 5434,
Anonymizer 8001; LibreDesk-DB/Redis nur Host-Loopback 5432/6379.
Proxy-/Basis-URLs und freie Ports für den Testserver passend konfigurieren.
Kein Zugriff auf produktive Endpunkte, keine echten Kundenmails.

Abnehmen: leerer LibreDesk-Start/Systemdatenprüfung, F15-01-Schema-Check,
Anonymizer mit synthetischem Text, neue Credentials, ein ausgewählter Eingang/
Analyse-Pfad sowie Feedback/Versandabschluss bei Bedarf. Erst nach diesen Tests
gewünschte Workflows bewusst aktivieren. Historische Datenübernahme, neue
Backup-Systeme, PDF-Service und Bewerbungsgenerator gehören nicht hierzu.
