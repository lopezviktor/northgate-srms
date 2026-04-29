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
- Password reset by email
- Multi-factor authentication
- File uploads
- Payroll processing
- Exact salary storage
- External HR integrations
- Complex UI styling
- Cloud deployment
- Kubernetes or production infrastructure

