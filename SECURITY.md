VERDICT: BLOCKED

## Sicherheitsbericht

**Scanner-Abdeckung:** Für dieses Projekt war kein anwendbarer Security-Scanner konfiguriert. Die Bewertung beruht daher vollständig auf manueller Codeanalyse. Abhängigkeiten sind laut `go.mod` keine externen Module vorhanden; bekannte CVEs in Dependencies sind nicht erkennbar.

### Befund 1: Hoch — Fehlende Authentifizierung und Autorisierung
- **Betroffen:** `main.go`, `flags.go` — sämtliche Handler, insbesondere `POST /flags`, `PUT /flags/{key}`, `DELETE /flags/{key}` und `GET /flags/{key}/evaluate`.
- **Risiko:** Der Dienst lauscht auf `:8080` und bietet ohne jede Zugriffskontrolle schreibende Operationen an. Jeder, der den Port erreicht, kann beliebig Feature-Flags anlegen, verändern, löschen und auswerten. Das ist ein klassischer Broken-Access-Control/Auth-Bypass, weil es keinerlei Authentifizierung gibt. Eine Manipulation der Feature-Flags kann unmittelbar Produktverhalten ändern oder Features abschalten.
- **Fix:**
  - Authentifizierung per Middleware vorschalten: API-Key, mTLS oder JWT; Vergleich z. B. mit `crypto/subtle.ConstantTimeCompare`.
  - Schreibzugriffe (`POST`, `PUT`, `DELETE`) zusätzlich autorisieren.
  - Bind-Adresse konfigurierbar machen und standardmäßig auf `127.0.0.1:8080` binden; öffentliche Exposition nur über ein vertrauenswürdiges Gateway/Reverse-Proxy mit Auth und TLS.
  - Beispiel: `newHandler()` um `RequireAuth` erweitern und `http.ListenAndServe` durch einen konfigurierten `http.Server` mit `Addr` aus Environment ersetzen.

### Befund 2: Hoch — Unsicherer HTTP-Server ohne Timeouts und ohne TLS
- **Betroffen:** `main.go`, Zeile `http.ListenAndServe(":8080", newHandler())`.
- **Risiko:** Der Default-Server hat keine Lese-/Schreib- oder Header-Timeout. Das macht den Dienst anfällig für Slowloris/Resource-Exhaustion. Zudem wird unverschlüsselt übertragen; Feature-Flag-Daten und der `user`-Parameter sind nicht transportverschlüsselt.
- **Fix:**
  - Eigenen `http.Server` mit Timeouts verwenden: `ReadHeaderTimeout: 5 * time.Second`, `ReadTimeout: 10 * time.Second`, `WriteTimeout: 10 * time.Second`, `IdleTimeout: 60 * time.Second`.
  - `Addr` über Konfiguration setzen; bei Netzwerkexposition TLS-Terminierung (`ListenAndServeTLS`) oder TLS am Reverse-Proxy erzwingen.

### Befund 3: Mittel — DoS durch unbegrenzte Ressourcennutzung
- **Betroffen:** `store.go` (In-Memory-Store), `flags.go` (Create/Update ohne Rate-Limit).
- **Risiko:** Es gibt keine Begrenzung der Anzahl der Flags, keine Rate-Limits und keine Längenbeschränkung für `key`/`description` über die 1-MiB-Body-Grenze hinaus. Ein erreichbarer Client kann gezielt sehr viele Flags mit großen Beschreibungen anlegen und den Speicher des Prozesses erschöpfen.
- **Fix:**
  - Maximale Flag-Anzahl einführen (z. B. 10 000) und bei Überschreitung `409`/`503` zurückgeben.
  - Key-/Description-Längen begrenzen (z. B. 128 bzw. 1024 Zeichen).
  - Pro Client/IP Rate-Limit in der Middleware (z. B. bei `POST/PUT/DELETE` 10 Requests/s); das erfordert eine vertrauenswürdige Reverse-Proxy- oder Middleware-Implementierung.

### Befund 4: Niedrig — Log-Injection über URL-Pfad
- **Betroffen:** `middleware.go`, Zeile `log.Printf("%s %s %d %s", r.Method, r.URL.Path, rec.status, time.Since(start))`.
- **Risiko:** `r.URL.Path` wird ungefiltert als `%s` geloggt. Ein Angreifer kann Steuerzeichen wie `%0A` (Zeilenumbruch) im Pfad codieren, dadurch Log-Einträge fälschen und das Monitoring/Audit-Log stören.
- **Fix:**
  - Pfad mit `%q` formatieren: `log.Printf("%s %q %d %s", r.Method, r.URL.Path, rec.status, time.Since(start))`.
  - Alternativ Pfad mit `strconv.Quote` escapen oder Steuerzeichen entfernen.

### Befund 5: Niedrig — Fehlende sicherheitsrelevante HTTP-Header
- **Betroffen:** API-Antworten in `flags.go`, `evaluate.go`, `middleware.go`.
- **Risiko:** Keine `X-Content-Type-Options: nosniff` und kein `Cache-Control: no-store` für Antworten mit benutzerspezifischen Daten (`evaluateResponse` enthält den `user`-Wert). Browser/Proxies könnten solche Antworten cachen.
- **Fix:**
  - In einer zentralen Middleware oder in `writeJSON` folgende Header setzen: `X-Content-Type-Options: nosniff`, `Cache-Control: no-store`.
  - CORS-Header nur explizit und restriktiv setzen, falls die API von einer Browser-Anwendung genutzt werden soll.

### Positiv geprüfte Bereiche
- **Secrets:** Keine hartkodierten Schlüssel, Passwörter, Token oder URLs im Code.
- **Injection/Input-Validierung:** JSON-Body wird über `mime.ParseMediaType` auf `application/json` geprüft, per `http.MaxBytesReader` auf 1 MiB begrenzt und JSON-Fehler werden sauber behandelt. Keine SQL-/Command-/Path-Injection erkennbar.
- **Datenschutz:** `r.URL.Path` wird geloggt, nicht `RawQuery`; der `user`-Parameter erscheint nicht im Log. Der `user`-Wert wird nicht im Store abgelegt.
- **Thread-Sicherheit:** Der Store ist korrekt mit `sync.RWMutex` synchronisiert.
- **Dependencies:** Keine externen Module, daher keine bekannte Schwachstelle aus dem Abhängigkeitsbaum sichtbar.