package config

import "github.com/zalando/go-keyring"

const keyringService = "ElasticDefendConsole"

func SaveSecret(kind, value string) error {
	return keyring.Set(keyringService, kind, value)
}

func LoadSecret(kind string) (string, error) {
	return keyring.Get(keyringService, kind)
}

func DeleteSecret(kind string) error {
	return keyring.Delete(keyringService, kind)
}
