using System.Net.Http.Json;
using System.Text.Json;

internal sealed class WorldClient
{
    private readonly HttpClient _httpClient = new();
    private readonly JsonSerializerOptions _jsonOptions = new() { PropertyNameCaseInsensitive = true };
    private readonly string _worldEndpoint;

    public WorldClient(string worldEndpoint)
    {
        _worldEndpoint = worldEndpoint.TrimEnd('/');
    }

    public Task<WorldSessionResponse> ConnectAsync(string ticketId)
    {
        return PostAsync<WorldSessionResponse>("/v1/world/connect", new { ticketId });
    }

    public Task<WorldSessionResponse> ReconnectAsync(string worldSessionToken)
    {
        return PostAsync<WorldSessionResponse>("/v1/world/reconnect", new { worldSessionToken });
    }

    public Task<WorldSessionResponse> GetStateAsync(string worldSessionToken)
    {
        return GetAsync<WorldSessionResponse>($"/v1/world/state?worldSessionToken={Uri.EscapeDataString(worldSessionToken)}");
    }

    public Task<WorldSessionResponse> MoveAsync(string worldSessionToken, double deltaX, double deltaY)
    {
        return PostAsync<WorldSessionResponse>("/v1/world/move", new { worldSessionToken, deltaX, deltaY });
    }

    public Task<WorldSessionResponse> TargetAsync(string worldSessionToken, string targetId)
    {
        return PostAsync<WorldSessionResponse>("/v1/world/target", new { worldSessionToken, targetId });
    }

    public Task<WorldSessionResponse> UseAbilityAsync(string worldSessionToken, string abilityId)
    {
        return PostAsync<WorldSessionResponse>("/v1/world/attack/ability", new { worldSessionToken, abilityId });
    }

    public async Task DisconnectAsync(string worldSessionToken)
    {
        await PostAsync<JsonElement>("/v1/world/disconnect", new { worldSessionToken });
    }

    private async Task<T> GetAsync<T>(string path)
    {
        using var response = await _httpClient.GetAsync(_worldEndpoint + path);
        return await ReadResponseAsync<T>(response);
    }

    private async Task<T> PostAsync<T>(string path, object payload)
    {
        using var response = await _httpClient.PostAsJsonAsync(_worldEndpoint + path, payload);
        return await ReadResponseAsync<T>(response);
    }

    private async Task<T> ReadResponseAsync<T>(HttpResponseMessage response)
    {
        var content = await response.Content.ReadAsStringAsync();
        if (!response.IsSuccessStatusCode)
        {
            throw new InvalidOperationException(content);
        }

        return JsonSerializer.Deserialize<T>(content, _jsonOptions)!;
    }
}
