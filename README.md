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
