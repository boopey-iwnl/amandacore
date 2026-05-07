# amandacore Launcher

The launcher is a Windows-first C# WinForms application that can:

- fetch patch/build status from the realm service
- resolve the preferred O3DE `GameLauncher`
- launch the game client with safe endpoint and O3DE runtime arguments

The launcher is not the player login surface. Account login, remembered-session restore, realm selection, character selection/creation, join-ticket request, and world entry happen inside the game client.

Settings are stored under `%LocalAppData%\amandacore\launcher-settings.json`. The launcher does not persist player login sessions.
