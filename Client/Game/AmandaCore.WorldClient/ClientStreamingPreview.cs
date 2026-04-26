internal sealed class ClientStreamingPreviewState
{
    private readonly IWorldStreamingPreviewSink _sink;
    private ClientStreamingFrame _current = ClientStreamingFrame.Disabled;

    public ClientStreamingPreviewState(IWorldStreamingPreviewSink? sink = null)
    {
        _sink = sink ?? NullWorldStreamingPreviewSink.Instance;
    }

    public bool Active => _current.Enabled;
    public string MapId => _current.Zone.MapId;
    public string ZoneId => _current.Zone.ZoneId;
    public ClientStreamingCell? CurrentCell => _current.CurrentCell;
    public IReadOnlyList<ClientStreamingCell> VisibleCells => _current.VisibleCells;
    public IReadOnlyList<string> AdjacentZoneIds => _current.Zone.AdjacentZoneIds;

    public ClientStreamingFrame Update(WorldSessionResponse response)
    {
        var next = ClientStreamingFrame.FromResponse(response);
        EmitPreviewChanges(_current, next);
        _current = next;
        return next;
    }

    private void EmitPreviewChanges(ClientStreamingFrame previous, ClientStreamingFrame next)
    {
        if (!next.Enabled)
        {
            foreach (var cell in previous.VisibleCells)
            {
                _sink.CellBecameHidden(cell);
            }

            if (previous.CurrentCell is not null)
            {
                _sink.CurrentCellChanged(null);
            }

            if (previous.NearestTransition is not null)
            {
                _sink.TransitionHintChanged(null);
            }

            return;
        }

        if (!previous.Enabled || previous.Zone.ZoneId != next.Zone.ZoneId)
        {
            _sink.ZoneEntered(next.Zone);
        }

        if (!previous.Enabled || previous.Zone.MapId != next.Zone.MapId || previous.Zone.Bounds != next.Zone.Bounds)
        {
            _sink.MapBoundsChanged(next.Zone.Bounds);
        }

        foreach (var cell in previous.VisibleCells.Where(cell => next.VisibleCells.All(nextCell => nextCell.CellId != cell.CellId)))
        {
            _sink.CellBecameHidden(cell);
        }

        foreach (var cell in next.VisibleCells.Where(cell => previous.VisibleCells.All(previousCell => previousCell.CellId != cell.CellId)))
        {
            _sink.CellBecameVisible(cell);
        }

        if (previous.CurrentCell?.CellId != next.CurrentCell?.CellId)
        {
            _sink.CurrentCellChanged(next.CurrentCell);
        }

        if (ShouldEmitTransitionHint(previous.NearestTransition, next.NearestTransition))
        {
            _sink.TransitionHintChanged(next.NearestTransition);
        }
    }

    private static bool ShouldEmitTransitionHint(ClientTransitionHint? previous, ClientTransitionHint? next)
    {
        if (previous is null || next is null)
        {
            return previous is not null || next is not null;
        }

        if (previous.TransitionId != next.TransitionId || previous.Ready != next.Ready)
        {
            return true;
        }

        return Math.Abs(previous.Distance - next.Distance) >= 1.0;
    }
}

internal sealed record ClientStreamingFrame(
    bool Enabled,
    ClientStreamingZone Zone,
    IReadOnlyList<ClientStreamingCell> VisibleCells,
    ClientStreamingCell? CurrentCell,
    ClientTransitionHint? NearestTransition)
{
    public static readonly ClientStreamingFrame Disabled = new(
        false,
        new ClientStreamingZone(string.Empty, string.Empty, [], ClientMapBounds.Empty),
        [],
        null,
        null);

    public static ClientStreamingFrame FromResponse(WorldSessionResponse response)
    {
        var streaming = response.Streaming;
        if (streaming is null || !streaming.Enabled)
        {
            return Disabled with
            {
                Zone = new ClientStreamingZone(response.ZoneId, string.Empty, [], ClientMapBounds.Empty)
            };
        }

        var zone = new ClientStreamingZone(
            streaming.ZoneId,
            streaming.MapId,
            streaming.AdjacentZoneIds.OrderBy(zoneID => zoneID, StringComparer.Ordinal).ToArray(),
            ClientMapBounds.FromResponse(streaming.Bounds));

        var cells = streaming.StreamingCells
            .Select(ClientStreamingCell.FromResponse)
            .OrderByDescending(cell => cell.Priority)
            .ThenBy(cell => cell.CellId, StringComparer.Ordinal)
            .ToArray();

        var currentCell = cells
            .Where(cell => cell.Bounds.Contains(response.Position))
            .OrderByDescending(cell => cell.Priority)
            .ThenBy(cell => cell.CellId, StringComparer.Ordinal)
            .FirstOrDefault();

        var nearestTransition = streaming.TransitionHints
            .Select(hint => ClientTransitionHint.FromResponse(hint, response.Position))
            .OrderBy(hint => hint.Distance)
            .ThenBy(hint => hint.TransitionId, StringComparer.Ordinal)
            .FirstOrDefault();

        return new ClientStreamingFrame(true, zone, cells, currentCell, nearestTransition);
    }
}

internal sealed record ClientStreamingZone(
    string ZoneId,
    string MapId,
    IReadOnlyList<string> AdjacentZoneIds,
    ClientMapBounds Bounds);

internal sealed record ClientStreamingCell(
    string CellId,
    string DisplayName,
    int Priority,
    IReadOnlyList<string> Tags,
    ClientMapBounds Bounds)
{
    public static ClientStreamingCell FromResponse(StreamingCellResponse response)
    {
        return new ClientStreamingCell(
            response.CellId,
            response.DisplayName,
            response.Priority,
            response.Tags.OrderBy(tag => tag, StringComparer.Ordinal).ToArray(),
            ClientMapBounds.FromResponse(response.Bounds));
    }
}

internal sealed record ClientTransitionHint(
    string TransitionId,
    string DisplayName,
    string TargetZoneId,
    string DestinationEntryId,
    string StreamingCellId,
    string Hint,
    Position Position,
    double Radius,
    double Distance,
    bool Ready)
{
    public static ClientTransitionHint FromResponse(TransitionHintResponse response, Position currentPosition)
    {
        var distance = Distance2D(response.Position, currentPosition);
        return new ClientTransitionHint(
            response.TransitionId,
            response.DisplayName,
            response.TargetZoneId,
            response.DestinationEntryId,
            response.StreamingCellId,
            response.Hint,
            response.Position,
            response.Radius,
            distance,
            distance <= response.Radius);
    }

    private static double Distance2D(Position left, Position right)
    {
        return Math.Sqrt(Math.Pow(left.X - right.X, 2) + Math.Pow(left.Y - right.Y, 2));
    }
}

internal sealed record ClientMapBounds(double MinX, double MinY, double MinZ, double MaxX, double MaxY, double MaxZ)
{
    public static readonly ClientMapBounds Empty = new(0, 0, 0, 0, 0, 0);

    public static ClientMapBounds FromResponse(MapBoundsResponse response)
    {
        return new ClientMapBounds(response.MinX, response.MinY, response.MinZ, response.MaxX, response.MaxY, response.MaxZ);
    }

    public bool Contains(Position position)
    {
        return position.X >= MinX && position.X <= MaxX &&
            position.Y >= MinY && position.Y <= MaxY &&
            position.Z >= MinZ && position.Z <= MaxZ;
    }
}
