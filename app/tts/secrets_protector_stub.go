//go:build !windows

package tts

func platformProtector() Protector { return nil }
