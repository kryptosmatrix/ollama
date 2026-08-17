//go:build !darwin || !cgo

package tts

func platformSecretStore() SecretStore {
	return &FileSecretStore{Protect: platformProtector()}
}
