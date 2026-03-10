# GOG (Google CLI) Tool

**GOG** is a powerful Google Workspace integration that gives YOLO direct access to Gmail, Calendar, Drive, Contacts, Sheets, Docs, Slides, Tasks, and more via the official `gog` CLI.

## What You Can Do

### 📧 Gmail Operations
- Search emails: `gmail search 'newer_than:7d' --max 10`
- Send emails: `gmail send --to user@example.com --subject "Hi" --body "Hello"`
- Create drafts: `gmail drafts create --to user@example.com --subject "Draft" --body "Content"`
- Reply to messages with threading support

### 📅 Calendar Operations
- List events: `calendar events primary --from 2026-03-10T00:00:00Z --to 2026-03-17T00:00:00Z`
- Create events: `calendar create primary --summary "Meeting" --from 2026-03-15T10:00:00Z --to 2026-03-15T11:00:00Z`
- Update events with custom colors
- List available calendar colors

### 📁 Google Drive
- List files: `drive ls --max 20`
- Search files: `drive search "presentation" --max 10`
- View file metadata and permissions

### 👥 Contacts
- List contacts: `contacts list --max 20`
- Search by name or email

### 📊 Google Sheets
- Read cells: `sheets get <sheetId> "Tab!A1:D10" --json`
- Update ranges: `sheets update <sheetId> "Tab!A1:B2" --values-json '[["A","B"],["1","2"]]'`
- Append rows: `sheets append <sheetId> "Tab!A:C" --values-json '[["x","y","z"]]''`

### 📝 Google Docs & Slides
- Export documents: `docs export <docId> --format txt --out ./doc.txt`
- View document content: `docs cat <docId>`

## Tool Usage Examples

```json
{
  "name": "gog",
  "arguments": {
    "command": "gmail search 'inbox:unread newer_than:1d' --max 5"
  }
}
```

```json
{
  "name": "gog",
  "arguments": {
    "command": "calendar events primary --from 2026-03-10T00:00:00Z --to 2026-03-17T23:59:59Z"
  }
}
```

```json
{
  "name": "gog",
  "arguments": {
    "command": "drive ls --max 10"
  }
}
```

## Configuration

The `gog` CLI is already configured on this system with OAuth credentials:
- Account: scott@griepentrog.com  
- Services: calendar, contacts, docs, drive, gmail, sheets
- Token expires: ~1 hour (auto-refresh)

## Output Format

By default, GOG outputs JSON for easy parsing. Add `--json` flag explicitly if needed.

## Limitations

- Requires OAuth setup (already configured)
- Some operations may require confirmation flags
- Large file downloads need explicit `--out` parameter
- Email sending should include recipient confirmation

## Learn More

- Documentation: https://gogcli.sh
- Source: https://github.com/danielmiessler/gog
- Installation: `brew install steipete/tap/gogcli`
