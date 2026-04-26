using System.Net.Http.Json;
using System.Text.Json;

var options = ClientOptions.Parse(args);
var client = new WorldClient(options.WorldEndpoint);

var session = await client.ConnectAsync(options.JoinTicketId);
PrintState("Connected", session);

if (options.AutoDemo)
{
    session = await client.MoveAsync(session.WorldSessionToken, 5, 0);
    PrintState("Moved east", session);

    session = await client.MoveAsync(session.WorldSessionToken, 0, 3);
    PrintState("Moved north", session);

    await client.DisconnectAsync(session.WorldSessionToken);
    Console.WriteLine("Disconnected.");

    session = await client.ReconnectAsync(session.WorldSessionToken);
    PrintState("Reconnected", session);
    return;
}

Console.WriteLine("Controls: W/A/S/D move, R reconnect, X disconnect, Q quit");
var current = session;
var disconnected = false;

while (true)
{
    var key = Console.ReadKey(intercept: true);
    switch (key.Key)
    {
        case ConsoleKey.W:
            current = await client.MoveAsync(current.WorldSessionToken, 0, 1);
            PrintState("Moved north", current);
            disconnected = false;
            break;
        case ConsoleKey.S:
            current = await client.MoveAsync(current.WorldSessionToken, 0, -1);
            PrintState("Moved south", current);
            disconnected = false;
            break;
        case ConsoleKey.A:
            current = await client.MoveAsync(current.WorldSessionToken, -1, 0);
            PrintState("Moved west", current);
            disconnected = false;
            break;
        case ConsoleKey.D:
            current = await client.MoveAsync(current.WorldSessionToken, 1, 0);
            PrintState("Moved east", current);
            disconnected = false;
            break;
        case ConsoleKey.R:
            current = await client.ReconnectAsync(current.WorldSessionToken);
            PrintState("Reconnected", current);
            disconnected = false;
            break;
        case ConsoleKey.X:
            await client.DisconnectAsync(current.WorldSessionToken);
            Console.WriteLine("Disconnected from world server.");
            disconnected = true;
            break;
        case ConsoleKey.Q:
            if (!disconnected)
            {
                await client.DisconnectAsync(current.WorldSessionToken);
            }
            return;
    }
}

static void PrintState(string label, WorldSessionResponse response)
{
    Console.WriteLine();
    Console.WriteLine($"[{label}] {response.DisplayName} in {response.ZoneId}");
    Console.WriteLine($"Position: ({response.Position.X}, {response.Position.Y}, {response.Position.Z})");
    Console.WriteLine("Visible entities:");
    foreach (var entity in response.Entities)
    {
        Console.WriteLine($"- {entity.Kind}: {entity.DisplayName} @ ({entity.X}, {entity.Y}, {entity.Z})");
    }
}

internal sealed record ClientOptions(string JoinTicketId, string WorldEndpoint, bool AutoDemo)
{
    public static ClientOptions Parse(string[] args)
    {
        string? joinTicket = null;
        var worldEndpoint = "http://127.0.0.1:8085";
        var autoDemo = false;

        for (var index = 0; index < args.Length; index++)
        {
            switch (args[index])
            {
                case "--join-ticket":
                    joinTicket = GetValue(args, ++index, "--join-ticket");
                    break;
                case "--world-endpoint":
                    worldEndpoint = GetValue(args, ++index, "--world-endpoint");
                    break;
                case "--auto-demo":
                    autoDemo = true;
                    break;
            }
        }

        if (string.IsNullOrWhiteSpace(joinTicket))
        {
            throw new InvalidOperationException("A --join-ticket value is required.");
        }

        return new ClientOptions(joinTicket, worldEndpoint, autoDemo);
    }

    private static string GetValue(string[] args, int index, string name)
    {
        if (index >= args.Length)
        {
            throw new InvalidOperationException($"Missing value for {name}.");
        }

        return args[index];
    }
}

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

internal sealed class WorldSessionResponse
{
    public string WorldSessionToken { get; set; } = string.Empty;
    public string CharacterId { get; set; } = string.Empty;
    public string RealmId { get; set; } = string.Empty;
    public string ZoneId { get; set; } = string.Empty;
    public string DisplayName { get; set; } = string.Empty;
    public Position Position { get; set; } = new();
    public List<VisibleEntity> Entities { get; set; } = [];
}

internal sealed class Position
{
    public double X { get; set; }
    public double Y { get; set; }
    public double Z { get; set; }
}

internal sealed class VisibleEntity
{
    public string Id { get; set; } = string.Empty;
    public string DisplayName { get; set; } = string.Empty;
    public string Kind { get; set; } = string.Empty;
    public double X { get; set; }
    public double Y { get; set; }
    public double Z { get; set; }
}
