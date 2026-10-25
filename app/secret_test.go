package app

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestSecretKeysAndSizesAreValidatedBeforeAnyStoreIsAsked(t *testing.T) {
	long := strings.Repeat("x", 257)
	for _, test := range []struct {
		name             string
		service, account string
		secret           []byte
	}{
		{"empty service", "", "a", []byte("s")},
		{"empty account", "s", "", []byte("s")},
		{"NUL in service", "s\x00", "a", []byte("s")},
		{"long account", "s", long, []byte("s")},
		{"empty secret", "s", "a", nil},
		{"oversized secret", "s", "a", make([]byte, MaxSecretSize+1)},
	} {
		if err := StoreSecret(test.service, test.account, test.secret); err == nil || errors.Is(err, ErrSecretStoreUnavailable) {
			t.Errorf("%s: StoreSecret = %v, want a validation error", test.name, err)
		}
	}
	if _, err := LoadSecret("", "a"); err == nil {
		t.Error("LoadSecret accepted an empty service")
	}
	if err := DeleteSecret("s", ""); err == nil {
		t.Error("DeleteSecret accepted an empty account")
	}
}

// VIVA_SECRET_TEST=1 runs this against the real store of the machine: the
// Keychain, the Credential Manager or the Secret Service. It writes one entry of
// its own and removes it again.
func TestSystemSecretStoreRoundTrip(t *testing.T) {
	if os.Getenv("VIVA_SECRET_TEST") != "1" {
		t.Skip("set VIVA_SECRET_TEST=1 to use the real secret store")
	}
	service, account := "viva-secret-test", "round-trip-"+t.Name()
	t.Cleanup(func() { _ = DeleteSecret(service, account) })

	if _, err := LoadSecret(service, account); !errors.Is(err, ErrSecretNotFound) {
		if errors.Is(err, ErrSecretStoreUnavailable) {
			t.Skipf("this machine has no secret store: %v", err)
		}
		t.Fatalf("LoadSecret of nothing = %v, want ErrSecretNotFound", err)
	}
	first := []byte{0, 1, 2, 'k', 'e', 'y', 255}
	if err := StoreSecret(service, account, first); err != nil {
		t.Fatal(err)
	}
	got, err := LoadSecret(service, account)
	if err != nil || !bytes.Equal(got, first) {
		t.Fatalf("LoadSecret = %v, %v; want %v", got, err, first)
	}
	second := bytes.Repeat([]byte("z"), MaxSecretSize)
	if err := StoreSecret(service, account, second); err != nil {
		t.Fatalf("replacing a secret: %v", err)
	}
	if got, err := LoadSecret(service, account); err != nil || !bytes.Equal(got, second) {
		t.Fatalf("after replace: len %d, err %v", len(got), err)
	}
	if err := DeleteSecret(service, account); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSecret(service, account); !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("after delete = %v, want ErrSecretNotFound", err)
	}
	if err := DeleteSecret(service, account); err != nil {
		t.Fatalf("deleting what is not there: %v", err)
	}
}
