//go:build windows

package platform

import "testing"

func TestOpenWithResultReportsOnlyFailures(t *testing.T) {
	for _, code := range []uint32{0, 1, 0x800704C7} {
		if err := openWithResult(code); err != nil {
			t.Fatalf("normal return or dismissal treated as failure: %x", code)
		}
	}
	if err := openWithResult(0x80070005); err == nil {
		t.Fatal("failed HRESULT accepted")
	}
}
