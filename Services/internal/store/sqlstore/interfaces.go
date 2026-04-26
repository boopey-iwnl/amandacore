package sqlstore

import filestore "amandacore/services/internal/store"

var (
	_ filestore.AccountRepository      = (*Store)(nil)
	_ filestore.SessionRepository      = (*Store)(nil)
	_ filestore.RealmRepository        = (*Store)(nil)
	_ filestore.CharacterRepository    = (*Store)(nil)
	_ filestore.ProgressionRepository  = (*Store)(nil)
	_ filestore.InventoryRepository    = (*Store)(nil)
	_ filestore.QuestRepository        = (*Store)(nil)
	_ filestore.AbilityRepository      = (*Store)(nil)
	_ filestore.ActionBarRepository    = (*Store)(nil)
	_ filestore.WorldTicketRepository  = (*Store)(nil)
	_ filestore.WorldSessionRepository = (*Store)(nil)
	_ filestore.AuditEventRepository   = (*Store)(nil)
)
