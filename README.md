# Northgate Stores – Secure HR Records System

Northgate Stores – Secure HR Records System is a small secure web application built for the Software and Web Security assignment.

The system represents a fictional retail company that manages sensitive employee HR records. It supports two roles:

- `employee`
- `admin`

Employees can log in, view their own HR record, and update only low-risk contact fields. HR administrators can view and update all employee records through admin-only routes.

The project is intentionally scoped as a small server-rendered Go web application so that the main focus remains on secure authentication, server-side access control, CSRF protection, input validation, auditability, and defensive handling of sensitive data.

---

## Technologies used

- Go
- `net/http`
- `html/template`
- SQLite
- `modernc.org/sqlite`
- `golang.org/x/crypto/bcrypt`

---

## How to run the application

From the project root:

```bash
go run ./cmd/server
```

Then open:

```text
http://localhost:8080
```

The application creates the SQLite database and demo data automatically if they do not already exist.

---

## Demo accounts

The following demo accounts are available for testing.

| Username | Password | Role |
|---|---|---|
| `admin` | `AdminPass123!` | admin |
| `alice` | `AlicePass123!` | employee |
| `bob` | `BobPass123!` | employee |

Passwords are shown here only for assessment testing. In the database, passwords are stored as bcrypt hashes, not as plaintext.

---

## Main features

### Employee features

Employees can:

- log in;
- log out;
- view their own HR record;
- update only their own:
  - phone;
  - emergency contact.

Employees cannot:

- view other employee records;
- access admin routes;
- update address, email, salary band, employment status, private HR notes, or audit fields.

### Admin features

HR administrators can:

- log in;
- log out;
- view all employee records;
- view individual employee records;
- view private HR notes;
- update permitted HR fields for any employee record.

Admins cannot directly edit technical or audit fields such as:

- `id`;
- `user_id`;
- `last_updated_by`;
- `last_updated_at`.

These fields are controlled by the system.

---

## Data model

The system uses two core tables:

- `users`
- `employee_records`

The relationship is one-to-one:

```text
users.id → employee_records.user_id
```

This separates authentication data from sensitive HR record data.

### `users`

Stores authentication and role information:

- `id`
- `username`
- `password_hash`
- `role`
- `is_active`
- `created_at`

### `employee_records`

Stores HR record data:

- `id`
- `user_id`
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
- `last_updated_by`
- `last_updated_at`

---

## Security controls implemented

### Password hashing

Passwords are hashed using bcrypt before being stored in SQLite.

The application never stores plaintext passwords.

### Server-side sessions

After successful login, the application creates a random server-side session ID and stores the authenticated user's ID, username, and role in server-side session state.

The browser receives only a session cookie containing the random session ID.

### Cookie security

The session cookie uses:

- `HttpOnly`
- `SameSite=Lax`
- path `/`

The `Secure` flag is not enabled in local development because the application runs over `http://localhost`. In a production HTTPS deployment, `Secure` should be enabled.

### Role-based access control

Access control is enforced server-side.

Employees can access only their own records. Admin-only routes check that the authenticated user's role is `admin` before retrieving or modifying records.

The application does not rely on hidden UI links as a security boundary.

### IDOR / Broken Access Control protection

Employee record access uses the authenticated user's ID from the server-side session:

```text
currentUser.id → employee_records.user_id
```

Employees do not choose which record ID to access.

Admin routes may use record IDs from the URL or form body, but only after the server has confirmed that the authenticated user is an admin.

### SQL injection protection

All database operations use parameterised queries with placeholders such as `?`.

User input is not concatenated directly into SQL strings.

### CSRF protection

All state-changing POST routes are protected with CSRF tokens:

- `POST /login`
- `POST /logout`
- `POST /record/update`
- `POST /admin/records/update`

Authenticated actions bind the CSRF token to the server-side session ID.

For the login form, where no authenticated session exists yet, the application uses a temporary login CSRF cookie and a server-generated unpredictable token. The token is validated server-side before credentials are processed.

### XSS protection

The application renders pages using Go `html/template`, which provides contextual output escaping.

Dynamic values are not manually written into HTML using unsafe string concatenation.

### Input validation

Input validation is centralised in the `internal/validation` package.

The system validates:

- usernames;
- names;
- email format;
- phone format;
- address length;
- emergency contact length;
- department whitelist;
- employment status whitelist;
- salary band whitelist;
- maximum length for HR notes.

Browser-side validation is used for usability, but it is not treated as a security boundary. Important validation is repeated server-side before database updates.

### Audit fields

Every employee record update automatically updates:

- `last_updated_by`
- `last_updated_at`

These fields are controlled by the server and cannot be directly edited through forms.

### Additional security feature 1: login rate limiting

The application temporarily locks login attempts for a username after repeated failed login attempts.

Current behaviour:

```text
5 failed login attempts → temporary lockout
```

During lockout, even the correct password is rejected with the same generic error message.

This reduces the risk of brute-force password guessing.

### Additional security feature 2: session timeout after inactivity

Sessions expire after inactivity.

Current behaviour:

```text
15 minutes of inactivity → session invalidated
```

During development, a shorter timeout was used to test this behaviour efficiently.

### Security headers

The application adds defensive HTTP security headers:

- `Content-Security-Policy`
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: same-origin`

These provide defence in depth and reduce the impact of browser-based attacks.

---

## Testing

The system was tested manually and with Go unit tests.

### Manual testing included

- valid and invalid login;
- logout;
- employee own-record access;
- employee update of permitted fields;
- admin list/view/update functionality;
- employee denial from admin routes;
- CSRF token validation;
- SQL injection-style login input;
- XSS-style input rendering;
- input validation errors;
- login rate limiting;
- session timeout;
- password hash inspection;
- security header inspection.

### Unit tests

Validation logic is tested using Go table-driven tests.

Run all tests with:

```bash
go test ./...
```

---

## Known limitations

This project is an assessment prototype and is intentionally scoped.

Known limitations:

- no public registration;
- no web-based user creation;
- no password reset workflow;
- no multi-factor authentication;
- no file uploads;
- no cloud deployment;
- sessions are stored in memory, so they are lost when the server restarts;
- the SQLite database is local to the application;
- the UI is intentionally simple and not production-styled.

In a production system, future improvements could include persistent session storage, HTTPS deployment with `Secure` cookies, stronger account lifecycle management, password reset, MFA, structured security logging, and more detailed audit history.