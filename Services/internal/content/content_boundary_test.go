package content

import (
	"errors"
	"reflect"
	"testing"
)

func TestRuntimeContentRegistryCatalogLookups(t *testing.T) {
	loaded := mustLoadDevPackage(t)
	registry := NewRuntimeContentRegistry(loaded)

	if _, err := registry.QuestByID("dev_first_hunt"); err != nil {
		t.Fatalf("quest lookup failed: %v", err)
	}
	if _, err := registry.NPCByID("dev_isle_stalker"); err != nil {
		t.Fatalf("npc lookup failed: %v", err)
	}
	if _, err := registry.LootTableByID("dev_isle_stalker_cache"); err != nil {
		t.Fatalf("loot lookup failed: %v", err)
	}
	if _, err := registry.AbilityByID("dev_pathfinder_focus"); err != nil {
		t.Fatalf("ability lookup failed: %v", err)
	}
	if _, err := registry.VendorByID("vendor_dev_pathfinder_cache"); err != nil {
		t.Fatalf("vendor lookup failed: %v", err)
	}
	if _, err := registry.TrainerByID("trainer_dev_pathfinder"); err != nil {
		t.Fatalf("trainer lookup failed: %v", err)
	}
	if _, err := registry.ZoneByID("dev_isle_edge"); err != nil {
		t.Fatalf("zone lookup failed: %v", err)
	}
	if _, err := registry.QuestByID("missing_quest"); !errors.Is(err, ErrContentNotFound) {
		t.Fatalf("expected missing content error, got %v", err)
	}
}

func TestRuntimeContentRegistryReturnsStableValues(t *testing.T) {
	loaded := mustLoadDevPackage(t)
	registry := NewRuntimeContentRegistry(loaded)

	first, err := registry.VendorByID("vendor_dev_pathfinder_cache")
	if err != nil {
		t.Fatalf("vendor lookup failed: %v", err)
	}
	first.Items[0].ItemID = "mutated_locally"
	second, err := registry.VendorByID("vendor_dev_pathfinder_cache")
	if err != nil {
		t.Fatalf("vendor lookup failed: %v", err)
	}
	if second.Items[0].ItemID != "dev_field_ration" {
		t.Fatalf("registry value was mutated through returned copy: %#v", second.Items[0])
	}
}

func TestAllowedContentHookNames(t *testing.T) {
	if !IsAllowedContentHook(ContentHookQuestAccept) {
		t.Fatalf("expected quest accept hook to be allowed")
	}
	if IsAllowedContentHook("on_unsafe_script") {
		t.Fatalf("unsafe hook name should not be allowed")
	}
}

func TestContentHookRuntimeInvokesBindingsDeterministically(t *testing.T) {
	handler := &recordingHookHandler{}
	runtime, err := NewContentHookRuntime([]HookBindingDefinition{
		{
			BindingID: "hook_second",
			Hook:      ContentHookQuestAccept,
			SourceID:  "dev_first_hunt",
			Priority:  20,
			Enabled:   true,
			Actions: []HookActionDefinition{
				{Action: HookActionNone},
			},
		},
		{
			BindingID: "hook_first",
			Hook:      ContentHookQuestAccept,
			SourceID:  "dev_first_hunt",
			Priority:  10,
			Enabled:   true,
			Actions: []HookActionDefinition{
				{Action: HookActionNone},
			},
		},
	}, handler)
	if err != nil {
		t.Fatalf("create hook runtime: %v", err)
	}

	results, err := runtime.Invoke(ContentHookQuestAccept, ContentHookPayload{ActorID: "character_test"})
	if err != nil {
		t.Fatalf("invoke hook: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected two hook results, got %d", len(results))
	}
	expected := []string{"hook_first", "hook_second"}
	if !reflect.DeepEqual(handler.bindingIDs, expected) {
		t.Fatalf("unexpected invocation order: %#v", handler.bindingIDs)
	}
}

func TestContentHookRuntimeRejectsUnsupportedHook(t *testing.T) {
	_, err := NewContentHookRuntime([]HookBindingDefinition{
		{
			BindingID: "hook_bad",
			Hook:      "on_unsafe_script",
			SourceID:  "dev_first_hunt",
			Enabled:   true,
			Actions: []HookActionDefinition{
				{Action: HookActionNone},
			},
		},
	}, NoopContentHookHandler{})
	if err == nil {
		t.Fatalf("expected unsupported hook to fail")
	}
}

type recordingHookHandler struct {
	bindingIDs []string
}

func (h *recordingHookHandler) HandleContentHook(payload ContentHookPayload) (ContentHookResult, error) {
	h.bindingIDs = append(h.bindingIDs, payload.BindingID)
	return ContentHookResult{
		BindingID: payload.BindingID,
		Hook:      payload.Hook,
		Handled:   true,
	}, nil
}
