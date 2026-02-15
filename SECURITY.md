# Security Documentation

This document describes the security architecture, practices, and known considerations for Twilight BBS.

## Authentication & Authorization

### SSH Authentication
- **SSH Server**: Validates credentials during SSH handshake against user database
- **Pre-authentication**: After SSH validation, credentials are passed to BBS for session establishment
- **Password Storage**: SSH passwords are temporarily stored in memory during connection setup for BBS authentication
  - This is by design to support the Lua menu system
  - Passwords are not logged or persisted to disk
  - Cleartext storage is limited to the connection lifetime

### User Authentication
- **Password Hashing**: bcrypt with cost factor 12
- **Session Management**: User sessions are managed per-node connection
- **Security Levels**: Role-based access control with numeric security levels
  - New users: Level 10 (default)
  - Regular users: Level 50-100
  - Sysops: Level 250+

## Input Validation

### Validation Framework
Location: `internal/scripting/validation.go`

All user inputs processed through Lua APIs are validated with:
- **Length limits**: Prevent buffer overflow and DoS attacks
- **Character validation**: UTF-8 encoding, no control characters
- **Format validation**: Email, username patterns
- **Path traversal prevention**: Filename and path sanitization

### Input Limits
- Username: 2-30 characters, alphanumeric + underscore/hyphen
- Password: 6-128 characters minimum
- Email: 128 characters, basic format validation
- Real name/Location: 60 characters
- Message subject: 128 characters
- Message body: 8,192 characters (8KB)
- Chat messages: 512 characters
- Filenames: 255 characters, no path separators

### Protected Operations
1. **User Registration**: Username, password, email, profile fields
2. **Message Posting**: Subject and body validation
3. **Chat Messages**: Private and broadcast messages
4. **File Operations**: Filename validation prevents directory traversal
5. **Door Commands**: Shell metacharacter filtering

## File Transfer Security

### Path Validation
Location: `internal/transfer/validator.go`

File transfer operations use `PathValidator` to prevent unauthorized access:
- **Send Operations**: Validates file paths are within authorized file areas
- **Receive Operations**: Validates upload directories are within allowed locations
- **Path Traversal Prevention**: Resolves absolute paths and checks boundaries
- **Logging**: Blocked attempts are logged for security monitoring

### Usage
```go
// Configure allowed roots (from file area disk paths)
validator := transfer.NewPathValidator([]string{
    "/path/to/file/areas/general",
    "/path/to/file/areas/uploads",
})

// Add to transfer config
transferConfig.PathValidator = validator
```

**Note**: Path validation is **optional** in the current implementation for backward compatibility. 
To enable protection, initialize and configure the PathValidator when creating TransferConfig.

## Door (External Program) Security

### Command Validation
Location: `internal/door/dosemu.go`

Door commands are validated to prevent command injection:
- **Shell metacharacters**: Blocks `&`, `|`, `;`, `>`, `<`, `` ` ``, `$`
- **Control characters**: Blocks null bytes, newlines, carriage returns
- **Length limit**: 256 characters maximum
- **Placeholders**: Supports `{NODE}` and `{DROP}` safely

### DOSEMU Isolation
- Doors run in DOSEMU2 environment (Linux only)
- Separate DOS drive (C:) mapped to restricted directory
- Communication via drop files (DOOR.SYS, DORINFO1.DEF)
- No direct host system access

## Database Security

### SQL Injection Prevention
- **Parameterized Queries**: All database operations use prepared statements
- **No Dynamic SQL**: Query structure is fixed, only parameters vary
- **LIKE Escaping**: Special escaping for LIKE patterns in search operations

### Example
```go
// Safe - parameterized query
result, err := r.db.Exec(`
    INSERT INTO users (username, password_hash)
    VALUES (?, ?)
`, username, hash)
```

## Lua Scripting Security

### Sandbox Limitations
The Lua scripting environment is **not sandboxed** by design. This is intentional to allow:
- Full access to BBS APIs for menu development
- File system operations for doors and transfers
- Direct terminal control

### Security Model
- **Trust Required**: Lua scripts must be written by trusted administrators
- **No User Scripts**: End users cannot upload or execute Lua code
- **Menu Security**: Menu files should be read-only for BBS process
- **API Boundaries**: All dangerous operations require authentication

### API Access Control
1. **Authentication Check**: Most operations require logged-in user
2. **Security Levels**: Door access, file uploads, sysop menus
3. **Input Validation**: All user-provided data validated before use
4. **Rate Limiting**: Not implemented - relies on connection limits

## Network Security

### SSH Configuration
Location: `internal/server/ssh.go`

**Modern Ciphers** (preferred):
- chacha20-poly1305@openssh.com
- aes128-gcm@openssh.com, aes256-gcm@openssh.com
- aes128-ctr, aes192-ctr, aes256-ctr

**Legacy Ciphers** (for SyncTerm compatibility):
- aes128-cbc
- 3des-cbc

**Key Exchange**:
- Modern: curve25519-sha256, ECDH with NIST curves
- Legacy: diffie-hellman-group14-sha1 (for older clients)

**Recommendation**: Modern ciphers are preferred, but legacy support is enabled for BBSing software compatibility.

### Telnet Protocol
- **No Encryption**: Telnet is unencrypted by design
- **Recommendation**: Use SSH for sensitive operations
- **Option Negotiation**: Echo, SGA, NAWS, TTYPE supported

## Known Security Considerations

### 1. Plaintext Password Storage in SSH Handler
- **Status**: By design
- **Scope**: Temporary storage during connection lifetime
- **Mitigation**: Passwords not logged or persisted
- **Context**: Required for BBS session establishment via Lua menus

### 2. Legacy SSH Ciphers
- **Status**: Intentional for compatibility
- **Scope**: 3DES-CBC, AES-128-CBC enabled
- **Mitigation**: Modern ciphers offered first, deprecate when possible
- **Context**: Required for SyncTerm and older BBS terminal software

### 3. No Rate Limiting
- **Status**: Not implemented
- **Scope**: Message posting, chat, registration
- **Mitigation**: Node limits (max concurrent connections)
- **Recommendation**: Add per-user rate limiting for production use

### 4. Path Validator Optional
- **Status**: Not enforced by default
- **Scope**: File transfers can access any path if validator not configured
- **Mitigation**: Lua scripts must provide authorized paths
- **Recommendation**: Initialize PathValidator with file area roots

### 5. Error Message Information Disclosure
- **Status**: Some errors may leak information
- **Scope**: Authentication failures, file operations
- **Mitigation**: Generic error messages for auth, logging for debugging
- **Recommendation**: Review and sanitize error messages for production

## Security Checklist for Deployment

### Required
- [ ] Change default SSH host key
- [ ] Set strong sysop password
- [ ] Configure file area paths appropriately
- [ ] Review and secure Lua menu scripts
- [ ] Set appropriate file permissions (menus read-only)
- [ ] Configure firewall rules
- [ ] Enable logging and monitoring

### Recommended
- [ ] Initialize PathValidator for file transfers
- [ ] Disable legacy SSH ciphers if possible
- [ ] Implement rate limiting for message/chat APIs
- [ ] Set up automated security updates
- [ ] Regular security audits of Lua scripts
- [ ] Monitor logs for suspicious activity
- [ ] Backup database regularly

### Optional (High Security)
- [ ] Disable Telnet (SSH only)
- [ ] Implement IP-based rate limiting
- [ ] Add 2FA for sysop accounts
- [ ] Run BBS as non-root user with minimal permissions
- [ ] Use SELinux or AppArmor policies
- [ ] Implement intrusion detection

## Reporting Security Issues

If you discover a security vulnerability:
1. **Do not** create a public GitHub issue
2. Email the maintainer privately with details
3. Allow reasonable time for patching before disclosure
4. Coordinated disclosure is appreciated

## Security Maintenance

### Regular Updates
- Keep Go dependencies updated (`go get -u all`)
- Monitor security advisories for:
  - modernc.org/sqlite
  - golang.org/x/crypto/ssh
  - gopher-lua

### Dependency Scanning
Run regular security scans:
```bash
# Check for known vulnerabilities using official Go tool
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

# Update dependencies
go get -u all
go mod tidy
```

### Code Auditing
- Review Lua scripts for injection risks
- Check database queries for SQL injection
- Audit file operations for path traversal
- Test input validation edge cases

## References

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Go Security](https://go.dev/doc/security)
- [SSH RFC 4251](https://tools.ietf.org/html/rfc4251)
- [bcrypt](https://en.wikipedia.org/wiki/Bcrypt)
