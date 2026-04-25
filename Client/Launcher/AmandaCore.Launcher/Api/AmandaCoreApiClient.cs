using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;
using AmandaCore.Launcher.Models;

namespace AmandaCore.Launcher.Api;

internal sealed class AmandaCoreApiClient
{
    private readonly HttpClient _httpClient = new();
    private readonly JsonSerializerOptions _jsonOptions = new() { PropertyNameCaseInsensitive = true };

    public async Task RegisterAsync(LauncherConfig config, string username, string password, CancellationToken cancellationToken)
    {
        await PostAsync<object>(
            $"{config.AuthServiceBaseUrl}/v1/accounts/register",
            new { username, password },
            null,
            cancellationToken);
    }

    public Task<AuthResponse> LoginAsync(LauncherConfig config, string username, string password, CancellationToken cancellationToken)
    {
        return PostAsync<AuthResponse>(
            $"{config.AuthServiceBaseUrl}/v1/auth/login",
            new { username, password },
            null,
            cancellationToken);
    }

    public Task<BuildManifest> GetBuildManifestAsync(LauncherConfig config, CancellationToken cancellationToken)
    {
        return GetAsync<BuildManifest>($"{config.RealmServiceBaseUrl}/v1/patch/manifest", null, cancellationToken);
    }

    public Task<RealmListResponse> GetRealmsAsync(LauncherConfig config, CancellationToken cancellationToken)
    {
        return GetAsync<RealmListResponse>($"{config.RealmServiceBaseUrl}/v1/realms", null, cancellationToken);
    }

    public Task<CharacterListResponse> GetCharactersAsync(LauncherConfig config, LauncherSession session, string realmId, CancellationToken cancellationToken)
    {
        return GetAsync<CharacterListResponse>(
            $"{config.CharacterServiceBaseUrl}/v1/characters?realmId={Uri.EscapeDataString(realmId)}",
            session.AccessToken,
            cancellationToken);
    }

    public Task<CharacterSummary> CreateCharacterAsync(
        LauncherConfig config,
        LauncherSession session,
        string realmId,
        string displayName,
        string raceId,
        string classId,
        string archetypeId,
        CancellationToken cancellationToken)
    {
        return PostAsync<CharacterSummary>(
            $"{config.CharacterServiceBaseUrl}/v1/characters",
            new { realmId, displayName, raceId, classId, archetypeId },
            session.AccessToken,
            cancellationToken);
    }

    public Task<WorldJoinTicket> CreateJoinTicketAsync(LauncherConfig config, LauncherSession session, string realmId, string characterId, CancellationToken cancellationToken)
    {
        return PostAsync<WorldJoinTicket>(
            $"{config.WorldServiceBaseUrl}/v1/world/join-ticket",
            new { realmId, characterId },
            session.AccessToken,
            cancellationToken);
    }

    private async Task<T> GetAsync<T>(string url, string? bearerToken, CancellationToken cancellationToken)
    {
        using var request = new HttpRequestMessage(HttpMethod.Get, url);
        if (!string.IsNullOrWhiteSpace(bearerToken))
        {
            request.Headers.Authorization = new AuthenticationHeaderValue("Bearer", bearerToken);
        }

        using var response = await _httpClient.SendAsync(request, cancellationToken);
        var content = await response.Content.ReadAsStringAsync(cancellationToken);
        EnsureSuccess(response, content);
        return JsonSerializer.Deserialize<T>(content, _jsonOptions)!;
    }

    private async Task<T> PostAsync<T>(string url, object payload, string? bearerToken, CancellationToken cancellationToken)
    {
        using var request = new HttpRequestMessage(HttpMethod.Post, url)
        {
            Content = new StringContent(JsonSerializer.Serialize(payload), Encoding.UTF8, "application/json")
        };

        if (!string.IsNullOrWhiteSpace(bearerToken))
        {
            request.Headers.Authorization = new AuthenticationHeaderValue("Bearer", bearerToken);
        }

        using var response = await _httpClient.SendAsync(request, cancellationToken);
        var content = await response.Content.ReadAsStringAsync(cancellationToken);
        EnsureSuccess(response, content);

        if (typeof(T) == typeof(object))
        {
            return (T)(object)new object();
        }

        return JsonSerializer.Deserialize<T>(content, _jsonOptions)!;
    }

    private static void EnsureSuccess(HttpResponseMessage response, string content)
    {
        if (response.IsSuccessStatusCode)
        {
            return;
        }

        throw new InvalidOperationException(string.IsNullOrWhiteSpace(content)
            ? $"Request failed with status code {(int)response.StatusCode}."
            : content);
    }
}
