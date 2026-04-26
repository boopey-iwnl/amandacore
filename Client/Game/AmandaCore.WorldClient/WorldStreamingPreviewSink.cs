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
