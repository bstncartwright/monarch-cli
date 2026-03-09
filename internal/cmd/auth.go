package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pquerna/otp/totp"

	"github.com/bstn/monarch-cli/internal/app"
	"github.com/bstn/monarch-cli/internal/monarch"
	"github.com/bstn/monarch-cli/internal/output"
)

type AuthCmd struct {
	Login  AuthLoginCmd  `cmd:"" help:"Log into Monarch and store a session token"`
	Token  AuthTokenCmd  `cmd:"" help:"Store a token directly"`
	Status AuthStatusCmd `cmd:"" help:"Show auth status for the active profile"`
	Logout AuthLogoutCmd `cmd:"" help:"Remove the stored token for the active profile"`
}

type AuthLoginCmd struct {
	Email        string `help:"Email address to log in with"`
	MFASecretKey string `name:"mfa-secret-key" help:"Generate TOTP automatically from the secret key"`
}

func (c *AuthLoginCmd) Run(rt *app.Runtime) error {
	ctx := context.Background()
	email := strings.TrimSpace(c.Email)
	if email == "" {
		var err error
		email, err = prompt(rt, "Email")
		if err != nil {
			return err
		}
	}
	password, err := promptPassword(rt, "Password")
	if err != nil {
		return err
	}
	client := rt.NewLoginClient()
	result, err := client.Login(ctx, email, password, "")
	if err != nil {
		if !errors.Is(err, monarch.ErrMFARequired) {
			return err
		}
		code := ""
		if strings.TrimSpace(c.MFASecretKey) != "" {
			code, err = totp.GenerateCode(strings.TrimSpace(c.MFASecretKey), rtNow(rt))
			if err != nil {
				return fmt.Errorf("generate mfa code: %w", err)
			}
		} else {
			code, err = prompt(rt, "Two Factor Code")
			if err != nil {
				return err
			}
		}
		result, err = client.Login(ctx, email, password, code)
		if err != nil {
			return err
		}
	}
	profile, err := rt.UpsertProfile(rt.ActiveProfileName(), email, result.Token)
	if err != nil {
		return err
	}
	status := map[string]any{
		"profile": profile.Name,
		"email":   profile.Email,
		"stored":  true,
	}
	return rt.Print(status, output.Table{
		Headers: []string{"PROFILE", "EMAIL", "STORED"},
		Rows:    [][]string{{profile.Name, profile.Email, "true"}},
	})
}

type AuthTokenCmd struct {
	Set AuthTokenSetCmd `cmd:"" help:"Store a Monarch token directly for the active profile"`
}

type AuthTokenSetCmd struct {
	Value string `arg:"" optional:"" help:"Monarch token value; if omitted use --stdin"`
	Email string `help:"Email to associate with this stored profile"`
	Stdin bool   `name:"stdin" help:"Read the token from stdin"`
}

func (c *AuthTokenSetCmd) Run(rt *app.Runtime) error {
	token := strings.TrimSpace(c.Value)
	if c.Stdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		token = strings.TrimSpace(string(data))
	}
	if token == "" {
		return fmt.Errorf("token is required")
	}
	email := strings.TrimSpace(c.Email)
	if email == "" {
		email = rt.ActiveProfileName()
	}
	profile, err := rt.UpsertProfile(rt.ActiveProfileName(), email, token)
	if err != nil {
		return err
	}
	return rt.Print(map[string]any{
		"profile": profile.Name,
		"email":   profile.Email,
		"stored":  true,
	}, output.Table{
		Headers: []string{"PROFILE", "EMAIL", "STORED"},
		Rows:    [][]string{{profile.Name, profile.Email, "true"}},
	})
}

type AuthStatusCmd struct{}

func (c *AuthStatusCmd) Run(rt *app.Runtime) error {
	cfg, err := rt.LoadConfig()
	if err != nil {
		return err
	}
	profile := rt.ResolveProfile(cfg)
	status := map[string]any{
		"profile":        rt.ActiveProfileName(),
		"defaultProfile": cfg.DefaultProfile,
		"configured":     profile != nil,
		"email":          "",
		"token_source":   "",
		"config_path":    rt.ConfigPath(),
	}
	if profile != nil {
		status["email"] = profile.Email
	}
	token, source, err := rt.ResolveToken(cfg)
	if err == nil && token != "" {
		status["token_source"] = source
	}
	rows := [][]string{{
		fmt.Sprint(status["profile"]),
		fmt.Sprint(status["email"]),
		fmt.Sprint(status["configured"]),
		fmt.Sprint(status["token_source"]),
		fmt.Sprint(status["config_path"]),
	}}
	return rt.Print(status, output.Table{
		Headers: []string{"PROFILE", "EMAIL", "CONFIGURED", "TOKEN_SOURCE", "CONFIG_PATH"},
		Rows:    rows,
	})
}

type AuthLogoutCmd struct{}

func (c *AuthLogoutCmd) Run(rt *app.Runtime) error {
	if err := rt.DeleteProfileToken(rt.ActiveProfileName()); err != nil {
		return err
	}
	return rt.Print(map[string]any{
		"profile": rt.ActiveProfileName(),
		"removed": true,
	}, output.Table{
		Headers: []string{"PROFILE", "REMOVED"},
		Rows:    [][]string{{rt.ActiveProfileName(), "true"}},
	})
}
