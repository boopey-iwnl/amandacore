package sqlstore

import "amandacore/services/internal/platform"

func DevRealm(endpoint string, supportedBuild string) platform.Realm {
	if endpoint == "" {
		endpoint = "http://127.0.0.1:8085"
	}
	if supportedBuild == "" {
		supportedBuild = "amandacore-sqlstore-test"
	}
	return platform.Realm{
		ID:             "sunset-frontier-dev",
		DisplayName:    "Sunset Frontier Dev",
		Region:         "local",
		Endpoint:       endpoint,
		SupportedBuild: supportedBuild,
		OnlinePlayers:  0,
		Online:         true,
	}
}

func SeedDevRealm(store *Store) (platform.Realm, error) {
	return store.UpsertRealm(DevRealm("", ""))
}

func SeedTestAccount(store *Store, username string, password string) (platform.Account, error) {
	if username == "" {
		username = "sqlstore_test_player"
	}
	if password == "" {
		password = "test_password"
	}
	return store.RegisterAccount(username, password)
}
