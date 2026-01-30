package bwrap

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

func CheckInstalled() error {
	_, err := exec.LookPath("bwrap")
	if err != nil {
		return errors.New("bubblewrap (bwrap) is not installed\nInstall with: sudo apt install bubblewrap")
	}
	return nil
}

func Exec(bwrapArgs []string, shellCmd []string) error {
	bwrapPath, err := exec.LookPath("bwrap")
	if err != nil {
		return err
	}

	args := make([]string, 0, len(bwrapArgs)+len(shellCmd)+2)
	args = append(args, "bwrap")
	args = append(args, bwrapArgs...)
	args = append(args, "--")
	args = append(args, shellCmd...)

	return syscall.Exec(bwrapPath, args, os.Environ())
}

// ExecWithPasta wraps bwrap execution inside pasta for network namespace isolation.
// This creates an isolated network namespace where all traffic must go through
// pasta's gateway, which we configure to route through our proxy.
//
// Unlike the regular Exec function, this uses exec.Command instead of syscall.Exec
// so that the calling process (and its proxy server goroutine) stays alive.
func ExecWithPasta(bwrapArgs []string, shellCmd []string) error {
	pastaPath, err := exec.LookPath("pasta")
	if err != nil {
		return errors.New("pasta is not installed (from passt package)")
	}

	bwrapPath, err := exec.LookPath("bwrap")
	if err != nil {
		return err
	}

	// Build pasta command with network isolation:
	// pasta --config-net --map-host-loopback 10.0.2.2 -f -- sh -c '...' _ bwrap [args] -- shell
	//
	// --config-net: Configure tap interface in namespace (required for network to work)
	// --map-host-loopback 10.0.2.2: Map 10.0.2.2 to host's 127.0.0.1 (for proxy access)
	// -f: Run in foreground (pasta exits when child exits)
	//
	// The wrapper script uses iptables to block all traffic except to the gateway.
	// This ensures only 10.0.2.2 (proxy gateway -> host loopback) is reachable:
	// - Traffic to 10.0.2.2 is allowed (our proxy)
	// - All other traffic is rejected
	// - UDP/TCP to internet is blocked at firewall level
	const wrapperScript = `iptables -I OUTPUT -d 10.0.2.2 -j ACCEPT 2>/dev/null; iptables -A OUTPUT -j REJECT 2>/dev/null; exec "$@"`

	args := make([]string, 0, len(bwrapArgs)+len(shellCmd)+16)
	args = append(args, "--config-net")                    // Configure network interface
	args = append(args, "--map-host-loopback", "10.0.2.2") // Map to host loopback
	args = append(args, "-f")                              // Foreground mode
	args = append(args, "--")
	args = append(args, "sh", "-c", wrapperScript, "_") // Wrapper to delete default route
	args = append(args, bwrapPath)
	args = append(args, bwrapArgs...)
	args = append(args, "--")
	args = append(args, shellCmd...)

	// Use exec.Command instead of syscall.Exec so the parent process stays alive
	// This is necessary because we have a proxy server goroutine running
	cmd := exec.Command(pastaPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	return cmd.Run()
}
