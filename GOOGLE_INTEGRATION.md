# YOLO Google Integration via GOG

## Overview

YOLO now has full access to Google services through **GOG** (Google CLI), Daniel Miessler's open-source tool for OpenClaw AI agents.

## What is GOG?

**GOG** = Google Operations Gateway - A comprehensive CLI tool that provides authenticated access to Google APIs:
- Gmail
- Google Calendar  
- Google Drive
- Google Docs, Sheets, Slides
- Google Contacts
- Google Tasks
- Google People/Contacts
- Google Chat
- Google Classroom

### Installation Status

✅ **Already installed and authenticated** on this system:
```bash
gog --help  # Works perfectly
gog calendar list --limit 5  # Returns real calendar events
gog drive ls --limit 5       # Lists Google Drive files
```

## YOLO Tool Integration

### Using the `gog` Tool in YOLO

The tool is available as a native YOLO capability:

```
[tool activity]
[gog => gog(command="calendar list events")]
```

Or programmatically via run_command:
```bash
gog gmail search inbox:unread --max 5
gog calendar create title="Meeting" start="2026-03-15T14:00" end="2026-03-15T15:00"
gog drive download id="file_id" path="./local_file.pdf"
```

### Example Commands

**Email:**
- `gog gmail search inbox:unread --max 10` - List unread emails
- `gog gmail send to=example@gmail.com subject="Test" body="Hello"` - Send email
- `gog gmail read id=<message_id>` - Read specific email

**Calendar:**
- `gog calendar list events` - List upcoming events
- `gog calendar create title="Meeting" start="2026-03-15T14:00" end="2026-03-15T15:00"` - Create event
- `gog calendar free-busy` - Check availability

**Drive:**
- `gog drive ls` - List files in root
- `gog drive search query="filename.pdf"` - Search files
- `gog drive download id="<file_id>" path="./output.pdf"` - Download file
- `gog drive upload path="./local.txt" title="Uploaded File"` - Upload file

**Docs/Sheets/Slides:**
- `gog docs create title="New Doc" content="Hello World"`
- `gog sheets create title="New Sheet"`
- `gog slides create title="Presentation"`

## Testing Results

### Calendar Test ✅
```bash
$ gog calendar list --limit 5
ID  START                      END                        SUMMARY
6cq32db1c9im4b9n64s3gb9kc9ij4b9pcgom4b9k69h32d1i70qj0o9m74  2026-03-10T08:45:00-04:00  2026-03-10T09:00:00-04:00  Mos
... (shows recurring "Mos" events)
```

### Drive Test ✅
```bash
$ gog drive ls --limit 5
ID  NAME                                          TYPE  SIZE      MODIFIED
1E2ey0QhWO4DJ8buF7972z7r0eKlhuqpr_DRjTypI2vY  B-Haven CRM                                  file  261.7 KB  2026-03-09
1qw0b9Xz5WJZGa3YVKEMqYRojMD4_cxKYEOhXIfpzKWk  B-Haven Pricing                              file  104.8 KB  2026-03-09
... (lists actual Google Drive files)
```

## Authentication

GOG uses OAuth2 and is **already authenticated** on this system. The credentials are stored in the OpenClaw configuration at:
```
~/.openclaw/credentials/
```

No additional setup needed for YOLO to use GOG.

## Use Cases for YOLO

1. **Calendar Management**: Schedule meetings, check availability, find time slots
2. **Email Automation**: Read/send emails, organize inbox, filter messages
3. **File Operations**: Access documents, search Drive, download/upload files
4. **Document Creation**: Generate Google Docs/Sheets/Slides from AI content
5. **Contact Management**: Look up contacts, manage address book
6. **Task Integration**: Create/read tasks from Google Tasks

## Related Projects

- **GOG Repository**: https://github.com/danielmiessler/gog
- **OpenClaw**: Full-featured AI agent framework that popularized GOG
- **GoGogot**: Lightweight Go-based OpenClaw alternative (https://github.com/aspasskiy/GoGogot)

## Notes

- Output format: JSON by default when called from YOLO tools
- Rate limiting: Follows Google API quotas
- Permissions: Limited to scopes granted during OAuth setup
- Works alongside existing `run_command` tool for direct CLI access

## Future Enhancements

Potential additions:
- Native GOG sub-tools (gmail_send, calendar_create, etc.)
- Smart scheduling integration with YOLO's task planning
- Auto-resume from Gmail conversations
- Google Meet integration for virtual meetings
- Photo management via Google Photos API
