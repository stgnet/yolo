# Email Setup for YOLO

YOLO can send automated progress reports and emails from `yolo@b-haven.org` to `scott@stg.net`.

## Required Configuration

Set the following environment variable before running YOLO:

```bash
export EMAIL_PASSWORD="your-app-password"  # Gmail app-specific password
```

## Optional Configuration

```bash
export EMAIL_USERNAME="yolo@b-haven.org"      # Default: yolo@b-haven.org
export EMAIL_SMTP_HOST="smtp.gmail.com"       # Default: smtp.gmail.com  
export EMAIL_SMTP_PORT=587                    # Default: 587
```

## How It Works

YOLO has two email tools available:

1. **`send_report`** - Send a progress report email
   - Defaults to scott@stg.net
   - Subject defaults to "YOLO Progress Report"

2. **`send_email`** - Send a custom email  
   - Specify subject, body, and recipient (optional)
   - Also defaults to scott@stg.net if no recipient specified

## Example Usage (from within YOLO context)

```go
// In your agent code:
send_report("Completed fixing LimitedConcurrency deadlock")
send_email("Bug Report", "Found issue in module X", "scott@stg.net")
```

## Testing

Once EMAIL_PASSWORD is configured, you can test email sending by running YOLO and having it call the email tools. The first successful send will confirm everything is working!
