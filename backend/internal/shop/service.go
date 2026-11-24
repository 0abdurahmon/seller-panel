package shop

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
)

const (
	maxNameLen        = 120
	maxDescriptionLen = 2000
	maxAddressLen     = 300
)

var phonePattern = regexp.MustCompile(`^\+?[0-9]{7,15}$`)

// Service holds the shop profile rules.
type Service struct {
	repo  Repository
	logos LogoStore
}

// NewService builds the shop service.
func NewService(repo Repository, logos LogoStore) *Service {
	return &Service{repo: repo, logos: logos}
}

// Get returns the shop with the given ID.
func (s *Service) Get(ctx context.Context, shopID int64) (Shop, error) {
	return s.repo.Get(ctx, shopID)
}

// Update validates the request and replaces the shop profile and, when asked, its logo, together:
// a new logo file is removed again when the save fails, and the old logo is deleted after the save.
func (s *Service) Update(ctx context.Context, shopID int64, req UpdateRequest, lc LogoChange) (Shop, error) {
	p, err := validate(req)
	if err != nil {
		return Shop{}, err
	}
	if lc.Remove && lc.Add != nil {
		return Shop{}, fmt.Errorf("%w: send a logo file or remove_logo, not both", ErrValidation)
	}
	current, err := s.repo.Get(ctx, shopID)
	if err != nil {
		return Shop{}, err
	}
	old := ""
	if current.LogoURL != nil {
		old = *current.LogoURL
	}
	p.LogoURL = old
	saved := ""
	switch {
	case lc.Add != nil:
		if saved, err = s.logos.Save(ctx, shopID, bytes.NewReader(lc.Add)); err != nil {
			return Shop{}, fmt.Errorf("logo: %w", err)
		}
		p.LogoURL = saved
	case lc.Remove:
		p.LogoURL = ""
	}
	updated, err := s.repo.Update(ctx, shopID, p)
	if err != nil {
		if saved != "" {
			s.logos.Remove(ctx, shopID, []string{saved})
		}
		return Shop{}, err
	}
	if old != "" && old != p.LogoURL {
		s.logos.Remove(ctx, shopID, []string{old})
	}
	return updated, nil
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrValidation, fmt.Sprintf(format, args...))
}

func validate(req UpdateRequest) (Profile, error) {
	p := Profile{
		Name:          strings.TrimSpace(req.Name),
		Description:   strings.TrimSpace(req.Description),
		ContactPhone:  normalizePhone(req.ContactPhone),
		PickupAddress: strings.TrimSpace(req.PickupAddress),
	}
	if p.Name == "" || len([]rune(p.Name)) > maxNameLen {
		return Profile{}, fmt.Errorf("%w: name must be 1-%d characters", ErrValidation, maxNameLen)
	}
	if len([]rune(p.Description)) > maxDescriptionLen {
		return Profile{}, fmt.Errorf("%w: description must be at most %d characters", ErrValidation, maxDescriptionLen)
	}
	if len([]rune(p.PickupAddress)) > maxAddressLen {
		return Profile{}, fmt.Errorf("%w: pickup_address must be at most %d characters", ErrValidation, maxAddressLen)
	}
	if p.ContactPhone != "" && !phonePattern.MatchString(p.ContactPhone) {
		return Profile{}, fmt.Errorf("%w: contact_phone must be 7-15 digits with an optional leading +", ErrValidation)
	}
	return p, nil
}

// normalizePhone removes spaces, dashes and parentheses so "+998 (90) 123-45-67" is accepted.
func normalizePhone(raw string) string {
	r := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "")
	return r.Replace(strings.TrimSpace(raw))
}
