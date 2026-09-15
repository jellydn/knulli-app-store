package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/jellydn/knulli-app-store/internal/catalog"
	"github.com/jellydn/knulli-app-store/internal/diagnostics"
	"github.com/jellydn/knulli-app-store/internal/installer"
	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/platform"
)

const usage = `Knulli App Store installer core

Usage:
  knulli-app validate MANIFEST
  knulli-app catalogue [-dir catalogue/packages] [-output build/catalog-index.json]
                     [-signing-key KEY] [-generate-signing-key KEY]
  knulli-app install [platform flags] MANIFEST
  knulli-app adopt [platform flags] MANIFEST
  knulli-app update [platform flags] MANIFEST
  knulli-app repair [platform flags] MANIFEST
  knulli-app uninstall [-root /] PACKAGE_ID

Platform flags override detected values:
  -root, -firmware, -firmware-version, -arch, -device, -resolution
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	if len(arguments) == 0 {
		fmt.Print(usage)
		return nil
	}
	switch arguments[0] {
	case "help", "-h", "--help":
		fmt.Print(usage)
		return nil
	case "validate":
		if len(arguments) != 2 {
			return fmt.Errorf("validate requires one manifest path")
		}
		pkg, err := manifest.Load(arguments[1])
		if err != nil {
			return err
		}
		fmt.Printf("valid %s (%s)\n", pkg.ID, pkg.Review.Status)
		return nil
	case "catalogue":
		flags := flag.NewFlagSet("catalogue", flag.ContinueOnError)
		directory := flags.String("dir", "catalogue/packages", "manifest directory")
		output := flags.String("output", "build/catalog-index.json", "index output path or -")
		signingKey := flags.String("signing-key", "", "ed25519 private key hex file")
		generateKey := flags.String("generate-signing-key", "", "write a new ed25519 key pair and sign the index")
		if err := flags.Parse(arguments[1:]); err != nil {
			return err
		}
		if *signingKey != "" && *generateKey != "" {
			return fmt.Errorf("use only one of -signing-key or -generate-signing-key")
		}
		if (*signingKey != "" || *generateKey != "") && *output == "-" {
			return fmt.Errorf("cannot sign catalogue output written to stdout")
		}
		index, err := catalog.Build(*directory)
		if err != nil {
			return err
		}
		if err := catalog.Write(index, *output); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "wrote %d packages to %s\n", len(index.Packages), *output)
		keyPath := *signingKey
		if *generateKey != "" {
			public, private, err := catalog.GenerateKey()
			if err != nil {
				return err
			}
			if err := os.WriteFile(*generateKey, []byte(hex.EncodeToString(private)+"\n"), 0600); err != nil {
				return err
			}
			if err := os.WriteFile(*generateKey+".pub", []byte(hex.EncodeToString(public)+"\n"), 0644); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "public key %s\n", hex.EncodeToString(public))
			keyPath = *generateKey
		}
		if keyPath == "" {
			return nil
		}
		raw, err := os.ReadFile(keyPath)
		if err != nil {
			return err
		}
		private, err := catalog.ParsePrivateKey(strings.TrimSpace(string(raw)))
		if err != nil {
			return err
		}
		if err := catalog.SignFile(*output, private); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "signed %s\n", catalog.SignaturePath(*output))
		return nil
	case "install", "adopt", "update", "repair":
		return runApply(arguments[0], arguments[1:])
	case "uninstall":
		flags := flag.NewFlagSet("uninstall", flag.ContinueOnError)
		root := flags.String("root", "/", "filesystem root")
		if err := flags.Parse(arguments[1:]); err != nil {
			return err
		}
		if flags.NArg() != 1 {
			return fmt.Errorf("uninstall requires one package id")
		}
		diagnosticLog, err := diagnostics.Open(*root)
		if err != nil {
			return fmt.Errorf("open diagnostics log: %w", err)
		}
		diagnosticLog.Event("startup", "component", "cli", "command", "uninstall")
		outcome := installer.OperationOutcome{}
		manager := installer.Manager{Root: *root, Diagnostics: diagnosticLog, Outcome: func(value installer.OperationOutcome) { outcome = value }}
		if err := manager.UninstallContext(context.Background(), flags.Arg(0)); err != nil {
			return err
		}
		fmt.Printf("uninstalled %s\n", flags.Arg(0))
		printGameListOutcome(outcome)
		return nil
	default:
		return fmt.Errorf("unknown command %q", arguments[0])
	}
}

func runApply(operation string, arguments []string) error {
	flags := flag.NewFlagSet(operation, flag.ContinueOnError)
	root := flags.String("root", "/", "filesystem root")
	firmware := flags.String("firmware", "", "firmware id")
	version := flags.String("firmware-version", "", "firmware version")
	arch := flags.String("arch", "", "CPU architecture")
	device := flags.String("device", "", "device family")
	resolution := flags.String("resolution", "", "display resolution")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return fmt.Errorf("%s requires one manifest path", operation)
	}
	diagnosticLog, err := diagnostics.Open(*root)
	if err != nil {
		return fmt.Errorf("open diagnostics log: %w", err)
	}
	diagnosticLog.Event("startup", "component", "cli", "command", operation)
	current := platform.Detect(*root)
	override(&current.Firmware, *firmware)
	override(&current.Version, *version)
	override(&current.Arch, *arch)
	override(&current.Device, *device)
	if *resolution != "" {
		candidates := append([]platform.ResolutionCandidate{platform.ResolutionCandidateFromString("command-line override", *resolution)}, current.ResolutionCandidates...)
		current = platform.WithResolutionCandidates(current, candidates)
	}
	setOverrideEvidence(&current, *firmware, *version)
	if current.Arch == "" {
		current.Arch = runtime.GOARCH
	}
	diagnosticLog.Event("platform_detected", "details", platform.Summary(current))
	pkg, err := manifest.Load(flags.Arg(0))
	if err != nil {
		diagnosticLog.Event("catalogue_error", "path", flags.Arg(0), "error", err.Error())
		return err
	}
	if current.Firmware == "" || current.Device == "" || current.Resolution == "" || (pkg.Compatibility != nil && pkg.Compatibility.MinimumVersion != "" && current.Version == "") {
		err := fmt.Errorf("platform detection is incomplete: %s", platform.Summary(current))
		diagnosticLog.Event("platform_detection_incomplete", "error", err.Error())
		return err
	}
	outcome := installer.OperationOutcome{}
	manager := installer.Manager{Root: *root, Platform: current, Diagnostics: diagnosticLog, Outcome: func(value installer.OperationOutcome) { outcome = value }}
	switch operation {
	case "install":
		err = manager.Install(context.Background(), pkg)
	case "adopt":
		err = manager.Adopt(context.Background(), pkg)
	case "update":
		err = manager.Update(context.Background(), pkg)
	case "repair":
		err = manager.Repair(context.Background(), pkg)
	}
	if err != nil {
		return err
	}
	fmt.Printf("%s completed for %s %s\n", operation, pkg.ID, pkg.Version)
	printGameListOutcome(outcome)
	return nil
}

func printGameListOutcome(outcome installer.OperationOutcome) {
	if outcome.GameListRefreshAccepted {
		fmt.Println("game list refresh accepted")
	} else if outcome.RestartRequired {
		fmt.Println("restart required to update game list")
	}
}

func override(target *string, value string) {
	if value != "" {
		*target = value
	}
}

func setOverrideEvidence(info *platform.Info, firmware, version string) {
	if firmware != "" {
		info.FirmwareRaw = firmware
		info.FirmwareSource = "command-line override"
	}
	if version != "" {
		info.VersionRaw = version
		info.VersionSource = "command-line override"
	}
}
