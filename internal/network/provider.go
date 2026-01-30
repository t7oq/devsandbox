package network

import "errors"

var ErrNoPastaProvider = errors.New("pasta not available (install passt package for proxy mode)")

// Provider defines the interface for user-mode network providers.
// This interface is intentionally minimal - it only includes methods
// that are actually used in the proxy flow.
type Provider interface {
	// Name returns the provider name
	Name() string

	// Available checks if the provider is installed and usable
	Available() bool

	// GatewayIP returns the gateway IP address accessible from the namespace
	GatewayIP() string

	// NetworkIsolated returns true if the provider creates an isolated network namespace
	NetworkIsolated() bool
}

// SelectProvider returns the pasta network provider if available.
// Proxy mode requires pasta for proper network isolation and traffic enforcement.
// Returns an error if pasta is not installed.
func SelectProvider() (Provider, error) {
	pasta := NewPasta()
	if pasta.Available() {
		return pasta, nil
	}

	return nil, ErrNoPastaProvider
}

// CheckPastaAvailable returns true if pasta is available
func CheckPastaAvailable() bool {
	pasta := NewPasta()
	return pasta.Available()
}
