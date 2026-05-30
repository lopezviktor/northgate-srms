# Northgate Stores — Secure HR Records System

![Go](https://img.shields.io/badge/Go-1.26.1-00ADD8?style=flat-square&logo=go&logoColor=white)
![SQLite](https://img.shields.io/badge/SQLite-3-003B57?style=flat-square&logo=sqlite&logoColor=white)
![Status](https://img.shields.io/badge/Status-Implemented-success?style=flat-square)
![Module](https://img.shields.io/badge/Module-COM6019M-blue?style=flat-square)

> **Document status:** This is the full design document for the Northgate SRMS, covering data model, access control, threat model, route design, testing plan, and implementation roadmap. All phases described in Section 9 have been implemented. Deviations and improvements made during implementation are documented in [Section 10](#10-implementation-notes).

---

## 1. Project Definition

Northgate Stores is a fictional retail company operating multiple stores and requiring a secure internal HR records system for managing employee information.

The system stores and manages sensitive employee records in a controlled, role-based, and accountable manner. It contains data such as contact details, emergency contact information, department, job title, employment status, salary band, accessibility notes, and private HR notes.

The system supports two main roles: employees and HR administrators. Employees may securely access only their own record and update a limited set of low-risk fields. HR administrators may view and manage all employee records.

Security is critical because the system handles sensitive personal and administrative information that must be protected from unauthorised access, unauthorised modification, and common web-based attacks. For this reason, the design of the system is centred on secure authentication, server-side access control, auditability, and defensive handling of user input and session data.

---

## 2. System Scope

### In scope

- Secure login and logout
- Employee access to own HR record only
- Employee update of low-risk fields only:
  - phone
  - emergency_contact
- HR admin access to all employee records
- HR admin update of permitted non-ID fields
- SQLite database
- Password hashing
- Server-side access control
- CSRF protection for state-changing actions
- Secure session cookies
- Database-backed server-side sessions with hashed session identifiers
- Prepared SQL statements
- Safe output rendering with Go html/template
- Input validation
- Audit fields:
  - last_updated_by
  - last_updated_at
- Basic runtime configuration using environment variables with safe defaults
- Two additional security features:
  - login rate limiting / temporary lockout
  - session timeout after inactivity

### Out of scope

- Public registration
- Web-based user creation
- Password reset by email
- Multi-factor authentication
- File uploads
- Payroll processing
- Exact salary storage
- External HR integrations
- Complex UI styling
- Cloud deployment
- Kubernetes or production infrastructure

---

## 3. Data Model

The system uses three core database tables:

- `users`
- `employee_records`
- `sessions`

The `users` table stores authentication and role information. The `employee_records` table stores the sensitive HR record linked to each user. This separation keeps login credentials and HR data in different parts of the data model, making the design easier to reason about and safer to maintain.

The relationship between `users` and `employee_records` is one-to-one:

```text
users.id  →  employee_records.user_id
```

Each user has one employee record, and each employee record belongs to one user.

Authenticated sessions are also stored server-side in SQLite. The browser stores only an unpredictable session identifier in a cookie, while the database stores a SHA-256 hash of that identifier. This keeps authentication state persistent across server restarts without storing reusable raw session tokens in the database.

### 3.1 `users` table

Purpose: stores the information required for authentication and authorisation.

```sql
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('employee', 'admin')),
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

| Column | Type | Constraint | Purpose |
|---|---|---|---|
| `id` | INTEGER | PRIMARY KEY AUTOINCREMENT | Internal user identifier |
| `username` | TEXT | NOT NULL, UNIQUE | Login name |
| `password_hash` | TEXT | NOT NULL | Securely hashed password |
| `role` | TEXT | employee/admin only | Defines system permissions |
| `is_active` | INTEGER | DEFAULT 1 | Allows accounts to be disabled without deleting records |
| `created_at` | TEXT | DEFAULT CURRENT_TIMESTAMP | Basic account creation audit field |

Passwords must never be stored in plain text. The system will store only password hashes generated with a suitable password hashing algorithm such as bcrypt.

### 3.2 `employee_records` table

Purpose: stores the sensitive HR information associated with each employee.

```sql
CREATE TABLE IF NOT EXISTS employee_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL UNIQUE,

    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    phone TEXT NOT NULL,
    address TEXT NOT NULL,
    emergency_contact TEXT NOT NULL,

    department TEXT NOT NULL,
    job_title TEXT NOT NULL,
    employment_status TEXT NOT NULL CHECK (
        employment_status IN ('active', 'on_leave', 'terminated')
    ),
    salary_band TEXT NOT NULL CHECK (
        salary_band IN ('A', 'B', 'C', 'D', 'E')
    ),

    accessibility_notes TEXT,
    private_hr_notes TEXT,

    last_updated_by INTEGER NOT NULL,
    last_updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (last_updated_by) REFERENCES users(id)
);
```

| Column | Type | Constraint | Sensitivity | Editable by employee | Editable by admin | Automatic |
|---|---|---|---|---|---|---|
| `id` | INTEGER | PRIMARY KEY AUTOINCREMENT | Technical | No | No | Yes |
| `user_id` | INTEGER | NOT NULL, UNIQUE, FOREIGN KEY | High | No | No | Created during setup |
| `first_name` | TEXT | NOT NULL | Medium | No | Yes | No |
| `last_name` | TEXT | NOT NULL | Medium | No | Yes | No |
| `email` | TEXT | NOT NULL, UNIQUE | Medium/High | No | Yes | No |
| `phone` | TEXT | NOT NULL | Medium | Yes | Yes | No |
| `address` | TEXT | NOT NULL | High | No | Yes | No |
| `emergency_contact` | TEXT | NOT NULL | High | Yes | Yes | No |
| `department` | TEXT | NOT NULL | Low/Medium | No | Yes | No |
| `job_title` | TEXT | NOT NULL | Low/Medium | No | Yes | No |
| `employment_status` | TEXT | CHECK constraint | High | No | Yes | No |
| `salary_band` | TEXT | CHECK constraint | High | No | Yes | No |
| `accessibility_notes` | TEXT | Optional | High | No | Yes | No |
| `private_hr_notes` | TEXT | Optional | Very high | No | Yes | No |
| `last_updated_by` | INTEGER | NOT NULL, FOREIGN KEY | Audit | No | No | Yes |
| `last_updated_at` | TEXT | DEFAULT CURRENT_TIMESTAMP | Audit | No | No | Yes |

### 3.3 `sessions` table

Purpose: stores server-side authentication sessions in SQLite.

```sql
CREATE TABLE IF NOT EXISTS sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_hash TEXT NOT NULL UNIQUE,
    user_id INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TEXT NOT NULL,
    last_activity_at TEXT NOT NULL,

    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

| Column | Type | Constraint | Purpose |
|---|---|---|---|
| `id` | INTEGER | PRIMARY KEY AUTOINCREMENT | Internal session row identifier |
| `session_hash` | TEXT | NOT NULL, UNIQUE | SHA-256 hash of the random session ID stored in the browser cookie |
| `user_id` | INTEGER | NOT NULL, FOREIGN KEY | Links the session to the authenticated user |
| `created_at` | TEXT | DEFAULT CURRENT_TIMESTAMP | Records when the session was created |
| `expires_at` | TEXT | NOT NULL | Absolute session expiry time |
| `last_activity_at` | TEXT | NOT NULL | Used to enforce inactivity timeout |

The raw session ID is never stored in the database. On each authenticated request, the server reads the session cookie, hashes the submitted session ID, and looks up the resulting hash in the `sessions` table. This reduces the impact of a database leak because the stored value cannot be directly reused as a valid session cookie.

### 3.4 Field editability rules

Employees can update only low-risk fields:

- `phone`
- `emergency_contact`

Employees cannot update administrative or sensitive HR-controlled fields such as:

- `address`
- `email`
- `department`
- `job_title`
- `employment_status`
- `salary_band`
- `accessibility_notes`
- `private_hr_notes`

HR administrators can update all permitted non-ID employee record fields:

- `first_name`
- `last_name`
- `email`
- `phone`
- `address`
- `emergency_contact`
- `department`
- `job_title`
- `employment_status`
- `salary_band`
- `accessibility_notes`
- `private_hr_notes`

No user, including administrators, should directly edit technical identifiers or audit fields:

- `id`
- `user_id`
- `last_updated_by`
- `last_updated_at`

These fields are controlled by the system.

### 3.5 Automatic audit behaviour

Whenever an employee record is updated, the system must automatically update:

- `last_updated_by`
- `last_updated_at`

`last_updated_by` stores the ID of the currently authenticated user who made the change.

`last_updated_at` stores the timestamp when the change was made.

This supports accountability and allows administrators to see who last changed a record and when.

### 3.6 Data validation rules

Input validation will be applied server-side before data is stored.

| Field | Validation rule |
|---|---|
| `username` | 3–30 characters; letters, numbers, dots, hyphens, and underscores only |
| `first_name` | 2–50 characters |
| `last_name` | 2–50 characters |
| `email` | 5–120 characters; basic email format |
| `phone` | 7–20 characters; numbers, spaces, plus signs, and hyphens only |
| `address` | 5–150 characters |
| `emergency_contact` | 5–120 characters |
| `department` | Must be one of the approved departments |
| `job_title` | 2–60 characters |
| `employment_status` | Must be `active`, `on_leave`, or `terminated` |
| `salary_band` | Must be `A`, `B`, `C`, `D`, or `E` |
| `accessibility_notes` | Maximum 500 characters |
| `private_hr_notes` | Maximum 1000 characters |

Approved departments:

- `Sales`
- `Stockroom`
- `Management`
- `HR`
- `Operations`

### 3.7 Example seed users

The system will include test accounts for assessment purposes.

| Username | Role | Purpose |
|---|---|---|
| `admin` | admin | Primary HR administrator account |
| `hrmanager` | admin | Second HR administrator account for assessment testing |
| `alice` | employee | Normal employee account |
| `bob` | employee | Normal employee account |


The README will include test login credentials. In the database, passwords will be stored only as hashes, not in plain text.

---

## 4. Access Control Matrix

Access control is enforced server-side. The user interface may hide unavailable actions, but the server must still verify the authenticated user's role and permissions on every protected request.

The system has two roles:

- `employee`
- `admin`

Authentication proves who the user is. Authorisation decides what that user is allowed to do.

### 4.1 Role permissions

| Action | Employee | HR Admin | Notes |
|---|---|---|---|
| Log in | Yes | Yes | Only active users may log in |
| Log out | Yes | Yes | Ends the current session |
| View own employee record | Yes | Yes | Employees can only access their own record |
| View all employee records | No | Yes | Admin-only function |
| View another employee's record | No | Yes | Admin-only function |
| Update own `phone` | Yes | Yes | Low-risk contact field |
| Update own `emergency_contact` | Yes | Yes | Low-risk contact field, still validated |
| Update own `address` | No | Yes | Requires HR validation |
| Update own `email` | No | Yes | Treated as HR-controlled contact data |
| Update `first_name` / `last_name` | No | Yes | HR-controlled identity data |
| Update `department` | No | Yes | HR-controlled employment data |
| Update `job_title` | No | Yes | HR-controlled employment data |
| Update `employment_status` | No | Yes | Sensitive employment data |
| Update `salary_band` | No | Yes | Sensitive compensation-related data |
| Update `accessibility_notes` | No | Yes | Sensitive personal/support information |
| Update `private_hr_notes` | No | Yes | Highly sensitive HR information |
| Update `id` | No | No | Technical identifier |
| Update `user_id` | No | No | Technical relationship field |
| Update `last_updated_by` | No | No | System-controlled audit field |
| Update `last_updated_at` | No | No | System-controlled audit field |

### 4.2 Server-side enforcement rules

The server must apply access control checks before reading or updating sensitive data.

Employee record access:

```text
If currentUser.role == "employee":
    only allow access where employee_records.user_id == currentUser.id
```

Admin record access:

```text
If currentUser.role == "admin":
    allow access to all employee_records
```

Employee update rule:

```text
If currentUser.role == "employee":
    allow updates only to phone and emergency_contact
    only where employee_records.user_id == currentUser.id
```

Admin update rule:

```text
If currentUser.role == "admin":
    allow updates to permitted non-ID employee record fields
    never allow direct updates to id, user_id, last_updated_by, or last_updated_at
```

Unauthenticated request rule:

```text
If no valid session exists:
    redirect to login or return an unauthorised response
```

### 4.3 Secure query patterns

Employee viewing their own record must use the authenticated user's ID from the server-side session, not a client-provided user ID.

```sql
SELECT *
FROM employee_records
WHERE user_id = ?;
```

Parameter:

```text
currentUser.id
```

Employee updating their own permitted fields:

```sql
UPDATE employee_records
SET phone = ?,
    emergency_contact = ?,
    last_updated_by = ?,
    last_updated_at = CURRENT_TIMESTAMP
WHERE user_id = ?;
```

Parameters:

```text
phone
emergency_contact
currentUser.id
currentUser.id
```

Admin viewing all records:

```sql
SELECT er.*, u.username
FROM employee_records er
JOIN users u ON er.user_id = u.id
ORDER BY er.last_name, er.first_name;
```

Admin updating a record:

```sql
UPDATE employee_records
SET first_name = ?,
    last_name = ?,
    email = ?,
    phone = ?,
    address = ?,
    emergency_contact = ?,
    department = ?,
    job_title = ?,
    employment_status = ?,
    salary_band = ?,
    accessibility_notes = ?,
    private_hr_notes = ?,
    last_updated_by = ?,
    last_updated_at = CURRENT_TIMESTAMP
WHERE id = ?;
```

### 4.4 Important access control risks

The main access control risk is Broken Access Control, especially an employee attempting to access or modify another employee's record by changing a URL parameter or form value.

Example risk:

```text
/records/edit?id=2
```

A normal employee must not gain access just because they know or guess another record ID. The server must verify ownership before returning or updating data.

The design therefore avoids trusting client-provided identifiers for employee actions. For employee access, the record is located using the authenticated user's ID from the session.

### 4.5 Access control design decision

The system deliberately separates authentication from authorisation.

- Authentication checks whether the user has a valid session.
- Authorisation checks whether that authenticated user can perform the requested action on the requested record.



### 4.6 User creation decision

The system does not include public registration or web-based user creation.

Initial users and employee records are created through seed data for assessment and testing purposes. This keeps the scope focused on secure authentication, role-based authorisation, record access, record updates, auditability, and the required web security controls.

In the implemented system:

- Employees cannot create user accounts.
- HR administrators cannot create user accounts through the web interface.
- Demo users are created during setup/seed logic.

In a real deployment, user creation would be restricted to authorised HR administrators and would require additional controls, including password handling, duplicate username/email checks, account activation rules, and audit logging. This feature is deliberately deferred to avoid unnecessary scope expansion in the assessment version.

---

## 5. Route Map

The route map defines the HTTP endpoints used by the application, the method each route accepts, the required role, and the main security controls applied.

State-changing actions must use `POST` and require CSRF protection. `GET` routes are used only to display pages or retrieve information.

### 5.1 Public and authentication routes

| Method | Route | Purpose | Access | CSRF required | Notes |
|---|---|---|---|---|---|
| GET | `/login` | Display login form | Public | No | Redirect logged-in users to their dashboard |
| POST | `/login` | Process login attempt | Public | Yes | Validates credentials and creates session |
| POST | `/logout` | End current session | Authenticated users | Yes | Deletes/invalidates session cookie |

### 5.2 Employee routes

| Method | Route | Purpose | Access | CSRF required | Notes |
|---|---|---|---|---|---|
| GET | `/record` | View own employee record | Employee/Admin | No | Uses authenticated user's ID, not client-provided ID |
| GET | `/record/edit` | Display employee edit form | Employee/Admin | No | Shows only fields employees are allowed to edit |
| POST | `/record/update` | Update own low-risk fields | Employee/Admin | Yes | Allows only `phone` and `emergency_contact` |

### 5.3 Admin routes

| Method | Route | Purpose | Access | CSRF required | Notes |
|---|---|---|---|---|---|
| GET | `/admin/records` | View all employee records | Admin only | No | Lists all employee records |
| GET | `/admin/records/view?id={id}` | View one employee record | Admin only | No | Requires admin role before fetching record |
| GET | `/admin/records/edit?id={id}` | Display admin edit form | Admin only | No | Allows admin-editable fields only |
| POST | `/admin/records/update` | Update an employee record | Admin only | Yes | Updates permitted non-ID fields and audit fields |

### 5.4 Optional/system routes

| Method | Route | Purpose | Access | CSRF required | Notes |
|---|---|---|---|---|---|
| GET | `/` | Entry point | Public/Auth-aware | No | Redirects to `/login`, `/record`, or `/admin/records` depending on session/role |

### 5.5 Route security rules

#### Login

`POST /login` must:

- validate input length before processing credentials;
- use a prepared statement to retrieve the user by username;
- compare the submitted password with the stored password hash;
- reject inactive users;
- create a new unpredictable session ID after successful login;
- store only a SHA-256 hash of the session ID in the SQLite `sessions` table;
- associate the stored session with the authenticated `user_id`;
- set the session cookie using secure attributes such as `HttpOnly` and `SameSite`;
- return a generic error message for failed login attempts.

#### Logout

`POST /logout` must:

- require a valid CSRF token;
- delete the server-side session row from SQLite;
- expire the session cookie in the browser;
- redirect the user to the login page.

#### Employee record access

`GET /record` must:

- require authentication;
- load the record using the authenticated user's ID from the session;
- never accept a user ID or record ID from the employee for this route.

#### Employee record update

`POST /record/update` must:

- require authentication;
- require a valid CSRF token;
- validate `phone` and `emergency_contact` server-side;
- ignore or reject any attempt to submit fields outside the employee edit scope;
- update `last_updated_by` and `last_updated_at` automatically;
- use a prepared statement with `WHERE user_id = currentUser.id`.

#### Admin record access

Admin routes must:

- require authentication;
- require `currentUser.role == "admin"`;
- deny access to employees even if they manually enter the admin URL;
- fetch records only after the admin role check has passed.

#### Admin record update

`POST /admin/records/update` must:

- require authentication;
- require the admin role;
- require a valid CSRF token;
- validate all editable fields server-side;
- prevent direct modification of `id`, `user_id`, `last_updated_by`, and `last_updated_at`;
- update `last_updated_by` and `last_updated_at` automatically;
- use prepared statements.

### 5.6 Route design decisions

The route design deliberately separates employee and admin update flows.

Employees use:

```text
GET  /record/edit
POST /record/update
```

Administrators use:

```text
GET  /admin/records/edit?id={id}
POST /admin/records/update
```

This separation reduces the risk of accidentally applying admin-level update logic to employee requests. It also makes testing clearer because employee and admin permissions can be tested independently.

The system avoids state-changing `GET` requests. All changes to records, sessions, or sensitive state are handled using `POST` with CSRF protection.

---

## 6. Threat and Risk Model

This section identifies the main assets, trust boundaries, threats, likely impact, and planned mitigations for the system. The goal is not to cover every possible attack, but to identify the risks most relevant to a small web-based HR records system handling sensitive employee data.

### 6.1 Key assets

The main assets that require protection are:

| Asset | Why it matters |
|---|---|
| Employee HR records | Contain personal, employment, accessibility, emergency contact, and HR notes |
| Password hashes | Could be targeted in a database breach |
| Session identifiers | Allow authenticated access while a user is logged in; raw values must not be stored server-side |
| Audit fields | Support accountability and record integrity |
| Admin functionality | Allows access to and modification of all records |
| Application availability | Users and HR admins need reliable access to records |
| System design and runtime configuration | Weak or unvalidated configuration can expose data, weaken controls, or change application behaviour unexpectedly |

The most sensitive record fields are:

- `salary_band`
- `employment_status`
- `private_hr_notes`
- `accessibility_notes`
- `address`
- `emergency_contact`
- `email`

These fields require stronger access control and validation than lower-risk fields such as `phone`.

### 6.2 Trust boundaries

The main trust boundaries are:

| Boundary | Risk |
|---|---|
| Browser → Go web server | User input, cookies, form values, and URL parameters may be modified by the user |
| Go web server → SQLite database | SQL queries must not allow user input to change query structure |
| Session cookie → server-side session store | A valid cookie must be checked against server-side session state |
| Templates → browser | Stored or reflected data must not become executable HTML or JavaScript |
| Configuration/startup environment → application | Environment variables are external configuration input and must not enable unsafe behaviour |

The server must treat all browser-supplied data as untrusted, including hidden fields, URL parameters, cookies, and form inputs.

Runtime configuration supplied through environment variables should also be treated carefully. Values such as ports, database paths, debug settings, or deployment-specific options can change application behaviour and should use safe defaults, basic validation, and restricted access in real deployments.

### 6.3 Main risks and mitigations

| Risk | How it could happen | Impact | Mitigation |
|---|---|---|---|
| Broken Access Control / IDOR | An employee changes a URL or form value to access another employee's record | Exposure or unauthorised modification of HR data | Server-side role checks; employee records loaded by `currentUser.id`; admin routes protected by explicit admin checks |
| Forced browsing | An employee manually visits `/admin/records` or another admin URL | Unauthorised access to admin functions | Admin middleware/checks on every admin route; UI hiding is not treated as a security control |
| SQL Injection | Malicious input is inserted into SQL queries | Login bypass, data exposure, or data modification | Prepared statements for all database queries; no string concatenation for SQL |
| Cross-Site Scripting (XSS) | HR notes or other stored fields contain HTML/JavaScript payloads | Browser executes attacker-controlled code, possibly exposing data or performing actions as the user | Render all pages through Go `html/template`; avoid unsafe template output; validate input length and format |
| Cross-Site Request Forgery (CSRF) | A malicious site tricks a logged-in user into submitting a state-changing request | Unwanted record updates or logout actions | CSRF token required on all state-changing `POST` routes; no state-changing `GET` requests; SameSite session cookies |
| Session hijacking or session misuse | A session cookie is stolen, guessed, reused, or remains valid longer than necessary | Attacker acts as the logged-in user | Random session IDs; HttpOnly and SameSite cookie flags; SQLite-backed server-side sessions; hashed session identifiers in the database; absolute expiry; inactivity timeout; logout deletes the server-side session row |
| Weak password storage | Passwords stored in plaintext or weak hashes | Breach exposes real credentials | Store only password hashes using bcrypt; never log passwords |
| Brute-force login attempts | Attacker repeatedly guesses usernames and passwords | Account compromise | Login rate limiting using the username + client IP combination; temporary lockout after repeated failures; generic error messages |
| Malformed or oversized input | User submits unexpected, very long, or invalid data | Validation bypass, crashes, inconsistent records, or stored malicious content | Server-side validation for length, format, and whitelist fields before database updates |
| Missing auditability | Records are changed without knowing who changed them | Loss of accountability and weaker integrity | Automatically update `last_updated_by` and `last_updated_at` on every record update |
| Information leakage through errors | Detailed database or server errors are shown to users | Attackers learn internal details | Generic user-facing error messages; detailed errors kept out of templates and not exposed to users |
| Security misconfiguration | Debug mode, unsafe defaults, exposed files, or unsafe runtime configuration are left enabled | Internal information or sensitive data could be exposed, or the system could behave insecurely | Avoid debug output in user responses; keep database and `.env` files out of version control; validate environment configuration; use safe defaults |
| Logging sensitive data | Passwords, session IDs, CSRF tokens, or HR notes are logged | Logs become a secondary data breach source | Do not log secrets or sensitive HR field values; log only necessary security events if logging is implemented |
| Dependency or supply chain weakness | Third-party packages contain vulnerabilities or are misused | Compromise through external code | Keep dependencies minimal; use standard library where possible; use well-known packages only when needed |
| Poor error handling or inconsistent code | Missing checks or ignored errors cause unsafe behaviour | Security controls may fail silently | Handle errors explicitly; use small focused functions; test negative cases |

### 6.4 Risks directly prioritised for this assessment

The highest-priority risks for this system are:

1. Broken Access Control / IDOR
2. SQL Injection
3. CSRF
4. XSS
5. Session misuse
6. Weak password storage
7. Invalid or abusive input
8. Missing auditability

These are prioritised because they directly affect the required user stories and the sensitive HR record data.

### 6.5 Risks considered but controlled by scope

Some risks are recognised but deliberately limited by project scope:

| Risk | Scope decision |
|---|---|
| Public account registration abuse | Public registration is not implemented |
| Unsafe user creation workflow | Web-based user creation is not implemented; demo users are created by seed logic |
| File upload attacks | File uploads are out of scope |
| Payroll fraud through exact salary editing | Exact salary storage is out of scope; the system stores `salary_band` only |
| Cloud/Kubernetes misconfiguration | Cloud deployment and Kubernetes are out of scope |
| CORS misconfiguration | The system is a same-origin server-rendered app and does not expose a cross-origin API |
| Buffer overflow in application code | The application is implemented in Go, a memory-safe language for normal application code |

These decisions reduce unnecessary attack surface and help keep the implementation focused on the security controls required by the assessment.

### 6.6 Design principles derived from the risk model

The risk model leads to the following design principles:

- Enforce access control on the server, not in the user interface.
- Use the authenticated session user as the source of identity.
- Treat all client input as untrusted.
- Use prepared statements for all database access.
- Use `POST` plus CSRF tokens for state-changing actions.
- Render dynamic output through `html/template`.
- Store password hashes only, never plaintext passwords.
- Store only hashed session identifiers in the database, never raw reusable session tokens.
- Update audit fields automatically.
- Keep the project scope small enough for controls to be applied consistently.
- Use environment variables only for limited runtime configuration, with safe defaults and no secrets committed to version control.

---

## 7. Additional Security Features Options

The assignment requires two additional security features beyond the mandatory controls. These features should improve the security, robustness, or resilience of the system without creating unnecessary scope or weakening the consistency of the core access-control design.

The options below are evaluated using five criteria:

- security value;
- implementation complexity;
- ease of testing;
- usefulness in the final report;
- relevance to an internal HR records system.

The best features for this project are not necessarily the most advanced ones. The strongest choices are the ones that can be implemented consistently, tested clearly, and justified as proportionate controls for sensitive employee records.

### 7.1 Option comparison

| Option | Security value | Complexity | Easy to test | Report value | Overall suitability | Notes |
|---|---|---|---|---|---|---|
| Login rate limiting / temporary lockout | High | Medium | High | High | Very strong | Protects against repeated password guessing |
| Session timeout after inactivity | High | Medium | High | High | Very strong | Reduces risk from unattended or reused sessions |
| Security event logging | High | Medium | High | High | Very strong | Improves accountability and supports investigation |
| Content Security Policy (CSP) header | Medium/High | Low | Medium | Medium/High | Strong | Defence-in-depth against XSS |
| Generic error handling | Medium | Low | High | Medium | Good | Prevents internal error details being exposed |
| Account deactivation check | Medium | Low | High | Medium | Good | Blocks disabled users from logging in |
| Password strength validation | Medium | Low/Medium | High | Medium | Limited for this scope | More useful if user creation or password change existed |
| Re-authentication for sensitive admin updates | High | High | Medium | High | Too much scope | Strong but likely unnecessary for this assessment version |
| Admin audit log viewer | Medium/High | Medium/High | Medium | Medium | Optional but risky | Useful, but adds UI and access-control complexity |

### 7.2 Shortlisted features

The strongest shortlisted features are:

1. Login rate limiting / temporary lockout
2. Session timeout after inactivity
3. Security event logging
4. Content Security Policy header

These options are relevant because the system handles sensitive HR records, relies on login sessions, and includes admin functionality that can access all employee records.

### 7.3 Candidate 1: Login rate limiting / temporary lockout

This feature limits repeated failed login attempts.

Proposed behaviour:

```text
If the same username + client IP combination has 5 failed login attempts:
    temporarily block further login attempts for that username from that client IP
    block duration: 2 minutes
```

Security benefit:

- reduces brute-force password guessing;
- slows automated login attacks;
- protects both employee and admin accounts;
- supports defence-in-depth around authentication.

Implementation approach:

- track failed login attempts server-side;
- key each attempt record by normalised username + client IP;
- extract the client IP from the address observed by the Go server via `r.RemoteAddr`;
- store the number of failed attempts and lockout expiry time;
- reset failed attempts for the matching username + client IP after a successful login;
- return a generic error message for failed and locked login attempts.

Testing approach:

| Test | Expected result |
|---|---|
| 1–4 failed attempts for the same username + IP | Login remains available |
| 5 failed attempts for the same username + IP | Login is temporarily blocked for that combination |
| Correct password during lockout from the same IP | Login remains blocked |
| Same username from a different IP | Not affected by the lockout created from the first IP |
| Different username from the same IP | Not affected by the first user's lockout |
| Correct password after lockout expires | Login succeeds |
| Successful login | Failed attempt counter resets for the matching username + IP |

Trade-off:

- legitimate users may be temporarily blocked after repeated mistakes from the same client IP;
- using username + IP reduces the risk that an attacker can deliberately lock out another user globally by repeatedly submitting that user's username;
- the control still remains simplified because production systems behind reverse proxies must handle trusted forwarding headers carefully;
- the short lockout window keeps the control proportionate for the assessment version.

Assessment value:

This is a strong additional feature because it is security-relevant, easy to test, and clearly linked to authentication risk.

### 7.4 Candidate 2: Session timeout after inactivity

This feature expires sessions after a period of inactivity.

Proposed behaviour:

```text
If a session is inactive for more than 15 minutes:
    invalidate the session
    redirect the user to login
```

Security benefit:

- reduces the risk of unattended sessions being reused;
- limits the usefulness of old session cookies;
- is especially relevant for HR administrators because admin sessions can access all employee records.

Implementation approach:

- store server-side sessions in the SQLite `sessions` table;
- store only `session_hash`, not the raw session ID;
- store `user_id`, `created_at`, `expires_at`, and `last_activity_at` for each session;
- update `last_activity_at` after each valid authenticated request;
- compare the current time with `expires_at` and `last_activity_at` on protected routes;
- delete the session row if the absolute expiry or inactivity limit has been exceeded.

Testing approach:

| Test | Expected result |
|---|---|
| Request before timeout | Session remains valid |
| Request after timeout | Session is invalidated |
| Expired user accesses protected page | User is redirected to login |
| User logs in again | New valid session is created |

Trade-off:

- shorter timeouts improve security but reduce usability;
- longer timeouts improve convenience but increase exposure;
- 15 minutes is a reasonable balance for an internal HR system prototype.

Assessment value:

This is a strong additional feature because it reinforces session management, which is central to the assignment and easy to demonstrate in testing.

### 7.5 Candidate 3: Security event logging

This feature records important security-related events.

Possible logged events:

- successful login;
- failed login;
- logout;
- denied access to an admin route;
- failed CSRF validation;
- employee record update;
- admin record update.

Possible table:

```sql
CREATE TABLE IF NOT EXISTS security_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER,
    event_type TEXT NOT NULL,
    event_details TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

Security benefit:

- improves accountability;
- supports investigation after suspicious behaviour;
- helps identify repeated failed login attempts or unauthorised access attempts;
- complements the existing `last_updated_by` and `last_updated_at` audit fields.

Testing approach:

| Test | Expected result |
|---|---|
| Successful login | Login event is recorded |
| Failed login | Failed login event is recorded |
| Employee updates record | Update event is recorded |
| Employee attempts admin route | Access denied event is recorded |
| Logout | Logout event is recorded |

Trade-off:

- logs must not contain passwords, session IDs, CSRF tokens, or sensitive HR notes;
- excessive logging can create another source of sensitive information;
- if implemented, event details should remain minimal and security-focused.

Assessment value:

This is a strong option because it demonstrates accountability and professional security judgement. However, it may add more database and UI work if an admin log viewer is included. For this assessment version, logging could be implemented without a full viewer if needed.

### 7.6 Candidate 4: Content Security Policy header

This feature adds a browser security header to reduce the impact of XSS.

Example header:

```text
Content-Security-Policy: default-src 'self'; script-src 'self'; object-src 'none'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'
```

Security benefit:

- provides defence-in-depth against XSS;
- restricts where scripts and other resources can load from;
- helps prevent clickjacking through `frame-ancestors 'none'`;
- limits the impact of accidental unsafe output.

Testing approach:

| Test | Expected result |
|---|---|
| Inspect response headers | CSP header is present |
| Submit a simple script payload in a text field | Payload is rendered as text and not executed |
| Normal page rendering | Page still works correctly |

Trade-off:

- CSP does not replace output encoding;
- incorrect CSP configuration can break legitimate scripts or forms;
- because the application uses simple server-rendered HTML, CSP is useful but should remain a defence-in-depth feature rather than the primary XSS control.

Assessment value:

This is a good option because it is simple and clearly linked to XSS mitigation. However, it may be less impressive than rate limiting, session timeout, or security event logging because it is mainly a response header rather than a broader behavioural control.

### 7.7 Features not selected as primary extras

Some options are useful but less suitable as the two main additional features.

| Feature | Reason not selected as a primary extra |
|---|---|
| Password strength validation | There is no public registration or password change feature, so its value is limited in the implemented system |
| Re-authentication for admin updates | Strong control, but adds extra complexity and could distract from the required user stories |
| Admin audit log viewer | Useful, but creates extra UI, route, and access-control work |
| Account deactivation check | Useful and low-cost, but may be too small to count as a strong additional feature on its own |
| Generic error handling | Important and should still be implemented, but it is better treated as a general defensive coding practice |

### 7.8 Recommended final choice

Recommended additional security features:

1. Login rate limiting / temporary lockout
2. Session timeout after inactivity

Reason:

These two features strengthen authentication and session management, which are central to the system. They are realistic to implement in Go, easy to test, and clearly relevant to a secure HR records system.

They also provide useful material for the report because they involve clear security trade-offs:

- rate limiting improves resistance to brute-force attempts but can temporarily inconvenience legitimate users;
- session timeout reduces exposure from unattended sessions but can slightly reduce usability.

Alternative strong choice:

1. Login rate limiting / temporary lockout
2. Security event logging

This alternative would be especially strong if the report focuses more heavily on accountability and incident investigation.

### 7.9 Implemented features

Both shortlisted features were implemented as planned:

```text
Additional Feature 1: Login rate limiting / temporary lockout  ✓ Implemented using username + client IP scope
Additional Feature 2: Session timeout after inactivity         ✓ Implemented
Session storage upgrade: SQLite-backed hashed sessions         ✓ Implemented
```

Additionally, the following defence-in-depth controls were implemented beyond the two required features:

- **Security headers middleware** — CSP, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: same-origin` applied to all responses via a dedicated middleware layer.
- **Runtime configuration hardening** — `PORT` and `DB_PATH` configurable via environment variables, validated on startup with safe defaults and path traversal prevention.

---

## 8. Testing Plan

This section defines the testing plan before implementation. The aim is to verify that the system meets the required user stories, applies security controls consistently, and handles invalid or hostile input safely.

Testing will include:

- functional testing;
- role-based access control testing;
- security control testing;
- negative testing;
- input validation testing;
- additional security feature testing.

The testing approach follows the idea that security must be verified through both expected behaviour and misuse cases.

### 8.1 Testing principles

The system will be tested using the following principles:

- test both authorised and unauthorised actions;
- test successful and failed login attempts;
- test that employees cannot access or modify other employees' records;
- test that admin-only routes are protected server-side;
- test all state-changing routes for CSRF protection;
- test malicious or malformed input;
- test that audit fields are updated automatically;
- test the two selected additional security features;
- avoid relying only on the user interface as proof of security.

### 8.2 Test accounts

The following demo accounts will be used for testing.

| Username | Role | Purpose |
|---|---|---|
| `admin` | admin | Tests primary HR administrator functionality |
| `hrmanager` | admin | Tests a second HR administrator account |
| `alice` | employee | Tests normal employee access |
| `bob` | employee | Tests access-control separation between employees |

The README will include the test passwords for these accounts. Passwords stored in the database must be hashed, not plaintext.

### 8.3 Functional user story tests

| ID | User story / requirement | Test steps | Expected behaviour | Pass criteria |
|---|---|---|---|---|
| F1 | User can log in securely | Log in as `alice` with valid credentials | Alice is authenticated and redirected to her record/dashboard | Valid login succeeds and session is created |
| F2 | User cannot log in with invalid credentials | Try logging in as `alice` with the wrong password | Login fails with a generic error message | No session is created |
| F3 | User can log out securely | Log in, then submit logout form | Session is invalidated and user is redirected to login | Protected routes require login again |
| F4 | Employee can view own record only | Log in as `alice` and open `/record` | Alice's own employee record is displayed | No other employee data is shown |
| F5 | Admin can view all records | Log in as `admin` and open `/admin/records` | All employee records are listed | Admin can see Alice and Bob records |
| F6 | Employee can update own phone | Log in as `alice`, update `phone` via `/record/update` | Alice's phone is updated | Database value changes for Alice only |
| F7 | Employee can update own emergency contact | Log in as `alice`, update `emergency_contact` | Alice's emergency contact is updated | Database value changes for Alice only |
| F8 | Admin can update permitted fields | Log in as `admin`, edit Bob's record | Admin-editable fields are updated | Bob's record changes and audit fields update |
| F9 | Audit fields update automatically | Update a record as employee or admin | `last_updated_by` and `last_updated_at` change automatically | Audit fields reflect the acting user and update time |

### 8.4 Access control tests

| ID | Risk tested | Test steps | Expected behaviour | Pass criteria |
|---|---|---|---|---|
| AC1 | Employee attempts admin route | Log in as `alice` and visit `/admin/records` | Access is denied or redirected | Alice cannot view admin page |
| AC2 | Employee attempts to view another record by ID | Log in as `alice` and try `/admin/records/view?id=2` | Access is denied | Alice cannot view Bob's record |
| AC3 | Employee attempts to edit another record | Log in as `alice` and submit a crafted request targeting Bob's record | Request is rejected or ignored | Bob's record is unchanged |
| AC4 | Employee submits admin-only fields | Log in as `alice` and submit `salary_band`, `address`, or `private_hr_notes` in the request | Fields are ignored or request is rejected | Admin-only fields are not changed |
| AC5 | Unauthenticated user accesses protected page | Open `/record` without logging in | Redirect to login or unauthorised response | No record data is exposed |
| AC6 | Unauthenticated user submits update | Submit POST request to `/record/update` without session | Request is rejected | No database update occurs |
| AC7 | Admin route checks are server-side | Manually enter admin URLs as employee | Server denies access regardless of UI | Hidden links are not the only protection |

### 8.5 SQL injection tests

| ID | Area tested | Test input | Expected behaviour | Pass criteria |
|---|---|---|---|---|
| SQL1 | Login form username | `' OR '1'='1' --` | Login fails | Injection does not bypass authentication |
| SQL2 | Login form password | `' OR '1'='1' --` | Login fails | Query treats input as data |
| SQL3 | Record update field | Submit SQL-like text in `emergency_contact` | Input is stored as text or rejected by validation | SQL query structure is not changed |
| SQL4 | Admin record ID | Use malformed ID such as `1 OR 1=1` | Request is rejected or returns error safely | No unexpected records are returned or updated |

All database operations must use prepared statements or parameterised queries.

### 8.6 XSS tests

| ID | Area tested | Test input | Expected behaviour | Pass criteria |
|---|---|---|---|---|
| XSS1 | Employee editable field | `<script>alert(1)</script>` in `emergency_contact` | Payload is displayed as text or rejected | Script does not execute |
| XSS2 | Admin editable HR notes | `<img src=x onerror=alert(1)>` in `private_hr_notes` | Payload is safely escaped or rejected | Browser does not execute JavaScript |
| XSS3 | Stored XSS check | Save payload, log out, log in again, view record | Payload remains non-executable | Stored content does not execute |
| XSS4 | Template rendering | View records containing special characters | Page renders safely | HTML structure is not broken |

The application should use Go `html/template` for automatic contextual output escaping.

### 8.7 CSRF tests

| ID | Area tested | Test steps | Expected behaviour | Pass criteria |
|---|---|---|---|---|
| CSRF1 | Employee update without token | Submit POST to `/record/update` without CSRF token | Request is rejected | No record update occurs |
| CSRF2 | Employee update with invalid token | Submit POST with wrong CSRF token | Request is rejected | No record update occurs |
| CSRF3 | Employee update with valid token | Submit form normally | Request succeeds | Record updates correctly |
| CSRF4 | Admin update without token | Submit POST to `/admin/records/update` without CSRF token | Request is rejected | No admin update occurs |
| CSRF5 | Logout without token | Submit logout request without CSRF token | Request is rejected | Session remains valid or request fails safely |
| CSRF6 | State-changing GET check | Try to update record using a GET request | Request does not perform update | GET routes do not change state |

All state-changing actions must use `POST` and require a valid CSRF token.

### 8.8 Session and cookie tests

| ID | Area tested | Test steps | Expected behaviour | Pass criteria |
|---|---|---|---|---|
| S1 | Session creation | Log in with valid credentials | Session cookie is created and a row appears in `sessions` | User can access protected routes |
| S2 | Session cookie flags | Inspect session cookie in browser/dev tools | Cookie includes `HttpOnly` and `SameSite` | Required cookie attributes are present |
| S3 | Hashed session storage | Compare browser cookie value with the `sessions.session_hash` value | Database stores a SHA-256 hash, not the raw cookie value | Raw session ID is not stored server-side |
| S4 | Session linked to user ID | Inspect the `sessions` table after login | Session row contains the authenticated `user_id` | Session identity is resolved server-side |
| S5 | Session invalidation | Log out, then inspect `sessions` and revisit `/record` | Session row is deleted and protected route requires login | Old session no longer grants access |
| S6 | Invalid session ID | Modify or delete session cookie | Access to protected routes is denied | Invalid session is not trusted |
| S7 | Session persistence after restart | Log in, restart the Go server without logging out, then refresh a protected page | Session remains valid because it is stored in SQLite | Session survives server restart |
| S8 | Inactivity timeout | Log in, wait beyond the inactivity timeout, then refresh a protected page | Session is deleted/invalidated and user must log in again | Inactive sessions cannot access data |
| S9 | Session identity source | Attempt to submit another user ID in a form | Server still uses session user ID | User cannot impersonate another account |

If HTTPS is not used in the local development environment, the `Secure` cookie flag may be discussed as a production requirement rather than enforced locally.

### 8.9 Password storage tests

| ID | Area tested | Test steps | Expected behaviour | Pass criteria |
|---|---|---|---|---|
| P1 | Password hashing | Inspect users table after seed setup | Passwords are stored as hashes | No plaintext passwords exist |
| P2 | Correct password verification | Log in with correct password | Login succeeds | Hash comparison works |
| P3 | Wrong password verification | Log in with wrong password | Login fails | Wrong password is rejected |
| P4 | Error/log safety | Check application output/logs during login | Passwords are not printed | No credentials appear in logs |

### 8.10 Input validation tests

| ID | Field | Invalid input | Expected behaviour |
|---|---|---|---|
| V1 | `phone` | Too short, letters only, or extremely long string | Rejected with safe error |
| V2 | `emergency_contact` | Empty or too long | Rejected with safe error |
| V3 | `email` | Invalid email format | Rejected for admin update |
| V4 | `department` | Value outside approved list | Rejected |
| V5 | `employment_status` | Value outside `active`, `on_leave`, `terminated` | Rejected |
| V6 | `salary_band` | Value outside `A`–`E` | Rejected |
| V7 | `private_hr_notes` | More than maximum allowed length | Rejected |
| V8 | `record id` | Non-numeric or missing ID for admin edit | Request handled safely |

Validation must happen server-side before database updates.

### 8.11 Auditability tests

| ID | Area tested | Test steps | Expected behaviour | Pass criteria |
|---|---|---|---|---|
| A1 | Employee update audit | Alice updates her phone | `last_updated_by` becomes Alice's user ID | Audit identifies Alice |
| A2 | Admin update audit | Admin updates Bob's record | `last_updated_by` becomes admin user ID | Audit identifies admin |
| A3 | Timestamp update | Any successful update occurs | `last_updated_at` changes | Timestamp reflects latest update |
| A4 | Failed update audit | Invalid update is submitted | Audit fields do not change | Failed actions do not create false record updates |

### 8.12 Additional feature tests

The current preferred additional security features are:

1. Login rate limiting / temporary lockout
2. Session timeout after inactivity

#### 8.12.1 Login rate limiting / temporary lockout tests

| ID | Test steps | Expected behaviour | Pass criteria |
|---|---|---|---|
| AF1 | Submit 1–4 wrong passwords for the same username + client IP | Login remains available | User is not locked too early |
| AF2 | Submit 5 wrong passwords for the same username + client IP | Login is temporarily blocked for that combination | Lockout is activated |
| AF3 | Submit correct password during lockout from the same client IP | Login remains blocked | Lockout cannot be bypassed during the lockout window |
| AF4 | Check same username from a different IP in unit test | Not locked | Lockout is not global for the username alone |
| AF5 | Check different username from the same IP in unit test | Not locked | Lockout does not block unrelated users from the same IP |
| AF6 | Successful login for the matching username + IP | Failed counter resets for that combination | Future failures for that combination start from zero |

#### 8.12.2 Session timeout tests

| ID | Test steps | Expected behaviour | Pass criteria |
|---|---|---|---|
| AF7 | Log in and make a request before timeout | Session remains valid | User stays authenticated |
| AF8 | Log in and wait beyond timeout | Session is invalidated | User must log in again |
| AF9 | Access protected route after timeout | Redirect to login | Expired session cannot access data |
| AF10 | Log in again after timeout | New session is created | User can continue normally |

If implementation time becomes a constraint, security event logging is the backup additional feature.

### 8.13 Security event logging backup tests

If security event logging is selected instead of session timeout, the following tests will apply.

| ID | Test steps | Expected behaviour | Pass criteria |
|---|---|---|---|
| L1 | Successful login | Security event is recorded | Event type and timestamp exist |
| L2 | Failed login | Security event is recorded | Failed attempt is traceable |
| L3 | Employee attempts admin route | Access denied event is recorded | Suspicious action is traceable |
| L4 | Employee updates own record | Update event is recorded | Record update is traceable |
| L5 | CSRF validation fails | CSRF failure event is recorded | Security failure is traceable |
| L6 | Inspect logged details | No passwords, tokens, or sensitive HR notes are stored | Logs do not expose secrets |

### 8.14 Testing evidence for submission

Testing evidence may include:

- screenshots of successful and failed logins;
- screenshots of employee and admin views;
- screenshots showing denied access;
- screenshots showing validation errors;
- database screenshots or query output showing audit fields;
- database query output showing hashed sessions, user IDs, expiry times, and logout deletion;
- short notes explaining manual test results;
- optional Go test output if unit tests are added;
- automated test output for the login limiter showing username + client IP lockout scope.
- automated test output for CSRF token acceptance, invalid token rejection, missing token rejection, and deleted token rejection;

The README should explain how to run the application and which demo accounts to use.

### 8.15 Pass/fail recording format

During implementation, test results can be recorded using this format:

| Test ID | Date | Result | Evidence / notes |
|---|---|---|---|
| F1 | TBD | Pass/Fail | TBD |
| AC1 | TBD | Pass/Fail | TBD |
| SQL1 | TBD | Pass/Fail | TBD |
| XSS1 | TBD | Pass/Fail | TBD |
| CSRF1 | TBD | Pass/Fail | TBD |
| AF1 | TBD | Pass/Fail | TBD |

This will make the report's Testing and Results section easier to write later.

---

## 9. Implementation Roadmap

This section defines the planned implementation order. The goal is to build the system in small, testable stages, applying security controls consistently rather than adding them at the end.

The implementation should follow this order:

1. Project structure
2. Database schema and seed data
3. Basic HTTP server and templates
4. Authentication and sessions
5. Employee record view
6. Employee record update
7. Admin record list and view
8. Admin record update
9. CSRF protection
10. Input validation
11. Additional security features
12. Security headers and runtime configuration hardening
13. Testing and evidence collection
14. README and final report preparation

### 9.1 Phase 1: Project structure

Create the basic Go project structure.

Planned structure:

```text
northgate-srms/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── auth/
│   ├── csrf/
│   ├── handlers/
│   ├── middleware/
│   ├── storage/
│   ├── validation/
│   └── security/
├── templates/
├── static/
├── README.md
├── DESIGN.md
├── go.mod
└── .gitignore
```

Purpose:

- keep responsibilities separated;
- avoid placing all logic in one file;
- make authentication, storage, validation, and handlers easier to test and maintain.

Expected outcome:

```text
The project compiles and has a clean folder structure.
```

### 9.2 Phase 2: Database schema and seed data

Implement SQLite database setup.

Tasks:

- create `users` table;
- create `employee_records` table;
- create `sessions` table for database-backed session storage;
- add seed users:
  - `admin`
  - `hrmanager`
  - `alice`
  - `bob`
- store only bcrypt password hashes;
- create linked employee records for each user.

Expected outcome:

```text
The application can initialise the database and create demo data safely.
```

Security focus:

- no plaintext passwords in the database;
- foreign key relationship between users and employee records;
- audit fields present from the start.

### 9.3 Phase 3: Basic HTTP server and templates

Create the initial web server using Go `net/http`.

Tasks:

- start HTTP server;
- add route registration;
- add basic templates;
- render pages using `html/template`;
- create simple layout or shared template structure if useful.

Initial routes:

```text
GET /
GET /login
```

Expected outcome:

```text
The server runs locally and renders basic pages.
```

Security focus:

- use `html/template`, not unsafe manual HTML construction;
- avoid exposing internal errors to the user.

### 9.4 Phase 4: Authentication and sessions

Implement login and logout.

Tasks:

- build login form;
- process login using username and password;
- retrieve user with prepared statement;
- compare password using bcrypt;
- create unpredictable session ID;
- store sessions server-side in SQLite;
- store only a SHA-256 hash of the session ID in the database;
- link each session to the authenticated `user_id`;
- record session creation time, absolute expiry time, and last activity time;
- set session cookie with secure attributes;
- implement logout.

Routes:

```text
GET  /login
POST /login
POST /logout
```

Expected outcome:

```text
Users can log in and log out securely.
```

Security focus:

- password hashes only;
- generic login failure messages;
- `HttpOnly` and `SameSite` cookie attributes;
- session invalidation on logout.
- database-backed sessions that survive server restarts;
- raw session tokens are not stored in the database.

### 9.5 Phase 5: Employee record view

Implement employee access to their own record.

Route:

```text
GET /record
```

Tasks:

- require authentication;
- retrieve current user from session;
- fetch employee record using `currentUser.id`;
- display only the authenticated employee's own record.

Expected outcome:

```text
An employee can view their own HR record only.
```

Security focus:

- do not accept employee record ID from the browser;
- use server-side session identity;
- prevent IDOR by design.

### 9.6 Phase 6: Employee record update

Implement employee update of low-risk fields.

Routes:

```text
GET  /record/edit
POST /record/update
```

Tasks:

- show editable form for:
  - `phone`
  - `emergency_contact`
- validate input server-side;
- update only those two fields;
- update `last_updated_by`;
- update `last_updated_at`;
- use prepared statements.

Expected outcome:

```text
Employees can update only their own phone and emergency contact.
```

Security focus:

- reject or ignore admin-only fields submitted by employees;
- use `WHERE user_id = currentUser.id`;
- prevent unauthorised modification of sensitive HR fields.

### 9.7 Phase 7: Admin record list and view

Implement HR admin record access.

Routes:

```text
GET /admin/records
GET /admin/records/view?id={id}
```

Tasks:

- require authentication;
- require admin role;
- list all employee records;
- allow admin to view one selected record.

Expected outcome:

```text
Admins can view all employee records. Employees cannot access admin pages.
```

Security focus:

- enforce admin role server-side;
- do not rely on hidden UI links;
- handle invalid or missing IDs safely.

### 9.8 Phase 8: Admin record update

Implement admin update functionality.

Routes:

```text
GET  /admin/records/edit?id={id}
POST /admin/records/update
```

Tasks:

- require authentication;
- require admin role;
- validate all editable fields;
- update permitted non-ID fields;
- prevent direct updates to:
  - `id`
  - `user_id`
  - `last_updated_by`
  - `last_updated_at`
- update audit fields automatically.

Expected outcome:

```text
Admins can update employee records while technical and audit fields remain system-controlled.
```

Security focus:

- explicit admin authorisation;
- server-side validation;
- prepared statements;
- automatic auditability.

### 9.9 Phase 9: CSRF protection

Add CSRF protection to all state-changing actions.

Protected routes:

```text
POST /login
POST /logout
POST /record/update
POST /admin/records/update
```

Tasks:

- generate CSRF token;
- store token server-side or associate it with the session;
- include token as hidden field in forms;
- verify token on POST requests;
- reject missing or invalid tokens.

Expected outcome:

```text
State-changing requests without valid CSRF tokens are rejected.
```

Security focus:

- no state-changing `GET` requests;
- all sensitive POST actions require CSRF validation.

### 9.10 Phase 10: Input validation

Centralise server-side validation.

Tasks:

- validate `phone`;
- validate `emergency_contact`;
- validate `email`;
- validate name fields;
- validate department whitelist;
- validate employment status whitelist;
- validate salary band whitelist;
- validate max lengths for HR notes.

Expected outcome:

```text
Invalid, malformed, or oversized input is rejected before database update.
```

Security focus:

- never trust form input;
- validate before database operations;
- return safe error messages.

### 9.11 Phase 11: Additional security features

Implement the selected additional security features.

Current selected features:

```text
Feature 1: Login rate limiting / temporary lockout
Feature 2: Session timeout after inactivity
```

#### Login rate limiting / temporary lockout

Tasks:

- track failed login attempts;
- scope failed attempts by normalised username + client IP;
- lock login temporarily after repeated failures for the same username + client IP;
- reset failed attempts for the matching username + client IP after successful login;
- use generic error messages.

Expected outcome:

```text
Repeated failed login attempts trigger a temporary lockout.
```

#### Session timeout after inactivity

Tasks:

- store `last_activity_at` in the SQLite `sessions` table;
- check inactivity on protected routes;
- invalidate expired sessions;
- redirect expired users to login.

Expected outcome:

```text
Inactive sessions expire automatically.
```

Backup feature if needed:

```text
Security event logging
```

### 9.12 Phase 12: Security headers and runtime configuration hardening

Add defence-in-depth headers and basic runtime configuration through environment variables.

Security headers:

- `Content-Security-Policy`
- `X-Content-Type-Options`
- `X-Frame-Options`
- `Referrer-Policy`

Runtime configuration:

| Variable | Purpose | Default | Security consideration |
|---|---|---|---|
| `PORT` | HTTP server port | `8080` | Should be validated and not default to privileged or unexpected ports |
| `DB_PATH` | SQLite database path | `northgate.db` | Should not point to sensitive system locations or be committed with real data |

Expected outcome:

```text
The application can be configured without changing source code while still using safe defaults.
```

Security focus:

- avoid hardcoding deployment-specific values where simple configuration is appropriate;
- treat environment variables as external input rather than blindly trusted values;
- avoid storing secrets, passwords, session IDs, CSRF tokens, or HR data in source code or logs;
- keep local database files and `.env` files out of version control;
- recognise that production deployment would require stricter configuration, HTTPS, protected secrets, and environment separation.

### 9.13 Phase 13: Testing and evidence collection

Run and record tests from the testing plan.

Tasks:

- test functional requirements;
- test access control;
- test SQL injection resistance;
- test XSS handling;
- test CSRF protection;
- test session behaviour;
- test password hashing;
- test input validation;
- test additional security features;
- capture evidence.

Expected outcome:

```text
Each required user story and security feature has clear test evidence.
```

Evidence may include:

- screenshots;
- terminal output;
- database query results;
- short testing notes;
- optional Go test output.

### 9.14 Phase 14: README and final report preparation

Prepare submission documentation.

README should include:

- project description;
- how to run the application;
- demo login credentials;
- security features implemented;
- known limitations.

Final report should use the design work from this document, especially:

- project definition;
- data model;
- access control matrix;
- route map;
- threat model;
- additional security feature decisions;
- testing results;
- evaluation and limitations.

Expected outcome:

```text
The project is ready for assessment submission with code, README, testing evidence, and report.
```

### 9.15 Implementation discipline

During implementation:

- build one feature at a time;
- commit after each stable milestone;
- do not add new features unless they support the assessment requirements;
- test each security control as soon as it is implemented;
- keep code simple, readable, and defensive;
- prioritise consistency over cleverness.

Suggested commit sequence:

```text
Add project structure             ✓
Add database schema and seed data ✓
Add basic HTTP server and templates ✓
Add authentication and sessions   ✓
Add database-backed sessions      ✓
Add employee record view          ✓
Add employee record update        ✓
Add admin record views            ✓
Add admin record update           ✓
Add CSRF protection               ✓
Add input validation              ✓
Add login rate limiting           ✓
Add session timeout               ✓
Add security headers              ✓
Add runtime configuration         ✓
Add testing evidence and README   ✓
```

---

## 10. Implementation Notes

This section documents decisions made or refined during implementation that deviate from or improve upon the original design.

### 10.1 CSRF protection for the login form

The original design used a single fixed key (`"login"`) in the CSRF token store for the login pre-session. This introduced a race condition: if two users loaded the login page concurrently, the second request would overwrite the first user's token in the shared map, causing the first user's subsequent login submission to be rejected with a CSRF error.

**Resolution:** A unique `preSessionID` is now generated per login page visit using `crypto/rand`. This ID is used as both the CSRF map key and the value stored in the `northgate_login_csrf` cookie. Each user visiting the login page receives their own independent token, eliminating the race condition entirely.

The `preSessionID` is explicitly deleted from the CSRF token store once it has been read from the cookie, including validation failure, authentication failure, locked account, and successful login paths. — to prevent orphaned entries accumulating in the store.

```
GET /login  → generatePreSessionID() → token stored at key=preSessionID, cookie value=preSessionID
POST /login → read cookie → validate token at key=cookie.Value → delete key=preSessionID
```

### 10.2 Username validation in the login handler

The original design specified that `validation.IsValidUsername` would be used to validate the submitted username during login. The initial implementation used manual length checks only. This was corrected so that the login handler now calls `validation.IsValidUsername`, which enforces both length bounds (3–30 characters) and character whitelist (letters, digits, dots, hyphens, underscores).

This does not affect the generic error message returned to the user — invalid usernames and wrong passwords produce the same response.

### 10.3 Audit field display

The `last_updated_by` field is stored as an integer foreign key referencing `users.id`. The original design did not specify how this should be rendered. During implementation, a `GetUsernameByID` lookup was added to the admin record view handler so that the username is displayed rather than the numeric ID, which is more meaningful for accountability purposes.

### 10.4 Login CSRF cookie expiry on success

On successful login, the `northgate_login_csrf` cookie is explicitly expired in the browser response in addition to the token being deleted from the server-side CSRF store. This ensures the client does not retain a stale cookie that no longer corresponds to any server-side token.


### 10.5 Database-backed hashed session storage

The original session implementation stored sessions in an in-memory Go map. This worked for authentication, but it had two limitations: sessions were lost whenever the server restarted, and session state could not be inspected or expired through the database layer.

**Resolution:** Sessions are now stored in a dedicated SQLite `sessions` table. The browser still receives a random session ID in the `northgate_session` cookie, but the database stores only `SHA-256(sessionID)` rather than the raw cookie value. Each session row is linked to `users.id` through `user_id` and includes `created_at`, `expires_at`, and `last_activity_at` timestamps.

```text
Browser cookie: raw random session ID
Server lookup: SHA-256(cookie value)
Database row: session_hash + user_id + created_at + expires_at + last_activity_at
```

This improves robustness because sessions survive server restarts and can be deleted during logout or expiry checks. It also reduces the impact of a database leak because the value stored in the database cannot be directly reused as a valid session cookie.

The implementation also preserves the existing inactivity timeout. On each authenticated request, the session is looked up by hash, checked against both absolute expiry and inactivity timeout, and then `last_activity_at` is updated. Logout removes the corresponding row from SQLite and expires the browser cookie.

This change was implemented instead of adding two-factor authentication because two-factor authentication was not required for the assessment once two additional security features had already been implemented. Database-backed hashed sessions were a more proportionate improvement for the current scope because they strengthened an existing core control without adding a new user-facing authentication flow.

### 10.6 Second administrator account added for assessment testing

During review, the seed data was updated to include a second administrator account. The final demo accounts now include two HR administrators (`admin` and `hrmanager`) and two regular employee accounts (`alice` and `bob`). This ensures the application fully supports testing both administrator and regular-user behaviour with more than one account in each role.

### 10.7 Rate limiting updated to use username and client IP

The initial login rate limiter tracked failed attempts by username only. This was simple and effective against repeated guessing, but it introduced a denial-of-service weakness: an attacker could repeatedly submit failed logins for another user's username and temporarily lock that user out.

**Resolution:** The limiter now scopes failed attempts by the combination of normalised username and client IP. The client IP is taken from the remote address observed by the Go server using `r.RemoteAddr`. This means a lockout applies to a specific username + client IP combination rather than to the username globally.

```text
Rate limit key: normalised username + "|" + client IP
Example: alice|127.0.0.1
```

This reduces the risk of deliberate account lockout abuse while still slowing repeated password guessing. The implementation remains intentionally simple for the local assessment prototype. In a production deployment behind a trusted reverse proxy, client IP extraction would need to be reviewed carefully and configured using trusted forwarding headers such as `X-Forwarded-For`, rather than blindly trusting client-supplied headers.

Automated unit tests were added for the login limiter to confirm that a lockout affects the same username + IP combination, does not affect the same username from a different IP, and does not affect a different username from the same IP.
