using System.Text.Json;
using System.Text.Json.Serialization;

namespace AmandaCore.Launcher.Models;

internal sealed class LocalVersionManifest
{
    public string BuildId { get; set; } = string.Empty;
    public string Channel { get; set; } = string.Empty;
    public string DisplayVersion { get; set; } = string.Empty;
    public string ClientVersion { get; set; } = string.Empty;
    public string LauncherVersion { get; set; } = string.Empty;
    public string ServerVersion { get; set; } = string.Empty;
    public string ContentVersion { get; set; } = string.Empty;
    public string ProtocolVersion { get; set; } = string.Empty;
    public string ApiVersion { get; set; } = string.Empty;
    public string GeneratedAtUtc { get; set; } = string.Empty;
    public string GitBranch { get; set; } = string.Empty;
    public string GitCommit { get; set; } = string.Empty;
    public bool GitDirty { get; set; }
    public List<string> CompatibleClientVersions { get; set; } = [];
    public List<string> CompatibleServerVersions { get; set; } = [];
    public List<string> CompatibleProtocolVersions { get; set; } = [];

    [JsonIgnore]
    public string SourcePath { get; private set; } = string.Empty;

    public static LocalVersionManifest? LoadFromRepoRoot(string repoRoot)
    {
        if (string.IsNullOrWhiteSpace(repoRoot))
        {
            return null;
        }

        var path = Path.Combine(repoRoot, "Infra", "dev", "version-manifest.json");
        if (!File.Exists(path))
        {
            return null;
        }

        var json = File.ReadAllText(path);
        var manifest = JsonSerializer.Deserialize<LocalVersionManifest>(
            json,
            new JsonSerializerOptions { PropertyNameCaseInsensitive = true });
        if (manifest is not null)
        {
            manifest.SourcePath = path;
        }

        return manifest;
    }
}
