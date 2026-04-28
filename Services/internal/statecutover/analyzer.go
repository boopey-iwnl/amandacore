package statecutover

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Report struct {
	SourceFile            string         `json:"sourceFile,omitempty"`
	GeneratedAtUTC        string         `json:"generatedAtUtc"`
	DryRun                bool           `json:"dryRun"`
	WritableImportEnabled bool           `json:"writableImportEnabled"`
	Counts                Counts         `json:"counts"`
	Rows                  map[string]int `json:"rows"`
	Warnings              []string       `json:"warnings,omitempty"`
}

type Counts struct {
	Accounts               int `json:"accounts"`
	Realms                 int `json:"realms"`
	Characters             int `json:"characters"`
	Sessions               int `json:"sessions"`
	WorldJoinTickets       int `json:"worldJoinTickets"`
	ExpiredWorldTickets    int `json:"expiredWorldTickets"`
	ImportableWorldTickets int `json:"importableWorldTickets"`
	Friends                int `json:"friends"`
	Parties                int `json:"parties"`
	Guilds                 int `json:"guilds"`
	GuildInvites           int `json:"guildInvites"`
	Auctions               int `json:"auctions"`
	Mail                   int `json:"mail"`
	AuditEvents            int `json:"auditEvents"`
	SupportTickets         int `json:"supportTickets"`
	Mutes                  int `json:"mutes"`
	HousingEntitlements    int `json:"housingEntitlements"`
	HousingSpaces          int `json:"housingSpaces"`
	AccountProgress        int `json:"accountProgress"`
	InventorySlots         int `json:"inventorySlots"`
	QuestStates            int `json:"questStates"`
	ActionBarSlots         int `json:"actionBarSlots"`
}

type Options struct {
	SourcePath            string
	IncludeExpiredTickets bool
	Now                   time.Time
}

func AnalyzeFile(options Options) (Report, error) {
	payload, err := os.ReadFile(options.SourcePath)
	if err != nil {
		return Report{}, err
	}
	report, err := Analyze(payload, options)
	if err != nil {
		return Report{}, err
	}
	report.SourceFile = filepath.Base(options.SourcePath)
	return report, nil
}

func Analyze(payload []byte, options Options) (Report, error) {
	now := options.Now
	if now.IsZero() {
		now = time.Now()
	}

	var state map[string]json.RawMessage
	if err := json.Unmarshal(payload, &state); err != nil {
		return Report{}, fmt.Errorf("legacy state must be a JSON object: %w", err)
	}
	if state == nil {
		return Report{}, fmt.Errorf("legacy state must be a JSON object")
	}

	report := Report{
		GeneratedAtUTC:        now.UTC().Format(time.RFC3339),
		DryRun:                true,
		WritableImportEnabled: false,
		Rows:                  map[string]int{},
	}
	report.Counts.Accounts = countObject(state["accounts"])
	report.Counts.Realms = countObject(state["realms"])
	report.Counts.Characters = countObject(state["characters"])
	report.Counts.Sessions = countObject(state["sessions"])
	report.Counts.WorldJoinTickets = countObject(state["worldJoinTickets"])
	report.Counts.Friends = countObject(state["friends"])
	report.Counts.Parties = countObject(state["parties"])
	report.Counts.Guilds = countObject(state["guilds"])
	report.Counts.GuildInvites = countObject(state["guildInvites"])
	report.Counts.Auctions = countObject(state["auctions"])
	report.Counts.Mail = countObject(state["mail"])
	report.Counts.AuditEvents = countObject(state["auditEvents"])
	report.Counts.SupportTickets = countObject(state["supportTickets"])
	report.Counts.Mutes = countObject(state["mutes"])
	report.Counts.HousingEntitlements = countObject(state["housingEntitlements"])
	report.Counts.HousingSpaces = countObject(state["housingSpaces"])
	report.Counts.AccountProgress = countObject(state["accountProgress"])

	characters := decodeObject(state["characters"])
	for _, raw := range characters {
		var character map[string]json.RawMessage
		if err := json.Unmarshal(raw, &character); err != nil {
			report.Warnings = append(report.Warnings, "character entry could not be decoded")
			continue
		}
		report.Counts.InventorySlots += countArray(character["inventory"])
		report.Counts.QuestStates += countObject(character["quests"])
		report.Counts.ActionBarSlots += countArray(character["actionBarSlots"])
	}

	tickets := decodeObject(state["worldJoinTickets"])
	nowUnix := now.Unix()
	for _, raw := range tickets {
		var ticket struct {
			ExpiresAt  int64 `json:"expiresAt"`
			ConsumedAt int64 `json:"consumedAt"`
		}
		if err := json.Unmarshal(raw, &ticket); err != nil {
			report.Warnings = append(report.Warnings, "world join ticket entry could not be decoded")
			continue
		}
		if ticket.ExpiresAt > 0 && ticket.ExpiresAt < nowUnix {
			report.Counts.ExpiredWorldTickets++
			if options.IncludeExpiredTickets {
				report.Counts.ImportableWorldTickets++
			}
			continue
		}
		if ticket.ConsumedAt == 0 {
			report.Counts.ImportableWorldTickets++
		}
	}

	report.Rows["accounts"] = report.Counts.Accounts
	report.Rows["realms"] = report.Counts.Realms
	report.Rows["characters"] = report.Counts.Characters
	report.Rows["sessions"] = report.Counts.Sessions
	report.Rows["world_join_tickets"] = report.Counts.ImportableWorldTickets
	report.Rows["inventory_slots"] = report.Counts.InventorySlots
	report.Rows["quest_states"] = report.Counts.QuestStates
	report.Rows["action_bar_slots"] = report.Counts.ActionBarSlots
	report.Rows["friends"] = report.Counts.Friends
	report.Rows["parties"] = report.Counts.Parties
	report.Rows["guilds"] = report.Counts.Guilds
	report.Rows["guild_invites"] = report.Counts.GuildInvites
	report.Rows["auctions"] = report.Counts.Auctions
	report.Rows["mail"] = report.Counts.Mail
	report.Rows["audit_events"] = report.Counts.AuditEvents
	report.Rows["support_tickets"] = report.Counts.SupportTickets
	report.Rows["mutes"] = report.Counts.Mutes
	report.Rows["housing_entitlements"] = report.Counts.HousingEntitlements
	report.Rows["housing_spaces"] = report.Counts.HousingSpaces
	report.Rows["account_progress"] = report.Counts.AccountProgress

	if report.Counts.WorldJoinTickets > report.Counts.ImportableWorldTickets {
		report.Warnings = append(report.Warnings, "expired or consumed runtime join tickets are excluded from importable rows by default")
	}
	return report, nil
}

func countObject(raw json.RawMessage) int {
	return len(decodeObject(raw))
}

func decodeObject(raw json.RawMessage) map[string]json.RawMessage {
	if len(raw) == 0 {
		return map[string]json.RawMessage{}
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil || values == nil {
		return map[string]json.RawMessage{}
	}
	return values
}

func countArray(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}
	var values []json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return 0
	}
	return len(values)
}
