VERDICT: CHANGES_REQUESTED

## Einordnung

Reines Go-Backend (REST-API, `net/http`, In-Memory-Store). Keine öffentliche Web-UI, daher keine Pflichten zu Cookie-Banner, Legal Notice, Barrierefreiheit oder AI-Act-Kennzeichnung. Relevant sind DSGVO und CRA. Der sichtbare Stand ist insgesamt solide, weist aber behebbare Lücken auf: unbegrenzte Speicherung von IP-Adressen im Rate-Limiter, Rate-Limit-Bypass bei unauthentifizierten Anfragen, potenziell personenbezogene Daten in Log-Pfaden sowie fehlende TLS-/SBOM-Dokumentation. Keine fundamentalen, sofort blockierenden Rechtsverstöße, daher keine Freigabe, aber auch kein „Blocked“.

---

## 1. DSGVO / Datenschutz

### Finding D1 — Unbegrenzte Speicherung von IP-Adressen im Rate-Limiter
**Schweregrad:** high

`ratelimit.go` legt für jede Quell-IP (`clientIP(r)`) einen dauerhaften Eintrag in der `buckets`-Map an. Es gibt weder TTL noch Eviction noch Löschroutine. IP-Adressen sind personenbezogene Daten (Art. 4 Nr. 1 DSGVO). Die Verarbeitung ist zwar durch berechtigtes Interesse (Missbrauchsverhinderung/Verfügbarkeit, Art. 6 Abs. 1 lit. f DSGVO) grundsätzlich begründbar, aber die **Speicherbegrenzung** (Art. 5 Abs. 1 lit. e DSGVO) ist verletzt: Die Daten bleiben bis zum Prozessende erhalten, ohne Lösch- oder Anonymisierungskonzept.

**Konkrete Abhilfe:**
- `tokenBucket` in `ratelimit.go` um ein Feld `lastSeen time.Time` erweitern.
- In `allow()` bei jedem Zugriff `lastSeen = now` setzen.
- Zusätzlich einen periodischen Cleanup einführen (z. B. alle 5–15 Minuten Einträge mit `now.Sub(b.lastSeen) > 15*time.Minute` entfernen) oder bei Überschreiten einer Maximalgröße alte Einträge verwerfen.
- In `README.md` / `SECURITY.md` die Rechtsgrundlage (Art. 6 Abs. 1 lit. f DSGVO), den Zweck (Rate-Limiting) und die Löschfrist dokumentieren.

> Funktion bleibt erhalten: Der Limiter muss weiterhin pro IP tokengesteuert arbeiten; ein TTL-Cleanup beeinträchtigt die Funktionalität nicht.

---

### Finding D2 — Protokollierung des vollständigen Pfads kann personenbezogene Daten enthalten
**Schweregrad:** medium

`middleware.go` loggt `r.URL.Path` mit `log.Printf("%s %q %d %s", r.Method, r.URL.Path, ...)`. Für `/flags/{key}/evaluate` und `/flags/{key}` wird der konkrete Flag-Key protokolliert. Der Flag-Key selbst kann vom Betreiber frei gewählt werden und darf bis 128 Zeichen beliebig sein (`store.go`). Enthält ein Flag-Key z. B. eine Kundennummer, E-Mail-Adresse oder sonstige Kennung, wird diese dauerhaft in Logs geschrieben — ohne zwingende Notwendigkeit. Der geforderte `user`-Parameter wird korrekterweise nicht geloggt; der Flag-Key bleibt aber ein Restrisiko.

**Konkrete Abhilfe:**
- In `middleware.go` eine Hilfsfunktion `sanitizePath(path string) string` ergänzen, die bei Pfaden unterhalb von `/flags/` das erste Pfadsegment nach `/flags/` maskiert, z. B. `/flags/{key}` und `/flags/{key}/evaluate` liefert.
- Alternativ dokumentieren und durchsetzen, dass Flag-Keys **keine personenbezogenen Daten enthalten dürfen** (z. B. Validierung in `flags.go` um verbotene Muster ergänzen). Die Maskierung ist der robustere datenschutzfreundliche Default.
- Tests in `middleware_test.go` entsprechend anpassen, wenn die Log-Ausgabe nun das Muster statt des konkreten Werts enthält.

> Funktion bleibt erhalten: Das Logging muss weiterhin Methode, Pfad, Status und Dauer ausgeben; die Maskierung ändert nur den Pfadbestandteil.

---

### Finding D3 — Fehlende Datenschutzdokumentation für Betreiber
**Schweregrad:** low

Für ein reines Backend besteht keine Pflicht zu einer eingebauten Datenschutzerklärung. Der Code selbst enthält jedoch keine Hinweise, welche personenbezogenen Daten verarbeitet werden (IP-Adressen im Rate-Limiter, API-Keys, `user`-Parameter). Die Dateien `README.md`, `SECURITY.md` und `COMPLIANCE.md` sind vorhanden, ihr Inhalt ist aber nicht Teil des sichtbaren Reviewstands. Betreiber müssen wissen, was sie verarbeiten und wie sie Betroffenenrechte erfüllen können.

**Konkrete Abhilfe:**
- In `README.md` oder `SECURITY.md` einen Abschnitt „Datenschutz / Verarbeitete Daten“ ergänzen:
  - Verarbeitung von Quell-IPs zum Rate-Limiting, Rechtsgrundlage Art. 6 Abs. 1 lit. f DSGVO, Löschfrist und Zweck.
  - Hinweis, dass `user` ausschließlich für die Hash-Berechnung und die Antwort verwendet und nicht persistiert wird.
  - Hinweis, dass Flag-Keys keine personenbezogenen Daten enthalten sollen (oder dokumentierte Konsequenz, falls doch).
  - Betroffenenrechte: Da keine dauerhaft gespeicherten Nutzerprofile bestehen, ist die Löschung durch Prozessneustart bzw. Löschung der Flags möglich; für IP-Daten gilt die TTL.

---

## 2. EU Cyber Resilience Act (CRA)

### Finding C1 — Rate-Limit greift erst nach Authentifizierung
**Schweregrad:** high

In `main.go` ist der Aufbau:

```go
mux.Handle("/", RequireAuth(RateLimit(protected)))
```

Damit durchläuft ein unauthentifizierter Request zuerst `RequireAuth` und wird bei fehlendem/ungültigem API-Key sofort mit 401 beantwortet, **ohne** den Rate-Limiter zu passieren. Ein Angreifer kann unbegrenzt viele Anfragen ohne gültigen Key senden (Brute-Force auf den API-Key, CPU-/Speicher-DoS). Das widerspricht dem CRA-Grundsatz „security by design/default“ und der Verfügbarkeit.

Zusätzlich begrenzt `RateLimit` nur POST/PUT/DELETE. GET-Anfragen (z. B. `/flags` oder `/flags/{key}/evaluate`) sind auch ohne gültigen Key unlimitiert, da der Limiter vor dem Auth-Check hängt und GET dort nie limitiert wird.

**Konkrete Abhilfe:**
- Einen separaten Fehlversuchs-/Request-Limiter in `RequireAuth` oder davor einbauen:
  - In `auth.go` bei fehlendem/ungültigem Key die Quell-IP zählen; nach z. B. 10 Fehlversuchen pro Minute mit 429 `rate limit exceeded` antworten oder eine kurze Sperre setzen.
  - Alternativ in `main.go` einen vorgelagerten Limiter für **alle** Anfragen an geschützte Routen einfügen (nicht nur Schreibzugriffe), z. B. `mux.Handle("/", RateLimitAuthFailures(RequireAuth(RateLimit(protected))))`, wobei `RateLimitAuthFailures` nur bei 401-Antworten zählt.
  - `GET /healthz` bleibt bewusst öffentlich und davon unberührt.

> Funktion bleibt erhalten: Berechtigte Nutzer mit gültigem API-Key können normale GET-Requests weiterhin ausführen; nur unauthentifizierte Fehlversuche werden begrenzt.

---

### Finding C2 — Kein TLS im sichtbaren Code
**Schweregrad:** medium

`main.go` startet den Server nur mit `ListenAndServe`, es gibt keine TLS-Option. Der Default-Bind `127.0.0.1:8080` ist sicher, aber sobald `FLAG_ADDR` auf eine öffentliche Adresse gesetzt wird, läuft der Dienst unverschlüsselt. API-Keys und Nutzerkennungen (`user`) wären dann im Klartext unterwegs. Die CRA verlangt angemessene Sicherheitsmaßnahmen für Produkte mit digitalen Elementen, einschließlich geschützter Kommunikation.

**Konkrete Abhilfe:**
- In `main.go` optionale TLS-Unterstützung ergänzen, z. B. bei gesetzten Umgebungsvariablen `FLAG_TLS_CERT`/`FLAG_TLS_KEY` `server.ListenAndServeTLS(cert, key)` verwenden.
- Zusätzlich in `README.md`/`SECURITY.md` dokumentieren: „Bei Betrieb hinter einem Reverse Proxy muss TLS-Terminierung vorgeschaltet sein; direkter öffentlicher Betrieb ohne TLS ist unzulässig.“

> Funktion bleibt erhalten: Der bestehende Default (nur localhost) bleibt unverändert; die TLS-Option ist additiv.

---

### Finding C3 — Unbegrenzt wachsende Rate-Limiter-Map (Ressourcen-DoS)
**Schweregrad:** medium

Dieselbe Ursache wie Finding D1, aber aus CRA-Sicht relevant als Speicher-/Ressourcen-Erschöpfung. Die `buckets`-Map in `ratelimit.go` wächst mit jeder neuen Quell-IP dauerhaft an. Ein Angreifer kann durch viele verschiedene Quell-IPs den Speicherverbrauch bis zum Absturz treiben.

**Konkrete Abhilfe:**
- Wie D1: TTL/Eviction implementieren.
- Optional eine maximale Map-Größe festlegen (z. B. 100.000 Einträge) und bei Erreichen alte Einträge entfernen oder den am längsten nicht gesehenen Eintrag verwerfen.

> Funktion bleibt erhalten: Normale Clients behalten ihren Bucket; nur veraltete/überzählige Einträge werden entfernt.

---

### Finding C4 — SBOM und Update-/Patch-Fähigkeit nicht im Code sichtbar
**Schweregrad:** low

Es gibt keine externen Module (`go.mod` ohne Dependencies), daher wäre eine SBOM trivial. Im sichtbaren Code ist jedoch keine SBOM-Erzeugung, kein dokumentierter Update-/Patch-Mechanismus und keine Sicherheitsdokumentation sichtbar. Die Dateien `COMPLIANCE.md`, `SECURITY.md` und `README.md` existieren, ihr Inhalt wurde aber nicht vorgelegt und kann im Review nicht bestätigt werden.

**Konkrete Abhilfe:**
- In `go.mod` darauf achten, dass keine externen Abhängigkeiten auftauchen; bei zukünftigen Abhängigkeiten `go.sum` pflegen.
- In `COMPLIANCE.md` oder `SECURITY.md` festschreiben:
  - SBOM (auch leer, z. B. „keine Fremdmodule, reine Go-Standardbibliothek“).
  - Update-/Patch-Prozess (Release-Tag, Deployment, Neustart mit sauberem State).
  - Sicherheitskontakt und Meldeweg für Schwachstellen.

---

## 3. EU AI Act

### Kein Befund

Der sichtbare Code enthält keine KI-Funktion. Der AI Act ist daher gegenständlich nicht einschlägig. Keine Transparenz- oder Kennzeichnungspflichten.

---

## 4. Pflichttexte, UI und Barrierefreiheit

### Kein Befund

Das Produkt ist ein reines Backend ohne Endnutzer- oder Browser-UI. Cookie-Banner, Legal Notice, Barrierefreiheit nach WCAG/BITV/EAA sind nicht anwendbar. Eine Datenschutzerklärung für Endnutzer kann erforderlich sein, wenn der Betreiber den Dienst in ein Endkundenprodukt integriert; diese liegt außerhalb dieses Repositorys.

---

## Zusammenfassung

Behebbare Mängel, daher **CHANGES_REQUESTED**. Die wichtigsten Maßnahmen:

1. **Rate-Limiter vor Auth / Fehlversuchs-Limiter ergänzen** (`main.go`, `auth.go`) — CRA, Brute-Force-Schutz.
2. **IP-Speicherung mit TTL/Eviction und Dokumentation versehen** (`ratelimit.go`, `README.md`) — DSGVO.
3. **Pfad-Logging maskieren oder Flag-Keys mit PII unterbinden** (`middleware.go`, `flags.go`) — DSGVO.
4. **Optional TLS und klare Betriebsdokumentation ergänzen** (`main.go`, `SECURITY.md`) — CRA.
5. **SBOM-/Update-Prozess in `COMPLIANCE.md`/`SECURITY.md` nachvollziehbar machen** — CRA.

Keine Fundamentalverstöße, die ein sofortiges Blockieren des Merges rechtfertigen, aber die genannten Punkte sollten vor Auslieferung an Kunden behoben werden.