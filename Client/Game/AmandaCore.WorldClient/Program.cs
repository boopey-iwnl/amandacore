using System.Net.Http.Json;
using System.Text.Json;

const string DevBasicStrikeAbilityId = "dev_basic_strike";

var options = ClientOptions.Parse(args);
var client = new WorldClient(options.WorldEndpoint);
var eventCursor = new CombatEventCursor();

var session = await client.ConnectAsync(options.JoinTicketId);
PrintState("Connected", session, eventCursor);

if (options.AutoDemo)
{
    session = await client.MoveAsync(session.WorldSessionToken, 5, 0);
    PrintState("Moved east", session, eventCursor);

    session = await client.MoveAsync(session.WorldSessionToken, 0, 3);
    PrintState("Moved north", session, eventCursor);

    await client.DisconnectAsync(session.WorldSessionToken);
    Console.WriteLine("Disconnected.");

    session = await client.ReconnectAsync(session.WorldSessionToken);
    PrintState("Reconnected", session, eventCursor);
    return;
}

if (options.AutoCombatDemo)
{
    session = await RunAutoCombatDemoAsync(client, session, eventCursor);
    await client.DisconnectAsync(session.WorldSessionToken);
    Console.WriteLine("Disconnected.");
    return;
}

Console.WriteLine("Controls: W/A/S/D move, T target nearest hostile, F Basic Strike, P poll, R reconnect, X disconnect, Q quit");
var current = session;
var disconnected = false;

while (true)
{
    var key = Console.ReadKey(intercept: true);
    try
    {
        switch (key.Key)
        {
            case ConsoleKey.W:
                current = await client.MoveAsync(current.WorldSessionToken, 0, 1);
                PrintState("Moved north", current, eventCursor);
                disconnected = false;
                break;
            case ConsoleKey.S:
                current = await client.MoveAsync(current.WorldSessionToken, 0, -1);
                PrintState("Moved south", current, eventCursor);
                disconnected = false;
                break;
            case ConsoleKey.A:
                current = await client.MoveAsync(current.WorldSessionToken, -1, 0);
                PrintState("Moved west", current, eventCursor);
                disconnected = false;
                break;
            case ConsoleKey.D:
                current = await client.MoveAsync(current.WorldSessionToken, 1, 0);
                PrintState("Moved east", current, eventCursor);
                disconnected = false;
                break;
            case ConsoleKey.T:
                current = await SelectNearestHostileAsync(client, current);
                PrintState("Target selected", current, eventCursor);
                disconnected = false;
                break;
            case ConsoleKey.F:
                current = await client.UseAbilityAsync(current.WorldSessionToken, DevBasicStrikeAbilityId);
                PrintState("Basic Strike", current, eventCursor);
                disconnected = false;
                break;
            case ConsoleKey.P:
                current = await client.GetStateAsync(current.WorldSessionToken);
                PrintState("Polled", current, eventCursor);
                disconnected = false;
                break;
            case ConsoleKey.R:
                current = await client.ReconnectAsync(current.WorldSessionToken);
                PrintState("Reconnected", current, eventCursor);
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
    catch (Exception exception)
    {
        Console.WriteLine();
        Console.WriteLine($"Command rejected: {exception.Message}");
        current = await client.GetStateAsync(current.WorldSessionToken);
        PrintState("Authoritative state", current, eventCursor);
    }
}

static async Task<WorldSessionResponse> RunAutoCombatDemoAsync(WorldClient client, WorldSessionResponse session, CombatEventCursor eventCursor)
{
    Console.WriteLine("Starting auto combat demo.");
    session = await client.GetStateAsync(session.WorldSessionToken);
    PrintState("Combat demo initial state", session, eventCursor);

    var target = session.FindNearestHostile();
    if (target is null)
    {
        Console.WriteLine("No hostile target is visible.");
        return session;
    }

    session = await MoveNearTargetAsync(client, session, target);
    session = await client.TargetAsync(session.WorldSessionToken, target.Id);
    PrintState($"Targeted {target.DisplayName}", session, eventCursor);

    for (var attempt = 1; attempt <= 8; attempt++)
    {
        var currentTarget = session.FindEntity(session.CurrentTargetId);
        if (currentTarget is null || !currentTarget.Alive)
        {
            PrintState("Target defeated", session, eventCursor);
            break;
        }

        try
        {
            session = await client.UseAbilityAsync(session.WorldSessionToken, DevBasicStrikeAbilityId);
            PrintState($"Basic Strike {attempt}", session, eventCursor);
        }
        catch (Exception exception)
        {
            Console.WriteLine($"Basic Strike {attempt} rejected: {exception.Message}");
            session = await client.GetStateAsync(session.WorldSessionToken);
            PrintState("Authoritative state", session, eventCursor);
        }

        await Task.Delay(TimeSpan.FromMilliseconds(1550));
        session = await client.GetStateAsync(session.WorldSessionToken);
    }

    return session;
}

static async Task<WorldSessionResponse> SelectNearestHostileAsync(WorldClient client, WorldSessionResponse session)
{
    var target = session.FindNearestHostile();
    if (target is null)
    {
        throw new InvalidOperationException("No alive hostile target is visible.");
    }

    session = await MoveNearTargetAsync(client, session, target);
    return await client.TargetAsync(session.WorldSessionToken, target.Id);
}

static async Task<WorldSessionResponse> MoveNearTargetAsync(WorldClient client, WorldSessionResponse session, VisibleEntity target)
{
    var distance = session.DistanceTo(target);
    if (distance <= 2.25)
    {
        return session;
    }

    var directionX = session.Position.X - target.X;
    var directionY = session.Position.Y - target.Y;
    var magnitude = Math.Sqrt((directionX * directionX) + (directionY * directionY));
    if (magnitude <= 0.001)
    {
        directionX = -1;
        directionY = 0;
        magnitude = 1;
    }

    var desiredX = target.X + (directionX / magnitude * 1.5);
    var desiredY = target.Y + (directionY / magnitude * 1.5);
    return await client.MoveAsync(
        session.WorldSessionToken,
        desiredX - session.Position.X,
        desiredY - session.Position.Y);
}

static void PrintState(string label, WorldSessionResponse response, CombatEventCursor cursor)
{
    Console.WriteLine();
    Console.WriteLine($"[{label}] {response.DisplayName} in {response.ZoneId}");
    Console.WriteLine($"Position: ({response.Position.X:0.##}, {response.Position.Y:0.##}, {response.Position.Z:0.##})");
    Console.WriteLine($"Health: {response.Health:0.#}/{response.MaxHealth:0.#} | {response.ResourceName}: {response.Resource:0.#}/{response.MaxResource:0.#} | Alive: {response.Alive}");
    PrintTargetFrame(response);
    PrintActionBar(response);
    PrintVisibleEntities(response);
    PrintKillCredits(response);
    PrintRecentCombatUpdates(response, cursor);
}

static void PrintTargetFrame(WorldSessionResponse response)
{
    if (string.IsNullOrWhiteSpace(response.CurrentTargetId))
    {
        Console.WriteLine("Target: none");
        return;
    }

    var target = response.FindEntity(response.CurrentTargetId);
    if (target is null)
    {
        Console.WriteLine($"Target: {response.CurrentTargetId} (not visible)");
        return;
    }

    var distance = response.DistanceTo(target);
    var auraText = target.Auras.Count == 0
        ? "none"
        : string.Join(", ", target.Auras.Select(aura => $"{aura.DisplayName} x{aura.StackCount}"));
    Console.WriteLine($"Target: {target.DisplayName} [{target.Kind}] {target.Health:0.#}/{target.MaxHealth:0.#} hp | alive={target.Alive} | distance={distance:0.##} | auras={auraText}");
}

static void PrintActionBar(WorldSessionResponse response)
{
    var slots = response.ActionBar
        .Where(slot => slot.Learned && !string.IsNullOrWhiteSpace(slot.AbilityId))
        .OrderBy(slot => slot.SlotIndex)
        .Take(8)
        .ToList();
    if (slots.Count == 0)
    {
        Console.WriteLine("Action bar: no learned abilities reported.");
        return;
    }

    Console.WriteLine("Action bar:");
    foreach (var slot in slots)
    {
        var hotkey = string.IsNullOrWhiteSpace(slot.Hotkey) ? "-" : slot.Hotkey;
        var cooldown = slot.CooldownRemainingMs > 0
            ? $"cd {slot.CooldownRemainingMs}ms"
            : "ready";
        Console.WriteLine($"- [{hotkey}] {slot.DisplayName} ({slot.AbilityId}) {cooldown}");
    }

    var globalCooldown = Math.Max(0, response.GlobalCooldownEndsAt - DateTimeOffset.UtcNow.ToUnixTimeMilliseconds());
    if (globalCooldown > 0)
    {
        Console.WriteLine($"Global cooldown: {globalCooldown}ms");
    }

    if (!string.IsNullOrWhiteSpace(response.CastingAbilityId))
    {
        var castRemaining = Math.Max(0, response.CastEndsAt - DateTimeOffset.UtcNow.ToUnixTimeMilliseconds());
        Console.WriteLine($"Casting: {response.CastingAbilityId} ({castRemaining}ms remaining)");
    }
}

static void PrintVisibleEntities(WorldSessionResponse response)
{
    Console.WriteLine("Visible entities:");
    foreach (var entity in response.Entities.OrderBy(entity => response.DistanceTo(entity)).Take(10))
    {
        var targetMarker = entity.Id == response.CurrentTargetId ? "*" : "-";
        var combat = entity.IsInCombat ? " combat" : string.Empty;
        Console.WriteLine($"{targetMarker} {entity.Kind}: {entity.DisplayName} @ ({entity.X:0.##}, {entity.Y:0.##}, {entity.Z:0.##}) hp={entity.Health:0.#}/{entity.MaxHealth:0.#} alive={entity.Alive}{combat}");
    }
}

static void PrintKillCredits(WorldSessionResponse response)
{
    if (response.KillCredits.Count == 0)
    {
        return;
    }

    Console.WriteLine("Kill credits:");
    foreach (var credit in response.KillCredits.OrderBy(credit => credit.ArchetypeId))
    {
        Console.WriteLine($"- {credit.ArchetypeId}: {credit.Count} ({credit.Reason})");
    }
}

static void PrintRecentCombatUpdates(WorldSessionResponse response, CombatEventCursor cursor)
{
    var newDiffs = response.StateDiffs
        .Where(diff => diff.Sequence > cursor.LastStateDiffSequence && IsCombatDiff(diff.Type))
        .OrderBy(diff => diff.Sequence)
        .ToList();
    var newEvents = response.DomainEvents
        .Where(domainEvent => domainEvent.Sequence > cursor.LastDomainEventSequence && IsCombatEvent(domainEvent.Type))
        .OrderBy(domainEvent => domainEvent.Sequence)
        .ToList();

    if (newDiffs.Count > 0)
    {
        Console.WriteLine("State updates:");
        foreach (var diff in newDiffs.TakeLast(6))
        {
            Console.WriteLine($"- #{diff.Sequence} {diff.Type} {FormatFields(diff.Fields)}");
        }
    }

    if (newEvents.Count > 0)
    {
        Console.WriteLine("Combat events:");
        foreach (var domainEvent in newEvents.TakeLast(6))
        {
            Console.WriteLine($"- #{domainEvent.Sequence} {domainEvent.Type} {FormatFields(domainEvent.Fields)}");
        }
    }

    cursor.Record(response);
}

static bool IsCombatDiff(string type)
{
    return type is
        "EntityHealthDelta" or
        "EntityCombatStateDelta" or
        "TargetSelectionDelta" or
        "AbilityResultDelta" or
        "EntityDeathDelta" or
        "ProgressionDelta" or
        "AuraStateDelta" or
        "CooldownDelta" or
        "CastStateDelta";
}

static bool IsCombatEvent(string type)
{
    return type.StartsWith("combat.", StringComparison.OrdinalIgnoreCase) ||
        type.StartsWith("npc.", StringComparison.OrdinalIgnoreCase) ||
        type.StartsWith("entity.", StringComparison.OrdinalIgnoreCase) ||
        type.StartsWith("ability.", StringComparison.OrdinalIgnoreCase) ||
        type.StartsWith("aura.", StringComparison.OrdinalIgnoreCase) ||
        type.StartsWith("cooldown.", StringComparison.OrdinalIgnoreCase) ||
        type.StartsWith("progression.kill_credit", StringComparison.OrdinalIgnoreCase);
}

static string FormatFields(Dictionary<string, JsonElement> fields)
{
    if (fields.Count == 0)
    {
        return string.Empty;
    }

    var included = fields
        .Where(pair => pair.Key is "entityId" or "targetId" or "targetEntityId" or "abilityId" or "damage" or "health" or "maxHealth" or "alive" or "action" or "auraId" or "count" or "reason")
        .Take(6)
        .Select(pair => $"{pair.Key}={FormatJsonValue(pair.Value)}");
    return string.Join(" ", included);
}

static string FormatJsonValue(JsonElement value)
{
    return value.ValueKind switch
    {
        JsonValueKind.String => value.GetString() ?? string.Empty,
        JsonValueKind.Number => value.TryGetInt64(out var longValue)
            ? longValue.ToString()
            : value.TryGetDouble(out var doubleValue)
                ? doubleValue.ToString("0.##")
                : value.GetRawText(),
        JsonValueKind.True => "true",
        JsonValueKind.False => "false",
        JsonValueKind.Null => "null",
        _ => value.GetRawText()
    };
}

internal sealed record ClientOptions(string JoinTicketId, string WorldEndpoint, bool AutoDemo, bool AutoCombatDemo)
{
    public static ClientOptions Parse(string[] args)
    {
        string? joinTicket = null;
        var worldEndpoint = "http://127.0.0.1:8085";
        var autoDemo = false;
        var autoCombatDemo = false;

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
                case "--auto-combat-demo":
                    autoCombatDemo = true;
                    break;
            }
        }

        if (string.IsNullOrWhiteSpace(joinTicket))
        {
            throw new InvalidOperationException("A --join-ticket value is required.");
        }

        return new ClientOptions(joinTicket, worldEndpoint, autoDemo, autoCombatDemo);
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

internal sealed class CombatEventCursor
{
    public long LastDomainEventSequence { get; private set; }
    public long LastStateDiffSequence { get; private set; }

    public void Record(WorldSessionResponse response)
    {
        LastDomainEventSequence = Math.Max(LastDomainEventSequence, response.DomainEvents.Select(domainEvent => domainEvent.Sequence).DefaultIfEmpty(0).Max());
        LastStateDiffSequence = Math.Max(LastStateDiffSequence, response.StateDiffs.Select(diff => diff.Sequence).DefaultIfEmpty(0).Max());
    }
}

internal sealed class WorldSessionResponse
{
    public string WorldSessionToken { get; set; } = string.Empty;
    public string CharacterId { get; set; } = string.Empty;
    public string RealmId { get; set; } = string.Empty;
    public string ZoneId { get; set; } = string.Empty;
    public string DisplayName { get; set; } = string.Empty;
    public int Level { get; set; }
    public Position Position { get; set; } = new();
    public double Health { get; set; }
    public double MaxHealth { get; set; }
    public double Resource { get; set; }
    public double MaxResource { get; set; }
    public string ResourceName { get; set; } = string.Empty;
    public bool Alive { get; set; }
    public string CurrentTargetId { get; set; } = string.Empty;
    public bool AutoAttackActive { get; set; }
    public long GlobalCooldownEndsAt { get; set; }
    public long CastEndsAt { get; set; }
    public string CastingAbilityId { get; set; } = string.Empty;
    public List<AuraState> Auras { get; set; } = [];
    public List<ActionBarSlot> ActionBar { get; set; } = [];
    public List<KillCreditEntry> KillCredits { get; set; } = [];
    public List<DomainEventEntry> DomainEvents { get; set; } = [];
    public List<StateDiffEntry> StateDiffs { get; set; } = [];
    public List<VisibleEntity> Entities { get; set; } = [];

    public VisibleEntity? FindEntity(string entityId)
    {
        if (string.IsNullOrWhiteSpace(entityId))
        {
            return null;
        }

        return Entities.FirstOrDefault(entity => string.Equals(entity.Id, entityId, StringComparison.OrdinalIgnoreCase));
    }

    public VisibleEntity? FindNearestHostile()
    {
        return Entities
            .Where(entity => entity.Alive && entity.Targetable && entity.IsHostile)
            .OrderBy(DistanceTo)
            .FirstOrDefault();
    }

    public double DistanceTo(VisibleEntity entity)
    {
        var deltaX = entity.X - Position.X;
        var deltaY = entity.Y - Position.Y;
        return Math.Sqrt((deltaX * deltaX) + (deltaY * deltaY));
    }
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
    public string ArchetypeId { get; set; } = string.Empty;
    public string DisplayName { get; set; } = string.Empty;
    public string Kind { get; set; } = string.Empty;
    public string Disposition { get; set; } = string.Empty;
    public double X { get; set; }
    public double Y { get; set; }
    public double Z { get; set; }
    public double Health { get; set; }
    public double MaxHealth { get; set; }
    public bool Alive { get; set; }
    public bool Targetable { get; set; }
    public bool IsInCombat { get; set; }
    public string CurrentTargetEntityId { get; set; } = string.Empty;
    public string AiState { get; set; } = string.Empty;
    public List<AuraState> Auras { get; set; } = [];

    public bool IsHostile =>
        string.Equals(Disposition, "Hostile", StringComparison.OrdinalIgnoreCase) ||
        string.Equals(Kind, "hostile_mob", StringComparison.OrdinalIgnoreCase);
}

internal sealed class ActionBarSlot
{
    public int SlotIndex { get; set; }
    public string Hotkey { get; set; } = string.Empty;
    public string AbilityId { get; set; } = string.Empty;
    public string DisplayName { get; set; } = string.Empty;
    public bool Learned { get; set; }
    public long CooldownRemainingMs { get; set; }
}

internal sealed class AuraState
{
    public string AuraId { get; set; } = string.Empty;
    public string DisplayName { get; set; } = string.Empty;
    public string Kind { get; set; } = string.Empty;
    public string SourceEntityId { get; set; } = string.Empty;
    public string TargetEntityId { get; set; } = string.Empty;
    public int StackCount { get; set; }
    public long ExpiresAtMs { get; set; }
    public long NextTickAtMs { get; set; }
}

internal sealed class KillCreditEntry
{
    public string ArchetypeId { get; set; } = string.Empty;
    public int Count { get; set; }
    public string Reason { get; set; } = string.Empty;
}

internal sealed class DomainEventEntry
{
    public long Sequence { get; set; }
    public string Type { get; set; } = string.Empty;
    public Dictionary<string, JsonElement> Fields { get; set; } = [];
}

internal sealed class StateDiffEntry
{
    public long Sequence { get; set; }
    public string Type { get; set; } = string.Empty;
    public Dictionary<string, JsonElement> Fields { get; set; } = [];
}
