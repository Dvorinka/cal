package nameday

import "context"

// NameDate is a single (day, month, name) tuple returned by name search.
type NameDate struct {
	Day   int    `json:"day"`
	Month int    `json:"month"`
	Name  string `json:"name"`
}

// CountryResult is the search result for one country: the country code plus
// every date on which the searched name (or a name containing it) is
// celebrated.
type CountryResult struct {
	Country string     `json:"country"`
	Dates   []NameDate `json:"dates"`
}

// Provider is the abstraction over nameday data sources (embedded CSVs, the
// abalin HTTP API, or any future source). All country codes are lowercase
// ISO-3166 alpha-2 codes.
type Provider interface {
	// GetByDate returns namedays for the given month/day across all
	// supported countries, keyed by country code with comma-joined names.
	GetByDate(ctx context.Context, month, day int) (map[string]string, error)
	// SearchByName searches for a name across all supported countries
	// and returns every matching (country, date) pair.
	SearchByName(ctx context.Context, name string) ([]CountryResult, error)
	// SupportedCountries returns the sorted list of country codes this
	// provider can serve.
	SupportedCountries() []string
}
