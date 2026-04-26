using System.Text.Json;

internal sealed class PlaceholderSceneStreamingAdapter : IWorldStreamingPreviewSink
{
    private readonly IPlaceholderSceneCommandSink _commandSink;
    private ClientStreamingZone? _activeZone;
    private ClientMapBounds? _lastBounds;

    public PlaceholderSceneStreamingAdapter(IPlaceholderSceneCommandSink commandSink)
    {
        _commandSink = commandSink;
    }

    public void ZoneEntered(ClientStreamingZone zone)
    {
        _activeZone = zone;
        EmitZoneBounds(zone.Bounds);
    }

    public void CellBecameVisible(ClientStreamingCell cell)
    {
        _commandSink.Emit(PlaceholderSceneCommand.CreateStreamingCellVolume(
            _activeZone?.ZoneId ?? string.Empty,
            _activeZone?.MapId ?? string.Empty,
            cell.CellId,
            cell.DisplayName,
            cell.Bounds,
            cell.Tags));
    }

    public void CellBecameHidden(ClientStreamingCell cell)
    {
        _commandSink.Emit(PlaceholderSceneCommand.HideStreamingCellVolume(
            _activeZone?.ZoneId ?? string.Empty,
            _activeZone?.MapId ?? string.Empty,
            cell.CellId,
            cell.DisplayName));
    }

    public void CurrentCellChanged(ClientStreamingCell? cell)
    {
        if (cell is null)
        {
            _commandSink.Emit(PlaceholderSceneCommand.ClearCurrentCellHighlight(
                _activeZone?.ZoneId ?? string.Empty,
                _activeZone?.MapId ?? string.Empty));
            return;
        }

        _commandSink.Emit(PlaceholderSceneCommand.HighlightCurrentCell(
            _activeZone?.ZoneId ?? string.Empty,
            _activeZone?.MapId ?? string.Empty,
            cell.CellId,
            cell.DisplayName));
    }

    public void TransitionHintChanged(ClientTransitionHint? transition)
    {
        if (transition is null)
        {
            _commandSink.Emit(PlaceholderSceneCommand.ClearTransitionAffordance(
                _activeZone?.ZoneId ?? string.Empty,
                _activeZone?.MapId ?? string.Empty));
            return;
        }

        _commandSink.Emit(PlaceholderSceneCommand.ShowTransitionAffordance(
            _activeZone?.ZoneId ?? string.Empty,
            _activeZone?.MapId ?? string.Empty,
            transition.TransitionId,
            transition.DisplayName,
            transition.TargetZoneId,
            transition.StreamingCellId,
            transition.Position,
            transition.Radius,
            transition.Ready,
            transition.Hint));
    }

    public void MapBoundsChanged(ClientMapBounds bounds)
    {
        if (_lastBounds == bounds)
        {
            return;
        }

        EmitZoneBounds(bounds);
    }

    private void EmitZoneBounds(ClientMapBounds bounds)
    {
        _lastBounds = bounds;
        _commandSink.Emit(PlaceholderSceneCommand.CreateZoneBoundsVolume(
            _activeZone?.ZoneId ?? string.Empty,
            _activeZone?.MapId ?? string.Empty,
            bounds));
    }
}

internal interface IPlaceholderSceneCommandSink
{
    void Emit(PlaceholderSceneCommand command);
}

internal sealed class CompositePlaceholderSceneCommandSink : IPlaceholderSceneCommandSink
{
    private readonly IReadOnlyList<IPlaceholderSceneCommandSink> _sinks;

    public CompositePlaceholderSceneCommandSink(IReadOnlyList<IPlaceholderSceneCommandSink> sinks)
    {
        _sinks = sinks;
    }

    public void Emit(PlaceholderSceneCommand command)
    {
        foreach (var sink in _sinks)
        {
            sink.Emit(command);
        }
    }
}

internal sealed class ConsolePlaceholderSceneCommandSink : IPlaceholderSceneCommandSink
{
    private readonly JsonSerializerOptions _jsonOptions = new() { WriteIndented = false };

    public void Emit(PlaceholderSceneCommand command)
    {
        Console.WriteLine($"[scene-command] {JsonSerializer.Serialize(command, _jsonOptions)}");
    }
}

internal sealed class JsonLinesPlaceholderSceneCommandSink : IPlaceholderSceneCommandSink
{
    private readonly string _path;
    private readonly JsonSerializerOptions _jsonOptions = new() { WriteIndented = false };

    public JsonLinesPlaceholderSceneCommandSink(string path)
    {
        _path = Path.GetFullPath(path);
        var directory = Path.GetDirectoryName(_path);
        if (!string.IsNullOrWhiteSpace(directory))
        {
            Directory.CreateDirectory(directory);
        }
        File.WriteAllText(_path, string.Empty);
    }

    public void Emit(PlaceholderSceneCommand command)
    {
        File.AppendAllText(_path, JsonSerializer.Serialize(command, _jsonOptions) + Environment.NewLine);
    }
}

internal sealed class RecordingPlaceholderSceneCommandSink : IPlaceholderSceneCommandSink
{
    private readonly List<PlaceholderSceneCommand> _commands = [];

    public IReadOnlyList<PlaceholderSceneCommand> Commands => _commands;

    public void Clear()
    {
        _commands.Clear();
    }

    public void Emit(PlaceholderSceneCommand command)
    {
        _commands.Add(command);
    }
}

internal sealed record PlaceholderSceneCommand(
    string Command,
    string ZoneId,
    string MapId,
    string? CellId = null,
    string? TransitionId = null,
    string? TargetZoneId = null,
    string? StreamingCellId = null,
    string? DisplayName = null,
    string? Hint = null,
    ClientMapBounds? Bounds = null,
    Position? Position = null,
    double? Radius = null,
    bool? Ready = null,
    IReadOnlyList<string>? Tags = null)
{
    public static PlaceholderSceneCommand CreateZoneBoundsVolume(string zoneId, string mapId, ClientMapBounds bounds)
    {
        return new PlaceholderSceneCommand(
            PlaceholderSceneCommandNames.CreateZoneBoundsVolume,
            zoneId,
            mapId,
            Bounds: bounds);
    }

    public static PlaceholderSceneCommand CreateStreamingCellVolume(
        string zoneId,
        string mapId,
        string cellId,
        string displayName,
        ClientMapBounds bounds,
        IReadOnlyList<string> tags)
    {
        return new PlaceholderSceneCommand(
            PlaceholderSceneCommandNames.CreateStreamingCellVolume,
            zoneId,
            mapId,
            CellId: cellId,
            DisplayName: displayName,
            Bounds: bounds,
            Tags: tags.ToArray());
    }

    public static PlaceholderSceneCommand HideStreamingCellVolume(string zoneId, string mapId, string cellId, string displayName)
    {
        return new PlaceholderSceneCommand(
            PlaceholderSceneCommandNames.HideStreamingCellVolume,
            zoneId,
            mapId,
            CellId: cellId,
            DisplayName: displayName);
    }

    public static PlaceholderSceneCommand HighlightCurrentCell(string zoneId, string mapId, string cellId, string displayName)
    {
        return new PlaceholderSceneCommand(
            PlaceholderSceneCommandNames.HighlightCurrentCell,
            zoneId,
            mapId,
            CellId: cellId,
            DisplayName: displayName);
    }

    public static PlaceholderSceneCommand ClearCurrentCellHighlight(string zoneId, string mapId)
    {
        return new PlaceholderSceneCommand(
            PlaceholderSceneCommandNames.ClearCurrentCellHighlight,
            zoneId,
            mapId);
    }

    public static PlaceholderSceneCommand ShowTransitionAffordance(
        string zoneId,
        string mapId,
        string transitionId,
        string displayName,
        string targetZoneId,
        string streamingCellId,
        Position position,
        double radius,
        bool ready,
        string hint)
    {
        return new PlaceholderSceneCommand(
            PlaceholderSceneCommandNames.ShowTransitionAffordance,
            zoneId,
            mapId,
            TransitionId: transitionId,
            TargetZoneId: targetZoneId,
            StreamingCellId: streamingCellId,
            DisplayName: displayName,
            Hint: hint,
            Position: position,
            Radius: radius,
            Ready: ready);
    }

    public static PlaceholderSceneCommand ClearTransitionAffordance(string zoneId, string mapId)
    {
        return new PlaceholderSceneCommand(
            PlaceholderSceneCommandNames.ClearTransitionAffordance,
            zoneId,
            mapId);
    }
}

internal static class PlaceholderSceneCommandNames
{
    public const string CreateZoneBoundsVolume = "CreateZoneBoundsVolume";
    public const string CreateStreamingCellVolume = "CreateStreamingCellVolume";
    public const string HighlightCurrentCell = "HighlightCurrentCell";
    public const string ClearCurrentCellHighlight = "ClearCurrentCellHighlight";
    public const string HideStreamingCellVolume = "HideStreamingCellVolume";
    public const string ShowTransitionAffordance = "ShowTransitionAffordance";
    public const string ClearTransitionAffordance = "ClearTransitionAffordance";
}
