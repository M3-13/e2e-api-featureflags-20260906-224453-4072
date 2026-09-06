# Sicherheitsdokumentation

Dieses Dokument beschreibt die Sicherheitseigenschaften, das Bedrohungsmodell
und den Update-/Patch-Prozess des Feature-Flag-Service.

## SBOM (Software Bill of Materials)

- **Abhängigkeiten:** keine Drittanbieter-Abhängigkeiten.
- **Standardbibliothek:** ausschließlich Go-Standardbibliothek (`net/http`,
  `crypto/sha256`, `encoding/json`, `sync`, u. a.).
- **Sprache / Toolchain:** Go 1.23.
- **Module:** kein externes `require` in `go.mod`.

## Sicherheitseigenschaften

| Eigenschaft                        | Beschreibung |
| ---------------------------------- | ------------ |
| Body-Limit (1 MiB)                 | `POST /flags` und `PUT /flags/{key}` begrenzen den Request-Body über `http.MaxBytesReader` auf 1 MiB; größere Bodies werden mit `413` abgelehnt. |
| Content-Type-Prüfung               | Nur `application/json` (auch mit `charset`) wird akzeptiert; andere Content-Types werden mit `415` abgelehnt. |
| Deterministische SHA-256-Auswertung | Rollout-Entscheidungen basieren auf dem SHA-256-Hash von `key:user` (erste 8 Bytes als big-endian `uint64`, modulo 100). |
| Keine PII im Log                   | Die Logging-Middleware protokolliert ausschließlich Methode, Pfad, Statuscode und Dauer — niemals den `user`-Query-Parameter. |
| API-Key-Authentifizierung          | Alle API-Zugriffe werden über `FLAG_API_KEY` authentifiziert (Vergleich in konstanter Zeit). |
| Rate-Limit                         | Anfragen werden pro Client begrenzt, um Missbrauch zu verhindern. |
| Server-Timeouts                    | Der `http.Server` setzt Read-/Write-/Idle-Timeouts gegen Slowloris und Ressourcenerschöpfung. |
| `Cache-Control: no-store`          | Antworten mit nutzerspezifischen Daten werden nicht zwischengespeichert. |

## Bedrohungsmodell

### 1. Unverschlüsselte Übertragung

**Bedrohung:** Der Dienst überträgt standardmäßig unverschlüsselt (HTTP).
Feature-Flag-Daten und die `user`-ID wären im Klartext im Netz lesbar.

**Gegenmaßnahme:** TLS-Terminierung über einen Reverse-Proxy. Der Dienst wird
nicht direkt öffentlich exponiert; TLS wird am vorgelagerten Reverse-Proxy
terminiert.

### 2. Unauthentifizierter Zugriff

**Bedrohung:** Unbefugte könnten Feature-Flags lesen, anlegen, verändern oder
löschen und damit das Produktverhalten manipulieren.

**Gegenmaßnahme:** API-Key-Authentifizierung über `FLAG_API_KEY` (Vergleich in
konstanter Zeit).

### 3. DoS / Ressourcenerschöpfung

**Bedrohung:** Ein erreichbarer Client kann durch übermäßig große Bodies, viele
Anfragen oder langsame Verbindungen Ressourcen (Speicher, CPU, Verbindungen)
erschöpfen.

**Gegenmaßnahme:** 1-MiB-Body-Limit, Rate-Limit und Server-Timeouts.

### 4. Log-Injection

**Bedrohung:** Steuerzeichen (z. B. `%0A`) im Pfad könnten Log-Einträge fälschen
und Monitoring/Audit stören.

**Gegenmaßnahme:** Der Pfad wird escaped formatiert (z. B. mit `%q`), sodass
Steuerzeichen neutralisiert werden.

## Update- und Patch-Prozess

- Sicherheitsrelevante Änderungen durchlaufen den regulären CI-Prozess
  (`go build ./...` und `go test ./...`).
- Patches werden nach Review als Hotfix-Version veröffentlicht (SemVer,
  `MAJOR.MINOR.PATCH`).
- Die `go.mod` bleibt frei von externen Abhängigkeiten; jede hinzukommende
  Abhängigkeit muss im SBOM nachgetragen und geprüft werden.
- Schwachstellenmeldungen werden an die im Repository hinterlegte
  Kontaktadresse gerichtet.
