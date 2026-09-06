VERDICT: CHANGES_REQUESTED

Geprüft wurde der vollständig gemergte Stand des Go-Backends „Feature-Flag-Service“. Das Projekt ist ein reines Backend ohne Endnutzer-UI. Daher entfallen die Prüfbereiche „Mandatory texts & UI“ und „Accessibility“; der EU AI Act ist mangels KI-Funktion nicht anwendbar. Maßgeblich sind DSGVO und Cyber Resilience Act (CRA).

---

## 1. DSGVO (GDPR)

### 1.1 Fehlende Transportverschlüsselung (TLS)
- **Schweregrad:** hoch
- **Ort:** `main.go`
- **Befund:** Der Server startet mit `http.ListenAndServe(":8080", newHandler())`. Die `user`-ID wird über den Query-Parameter `GET /flags/{key}/evaluate?user=...` übertragen und im Antwort-JSON zurückgegeben. Ohne TLS sind personenbezogene Daten im Klartext im Netz lesbar. Das verletzt die Vertraulichkeitsanforderungen aus Art. 32 DSGVO.
- **Konkrete Abhilfe:**  
  In `main.go` TLS aktivieren, z. B.:
  ```go
  http.ListenAndServeTLS(":8443", "server.crt", "server.key", newHandler())
  ```
  Alternativ den Dienst ausschließlich hinter einem TLS-terminierenden Reverse-Proxy betreiben. Zusätzlich sollte die Adresse nicht pauschal an alle Interfaces gebunden werden; sicherer Default ist `127.0.0.1:8080`, sofern keine externe Erreichbarkeit erforderlich ist.

### 1.2 Fehlende Zugriffskontrolle für mutierende Endpunkte
- **Schweregrad:** hoch
- **Ort:** `main.go`, `flags.go`
- **Befund:** `POST /flags`, `PUT /flags/{key}` und `DELETE /flags/{key}` sind ohne Authentifizierung oder Autorisierung erreichbar. Wenn `description` personenbezogene Daten enthält, können Unbefugte diese lesen, verändern oder löschen. Auch die Auswertung mit fremden `user`-Werten ist unkontrolliert möglich. Das ist kein angemessenes Schutzniveau nach Art. 32 DSGVO.
- **Konkrete Abhilfe:**  
  In `newHandler()` eine Authentifizierungs-Middleware einführen, z. B. API-Key- oder Bearer-Token-Prüfung:
  ```go
  return AuthMiddleware(Logging(mux))
  ```
  Für Tests ist die Middleware mit einem Test-Token zu versehen. Mindestens müssen mutierende Endpunkte geschützt werden; `GET /healthz` darf öffentlich bleiben.

### 1.3 Fehlende Cache-Control-Header bei PII-haltigen Antworten
- **Schweregrad:** mittel
- **Ort:** `evaluate.go`, `middleware.go`
- **Befund:** `GET /flags/{key}/evaluate` gibt die `user`-ID im JSON-Body zurück. Antworten auf GET-Anfragen können ohne `Cache-Control: no-store` von Browsern, Proxys oder CDNs zwischengespeichert werden. Dadurch können personenbezogene Daten unkontrolliert vervielfältigt werden.
- **Konkrete Abhilfe:**  
  In `Logging` oder direkt in `Evaluate` setzen:
  ```go
  w.Header().Set("Cache-Control", "no-store")
  ```
  Sinnvoll ist ein globaler Header für alle API-Antworten, da Konfigurationsdaten ebenfalls nicht ohne Weiteres gecacht werden sollen.

### 1.4 Unnötige Rückgabe des `user`-Werts
- **Schweregrad:** mittel
- **Ort:** `evaluate.go`
- **Befund:** `evaluateResponse` enthält das Feld `User` und spiegelt den vom Client gesendeten `user`-Wert zurück. Der Client kennt den Wert bereits; die Antwort erhöht die Exposition personenbezogener Daten. Zwar verlangt AC-16 diese Rückgabe und die Logging-Middleware protokolliert sie nicht, aber aus Datensparsamkeitsgründen (Art. 5 Abs. 1 lit. c DSGVO) ist das unnötig.
- **Konkrete Abhilfe:**  
  Das Feld `User` aus `evaluateResponse` entfernen oder nur dann zurückgeben, wenn der Client dies ausdrücklich benötigt. Falls die Rückgabe beibehalten wird, ist zwingend `Cache-Control: no-store` zu setzen und die Verarbeitung in der Datenschutzdokumentation zu begründen.

### 1.5 Fehlende Dokumentation von Rechtsgrundlage und Verarbeitungszweck
- **Schweregrad:** niedrig
- **Ort:** `README.md`, `SECURITY.md` oder separates Datenschutz-Dokument
- **Befund:** Der Code verarbeitet mit der `user`-ID personenbezogene Daten, dokumentiert aber weder Zweck, Rechtsgrundlage (Art. 6 DSGVO) noch die Rollen (Verantwortlicher/Auftragsverarbeiter). Die Rechenschaftspflicht aus Art. 5 Abs. 2 DSGVO ist nicht erfüllt.
- **Konkrete Abhilfe:**  
  Im `README.md` einen Abschnitt „Datenschutz“ ergänzen: Zweck (deterministische Feature-Auswertung), Rechtsgrundlage, Speicherdauer (`user` wird nicht gespeichert), Löschung, Kontakt. Falls der Dienst als Auftragsverarbeiter betrieben wird, ist ein AV-Vertrag erforderlich.

### 1.6 Retention bei potenziell personenbezogenen Flag-Beschreibungen
- **Schweregrad:** niedrig
- **Ort:** `store.go`, `README.md`
- **Befund:** Der In-Memory-Store hält Flags bis zum Prozessende. `description` ist ein Freitextfeld und kann personenbezogene Daten enthalten. Es gibt keine dokumentierte Beschränkung oder Löschfrist.
- **Konkrete Abhilfe:**  
  Dokumentieren, dass `description` keine personenbezogenen Daten enthalten darf. Falls das nicht garantiert werden kann, eine TTL für Flags oder eine regelmäßige Löschroutine im Store vorsehen.

---

## 2. Cyber Resilience Act (CRA)

### 2.1 Fehlende Dokumentation der Sicherheitseigenschaften und SBOM
- **Schweregrad:** mittel
- **Ort:** `README.md`, `SECURITY.md`
- **Befund:** Es sind keine dokumentierten Sicherheitseigenschaften, kein Bedrohungsmodell und keine SBOM sichtbar. Das Produkt hat keine externen Abhängigkeiten, was die SBOM vereinfacht; dennoch verlangt der CRA für Produkte mit digitalen Elementen eine dokumentierte Sicherheitsanalyse und Nachvollziehbarkeit der Komponenten.
- **Konkrete Abhilfe:**  
  Eine `SECURITY.md` ergänzen mit:
  - SBOM: „Keine Drittanbieter-Abhängigkeiten; ausschließlich Go-Standardbibliothek, Go 1.23“.
  - Sicherheitseigenschaften: Body-Limit 1 MiB, Content-Type-Prüfung, deterministische SHA-256-Auswertung, keine PII-Logs.
  - Bedrohungsmodell: unverschlüsselte Übertragung, fehlende Authentifizierung, DoS-Risiken.

### 2.2 Unsichere Standardkonfiguration des HTTP-Servers
- **Schweregrad:** mittel
- **Ort:** `main.go`
- **Befund:** `http.ListenAndServe` startet ohne Timeouts. Slowloris, Idle-Connection-Erschöpfung oder langsame Clients können den Dienst blockieren. Das widerspricht „security by design/default“.
- **Konkrete Abhilfe:**  
  Einen `http.Server` mit Timeouts verwenden:
  ```go
  srv := &http.Server{
      Addr:              ":8080",
      Handler:           newHandler(),
      ReadHeaderTimeout: 5 * time.Second,
      ReadTimeout:       10 * time.Second,
      WriteTimeout:      10 * time.Second,
      IdleTimeout:       120 * time.Second,
  }
  srv.ListenAndServe()
  ```
  Dazu `import "time"` ergänzen.

### 2.3 Kein dokumentierter Update-/Patch-Prozess
- **Schweregrad:** niedrig bis mittel
- **Ort:** `README.md`, `SECURITY.md`
- **Befund:** Es ist kein Prozess für Sicherheitsupdates, Versionsverwaltung oder Schwachstellenmanagement sichtbar.
- **Konkrete Abhilfe:**  
  Im `README.md` einen Wartungsabschnitt ergänzen: Versionsschema, Verantwortlichkeit, Prozess für Sicherheits-Patches, Ort der Veröffentlichung.

### 2.4 Fehlende Rate-Limits / DoS-Schutz auf Anwendungsebene
- **Schweregrad:** niedrig
- **Ort:** `middleware.go`, `main.go`
- **Befund:** Es gibt keine Begrenzung der Request-Rate. Das erleichtert Missbrauch und Ressourcenerschöpfung.
- **Konkrete Abhilfe:**  
  Eine Rate-Limit-Middleware einführen (z. B. Token-Bucket pro Client-IP) oder den Betrieb hinter einem Reverse-Proxy mit Rate-Limiting dokumentieren.

### 2.5 Fehlende Authentifizierung (auch CRA-relevant)
- **Schweregrad:** hoch
- **Ort:** `main.go`
- **Befund:** Wie unter 1.2 beschrieben, ist der Verwaltungsendpunkt ungeschützt. Unter CRA ist der Schutz vor unbefugtem Zugriff eine zentrale Anforderung.
- **Konkrete Abhilfe:**  
  Siehe 1.2. Die Authentifizierung ist die wichtigste CRA-Maßnahme vor Inbetriebnahme.

---

## 3. EU AI Act

- **Nicht anwendbar.** Der Dienst enthält keine KI-Funktion im Sinne des AI Act. Es handelt sich um eine deterministische Feature-Flag-Auswertung ohne maschinelles Lernen oder Profiling.

---

## 4. Pflichttexte & UI

- **Entfällt.** Das Produkt ist ein reines Backend ohne öffentliche Web-Oberfläche. Es gibt keine rechtlichen Hinweispflichten wie Impressum, Cookie-Banner oder Barrierefreiheitserklärung. Datenschutzhinweise sind dennoch sinnvoll (siehe 1.5), aber keine UI-Pflicht.

---

## 5. Barrierefreiheit

- **Entfällt.** Keine öffentliche Web-UI vorhanden; WCAG/BITV/EAA sind nicht anwendbar.

---

## Positivbefunde

- **Logging erfüllt die Datenschutzanforderungen:** Die Middleware protokolliert ausschließlich Methode, Pfad, Statuscode und Dauer. Der Query-Parameter `user` wird nicht geloggt. Die Tests `TestLoggingDoesNotLogQueryParameters` und `TestEvaluateUserNotStored` decken dies ab.
- **Body-Limit und Content-Type-Prüfung:** `maxBodyBytes` in Kombination mit `http.MaxBytesReader` und `mime.ParseMediaType` ist sauber umgesetzt und verhindert übermäßig große oder falsch typisierte Requests.
- **Flüchtige Verarbeitung der `user`-ID:** Der `user`-Wert wird ausschließlich für SHA-256 und die Antwort verwendet und nicht im Store gehalten. Das entspricht Datenminimierung.
- **Thread-sicherer In-Memory-Store:** `sync.RWMutex` verhindert Datenrennen.
- **Fehlerbehandlung als einheitliche JSON-Objekte:** Gute Voraussetzung für einen sicheren API-Betrieb.

---

## Fazit

Der Code erfüllt die fachlichen Anforderungen und hat solide datenschutzfreundliche Logging- und Validierungsmechanismen. Für eine marktreife Inbetriebnahme fehlen jedoch zwingend Transportverschlüsselung und Zugriffskontrolle. Daneben sind Cache-Control-Header, Server-Timeouts, eine SBOM sowie eine Datenschutz-/Sicherheitsdokumentation erforderlich. Es liegen keine fundamentalen Rechtsverstöße vor, daher kein „BLOCKED“, aber die genannten Lücken müssen vor dem produktiven Einsatz behoben werden.