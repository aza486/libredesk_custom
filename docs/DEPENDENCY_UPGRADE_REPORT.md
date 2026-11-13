# Dependency Upgrade Report

Stand: 11.11.2026. Audit vor Manifeständerungen; abschließende Ergebnisse unten. Uhrzeiten der Rohlogs sind UTC, Berichtdatum Europe/Berlin.

## A. Ausgangszustand und Strategie

Sauberer Working Tree zu Beginn. Node lokal 24.15.0, pnpm lokal 11.4.0, Go lokal 1.26.8. Manifest pnpm 9.15.3, Go 1.25.0; CI Node 20. Keine Commits, Deployments oder Zugriffe auf produktive Datenbanken.

Die Versionsangaben stammen aus Live-Abfragen von registry.npmjs.org, proxy.golang.org, go.dev/dl und nodejs.org/dist am Auditdatum. `latest` kann unerwartet auf ältere oder Vorabversionen zeigen: vuedraggable latest=2.24.3 ist die Vue-2-Linie; Node latest zeigt eine Alpha-Version. Diese Tags werden nicht blind übernommen.

A: kompatible Updates innerhalb einer Major-Linie. B: Toolchain-/Frontend-Migration mit Tests. C: persistente Services und externe Projekte, nur Inventar und separater Plan. Tiptap bleibt gemeinsam auf 2.27.3; Tailwind bleibt wegen JS-Tokens, Plugins und bestehender Utilities auf 3.4.19. Zod 3 bleibt wegen @vee-validate/zod 4 peer ^3.24.0.

### Frontend: vollständiger direkter Audit

| Dependency                            | Manifest vorher | Installiert vorher | Neueste Registry latest | Kompatibler Zweig  | Risiko |
| ------------------------------------- | --------------- | ------------------ | ----------------------- | ------------------ | ------ |
| @codemirror/lang-html                 | ^6.4.9          | 6.4.9              | 6.4.12                  | 6.4.12             | A      |
| @codemirror/lang-javascript           | ^6.2.4          | 6.2.4              | 6.2.5                   | 6.2.5              | A      |
| @codemirror/theme-one-dark            | ^6.1.3          | 6.1.3              | 6.1.3                   | 6.1.3              | A      |
| @formkit/auto-animate                 | ^0.8.2          | 0.8.2              | 0.10.2                  | 0.10.2             | A      |
| @internationalized/date               | ^3.6.0          | 3.6.0              | 3.12.4                  | 3.12.4             | A      |
| @radix-icons/vue                      | ^1.0.0          | 1.0.0              | 1.0.0                   | 1.0.0              | A      |
| @tailwindcss/typography               | ^0.5.19         | 0.5.19             | 0.5.20                  | 0.5.20             | A      |
| @tanstack/vue-table                   | ^8.20.5         | 8.20.5             | 9.3.1                   | 8.21.3             | A      |
| @tanstack/vue-virtual                 | ^3.13.23        | 3.13.23            | 3.13.52                 | 3.13.52            | A      |
| @tiptap/extension-code-block-lowlight | ^2.11.2         | 2.27.2             | 3.33.1                  | 2.27.3             | A      |
| @tiptap/extension-image               | ^2.11.2         | 2.11.2             | 3.33.1                  | 2.27.3             | A      |
| @tiptap/extension-link                | ^2.11.2         | 2.11.2             | 3.33.1                  | 2.27.3             | A      |
| @tiptap/extension-mention             | ^2.27.1         | 2.27.1             | 3.33.1                  | 2.27.3             | A      |
| @tiptap/extension-placeholder         | ^2.11.2         | 2.11.2             | 3.33.1                  | 2.27.3             | A      |
| @tiptap/extension-table               | ^2.11.5         | 2.11.5             | 3.33.1                  | 2.27.3             | A      |
| @tiptap/extension-table-cell          | ^2.11.5         | 2.11.5             | 3.33.1                  | 2.27.3             | A      |
| @tiptap/extension-table-header        | ^2.11.5         | 2.11.5             | 3.33.1                  | 2.27.3             | A      |
| @tiptap/extension-table-row           | ^2.11.5         | 2.11.5             | 3.33.1                  | 2.27.3             | A      |
| @tiptap/extension-text-align          | ^2.11.2         | 2.27.2             | 3.33.1                  | 2.27.3             | A      |
| @tiptap/extension-underline           | ^2.11.2         | 2.27.2             | 3.33.1                  | 2.27.3             | A      |
| @tiptap/extension-youtube             | ^2.11.2         | 2.27.2             | 3.33.1                  | 2.27.3             | A      |
| @tiptap/pm                            | ^2.11.2         | 2.11.2             | 3.33.1                  | 2.27.3             | A      |
| @tiptap/starter-kit                   | ^2.11.2         | 2.11.2             | 3.33.1                  | 2.27.3             | A      |
| @tiptap/suggestion                    | ^2.27.1         | 2.27.1             | 3.33.1                  | 2.27.3             | A      |
| @tiptap/vue-3                         | ^2.11.2         | 2.11.2             | 3.33.1                  | 2.27.3             | A      |
| @unovis/ts                            | ^1.6.5          | 1.6.5              | 1.8.1                   | 1.8.1              | A      |
| @unovis/vue                           | ^1.6.5          | 1.6.5              | 1.8.1                   | 1.8.1              | A      |
| @vee-validate/zod                     | ^4.15.0         | 4.15.0             | 4.15.1                  | 4.15.1             | A      |
| @vueuse/core                          | ^12.4.0         | 12.4.0             | 15.0.0                  | 12.8.2             | B      |
| axios                                 | ^1.18.0         | 1.18.0             | 1.21.1                  | 1.21.1             | A      |
| class-variance-authority              | ^0.7.1          | 0.7.1              | 0.7.1                   | 0.7.1              | A      |
| clsx                                  | ^2.1.1          | 2.1.1              | 2.1.1                   | 2.1.1              | A      |
| codemirror                            | ^6.0.2          | 6.0.2              | 6.0.2                   | 6.0.2 (latest-Tag) | A      |
| date-fns                              | ^3.6.0          | 3.6.0              | 4.4.0                   | 3.6.0              | A      |
| highlight.js                          | ^11.12.0        | 11.12.0            | 11.12.0                 | 11.12.0            | A      |
| lettersanitizer                       | ^1.0.7          | 1.0.7              | 1.0.9                   | 1.0.9              | A      |
| lowlight                              | ^3.3.0          | 3.3.0              | 3.3.0                   | 3.3.0              | A      |
| lucide-vue-next                       | ^0.525.0        | 0.525.0            | 1.0.0                   | 0.577.0            | A      |
| mitt                                  | ^3.0.1          | 3.0.1              | 3.0.1                   | 3.0.1              | A      |
| pinia                                 | ^2.3.0          | 2.3.0              | 4.1.0                   | 2.3.1              | B      |
| qs                                    | ^6.15.2         | 6.15.2             | 6.17.1                  | 6.17.1             | A      |
| radix-vue                             | ^1.9.17         | 1.9.17             | 1.9.17                  | 1.9.17             | A      |
| reka-ui                               | ^2.9.7          | 2.9.7              | 2.11.0                  | 2.11.0             | A      |
| tailwind-merge                        | ^2.6.0          | 2.6.0              | 3.7.0                   | 2.6.1              | A      |
| vee-validate                          | ^4.15.0         | 4.15.0             | 4.15.1                  | 4.15.1             | A      |
| vue                                   | ^3.5.13         | 3.5.13             | 3.5.45                  | 3.5.45             | A      |
| vue-easy-lightbox                     | ^1.19.0         | 1.19.0             | 1.19.0                  | 1.19.0             | A      |
| vue-i18n                              | ^9.14.5         | 9.14.5             | 11.4.18                 | 9.14.5             | B      |
| vue-letter                            | ^0.2.1          | 0.2.1              | 0.2.3                   | 0.2.3              | A      |
| vue-picture-cropper                   | ^0.7.0          | 0.7.0              | 1.0.0                   | 0.7.0              | A      |
| vue-router                            | ^4.5.0          | 4.5.0              | 5.5.0                   | 4.6.4              | A      |
| vue-sonner                            | ^1.3.0          | 1.3.0              | 2.0.9                   | 1.3.2              | A      |
| vue3-emoji-picker                     | ^1.1.8          | 1.1.8              | 2.1.0                   | 1.1.8              | A      |
| vuedraggable                          | ^4.1.0          | 4.1.0              | 2.24.3                  | 4.1.0              | A      |
| zod                                   | ^3.24.1         | 3.24.1             | 4.6.7                   | 3.25.76            | A      |
| @rushstack/eslint-patch               | ^1.10.5         | 1.10.5             | 1.16.1                  | 1.16.1             | A      |
| @vitejs/plugin-vue                    | ^5.2.1          | 5.2.1              | 6.0.10                  | 5.2.4              | B      |
| @vue/eslint-config-prettier           | ^8.0.0          | 8.0.0              | 10.2.0                  | 8.0.0              | A      |
| autoprefixer                          | ^10.4.20        | 10.4.20            | 10.6.2                  | 10.6.2             | A      |
| cypress                               | ^15.15.0        | 15.15.0            | 16.3.0                  | 15.21.1            | B      |
| eslint                                | ^8.57.1         | 8.57.1             | 10.14.0                 | 8.57.1             | B      |
| eslint-plugin-cypress                 | ^2.15.2         | 2.15.2             | 7.0.4                   | 2.15.2             | A      |
| eslint-plugin-vue                     | ^9.32.0         | 9.32.0             | 10.12.1                 | 9.33.0             | A      |
| jsdom                                 | ^29.0.1         | 29.0.1             | 31.0.0                  | 29.1.1             | B      |
| postcss                               | ^8.5.23         | 8.5.23             | 8.5.32                  | 8.5.32             | A      |
| prettier                              | ^3.4.2          | 3.4.2              | 3.10.0                  | 3.10.0             | A      |
| sass                                  | ^1.99.0         | 1.99.0             | 1.108.2                 | 1.108.2            | A      |
| start-server-and-test                 | ^2.0.9          | 2.0.9              | 3.0.13                  | 2.1.5              | A      |
| tailwindcss                           | ^3.4.17         | 3.4.17             | 4.3.5                   | 3.4.19             | A      |
| tailwindcss-animate                   | ^1.0.7          | 1.0.7              | 1.0.7                   | 1.0.7              | A      |
| vite                                  | ^6.4.3          | 6.4.3              | 8.3.8                   | 6.4.6              | B      |
| vitest                                | ^4.1.0          | 4.1.0              | 5.1.0                   | 4.1.12             | B      |
| pnpm                                  | 9.15.3          | 11.4.0             | 12.18.0                 | 9.15.9             | B      |
| node                                  | 24.15.0         | 24.15.0            | 26.13.0 LTS (stabil)    | 24.22.2 / 26.13.0  | B      |

Installierter Stand ist aus node_modules gelesen. Die ursprünglichen Lockfile-Auflösungen werden in der Ergebnistabelle separat gegenübergestellt. Bei 0.x-Paketen bedeutet gleiche Major-Zahl keine API-Garantie. codemirror bleibt auf dem offiziellen latest 6.0.2 statt einer höheren nicht als latest markierten Nummer.

### Go: vollständiger direkter Audit

| Modul                                             | Vorher                             | Neueste Version im Modulpfad       | Risiko |
| ------------------------------------------------- | ---------------------------------- | ---------------------------------- | ------ |
| github.com/abhinavxd/ssrfguard                    | v0.1.0                             | v0.1.0                             | A      |
| github.com/alicebob/miniredis/v2                  | v2.33.0                            | v2.40.0                            | B      |
| github.com/coreos/go-oidc/v3                      | v3.11.0                            | v3.21.0                            | B      |
| github.com/disintegration/imaging                 | v1.6.2                             | v1.6.2                             | A      |
| github.com/emersion/go-imap/v2                    | v2.0.0-beta.3                      | v2.0.0-beta.8                      | B      |
| github.com/emersion/go-message                    | v0.18.1                            | v0.18.2                            | B      |
| github.com/fasthttp/websocket                     | v1.5.9                             | v1.5.12                            | B      |
| github.com/ferluci/fast-realip                    | v1.0.1                             | v1.0.1                             | A      |
| github.com/gabriel-vasile/mimetype                | v1.4.11                            | v1.4.16                            | B      |
| github.com/golang-jwt/jwt/v5                      | v5.2.2                             | v5.3.1                             | B      |
| github.com/google/uuid                            | v1.6.0                             | v1.6.0                             | A      |
| github.com/inbucket/html2text                     | v1.0.0                             | v1.0.0                             | A      |
| github.com/jhillyerd/enmime/v2                    | v2.4.1                             | v2.5.0                             | B      |
| github.com/jmoiron/sqlx                           | v1.4.0                             | v1.4.0                             | A      |
| github.com/knadh/go-i18n                          | v0.1.0                             | v1.0.0                             | B      |
| github.com/knadh/goyesql/v2                       | v2.2.0                             | v2.2.0                             | A      |
| github.com/knadh/koanf/parsers/json               | v0.1.0                             | v1.0.1                             | B      |
| github.com/knadh/koanf/parsers/toml               | v0.1.0                             | v0.1.0                             | A      |
| github.com/knadh/koanf/providers/confmap          | v0.1.0                             | v1.0.1                             | B      |
| github.com/knadh/koanf/providers/env/v2           | v2.0.0                             | v2.0.2                             | B      |
| github.com/knadh/koanf/providers/file             | v0.1.0                             | v1.2.1                             | B      |
| github.com/knadh/koanf/providers/posflag          | v0.1.0                             | v1.0.2                             | B      |
| github.com/knadh/koanf/providers/rawbytes         | v0.1.0                             | v1.0.1                             | B      |
| github.com/knadh/koanf/v2                         | v2.1.1                             | v2.4.1                             | B      |
| github.com/knadh/smtppool                         | v1.1.0                             | v1.3.0                             | B      |
| github.com/knadh/stuffbin                         | v1.3.0                             | v1.3.0                             | A      |
| github.com/lib/pq                                 | v1.10.9                            | v1.12.3                            | B      |
| github.com/mackee/go-readability                  | v0.3.1                             | v0.3.1                             | A      |
| github.com/microcosm-cc/bluemonday                | v1.0.27                            | v1.0.27                            | A      |
| github.com/mr-karan/balance                       | v0.0.0-20250317053523-d32c6ade6cf1 | v0.0.0-20250317053523-d32c6ade6cf1 | A      |
| github.com/pkoukk/tiktoken-go                     | v0.1.8                             | v0.1.8                             | A      |
| github.com/pkoukk/tiktoken-go-loader              | v0.0.2                             | v0.0.2                             | A      |
| github.com/redis/go-redis/v9                      | v9.5.5                             | v9.23.0                            | B      |
| github.com/rhnvrm/simples3                        | v0.10.1                            | v0.11.1                            | B      |
| github.com/spf13/pflag                            | v1.0.6                             | v1.1.0                             | B      |
| github.com/stretchr/testify                       | v1.11.1                            | v1.12.1                            | B      |
| github.com/valyala/fasthttp                       | v1.62.0                            | v1.78.0                            | B      |
| github.com/volatiletech/null/v9                   | v9.0.0                             | v9.0.1                             | B      |
| github.com/yuin/goldmark                          | v1.8.4                             | v1.8.7                             | B      |
| github.com/zerodha/fastcache/stores/goredis/v9    | v9.1.0                             | v9.1.0                             | A      |
| github.com/zerodha/fastcache/v4                   | v4.2.1                             | v4.2.1                             | A      |
| github.com/zerodha/fastglue                       | v1.8.0                             | v1.8.1                             | B      |
| github.com/zerodha/logf                           | v0.5.5                             | v0.5.5                             | A      |
| github.com/zerodha/simplesessions/stores/redis/v3 | v3.0.0                             | v3.0.0                             | A      |
| github.com/zerodha/simplesessions/v3              | v3.0.0                             | v3.0.0                             | A      |
| golang.org/x/crypto                               | v0.52.0                            | v0.58.0                            | B      |
| golang.org/x/mod                                  | v0.35.0                            | v0.43.0                            | B      |
| golang.org/x/net                                  | v0.55.0                            | v0.62.0                            | B      |
| golang.org/x/oauth2                               | v0.27.0                            | v0.38.0                            | B      |
| golang.org/x/text                                 | v0.37.0                            | v0.44.0                            | B      |

Go-Updates werden pro explizitem Modulpfad ausgewählt, keine pauschale go-get-u-Operation. go-imap/v2 bleibt vorerst beta.3: das Modul hat noch keine stabile v2-Veröffentlichung, Änderungen an IMAP-APIs erfordern gesonderte Mailserver-Regression.

## B. Upgrade-Ergebnis

Node wurde zum Testen ausschließlich unter `/tmp` installiert; globale Host-Installationen wurden nicht ersetzt. `.node-version`, Frontend-Engines und CI stimmen auf Node 26.13.0 LTS überein. Go wird über `go.mod` und `setup-go` auf 1.27.4 ausgerichtet. pnpm 12.18.0 ist exakt gepinnt; Corepack wird nicht vorausgesetzt. CI liest die pnpm-Version über `pnpm/action-setup` direkt aus `frontend/package.json`.

| Komponente     | Vorher                           | Nachher     | Neueste stabil               | Ergebnis                                                          |
| -------------- | -------------------------------- | ----------- | ---------------------------- | ----------------------------------------------------------------- |
| Node           | lokal 24.15.0, CI 20             | 26.13.0 LTS | 26.13.0 LTS                  | Unter temporärer Toolchain getestet; Host bleibt unverändert      |
| pnpm           | Manifest 9.15.3, lokal 11.4.0    | 12.18.0     | 12.18.0                      | Frozen-Install erfolgreich                                        |
| Go             | Manifest/CI 1.25.0, lokal 1.26.8 | 1.27.4      | 1.27.4                       | Toolchain heruntergeladen und getestet                            |
| Alpine         | 3.18                             | 3.24.4      | 3.24.4                       | Isolierter Build und netzwerkloser Version-Smoke-Test erfolgreich |
| PostgreSQL App | laufend 17.10 / Tag 17-alpine    | unverändert | 18.6, Wartungszweig 17.11    | Separater Rollout; Tests zusätzlich mit 17.11                     |
| Redis          | laufend 7.4.9 / Tag 7-alpine     | unverändert | 8.10.3, Wartungszweig 7.4.12 | Separater Rollout                                                 |

### Direkte Frontendpakete: Lockfile vorher/nachher und gewähltes Ziel

Die Auflösung stammt aus den jeweiligen Lockfiles, nicht aus Manifest-Ranges. Peer-Suffixe sind hier zur Lesbarkeit entfernt. Alle ursprünglichen direkten Pakete sind enthalten; neue Lint-Pakete folgen separat. 56 von 72 ursprünglichen Manifest-Einträgen wurden angepasst, davon zwei durch Flat-Config-Ersatz entfernt.

| Paket                                 | Lock vorher | Manifest nachher | Lock nachher | Neueste Registry latest | Ergebnis / Empfehlung                      |
| ------------------------------------- | ----------- | ---------------- | ------------ | ----------------------- | ------------------------------------------ |
| @codemirror/lang-html                 | 6.4.9       | ^6.4.12          | 6.4.12       | 6.4.12                  | Aktualisiert                               |
| @codemirror/lang-javascript           | 6.2.4       | ^6.2.5           | 6.2.5        | 6.2.5                   | Aktualisiert                               |
| @codemirror/theme-one-dark            | 6.1.3       | ^6.1.3           | 6.1.3        | 6.1.3                   | Beibehalten / bereits aktuell              |
| @formkit/auto-animate                 | 0.8.2       | ^0.8.2           | 0.8.2        | 0.10.2                  | Beibehalten / bereits aktuell              |
| @internationalized/date               | 3.6.0       | ^3.12.4          | 3.12.4       | 3.12.4                  | Aktualisiert                               |
| @radix-icons/vue                      | 1.0.0       | ^1.0.0           | 1.0.0        | 1.0.0                   | Beibehalten / bereits aktuell              |
| @tailwindcss/typography               | 0.5.19      | ^0.5.20          | 0.5.20       | 0.5.20                  | Aktualisiert                               |
| @tanstack/vue-table                   | 8.20.5      | ^8.21.3          | 8.21.3       | 9.3.1                   | Aktualisiert                               |
| @tanstack/vue-virtual                 | 3.13.23     | ^3.13.52         | 3.13.52      | 3.13.52                 | Aktualisiert                               |
| @tiptap/extension-code-block-lowlight | 2.27.2      | 2.27.3           | 2.27.3       | 3.33.1                  | Aktualisiert                               |
| @tiptap/extension-image               | 2.11.2      | 2.27.3           | 2.27.3       | 3.33.1                  | Aktualisiert                               |
| @tiptap/extension-link                | 2.11.2      | 2.27.3           | 2.27.3       | 3.33.1                  | Aktualisiert                               |
| @tiptap/extension-mention             | 2.27.1      | 2.27.3           | 2.27.3       | 3.33.1                  | Aktualisiert                               |
| @tiptap/extension-placeholder         | 2.11.2      | 2.27.3           | 2.27.3       | 3.33.1                  | Aktualisiert                               |
| @tiptap/extension-table               | 2.11.5      | 2.27.3           | 2.27.3       | 3.33.1                  | Aktualisiert                               |
| @tiptap/extension-table-cell          | 2.11.5      | 2.27.3           | 2.27.3       | 3.33.1                  | Aktualisiert                               |
| @tiptap/extension-table-header        | 2.11.5      | 2.27.3           | 2.27.3       | 3.33.1                  | Aktualisiert                               |
| @tiptap/extension-table-row           | 2.11.5      | 2.27.3           | 2.27.3       | 3.33.1                  | Aktualisiert                               |
| @tiptap/extension-text-align          | 2.27.2      | 2.27.3           | 2.27.3       | 3.33.1                  | Aktualisiert                               |
| @tiptap/extension-underline           | 2.27.2      | 2.27.3           | 2.27.3       | 3.33.1                  | Aktualisiert                               |
| @tiptap/extension-youtube             | 2.27.2      | 2.27.3           | 2.27.3       | 3.33.1                  | Aktualisiert                               |
| @tiptap/pm                            | 2.11.2      | 2.27.3           | 2.27.3       | 3.33.1                  | Aktualisiert                               |
| @tiptap/starter-kit                   | 2.11.2      | 2.27.3           | 2.27.3       | 3.33.1                  | Aktualisiert                               |
| @tiptap/suggestion                    | 2.27.1      | 2.27.3           | 2.27.3       | 3.33.1                  | Aktualisiert                               |
| @tiptap/vue-3                         | 2.11.2      | 2.27.3           | 2.27.3       | 3.33.1                  | Aktualisiert                               |
| @unovis/ts                            | 1.6.5       | ^1.8.1           | 1.8.1        | 1.8.1                   | Aktualisiert                               |
| @unovis/vue                           | 1.6.5       | ^1.8.1           | 1.8.1        | 1.8.1                   | Aktualisiert                               |
| @vee-validate/zod                     | 4.15.0      | ^4.15.1          | 4.15.1       | 4.15.1                  | Aktualisiert                               |
| @vueuse/core                          | 12.4.0      | ^12.8.2          | 12.8.2       | 15.0.0                  | Aktualisiert                               |
| axios                                 | 1.18.0      | ^1.21.1          | 1.21.1       | 1.21.1                  | Aktualisiert                               |
| class-variance-authority              | 0.7.1       | ^0.7.1           | 0.7.1        | 0.7.1                   | Beibehalten / bereits aktuell              |
| clsx                                  | 2.1.1       | ^2.1.1           | 2.1.1        | 2.1.1                   | Beibehalten / bereits aktuell              |
| codemirror                            | 6.0.2       | ^6.0.2           | 6.0.2        | 6.0.2                   | Beibehalten / bereits aktuell              |
| date-fns                              | 3.6.0       | ^3.6.0           | 3.6.0        | 4.4.0                   | Beibehalten / bereits aktuell              |
| highlight.js                          | 11.12.0     | ^11.12.0         | 11.12.0      | 11.12.0                 | Beibehalten / bereits aktuell              |
| lettersanitizer                       | 1.0.7       | ^1.0.9           | 1.0.9        | 1.0.9                   | Aktualisiert                               |
| lowlight                              | 3.3.0       | ^3.3.0           | 3.3.0        | 3.3.0                   | Beibehalten / bereits aktuell              |
| lucide-vue-next                       | 0.525.0     | ^0.577.0         | 0.577.0      | 1.0.0                   | Aktualisiert                               |
| mitt                                  | 3.0.1       | ^3.0.1           | 3.0.1        | 3.0.1                   | Beibehalten / bereits aktuell              |
| pinia                                 | 2.3.0       | ^2.3.1           | 2.3.1        | 4.1.0                   | Aktualisiert                               |
| qs                                    | 6.15.2      | ^6.17.1          | 6.17.1       | 6.17.1                  | Aktualisiert                               |
| radix-vue                             | 1.9.17      | ^1.9.17          | 1.9.17       | 1.9.17                  | Beibehalten / bereits aktuell              |
| reka-ui                               | 2.9.7       | ^2.11.0          | 2.11.0       | 2.11.0                  | Aktualisiert                               |
| tailwind-merge                        | 2.6.0       | ^2.6.1           | 2.6.1        | 3.7.0                   | Aktualisiert                               |
| vee-validate                          | 4.15.0      | ^4.15.1          | 4.15.1       | 4.15.1                  | Aktualisiert                               |
| vue                                   | 3.5.13      | ^3.5.45          | 3.5.45       | 3.5.45                  | Aktualisiert                               |
| vue-easy-lightbox                     | 1.19.0      | ^1.19.0          | 1.19.0       | 1.19.0                  | Beibehalten / bereits aktuell              |
| vue-i18n                              | 9.14.5      | ^11.4.18         | 11.4.18      | 11.4.18                 | Aktualisiert                               |
| vue-letter                            | 0.2.1       | ^0.2.3           | 0.2.3        | 0.2.3                   | Aktualisiert                               |
| vue-picture-cropper                   | 0.7.0       | ^0.7.0           | 0.7.0        | 1.0.0                   | Beibehalten / bereits aktuell              |
| vue-router                            | 4.5.0       | ^4.6.4           | 4.6.4        | 5.5.0                   | Aktualisiert                               |
| vue-sonner                            | 1.3.0       | ^1.3.2           | 1.3.2        | 2.0.9                   | Aktualisiert                               |
| vue3-emoji-picker                     | 1.1.8       | ^1.1.8           | 1.1.8        | 2.1.0                   | Beibehalten / bereits aktuell              |
| vuedraggable                          | 4.1.0       | ^4.1.0           | 4.1.0        | 2.24.3                  | Beibehalten / bereits aktuell              |
| zod                                   | 3.24.1      | ^3.25.76         | 3.25.76      | 4.6.7                   | Aktualisiert                               |
| @rushstack/eslint-patch               | 1.10.5      | —                | —            | 1.16.1                  | Entfernt: durch native Flat Config ersetzt |
| @vitejs/plugin-vue                    | 5.2.1       | ^6.0.10          | 6.0.10       | 6.0.10                  | Aktualisiert                               |
| @vue/eslint-config-prettier           | 8.0.0       | —                | —            | 10.2.0                  | Entfernt: durch native Flat Config ersetzt |
| autoprefixer                          | 10.4.20     | ^10.6.2          | 10.6.2       | 10.6.2                  | Aktualisiert                               |
| cypress                               | 15.15.0     | ^15.21.1         | 15.21.1      | 16.3.0                  | Aktualisiert                               |
| eslint                                | 8.57.1      | ^10.14.0         | 10.14.0      | 10.14.0                 | Aktualisiert                               |
| eslint-plugin-cypress                 | 2.15.2      | ^7.0.4           | 7.0.4        | 7.0.4                   | Aktualisiert                               |
| eslint-plugin-vue                     | 9.32.0      | ^10.12.1         | 10.12.1      | 10.12.1                 | Aktualisiert                               |
| jsdom                                 | 29.0.1      | ^31.0.0          | 31.0.0       | 31.0.0                  | Aktualisiert                               |
| postcss                               | 8.5.23      | ^8.5.32          | 8.5.32       | 8.5.32                  | Aktualisiert                               |
| prettier                              | 3.4.2       | ^3.10.0          | 3.10.0       | 3.10.0                  | Aktualisiert                               |
| sass                                  | 1.99.0      | ^1.108.2         | 1.108.2      | 1.108.2                 | Aktualisiert                               |
| start-server-and-test                 | 2.0.9       | ^2.1.5           | 2.1.5        | 3.0.13                  | Aktualisiert                               |
| tailwindcss                           | 3.4.17      | ^3.4.19          | 3.4.19       | 4.3.5                   | Aktualisiert                               |
| tailwindcss-animate                   | 1.0.7       | ^1.0.7           | 1.0.7        | 1.0.7                   | Beibehalten / bereits aktuell              |
| vite                                  | 6.4.3       | ^7.3.8           | 7.3.8        | 8.3.8                   | Aktualisiert                               |
| vitest                                | 4.1.0       | ^4.1.12          | 4.1.12       | 5.1.0                   | Aktualisiert                               |

Neue direkte Lint-Abhängigkeiten:

| Paket                  | Manifest | Neueste stabil / aufgelöst |
| ---------------------- | -------- | -------------------------- |
| @eslint/js             | ^10.0.1  | 10.0.1                     |
| globals                | ^17.14.0 | 17.14.0                    |
| eslint-config-prettier | ^10.1.8  | 10.1.8                     |
| vue-eslint-parser      | ^10.3.0  | 10.4.2                     |

### Direkte Go-Module: Ergebnis

| Modul                                             | Vorher                             | Nachher / empfohlen                | Neueste im Modulpfad               | Ergebnis     |
| ------------------------------------------------- | ---------------------------------- | ---------------------------------- | ---------------------------------- | ------------ |
| github.com/abhinavxd/ssrfguard                    | v0.1.0                             | v0.1.0                             | v0.1.0                             | Beibehalten  |
| github.com/alicebob/miniredis/v2                  | v2.33.0                            | v2.40.0                            | v2.40.0                            | Aktualisiert |
| github.com/coreos/go-oidc/v3                      | v3.11.0                            | v3.21.0                            | v3.21.0                            | Aktualisiert |
| github.com/disintegration/imaging                 | v1.6.2                             | v1.6.2                             | v1.6.2                             | Beibehalten  |
| github.com/emersion/go-imap/v2                    | v2.0.0-beta.3                      | v2.0.0-beta.3                      | v2.0.0-beta.8                      | Beibehalten  |
| github.com/emersion/go-message                    | v0.18.1                            | v0.18.2                            | v0.18.2                            | Aktualisiert |
| github.com/fasthttp/websocket                     | v1.5.9                             | v1.5.12                            | v1.5.12                            | Aktualisiert |
| github.com/ferluci/fast-realip                    | v1.0.1                             | v1.0.1                             | v1.0.1                             | Beibehalten  |
| github.com/gabriel-vasile/mimetype                | v1.4.11                            | v1.4.16                            | v1.4.16                            | Aktualisiert |
| github.com/golang-jwt/jwt/v5                      | v5.2.2                             | v5.3.1                             | v5.3.1                             | Aktualisiert |
| github.com/google/uuid                            | v1.6.0                             | v1.6.0                             | v1.6.0                             | Beibehalten  |
| github.com/inbucket/html2text                     | v1.0.0                             | v1.0.0                             | v1.0.0                             | Beibehalten  |
| github.com/jhillyerd/enmime/v2                    | v2.4.1                             | v2.5.0                             | v2.5.0                             | Aktualisiert |
| github.com/jmoiron/sqlx                           | v1.4.0                             | v1.4.0                             | v1.4.0                             | Beibehalten  |
| github.com/knadh/go-i18n                          | v0.1.0                             | v1.0.0                             | v1.0.0                             | Aktualisiert |
| github.com/knadh/goyesql/v2                       | v2.2.0                             | v2.2.0                             | v2.2.0                             | Beibehalten  |
| github.com/knadh/koanf/parsers/json               | v0.1.0                             | v1.0.1                             | v1.0.1                             | Aktualisiert |
| github.com/knadh/koanf/parsers/toml               | v0.1.0                             | v0.1.0                             | v0.1.0                             | Beibehalten  |
| github.com/knadh/koanf/providers/confmap          | v0.1.0                             | v1.0.1                             | v1.0.1                             | Aktualisiert |
| github.com/knadh/koanf/providers/env/v2           | v2.0.0                             | v2.0.2                             | v2.0.2                             | Aktualisiert |
| github.com/knadh/koanf/providers/file             | v0.1.0                             | v1.2.1                             | v1.2.1                             | Aktualisiert |
| github.com/knadh/koanf/providers/posflag          | v0.1.0                             | v1.0.2                             | v1.0.2                             | Aktualisiert |
| github.com/knadh/koanf/providers/rawbytes         | v0.1.0                             | v1.0.1                             | v1.0.1                             | Aktualisiert |
| github.com/knadh/koanf/v2                         | v2.1.1                             | v2.4.1                             | v2.4.1                             | Aktualisiert |
| github.com/knadh/smtppool                         | v1.1.0                             | v1.3.0                             | v1.3.0                             | Aktualisiert |
| github.com/knadh/stuffbin                         | v1.3.0                             | v1.3.0                             | v1.3.0                             | Beibehalten  |
| github.com/lib/pq                                 | v1.10.9                            | v1.12.3                            | v1.12.3                            | Aktualisiert |
| github.com/mackee/go-readability                  | v0.3.1                             | v0.3.1                             | v0.3.1                             | Beibehalten  |
| github.com/microcosm-cc/bluemonday                | v1.0.27                            | v1.0.27                            | v1.0.27                            | Beibehalten  |
| github.com/mr-karan/balance                       | v0.0.0-20250317053523-d32c6ade6cf1 | v0.0.0-20250317053523-d32c6ade6cf1 | v0.0.0-20250317053523-d32c6ade6cf1 | Beibehalten  |
| github.com/pkoukk/tiktoken-go                     | v0.1.8                             | v0.1.8                             | v0.1.8                             | Beibehalten  |
| github.com/pkoukk/tiktoken-go-loader              | v0.0.2                             | v0.0.2                             | v0.0.2                             | Beibehalten  |
| github.com/redis/go-redis/v9                      | v9.5.5                             | v9.23.0                            | v9.23.0                            | Aktualisiert |
| github.com/rhnvrm/simples3                        | v0.10.1                            | v0.11.1                            | v0.11.1                            | Aktualisiert |
| github.com/spf13/pflag                            | v1.0.6                             | v1.1.0                             | v1.1.0                             | Aktualisiert |
| github.com/stretchr/testify                       | v1.11.1                            | v1.12.1                            | v1.12.1                            | Aktualisiert |
| github.com/valyala/fasthttp                       | v1.62.0                            | v1.78.0                            | v1.78.0                            | Aktualisiert |
| github.com/volatiletech/null/v9                   | v9.0.0                             | v9.0.0                             | v9.0.1                             | Beibehalten  |
| github.com/yuin/goldmark                          | v1.8.4                             | v1.8.7                             | v1.8.7                             | Aktualisiert |
| github.com/zerodha/fastcache/stores/goredis/v9    | v9.1.0                             | v9.1.0                             | v9.1.0                             | Beibehalten  |
| github.com/zerodha/fastcache/v4                   | v4.2.1                             | v4.2.1                             | v4.2.1                             | Beibehalten  |
| github.com/zerodha/fastglue                       | v1.8.0                             | v1.8.1                             | v1.8.1                             | Aktualisiert |
| github.com/zerodha/logf                           | v0.5.5                             | v0.5.5                             | v0.5.5                             | Beibehalten  |
| github.com/zerodha/simplesessions/stores/redis/v3 | v3.0.0                             | v3.0.0                             | v3.0.0                             | Beibehalten  |
| github.com/zerodha/simplesessions/v3              | v3.0.0                             | v3.0.0                             | v3.0.0                             | Beibehalten  |
| golang.org/x/crypto                               | v0.52.0                            | v0.58.0                            | v0.58.0                            | Aktualisiert |
| golang.org/x/mod                                  | v0.35.0                            | v0.43.0                            | v0.43.0                            | Aktualisiert |
| golang.org/x/net                                  | v0.55.0                            | v0.62.0                            | v0.62.0                            | Aktualisiert |
| golang.org/x/oauth2                               | v0.27.0                            | v0.38.0                            | v0.38.0                            | Aktualisiert |
| golang.org/x/text                                 | v0.37.0                            | v0.44.0                            | v0.44.0                            | Aktualisiert |

Wichtige transitive Go-Module:

| Modul                         | Vorher  | Nachher |
| ----------------------------- | ------- | ------- |
| golang.org/x/image            | v0.41.0 | v0.48.0 |
| golang.org/x/sys              | v0.45.0 | v0.50.0 |
| github.com/go-jose/go-jose/v4 | v4.1.4  | v4.1.9  |
| github.com/klauspost/compress | v1.18.0 | v1.20.4 |
| github.com/yuin/gopher-lua    | v1.1.1  | v1.1.2  |

### GitHub Actions und Release-Werkzeuge

| Komponente                   | Vorher             | Nachher / neueste stabil | Nachweis                                                                       |
| ---------------------------- | ------------------ | ------------------------ | ------------------------------------------------------------------------------ |
| actions/checkout             | v3 / v4            | v7.0.1                   | [Release](https://github.com/actions/checkout/releases/tag/v7.0.1)             |
| actions/setup-go             | v3 / v5            | v7.0.0                   | [Release](https://github.com/actions/setup-go/releases/tag/v7.0.0)             |
| actions/setup-node           | v3                 | v7.1.0                   | [Release](https://github.com/actions/setup-node/releases/tag/v7.1.0)           |
| actions/cache                | v4                 | v6.1.0                   | [Release](https://github.com/actions/cache/releases/tag/v6.1.0)                |
| actions/stale                | v9                 | v11.0.0                  | [Release](https://github.com/actions/stale/releases/tag/v11.0.0)               |
| actions/first-interaction    | v3.1.0 SHA-gepinnt | v3.1.0                   | [Release](https://github.com/actions/first-interaction/releases/tag/v3.1.0)    |
| pnpm/action-setup            | npm global 9.15.3  | v6.1.1                   | [Release](https://github.com/pnpm/action-setup/releases/tag/v6.1.1)            |
| docker/setup-buildx-action   | v3                 | v4.4.1                   | [Release](https://github.com/docker/setup-buildx-action/releases/tag/v4.4.1)   |
| docker/login-action          | v3                 | v4.8.0                   | [Release](https://github.com/docker/login-action/releases/tag/v4.8.0)          |
| goreleaser/goreleaser-action | v5                 | v7.3.0                   | [Release](https://github.com/goreleaser/goreleaser-action/releases/tag/v7.3.0) |
| crowdin/github-action        | v2                 | v3.6.0                   | [Release](https://github.com/crowdin/github-action/releases/tag/v3.6.0)        |

`actions/first-interaction` bleibt auf seinem bereits aktuellen Commit. Die Setup-Actions verwenden intern Node 24; das ist unabhängig von Node 26 für den Projektbuild. Die ausgewählten Inputs wurden in den offiziellen `action.yml`-Dateien geprüft. Crowdin v2/v3 besitzen sämtliche hier verwendeten Inputs; kein Übersetzungs-Push wurde ausgeführt. GoReleaser ist jetzt exakt auf 2.20.1 statt `latest` festgelegt, und die Konfiguration deklariert `version: 2` sowie `archives.formats`. Die alte Docker-/Manifest-Konfiguration ist noch unterstützt, aber als deprecated markiert; ihre spätere Umstellung auf `dockers_v2` ist separat zu testen.

## C. Zurückgestellte Upgrades und konkrete Grenzen

Ein hier genanntes Prüfdefizit ist kein bewiesener API-Fehler. Wo keine verifizierte Inkompatibilität vorliegt, wird ausdrücklich nur fehlende Abnahme genannt. Aktuelle direkte Pakete ohne höhere passende Version bleiben entsprechend der Tabelle unverändert.

| Paket / Familie                  | Beibehalten | Gewünschter neuer Zweig    | Grund und späterer Lösungsweg                                                                                                                                                                                         |
| -------------------------------- | ----------- | -------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Vite                             | 7.3.8       | 8.3.8                      | Nativer Rolldown/Oxc-Import endet auf diesem Host mit SIGILL, auch als Minimalimport. Bundler-/CPU-Kompatibilität zuerst isoliert klären, dann beide Apps und Devserver abnehmen. Vite 7 nutzt weiter Rollup/esbuild. |
| Tiptap, alle direkten Extensions | 2.27.3      | 3.33.1                     | v3 ändert Tabellen-/Menüimporte, Tippy/Floating UI und Editorverhalten. Bestehender Composer, Mentions und Tabellen benötigen eine gemeinsame Migration und Browserabnahme; 2.x-Sicherheitsbefund bleibt offen.       |
| Tailwind CSS                     | 3.4.19      | 4.3.5                      | Aktuelle JS-Konfiguration mit HSL-CSS-Variablen, Safelist und Plugins, bestehende Utility-Semantik und Browserziele verlangen eine gezielte CSS-Migration samt visueller Abnahme. Kein CSS-Redesign vorgenommen.      |
| tailwind-merge                   | 2.6.1       | 3.7.0                      | Zusammen mit Tailwind 4 migrieren; nicht unabhängig auf eine andere Utility-Semantik umstellen.                                                                                                                       |
| Zod                              | 3.25.76     | 4.6.7                      | @vee-validate/zod 4.15.1 verlangt ausdrücklich Zod ^3.24.0. v4 ist kein kompatibler Override. Adapter/Formulare gemeinsam migrieren, Validierungsfehler und alle Schematests prüfen. High-Advisory bleibt offen.      |
| @vee-validate/zod / vee-validate | 4.15.1      | Registry latest 4.15.1     | Bereits neueste stabile Adapterlinie; blockiert Zod 4. Keine Vorabversion eingesetzt.                                                                                                                                 |
| TanStack Table                   | 8.21.3      | 9.3.1                      | Keine nachgewiesene Inkompatibilität geprüft; Major-Abnahme für bestehende Tabellen, Sortierung, Pagination und Slots fehlt. Separater Tabellen-Migrationsschritt.                                                    |
| Pinia                            | 2.3.1       | 4.1.0                      | Neue Peer-Anforderungen unter anderem @vue/devtools-api ^8.2.1 und TypeScript >=5.6.0; vollständige Store-/Devtools-Migration nicht abgenommen. Kein Parallel-State eingeführt.                                       |
| VueUse                           | 12.8.2      | 15.0.0                     | Mehrere Major-Wechsel und breit genutzte Storage-, Sichtbarkeits-, Resize- und Focus-Composables. Bestehende Liste/Scroll-/Dark-Mode-Interaktionen benötigen Browserabnahme. Kein konkreter API-Bruch behauptet.      |
| date-fns                         | 3.6.0       | 4.4.0                      | Datums-/Geschäftszeiten-Semantik in Main und Widget erfordert gezielte Zeitzonen-/DST-Abnahme. Keine nachgewiesene Inkompatibilität behauptet.                                                                        |
| lucide-vue-next                  | 0.577.0     | 1.0.0                      | 0.x-Zweig ist deprecated; v1-Icon-/Alias-/Darstellungsabnahme steht aus. Nächster kleiner separater UI-Schritt, kein Sicherheitsfix durch Major-Override erzwungen.                                                   |
| vue-picture-cropper              | 0.7.0       | 1.0.0                      | Cropper-/Upload-Interaktionen nicht im Browser geprüft. Major zurückgestellt.                                                                                                                                         |
| vue-sonner                       | 1.3.2       | 2.0.9                      | Toast-Interaktion, Styles und Accessibility nicht mit v2 abgenommen.                                                                                                                                                  |
| vue3-emoji-picker                | 1.1.8       | 2.1.0                      | Composer-/Widget-Auswahl und Styles benötigen Browserabnahme.                                                                                                                                                         |
| vue-router                       | 4.6.4       | 5.5.0                      | Route-/Guard-/Deep-Link-Migration nicht abgenommen; keine konkrete Inkompatibilität behauptet.                                                                                                                        |
| @formkit/auto-animate            | 0.8.2       | 0.10.2                     | Vor-1.0-Minor kann brechen; Animationen sind automatisiert nicht visuell abgedeckt.                                                                                                                                   |
| Cypress                          | 15.21.1     | 16.3.0                     | Browser startet wegen fehlender libatk-Bibliothek nicht. Ohne ausführbare E2E-Abnahme kein zusätzlicher Major-Wechsel.                                                                                                |
| Vitest                           | 4.1.12      | 5.1.0                      | Aktueller 4.x-Patch erfolgreich geprüft. v5-Migration nicht zusätzlich durchgeführt; kein konkreter API-Bruch behauptet.                                                                                              |
| start-server-and-test            | 2.1.5       | 3.0.13                     | Start-/Ready-/Shutdown-Verhalten mit Cypress lokal nicht abnehmbar.                                                                                                                                                   |
| go-imap/v2                       | beta.3      | beta.8                     | Keine stabile v2 verfügbar. Neue Vorabversion nicht automatisch übernommen; reale SMTP/IMAP-/Threading-Abnahme separat.                                                                                               |
| volatiletech/null/v9             | 9.0.0       | 9.0.1                      | Tatsächlicher go-get-Fehler: neues Tag deklariert Modulpfad github.com/aarondl/null/v9. Ein Importwechsel beträfe zahlreiche produktive und Testdateien; gemeinsame Pfadmigration separat.                            |
| codemirror                       | 6.0.2       | Registry latest 6.0.2      | Höhere Nummern im Versionsarchiv werden nicht blind gegenüber dem offiziellen latest-Tag bevorzugt.                                                                                                                   |
| vuedraggable                     | 4.1.0       | passende Vue-3-Linie 4.1.0 | Registry latest zeigt 2.24.3 für Vue 2; kein Downgrade.                                                                                                                                                               |

Bereits aktuelle, unveränderte direkte Pakete: @codemirror/theme-one-dark, @radix-icons/vue, class-variance-authority, clsx, highlight.js, lowlight, mitt, radix-vue, vue-easy-lightbox und tailwindcss-animate. Reka wurde aktualisiert; Radix Vue wird nicht als Nebenprodukt vollständig ersetzt, da beide Bibliotheken bestehende Komponenten versorgen. Unveränderte Go-Module ohne neuere Version stehen vollständig in den Tabellen. Seltene Releases sind allein kein Beleg für aufgegebene Wartung; für SSRF-, SQL-, Mail-, Session- und Sanitizer-Abhängigkeiten wird keine unbelegte EOL-Aussage gemacht.

## D. Geänderte Dateien

- `.github/workflows/crowdin.yml`: Aktuelle kompatible Action-Pins und zentrale Toolchain-/CI-Einstellungen.
- `.github/workflows/frontend-ci.yml`: Aktuelle kompatible Action-Pins und zentrale Toolchain-/CI-Einstellungen.
- `.github/workflows/go-build-cache.yml`: Aktuelle kompatible Action-Pins und zentrale Toolchain-/CI-Einstellungen.
- `.github/workflows/go.yml`: Aktuelle kompatible Action-Pins und zentrale Toolchain-/CI-Einstellungen.
- `.github/workflows/issues.yml`: Aktuelle kompatible Action-Pins und zentrale Toolchain-/CI-Einstellungen.
- `.github/workflows/release.yml`: Aktuelle kompatible Action-Pins und zentrale Toolchain-/CI-Einstellungen.
- `.gitignore`: Nur Upgradebericht in bisher ignoriertem docs-Verzeichnis freigeben; pnpm-Store ignorieren.
- `.goreleaser.yaml`: v2-Schema, formats, deterministischer Moduldownload/verify.
- `Dockerfile`: Isoliert geprüftes Alpine 3.24.4.
- `Makefile`: Vorhandenes pnpm statt ungebundenem npx pnpm im Testziel.
- `frontend/.eslintrc.cjs`: Durch Flat Config ersetzt.
- `frontend/apps/main/src/i18n.test.js`: Nur obsoleten eslint-env-Kommentar entfernt.
- `frontend/apps/main/src/stores/conversation.i18n.test.js`: Nur obsoleten eslint-env-Kommentar entfernt.
- `frontend/package.json`: Geprüfte Paketziele, Engines, pnpm-Pin, nicht mutierendes lint und gesondertes lint:fix.
- `frontend/pnpm-lock.yaml`: Mit pnpm 12 regeneriert und frozen installiert.
- `frontend/vite.config.js`: Vite-7-kompatible Sass-Konfiguration, bisherige Browserziele ausdrücklich erhalten; vorhandene Formatierung angepasst.
- `go.mod`: Go-Toolchain und kontrollierte Modulziele.
- `go.sum`: Verifizierte Modulprüfsummen.
- `.node-version`: Zentrale, exakt getestete Node-LTS-Version.
- `frontend/eslint.config.js`: ESLint-10-/Vue-/Cypress-Flat-Config mit Browser-/Node-Globals.
- `frontend/pnpm-workspace.yaml`: Explizite Install-Script-Allowlist und kompatible Sicherheits-Overrides.
- `docs/DEPENDENCY_UPGRADE_REPORT.md`: Audit, Ergebnisse, Grenzen und Rolloutplan.

Die ursprünglich lokal vorhandene, nicht Git-getrackte `serviceMailWorkflow.test.js` verschwand während der Arbeit. Der Benutzer bestätigte im Chat ausdrücklich: „Beabsichtigt, ignorieren“. Die Entfernung ist keine Upgradeänderung und wurde nicht rückgängig gemacht. Keine Workflow-Exporte, SQL-Fixtures, produktiven Go-Funktionen, Berechtigungsregeln, Composer-Implementierungen oder fremden Compose-Dateien wurden geändert. Kein Commit, Push oder Staging ausgeführt. Der bereits vorhandene ignorierte npm-Lockfile bleibt unverändert und wird nicht als Paketquelle verwendet; es wurde kein zusätzliches npm-/Yarn-Lockfile eingecheckt.

## E. Kompatibilitätsanpassungen

- Vue und seine aufgelösten Compilerpakete verwenden dieselbe Version 3.5.45. Composition API, Teleport und Slots bleiben im Anwendungscode unverändert.
- Alle direkten Tiptap-Pakete sind exakt 2.27.3, einschließlich PM, Suggestion, Mentions und Tabellen. Keine gemischte Tiptap-Major-Linie.
- Vue I18n 11 verwendet die vorhandene Composition-Konfiguration (`legacy: false`). Kein Wechsel von gespeicherten/API-Werten auf übersetzte Werte. Die vorhandenen i18n-Tests bestanden.
- Main-/Widget-Modi, Roots, Aliase, separate Ausgabeverzeichnisse, Assets, Proxy, allowedHosts und CORS-Verhalten bleiben erhalten. Vite 7 bekommt ausdrücklich die Browserziele von Vite 6; kein unbeabsichtigtes Anheben auf Vites neue Default-Baseline. Die bereits moderne Sass-API benötigt kein `api: modern` mehr.
- ESLint 10 nutzt native Flat Config. Der alte Rushstack-Patch und die alte Vue-Prettier-Brücke entfallen; Prettier-Regelabschaltung bleibt erhalten. lint verändert keine Dateien mehr, lint:fix ist explizit. Bestehende Regeln werden nicht mit einer pauschalen Ausnahme deaktiviert.
- pnpm 12 erzeugt einen YAML-Stream mit zwei Dokumenten: Bootstrap-/Paketmanager-Lock und App-Lock, jeweils Version 9.0. Das ist vom Paketmanager erzeugt, keine manuelle Lockfile-Zusammenführung. `autoInstallPeers` bleibt aktiv. Peer-Ranges wurden aus Registry-Metadaten geprüft; keine Peer-Warnung durch erzwungenen fremden Major unterdrückt. Die Install-Script-Allowlist erlaubt Cypress, esbuild, @parcel/watcher und vue-demi. Eng begrenzte `minimumReleaseAgeExclude`-Einträge für konkret geprüfte Releases werden vom pnpm-Bootstrap berücksichtigt; kein globales Abschalten des Release-Alters.
- go get erhielt explizite Modul@Version-Ziele; `go mod tidy` wurde danach ausgeführt. Keine globale `go get -u ./...`-Operation. CI lädt/verifiziert eingecheckte Versionen und aktualisiert keine Dependencies. `go.mod` fließt auch in Release-Cache-Schlüssel ein.
- Das App-Dockerfile baut weiterhin aus einem vorbereiteten statischen Binary. Der isolierte Test verwendete nur Dockerfile und temporäre Binärdatei, keine Secrets, Uploads oder produktiven Konfigurationen. Das ist kein vollständiger App-/Schema-/Staging-Test. Compose-Startbefehle führen schon bisher Installation/Upgrade aus; deshalb muss ein späterer Rollout Schema und Binary gemeinsam behandeln.
- CI umfasst Backendtests mit PostgreSQL 17, Frontendtests, beide Builds und einen verbindlichen Frontend-Lintschritt. Die bekannten Lintfehler blockieren diesen Schritt weiterhin; CI ist deshalb nicht pauschal als grün freigegeben. Neue Jobs werden nicht vervielfacht. GitHub-hosted ubuntu-latest muss die Node-24-Action-Runtime unterstützen; die Remote-Workflows wurden nicht ausgeführt.

## F. Testergebnisse

Die Rohlogs liegen lokal unter `/tmp/libredesk-upgrade/` und enthalten ausschließlich Test-/Werkzeugausgaben; sie sind nicht für einen Commit vorgesehen. Shell-Toolchains wurden über PATH auf die temporäre Node-Installation und pnpm-Installation ausgerichtet. Zugangsdaten der eigens erstellten Testinstanz werden hier bewusst nicht ausgeschrieben. `TEST_DSN` bezeichnet ausschließlich 127.0.0.1:55437; Learning verwendet dort ausschließlich `/learning_test` und wird im Test zusätzlich hart darauf geprüft.

| Prüfung                    | Befehl / Umgebungswahl                                                                                                                                                        | Ergebnis                                                                                        |
| -------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| Frozen-Install             | `cd frontend; pnpm install --frozen-lockfile` mit pnpm 12.18.0 / Node 26.13.0                                                                                                 | Erfolg, finale Overrides enthalten, keine Lockfileänderung                                      |
| Vitest Baseline            | `node node_modules/vitest/vitest.mjs run --maxWorkers=2` unter ursprünglichem Node 24.15.0                                                                                    | 1.087 bestanden; 1 zusätzliche Suite fehlgeschlagen wegen fehlender lokaler n8n-Exporte         |
| Vitest final               | `pnpm test:run --maxWorkers=1 --no-file-parallelism`, `NODE_OPTIONS=--max-old-space-size=768`                                                                                 | 73 Dateien, 1.087 Tests bestanden; bewusst vom Benutzer entfernte lokale Vertragssuite fehlt    |
| Signaturtest einzeln       | `timeout 75s node node_modules/vitest/vitest.mjs run apps/main/src/features/admin/inbox/EmailInboxForm.signature.test.js --maxWorkers=1 --reporter=verbose`                   | Beide Public-/Personal-Tests bestanden                                                          |
| Main-Build                 | `pnpm build:main`, `NODE_OPTIONS=--max-old-space-size=2560`                                                                                                                   | Erfolgreich, 5.719 Module, 48,43 s                                                              |
| Widget-Build               | `pnpm build:widget`                                                                                                                                                           | Erfolgreich, finale Sicherheits-Overrides enthalten                                             |
| Frontend-Lint              | `pnpm exec eslint . -f json`                                                                                                                                                  | Baseline 285 Fehler; final 287; Details unten                                                   |
| Prettier geänderte Tests   | `pnpm exec prettier --check apps/main/src/i18n.test.js apps/main/src/stores/conversation.i18n.test.js`                                                                        | Zwei bereits bestehende Formatbefunde; keine breite Test-Neuformatierung                        |
| Prettier Konfiguration     | `pnpm exec prettier --check package.json eslint.config.js pnpm-workspace.yaml vite.config.js`                                                                                 | Erfolg                                                                                          |
| Go Baseline                | `go test -json ./...` mit isolierter PostgreSQL-17-DSN                                                                                                                        | 1.550 Tests/Subtests bestanden, 28 Skips, keine Fehler                                          |
| Go nach direkten Updates   | `go test -p 1 -json ./...`, `GOMAXPROCS=2`, `GOMEMLIMIT=768MiB`, isolierte DSN                                                                                                | 1.550 bestanden, dieselben 28 Skips, keine Fehler                                               |
| Go nach Sicherheitsupdates | `go test -p 1 -json ./...`, `GOMAXPROCS=2`, `GOMEMLIMIT=512MiB`, isolierte DSN auf PostgreSQL 17.11                                                                           | 1.550 Tests/Subtests bestanden, dieselben 28 Skips, keine Fehler (Exit 0)                       |
| Learning mit pgvector      | `go test -p 1 -json ./internal/learning` mit `LIBREDESK_LEARNING_TEST_DSN` ausschließlich auf `/learning_test`                                                                | 63 bestanden, 2 Fehler wegen fehlender Artefakte; keine Skips                                   |
| Learning-Baselinevergleich | `go test -p 1 -modfile=/tmp/libredesk-upgrade/baseline.mod -json ./internal/learning -run 'TestPostgresRetrievalAndV9PublicationContract                                      | TestPostgresV10PreparedDraftContract'` mit ursprünglichen Modulen                               | Dieselben 2 fehlenden Artefakte reproduziert |
| Race Detector              | `go test -p 1 -race -json ./internal/authz ./internal/conversation ./internal/inbox ./internal/media ./internal/ws ./internal/httputil ./internal/aiagent` mit isolierter DSN | 516 Tests/Subtests bestanden, 1 bestehender optionaler Export-Skip, keine Race-Befunde (Exit 0) |
| go vet                     | `go vet -p 1 ./...`                                                                                                                                                           | Dieselben 2 vorhandenen Copylock-Befunde, keine weiteren                                        |
| gofmt                      | Keine produktive Go-Datei geändert; finaler Diff enthält keine Go-Quelldatei                                                                                                  | Keine Go-Formatänderung erforderlich                                                            |
| Modulkonsistenz            | `go mod verify`; `go mod tidy -diff`                                                                                                                                          | Verify erfolgreich; tidy-Diff leer                                                              |
| Go-Build                   | `CGO_ENABLED=0 go build -p 1 -o /tmp/libredesk-upgrade/image-context/libredesk ./cmd`                                                                                         | Erfolgreich; Binary nur unter /tmp                                                              |
| Runtime-Image              | `docker build -t libredesk-dependency-audit:alpine-3.24.4 /tmp/libredesk-upgrade/image-context`                                                                               | Erfolg mit Alpine 3.24.4 und aktuellen CA-/TZ-Paketen                                           |
| Runtime-Smoke              | `docker run --rm --network none --read-only --cap-drop ALL libredesk-dependency-audit:alpine-3.24.4 ./libredesk --version`                                                    | Exit 0; keine DB/Redis-/App-Verbindung                                                          |
| GoReleaser                 | `/tmp/libredesk-upgrade/goreleaser/goreleaser check` mit 2.20.1                                                                                                               | Erfolg; alte Docker-Konfiguration als deprecated gemeldet                                       |
| npm-Audit                  | `pnpm audit --json` auf ursprünglichem und finalem Lockfile                                                                                                                   | 121 → 6 gemeldete Befunde; Details unten                                                        |
| govulncheck                | `govulncheck -json ./...`, `GOFLAGS=-p=1`, begrenzter Speicher                                                                                                                | Exit 0, keine erreichbare Schwachstelle; 1 Modulhinweis x/crypto                                |
| Cypress                    | `pnpm exec cypress verify`                                                                                                                                                    | Nicht ausführbar: libatk-1.0.so.0 fehlt; kein Zugriff auf laufende App                          |
| Container-CVE-Scan         | `docker scout version` / Prüfung auf trivy                                                                                                                                    | Kein Scanner vorhanden; kein Image-CVE-Freibrief                                                |
| YAML / Git                 | YAML-Parser für Workflows und Compose; `git diff --check`                                                                                                                     | Erfolg                                                                                          |

### Bekannte und neue Lintdiagnosen

Baseline: 148 no-unused-vars, 75 cypress/unsafe-to-chain-command, 43 vue/multi-word-component-names, 14 no-undef, 4 no-useless-escape, 1 cypress/no-unnecessary-waiting. ESLint 10 final meldet 287 Fehler. Browser-/Node-Globals beheben falsche Umgebungsdiagnosen; die neuere Konfiguration erkennt zusätzlich vorhandene ungenutzte Bindungen, zwei `vue/no-ref-as-operand`-Stellen (ActionBox.vue, RuleBox.vue) und zwei `no-useless-assignment`-Stellen (ReplyBox.vue, Widget useBusinessHours.js). Diese Produktdateien wurden nicht geändert. Drei vorhandene `attempt`-Referenzen im Widget-WebSocket bleiben als no-undef sichtbar. Zwei nicht mehr unterstützte eslint-env-Kommentare wurden ohne Änderung des Testinhalts entfernt. Es wird weder ein sauberer Gesamtlint noch eine erfolgreiche Remote-CI behauptet; die verbleibende Lintbereinigung ist offen.

### Skips und fehlende Artefakte

26 der 28 Skips der normalen Go-Suite sind Learning-Tests ohne deren gesonderte DSN. Diese wurden zusätzlich mit pgvector 0.8.7 / PostgreSQL 17.11 ausgeführt (63 bestanden, 2 fehlende Artefakte). Die anderen beiden Skips sind der vorhandene `TestValidateAgentRequestEmptyRolesSlice` und der optionale `TestServiceWorkflowRegistryQueryMatchesLocalExport`. Keine neue Skip-/Only-Markierung eingeführt. Die erforderliche SQL-Fixture bleibt erhalten; ihr Integrationstest läuft gegen die isolierte Datenbank.

Die zwei Learning-Fehler verlangen `internal/learning/artifacts/AI_Learning_Processor_v9.json` und `v10.json`. Es wurden keine Ersatz-Workflows erfunden und keine Tests abgeschwächt. Vorhandene Exporte müssen später sicher, frei von Credentials und mit überprüfter Provenienz bereitgestellt oder die Vertragsfixtures bewusst fachlich gelöst werden.

### Ausführungsprobleme und Grenzen

Anfängliche Sandboxläufe konnten keine lokalen HTTP-/Redis-Testserver starten. Die autorisierten Tests wurden deshalb außerhalb der Netzwerksandbox mit expliziter isolierter DSN wiederholt. Parallele Vollbuilds/Scans verursachten Speicherabbrüche (Exit 137); ein zu kleiner 1-GB-Node-Heap brach ebenfalls ab. Ein Go-Build scheiterte am temporären Speicherlimit. `/tmp` ist tmpfs; der neue Go-Cache wurde anschließend in das bereits ignorierte `frontend/node_modules/.cache/dependency-upgrade-go-cache` verschoben. Danach waren Main-Build, statischer Backend-Build und Scan erfolgreich. Kein vorhandener Cache, Backup oder produktives Volume wurde gelöscht; lediglich ein von dieser Arbeit zurückgelassener temporärer go-build-Ordner wurde bereinigt.

Vite 8 bleibt wegen reproduzierbarem SIGILL zurückgestellt. Der isolierte Docker-Build wurde beim ersten Versuch zu früh vor der fertigen Binärdatei gestartet und scheiterte am fehlenden Binary; der abschließende Build mit vorhandenem Binary bestand. Node 26 meldet in jsdom-Tests die ExperimentalWarning zu globalem localStorage ohne --localstorage-file; die Suite besteht trotzdem. Builds melden die bereits zur Laufzeit aufgelöste fonts.css, entfernte PURE-Kommentare aus VueUse und einen großen Main-Chunk. Keine Grenzwerte zur Verdeckung von Warnungen angehoben. Keine visuelle Browser-, vollständige E-Mail-/SSO-/S3- oder produktive Widget-End-to-End-Abnahme durchgeführt. Ein erfolgreiches Build beweist diese Funktionen nicht.

## G. Sicherheitsbewertung

| Lockfile | Kritisch | Hoch | Mittel | Niedrig | Summe |
| -------- | -------- | ---- | ------ | ------- | ----- |
| Vorher   | 1        | 61   | 45     | 14      | 121   |
| Nachher  | 0        | 2    | 4      | 0       | 6     |

Das sind Meldungen des Paketmanager-Audits einschließlich transitiver Dev-/Optional-Abhängigkeiten, nicht die Anzahl praktisch ausnutzbarer Anwendungslücken. Behoben wurden unter anderem Befunde in Axios/Follow-Redirects, Vue I18n/Intlify, Vue-Renderer, Vitest/Mocker, Vite, ProseMirror, MapLibre sowie durch kompatible Overrides in brace-expansion, qs, tmp, markdown-it, linkify-it, systeminformation, nanoid, immutable, postcss-selector-parser und source-map-js. Der zuvor kritische MapLibre-Befund ist im neuen Graph nicht mehr enthalten. Keine Security-Policy oder Zugriffskontrolle wurde abgeschaltet.

Offene npm-Meldungen:

| Modul                | Schwere  | Betroffener Pfad / Abnahmegrenze                                                         | Advisory                                                                 |
| -------------------- | -------- | ---------------------------------------------------------------------------------------- | ------------------------------------------------------------------------ |
| decode-uri-component | moderate | . → @vitejs/plugin-vue → vite → stylus → css → source-map-resolve → decode-uri-component | [GHSA-vcc3-ghjq-m6fr](https://github.com/advisories/GHSA-vcc3-ghjq-m6fr) |
| @tiptap/core         | moderate | . → @tiptap/vue-3 → @tiptap/extension-bubble-menu → @tiptap/core                         | [GHSA-cp6q-959q-f8rh](https://github.com/advisories/GHSA-cp6q-959q-f8rh) |
| zod                  | high     | . → @vee-validate/zod → zod                                                              | [GHSA-mrhx-c7xc-m3w5](https://github.com/advisories/GHSA-mrhx-c7xc-m3w5) |
| decode-uri-component | high     | . → @vitejs/plugin-vue → vite → stylus → css → source-map-resolve → decode-uri-component | [GHSA-qr28-fpjm-wq7f](https://github.com/advisories/GHSA-qr28-fpjm-wq7f) |
| glob                 | moderate | . → @tailwindcss/typography → tailwindcss → sucrase → glob                               | [GHSA-3g3m-2fjr-6pw4](https://github.com/advisories/GHSA-3g3m-2fjr-6pw4) |
| minimatch            | moderate | . → @eslint/js → eslint → @eslint/config-array → minimatch                               | [GHSA-g3hv-75wg-m249](https://github.com/advisories/GHSA-g3hv-75wg-m249) |

Zod 4 und Tiptap 3 werden nicht in inkompatible Peer-/Editorgraphen gezwungen. Die übrigen Meldungen verlangen bei decode-uri-component eine 0.x-Minor-Migration beziehungsweise bei glob/minimatch einen Major-Sprung außerhalb des verwendeten Buildzweigs. Diese Pakete sind in den gemeldeten Pfaden Dev-/Build-Werkzeuge, während Zod und Tiptap Anwendungsabhängigkeiten sind. Daraus folgt keine pauschale Unausnutzbarkeit; spätere Migration und erneuter Audit bleiben erforderlich.

`golang.org/x/image` wurde von 0.41.0 auf 0.48.0 gebracht; die im Zwischenlauf gefundenen GO-2026-4961, -5061, -5062, -5066 und -6222 sind danach nicht mehr im Scan vorhanden. Der vollständige finale govulncheck endet mit Exit 0 und meldet lediglich GO-2026-5932 auf Modulebene für x/crypto 0.58.0, ohne Paket-/Symbol-Aufrufpfad. Das Advisory nennt `v0.58.1-0.20261009181718-2f1f834330b2` als Fix; der offiziell ermittelte neueste stabile Modulstand war 0.58.0. Keine Vorabversion automatisch eingeführt und keine Entwarnung für den gesamten Modulinhalt behauptet. [Go-Advisory](https://pkg.go.dev/vuln/GO-2026-5932).

lucide-vue-next 0.577.0 ist deprecated. Verbliebene deprecated transitive Werkzeuge umfassen glob, inflight und source-map-resolve. Für Container-OS-Pakete und Python-Transitives wurde kein vollständiger CVE-Scan durchgeführt, weil kein entsprechender Scanner vorhanden war. Externe Dienste erhalten keinen Sicherheitsfreibrief aus dem LibreDesk-Scan.

### Externe Services: tatsächlich inventarisiert, ausschließlich read-only

Gelesen wurden nur Compose-Struktur, Image-/Versionslabels, installierte Python-Distributionen und harmlose Versionsbefehle. Keine Umgebungswerte mit Credentials, Datenbankinhalte, Workflowdaten oder Credential-Exporte wurden übernommen. `/opt/n8n` und `/opt/anschreiben_n8n` wurden nicht geändert.

| Service / Komponente          | Tatsächlich ermittelt                                                | Neueste stabil / Ziel                         | Kompatibilität und Vorgehen                                                                                                                                                                                                   |
| ----------------------------- | -------------------------------------------------------------------- | --------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| n8n                           | Image 2.25.7, Node 24.15.0                                           | npm latest=stable 2.47.4, Node >=24           | Separates Image-Pinning und Staging; beta/next 2.48.1 nicht automatisch verwenden. Community-Nodes-Manifest am Standardpfad enthält keine Dependencies; das beweist nicht die Abwesenheit beliebiger extern montierter Nodes. |
| n8n PostgreSQL                | 16.14, Tag postgres:16                                               | 16.15 als Minor; 18.6 nur eigener Major-Plan  | Keine neue Major auf alte Volumes starten. Dump/Restore-/pg_upgrade-Probe getrennt.                                                                                                                                           |
| Anonymizer Python             | 3.12.13, python:3.12-slim                                            | zunächst 3.12.16; neuester Gesamtzweig 3.15.2 | spaCy-Modell-/Wheel-/PII-Abnahme nötig, kein Python-Major im fremden Projekt.                                                                                                                                                 |
| Anonymizer FastAPI            | 0.139.2, requirements ungebunden                                     | 0.162.4                                       | Schemas, Fehler-/503-Verhalten und unveränderte Privacy-API testen; danach exakt pinnen.                                                                                                                                      |
| Anonymizer Uvicorn            | 0.51.0                                                               | 0.54.0                                        | Timeouts, Worker, Requestlimits und Shutdown isoliert prüfen.                                                                                                                                                                 |
| Anonymizer spaCy              | 3.8.14                                                               | 3.8.18                                        | Passendes de_core_news_lg verwenden, Erkennungs-/Anonymisierungsregression und Ressourcenverbrauch testen.                                                                                                                    |
| Anonymizer Modell             | de_core_news_lg 3.8.0                                                | zur geprüften spaCy-Linie passendes Modell    | Modell wird aktuell ungebunden beim Build heruntergeladen. Version und Downloadhash künftig festlegen, keine ungeprüfte automatische Modellmigration.                                                                         |
| PDF-Service Python            | 3.12.15                                                              | zunächst 3.12.16                              | Unveränderte Formular-/PDF-Ausgabe und Bibliotheks-Wheels prüfen.                                                                                                                                                             |
| PDF-Service FastAPI / Uvicorn | 0.143.0 / 0.54.0                                                     | 0.162.4 / 0.54.0                              | Fremdes Projekt, isolierte API-Regression.                                                                                                                                                                                    |
| PDF-Service PyMuPDF           | 1.28.2; Docker erlaubt >=1.26,<2                                     | Registry 2.1; erst geprüfter 1.x-Patch        | Explizite <2-Grenze und PDF-Ausgabeformat verhindern blinden Major-Wechsel. Separat Rendering, Fonts und Dateisicherheit testen.                                                                                              |
| Portainer                     | latest-Tag, Imagecommit ece7e56; Semver nicht zuverlässig abgeleitet | Release 2.45.3                                | Vor Upgrade tatsächliche Version/Edition und Datenformat offiziell bestimmen, Management-Volume sichern; keine Änderung durchgeführt.                                                                                         |
| App PostgreSQL / Redis        | 17.10 / 7.4.9                                                        | zunächst 17.11 / 7.4.12                       | Wartungszweige erhalten; Redis 8.10.3 nur eigener Test-/Recoveryplan.                                                                                                                                                         |

n8n 2.25.7 liegt im betroffenen Bereich des Hersteller-Advisorys [GHSA-9p7v-45m2-wxgj](https://github.com/n8n-io/n8n/security/advisories/GHSA-9p7v-45m2-wxgj) zu veralteten Code-Nodes und Ausführung im Hauptprozess. Die praktische Relevanz hängt von verwendeten Node-Typen und Workflowrechten ab; produktive Workflows wurden hierfür nicht ausgelesen. Die geprüfte Wartungslinie 2.47.4 enthält die genannten 2.x-Fixstände. Die Python-3.12-Instanzen liegen vor dem [3.12.16-Sicherheitsrelease](https://www.python.org/downloads/release/python-31216/), das unter anderem CA-/tarfile-/XML-Fixes enthält. Diese Befunde begründen einen priorisierten separaten Rollout, keinen Eingriff in die laufenden Dienste.

## H. Späterer Rollout und Recovery

1. **Build-/Toolchain-Übernahme:** Changes im aktuellen Branch prüfen, Node 26.13.0 aus `.node-version` aktivieren, pnpm 12.18.0 aus packageManager installieren, Go 1.27.4 verwenden. Mit unverändertem Lockfile installieren, Module verifizieren und CI-Prüfungen ausführen. Zuerst die verbleibenden Lintfehler und zwei fehlenden Learning-Artefakte fachlich klären; fehlende Cypress-Systembibliotheken in einer isolierten Umgebung bereitstellen. Kein erzwungener grüner Status durch Skip oder `continue-on-error`.
2. **App-Staging vor Deployment:** Main, Widget und Go gemeinsam bauen, Assets mit bestehendem stuffbin-Schritt einbetten, Image isoliert erstellen und eine Kopie von Schema/Daten im Staging migrieren. ACL-/UUID-/private Inbox-/Attachment-/WebSocket-Tests sowie Sidebar-Scroll, Filter, Spam, Service-Mail-KI-Sperren, Signaturen, Fullscreen-Composer, KI-/Feedback-Provenienz und Widget im Browser und über reale Test-Mailwege abnehmen. Releaseplattformen einschließlich ARM/BSD/Windows wurden nicht vollständig neu gebaut; vollständige Release-Matrix vor Freigabe prüfen.
3. **App-Deployment später separat:** Erst nach erfolgreichem Staging und geprüftem Restore. Bestehende Compose-Startbefehle führen `--install` und `--upgrade` aus; App-Version, Migrationen und Image müssen gemeinsam freigegeben werden. Aktuell wurde kein App-Container ersetzt oder gestartet.
4. **PostgreSQL-/Redis-Minors:** In eigenem Wartungsfenster zunächst Postgres 17.11 beziehungsweise n8n-PG16.15 und Redis 7.4.12 mit expliziten Image-Pins/Digests vorbereiten. Vorher release notes, Erweiterungen und verwendete Client-/ACL-Konfiguration prüfen. Bestehende Namen, Ports, Netzwerke, Healthchecks und Volumes erhalten. Healthchecks und reale Session-/Cache-/Queue-Funktionen vor und nach Rollout vergleichen. Keine Image-Tags jetzt blind hochgesetzt.
5. **PostgreSQL-Major separat:** PostgreSQL 18.6 ist aktuell stabil, 19 nur Vorablinie. Erweiterungen inventarisieren; passende vollständige Backups inklusive Rollen, Extensions, Schema und Nutzdaten erstellen und Restore messen. Auf neuen Volumes mit dump/restore oder geprüftem pg_upgrade-Prozess testen, Zeichensätze/Collations/Indizes und Anwendungs-/n8n-Migrationen abnehmen. Umschalt-/Downtimefenster festlegen und nach Abnahme erst umschalten. Vorhandene PG17-/PG16-Volumes unverändert für Recovery aufbewahren.
6. **Redis-Major separat:** RDB/AOF-Format, ACLs, Maxmemory/Eviction und Clientprotokoll prüfen. Kopie in isolierter Redis-8.10.3-Instanz laden, Sessions/TTLs, Fastcache, OTP und gegebenenfalls n8n-Queue mit realistischen Testjobs prüfen. Keine alten Instanzen mit von neuer Major geschriebenen Dateien starten, bevor Rückwärtslesbarkeit belegt ist. Bei Sessionverlust kontrollierte Neuanmeldung planen; keine Wiederverwendbarkeit alter Sessions behaupten.
7. **n8n separat:** Laufende Ausführungen/Queueverhalten erfassen und einen späteren Drain planen. Datenbank und Workflow-/Credential-Metadaten sowie Encryption-Key geschützt gemeinsam sichern; keine Secretwerte ins Repo. Kopie mit 2.47.4 und denselben Postgres-/Netzwerk-/Endpoint-Einstellungen starten. Node-Typen, Community-Nodes, Taskrunner, Webhooks, Expressions, Execute-SQL-Parameter, Service-Mail-Ausschluss und KI-/Feedbackverträge mit isolierten API-Zielen abnehmen. Keine produktiven Exporte bearbeiten/importieren. Major 3.x nur als gesondertes Vorhaben nach eindeutiger stabiler Release- und Migrationseignung.
8. **Anonymizer/PDF-Service separat:** Exakte Python-/Paket-/Modellversionen und Hashes im jeweiligen fremden Projekt erst nach Freigabe pinnen. FastAPI-Request-/Response-Verträge, Fail-closed-Privacy, PII-Beispiele, Telefonnummern/Adressen, Timeouts und CPU/RAM testen. spaCy-Modelländerungen anhand eines bekannten Korpus vergleichen. PDF-Seiten, Schriften, Layout und Dateipfade gegen vorhandene Referenzen prüfen. Endpoints und Credentials erhalten.
9. **Backups und Restore:** Vor jedem zustandsbehafteten Rollout konsistente, zugriffsgeschützte DB-, Upload-, Redis-, n8n- und Konfigurationssicherungen erstellen; Restore auf getrenntem Speicher tatsächlich durchführen. Aufbewahrungsfrist und Verantwortliche festlegen. Bestehende Backups/Volumes nicht bereinigen.
10. **Rollback:** Das vorige geprüfte Binary/Image und den dazugehörigen Schema-/Datenstand dokumentieren. Keine ungeprüfte Binär-/Schema-Rückwärtskompatibilität annehmen. Bei Datenmigrationen im Problemfall getesteten Restore auf passenden separaten Volumes und passender alter App-/Serviceversion verwenden. Neue Schreibzugriffe seit Umschaltung bewusst abgleichen, nicht durch simples Image-Downgrade verlieren. Ein erfolgreicher Test mit `--version` ersetzt diese Probe nicht.

## Quellen und Abnahmestatus

Live-Metadaten: [npm Registry](https://registry.npmjs.org/), [Go-Modulproxy](https://proxy.golang.org/), [Go-Releases](https://go.dev/dl/?mode=json), [Node-Releases](https://nodejs.org/dist/index.json), [PostgreSQL-Versionen](https://www.postgresql.org/versions.json), [Redis-Releases](https://github.com/redis/redis/releases), [Alpine-Supportzweige](https://alpinelinux.org/releases/), [PyPI](https://pypi.org/). Die Paketzeilen beruhen auf pro Paket/Modul abgefragten offiziellen Metadaten; es wurden keine Versionsnummern aus Modellwissen ergänzt.

Geprüfte Migrationen: [Vite 6→7](https://v7.vite.dev/guide/migration), [Vite 7→8](https://vite.dev/guide/migration), [Tiptap 2→3](https://tiptap.dev/docs/guides/upgrade-tiptap-v2), [Vue I18n 11](https://vue-i18n.intlify.dev/guide/migration/breaking11), [ESLint Flat Config](https://eslint.org/docs/latest/use/configure/migration-guide), [pnpm Installation](https://pnpm.io/installation), [GoReleaser-Deprecations](https://goreleaser.com/resources/deprecations/), [PostgreSQL-Major-/Minor-Regeln](https://www.postgresql.org/support/versioning/).

Die kompatiblen Updates sind implementiert und die verfügbaren Prüfungen dokumentiert. Eine vollständige Produktivfreigabe wird wegen offenem Lint, fehlenden Vertragsartefakten, ausstehender Browser-/E2E-/Mail-/SSO-Abnahme und verbleibenden Security-Migrationspunkten ausdrücklich nicht behauptet. Produktionsdaten, bestehende Container, fremde Projekte und Workflows bleiben unverändert; nur eigens angelegte temporäre Testcontainer wurden verwendet und anschließend entfernt. Kein Commit, Push, Deployment oder produktiver Migrationslauf.

Weitere Formatprüfung: Alle geänderten Konfigurationen, sechs Workflows, GoReleaser-YAML und dieser Bericht bestehen Prettier. Die zwei betroffenen Testdateien haben dieselben Formatbefunde auch im per git show gesicherten Ausgangsstand; nur obsoleter eslint-env-Kommentar wurde entfernt. Die Workflow-Trigger wurden beim YAML-Vergleich unverändert bestätigt.
