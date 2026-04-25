using System.Diagnostics;
using System.Drawing;
using System.Text;
using AmandaCore.Launcher.Api;
using AmandaCore.Launcher.Models;

namespace AmandaCore.Launcher.Forms;

internal sealed class LauncherForm : Form
{
    private const string DefaultRaceId = "human";
    private const string DefaultClassId = "warrior";
    private const string LegacyArchetypeId = "wayfarer_warden";

    private readonly AmandaCoreApiClient _apiClient = new();
    private readonly LauncherConfig _config = LauncherConfig.Load();

    private LauncherSession? _session = LauncherSession.Load();
    private LocalVersionManifest? _localManifest;
    private BuildManifest? _serverManifest;
    private readonly TextBox _usernameTextBox = new() { Width = 180 };
    private readonly TextBox _passwordTextBox = new() { Width = 180, UseSystemPasswordChar = true };
    private readonly Button _registerButton = new() { Text = "Register", Width = 100 };
    private readonly Button _loginButton = new() { Text = "Login", Width = 100 };
    private readonly Button _logoutButton = new() { Text = "Logout", Width = 100 };
    private readonly Button _loadRealmsButton = new() { Text = "Load Realms", Width = 120 };
    private readonly ComboBox _realmComboBox = new() { Width = 280, DropDownStyle = ComboBoxStyle.DropDownList };
    private readonly Button _loadCharactersButton = new() { Text = "Load Characters", Width = 120 };
    private readonly ListBox _characterListBox = new() { Width = 280, Height = 120 };
    private readonly TextBox _characterNameTextBox = new() { Width = 180 };
    private readonly Button _createCharacterButton = new() { Text = "Create Character", Width = 140 };
    private readonly Button _joinWorldButton = new() { Text = "Join World", Width = 120 };
    private readonly TextBox _logTextBox = new() { Multiline = true, Width = 560, Height = 160, ScrollBars = ScrollBars.Vertical, ReadOnly = true };
    private readonly Label _statusLabel = new() { AutoSize = true, Text = "Status: Ready" };
    private readonly Label _buildInfoLabel = new() { AutoSize = true, MaximumSize = new Size(560, 0), Text = "Build: local manifest not loaded" };

    public LauncherForm()
    {
        _localManifest = LocalVersionManifest.LoadFromRepoRoot(_config.Resolution.RepoRoot);
        _buildInfoLabel.Text = BuildInfoText(null);

        Text = "amandacore Launcher";
        Width = 620;
        Height = 620;
        StartPosition = FormStartPosition.CenterScreen;

        var root = new FlowLayoutPanel
        {
            Dock = DockStyle.Fill,
            FlowDirection = FlowDirection.TopDown,
            WrapContents = false,
            AutoScroll = true,
            Padding = new Padding(12)
        };

        root.Controls.Add(_buildInfoLabel);
        root.Controls.Add(BuildCredentialsRow());
        root.Controls.Add(BuildRealmRow());
        root.Controls.Add(BuildCharacterRow());
        root.Controls.Add(_joinWorldButton);
        root.Controls.Add(_statusLabel);
        root.Controls.Add(_logTextBox);

        Controls.Add(root);

        _registerButton.Click += async (_, _) => await ExecuteAsync(RegisterAsync);
        _loginButton.Click += async (_, _) => await ExecuteAsync(LoginAsync);
        _logoutButton.Click += (_, _) => Logout();
        _loadRealmsButton.Click += async (_, _) => await ExecuteAsync(LoadRealmsAsync);
        _loadCharactersButton.Click += async (_, _) => await ExecuteAsync(LoadCharactersAsync);
        _createCharacterButton.Click += async (_, _) => await ExecuteAsync(CreateCharacterAsync);
        _joinWorldButton.Click += async (_, _) => await ExecuteAsync(JoinWorldAsync);
        Shown += async (_, _) => await ExecuteAsync(RefreshServerBuildInfoAsync);

        if (_session != null)
        {
            Log($"Restored launcher session for account {_session.AccountId}.");
        }
    }

    private Control BuildCredentialsRow()
    {
        var row = new FlowLayoutPanel { Width = 560, AutoSize = true };
        row.Controls.Add(new Label { Text = "Username", AutoSize = true, Margin = new Padding(0, 8, 8, 0) });
        row.Controls.Add(_usernameTextBox);
        row.Controls.Add(new Label { Text = "Password", AutoSize = true, Margin = new Padding(12, 8, 8, 0) });
        row.Controls.Add(_passwordTextBox);
        row.Controls.Add(_registerButton);
        row.Controls.Add(_loginButton);
        row.Controls.Add(_logoutButton);
        return row;
    }

    private Control BuildRealmRow()
    {
        var row = new FlowLayoutPanel { Width = 560, AutoSize = true };
        row.Controls.Add(_loadRealmsButton);
        row.Controls.Add(_realmComboBox);
        row.Controls.Add(_loadCharactersButton);
        return row;
    }

    private Control BuildCharacterRow()
    {
        var panel = new FlowLayoutPanel { Width = 560, AutoSize = true };
        panel.Controls.Add(_characterListBox);

        var actions = new FlowLayoutPanel { FlowDirection = FlowDirection.TopDown, AutoSize = true };
        actions.Controls.Add(new Label { Text = "New Character Name", AutoSize = true });
        actions.Controls.Add(_characterNameTextBox);
        actions.Controls.Add(new Label { Text = "Race", AutoSize = true });
        actions.Controls.Add(new Label { Text = "Human", AutoSize = true });
        actions.Controls.Add(new Label { Text = "Class", AutoSize = true });
        actions.Controls.Add(new Label { Text = "Warrior", AutoSize = true });
        actions.Controls.Add(_createCharacterButton);
        panel.Controls.Add(actions);
        return panel;
    }

    private async Task RegisterAsync(CancellationToken cancellationToken)
    {
        await _apiClient.RegisterAsync(_config, _usernameTextBox.Text.Trim(), _passwordTextBox.Text, cancellationToken);
        Log("Registration completed.");
    }

    private async Task LoginAsync(CancellationToken cancellationToken)
    {
        var manifest = await _apiClient.GetBuildManifestAsync(_config, cancellationToken);
        _serverManifest = manifest;
        _buildInfoLabel.Text = BuildInfoText(manifest);
        LogServerBuildManifest(manifest);
        LogCompatibilityWarnings(manifest);
        if (!manifest.AllowedForLogin)
        {
            throw new InvalidOperationException($"Build {manifest.DisplayVersion} is not allowed for login.");
        }

        var auth = await _apiClient.LoginAsync(_config, _usernameTextBox.Text.Trim(), _passwordTextBox.Text, cancellationToken);
        _session = new LauncherSession
        {
            AccessToken = auth.AccessToken,
            RefreshToken = auth.RefreshToken,
            AccountId = auth.AccountId
        };
        _session.Save();

        _statusLabel.Text = $"Status: Logged in as {_usernameTextBox.Text.Trim()}";
        Log($"Login succeeded for account {auth.AccountId}. Build: {manifest.DisplayVersion}");
    }

    private async Task RefreshServerBuildInfoAsync(CancellationToken cancellationToken)
    {
        try
        {
            var manifest = await _apiClient.GetBuildManifestAsync(_config, cancellationToken);
            _serverManifest = manifest;
            _buildInfoLabel.Text = BuildInfoText(manifest);
            LogServerBuildManifest(manifest);
            LogCompatibilityWarnings(manifest);
        }
        catch (Exception exception)
        {
            _serverManifest = null;
            _buildInfoLabel.Text = BuildInfoText(null);
            Log($"Server status unavailable: {exception.Message}");
        }
    }

    private void Logout()
    {
        _session = null;
        LauncherSession.Clear();
        _realmComboBox.Items.Clear();
        _characterListBox.Items.Clear();
        _statusLabel.Text = "Status: Logged out";
        Log("Launcher session cleared.");
    }

    private async Task LoadRealmsAsync(CancellationToken cancellationToken)
    {
        var realms = await _apiClient.GetRealmsAsync(_config, cancellationToken);
        _realmComboBox.Items.Clear();
        foreach (var realm in realms.Realms)
        {
            _realmComboBox.Items.Add(realm);
            Log($"Realm {realm.DisplayName}: online={realm.Online}, supportedBuild={ValueOrUnknown(realm.SupportedBuild)}, endpoint={ValueOrUnknown(realm.Endpoint)}");
        }

        _realmComboBox.DisplayMember = nameof(RealmDescriptor.DisplayName);
        if (_realmComboBox.Items.Count > 0)
        {
            _realmComboBox.SelectedIndex = 0;
        }

        Log($"Loaded {realms.Realms.Count} realm entries.");
    }

    private async Task LoadCharactersAsync(CancellationToken cancellationToken)
    {
        var session = RequireSession();
        var realm = RequireRealm();
        var characters = await _apiClient.GetCharactersAsync(_config, session, realm.Id, cancellationToken);
        _characterListBox.Items.Clear();
        foreach (var character in characters.Characters)
        {
            _characterListBox.Items.Add(character);
        }

        _characterListBox.DisplayMember = nameof(CharacterSummary.DisplayName);
        Log($"Loaded {characters.Characters.Count} characters for realm {realm.DisplayName}.");
    }

    private async Task CreateCharacterAsync(CancellationToken cancellationToken)
    {
        var session = RequireSession();
        var realm = RequireRealm();
        var character = await _apiClient.CreateCharacterAsync(
            _config,
            session,
            realm.Id,
            _characterNameTextBox.Text.Trim(),
            DefaultRaceId,
            DefaultClassId,
            LegacyArchetypeId,
            cancellationToken);
        Log($"Created character {character.DisplayName} ({character.RaceId}/{character.ClassId}) in realm {realm.DisplayName}.");
        await LoadCharactersAsync(cancellationToken);
    }

    private async Task JoinWorldAsync(CancellationToken cancellationToken)
    {
        var session = RequireSession();
        var realm = RequireRealm();
        var character = _characterListBox.SelectedItem as CharacterSummary
            ?? throw new InvalidOperationException("Select a character first.");

        var ticket = await _apiClient.CreateJoinTicketAsync(_config, session, realm.Id, character.Id, cancellationToken);
        Log($"Issued world join ticket {ticket.TicketId} for {character.DisplayName}.");
        LogClientExecutableResolution();

        if (!string.IsNullOrWhiteSpace(_config.ClientExecutablePath) && File.Exists(_config.ClientExecutablePath))
        {
            Log($"Launch command: {_config.ClientExecutablePath} {FormatLaunchArgumentsForLog(ticket.TicketId, ticket.WorldEndpoint)}");
            try
            {
                var startInfo = new ProcessStartInfo
                {
                    FileName = _config.ClientExecutablePath,
                    UseShellExecute = false
                };

                var workingDirectory = ResolveClientWorkingDirectory();
                if (!string.IsNullOrWhiteSpace(workingDirectory))
                {
                    startInfo.WorkingDirectory = workingDirectory;
                }

                if (ShouldPassProjectPath())
                {
                    startInfo.ArgumentList.Add("--project-path");
                    startInfo.ArgumentList.Add(_config.Resolution.RepoRoot);
                }

                startInfo.ArgumentList.Add("--join-ticket");
                startInfo.ArgumentList.Add(ticket.TicketId);
                startInfo.ArgumentList.Add("--world-endpoint");
                startInfo.ArgumentList.Add(ticket.WorldEndpoint);

                var process = Process.Start(startInfo);
                if (process == null)
                {
                    throw new InvalidOperationException("Process.Start returned null.");
                }

                Log($"Client process start succeeded. Pid: {process.Id}");
                Log("Launched configured game client executable.");
                return;
            }
            catch (Exception exception)
            {
                Log($"Client process start failed: {exception.Message}");
                throw;
            }
        }

        var checkedPaths = _config.Resolution.Candidates.Count == 0
            ? "<none>"
            : string.Join("; ", _config.Resolution.Candidates.Select(candidate => candidate.Path));
        Log($"No client executable configured. Checked paths: {checkedPaths}");
        Log("Ticket is ready for manual game bootstrap.");
    }

    private RealmDescriptor RequireRealm()
    {
        return _realmComboBox.SelectedItem as RealmDescriptor
            ?? throw new InvalidOperationException("Select a realm first.");
    }

    private LauncherSession RequireSession()
    {
        return _session ?? throw new InvalidOperationException("Log in first.");
    }

    private async Task ExecuteAsync(Func<CancellationToken, Task> action)
    {
        using var cancellationTokenSource = new CancellationTokenSource(TimeSpan.FromSeconds(20));
        try
        {
            ToggleButtons(false);
            await action(cancellationTokenSource.Token);
        }
        catch (Exception exception)
        {
            Log(exception.Message);
        }
        finally
        {
            ToggleButtons(true);
        }
    }

    private string ResolveClientWorkingDirectory()
    {
        if (!string.IsNullOrWhiteSpace(_config.Resolution.RepoRoot) && Directory.Exists(_config.Resolution.RepoRoot))
        {
            return _config.Resolution.RepoRoot;
        }

        var executableDirectory = Path.GetDirectoryName(_config.ClientExecutablePath);
        return !string.IsNullOrWhiteSpace(executableDirectory) && Directory.Exists(executableDirectory)
            ? executableDirectory
            : string.Empty;
    }

    private bool ShouldPassProjectPath()
    {
        if (string.IsNullOrWhiteSpace(_config.Resolution.RepoRoot) || !Directory.Exists(_config.Resolution.RepoRoot))
        {
            return false;
        }

        var fileName = Path.GetFileName(_config.ClientExecutablePath);
        return fileName.Contains("GameLauncher", StringComparison.OrdinalIgnoreCase);
    }

    private void ToggleButtons(bool enabled)
    {
        _registerButton.Enabled = enabled;
        _loginButton.Enabled = enabled;
        _logoutButton.Enabled = enabled;
        _loadRealmsButton.Enabled = enabled;
        _loadCharactersButton.Enabled = enabled;
        _createCharacterButton.Enabled = enabled;
        _joinWorldButton.Enabled = enabled;
    }

    private void Log(string message)
    {
        _logTextBox.AppendText($"[{DateTime.Now:T}] {message}{Environment.NewLine}");
    }

    private string BuildInfoText(BuildManifest? serverManifest)
    {
        var localBuild = _localManifest?.BuildId ?? "missing";
        var localClient = _localManifest?.ClientVersion ?? "unknown";
        var localProtocol = _localManifest?.ProtocolVersion ?? "unknown";
        var serverBuild = serverManifest?.Id ?? "unavailable";
        var serverProtocol = serverManifest?.ProtocolVersion ?? "unknown";

        return $"Build: local {ShortBuild(localBuild)} | server {ShortBuild(serverBuild)} | client {localClient} | protocol {localProtocol}/{serverProtocol}";
    }

    private void LogServerBuildManifest(BuildManifest manifest)
    {
        Log(
            "Server manifest: " +
            $"build={ValueOrUnknown(manifest.Id)}, " +
            $"display={ValueOrUnknown(manifest.DisplayVersion)}, " +
            $"channel={ValueOrUnknown(manifest.Channel)}, " +
            $"client={ValueOrUnknown(manifest.ClientVersion)}, " +
            $"server={ValueOrUnknown(manifest.ServerVersion)}, " +
            $"content={ValueOrUnknown(manifest.ContentVersion)}, " +
            $"protocol={ValueOrUnknown(manifest.ProtocolVersion)}, " +
            $"api={ValueOrUnknown(manifest.ApiVersion)}");
    }

    private void LogCompatibilityWarnings(BuildManifest manifest)
    {
        if (_localManifest is null)
        {
            Log("Warning: local version manifest was not found. Compatibility is warning-only for this milestone.");
            return;
        }

        if (!string.IsNullOrWhiteSpace(manifest.Id) &&
            !string.Equals(_localManifest.BuildId, manifest.Id, StringComparison.OrdinalIgnoreCase))
        {
            Log($"Warning: local build {_localManifest.BuildId} differs from server build {manifest.Id}. Join is still allowed in this milestone.");
        }

        if (!AllowedByManifest(manifest.CompatibleClientVersions, _localManifest.ClientVersion, manifest.ClientVersion))
        {
            Log($"Warning: local client version {_localManifest.ClientVersion} is not listed as compatible with server manifest client version {ValueOrUnknown(manifest.ClientVersion)}.");
        }

        if (!AllowedByManifest(manifest.CompatibleProtocolVersions, _localManifest.ProtocolVersion, manifest.ProtocolVersion))
        {
            Log($"Warning: local protocol version {_localManifest.ProtocolVersion} is not listed as compatible with server protocol version {ValueOrUnknown(manifest.ProtocolVersion)}.");
        }
    }

    private static bool AllowedByManifest(IReadOnlyCollection<string> allowedValues, string localValue, string exactValue)
    {
        if (string.IsNullOrWhiteSpace(localValue))
        {
            return true;
        }

        if (allowedValues.Count > 0)
        {
            return allowedValues.Any(value => string.Equals(value, localValue, StringComparison.OrdinalIgnoreCase));
        }

        return string.IsNullOrWhiteSpace(exactValue) ||
            string.Equals(localValue, exactValue, StringComparison.OrdinalIgnoreCase);
    }

    private static string ShortBuild(string value)
    {
        if (string.IsNullOrWhiteSpace(value))
        {
            return "unknown";
        }

        return value.Length <= 36 ? value : $"{value[..28]}...{value[^5..]}";
    }

    private static string ValueOrUnknown(string value)
    {
        return string.IsNullOrWhiteSpace(value) ? "unknown" : value;
    }

    private void LogClientExecutableResolution()
    {
        var repoRoot = string.IsNullOrWhiteSpace(_config.Resolution.RepoRoot)
            ? "<unresolved>"
            : _config.Resolution.RepoRoot;
        Log($"Resolved repo root: {repoRoot} source={_config.Resolution.RepoRootSource}");

        foreach (var candidate in _config.Resolution.Candidates)
        {
            var effectiveWriteTime = candidate.EffectiveLastWriteTimeUtc == DateTime.MinValue
                ? "<missing>"
                : candidate.EffectiveLastWriteTimeUtc.ToLocalTime().ToString("s");
            Log($"Candidate client executable [{candidate.Label}]: {candidate.Path} exists={candidate.Exists} freshness={effectiveWriteTime}");
        }

        var selectedCandidate = _config.Resolution.Candidates.FirstOrDefault(
            candidate => string.Equals(candidate.Path, _config.ClientExecutablePath, StringComparison.OrdinalIgnoreCase));
        if (selectedCandidate is not null)
        {
            if (selectedCandidate.Label.StartsWith("o3de-build-", StringComparison.OrdinalIgnoreCase))
            {
                Log($"Selected latest O3DE GameLauncher for the playable zone slice: {selectedCandidate.Label}");
            }
            else
            {
                Log("Selected fallback .NET world client. The O3DE GameLauncher remains the supported playable slice path; fallback launch is diagnostic only.");
            }
        }

        if (!string.IsNullOrWhiteSpace(_config.ClientExecutablePath))
        {
            Log($"Resolved client executable path: {_config.ClientExecutablePath}");
        }
    }

    private string FormatLaunchArgumentsForLog(string ticketId, string worldEndpoint)
    {
        var maskedTicket = ticketId.Length <= 8
            ? ticketId
            : $"{ticketId[..4]}...{ticketId[^4..]}";

        var builder = new StringBuilder();
        if (!string.IsNullOrWhiteSpace(_config.Resolution.RepoRoot) &&
            Path.GetFileName(_config.ClientExecutablePath).Contains("GameLauncher", StringComparison.OrdinalIgnoreCase))
        {
            builder.Append("--project-path ");
            builder.Append(_config.Resolution.RepoRoot);
            builder.Append(' ');
        }

        builder.Append("--join-ticket ");
        builder.Append(maskedTicket);
        builder.Append(" --world-endpoint ");
        builder.Append(worldEndpoint);
        return builder.ToString();
    }
}
