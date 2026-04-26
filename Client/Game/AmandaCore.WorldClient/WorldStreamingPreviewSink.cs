internal interface IWorldStreamingPreviewSink
{
    void ZoneEntered(ClientStreamingZone zone);
    void CellBecameVisible(ClientStreamingCell cell);
    void CellBecameHidden(ClientStreamingCell cell);
    void CurrentCellChanged(ClientStreamingCell? cell);
    void TransitionHintChanged(ClientTransitionHint? transition);
    void MapBoundsChanged(ClientMapBounds bounds);
}

internal sealed class ConsoleWorldStreamingPreviewSink : IWorldStreamingPreviewSink
{
    public void ZoneEntered(ClientStreamingZone zone)
    {
        Console.WriteLine($"[streaming] zone entered: {zone.ZoneId} on map {zone.MapId}");
    }

    public void CellBecameVisible(ClientStreamingCell cell)
    {
        Console.WriteLine($"[streaming] cell visible: {cell.DisplayName} ({cell.CellId})");
    }

    public void CellBecameHidden(ClientStreamingCell cell)
    {
        Console.WriteLine($"[streaming] cell hidden: {cell.DisplayName} ({cell.CellId})");
    }

    public void CurrentCellChanged(ClientStreamingCell? cell)
    {
        var label = cell is null ? "outside streaming cells" : $"{cell.DisplayName} ({cell.CellId})";
        Console.WriteLine($"[streaming] current cell: {label}");
    }

    public void TransitionHintChanged(ClientTransitionHint? transition)
    {
        if (transition is null)
        {
            Console.WriteLine("[streaming] nearest transition: none");
            return;
        }

        var state = transition.Ready ? "ready" : $"distance {transition.Distance:0.0}";
        Console.WriteLine($"[streaming] nearest transition: {transition.DisplayName} -> {transition.TargetZoneId} ({state})");
    }

    public void MapBoundsChanged(ClientMapBounds bounds)
    {
        Console.WriteLine($"[streaming] map bounds: ({bounds.MinX}, {bounds.MinY}, {bounds.MinZ}) -> ({bounds.MaxX}, {bounds.MaxY}, {bounds.MaxZ})");
    }
}

internal sealed class NullWorldStreamingPreviewSink : IWorldStreamingPreviewSink
{
    public static readonly NullWorldStreamingPreviewSink Instance = new();

    private NullWorldStreamingPreviewSink()
    {
    }

    public void ZoneEntered(ClientStreamingZone zone)
    {
    }

    public void CellBecameVisible(ClientStreamingCell cell)
    {
    }

    public void CellBecameHidden(ClientStreamingCell cell)
    {
    }

    public void CurrentCellChanged(ClientStreamingCell? cell)
    {
    }

    public void TransitionHintChanged(ClientTransitionHint? transition)
    {
    }

    public void MapBoundsChanged(ClientMapBounds bounds)
    {
    }
}

internal static class WorldStreamingPreviewSinkFactory
{
    public static IWorldStreamingPreviewSink Create(StreamingSinkMode mode, string? commandFilePath = null)
    {
        var commandSinks = new List<IPlaceholderSceneCommandSink>();
        if (mode is StreamingSinkMode.SceneCommands or StreamingSinkMode.Both)
        {
            commandSinks.Add(new ConsolePlaceholderSceneCommandSink());
        }
        if (!string.IsNullOrWhiteSpace(commandFilePath))
        {
            commandSinks.Add(new JsonLinesPlaceholderSceneCommandSink(commandFilePath));
        }

        var sceneCommandSink = commandSinks.Count switch
        {
            0 => null,
            1 => commandSinks[0],
            _ => new CompositePlaceholderSceneCommandSink(commandSinks)
        };

        IWorldStreamingPreviewSink primarySink = mode switch
        {
            StreamingSinkMode.Console => new ConsoleWorldStreamingPreviewSink(),
            StreamingSinkMode.SceneCommands => new PlaceholderSceneStreamingAdapter(sceneCommandSink ?? new ConsolePlaceholderSceneCommandSink()),
            StreamingSinkMode.Both => new CompositeWorldStreamingPreviewSink(
                new ConsoleWorldStreamingPreviewSink(),
                new PlaceholderSceneStreamingAdapter(sceneCommandSink ?? new ConsolePlaceholderSceneCommandSink())),
            _ => new ConsoleWorldStreamingPreviewSink()
        };

        if (sceneCommandSink is not null && mode == StreamingSinkMode.Console)
        {
            return new CompositeWorldStreamingPreviewSink(
                primarySink,
                new PlaceholderSceneStreamingAdapter(sceneCommandSink));
        }

        return primarySink;
    }
}

internal sealed class CompositeWorldStreamingPreviewSink : IWorldStreamingPreviewSink
{
    private readonly IWorldStreamingPreviewSink[] _sinks;

    public CompositeWorldStreamingPreviewSink(params IWorldStreamingPreviewSink[] sinks)
    {
        _sinks = sinks;
    }

    public void ZoneEntered(ClientStreamingZone zone)
    {
        foreach (var sink in _sinks)
        {
            sink.ZoneEntered(zone);
        }
    }

    public void CellBecameVisible(ClientStreamingCell cell)
    {
        foreach (var sink in _sinks)
        {
            sink.CellBecameVisible(cell);
        }
    }

    public void CellBecameHidden(ClientStreamingCell cell)
    {
        foreach (var sink in _sinks)
        {
            sink.CellBecameHidden(cell);
        }
    }

    public void CurrentCellChanged(ClientStreamingCell? cell)
    {
        foreach (var sink in _sinks)
        {
            sink.CurrentCellChanged(cell);
        }
    }

    public void TransitionHintChanged(ClientTransitionHint? transition)
    {
        foreach (var sink in _sinks)
        {
            sink.TransitionHintChanged(transition);
        }
    }

    public void MapBoundsChanged(ClientMapBounds bounds)
    {
        foreach (var sink in _sinks)
        {
            sink.MapBoundsChanged(bounds);
        }
    }
}
