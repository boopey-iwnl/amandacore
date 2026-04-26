using System.Net.Http.Json;
using System.Text.Json;

var options = ClientOptions.Parse(args);
var client = new WorldClient(options.WorldEndpoint);
var streamingPreview = new ClientStreamingPreviewState();

var session = await client.ConnectAsync(options.JoinTicketId);
PrintState("Connected", session, streamingPreview);

if (options.AutoDemo)
{
    session = await client.MoveAsync(session.WorldSessionToken, 5, 0);
    PrintState("Moved east", session, streamingPreview);

    session = await client.MoveAsync(session.WorldSessionToken, 0, 3);
    PrintState("Moved north", session, streamingPreview);

    await client.DisconnectAsync(session.WorldSessionToken);
    Console.WriteLine("Disconnected.");

    session = await client.ReconnectAsync(session.WorldSessionToken);
    PrintState("Reconnected", session, streamingPreview);
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
            PrintState("Moved north", current, streamingPreview);
            disconnected = false;
            break;
        case ConsoleKey.S:
            current = await client.MoveAsync(current.WorldSessionToken, 0, -1);
            PrintState("Moved south", current, streamingPreview);
            disconnected = false;
            break;
        case ConsoleKey.A:
            current = await client.MoveAsync(current.WorldSessionToken, -1, 0);
            PrintState("Moved west", current, streamingPreview);
            disconnected = false;
            break;
        case ConsoleKey.D:
            current = await client.MoveAsync(current.WorldSessionToken, 1, 0);
            PrintState("Moved east", current, streamingPreview);
            disconnected = false;
            break;
        case ConsoleKey.R:
            current = await client.ReconnectAsync(current.WorldSessionToken);
            PrintState("Reconnected", current, streamingPreview);
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

static void PrintState(string label, WorldSessionResponse response, ClientStreamingPreviewState streamingPreview)
{
    Console.WriteLine();
    Console.WriteLine($"[{label}] {response.DisplayName} in {response.ZoneId}");
    Console.WriteLine($"Position: ({response.Position.X}, {response.Position.Y}, {response.Position.Z})");
    streamingPreview.Update(response);
    if (streamingPreview.Active)
    {
        Console.WriteLine($"Streaming: map {streamingPreview.MapId}, cells {streamingPreview.StreamingCellCount}, adjacent [{string.Join(", ", streamingPreview.AdjacentZoneIds)}]");
        var nearest = streamingPreview.NearestTransition(response.Position);
        if (nearest is not null)
        {
            var state = nearest.Ready ? "ready" : $"distance {nearest.Distance:0.0}";
            Console.WriteLine($"Nearest transition: {nearest.DisplayName} -> {nearest.TargetZoneId} ({state})");
        }
    }
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
        var worldEndpoint = "http://localhost:8085";
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
    public StreamingStateResponse Streaming { get; set; } = new();
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

internal sealed class StreamingStateResponse
{
    public bool Enabled { get; set; }
    public string ZoneId { get; set; } = string.Empty;
    public string MapId { get; set; } = string.Empty;
    public List<string> AdjacentZoneIds { get; set; } = [];
    public MapBounds Bounds { get; set; } = new();
    public List<TransitionHintResponse> TransitionHints { get; set; } = [];
    public List<StreamingCellResponse> StreamingCells { get; set; } = [];
}

internal sealed class MapBounds
{
    public double MinX { get; set; }
    public double MinY { get; set; }
    public double MinZ { get; set; }
    public double MaxX { get; set; }
    public double MaxY { get; set; }
    public double MaxZ { get; set; }
}

internal sealed class TransitionHintResponse
{
    public string TransitionId { get; set; } = string.Empty;
    public string DisplayName { get; set; } = string.Empty;
    public string TargetZoneId { get; set; } = string.Empty;
    public string DestinationEntryId { get; set; } = string.Empty;
    public string StreamingCellId { get; set; } = string.Empty;
    public string Hint { get; set; } = string.Empty;
    public Position Position { get; set; } = new();
    public double Radius { get; set; }
}

internal sealed class StreamingCellResponse
{
    public string CellId { get; set; } = string.Empty;
    public string DisplayName { get; set; } = string.Empty;
    public int Priority { get; set; }
    public List<string> Tags { get; set; } = [];
    public MapBounds Bounds { get; set; } = new();
}

internal sealed class ClientStreamingPreviewState
{
    private StreamingStateResponse _current = new();

    public bool Active => _current.Enabled;
    public string MapId => _current.MapId;
    public IReadOnlyList<string> AdjacentZoneIds => _current.AdjacentZoneIds;
    public int StreamingCellCount => _current.StreamingCells.Count;

    public void Update(WorldSessionResponse response)
    {
        _current = response.Streaming ?? new StreamingStateResponse();
    }

    public TransitionPreview? NearestTransition(Position currentPosition)
    {
        if (!_current.Enabled || _current.TransitionHints.Count == 0)
        {
            return null;
        }

        TransitionHintResponse? nearest = null;
        var nearestDistance = double.MaxValue;
        foreach (var hint in _current.TransitionHints)
        {
            var distance = Math.Sqrt(Math.Pow(hint.Position.X - currentPosition.X, 2) + Math.Pow(hint.Position.Y - currentPosition.Y, 2));
            if (distance < nearestDistance)
            {
                nearest = hint;
                nearestDistance = distance;
            }
        }

        if (nearest is null)
        {
            return null;
        }

        return new TransitionPreview(
            nearest.TransitionId,
            nearest.DisplayName,
            nearest.TargetZoneId,
            nearest.StreamingCellId,
            nearestDistance,
            nearestDistance <= nearest.Radius);
    }
}

internal sealed record TransitionPreview(
    string TransitionId,
    string DisplayName,
    string TargetZoneId,
    string StreamingCellId,
    double Distance,
    bool Ready);
