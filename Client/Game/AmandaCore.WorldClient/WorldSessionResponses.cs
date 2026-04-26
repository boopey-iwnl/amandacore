internal sealed class WorldSessionResponse
{
    public string WorldSessionToken { get; set; } = string.Empty;
    public string CharacterId { get; set; } = string.Empty;
    public string RealmId { get; set; } = string.Empty;
    public string ZoneId { get; set; } = string.Empty;
    public string DisplayName { get; set; } = string.Empty;
    public Position Position { get; set; } = new();
    public StreamingStateResponse? Streaming { get; set; } = new();
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
    public MapBoundsResponse Bounds { get; set; } = new();
    public List<TransitionHintResponse> TransitionHints { get; set; } = [];
    public List<StreamingCellResponse> StreamingCells { get; set; } = [];
}

internal sealed class MapBoundsResponse
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
    public MapBoundsResponse Bounds { get; set; } = new();
}
