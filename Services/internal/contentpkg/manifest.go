package contentpkg

import "fmt"

type Manifest struct {
	PackageID     string                 `json:"packageId"`
	Version       string                 `json:"version"`
	DisplayName   string                 `json:"displayName"`
	Zones         []ZoneManifest         `json:"zones"`
	SpawnPoints   []SpawnPointManifest   `json:"spawnPoints"`
	NPCArchetypes []NPCArchetypeManifest `json:"npcArchetypes"`
	Abilities     []AbilitySpecManifest  `json:"abilities"`
	LootRules     []LootRuleManifest     `json:"lootRules"`
	Quests        []QuestGraphManifest   `json:"quests"`
	Dialogues     []DialogueManifest     `json:"dialogues"`
}

type ZoneManifest struct {
	ZoneID      string `json:"zoneId"`
	DisplayName string `json:"displayName"`
	RegionID    string `json:"regionId,omitempty"`
}

type SpawnPointManifest struct {
	SpawnPointID string  `json:"spawnPointId"`
	ZoneID       string  `json:"zoneId"`
	EntityRef    string  `json:"entityRef"`
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	Z            float64 `json:"z"`
}

type NPCArchetypeManifest struct {
	ArchetypeID string   `json:"archetypeId"`
	DisplayName string   `json:"displayName"`
	Roles       []string `json:"roles,omitempty"`
}

type AbilitySpecManifest struct {
	AbilityID   string   `json:"abilityId"`
	DisplayName string   `json:"displayName"`
	EffectRefs  []string `json:"effectRefs,omitempty"`
}

type LootRuleManifest struct {
	RewardRuleID string   `json:"rewardRuleId"`
	DisplayName  string   `json:"displayName"`
	ItemRefs     []string `json:"itemRefs,omitempty"`
}

type QuestGraphManifest struct {
	QuestID      string   `json:"questId"`
	DisplayName  string   `json:"displayName"`
	ObjectiveIDs []string `json:"objectiveIds,omitempty"`
	RewardRefs   []string `json:"rewardRefs,omitempty"`
}

type DialogueManifest struct {
	DialogueID string   `json:"dialogueId"`
	SpeakerRef string   `json:"speakerRef,omitempty"`
	TopicIDs   []string `json:"topicIds,omitempty"`
}

func (m Manifest) Validate() error {
	if m.PackageID == "" {
		return fmt.Errorf("package id is required")
	}
	if m.Version == "" {
		return fmt.Errorf("package version is required")
	}
	if err := requireUniqueIDs("zone", zoneIDs(m.Zones)); err != nil {
		return err
	}
	if err := requireUniqueIDs("spawn point", spawnPointIDs(m.SpawnPoints)); err != nil {
		return err
	}
	if err := requireUniqueIDs("npc archetype", npcArchetypeIDs(m.NPCArchetypes)); err != nil {
		return err
	}
	if err := requireUniqueIDs("ability", abilityIDs(m.Abilities)); err != nil {
		return err
	}
	if err := requireUniqueIDs("loot rule", lootRuleIDs(m.LootRules)); err != nil {
		return err
	}
	if err := requireUniqueIDs("quest", questIDs(m.Quests)); err != nil {
		return err
	}
	if err := requireUniqueIDs("dialogue", dialogueIDs(m.Dialogues)); err != nil {
		return err
	}
	return nil
}

func requireUniqueIDs(kind string, ids []string) error {
	seen := map[string]struct{}{}
	for _, id := range ids {
		if id == "" {
			return fmt.Errorf("%s id is required", kind)
		}
		if _, exists := seen[id]; exists {
			return fmt.Errorf("%s id %q is duplicated", kind, id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

func zoneIDs(source []ZoneManifest) []string {
	ids := make([]string, 0, len(source))
	for _, item := range source {
		ids = append(ids, item.ZoneID)
	}
	return ids
}

func spawnPointIDs(source []SpawnPointManifest) []string {
	ids := make([]string, 0, len(source))
	for _, item := range source {
		ids = append(ids, item.SpawnPointID)
	}
	return ids
}

func npcArchetypeIDs(source []NPCArchetypeManifest) []string {
	ids := make([]string, 0, len(source))
	for _, item := range source {
		ids = append(ids, item.ArchetypeID)
	}
	return ids
}

func abilityIDs(source []AbilitySpecManifest) []string {
	ids := make([]string, 0, len(source))
	for _, item := range source {
		ids = append(ids, item.AbilityID)
	}
	return ids
}

func lootRuleIDs(source []LootRuleManifest) []string {
	ids := make([]string, 0, len(source))
	for _, item := range source {
		ids = append(ids, item.RewardRuleID)
	}
	return ids
}

func questIDs(source []QuestGraphManifest) []string {
	ids := make([]string, 0, len(source))
	for _, item := range source {
		ids = append(ids, item.QuestID)
	}
	return ids
}

func dialogueIDs(source []DialogueManifest) []string {
	ids := make([]string, 0, len(source))
	for _, item := range source {
		ids = append(ids, item.DialogueID)
	}
	return ids
}
