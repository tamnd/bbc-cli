package bbc

import (
	"testing"

	"github.com/tamnd/any-cli/kit"
)

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "bbc" {
		t.Errorf("Scheme = %q, want bbc", info.Scheme)
	}
	if info.Identity.Binary != "bbc" {
		t.Errorf("Identity.Binary = %q, want bbc", info.Identity.Binary)
	}
}

func TestHostWiring(t *testing.T) {
	h, err := kit.Open()
	if err != nil {
		t.Fatal(err)
	}
	domains := h.Domains()
	found := false
	for _, d := range domains {
		if d == "bbc" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("bbc domain not registered; got %v", domains)
	}
}
