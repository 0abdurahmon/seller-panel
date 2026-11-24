package shop

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// ErrValidation is returned for invalid input. The handler maps it to 400.
var ErrValidation = errors.New("validation failed")

const (
	maxNameLen        = 120
	maxDescriptionLen = 2000
	maxAddressLen     = 300
	maxLogoURLLen     = 2048
)

var phonePattern = regexp.MustCompile(`^\+?[0-9]{7,15}$`)

// Service holds the shop profile rules.
type Service struct {
	repo Repository
}

// NewService builds the shop service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Get returns the shop with the given ID.
func (s *Service) Get(ctx context.Context, shopID int64) (Shop, error) {
	return s.repo.Get(ctx, shopID)
}

// Update validates the request and replaces the shop profile.
func (s *Service) Update(ctx context.Context, shopID int64, req UpdateRequest) (Shop, error) {
	p, err := validate(req)
	if err != nil {
		return Shop{}, err
	}
	return s.repo.Update(ctx, shopID, p)
}

func validate(req UpdateRequest) (Profile, error) {
	p := Profile{
		Name:          strings.TrimSpace(req.Name),
		LogoURL:       strings.TrimSpace(req.LogoURL),
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
	if p.LogoURL != "" {
		if len(p.LogoURL) > maxLogoURLLen {
			return Profile{}, fmt.Errorf("%w: logo_url is too long", ErrValidation)
		}
		u, err := url.Parse(p.LogoURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return Profile{}, fmt.Errorf("%w: logo_url must be an http or https URL", ErrValidation)
		}
	}
	return p, nil
}

// normalizePhone removes spaces, dashes and parentheses so "+998 (90) 123-45-67" is accepted.
func normalizePhone(raw string) string {
	r := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "")
	return r.Replace(strings.TrimSpace(raw))
}
