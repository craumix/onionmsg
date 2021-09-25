package types_test

import (
	"crypto/ed25519"
	. "github.com/craumix/onionmsg/internal/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewIdentity(t *testing.T) {
	testcases := []struct {
		name string

		iType       IdentityType
		fingerprint string

		expectPrivKey bool
		expectPubKey  bool
		expectMeta    bool
	}{
		{
			name:          "Test Self",
			iType:         Self,
			fingerprint:   "",
			expectPrivKey: true,
			expectPubKey:  true,
			expectMeta:    true,
		},
		{
			name:          "Test Contact",
			iType:         Contact,
			fingerprint:   "",
			expectPrivKey: true,
			expectPubKey:  true,
			expectMeta:    false,
		},
		{
			name:          "Test Remote",
			iType:         Remote,
			fingerprint:   "test-fingerprint",
			expectPrivKey: false,
			expectPubKey:  true,
			expectMeta:    true,
		},
	}

	for _, tc := range testcases {
		actualID, err := NewIdentity(tc.iType, tc.fingerprint)

		assert.NoError(t, err, tc.name+": NewIdentity error")

		if tc.expectPrivKey {
			assert.NotNil(t, actualID.Priv, tc.name+": PrivKey missing")
		} else {
			assert.Nil(t, actualID.Priv, tc.name+": PrivKey missing")
		}

		if tc.expectPubKey {
			assert.NotNil(t, actualID.Pub, tc.name+": PubKey missing")
		} else {
			assert.Nil(t, actualID.Pub, tc.name+": PubKey found")
		}

		if tc.expectMeta {
			assert.NotNil(t, actualID.Meta, tc.name+": Meta missing")
			assert.False(t, actualID.Meta.Admin, tc.name+": is admin")
			assert.Equal(t, "", actualID.Meta.Nick, tc.name+": has nickname")
		} else {
			assert.Nil(t, actualID.Meta, tc.name+": Meta found")
		}
	}
}

func TestSign(t *testing.T) {
	id, _ := NewIdentity(Self, "")
	data := []byte("test-signing")

	sig, err := id.Sign(data)
	assert.NoError(t, err, "Sign threw an error")

	success := ed25519.Verify(*id.Pub, data, sig)

	assert.True(t, success, "Verify failed")
}

func TestVerify(t *testing.T) {
	id, _ := NewIdentity(Self, "")
	data := []byte("test-signing")

	sig := ed25519.Sign(*id.Priv, data)

	success, err := id.Verify(data, sig)
	assert.NoError(t, err, "Verify threw an error")

	assert.True(t, success, "Verify failed")
}

func TestSignKeyMissing(t *testing.T) {
	id := Identity{}

	sign, err := id.Sign([]byte("data"))

	assert.Error(t, err)
	assert.Nil(t, sign)
}

func TestVerifyKeyMissing(t *testing.T) {
	id := Identity{}

	success, err := id.Verify([]byte("data"), []byte("data"))

	assert.Error(t, err)
	assert.False(t, success)
}

func TestEmpty(t *testing.T) {
	id := Identity{}

	assert.Empty(t, id.Admin())
	assert.Empty(t, id.Nick())
	assert.Empty(t, id.Fingerprint())
	assert.Empty(t, id.ServiceID())
}
