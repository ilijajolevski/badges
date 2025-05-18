# Google OAuth Setup Guide for Badge Service

This guide provides step-by-step instructions for setting up Google OAuth authentication for the Badge Service application.

## Prerequisites

- A Google account
- Access to the [Google Cloud Console](https://console.cloud.google.com/)
- Badge Service application code

## Step 1: Create a Google Cloud Project

1. Go to the [Google Cloud Console](https://console.cloud.google.com/)
2. Click on the project dropdown at the top of the page
3. Click on "New Project"
4. Enter a project name (e.g., "Badge Service")
5. Click "Create"
6. Wait for the project to be created and then select it from the project dropdown

## Step 2: Configure OAuth Consent Screen

1. In the Google Cloud Console, navigate to "APIs & Services" > "OAuth consent screen"
2. Select "External" as the user type (unless you're using Google Workspace)
3. Click "Create"
4. Fill in the required information:
   - App name: "Badge Service"
   - User support email: Your email address
   - Developer contact information: Your email address
5. Click "Save and Continue"
6. On the "Scopes" page, click "Add or Remove Scopes"
7. Add the following scopes:
   - `openid`
   - `https://www.googleapis.com/auth/userinfo.email`
   - `https://www.googleapis.com/auth/userinfo.profile`
8. Click "Save and Continue"
9. On the "Test users" page, click "Add Users"
10. Add your email address and any other test users
11. Click "Save and Continue"
12. Review your settings and click "Back to Dashboard"

## Step 3: Create OAuth 2.0 Credentials

1. In the Google Cloud Console, navigate to "APIs & Services" > "Credentials"
2. Click "Create Credentials" and select "OAuth client ID"
3. Select "Web application" as the application type
4. Enter a name for the client (e.g., "Badge Service Web Client")
5. Add authorized JavaScript origins:
   - For development: `http://localhost:80`
   - For production: `https://badges.finki.edu.mk`
6. Add authorized redirect URIs:
   - For development: `http://localhost:80/auth/callback`
   - For production: `https://badges.finki.edu.mk/auth/callback`
7. Click "Create"
8. A popup will display your client ID and client secret. Save these values securely.

## Step 4: Configure Environment Variables

Set the following environment variables in your Badge Service application:

```bash
# Authentication type (always "oidc" for Google)
export AUTH_TYPE="oidc"

# Google OAuth settings
export OIDC_ISSUER_URL="https://accounts.google.com"
export OIDC_CLIENT_ID="your-client-id"
export OIDC_CLIENT_SECRET="your-client-secret"

# Redirect URL (must match one of the authorized redirect URIs)
# For development:
export OIDC_REDIRECT_URL="http://localhost:80/auth/callback"
# For production:
# export OIDC_REDIRECT_URL="https://badges.finki.edu.mk/auth/callback"

# Session settings
export SESSION_SECRET="generate-a-secure-random-string"
export SESSION_DURATION="24h"

# Initial superadmin email
export INITIAL_SUPER_ADMIN_EMAIL="badge-admin@gmail.com"
```

For production, it's recommended to use a secure method for managing environment variables, such as a `.env` file or a secrets management service.

## Step 5: Generate a Secure Session Secret

For the `SESSION_SECRET` environment variable, generate a secure random string:

```bash
# On Linux/macOS
openssl rand -base64 32

# On Windows with PowerShell
[Convert]::ToBase64String((New-Object Security.Cryptography.RNGCryptoServiceProvider).GetBytes(32))
```

## Step 6: Verify Configuration

1. Start the Badge Service application
2. Navigate to the login page at `/login`
3. Click the "Sign in with Google" button
4. You should be redirected to Google's authentication page
5. After signing in, you should be redirected back to the Badge Service dashboard

## Troubleshooting

### Redirect URI Mismatch

If you see an error about the redirect URI not matching, ensure that:
1. The `OIDC_REDIRECT_URL` environment variable exactly matches one of the authorized redirect URIs in the Google Cloud Console
2. The URI includes the correct protocol (http or https)
3. The port number is included if not using the default (80 for HTTP, 443 for HTTPS)

### Invalid Client ID or Secret

If you see an error about an invalid client ID or secret:
1. Double-check that you've copied the correct values from the Google Cloud Console
2. Ensure there are no extra spaces or characters in the environment variables

### Consent Screen Not Showing

If the consent screen doesn't appear:
1. Ensure you've configured the OAuth consent screen correctly
2. Check that you've added the required scopes
3. Verify that the test user is added to the consent screen configuration

## Next Steps

After setting up Google OAuth authentication, you may want to:

1. Customize the login page and dashboard to match your organization's branding
2. Configure domain restrictions to limit sign-in to specific email domains
3. Set up additional security measures such as rate limiting and IP restrictions