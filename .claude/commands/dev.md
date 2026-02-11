Start the First Sip development server.

1. Source environment variables from `local.env` in the project root
2. Run `make dev` (which runs `templ generate && go run cmd/server/main.go`)
3. Run the server in the background so the user can continue working
4. Verify the server is healthy by curling http://localhost:8080/health
5. Report the server URL to the user: http://localhost:8080

If `local.env` doesn't have real Google OAuth credentials (still has placeholder values), warn the user that OAuth login won't work until they configure real credentials.

If port 8080 is already in use, let the user know and offer to kill the existing process.
