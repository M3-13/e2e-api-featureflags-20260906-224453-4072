VERDICT: CHANGES_REQUESTED

**Scanner-Hinweis:** Es wurden keine Security-Scanner für diesen Projekttyp ausgeführt. Die Bewertung basiert auf manueller Codeanalyse.

## Sicherheitsbericht

### 1. Mittel – Authentifizierungs-Brute-Force und unauthentifizierte Anfragen nicht wirksam ratenbegrenzt
**Betroffene Stellen:** `main.go`, `auth.go`, `ratelimit.go`

**Problem:**  
In `newHandler()` wird `RequireAuth(RateLimit(protected))` registriert. Das Rate-Limit liegt damit *hinter* der Authentifizierung. Zusätzlich begrenzt `RateLimit` nur `POST`, `PUT` und `DELETE`. Ein Angreifer kann dadurch unbegrenzt `GET`-Anfragen mit falschen API-Keys senden (z. B. `GET /flags`) und so ungestört Brute-Force-Versuche gegen den `X-API-Key` durchführen oder CPU/Netzwerk belasten, ohne je einen `429` zu erhalten.

**Konkreter Fix:**  
Eine eigene Begrenzung für fehlgeschlagene Authentifizierungsversuche pro Client-IP in `RequireAuth` ergänzen, z. B.:
- maximal 5 Fehlversuche pro IP und Zeitfenster,
- danach `429 {"error":"too many unauthorized attempts"}`,
- Zustand mit mutex-geschütztem Map- und Timer-Cleanup verwalten.

Alternativ `RateLimit` vor `RequireAuth` schalten und zusätzlich Authentifizierungsfehlversuche unabhängig von der HTTP-Methode limitieren. Wichtig: Legitime `GET`-Anfragen authentifizierter Clients sollten nicht unverhältnismäßig ausgebremst werden.

---

### 2. Mittel – API-Key kann bei nicht-lokalem Betrieb unverschlüsselt übertragen werden
**Betroffene Stelle:** `main.go`

**Problem:**  
Der Server startet ausschließlich als `http.Server` ohne TLS. Das Standard-Binding `127.0.0.1:8080` schützt lokal, aber sobald `FLAG_ADDR` auf eine nicht-lokale Adresse gesetzt wird, wird der `X-API-Key` im Klartext über das Netzwerk übertragen und kann abgehört werden.

**Konkreter Fix:**  
- `ListenAndServeTLS` mit konfigurierbaren Zertifikatspfaden unterstützen oder
- dokumentieren/erzwingen, dass der Dienst nur hinter einem TLS-terminierenden Reverse-Proxy betrieben wird, wenn das Binding nicht loopback ist.
- Optional: Beim Start mit nicht-lokalem `FLAG_ADDR` ohne TLS eine Warnung ausgeben oder den Start verweigern.

---

### 3. Niedrig – Rate-Limiter-Map wächst unbegrenzt
**Betroffene Stelle:** `ratelimit.go`

**Problem:**  
`newRateLimiter()` erzeugt `buckets map[string]*tokenBucket`. Für jede neue Client-IP wird ein Eintrag angelegt, der nie entfernt wird. Bei langlebigem Betrieb oder vielen unterschiedlichen Client-IPs wächst der Speicher unbegrenzt.

**Konkreter Fix:**  
Periodisch veraltete Buckets entfernen, z. B.:
- im Hintergrund alle 1–5 Minuten Buckets löschen, deren `lastRefill` älter als 10–30 Minuten ist, oder
- eine maximale Anzahl Einträge mit LRU-Eviction einführen.

---

### 4. Niedrig – JSON-Body akzeptiert weitere Daten nach dem ersten JSON-Objekt
**Betroffene Stelle:** `flags.go`, Funktion `readJSONBody`

**Problem:**  
Nach `dec.Decode(dst)` wird der restliche Body lediglich per `io.Copy` verworfen. Dadurch wird z. B.  
`{"key":"a","enabled":true}{"key":"b"}` als gültiger Body akzeptiert, obwohl genau ein JSON-Objekt erwartet wird. Das ist primär ein Validierungsproblem; bei vorgeschalteten Proxys kann es zu unklarer Body-Semantik führen.

**Konkreter Fix:**  
Statt des bisherigen Decoders den gesamten Body bis zum Limit zu lesen und anschließend `json.Unmarshal` zu verwenden, da `json.Unmarshal` zusätzliche Nicht-Whitespace-Daten ablehnt. Alternativ nach dem ersten `Decode` prüfen, dass nur noch Whitespace folgt. Das bestehende `http.MaxBytesReader`-Verhalten bleibt erhalten, damit übergroße Bodies weiterhin `413` auslösen.

---

**Nicht als Schwachstelle gewertet:**  
- Die harten Test-API-Keys in `auth_test.go` sind reine Testwerte und kein produktives Geheimnis.  
- Die Logging-Middleware protokolliert ausschließlich Methode, Pfad, Status und Dauer; Query-Parameter wie `user` werden nicht geloggt.  
- Die SHA-256-basierte Rollout-Entscheidung ist deterministisch und gibt keine sensiblen Daten preis.