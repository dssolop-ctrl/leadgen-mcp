package auth

import (
	"fmt"

	"github.com/leadgen-mcp/server/config"
)

// AccountResolver resolves account names to API tokens.
type AccountResolver struct {
	accounts map[string][]config.AccountEntry // platform -> entries
}

func NewAccountResolver(accounts map[string][]config.AccountEntry) *AccountResolver {
	return &AccountResolver{accounts: accounts}
}

// Resolve returns the token for the given platform and account name.
// If accountName is empty, returns the default account token.
func (r *AccountResolver) Resolve(platform, accountName string) (string, error) {
	entries, ok := r.accounts[platform]
	if !ok {
		return "", fmt.Errorf("platform %q not configured", platform)
	}

	if accountName == "" {
		// Find default
		for _, e := range entries {
			if e.Default {
				return e.Token, nil
			}
		}
		// If no default, use first
		if len(entries) > 0 {
			return entries[0].Token, nil
		}
		return "", fmt.Errorf("no accounts configured for platform %q", platform)
	}

	for _, e := range entries {
		if e.Name == accountName {
			return e.Token, nil
		}
	}
	return "", fmt.Errorf("account %q not found for platform %q", accountName, platform)
}

// ResolveYandex is a shortcut for Resolve("yandex", name).
func (r *AccountResolver) ResolveYandex(accountName string) (string, error) {
	return r.Resolve("yandex", accountName)
}

// ResolveVK is a shortcut for Resolve("vk", name).
func (r *AccountResolver) ResolveVK(accountName string) (string, error) {
	return r.Resolve("vk", accountName)
}

// ListAccounts returns account names for a platform.
func (r *AccountResolver) ListAccounts(platform string) []string {
	entries := r.accounts[platform]
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
	}
	return names
}
