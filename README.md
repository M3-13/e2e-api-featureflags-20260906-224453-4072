# Feature-Flag-Service als REST-API in Go

Ein Feature-Flag-Service als REST-API, ausschließlich mit `net/http` aus der
Standardbibliothek umgesetzt. Feature-Flags lassen sich anlegen, auflisten,
lesen, ändern, löschen und deterministisch pro Nutzer auswerten. Die Daten
liegen in einem thread-sicheren In-Memory-Store, alle Eingaben werden
validiert und Fehler kommen als einheitliche JSON-Objekte zurück.

## Tech Stack

- **Sprache**: Go 1.23
- **Framework**: `net/http` (nur Standardbibliothek, keine externen Module)
- **Routing**: `ServeMux` mit Methoden- und Pfadmustern (Go 1.22+)
- **Storage**: In-Memory mit `sync.RWMutex`
- **Testing**: `go test` mit `net/http/httptest`

## Installation

Voraussetzung: Go 1.23 oder neuer.

```sh
go build ./...
```

## Start (Entwicklung)

```sh
go run .
```

Der Dienst lauscht anschließend auf Port `8080`.

## Build (Produktion)

```sh
go build ./...
```

Das erzeugt ein ausführbares Binary, das den Dienst auf Port `8080` startet.

## Konfiguration

Der Dienst wird über zwei Umgebungsvariablen konfiguriert:

| Variable       | Beschreibung                                                        | Standard |
| -------------- | ------------------------------------------------------------------- | -------- |
| `FLAG_API_KEY` | API-Key zur Authentifizierung der API-Zugriffe.                     | –        |
| `FLAG_ADDR`    | Bind-Adresse des HTTP-Servers, z. B. `127.0.0.1:8080`.              | `:8080`  |

Beispiel:

```sh
FLAG_API_KEY=... FLAG_ADDR=127.0.0.1:8080 go run .
```

## Endpunkte

Fehler werden immer als JSON-Objekt in der Form `{"error":"..."}`
zurückgegeben.

| Methode | Pfad                        | Beschreibung                                                        |
| ------- | --------------------------- | ------------------------------------------------------------------- |
| `POST`  | `/flags`                    | Legt ein Flag an. Body: `{key:string, enabled:bool, description?:string, rollout_percent?:int}` |
| `GET`   | `/flags`                    | Listet alle Flags, alphabetisch nach `key` sortiert.                 |
| `GET`   | `/flags/{key}`              | Liefert ein einzelnes Flag.                                          |
| `PUT`   | `/flags/{key}`              | Teil-Update eines Flags. Body: `{enabled?:bool, description?:string, rollout_percent?:int}` |
| `DELETE`| `/flags/{key}`              | Entfernt ein Flag.                                                   |
| `GET`   | `/flags/{key}/evaluate`     | Wertet ein Flag deterministisch für einen Nutzer aus (`?user={id}`). |
| `GET`   | `/healthz`                  | Health-Check, antwortet mit `200` und `{"status":"ok"}`.             |

## Features

- CRUD-Endpunkte für Feature-Flags (In-Memory-Store)
- Deterministische Rollout-Auswertung pro Nutzer (SHA-256-basiert)
- Einheitliche JSON-Fehlerobjekte
- Logging-Middleware (Methode, Pfad, Statuscode, Dauer)
- Health-Endpoint `GET /healthz`

## Datenschutz

**Zweck der Verarbeitung:** Die über `GET /flags/{key}/evaluate?user={id}`
übergebene Nutzer-ID wird ausschließlich zur deterministischen
Feature-Auswertung verwendet. Dazu wird der SHA-256-Hash von `key:user`
berechnet, um pro Nutzer eine stabile Rollout-Entscheidung zu treffen.

**Rechtsgrundlage:** Die Verarbeitung erfolgt auf Grundlage von Art. 6 Abs. 1
DSGVO (berechtigtes Interesse an einem korrekten, reproduzierbaren
Feature-Rollout).

**Datenminimierung:**

- Der `user`-Wert wird nicht gespeichert und ausschließlich für die
  SHA-256-Berechnung sowie die Rückgabe im Antwort-JSON genutzt.
- Das Freitextfeld `description` eines Flags darf keine personenbezogenen
  Daten enthalten.

**Löschung:** Flags liegen ausschließlich im flüchtigen In-Memory-Store und
werden mit Prozessende automatisch gelöscht. Der `user`-Wert wird zu keinem
Zeitpunkt gespeichert oder protokolliert.

**Verantwortlicher / Kontakt:** Verantwortliche Stelle ist der jeweilige
Betreiber der Dienst-Instanz. Anfragen zum Datenschutz (Auskunft, Berichtigung,
Löschung) sind an die im Repository hinterlegte Kontaktadresse zu richten.

## Wartung

**Versionsschema:** Semantic Versioning in der Form `MAJOR.MINOR.PATCH`.

**Verantwortlichkeit:** Die Wartung erfolgt durch das Entwicklungsteam des
Feature-Flag-Service. Änderungen werden über den regulären Review- und
Merge-Prozess eingespielt.

**Prozess für Sicherheits-Patches:** Sicherheitslücken werden priorisiert
behandelt. Ein Patch durchläuft den regulären CI-Prozess (`go build ./...` und
`go test ./...`) und wird nach Review als Hotfix veröffentlicht. Die
Veröffentlichung erfolgt als neue Version gemäß SemVer.
