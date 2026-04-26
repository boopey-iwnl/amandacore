package worlds

import (
	"fmt"
	"sort"

	contentpkg "amandacore/services/internal/content"
)

func BuildContentZoneShardAssignments(registry contentpkg.RuntimeContentRegistry, policy ShardAssignmentPolicy) (map[string]ZoneShardAssignment, error) {
	if policy.ShardCount <= 0 {
		policy.ShardCount = 1
	}
	if len(registry.Zones) == 0 {
		return nil, fmt.Errorf("cannot assign shards without loaded zones")
	}
	assignments := map[string]ZoneShardAssignment{}
	for index, zoneID := range contentpkg.SortedKeys(registry.Zones) {
		shardIndex := index % policy.ShardCount
		assignments[zoneID] = ZoneShardAssignment{
			ZoneID:  zoneID,
			ShardID: ShardID(fmt.Sprintf("zone_shard_%02d", shardIndex+1)),
			Index:   index,
		}
	}
	return assignments, nil
}

func shardAssignmentSummary(assignments map[string]ZoneShardAssignment) map[string]string {
	summary := map[string]string{}
	for _, zoneID := range sortedZoneShardKeys(assignments) {
		summary[zoneID] = string(assignments[zoneID].ShardID)
	}
	return summary
}

func sortedZoneShardKeys(assignments map[string]ZoneShardAssignment) []string {
	keys := make([]string, 0, len(assignments))
	for zoneID := range assignments {
		keys = append(keys, zoneID)
	}
	sort.Strings(keys)
	return keys
}
