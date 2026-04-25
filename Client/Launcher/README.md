# amandacore Launcher

The launcher is a Windows-first C# WinForms application that can:

- register a new player account
- log in against the auth service
- fetch the realm directory and patch manifest
- list and create characters
- request a world join ticket
- optionally launch a configured game client executable with that ticket

Settings are stored under `%LocalAppData%\amandacore\launcher-settings.json`.
