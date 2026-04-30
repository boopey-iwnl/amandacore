using System.Text.Json;
using AmandaCore.Launcher.Models;

namespace AmandaCore.Launcher.Api;

internal sealed class AmandaCoreApiClient
{
    private readonly HttpClient _httpClient = new();
    private readonly JsonSerializerOptions _jsonOptions = new() { PropertyNameCaseInsensitive = true };

    public Task<BuildManifest> GetBuildManifestAsync(LauncherConfig config, CancellationToken cancellationToken)
    {
        return GetAsync<BuildManifest>($"{config.RealmServiceBaseUrl}/v1/patch/manifest", cancellationToken);
    }

    private async Task<T> GetAsync<T>(string url, CancellationToken cancellationToken)
    {
        using var request = new HttpRequestMessage(HttpMethod.Get, url);

        using var response = await _httpClient.SendAsync(request, cancellationToken);
        var content = await response.Content.ReadAsStringAsync(cancellationToken);
        EnsureSuccess(response, content);
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
