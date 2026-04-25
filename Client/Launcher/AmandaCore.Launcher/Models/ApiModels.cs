namespace AmandaCore.Launcher.Models;

internal sealed class AuthResponse
{
    public string AccessToken { get; set; } = string.Empty;
    public string RefreshToken { get; set; } = string.Empty;
    public string AccountId { get; set; } = string.Empty;
}

internal sealed class RealmListResponse
{
    public List<RealmDescriptor> Realms { get; set; } = [];
}

internal sealed class RealmDescriptor
{
    public string Id { get; set; } = string.Empty;
    public string DisplayName { get; set; } = string.Empty;
    public string Region { get; set; } = string.Empty;
    public string Endpoint { get; set; } = string.Empty;
    public string SupportedBuild { get; set; } = string.Empty;
    public int OnlinePlayers { get; set; }
    public bool Online { get; set; }
}

internal sealed class CharacterListResponse
{
    public List<CharacterSummary> Characters { get; set; } = [];
}

internal sealed class CharacterSummary
{
    public string Id { get; set; } = string.Empty;
    public string RealmId { get; set; } = string.Empty;
    public string DisplayName { get; set; } = string.Empty;
    public string RaceId { get; set; } = string.Empty;
    public string ClassId { get; set; } = string.Empty;
    public string ArchetypeId { get; set; } = string.Empty;
    public int Level { get; set; }
    public string ZoneId { get; set; } = string.Empty;
}

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

internal sealed class WorldJoinTicket
{
    public string TicketId { get; set; } = string.Empty;
    public string SessionId { get; set; } = string.Empty;
    public string AccountId { get; set; } = string.Empty;
    public string CharacterId { get; set; } = string.Empty;
    public string RealmId { get; set; } = string.Empty;
    public string WorldEndpoint { get; set; } = string.Empty;
    public long ExpiresAt { get; set; }
}
