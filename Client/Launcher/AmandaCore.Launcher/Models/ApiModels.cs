namespace AmandaCore.Launcher.Models;

internal sealed class BuildManifest
{
    public string Id { get; set; } = string.Empty;
    public string Channel { get; set; } = string.Empty;
    public string DisplayVersion { get; set; } = string.Empty;
    public string ClientVersion { get; set; } = string.Empty;
    public string ServerVersion { get; set; } = string.Empty;
    public string ContentVersion { get; set; } = string.Empty;
    public string ProtocolVersion { get; set; } = string.Empty;
    public string ApiVersion { get; set; } = string.Empty;
    public List<string> CompatibleClientVersions { get; set; } = [];
    public List<string> CompatibleServerVersions { get; set; } = [];
    public List<string> CompatibleProtocolVersions { get; set; } = [];
    public List<string> RequiredServices { get; set; } = [];
    public bool AllowedForLogin { get; set; }
    public string LauncherNews { get; set; } = string.Empty;
    public string WorldEndpointHint { get; set; } = string.Empty;
    public string GeneratedAtUtc { get; set; } = string.Empty;
}
