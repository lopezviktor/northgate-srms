# Northgate Stores – Secure HR Records System

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
- Prepared SQL statements
- Safe output rendering with Go html/template
- Input validation
- Audit fields:
  - last_updated_by
  - last_updated_at
- Two additional security features to be selected later

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

The system uses two core database tables:

- `users`
- `employee_records`

The `users` table stores authentication and role information. The `employee_records` table stores the sensitive HR record linked to each user. This separation keeps login credentials and HR data in different parts of the data model, making the design easier to reason about and safer to maintain.

The relationship between the two tables is one-to-one:

```text
users.id  →  employee_records.user_id
```

Each user has one employee record, and each employee record belongs to one user.

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

### 3.3 Field editability rules

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

### 3.4 Automatic audit behaviour

Whenever an employee record is updated, the system must automatically update:

- `last_updated_by`
- `last_updated_at`

`last_updated_by` stores the ID of the currently authenticated user who made the change.

`last_updated_at` stores the timestamp when the change was made.

This supports accountability and allows administrators to see who last changed a record and when.

### 3.5 Data validation rules

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

### 3.6 Example seed users

The system will include test accounts for assessment purposes.

| Username | Role | Purpose |
|---|---|---|
| `admin` | admin | HR administrator account |
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
| GET | `/health` | Basic development health check | Public or local only | No | Optional; should not expose sensitive information |

### 5.5 Route security rules

#### Login

`POST /login` must:

- validate input length before processing credentials;
- use a prepared statement to retrieve the user by username;
- compare the submitted password with the stored password hash;
- reject inactive users;
- create a new unpredictable session ID after successful login;
- set the session cookie using secure attributes such as `HttpOnly` and `SameSite`;
- return a generic error message for failed login attempts.

#### Logout

`POST /logout` must:

- require a valid CSRF token;
- invalidate the server-side session;
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
| Session identifiers | Allow authenticated access while a user is logged in |
| Audit fields | Support accountability and record integrity |
| Admin functionality | Allows access to and modification of all records |
| Application availability | Users and HR admins need reliable access to records |
| System design and configuration | Weak configuration can expose data or weaken controls |

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
| Configuration/startup environment → application | Configuration values must not enable unsafe behaviour |

The server must treat all browser-supplied data as untrusted, including hidden fields, URL parameters, cookies, and form inputs.

### 6.3 Main risks and mitigations

| Risk | How it could happen | Impact | Mitigation |
|---|---|---|---|
| Broken Access Control / IDOR | An employee changes a URL or form value to access another employee's record | Exposure or unauthorised modification of HR data | Server-side role checks; employee records loaded by `currentUser.id`; admin routes protected by explicit admin checks |
| Forced browsing | An employee manually visits `/admin/records` or another admin URL | Unauthorised access to admin functions | Admin middleware/checks on every admin route; UI hiding is not treated as a security control |
| SQL Injection | Malicious input is inserted into SQL queries | Login bypass, data exposure, or data modification | Prepared statements for all database queries; no string concatenation for SQL |
| Cross-Site Scripting (XSS) | HR notes or other stored fields contain HTML/JavaScript payloads | Browser executes attacker-controlled code, possibly exposing data or performing actions as the user | Render all pages through Go `html/template`; avoid unsafe template output; validate input length and format |
| Cross-Site Request Forgery (CSRF) | A malicious site tricks a logged-in user into submitting a state-changing request | Unwanted record updates or logout actions | CSRF token required on all state-changing `POST` routes; no state-changing `GET` requests; SameSite session cookies |
| Session hijacking or session misuse | A session cookie is stolen, guessed, or reused | Attacker acts as the logged-in user | Random session IDs; server-side sessions; HttpOnly and SameSite cookie flags; logout invalidates session |
| Weak password storage | Passwords stored in plaintext or weak hashes | Breach exposes real credentials | Store only password hashes using bcrypt; never log passwords |
| Brute-force login attempts | Attacker repeatedly guesses usernames and passwords | Account compromise | Candidate additional feature: login rate limiting or temporary lockout |
| Malformed or oversized input | User submits unexpected, very long, or invalid data | Validation bypass, crashes, inconsistent records, or stored malicious content | Server-side validation for length, format, and whitelist fields before database updates |
| Missing auditability | Records are changed without knowing who changed them | Loss of accountability and weaker integrity | Automatically update `last_updated_by` and `last_updated_at` on every record update |
| Information leakage through errors | Detailed database or server errors are shown to users | Attackers learn internal details | Generic user-facing error messages; detailed errors kept out of templates and not exposed to users |
| Security misconfiguration | Debug mode, unsafe defaults, or exposed files are left enabled | Internal information or sensitive data could be exposed | Avoid debug output in user responses; keep database and `.env` files out of version control; use safe defaults |
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
- Update audit fields automatically.
- Keep the project scope small enough for controls to be applied consistently.

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
If a username has 5 failed login attempts:
    temporarily block further login attempts for that username
    block duration: 2 minutes
```

Security benefit:

- reduces brute-force password guessing;
- slows automated login attacks;
- protects both employee and admin accounts;
- supports defence-in-depth around authentication.

Implementation approach:

- track failed login attempts server-side;
- store the number of failed attempts and lockout expiry time;
- reset failed attempts after a successful login;
- return a generic error message for failed and locked login attempts.

Testing approach:

| Test | Expected result |
|---|---|
| 1–4 failed attempts | Login remains available |
| 5 failed attempts | Login is temporarily blocked |
| Correct password during lockout | Login remains blocked |
| Correct password after lockout expires | Login succeeds |
| Successful login | Failed attempt counter resets |

Trade-off:

- legitimate users may be temporarily blocked after repeated mistakes;
- a username-based lockout could be abused to inconvenience another user;
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

- store `last_activity` in the server-side session;
- update `last_activity` after each valid authenticated request;
- compare the current time with `last_activity` on protected routes;
- invalidate the session if the inactivity limit has been exceeded.

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

### 7.9 Current decision

The current preferred choice is:

```text
Additional Feature 1: Login rate limiting / temporary lockout
Additional Feature 2: Session timeout after inactivity
```

Security event logging remains the strongest backup option if session timeout becomes unnecessarily complex during implementation.

CSP may still be implemented as a simple defensive header, but it should not be relied on as one of the two main additional features unless one of the preferred features is dropped.