using System.Text.Json;
using System.Text.Json.Serialization;

namespace AmandaCore.Launcher.Models;

internal sealed class LauncherConfig
{
    public string AuthServiceBaseUrl { get; set; } = "http://127.0.0.1:8081";
    public string RealmServiceBaseUrl { get; set; } = "http://127.0.0.1:8083";
    public string CharacterServiceBaseUrl { get; set; } = "http://127.0.0.1:8084";
    public string WorldServiceBaseUrl { get; set; } = "http://127.0.0.1:8085";
    public string ClientExecutablePath { get; set; } = string.Empty;
    [JsonIgnore]
    public ClientExecutableResolution Resolution { get; private set; } = ClientExecutableResolution.Empty;

    public static LauncherConfig Load()
    {
        var path = GetConfigPath();
        LauncherConfig config;
        if (!File.Exists(path))
        {
            config = new LauncherConfig();
        }
        else
        {
            var json = File.ReadAllText(path);
            config = JsonSerializer.Deserialize<LauncherConfig>(json) ?? new LauncherConfig();
        }

        config.ApplyClientExecutableResolution();
        return config;
    }

    public void Save()
    {
        var path = GetConfigPath();
        Directory.CreateDirectory(Path.GetDirectoryName(path)!);
        var json = JsonSerializer.Serialize(this, new JsonSerializerOptions { WriteIndented = true });
        File.WriteAllText(path, json);
    }

    private static string GetConfigPath()
    {
        var root = Path.Combine(
            Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData),
            "amandacore");
        return Path.Combine(root, "launcher-settings.json");
    }

    private void ApplyClientExecutableResolution()
    {
        Resolution = ResolveClientExecutable();
        ClientExecutablePath = Resolution.SelectedPath;
    }

    private static ClientExecutableResolution ResolveClientExecutable()
    {
        var resolution = new ClientExecutableResolution();
        var repoRoot = ResolveRepoRoot(out var source);
        resolution.RepoRoot = repoRoot ?? string.Empty;
        resolution.RepoRootSource = source;

        if (string.IsNullOrWhiteSpace(repoRoot))
        {
            return resolution;
        }

        var o3deWindowsGameLauncher = Path.Combine(
            repoRoot,
            "build",
            "o3de-windows",
            "bin",
            "profile",
            "amandacore.GameLauncher.exe");
        var windowsGameLauncher = Path.Combine(
            repoRoot,
            "build",
            "windows",
            "bin",
            "profile",
            "amandacore.GameLauncher.exe");
        var fallbackWorldClient = Path.Combine(
            repoRoot,
            "Client",
            "Game",
            "AmandaCore.WorldClient",
            "bin",
            "Debug",
            "net8.0",
            "AmandaCore.WorldClient.exe");

        resolution.Candidates.Add(ClientExecutableCandidate.Create("o3de-build-windows", windowsGameLauncher));
        resolution.Candidates.Add(ClientExecutableCandidate.Create("o3de-build-o3de-windows", o3deWindowsGameLauncher));
        resolution.Candidates.Add(ClientExecutableCandidate.Create("fallback-dotnet", fallbackWorldClient));

        var selected = resolution.Candidates
            .Where(candidate => candidate.Exists && candidate.Label.StartsWith("o3de-build-", StringComparison.OrdinalIgnoreCase))
            .OrderByDescending(candidate => candidate.EffectiveLastWriteTimeUtc)
            .FirstOrDefault()
            ?? resolution.Candidates.FirstOrDefault(candidate => candidate.Exists);
        if (selected is not null)
        {
            resolution.SelectedPath = selected.Path;
        }

        return resolution;
    }

    private static string? ResolveRepoRoot(out string source)
    {
        var envRepoRoot = Environment.GetEnvironmentVariable("AMANDACORE_REPO_ROOT");
        if (!string.IsNullOrWhiteSpace(envRepoRoot))
        {
            var fullEnvRepoRoot = Path.GetFullPath(envRepoRoot);
            if (IsRepoRoot(fullEnvRepoRoot))
            {
                source = "environment";
                return fullEnvRepoRoot;
            }
        }

        var current = new DirectoryInfo(AppContext.BaseDirectory);
        while (current is not null)
        {
            if (IsRepoRoot(current.FullName))
            {
                source = "ancestor";
                return current.FullName;
            }

            current = current.Parent;
        }

        source = "unresolved";
        return null;
    }

    private static bool IsRepoRoot(string path)
    {
        return File.Exists(Path.Combine(path, "project.json"));
    }
}

internal sealed class ClientExecutableResolution
{
    public static ClientExecutableResolution Empty { get; } = new();

    public string RepoRoot { get; set; } = string.Empty;
    public string RepoRootSource { get; set; } = "unresolved";
    public string SelectedPath { get; set; } = string.Empty;
    public List<ClientExecutableCandidate> Candidates { get; } = [];
}

internal sealed class ClientExecutableCandidate
{
    private static readonly string[] O3deFreshnessFiles =
    [
        "amandacore.GameLauncher.exe",
        "UiClient.dll",
        "GameCore.dll",
        "CombatRules.dll",
        "MovementPhysics.dll",
        "NpcAi.dll",
        "NetClient.dll"
    ];

    public string Label { get; init; } = string.Empty;
    public string Path { get; init; } = string.Empty;
    public bool Exists { get; init; }
    public DateTime LastWriteTimeUtc { get; init; }
    public DateTime EffectiveLastWriteTimeUtc { get; init; }

    public static ClientExecutableCandidate Create(string label, string path)
    {
        var exists = File.Exists(path);
        var lastWriteTimeUtc = exists ? File.GetLastWriteTimeUtc(path) : DateTime.MinValue;
        var effectiveLastWriteTimeUtc = label.StartsWith("o3de-build-", StringComparison.OrdinalIgnoreCase)
            ? ResolveO3deEffectiveLastWriteTimeUtc(path, lastWriteTimeUtc)
            : lastWriteTimeUtc;

        return new ClientExecutableCandidate
        {
            Label = label,
            Path = path,
            Exists = exists,
            LastWriteTimeUtc = lastWriteTimeUtc,
            EffectiveLastWriteTimeUtc = effectiveLastWriteTimeUtc
        };
    }

    private static DateTime ResolveO3deEffectiveLastWriteTimeUtc(string executablePath, DateTime fallback)
    {
        var directory = System.IO.Path.GetDirectoryName(executablePath);
        if (string.IsNullOrWhiteSpace(directory) || !Directory.Exists(directory))
        {
            return fallback;
        }

        var newest = fallback;
        foreach (var fileName in O3deFreshnessFiles)
        {
            var candidatePath = System.IO.Path.Combine(directory, fileName);
            if (!File.Exists(candidatePath))
            {
                continue;
            }

            var candidateWriteTime = File.GetLastWriteTimeUtc(candidatePath);
            if (candidateWriteTime > newest)
            {
                newest = candidateWriteTime;
            }
        }

        return newest;
    }
}
