package content

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"amandacore/services/internal/observability"
)

const (
	ContentHookNPCInteract            = "on_npc_interact"
	ContentHookQuestAccept            = "on_quest_accept"
	ContentHookQuestObjectiveProgress = "on_quest_objective_progress"
	ContentHookQuestComplete          = "on_quest_complete"
	ContentHookQuestRewardClaim       = "on_quest_reward_claim"
	ContentHookNPCDefeated            = "on_npc_defeated"
	ContentHookLootGenerated          = "on_loot_generated"
	ContentHookLootClaimed            = "on_loot_claimed"
	ContentHookVendorBuy              = "on_vendor_buy"
	ContentHookVendorSell             = "on_vendor_sell"
	ContentHookTrainerLearn           = "on_trainer_learn"
	ContentHookZoneEnter              = "on_zone_enter"
	ContentHookLandmarkEnter          = "on_landmark_enter"
)

const (
	HookActionNone                   = "none"
	HookActionEmitEvent              = "emit_event"
	HookActionProgressQuestObjective = "progress_quest_objective"
	HookActionGrantItem              = "grant_item"
	HookActionGrantCurrency          = "grant_currency"
	HookActionShowDialogue           = "show_dialogue"
	HookActionUnlockTrainerAbility   = "unlock_trainer_ability"
)

var ErrContentNotFound = errors.New("content not found")

type VendorDefinition struct {
	VendorID    string                 `json:"vendor_id"`
	DisplayName string                 `json:"display_name"`
	NPCID       string                 `json:"npc_id,omitempty"`
	Items       []VendorItemDefinition `json:"items"`
	Tags        []string               `json:"tags"`
}

type VendorItemDefinition struct {
	ItemID      string   `json:"item_id"`
	PriceCopper int      `json:"price_copper"`
	StackLimit  int      `json:"stack_limit"`
	Available   bool     `json:"available"`
	Tags        []string `json:"tags"`
}

type TrainerDefinition struct {
	TrainerID   string                     `json:"trainer_id"`
	DisplayName string                     `json:"display_name"`
	NPCID       string                     `json:"npc_id,omitempty"`
	Abilities   []TrainerAbilityDefinition `json:"abilities"`
	Tags        []string                   `json:"tags"`
}

type TrainerAbilityDefinition struct {
	AbilityID     string   `json:"ability_id"`
	CostCopper    int      `json:"cost_copper"`
	RequiredLevel int      `json:"required_level"`
	Tags          []string `json:"tags"`
}

type DialogueDefinition struct {
	DialogueID string          `json:"dialogue_id"`
	SpeakerID  string          `json:"speaker_id,omitempty"`
	Entries    []DialogueEntry `json:"entries"`
	Tags       []string        `json:"tags"`
}

type DialogueEntry struct {
	EntryID        string   `json:"entry_id"`
	Text           string   `json:"text"`
	NextEntryIDs   []string `json:"next_entry_ids,omitempty"`
	HookBindingIDs []string `json:"hook_binding_ids,omitempty"`
	Tags           []string `json:"tags"`
}

type HookBindingDefinition struct {
	BindingID string                 `json:"binding_id"`
	Hook      string                 `json:"hook"`
	SourceID  string                 `json:"source_id"`
	Priority  int                    `json:"priority"`
	Enabled   bool                   `json:"enabled"`
	Actions   []HookActionDefinition `json:"actions"`
	Tags      []string               `json:"tags"`
}

type HookActionDefinition struct {
	Action    string `json:"action"`
	TargetID  string `json:"target_id,omitempty"`
	Quantity  int    `json:"quantity,omitempty"`
	EventName string `json:"event_name,omitempty"`
	Message   string `json:"message,omitempty"`
}

type ContentLookupError struct {
	Kind string
	ID   string
}

func (e ContentLookupError) Error() string {
	return fmt.Sprintf("%s %q was not found in the loaded content registry", e.Kind, e.ID)
}

func (e ContentLookupError) Is(target error) bool {
	return target == ErrContentNotFound
}

type ContentPackageBuildInfo struct {
	PackageID string `json:"package_id"`
	Version   string `json:"version"`
}

type ContentRegistry interface {
	PackageBuildInfo() ContentPackageBuildInfo
	QuestCatalog
	NpcCatalog
	LootCatalog
	AbilityCatalog
	VendorCatalog
	TrainerCatalog
	ZoneCatalog
}

type QuestCatalog interface {
	QuestByID(questID string) (QuestDefinition, error)
}

type NpcCatalog interface {
	NPCByID(npcID string) (NpcArchetype, error)
}

type LootCatalog interface {
	LootTableByID(lootTableID string) (LootTableDefinition, error)
}

type AbilityCatalog interface {
	AbilityByID(abilityID string) (AbilityDefinition, error)
}

type VendorCatalog interface {
	VendorByID(vendorID string) (VendorDefinition, error)
}

type TrainerCatalog interface {
	TrainerByID(trainerID string) (TrainerDefinition, error)
}

type ZoneCatalog interface {
	ZoneByID(zoneID string) (ZoneDefinition, error)
}

func (r RuntimeContentRegistry) PackageBuildInfo() ContentPackageBuildInfo {
	return ContentPackageBuildInfo{
		PackageID: r.PackageID,
		Version:   r.Version,
	}
}

func (r RuntimeContentRegistry) QuestByID(questID string) (QuestDefinition, error) {
	if quest, found := r.Quests[questID]; found {
		return cloneQuestDefinition(quest), nil
	}
	return QuestDefinition{}, ContentLookupError{Kind: "quest", ID: questID}
}

func (r RuntimeContentRegistry) NPCByID(npcID string) (NpcArchetype, error) {
	if npc, found := r.NPCs[npcID]; found {
		return cloneNPCArchetype(npc), nil
	}
	return NpcArchetype{}, ContentLookupError{Kind: "npc", ID: npcID}
}

func (r RuntimeContentRegistry) LootTableByID(lootTableID string) (LootTableDefinition, error) {
	if loot, found := r.LootTables[lootTableID]; found {
		return cloneLootTableDefinition(loot), nil
	}
	return LootTableDefinition{}, ContentLookupError{Kind: "loot_table", ID: lootTableID}
}

func (r RuntimeContentRegistry) AbilityByID(abilityID string) (AbilityDefinition, error) {
	if ability, found := r.Abilities[abilityID]; found {
		return cloneAbilityDefinition(ability), nil
	}
	return AbilityDefinition{}, ContentLookupError{Kind: "ability", ID: abilityID}
}

func (r RuntimeContentRegistry) VendorByID(vendorID string) (VendorDefinition, error) {
	if vendor, found := r.Vendors[vendorID]; found {
		return cloneVendorDefinition(vendor), nil
	}
	return VendorDefinition{}, ContentLookupError{Kind: "vendor", ID: vendorID}
}

func (r RuntimeContentRegistry) TrainerByID(trainerID string) (TrainerDefinition, error) {
	if trainer, found := r.Trainers[trainerID]; found {
		return cloneTrainerDefinition(trainer), nil
	}
	return TrainerDefinition{}, ContentLookupError{Kind: "trainer", ID: trainerID}
}

func (r RuntimeContentRegistry) ZoneByID(zoneID string) (ZoneDefinition, error) {
	if zone, found := r.Zones[zoneID]; found {
		return cloneZoneDefinition(zone), nil
	}
	return ZoneDefinition{}, ContentLookupError{Kind: "zone", ID: zoneID}
}

func AllowedContentHookNames() []string {
	return []string{
		ContentHookNPCInteract,
		ContentHookQuestAccept,
		ContentHookQuestObjectiveProgress,
		ContentHookQuestComplete,
		ContentHookQuestRewardClaim,
		ContentHookNPCDefeated,
		ContentHookLootGenerated,
		ContentHookLootClaimed,
		ContentHookVendorBuy,
		ContentHookVendorSell,
		ContentHookTrainerLearn,
		ContentHookZoneEnter,
		ContentHookLandmarkEnter,
	}
}

func IsAllowedContentHook(name string) bool {
	normalized := strings.TrimSpace(name)
	for _, hook := range AllowedContentHookNames() {
		if normalized == hook {
			return true
		}
	}
	return false
}

type ContentHookPayload struct {
	Hook      string         `json:"hook"`
	BindingID string         `json:"binding_id,omitempty"`
	ActorID   string         `json:"actor_id,omitempty"`
	SourceID  string         `json:"source_id,omitempty"`
	Fields    map[string]any `json:"fields,omitempty"`
}

type ContentHookResult struct {
	BindingID string   `json:"binding_id"`
	Hook      string   `json:"hook"`
	Handled   bool     `json:"handled"`
	Events    []string `json:"events,omitempty"`
}

type ContentHookHandler interface {
	HandleContentHook(ContentHookPayload) (ContentHookResult, error)
}

type NoopContentHookHandler struct{}

func (NoopContentHookHandler) HandleContentHook(payload ContentHookPayload) (ContentHookResult, error) {
	return ContentHookResult{
		BindingID: payload.BindingID,
		Hook:      payload.Hook,
		Handled:   true,
	}, nil
}

type ContentHookRuntime struct {
	bindingsByHook map[string][]HookBindingDefinition
	handler        ContentHookHandler
}

func NewContentHookRuntime(bindings []HookBindingDefinition, handler ContentHookHandler) (*ContentHookRuntime, error) {
	if handler == nil {
		handler = NoopContentHookHandler{}
	}
	runtime := &ContentHookRuntime{
		bindingsByHook: map[string][]HookBindingDefinition{},
		handler:        handler,
	}
	for _, binding := range bindings {
		if !binding.Enabled {
			continue
		}
		if !IsAllowedContentHook(binding.Hook) {
			observability.LogEvent("content-runtime", observability.EventContentHookRejected, map[string]any{
				"bindingId": binding.BindingID,
				"hook":      binding.Hook,
			})
			return nil, fmt.Errorf("hook binding %q uses unsupported hook %q", binding.BindingID, binding.Hook)
		}
		runtime.bindingsByHook[binding.Hook] = append(runtime.bindingsByHook[binding.Hook], binding)
	}
	for hook := range runtime.bindingsByHook {
		sortHookBindings(runtime.bindingsByHook[hook])
	}
	return runtime, nil
}

func (r *ContentHookRuntime) Invoke(hook string, payload ContentHookPayload) ([]ContentHookResult, error) {
	if r == nil {
		return nil, fmt.Errorf("content hook runtime is not initialized")
	}
	if !IsAllowedContentHook(hook) {
		observability.LogEvent("content-runtime", observability.EventContentHookRejected, map[string]any{
			"hook": hook,
		})
		return nil, fmt.Errorf("unsupported content hook %q", hook)
	}
	bindings := r.bindingsByHook[hook]
	results := make([]ContentHookResult, 0, len(bindings))
	for _, binding := range bindings {
		nextPayload := payload
		nextPayload.Hook = hook
		nextPayload.BindingID = binding.BindingID
		nextPayload.SourceID = binding.SourceID
		result, err := r.handler.HandleContentHook(nextPayload)
		if err != nil {
			return results, err
		}
		if result.BindingID == "" {
			result.BindingID = binding.BindingID
		}
		if result.Hook == "" {
			result.Hook = hook
		}
		results = append(results, result)
		observability.LogEvent("content-runtime", observability.EventContentHookInvoked, map[string]any{
			"bindingId": binding.BindingID,
			"hook":      hook,
			"sourceId":  binding.SourceID,
		})
	}
	return results, nil
}

func validateVendor(vendor VendorDefinition, index int, itemIDs map[string]struct{}, report *ContentValidationReport) {
	path := fmt.Sprintf("vendors[%d]", index)
	requiredID(report, path+".vendor_id", vendor.VendorID)
	requiredString(report, path+".display_name", vendor.DisplayName)
	if vendor.NPCID != "" {
		requiredID(report, path+".npc_id", vendor.NPCID)
	}
	if len(vendor.Items) == 0 {
		report.Add(ErrorMissingRequiredField, path+".items", "vendor must define at least one item")
	}
	itemIDsInVendor := map[string]struct{}{}
	for itemIndex, item := range vendor.Items {
		itemPath := fmt.Sprintf("%s.items[%d]", path, itemIndex)
		requiredID(report, itemPath+".item_id", item.ItemID)
		if item.ItemID != "" {
			if _, exists := itemIDsInVendor[item.ItemID]; exists {
				report.Addf(ErrorDuplicateID, itemPath+".item_id", "vendor %q lists item %q more than once", vendor.VendorID, item.ItemID)
			}
			itemIDsInVendor[item.ItemID] = struct{}{}
			if !containsID(itemIDs, item.ItemID) {
				report.Addf(ErrorBrokenReference, itemPath+".item_id", "vendor %q references missing item %q", vendor.VendorID, item.ItemID)
				logBrokenReference("vendor", vendor.VendorID, "item", item.ItemID)
			}
		}
		if item.PriceCopper < 0 {
			report.Add(ErrorInvalidNumberRange, itemPath+".price_copper", "vendor item price must be non-negative")
		}
		if item.StackLimit < 0 {
			report.Add(ErrorInvalidNumberRange, itemPath+".stack_limit", "vendor item stack limit must be non-negative")
		}
	}
}

func validateTrainer(trainer TrainerDefinition, index int, abilityIDs map[string]struct{}, report *ContentValidationReport) {
	path := fmt.Sprintf("trainers[%d]", index)
	requiredID(report, path+".trainer_id", trainer.TrainerID)
	requiredString(report, path+".display_name", trainer.DisplayName)
	if trainer.NPCID != "" {
		requiredID(report, path+".npc_id", trainer.NPCID)
	}
	if len(trainer.Abilities) == 0 {
		report.Add(ErrorMissingRequiredField, path+".abilities", "trainer must define at least one ability")
	}
	abilitiesInTrainer := map[string]struct{}{}
	for abilityIndex, ability := range trainer.Abilities {
		abilityPath := fmt.Sprintf("%s.abilities[%d]", path, abilityIndex)
		requiredID(report, abilityPath+".ability_id", ability.AbilityID)
		if ability.AbilityID != "" {
			if _, exists := abilitiesInTrainer[ability.AbilityID]; exists {
				report.Addf(ErrorDuplicateID, abilityPath+".ability_id", "trainer %q lists ability %q more than once", trainer.TrainerID, ability.AbilityID)
			}
			abilitiesInTrainer[ability.AbilityID] = struct{}{}
			if !containsID(abilityIDs, ability.AbilityID) {
				report.Addf(ErrorBrokenReference, abilityPath+".ability_id", "trainer %q references missing ability %q", trainer.TrainerID, ability.AbilityID)
				logBrokenReference("trainer", trainer.TrainerID, "ability", ability.AbilityID)
			}
		}
		if ability.CostCopper < 0 {
			report.Add(ErrorInvalidNumberRange, abilityPath+".cost_copper", "trainer ability cost must be non-negative")
		}
		if ability.RequiredLevel < 0 {
			report.Add(ErrorInvalidNumberRange, abilityPath+".required_level", "trainer ability required level must be non-negative")
		}
	}
}

func validateDialogue(dialogue DialogueDefinition, index int, npcIDs map[string]struct{}, providerIDs map[string]struct{}, hookBindingIDs map[string]struct{}, report *ContentValidationReport) {
	path := fmt.Sprintf("dialogues[%d]", index)
	requiredID(report, path+".dialogue_id", dialogue.DialogueID)
	if dialogue.SpeakerID != "" {
		requiredID(report, path+".speaker_id", dialogue.SpeakerID)
		if !containsID(npcIDs, dialogue.SpeakerID) && !containsID(providerIDs, dialogue.SpeakerID) {
			report.Addf(ErrorBrokenReference, path+".speaker_id", "dialogue %q references missing speaker %q", dialogue.DialogueID, dialogue.SpeakerID)
			logBrokenReference("dialogue", dialogue.DialogueID, "speaker", dialogue.SpeakerID)
		}
	}
	if len(dialogue.Entries) == 0 {
		report.Add(ErrorMissingRequiredField, path+".entries", "dialogue must define at least one entry")
	}
	entryIDs := map[string]struct{}{}
	for entryIndex, entry := range dialogue.Entries {
		entryPath := fmt.Sprintf("%s.entries[%d]", path, entryIndex)
		requiredID(report, entryPath+".entry_id", entry.EntryID)
		requiredString(report, entryPath+".text", entry.Text)
		if entry.EntryID != "" {
			if _, exists := entryIDs[entry.EntryID]; exists {
				report.Addf(ErrorDuplicateID, entryPath+".entry_id", "dialogue entry %q is duplicated in dialogue %q", entry.EntryID, dialogue.DialogueID)
			}
			entryIDs[entry.EntryID] = struct{}{}
		}
		for hookIndex, hookBindingID := range entry.HookBindingIDs {
			hookPath := fmt.Sprintf("%s.hook_binding_ids[%d]", entryPath, hookIndex)
			requiredID(report, hookPath, hookBindingID)
			if hookBindingID != "" && !containsID(hookBindingIDs, hookBindingID) {
				report.Addf(ErrorBrokenReference, hookPath, "dialogue %q references missing hook binding %q", dialogue.DialogueID, hookBindingID)
			}
		}
	}
	for entryIndex, entry := range dialogue.Entries {
		entryPath := fmt.Sprintf("%s.entries[%d]", path, entryIndex)
		for nextIndex, nextEntryID := range entry.NextEntryIDs {
			nextPath := fmt.Sprintf("%s.next_entry_ids[%d]", entryPath, nextIndex)
			requiredID(report, nextPath, nextEntryID)
			if nextEntryID != "" && !containsID(entryIDs, nextEntryID) {
				report.Addf(ErrorBrokenReference, nextPath, "dialogue entry %q references missing next entry %q", entry.EntryID, nextEntryID)
			}
		}
	}
}

func validateHookBinding(binding HookBindingDefinition, index int, npcIDs map[string]struct{}, providerIDs map[string]struct{}, questIDs map[string]struct{}, itemIDs map[string]struct{}, lootIDs map[string]struct{}, abilityIDs map[string]struct{}, vendorIDs map[string]struct{}, trainerIDs map[string]struct{}, dialogueIDs map[string]struct{}, report *ContentValidationReport) {
	path := fmt.Sprintf("hook_bindings[%d]", index)
	requiredID(report, path+".binding_id", binding.BindingID)
	requiredString(report, path+".hook", binding.Hook)
	if binding.Hook != "" && !IsAllowedContentHook(binding.Hook) {
		report.Addf(ErrorInvalidEnum, path+".hook", "hook %q is not supported by AmandaCore content runtime", binding.Hook)
		observability.LogEvent("content-loader", observability.EventContentHookRejected, map[string]any{
			"bindingId": binding.BindingID,
			"hook":      binding.Hook,
		})
	}
	requiredID(report, path+".source_id", binding.SourceID)
	if binding.SourceID != "" && !contentSourceKnown(binding.SourceID, npcIDs, providerIDs, questIDs, itemIDs, lootIDs, abilityIDs, vendorIDs, trainerIDs, dialogueIDs) {
		report.Addf(ErrorBrokenReference, path+".source_id", "hook binding %q references missing source %q", binding.BindingID, binding.SourceID)
		logBrokenReference("hook_binding", binding.BindingID, "content_source", binding.SourceID)
	}
	if len(binding.Actions) == 0 {
		report.Add(ErrorMissingRequiredField, path+".actions", "hook binding must define at least one declarative action")
	}
	for actionIndex, action := range binding.Actions {
		validateHookAction(binding, action, fmt.Sprintf("%s.actions[%d]", path, actionIndex), questIDs, itemIDs, abilityIDs, dialogueIDs, report)
	}
}

func validateHookAction(binding HookBindingDefinition, action HookActionDefinition, path string, questIDs map[string]struct{}, itemIDs map[string]struct{}, abilityIDs map[string]struct{}, dialogueIDs map[string]struct{}, report *ContentValidationReport) {
	requiredString(report, path+".action", action.Action)
	switch action.Action {
	case HookActionNone:
		return
	case HookActionEmitEvent:
		requiredID(report, path+".event_name", action.EventName)
	case HookActionProgressQuestObjective:
		requiredID(report, path+".target_id", action.TargetID)
		if action.TargetID != "" && !containsID(questIDs, action.TargetID) {
			report.Addf(ErrorBrokenReference, path+".target_id", "hook binding %q references missing quest %q", binding.BindingID, action.TargetID)
		}
	case HookActionGrantItem:
		requiredID(report, path+".target_id", action.TargetID)
		if action.TargetID != "" && !containsID(itemIDs, action.TargetID) {
			report.Addf(ErrorBrokenReference, path+".target_id", "hook binding %q references missing item %q", binding.BindingID, action.TargetID)
		}
		if action.Quantity <= 0 {
			report.Add(ErrorInvalidNumberRange, path+".quantity", "grant_item quantity must be positive")
		}
	case HookActionGrantCurrency:
		if action.Quantity <= 0 {
			report.Add(ErrorInvalidNumberRange, path+".quantity", "grant_currency quantity must be positive")
		}
	case HookActionShowDialogue:
		requiredID(report, path+".target_id", action.TargetID)
		if action.TargetID != "" && !containsID(dialogueIDs, action.TargetID) {
			report.Addf(ErrorBrokenReference, path+".target_id", "hook binding %q references missing dialogue %q", binding.BindingID, action.TargetID)
		}
	case HookActionUnlockTrainerAbility:
		requiredID(report, path+".target_id", action.TargetID)
		if action.TargetID != "" && !containsID(abilityIDs, action.TargetID) {
			report.Addf(ErrorBrokenReference, path+".target_id", "hook binding %q references missing ability %q", binding.BindingID, action.TargetID)
		}
	default:
		report.Addf(ErrorInvalidEnum, path+".action", "declarative hook action %q is not supported", action.Action)
	}
}

func contentSourceKnown(sourceID string, sources ...map[string]struct{}) bool {
	for _, source := range sources {
		if containsID(source, sourceID) {
			return true
		}
	}
	return false
}

func sortHookBindings(bindings []HookBindingDefinition) {
	sort.SliceStable(bindings, func(i int, j int) bool {
		if bindings[i].Priority != bindings[j].Priority {
			return bindings[i].Priority < bindings[j].Priority
		}
		return bindings[i].BindingID < bindings[j].BindingID
	})
}

func cloneQuestDefinition(quest QuestDefinition) QuestDefinition {
	quest.PrerequisiteQuestIDs = append([]string(nil), quest.PrerequisiteQuestIDs...)
	quest.Rewards = append([]QuestReward(nil), quest.Rewards...)
	quest.Tags = append([]string(nil), quest.Tags...)
	quest.ObjectiveGraph.Nodes = append([]QuestObjectiveNode(nil), quest.ObjectiveGraph.Nodes...)
	for index := range quest.ObjectiveGraph.Nodes {
		quest.ObjectiveGraph.Nodes[index].DependsOn = append([]string(nil), quest.ObjectiveGraph.Nodes[index].DependsOn...)
	}
	return quest
}

func cloneNPCArchetype(npc NpcArchetype) NpcArchetype {
	npc.DefaultAbilityIDs = append([]string(nil), npc.DefaultAbilityIDs...)
	npc.Tags = append([]string(nil), npc.Tags...)
	return npc
}

func cloneLootTableDefinition(loot LootTableDefinition) LootTableDefinition {
	loot.Entries = append([]LootTableEntry(nil), loot.Entries...)
	for index := range loot.Entries {
		loot.Entries[index].Tags = append([]string(nil), loot.Entries[index].Tags...)
	}
	loot.Tags = append([]string(nil), loot.Tags...)
	return loot
}

func cloneAbilityDefinition(ability AbilityDefinition) AbilityDefinition {
	ability.Effects = append([]AbilityEffect(nil), ability.Effects...)
	ability.Tags = append([]string(nil), ability.Tags...)
	return ability
}

func cloneVendorDefinition(vendor VendorDefinition) VendorDefinition {
	vendor.Items = append([]VendorItemDefinition(nil), vendor.Items...)
	for index := range vendor.Items {
		vendor.Items[index].Tags = append([]string(nil), vendor.Items[index].Tags...)
	}
	vendor.Tags = append([]string(nil), vendor.Tags...)
	return vendor
}

func cloneTrainerDefinition(trainer TrainerDefinition) TrainerDefinition {
	trainer.Abilities = append([]TrainerAbilityDefinition(nil), trainer.Abilities...)
	for index := range trainer.Abilities {
		trainer.Abilities[index].Tags = append([]string(nil), trainer.Abilities[index].Tags...)
	}
	trainer.Tags = append([]string(nil), trainer.Tags...)
	return trainer
}

func cloneZoneDefinition(zone ZoneDefinition) ZoneDefinition {
	zone.EntryPoints = append([]ZoneEntryPoint(nil), zone.EntryPoints...)
	zone.SpawnPoints = append([]ZoneSpawnPointDefinition(nil), zone.SpawnPoints...)
	for index := range zone.SpawnPoints {
		zone.SpawnPoints[index].Tags = append([]string(nil), zone.SpawnPoints[index].Tags...)
	}
	zone.SpawnGroups = append([]SpawnGroupDefinition(nil), zone.SpawnGroups...)
	for index := range zone.SpawnGroups {
		zone.SpawnGroups[index].SpawnPointIDs = append([]string(nil), zone.SpawnGroups[index].SpawnPointIDs...)
		zone.SpawnGroups[index].SpawnPoints = append([]SpawnPointDefinition(nil), zone.SpawnGroups[index].SpawnPoints...)
		zone.SpawnGroups[index].Tags = append([]string(nil), zone.SpawnGroups[index].Tags...)
	}
	zone.HandoffGates = append([]HandoffGateDefinition(nil), zone.HandoffGates...)
	for index := range zone.HandoffGates {
		zone.HandoffGates[index].Tags = append([]string(nil), zone.HandoffGates[index].Tags...)
	}
	zone.QuestProviders = append([]QuestProviderDefinition(nil), zone.QuestProviders...)
	for index := range zone.QuestProviders {
		zone.QuestProviders[index].OfferedQuestIDs = append([]string(nil), zone.QuestProviders[index].OfferedQuestIDs...)
		zone.QuestProviders[index].Tags = append([]string(nil), zone.QuestProviders[index].Tags...)
	}
	zone.TransitionGates = append([]ZoneTransitionGate(nil), zone.TransitionGates...)
	for index := range zone.TransitionGates {
		zone.TransitionGates[index].Tags = append([]string(nil), zone.TransitionGates[index].Tags...)
	}
	zone.Transitions = append([]ZoneTransitionDefinition(nil), zone.Transitions...)
	for index := range zone.Transitions {
		zone.Transitions[index].Tags = append([]string(nil), zone.Transitions[index].Tags...)
	}
	zone.Tags = append([]string(nil), zone.Tags...)
	return zone
}
