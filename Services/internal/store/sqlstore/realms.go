package sqlstore

import "amandacore/services/internal/platform"

func (s *Store) UpsertRealm(realm platform.Realm) (platform.Realm, error) {
	_, err := s.db.Exec(
		`INSERT INTO ac_realms (id, display_name, region, endpoint, supported_build, online_players, online)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			display_name = excluded.display_name,
			region = excluded.region,
			endpoint = excluded.endpoint,
			supported_build = excluded.supported_build,
			online_players = excluded.online_players,
			online = excluded.online`,
		realm.ID,
		realm.DisplayName,
		realm.Region,
		realm.Endpoint,
		realm.SupportedBuild,
		realm.OnlinePlayers,
		boolToInt(realm.Online))
	return realm, err
}

func (s *Store) ListRealms() ([]platform.Realm, error) {
	rows, err := s.db.Query(
		`SELECT id, display_name, region, endpoint, supported_build, online_players, online FROM ac_realms ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var realms []platform.Realm
	for rows.Next() {
		var realm platform.Realm
		var online int
		if err := rows.Scan(
			&realm.ID,
			&realm.DisplayName,
			&realm.Region,
			&realm.Endpoint,
			&realm.SupportedBuild,
			&realm.OnlinePlayers,
			&online); err != nil {
			return nil, err
		}
		realm.Online = intToBool(online)
		realms = append(realms, realm)
	}
	return realms, rows.Err()
}

func (s *Store) GetBuildManifest() platform.BuildManifest {
	return s.buildManifest
}
