package network

import (
	"testing"
)

func TestPastaAvailable(t *testing.T) {
	p := NewPasta()

	// Just test that it doesn't panic
	available := p.Available()
	t.Logf("pasta available: %v", available)
}

func TestSelectProvider(t *testing.T) {
	provider, err := SelectProvider()

	if err == ErrNoPastaProvider {
		t.Skip("pasta not available")
	}

	if err != nil {
		t.Fatalf("SelectProvider failed: %v", err)
	}

	if provider == nil {
		t.Fatal("provider is nil")
	}

	t.Logf("Selected provider: %s", provider.Name())
}

func TestCheckPastaAvailable(t *testing.T) {
	available := CheckPastaAvailable()
	t.Logf("pasta available: %v", available)
}

func TestPastaGatewayIP(t *testing.T) {
	p := NewPasta()
	ip := p.GatewayIP()

	if ip != "10.0.2.2" {
		t.Errorf("unexpected gateway IP: %s", ip)
	}
}

func TestPastaName(t *testing.T) {
	p := NewPasta()
	if p.Name() != "pasta" {
		t.Errorf("unexpected name: %s", p.Name())
	}
}

func TestPastaNetworkIsolated(t *testing.T) {
	p := NewPasta()
	if !p.NetworkIsolated() {
		t.Error("pasta should report network isolated")
	}
}

func TestPastaImplementsProvider(t *testing.T) {
	var _ Provider = (*Pasta)(nil)
}
