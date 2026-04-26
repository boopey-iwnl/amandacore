var options = ClientOptions.Parse(args);
var client = new WorldClient(options.WorldEndpoint);
var streamingPreview = new ClientStreamingPreviewState(new ConsoleWorldStreamingPreviewSink());

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

    var streamingFrame = streamingPreview.Update(response);
    if (streamingFrame.Enabled)
    {
        var currentCell = streamingFrame.CurrentCell?.DisplayName ?? "outside streaming cells";
        Console.WriteLine($"Streaming: zone {streamingFrame.Zone.ZoneId}, map {streamingFrame.Zone.MapId}, current cell {currentCell}");
        Console.WriteLine($"Visible cells: {streamingFrame.VisibleCells.Count}; adjacent zones [{string.Join(", ", streamingFrame.Zone.AdjacentZoneIds)}]");
        if (streamingFrame.NearestTransition is not null)
        {
            var transition = streamingFrame.NearestTransition;
            var state = transition.Ready ? "ready" : $"distance {transition.Distance:0.0}";
            Console.WriteLine($"Nearest transition: {transition.DisplayName} -> {transition.TargetZoneId} ({state})");
        }
    }

    Console.WriteLine("Visible entities:");
    foreach (var entity in response.Entities)
    {
        Console.WriteLine($"- {entity.Kind}: {entity.DisplayName} @ ({entity.X}, {entity.Y}, {entity.Z})");
    }
}
