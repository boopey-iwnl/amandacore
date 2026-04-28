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
    public long SnapshotVersion { get; set; }
    public long DeltaVersion { get; set; }
    public string Cursor { get; set; } = string.Empty;
    public bool FullSnapshot { get; set; }
    public bool ResyncRequired { get; set; }
    public ReplicationMetadata Replication { get; set; } = new();
    public StreamingStateResponse? Streaming { get; set; } = new();
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

internal sealed class ReplicationMetadata
{
    public string ProtocolVersion { get; set; } = string.Empty;
    public string Kind { get; set; } = string.Empty;
    public long SnapshotVersion { get; set; }
    public long DeltaVersion { get; set; }
    public string Cursor { get; set; } = string.Empty;
    public CursorState CursorState { get; set; } = new();
    public bool FullSnapshot { get; set; }
    public bool ResyncRequired { get; set; }
    public string Reason { get; set; } = string.Empty;
    public List<ReplicationChangedField> Changed { get; set; } = [];
}

internal sealed class CursorState
{
    public string ShardId { get; set; } = string.Empty;
    public string ZoneId { get; set; } = string.Empty;
    public long StateVersion { get; set; }
    public long Sequence { get; set; }
    public long Tick { get; set; }
}

internal sealed class ReplicationChangedField
{
    public string Domain { get; set; } = string.Empty;
    public string EntityId { get; set; } = string.Empty;
    public List<string> Fields { get; set; } = [];
    public long Version { get; set; }
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
    public Dictionary<string, System.Text.Json.JsonElement> Fields { get; set; } = [];
}

internal sealed class StateDiffEntry
{
    public long Sequence { get; set; }
    public string Type { get; set; } = string.Empty;
    public Dictionary<string, System.Text.Json.JsonElement> Fields { get; set; } = [];
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
