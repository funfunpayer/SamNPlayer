// Command license-tool issues SamNPlayer personal license keys (offline).
//
//	license-tool genkey --out ~/.samn/issuer
//	license-tool issue --key ~/.samn/issuer --sub anna@… --years 1 --out Anna.key
//	license-tool issue --key ~/.samn/issuer --sub friend --tier invite --exp never --out invite.key
//
// Keep the private key outside git. The app embeds only the public key
// (license.EmbeddedPublicKeyHex). For local testing against the committed
// public key, use --key license/testdata/issuer.ed25519 (DEV ONLY).
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/funfunpayer/SamNPlayer/license"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "genkey":
		os.Exit(cmdGenkey(os.Args[2:]))
	case "issue":
		os.Exit(cmdIssue(os.Args[2:]))
	case "verify":
		os.Exit(cmdVerify(os.Args[2:]))
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `license-tool — issue SamNPlayer personal license keys (offline)

Commands:
  genkey  Create a new Ed25519 issuer keypair (raw files)
  issue   Sign a personal license (1 year standard, or invite/internal)
  verify  Check a key file against the embedded (or --pub) public key

Examples:
  license-tool genkey --out ~/.samn/issuer
  license-tool issue --key ~/.samn/issuer --sub anna@example.com --years 1 --out Anna.key
  license-tool issue --key license/testdata/issuer.ed25519 --sub owner --tier internal --exp never --out owner.key
  license-tool verify --file owner.key
`)
}

func cmdGenkey(args []string) int {
	fs := flag.NewFlagSet("genkey", flag.ExitOnError)
	out := fs.String("out", "", "output path prefix (writes OUT and OUT.pub)")
	_ = fs.Parse(args)
	if *out == "" {
		fmt.Fprintln(os.Stderr, "--out required")
		return 2
	}
	pub, priv, err := license.GenerateKeyPair()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	dir := filepath.Dir(*out)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	if err := license.WriteKeyPair(*out, *out+".pub", pub, priv); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("Wrote %s (private) and %s.pub\n", *out, *out)
	fmt.Printf("Public key hex (paste into license/pubkey.go if replacing embedded):\n%s\n", hex.EncodeToString(pub))
	fmt.Println("Keep the private file offline — never commit it.")
	return 0
}

func cmdIssue(args []string) int {
	fs := flag.NewFlagSet("issue", flag.ExitOnError)
	keyPath := fs.String("key", "", "path to raw Ed25519 private key")
	sub := fs.String("sub", "", "person id (email or stable id) — one person per key")
	tier := fs.String("tier", license.TierStandard, "standard | invite | internal")
	years := fs.Int("years", 1, "validity in years (standard tier)")
	exp := fs.String("exp", "", "set to 'never' for invite/internal (no expiry)")
	out := fs.String("out", "", "output license file (default: stdout)")
	note := fs.String("note", "", "optional note in claims")
	_ = fs.Parse(args)

	if *keyPath == "" || *sub == "" {
		fmt.Fprintln(os.Stderr, "--key and --sub required")
		return 2
	}
	priv, err := license.LoadPrivateKey(*keyPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	now := time.Now().UTC()
	t := strings.ToLower(strings.TrimSpace(*tier))
	switch t {
	case license.TierStandard, license.TierInvite, license.TierInternal:
	default:
		fmt.Fprintf(os.Stderr, "unknown tier %q (want standard|invite|internal)\n", *tier)
		return 2
	}
	never := strings.EqualFold(*exp, "never") || t == license.TierInvite || t == license.TierInternal

	var claims license.Claims
	if never {
		if t == license.TierStandard {
			t = license.TierInvite
		}
		claims = license.NewInviteClaims(*sub, t, now)
	} else {
		claims = license.NewStandardClaims(*sub, *years, now)
	}
	claims.Note = *note

	tok, err := license.Issue(priv, claims)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if *out == "" || *out == "-" {
		fmt.Println(tok)
		return 0
	}
	if err := license.SaveToken(*out, tok); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "Wrote %s (tier=%s sub=%s until=%s)\n", *out, claims.Tier, claims.Sub, claims.ValidUntil())
	return 0
}

func cmdVerify(args []string) int {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	file := fs.String("file", "", "license key file")
	pubPath := fs.String("pub", "", "optional raw public key (default: embedded)")
	_ = fs.Parse(args)
	if *file == "" {
		fmt.Fprintln(os.Stderr, "--file required")
		return 2
	}
	tok, err := license.LoadFile(*file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	pub := license.EmbeddedPublicKey
	if *pubPath != "" {
		pub, err = license.LoadPublicKey(*pubPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	st := license.Evaluate(tok, pub, time.Now().UTC())
	fmt.Printf("state=%s licensed=%v effective=%v enforcement=%v\n", st.State, st.Licensed, st.Effective, st.Enforcement)
	fmt.Printf("sub=%s tier=%s until=%s\n", st.Sub, st.Tier, st.ValidUntil)
	fmt.Println(st.Message)
	if st.Error != "" {
		fmt.Println("error:", st.Error)
		return 1
	}
	if !st.Licensed {
		return 1
	}
	return 0
}
