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

    public Task<WorldSessionResponse> MoveAsync(string worldSessionToken, int deltaX, int deltaY)
    {
        return PostAsync<WorldSessionResponse>("/v1/world/move", new { worldSessionToken, deltaX, deltaY });
    }

    public async Task DisconnectAsync(string worldSessionToken)
    {
        await PostAsync<JsonElement>("/v1/world/disconnect", new { worldSessionToken });
    }

    private async Task<T> PostAsync<T>(string path, object payload)
    {
        using var response = await _httpClient.PostAsJsonAsync(_worldEndpoint + path, payload);
        var content = await response.Content.ReadAsStringAsync();
        if (!response.IsSuccessStatusCode)
        {
            throw new InvalidOperationException(content);
        }

        return JsonSerializer.Deserialize<T>(content, _jsonOptions)!;
    }
}
