package appstore

import (
	"fmt"

	"github.com/jellydn/knulli-app-store/internal/installer"
	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/platform"
)

// State is the typed catalogue state of one package: the single answer to
// "can the user act on this, and what is blocking it?".
type State string

const (
	StateCandidate    State = "candidate"    // technical review has not passed
	StateIncompatible State = "incompatible" // review passed but the platform rejects
	StateExternal     State = "external"     // review passed, unowned files at the destination
	StateIssue        State = "issue"        // installed but failing health checks
	StateInstalled    State = "installed"    // installed and healthy
	StateAvailable    State = "available"    // ready to install
)

// Reason is one typed element of why the state is what it is. Reasons are
// data so every consumer renders from the same truth instead of parsing
// English prose.
type Reason struct {
	Kind   string // community | platform | review | install | health
	Detail string // rendered text, for diagnostics and UI display
}

// Verdict joins review status, platform compatibility, install state, and
// health into one typed catalogue decision. Policy lives in manifest,
// platform, and installer; assess only combines their answers.
type Verdict struct {
	State   State
	Reasons []Reason
	Actions []Action
}

// assess is the single home of the catalogue verdict.
func assess(pkg manifest.Package, current platform.Info, status installer.Status, preExisting bool) Verdict {
	var verdict Verdict

	// Installed packages: uninstall stays available even when the platform
	// gate fails; repair and update require full compatibility.
	if status.Installed {
		actions := []Action{Uninstall}
		compatible := pkg.Installable()
		if compatible {
			if err := platform.Check(pkg, current); err != nil {
				compatible = false
				verdict.Reasons = append(verdict.Reasons, Reason{Kind: "platform", Detail: err.Error()})
			}
		}
		if compatible {
			if pkg.Experimental() {
				verdict.Reasons = append(verdict.Reasons, Reason{Kind: "review", Detail: experimentalReason(current)})
			}
			if status.Healthy {
				verdict.State = StateInstalled
			} else {
				verdict.State = StateIssue
				if len(status.Issues) > 0 {
					verdict.Reasons = append(verdict.Reasons, Reason{Kind: "health", Detail: status.Issues[0].String()})
				}
			}
			if status.Version != pkg.Version {
				actions = append([]Action{Update}, actions...)
			}
			actions = append(actions, Repair)
		} else {
			if status.Healthy {
				verdict.State = StateInstalled
			} else {
				verdict.State = StateIssue
				if len(status.Issues) > 0 {
					verdict.Reasons = append(verdict.Reasons, Reason{Kind: "health", Detail: status.Issues[0].String()})
				}
			}
		}
		verdict.Actions = actions
		return verdict
	}

	// Review gate: technical status decides whether anything can be done,
	// regardless of community approval (ADR-0002 keeps the two separate).
	if !pkg.Installable() {
		verdict.State = StateCandidate
		if pkg.Review.Approval != nil {
			verdict.Reasons = append(verdict.Reasons, Reason{Kind: "community", Detail: "Community approved; installation is blocked by technical review"})
		} else {
			verdict.Reasons = append(verdict.Reasons, Reason{Kind: "review", Detail: "Candidate: compatibility is not approved"})
		}
		return verdict
	}

	// Platform gate: the declared compatibility matrix against the detected
	// platform.
	if err := platform.Check(pkg, current); err != nil {
		verdict.State = StateIncompatible
		verdict.Reasons = append(verdict.Reasons, Reason{Kind: "platform", Detail: err.Error()})
		return verdict
	}

	// Experimental packages stay actionable but record the weaker evidence
	// in the verdict.
	if pkg.Experimental() {
		verdict.Reasons = append(verdict.Reasons, Reason{Kind: "review", Detail: experimentalReason(current)})
	}

	if preExisting {
		verdict.State = StateExternal
		verdict.Reasons = append(verdict.Reasons, Reason{Kind: "install", Detail: "An external installation exists; use Manage existing"})
		verdict.Actions = []Action{Adopt}
		return verdict
	}
	verdict.State = StateAvailable
	verdict.Actions = []Action{Install}
	return verdict
}

func experimentalReason(current platform.Info) string {
	return fmt.Sprintf("Experimental compatibility: Knulli identity confirmed from %s; no minimum version is claimed; device=%s architecture=%s resolution=%s source=%s version=%s", current.Evidence.FirmwareSource, current.Device, current.Arch, current.Resolution, current.ResolutionSource, current.Version)
}

// Message renders the verdict's first reason for display. A verdict without
// reasons is fully allowed.
func (v Verdict) Message() string {
	if len(v.Reasons) == 0 {
		return "Compatible with detected platform"
	}
	return v.Reasons[0].Detail
}
