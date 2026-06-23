# Security Account, MFA, and Audit Design

## Background

webcron currently uses a lightweight single-system account model:

- `t_user` stores username, email, MD5 password hash, salt, login time, login IP, and status.
- Login sets an `auth` cookie after username and password validation.
- There is no account management UI.
- Task execution logs record command execution output, but not which user performed manual operations.
- Task and group changes are not audited.

The new security work should make later incidents traceable to an exact user, action, time, request source, and affected record.

## Goals

- Add account management so administrators can create and manage users.
- Enforce MFA for all users with authenticator-app TOTP.
- Record manual and security-sensitive operations with exact actor and timestamp.
- Record task create, edit, delete, status changes, and manual execution details.
- Keep the first phase small enough to ship safely on this legacy Beego app.

## Non-Goals For Phase 1

- No complex role-based permission system.
- No SSO.
- No WebAuthn/passkey support.
- No deep command-output redaction.
- No change to scheduled task execution semantics.

All logged-in users in phase 1 can manage tasks, users, MFA reset, and audit logs. A future phase can add roles.

## User And MFA Data Model

Add fields to `t_user`:

```sql
ALTER TABLE `t_user`
  ADD COLUMN `mfa_secret` varchar(64) NOT NULL DEFAULT '' COMMENT 'TOTP secret' AFTER `salt`,
  ADD COLUMN `mfa_enabled` tinyint(4) NOT NULL DEFAULT '0' COMMENT '1 enabled, 0 not enabled' AFTER `mfa_secret`,
  ADD COLUMN `mfa_confirmed_at` int(11) unsigned NOT NULL DEFAULT '0' COMMENT 'MFA confirmation time' AFTER `mfa_enabled`,
  ADD COLUMN `create_time` int(11) unsigned NOT NULL DEFAULT '0' COMMENT '创建时间' AFTER `status`;
```

Existing users with `mfa_enabled = 0` must bind MFA on their next successful password login.

## Audit Data Model

Create `t_operation_log`:

```sql
CREATE TABLE `t_operation_log` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` int(11) NOT NULL DEFAULT '0' COMMENT '操作者ID',
  `user_name` varchar(50) NOT NULL DEFAULT '' COMMENT '操作者用户名',
  `action` varchar(50) NOT NULL DEFAULT '' COMMENT '操作类型',
  `target_type` varchar(50) NOT NULL DEFAULT '' COMMENT '对象类型',
  `target_id` int(11) NOT NULL DEFAULT '0' COMMENT '对象ID',
  `result` tinyint(4) NOT NULL DEFAULT '0' COMMENT '0成功 -1失败',
  `ip` varchar(45) NOT NULL DEFAULT '' COMMENT 'IP',
  `user_agent` varchar(255) NOT NULL DEFAULT '' COMMENT 'User-Agent',
  `request_method` varchar(10) NOT NULL DEFAULT '' COMMENT '请求方法',
  `request_path` varchar(255) NOT NULL DEFAULT '' COMMENT '请求路径',
  `detail` mediumtext NOT NULL COMMENT 'JSON详情',
  `create_time` int(11) unsigned NOT NULL DEFAULT '0' COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_time` (`user_id`, `create_time`),
  KEY `idx_action_time` (`action`, `create_time`),
  KEY `idx_target` (`target_type`, `target_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
```

`detail` stores compact JSON snapshots. Examples:

- Task create: full task fields after creation.
- Task edit: `before` and `after` task snapshots.
- Task delete: task snapshot before deletion.
- Manual run: task snapshot and generated `task_log_id`.
- Login failure: username, failure reason category, IP, User-Agent.

## MFA Login Flow

1. User submits username and password.
2. If password is invalid, write a login failure audit record and show the existing generic error.
3. If account is disabled, write a login failure audit record.
4. If password is valid but MFA is not enabled:
   - Store a short-lived pending-login state in session.
   - Redirect to MFA setup page.
   - Generate a TOTP secret and QR code/provisioning URI.
   - User scans with authenticator app and submits a 6-digit code.
   - On valid code, save `mfa_secret`, set `mfa_enabled = 1`, set `mfa_confirmed_at`, write audit log, and log in.
5. If MFA is enabled:
   - Redirect to MFA verify page or render the MFA code field.
   - On valid 6-digit code, set the `auth` cookie and write login success audit log.
   - On invalid code, write MFA failure audit log.

Recommended dependency: `github.com/pquerna/otp/totp`.

## Account Management

Add `UserController` and views:

- `/user/list`: list users with username, email, status, MFA status, last login, last IP.
- `/user/add`: create a user with username, email, temporary password.
- `/user/edit`: update email/status and optionally reset password.
- `/user/reset-mfa`: clear MFA secret and status so the user must bind again on next login.

Validation:

- Username is required and unique.
- Password is required for new users and must be at least 8 characters.
- Email format should be validated when present.
- The current user should not be able to disable themselves.

Every account operation writes an operation log.

Because MFA is mandatory, there should be no normal UI action to disable MFA.
Reset MFA only clears the current binding and forces the user to bind a new authenticator on the next login.
If all users are locked out, recovery should be handled as an explicit server/database maintenance action, not as a web UI bypass.

## Audit UI

Add `OperationLogController` and views:

- `/operation-log/list`
- `/operation-log/view`

List filters:

- Time range.
- User.
- Action.
- Target type.
- Target ID.
- Result.

Detail view shows:

- Operator.
- Exact server time.
- IP and User-Agent.
- Request method/path.
- Action and target.
- Formatted JSON detail.

## Operations To Audit In Phase 1

Authentication:

- `login_success`
- `login_failed`
- `logout`
- `mfa_setup_success`
- `mfa_setup_failed`
- `mfa_verify_failed`

User management:

- `user_create`
- `user_update`
- `user_enable`
- `user_disable`
- `user_reset_password`
- `user_reset_mfa`

Task management:

- `task_create`
- `task_update`
- `task_start`
- `task_pause`
- `task_delete`
- `task_batch_active`
- `task_batch_pause`
- `task_batch_delete`
- `task_manual_run`
- `task_log_delete`

Group management:

- `group_create`
- `group_update`
- `group_delete`
- `group_batch_delete`

## Implementation Shape

Add model:

- `app/models/operation_log.go`

Add controller helpers:

- `BaseController.audit(action, targetType string, targetID int, result int, detail interface{})`
- `BaseController.requestMeta()` for IP, User-Agent, method, path.

Use the helper at the controller boundary after the operation outcome is known. For failed validation, write a failure log only for security-sensitive events such as login/MFA and destructive actions.

For task edit/delete, load the original task before mutation and store a snapshot. For batch operations, write one operation log per affected object and include the batch action name.

Register new models in `app/models/init.go`.

Add SQL migration file:

- `migrations/2026-06-23-security-account-mfa-audit.sql`

## Deployment Notes

This change includes database migrations and new Go code, so deployment requires:

1. Backup database.
2. Apply SQL migration.
3. Build Linux package locally.
4. Deploy binary and new/changed views.
5. Restart service.
6. Log in with the existing admin account and bind MFA.

Existing admin users will be forced into MFA setup after password login.

## Testing Plan

Manual checks:

- Existing admin can log in with password and is forced to bind MFA.
- Admin cannot enter the app before MFA is confirmed.
- Admin can create a new user.
- New user is forced to bind MFA on first login.
- Invalid password and invalid MFA attempts are logged without exposing sensitive details.
- Task create/edit/delete/start/pause/manual run each creates audit records.
- Batch task operations create per-task audit records.
- Operation log list filters by time, user, action, target type, target ID, and result.
- Operation log detail shows readable JSON.

Automated checks where practical:

- TOTP verification helper accepts valid code and rejects invalid code.
- User creation validation rejects duplicate username and weak password.
- Task snapshot serialization includes fields needed for incident review.

## Open Follow-Up For Phase 2

- Role-based permissions.
- Password hash upgrade from MD5+salt to bcrypt/argon2.
- CSRF protection for state-changing actions.
- Login rate limiting.
- Session invalidation after password/MFA reset.
