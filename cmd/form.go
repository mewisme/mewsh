package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/x/term"
	"github.com/mewisme/mewsh/internal/profile"
	"github.com/mewisme/mewsh/internal/secret"
)

type formOptions struct {
	Editing   bool
	OrigAlias string
}

func promptProfile(existing *profile.Profile, aliases []string, opts formOptions) (profile.Profile, error) {
	if !term.IsTerminal(os.Stdin.Fd()) {
		return profile.Profile{}, fmt.Errorf("interactive prompts require a terminal (stdin is not a TTY)")
	}
	return promptProfileHuh(existing, aliases, opts)
}

func promptProfileHuh(existing *profile.Profile, aliases []string, opts formOptions) (profile.Profile, error) {
	p := profile.Default()
	if existing != nil {
		p = *existing
		p.ApplyDefaults()
	}

	var (
		alias      = p.Alias
		connType   = p.ConnectionType
		host       = p.Host
		cfHostname = p.CFHostname
		portStr    = strconv.Itoa(p.Port)
		user       = p.User
		authType   = p.AuthType
		keyPath    = p.KeyPath
		password   string
		note       = p.Note
	)

	modeTitle := "mewsh — Add Profile"
	if opts.Editing {
		modeTitle = "mewsh — Edit Profile"
	}

	run := func(stepTitle string, fields ...huh.Field) error {
		err := huh.NewForm(
			huh.NewGroup(fields...).
				Title(modeTitle).
				Description(stepTitle),
		).WithShowHelp(true).Run()
		return wrapFormErr(err)
	}

	// Step: alias
	if !opts.Editing {
		if err := run("Step — Profile alias",
			huh.NewInput().
				Title("Profile alias").
				Value(&alias).
				Validate(func(s string) error {
					s = strings.TrimSpace(s)
					if s == "" {
						return fmt.Errorf("alias is required")
					}
					if aliasExists(aliases, s) {
						return fmt.Errorf("alias %q already exists", s)
					}
					return nil
				}),
		); err != nil {
			return profile.Profile{}, wrapFormErr(err)
		}
	}

	// Step: connection type
	if err := run("Step — Connection type",
		huh.NewSelect[string]().
			Title("Connection type").
			Options(
				huh.NewOption("Direct SSH", profile.ConnectionDirect),
				huh.NewOption("Cloudflare Access", profile.ConnectionCloudflareAccess),
			).
			Value(&connType),
	); err != nil {
		return profile.Profile{}, wrapFormErr(err)
	}

	// Steps: target
	if connType == profile.ConnectionDirect {
		if err := run("Step — Host",
			huh.NewInput().
				Title("Host").
				Value(&host).
				Validate(requiredNonEmpty("host")),
		); err != nil {
			return profile.Profile{}, wrapFormErr(err)
		}
		if err := run("Step — Port",
			huh.NewInput().
				Title("Port").
				Placeholder("22").
				Value(&portStr),
		); err != nil {
			return profile.Profile{}, wrapFormErr(err)
		}
	} else {
		if err := run("Step — Cloudflare hostname",
			huh.NewInput().
				Title("CF hostname").
				Value(&cfHostname).
				Validate(requiredNonEmpty("cloudflare hostname")),
		); err != nil {
			return profile.Profile{}, wrapFormErr(err)
		}
	}

	// Step: user
	if err := run("Step — User",
		huh.NewInput().
			Title("User").
			Value(&user).
			Validate(requiredNonEmpty("user")),
	); err != nil {
		return profile.Profile{}, wrapFormErr(err)
	}

	// Step: auth type
	if err := run("Step — Auth type",
		huh.NewSelect[string]().
			Title("Auth type").
			Options(
				huh.NewOption("SSH Agent", profile.AuthAgent),
				huh.NewOption("Key", profile.AuthKey),
				huh.NewOption("Password", profile.AuthPassword),
			).
			Value(&authType),
	); err != nil {
		return profile.Profile{}, wrapFormErr(err)
	}

	// Step: credentials
	if authType == profile.AuthKey {
		if err := run("Step — Key path",
			huh.NewInput().
				Title("Key path").
				Value(&keyPath).
				Validate(requiredNonEmpty("key path")),
		); err != nil {
			return profile.Profile{}, wrapFormErr(err)
		}
	}
	if authType == profile.AuthPassword {
		desc := "Required for new profiles"
		if opts.Editing {
			desc = "Leave blank to keep existing password"
		}
		if err := run("Step — Password",
			huh.NewInput().
				Title("Password").
				Description(desc).
				EchoMode(huh.EchoModePassword).
				Value(&password),
		); err != nil {
			return profile.Profile{}, wrapFormErr(err)
		}
	}

	// Step: note
	if err := run("Step — Note (optional)",
		huh.NewInput().
			Title("Note").
			Value(&note),
	); err != nil {
		return profile.Profile{}, wrapFormErr(err)
	}

	port, err := strconv.Atoi(strings.TrimSpace(portStr))
	if err != nil || strings.TrimSpace(portStr) == "" {
		port = 22
	}

	result := profile.Profile{
		Alias:          strings.TrimSpace(alias),
		Host:           strings.TrimSpace(host),
		Port:           port,
		User:           strings.TrimSpace(user),
		AuthType:       authType,
		KeyPath:        strings.TrimSpace(keyPath),
		PasswordMode:   profile.PasswordManual,
		Note:           strings.TrimSpace(note),
		ConnectionType: connType,
		CFHostname:     strings.TrimSpace(cfHostname),
	}

	if opts.Editing {
		result.Alias = opts.OrigAlias
		result.PasswordRef = opts.OrigAlias
	} else {
		result.PasswordRef = result.Alias
	}

	// Step: summary
	save := true
	if err := run("Step — Summary",
		huh.NewNote().
			Title("Profile summary").
			Description(formatSummary(result, password != "", opts.Editing)),
		huh.NewConfirm().
			Title("Save profile?").
			Affirmative("Save").
			Negative("Cancel").
			Value(&save),
	); err != nil {
		return profile.Profile{}, wrapFormErr(err)
	}
	if !save {
		return profile.Profile{}, errFormCancelled
	}

	if authType == profile.AuthPassword {
		if password != "" {
			if err := secret.SetPassword(result.PasswordRef, password); err != nil {
				return profile.Profile{}, err
			}
		} else if !opts.Editing {
			return profile.Profile{}, fmt.Errorf("password is required for new password auth profiles")
		}
	}

	if err := result.Validate(aliases, opts.Editing); err != nil {
		return profile.Profile{}, err
	}
	return result, nil
}

func requiredNonEmpty(label string) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("%s is required", label)
		}
		return nil
	}
}

func formatSummary(p profile.Profile, hasPassword bool, editing bool) string {
	conn := "Direct SSH"
	target := fmt.Sprintf("%s:%d", p.Host, p.Port)
	if p.ConnectionType == profile.ConnectionCloudflareAccess {
		conn = "Cloudflare Access"
		target = p.CFHostname
	}
	lines := []string{
		fmt.Sprintf("Alias: %s", p.Alias),
		fmt.Sprintf("Connection: %s", conn),
		fmt.Sprintf("Target: %s", target),
		fmt.Sprintf("User: %s", p.User),
		fmt.Sprintf("Auth: %s", p.AuthType),
	}
	if p.AuthType == profile.AuthKey {
		lines = append(lines, fmt.Sprintf("Key: %s", p.KeyPath))
	}
	if p.AuthType == profile.AuthPassword {
		switch {
		case hasPassword:
			lines = append(lines, "Password: (will be saved to keyring)")
		case editing:
			lines = append(lines, "Password: (unchanged)")
		}
	}
	if p.Note != "" {
		lines = append(lines, fmt.Sprintf("Note: %s", p.Note))
	}
	return strings.Join(lines, "\n")
}

func aliasExists(aliases []string, name string) bool {
	for _, a := range aliases {
		if a == name {
			return true
		}
	}
	return false
}

func aliasesFromConfig(cfgProfiles []profile.Profile) []string {
	out := make([]string, len(cfgProfiles))
	for i, p := range cfgProfiles {
		out[i] = p.Alias
	}
	return out
}
