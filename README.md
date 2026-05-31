# Northgate Stores — Secure HR Records System

![Go](https://img.shields.io/badge/Go-1.26.1-00ADD8?style=flat-square&logo=go&logoColor=white)
![SQLite](https://img.shields.io/badge/SQLite-3-003B57?style=flat-square&logo=sqlite&logoColor=white)
![bcrypt](https://img.shields.io/badge/Passwords-bcrypt-critical?style=flat-square)
![CSRF](https://img.shields.io/badge/CSRF-Protected-success?style=flat-square)
![XSS](https://img.shields.io/badge/XSS-Mitigated-success?style=flat-square)
![SQLi](https://img.shields.io/badge/SQL_Injection-Prevented-success?style=flat-square)
![OWASP](https://img.shields.io/badge/OWASP-Aligned-blueviolet?style=flat-square)
![Module](https://img.shields.io/badge/Module-COM6019M-blue?style=flat-square)

A secure web application for managing employee HR records, built for the Software and Web Security module (COM6019M) at York St John University.

The system models a fictional retail company — Northgate Stores — where HR administrators and employees interact with sensitive personnel data through a role-based, server-rendered Go application. The primary focus of the project is security: every design decision is driven by defensive programming principles, server-side access control, and protection against common web application attacks.

---

## Security controls at a glance

| Control | Details |
|---|---|
| Password hashing | bcrypt with `DefaultCost` — plaintext passwords never stored |
| Session management | SQLite-backed server-side sessions, `crypto/rand` 256-bit session IDs, SHA-256 session hashes stored in database |
| Cookie security | `HttpOnly` + `SameSite=Lax` on all session cookies |
| CSRF protection | Per-session tokens on all state-changing POSTs; unique pre-session ID per login visit |
| SQL injection | Prepared statements with `?` placeholders throughout — no string concatenation |
| XSS prevention | Contextual output escaping via Go `html/template` |
| Input validation | Centralised `internal/validation` package — whitelist, length, and format checks |
| Broken access control | IDOR prevented by design — employee records loaded from `session.User.ID`, never from client input |
| Login rate limiting | 5 failed attempts → 2-minute lockout per username + client IP combination |
| Session inactivity timeout | Sessions expire after 15 minutes of inactivity and have a 1-hour absolute expiry |
| Security headers | CSP, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: same-origin` |
| Audit trail | `last_updated_by` and `last_updated_at` automatically updated on every record change |
| Role-based access control | Server-side enforcement — employees cannot access admin routes regardless of URL |

---

## Technologies

- **Language:** Go 1.26.1
- **HTTP & templates:** `net/http`, `html/template` (standard library)
- **Database:** SQLite via `modernc.org/sqlite` v1.50.0
- **Password hashing:** `golang.org/x/crypto/bcrypt` v0.50.0

---

## How to run

From the project root:

```bash
go run ./cmd/server
```

Then open:

```
http://localhost:8080
```

The application initialises the SQLite database and seeds demo data automatically on first run.

---

## Runtime configuration

Configuration is read from environment variables. Safe defaults are used if no variables are set. Values are validated before the application starts.

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | HTTP server port (must be 1024–65535) |
| `DB_PATH` | `northgate.db` | SQLite database file path |

Example:

```bash
PORT=9090 DB_PATH=custom.db go run ./cmd/server
```

---

## Demo accounts

The following accounts are available for testing. Passwords are shown here for assessment purposes only — the database stores bcrypt hashes, never plaintext.

| Username | Password | Role |
|---|---|---|
| `admin` | `AdminPass123!` | HR Administrator |
| `hrmanager` | `HRManager123!` | HR Administrator |
| `alice` | `AlicePass123!` | Employee |
| `bob` | `BobPass123!` | Employee |

---

## Project structure

```
northgate-srms/
├── cmd/
│   └── server/
│       └── main.go               # Entry point — wires dependencies and starts server
├── internal/
│   ├── auth/
│   │   ├── auth.go               # bcrypt credential verification
│   │   └── session.go            # SQLite-backed hashed sessions with expiry and inactivity timeout
│   ├── config/
│   │   └── config.go             # Environment-based runtime config with validation
│   ├── csrf/
│   │   ├── csrf.go               # CSRF token generation and constant-time validation
│   │   └── csrf_test.go          # Unit tests for CSRF token validation and rejection
│   ├── handlers/
│   │   ├── auth_handlers.go      # Login, logout — CSRF and rate limiting integrated
│   │   ├── admin_handlers.go     # Admin record list, view, edit, update
│   │   ├── record_handlers.go    # Employee own-record view and update
│   │   ├── home.go               # Home redirect based on session/role
│   │   └── templates.go          # Template rendering helper
│   ├── middleware/
│   │   └── security_headers.go   # CSP, X-Frame-Options, X-Content-Type-Options, Referrer-Policy
│   ├── security/
│   │   ├── login_limiter.go      # Per-username + client IP rate limiting and lockout
│   │   └── login_limiter_test.go # Unit tests for lockout scope and reset behaviour
│   ├── storage/
│   │   ├── db.go                 # Database initialisation and schema creation
│   │   ├── records.go            # Employee record queries — all parameterised
│   │   ├── users.go              # User and credential queries
│   │   └── seed.go               # Demo data with bcrypt-hashed passwords
│   └── validation/
│       ├── validation.go         # Input validation — format, length, whitelist
│       └── validation_test.go    # Table-driven unit tests
├── templates/
│   ├── login.html
│   ├── home.html
│   ├── record.html
│   ├── record_edit.html
│   ├── admin_records.html
│   ├── admin_record_view.html
│   └── admin_record_edit.html
├── DESIGN.md                     # Full design document — data model, access control matrix, threat model
├── go.mod
└── go.sum
```

---

## Roles and permissions

### Employee

- Log in and log out
- View own HR record only
- Update own `phone` and `emergency_contact` (validated server-side)

Employees cannot access other employees' records, admin routes, or modify sensitive HR fields such as address, salary band, employment status, or private HR notes.

### HR Administrator

- Log in and log out
- View all employee records
- View individual employee records including private HR notes
- Update any permitted non-ID field in any record
- See who last updated each record and when (`last_updated_by`, `last_updated_at`)

Administrators cannot directly edit system-controlled fields: `id`, `user_id`, `last_updated_by`, or `last_updated_at`. These are managed by the server.

---

## Data model

Three core tables are used:

```text
users.id  →  employee_records.user_id
users.id  →  sessions.user_id
```

### `users`

Stores authentication and role data only — separated from HR record data by design.

| Field | Notes |
|---|---|
| `id` | Primary key |
| `username` | Unique login name |
| `password_hash` | bcrypt hash — plaintext never stored |
| `role` | `employee` or `admin` — CHECK constraint enforced in DB |
| `is_active` | Allows account deactivation without deletion |
| `created_at` | Account creation timestamp |

### `employee_records`

Stores sensitive HR data associated with each user.

| Field | Editable by employee | Editable by admin |
|---|---|---|
| `first_name`, `last_name` | No | Yes |
| `email` | No | Yes |
| `phone` | Yes | Yes |
| `address` | No | Yes |
| `emergency_contact` | Yes | Yes |
| `department` | No | Yes |
| `job_title` | No | Yes |
| `employment_status` | No | Yes |
| `salary_band` | No | Yes |
| `accessibility_notes` | No | Yes |
| `private_hr_notes` | No | Yes |
| `last_updated_by` | No — system controlled | No — system controlled |
| `last_updated_at` | No — system controlled | No — system controlled |

### `sessions`

Stores server-side authentication sessions. The browser stores only the raw random session ID in the `northgate_session` cookie. SQLite stores only a SHA-256 hash of that value.

| Field | Notes |
|---|---|
| `id` | Internal session row identifier |
| `session_hash` | SHA-256 hash of the random session ID; the raw cookie value is not stored |
| `user_id` | Foreign key linking the session to the authenticated user |
| `created_at` | Session creation timestamp |
| `expires_at` | Absolute session expiry timestamp |
| `last_activity_at` | Updated after authenticated requests to enforce inactivity timeout |

This improves robustness because sessions survive server restarts and can be removed on logout or expiry. It also reduces the impact of a database leak because the stored session value cannot be directly reused as a valid browser cookie.

---

## Running tests

Unit tests cover input validation, CSRF token validation, and the login rate limiter:

```bash
go test ./...
```

---

## Manual security checks

The following manual checks were used to verify the session storage upgrade and related controls:

| Check | Expected result |
|---|---|
| Valid login | Creates a `northgate_session` cookie and a row in the `sessions` table |
| Session hash storage | `sessions.session_hash` contains a SHA-256 hash, not the raw cookie value |
| User binding | `sessions.user_id` matches the authenticated user |
| Logout | Deletes the matching session row and expires the browser cookie |
| Server restart | Session remains valid after restarting the Go server, until expiry or inactivity timeout |
| Inactivity timeout | Session becomes invalid after 15 minutes of inactivity |
| Invalid cookie | Modified or deleted session cookie is rejected |
| Login rate limiting | Five failed login attempts for the same username and client IP trigger a temporary lockout |
| Lockout scope | A lockout applies to the username + client IP combination rather than the username alone |

Example SQLite inspection after login:

```sql
SELECT id, session_hash, user_id, created_at, expires_at, last_activity_at
FROM sessions;
```

---

## Known limitations

This is an assessment prototype intentionally scoped for clarity and focus on security controls.

| Limitation | Notes |
|---|---|
| No HTTPS | `Secure` cookie flag disabled for `localhost`; required in any real deployment |
| No security event logging | Failed logins, denied access, and CSRF rejections are not persisted |
| CSRF tokens stored in memory | Appropriate for this single-instance assessment prototype, but tokens are lost on server restart and would not work well in a multi-instance deployment; a future improvement would be to bind CSRF tokens to persistent session state or store them in a database-backed token store |
| Simplified IP handling for rate limiting | The limiter uses the client address observed by the Go server through `r.RemoteAddr`; production deployments behind trusted reverse proxies should carefully configure trusted forwarding headers such as `X-Forwarded-For` |
| No MFA | Especially relevant for admin accounts; not implemented because the assessment already includes two additional security features and database-backed hashed sessions were prioritised as a lower-risk session-management improvement |
| No public registration | Users created via seed data only |
| No password reset | Out of scope for this prototype |

---

## AI Use Statement

I used ChatGPT to support brainstorming, report structure planning, grammar review, and feedback on my own draft text and implementation decisions. I also used it to discuss testing evidence and possible security trade-offs. 
The final code, testing, debugging, evidence collection, and submission decisions were reviewed and completed by me.