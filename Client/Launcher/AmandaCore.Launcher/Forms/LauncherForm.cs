using System.Diagnostics;
using System.Drawing;
using System.Text;
using AmandaCore.Launcher.Api;
using AmandaCore.Launcher.Models;

namespace AmandaCore.Launcher.Forms;

internal sealed class LauncherForm : Form
{
    private readonly AmandaCoreApiClient _apiClient = new();
    private readonly LauncherConfig _config = LauncherConfig.Load();

    private LocalVersionManifest? _localManifest;
    private BuildManifest? _serverManifest;
    private readonly Label _modeLabel = new()
    {
        AutoSize = true,
        MaximumSize = new Size(620, 0),
        Text = "Launcher mode: patcher and client bootstrapper. Account login happens in the game client."
    };
    private readonly Label _buildInfoLabel = new() { AutoSize = true, MaximumSize = new Size(620, 0), Text = "Build: local manifest not loaded" };
    private readonly Label _endpointInfoLabel = new() { AutoSize = true, MaximumSize = new Size(620, 0), Text = "Endpoints: unresolved" };
    private readonly Label _clientInfoLabel = new() { AutoSize = true, MaximumSize = new Size(620, 0), Text = "Client: unresolved" };
    private readonly Button _refreshButton = new() { Text = "Refresh Status", Width = 140 };
    private readonly Button _playButton = new() { Text = "Play", Width = 160, Height = 40 };
    private readonly TextBox _logTextBox = new() { Multiline = true, Width = 620, Height = 190, ScrollBars = ScrollBars.Vertical, ReadOnly = true };
    private readonly Label _statusLabel = new() { AutoSize = true, Text = "Status: Ready" };

    public LauncherForm()
    {
        _localManifest = LocalVersionManifest.LoadFromRepoRoot(_config.Resolution.RepoRoot);
        _buildInfoLabel.Text = BuildInfoText(null);
        _endpointInfoLabel.Text = EndpointInfoText();
        _clientInfoLabel.Text = ClientInfoText();

        Text = "amandacore Launcher";
        Width = 700;
        Height = 520;
        StartPosition = FormStartPosition.CenterScreen;

        var root = new FlowLayoutPanel
        {
            Dock = DockStyle.Fill,
            FlowDirection = FlowDirection.TopDown,
            WrapContents = false,
            AutoScroll = true,
            Padding = new Padding(14)
        };

        root.Controls.Add(_modeLabel);
        root.Controls.Add(_buildInfoLabel);
        root.Controls.Add(_endpointInfoLabel);
        root.Controls.Add(_clientInfoLabel);
        root.Controls.Add(BuildActionRow());
        root.Controls.Add(_statusLabel);
        root.Controls.Add(_logTextBox);

        Controls.Add(root);

        _refreshButton.Click += async (_, _) => await ExecuteAsync(RefreshServerBuildInfoAsync);
        _playButton.Click += async (_, _) => await ExecuteAsync(PlayAsync);
        Shown += async (_, _) => await ExecuteAsync(RefreshServerBuildInfoAsync);

        if (ClearLegacyLauncherSessionFile())
        {
            Log("Cleared legacy launcher session cache. The launcher no longer stores player login sessions.");
        }
        Log("Launcher will start the game client without username, password, tokens, selected character, or join ticket.");
    }

    private Control BuildActionRow()
    {
        var row = new FlowLayoutPanel { Width = 620, AutoSize = true };
        row.Controls.Add(_playButton);
        row.Controls.Add(_refreshButton);
        return row;
    }

    private async Task RefreshServerBuildInfoAsync(CancellationToken cancellationToken)
    {
        try
        {
            var manifest = await _apiClient.GetBuildManifestAsync(_config, cancellationToken);
            _serverManifest = manifest;
            _buildInfoLabel.Text = BuildInfoText(manifest);
            _endpointInfoLabel.Text = EndpointInfoText();
            _clientInfoLabel.Text = ClientInfoText();
            _statusLabel.Text = "Status: Patch/build status refreshed";
            LogServerBuildManifest(manifest);
            LogCompatibilityWarnings(manifest);
            LogClientExecutableResolution();
        }
        catch (Exception exception)
        {
            _serverManifest = null;
            _buildInfoLabel.Text = BuildInfoText(null);
            _statusLabel.Text = "Status: Server status unavailable";
            Log($"Server status unavailable: {exception.Message}");
        }
    }

    private async Task PlayAsync(CancellationToken cancellationToken)
    {
        var manifest = await _apiClient.GetBuildManifestAsync(_config, cancellationToken);
        _serverManifest = manifest;
        _buildInfoLabel.Text = BuildInfoText(manifest);
        LogServerBuildManifest(manifest);
        LogCompatibilityWarnings(manifest);

        if (!manifest.AllowedForLogin)
        {
            throw new InvalidOperationException($"Build {manifest.DisplayVersion} is not allowed for client login.");
        }

        LogClientExecutableResolution();
        if (string.IsNullOrWhiteSpace(_config.ClientExecutablePath) || !File.Exists(_config.ClientExecutablePath))
        {
            var checkedPaths = _config.Resolution.Candidates.Count == 0
                ? "<none>"
                : string.Join("; ", _config.Resolution.Candidates.Select(candidate => candidate.Path));
            throw new InvalidOperationException($"No game client executable was found. Checked paths: {checkedPaths}");
        }

        if (!IsO3deGameLauncherPath(_config.ClientExecutablePath))
        {
            throw new InvalidOperationException("The patcher Play flow requires the O3DE GameLauncher; the .NET world client remains diagnostic-only.");
        }

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
            AddO3deRuntimeArguments(startInfo);
        }
        AddPreWorldEndpointArguments(startInfo);

        Log($"Launch command: {_config.ClientExecutablePath} {FormatLaunchArgumentsForLog()}");
        var process = Process.Start(startInfo) ?? throw new InvalidOperationException("Process.Start returned null.");
        _statusLabel.Text = "Status: Game client launched";
        Log($"Client process start succeeded. Pid: {process.Id}");
        Log("Launched game client for in-client login.");
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
            _statusLabel.Text = "Status: Action failed";
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

        return IsO3deGameLauncherPath(_config.ClientExecutablePath);
    }

    private static bool IsO3deGameLauncherPath(string path)
    {
        return Path.GetFileName(path).Contains("GameLauncher", StringComparison.OrdinalIgnoreCase);
    }

    private void AddO3deRuntimeArguments(ProcessStartInfo startInfo)
    {
        var packageRoot = _config.Resolution.RepoRoot;
        startInfo.ArgumentList.Add("--project-path");
        startInfo.ArgumentList.Add(packageRoot);

        var cacheRoot = Path.Combine(packageRoot, "Cache");
        if (Directory.Exists(cacheRoot))
        {
            startInfo.ArgumentList.Add("--project-cache-path");
            startInfo.ArgumentList.Add(cacheRoot);
        }

        startInfo.ArgumentList.Add("--project-user-path");
        startInfo.ArgumentList.Add(Path.Combine(packageRoot, "user"));
        startInfo.ArgumentList.Add("--project-log-path");
        startInfo.ArgumentList.Add(Path.Combine(packageRoot, "user", "log"));
    }

    private void AddPreWorldEndpointArguments(ProcessStartInfo startInfo)
    {
        startInfo.ArgumentList.Add("--auth-endpoint");
        startInfo.ArgumentList.Add(_config.AuthServiceBaseUrl);
        startInfo.ArgumentList.Add("--realm-endpoint");
        startInfo.ArgumentList.Add(_config.RealmServiceBaseUrl);
        startInfo.ArgumentList.Add("--character-endpoint");
        startInfo.ArgumentList.Add(_config.CharacterServiceBaseUrl);
        startInfo.ArgumentList.Add("--world-service-endpoint");
        startInfo.ArgumentList.Add(_config.WorldServiceBaseUrl);
    }

    private void ToggleButtons(bool enabled)
    {
        _playButton.Enabled = enabled;
        _refreshButton.Enabled = enabled;
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

    private string EndpointInfoText()
    {
        return "Endpoints: " +
            $"auth {_config.AuthServiceBaseUrl} | " +
            $"realm {_config.RealmServiceBaseUrl} | " +
            $"characters {_config.CharacterServiceBaseUrl} | " +
            $"world {_config.WorldServiceBaseUrl}";
    }

    private string ClientInfoText()
    {
        return string.IsNullOrWhiteSpace(_config.ClientExecutablePath)
            ? "Client: no executable resolved"
            : $"Client: {Path.GetFileName(_config.ClientExecutablePath)}";
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
            Log($"Warning: local build {_localManifest.BuildId} differs from server build {manifest.Id}. Play is still allowed in this milestone.");
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
                Log($"Selected O3DE GameLauncher for in-client login: {selectedCandidate.Label}");
            }
            else
            {
                Log("Selected fallback .NET world client. Play requires O3DE GameLauncher; fallback is diagnostic-only.");
            }
        }

        if (!string.IsNullOrWhiteSpace(_config.ClientExecutablePath))
        {
            Log($"Resolved client executable path: {_config.ClientExecutablePath}");
        }
    }

    private string FormatLaunchArgumentsForLog()
    {
        var builder = new StringBuilder();
        if (!string.IsNullOrWhiteSpace(_config.Resolution.RepoRoot) &&
            IsO3deGameLauncherPath(_config.ClientExecutablePath))
        {
            builder.Append("--project-path ");
            builder.Append(_config.Resolution.RepoRoot);
            builder.Append(' ');
            var cacheRoot = Path.Combine(_config.Resolution.RepoRoot, "Cache");
            if (Directory.Exists(cacheRoot))
            {
                builder.Append("--project-cache-path ");
                builder.Append(cacheRoot);
                builder.Append(' ');
            }

            builder.Append("--project-user-path ");
            builder.Append(Path.Combine(_config.Resolution.RepoRoot, "user"));
            builder.Append(" --project-log-path ");
            builder.Append(Path.Combine(_config.Resolution.RepoRoot, "user", "log"));
            builder.Append(' ');
        }

        builder.Append("--auth-endpoint ");
        builder.Append(_config.AuthServiceBaseUrl);
        builder.Append(" --realm-endpoint ");
        builder.Append(_config.RealmServiceBaseUrl);
        builder.Append(" --character-endpoint ");
        builder.Append(_config.CharacterServiceBaseUrl);
        builder.Append(" --world-service-endpoint ");
        builder.Append(_config.WorldServiceBaseUrl);
        return builder.ToString();
    }

    private static bool ClearLegacyLauncherSessionFile()
    {
        var root = Path.Combine(
            Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData),
            "amandacore");
        var path = Path.Combine(root, "launcher-session.json");
        if (!File.Exists(path))
        {
            return false;
        }

        File.Delete(path);
        return true;
    }
}
