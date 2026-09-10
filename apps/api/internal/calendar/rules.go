package calendar

import "time"

// rule resolves a holiday to a concrete date for a given Gregorian year.
// Rules cover fixed dates, Easter-relative dates (Western and Orthodox),
// and nth/last weekday patterns, which together describe the public
// holidays of most countries without shipping a static dataset.
type rule func(year int) time.Time

// fixed is a holiday on the same month/day every year.
func fixed(month time.Month, day int) rule {
	return func(year int) time.Time {
		return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	}
}

// easterOff is a holiday offset days from Western (Gregorian) Easter Sunday.
func easterOff(offset int) rule {
	return func(year int) time.Time {
		return gregorianEaster(year).AddDate(0, 0, offset)
	}
}

// orthodoxOff is a holiday offset days from Orthodox (Julian) Easter Sunday.
func orthodoxOff(offset int) rule {
	return func(year int) time.Time {
		return orthodoxEaster(year).AddDate(0, 0, offset)
	}
}

// nthWeekday is the n-th weekday of a month (n >= 1, e.g. 3rd Monday).
func nthWeekday(month time.Month, weekday time.Weekday, n int) rule {
	return func(year int) time.Time {
		first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
		shift := (int(weekday) - int(first.Weekday()) + 7) % 7
		return first.AddDate(0, 0, shift+7*(n-1))
	}
}

// lastWeekday is the final weekday of a month (e.g. last Monday of May).
func lastWeekday(month time.Month, weekday time.Weekday) rule {
	return func(year int) time.Time {
		last := time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
		shift := (int(last.Weekday()) - int(weekday) + 7) % 7
		return last.AddDate(0, 0, -shift)
	}
}

// weekdayOnOrBefore is the nearest weekday on or before month/day
// (e.g. Victoria Day: last Monday on or before May 24).
func weekdayOnOrBefore(month time.Month, day int, weekday time.Weekday) rule {
	return func(year int) time.Time {
		d := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		shift := (int(d.Weekday()) - int(weekday) + 7) % 7
		return d.AddDate(0, 0, -shift)
	}
}

// weekdayOnOrAfter is the nearest weekday on or after month/day
// (e.g. Colombia's Emiliani rule moving holidays to the next Monday).
func weekdayOnOrAfter(month time.Month, day int, weekday time.Weekday) rule {
	return func(year int) time.Time {
		d := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		shift := (int(weekday) - int(d.Weekday()) + 7) % 7
		return d.AddDate(0, 0, shift)
	}
}

// approxEquinox approximates the March/September equinox day.
// Accurate for years 2000-2099 which covers any realistic usage.
func approxEquinox(base float64) rule {
	return func(year int) time.Time {
		day := int(base+0.242194*float64(year-1980)) - (year-1980)/4
		month := time.March
		if base > 22 {
			month = time.September
		}
		return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	}
}

// gregorianEaster computes Western Easter Sunday via the Anonymous
// Gregorian algorithm (Computus).
func gregorianEaster(year int) time.Time {
	a := year % 19
	b, c := year/100, year%100
	d, e := b/4, b%4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i, k := c/4, c%4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := (h + l - 7*m + 114) / 31
	day := ((h + l - 7*m + 114) % 31) + 1
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

// orthodoxEaster computes Orthodox Easter Sunday via the Julian computus,
// converted to the Gregorian calendar (13-day offset through 2100).
func orthodoxEaster(year int) time.Time {
	a := year % 4
	b := year % 7
	c := year % 19
	d := (19*c + 15) % 30
	e := (2*a + 4*b - d + 34) % 7
	month := (d + e + 114) / 31
	day := ((d + e + 114) % 31) + 1
	julian := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	return julian.AddDate(0, 0, 13)
}

type holidayDef struct {
	Name string
	Rule rule
}

// Country describes a supported holiday region for the settings UI.
type Country struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

var countries = map[string]struct {
	Name     string
	Holidays []holidayDef
}{
	"US": {"United States", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Martin Luther King Jr. Day", nthWeekday(time.January, time.Monday, 3)},
		{"Presidents' Day", nthWeekday(time.February, time.Monday, 3)},
		{"Memorial Day", lastWeekday(time.May, time.Monday)},
		{"Juneteenth", fixed(time.June, 19)},
		{"Independence Day", fixed(time.July, 4)},
		{"Labor Day", nthWeekday(time.September, time.Monday, 1)},
		{"Columbus Day", nthWeekday(time.October, time.Monday, 2)},
		{"Veterans Day", fixed(time.November, 11)},
		{"Thanksgiving", nthWeekday(time.November, time.Thursday, 4)},
		{"Christmas Day", fixed(time.December, 25)},
	}},
	"CA": {"Canada", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Good Friday", easterOff(-2)},
		{"Victoria Day", weekdayOnOrBefore(time.May, 24, time.Monday)},
		{"Canada Day", fixed(time.July, 1)},
		{"Labour Day", nthWeekday(time.September, time.Monday, 1)},
		{"National Day for Truth and Reconciliation", fixed(time.September, 30)},
		{"Thanksgiving", nthWeekday(time.October, time.Monday, 2)},
		{"Remembrance Day", fixed(time.November, 11)},
		{"Christmas Day", fixed(time.December, 25)},
		{"Boxing Day", fixed(time.December, 26)},
	}},
	"GB": {"United Kingdom", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Good Friday", easterOff(-2)},
		{"Easter Monday", easterOff(1)},
		{"Early May Bank Holiday", nthWeekday(time.May, time.Monday, 1)},
		{"Spring Bank Holiday", lastWeekday(time.May, time.Monday)},
		{"Summer Bank Holiday", lastWeekday(time.August, time.Monday)},
		{"Christmas Day", fixed(time.December, 25)},
		{"Boxing Day", fixed(time.December, 26)},
	}},
	"IE": {"Ireland", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Saint Brigid's Day", nthWeekday(time.February, time.Monday, 1)},
		{"Saint Patrick's Day", fixed(time.March, 17)},
		{"Easter Monday", easterOff(1)},
		{"May Day", nthWeekday(time.May, time.Monday, 1)},
		{"June Bank Holiday", nthWeekday(time.June, time.Monday, 1)},
		{"August Bank Holiday", nthWeekday(time.August, time.Monday, 1)},
		{"October Bank Holiday", lastWeekday(time.October, time.Monday)},
		{"Christmas Day", fixed(time.December, 25)},
		{"Saint Stephen's Day", fixed(time.December, 26)},
	}},
	"CZ": {"Czechia", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Good Friday", easterOff(-2)},
		{"Easter Monday", easterOff(1)},
		{"Labour Day", fixed(time.May, 1)},
		{"Liberation Day", fixed(time.May, 8)},
		{"Saints Cyril and Methodius Day", fixed(time.July, 5)},
		{"Jan Hus Day", fixed(time.July, 6)},
		{"Czech Statehood Day", fixed(time.September, 28)},
		{"Independent Czechoslovak State Day", fixed(time.October, 28)},
		{"Freedom and Democracy Day", fixed(time.November, 17)},
		{"Christmas Eve", fixed(time.December, 24)},
		{"Christmas Day", fixed(time.December, 25)},
		{"Saint Stephen's Day", fixed(time.December, 26)},
	}},
	"SK": {"Slovakia", []holidayDef{
		{"Republic Day", fixed(time.January, 1)},
		{"Epiphany", fixed(time.January, 6)},
		{"Good Friday", easterOff(-2)},
		{"Easter Monday", easterOff(1)},
		{"Labour Day", fixed(time.May, 1)},
		{"Victory over Fascism Day", fixed(time.May, 8)},
		{"Saints Cyril and Methodius Day", fixed(time.July, 5)},
		{"Slovak National Uprising Day", fixed(time.August, 29)},
		{"Constitution Day", fixed(time.September, 1)},
		{"Day of Our Lady of Sorrows", fixed(time.September, 15)},
		{"All Saints' Day", fixed(time.November, 1)},
		{"Freedom and Democracy Day", fixed(time.November, 17)},
		{"Christmas Eve", fixed(time.December, 24)},
		{"Christmas Day", fixed(time.December, 25)},
		{"Saint Stephen's Day", fixed(time.December, 26)},
	}},
	"DE": {"Germany", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Good Friday", easterOff(-2)},
		{"Easter Monday", easterOff(1)},
		{"Labour Day", fixed(time.May, 1)},
		{"Ascension Day", easterOff(39)},
		{"Whit Monday", easterOff(50)},
		{"German Unity Day", fixed(time.October, 3)},
		{"Christmas Day", fixed(time.December, 25)},
		{"Second Day of Christmas", fixed(time.December, 26)},
	}},
	"AT": {"Austria", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Epiphany", fixed(time.January, 6)},
		{"Easter Monday", easterOff(1)},
		{"Labour Day", fixed(time.May, 1)},
		{"Ascension Day", easterOff(39)},
		{"Whit Monday", easterOff(50)},
		{"Corpus Christi", easterOff(60)},
		{"Assumption Day", fixed(time.August, 15)},
		{"National Day", fixed(time.October, 26)},
		{"All Saints' Day", fixed(time.November, 1)},
		{"Immaculate Conception", fixed(time.December, 8)},
		{"Christmas Day", fixed(time.December, 25)},
		{"Saint Stephen's Day", fixed(time.December, 26)},
	}},
	"CH": {"Switzerland", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Good Friday", easterOff(-2)},
		{"Easter Monday", easterOff(1)},
		{"Ascension Day", easterOff(39)},
		{"Whit Monday", easterOff(50)},
		{"Swiss National Day", fixed(time.August, 1)},
		{"Christmas Day", fixed(time.December, 25)},
		{"Saint Stephen's Day", fixed(time.December, 26)},
	}},
	"FR": {"France", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Easter Monday", easterOff(1)},
		{"Labour Day", fixed(time.May, 1)},
		{"Victory Day", fixed(time.May, 8)},
		{"Ascension Day", easterOff(39)},
		{"Whit Monday", easterOff(50)},
		{"Bastille Day", fixed(time.July, 14)},
		{"Assumption Day", fixed(time.August, 15)},
		{"All Saints' Day", fixed(time.November, 1)},
		{"Armistice Day", fixed(time.November, 11)},
		{"Christmas Day", fixed(time.December, 25)},
	}},
	"ES": {"Spain", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Epiphany", fixed(time.January, 6)},
		{"Good Friday", easterOff(-2)},
		{"Labour Day", fixed(time.May, 1)},
		{"Assumption Day", fixed(time.August, 15)},
		{"National Day", fixed(time.October, 12)},
		{"All Saints' Day", fixed(time.November, 1)},
		{"Constitution Day", fixed(time.December, 6)},
		{"Immaculate Conception", fixed(time.December, 8)},
		{"Christmas Day", fixed(time.December, 25)},
	}},
	"IT": {"Italy", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Epiphany", fixed(time.January, 6)},
		{"Easter Monday", easterOff(1)},
		{"Liberation Day", fixed(time.April, 25)},
		{"Labour Day", fixed(time.May, 1)},
		{"Republic Day", fixed(time.June, 2)},
		{"Assumption Day", fixed(time.August, 15)},
		{"All Saints' Day", fixed(time.November, 1)},
		{"Immaculate Conception", fixed(time.December, 8)},
		{"Christmas Day", fixed(time.December, 25)},
		{"Saint Stephen's Day", fixed(time.December, 26)},
	}},
	"PT": {"Portugal", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Good Friday", easterOff(-2)},
		{"Easter Sunday", easterOff(0)},
		{"Freedom Day", fixed(time.April, 25)},
		{"Labour Day", fixed(time.May, 1)},
		{"Corpus Christi", easterOff(60)},
		{"Portugal Day", fixed(time.June, 10)},
		{"Assumption Day", fixed(time.August, 15)},
		{"Republic Day", fixed(time.October, 5)},
		{"All Saints' Day", fixed(time.November, 1)},
		{"Restoration of Independence", fixed(time.December, 1)},
		{"Immaculate Conception", fixed(time.December, 8)},
		{"Christmas Day", fixed(time.December, 25)},
	}},
	"NL": {"Netherlands", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Good Friday", easterOff(-2)},
		{"Easter Sunday", easterOff(0)},
		{"Easter Monday", easterOff(1)},
		{"King's Day", fixed(time.April, 27)},
		{"Liberation Day", fixed(time.May, 5)},
		{"Ascension Day", easterOff(39)},
		{"Whit Sunday", easterOff(49)},
		{"Whit Monday", easterOff(50)},
		{"Christmas Day", fixed(time.December, 25)},
		{"Second Day of Christmas", fixed(time.December, 26)},
	}},
	"BE": {"Belgium", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Easter Monday", easterOff(1)},
		{"Labour Day", fixed(time.May, 1)},
		{"Ascension Day", easterOff(39)},
		{"Whit Monday", easterOff(50)},
		{"National Day", fixed(time.July, 21)},
		{"Assumption Day", fixed(time.August, 15)},
		{"All Saints' Day", fixed(time.November, 1)},
		{"Armistice Day", fixed(time.November, 11)},
		{"Christmas Day", fixed(time.December, 25)},
	}},
	"LU": {"Luxembourg", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Easter Monday", easterOff(1)},
		{"Labour Day", fixed(time.May, 1)},
		{"Europe Day", fixed(time.May, 9)},
		{"Ascension Day", easterOff(39)},
		{"Whit Monday", easterOff(50)},
		{"National Day", fixed(time.June, 23)},
		{"Assumption Day", fixed(time.August, 15)},
		{"All Saints' Day", fixed(time.November, 1)},
		{"Christmas Day", fixed(time.December, 25)},
		{"Saint Stephen's Day", fixed(time.December, 26)},
	}},
	"PL": {"Poland", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Epiphany", fixed(time.January, 6)},
		{"Easter Sunday", easterOff(0)},
		{"Easter Monday", easterOff(1)},
		{"Labour Day", fixed(time.May, 1)},
		{"Constitution Day", fixed(time.May, 3)},
		{"Whit Sunday", easterOff(49)},
		{"Corpus Christi", easterOff(60)},
		{"Assumption Day", fixed(time.August, 15)},
		{"All Saints' Day", fixed(time.November, 1)},
		{"Independence Day", fixed(time.November, 11)},
		{"Christmas Day", fixed(time.December, 25)},
		{"Second Day of Christmas", fixed(time.December, 26)},
	}},
	"HU": {"Hungary", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Revolution Day", fixed(time.March, 15)},
		{"Good Friday", easterOff(-2)},
		{"Easter Monday", easterOff(1)},
		{"Labour Day", fixed(time.May, 1)},
		{"Whit Monday", easterOff(50)},
		{"Saint Stephen's Day", fixed(time.August, 20)},
		{"Republic Day", fixed(time.October, 23)},
		{"All Saints' Day", fixed(time.November, 1)},
		{"Christmas Day", fixed(time.December, 25)},
		{"Second Day of Christmas", fixed(time.December, 26)},
	}},
	"SE": {"Sweden", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Epiphany", fixed(time.January, 6)},
		{"Good Friday", easterOff(-2)},
		{"Easter Sunday", easterOff(0)},
		{"Easter Monday", easterOff(1)},
		{"Labour Day", fixed(time.May, 1)},
		{"Ascension Day", easterOff(39)},
		{"National Day", fixed(time.June, 6)},
		{"Midsummer Day", weekdayOnOrAfter(time.June, 20, time.Saturday)},
		{"Christmas Day", fixed(time.December, 25)},
		{"Second Day of Christmas", fixed(time.December, 26)},
	}},
	"NO": {"Norway", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Maundy Thursday", easterOff(-3)},
		{"Good Friday", easterOff(-2)},
		{"Easter Sunday", easterOff(0)},
		{"Easter Monday", easterOff(1)},
		{"Labour Day", fixed(time.May, 1)},
		{"Constitution Day", fixed(time.May, 17)},
		{"Ascension Day", easterOff(39)},
		{"Whit Sunday", easterOff(49)},
		{"Whit Monday", easterOff(50)},
		{"Christmas Day", fixed(time.December, 25)},
		{"Second Day of Christmas", fixed(time.December, 26)},
	}},
	"DK": {"Denmark", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Maundy Thursday", easterOff(-3)},
		{"Good Friday", easterOff(-2)},
		{"Easter Sunday", easterOff(0)},
		{"Easter Monday", easterOff(1)},
		{"Ascension Day", easterOff(39)},
		{"Whit Sunday", easterOff(49)},
		{"Whit Monday", easterOff(50)},
		{"Constitution Day", fixed(time.June, 5)},
		{"Christmas Eve", fixed(time.December, 24)},
		{"Christmas Day", fixed(time.December, 25)},
		{"Second Day of Christmas", fixed(time.December, 26)},
	}},
	"FI": {"Finland", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Epiphany", fixed(time.January, 6)},
		{"Good Friday", easterOff(-2)},
		{"Easter Sunday", easterOff(0)},
		{"Easter Monday", easterOff(1)},
		{"May Day", fixed(time.May, 1)},
		{"Ascension Day", easterOff(39)},
		{"Midsummer Eve", weekdayOnOrAfter(time.June, 19, time.Friday)},
		{"Independence Day", fixed(time.December, 6)},
		{"Christmas Eve", fixed(time.December, 24)},
		{"Christmas Day", fixed(time.December, 25)},
		{"Saint Stephen's Day", fixed(time.December, 26)},
	}},
	"IS": {"Iceland", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Maundy Thursday", easterOff(-3)},
		{"Good Friday", easterOff(-2)},
		{"Easter Sunday", easterOff(0)},
		{"Easter Monday", easterOff(1)},
		{"First Day of Summer", weekdayOnOrAfter(time.April, 19, time.Thursday)},
		{"Labour Day", fixed(time.May, 1)},
		{"Ascension Day", easterOff(39)},
		{"Whit Sunday", easterOff(49)},
		{"Whit Monday", easterOff(50)},
		{"National Day", fixed(time.June, 17)},
		{"Commerce Day", nthWeekday(time.August, time.Monday, 1)},
		{"Christmas Eve", fixed(time.December, 24)},
		{"Christmas Day", fixed(time.December, 25)},
		{"Second Day of Christmas", fixed(time.December, 26)},
	}},
	"GR": {"Greece", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Epiphany", fixed(time.January, 6)},
		{"Clean Monday", orthodoxOff(-48)},
		{"Independence Day", fixed(time.March, 25)},
		{"Orthodox Easter Sunday", orthodoxOff(0)},
		{"Orthodox Easter Monday", orthodoxOff(1)},
		{"Labour Day", fixed(time.May, 1)},
		{"Assumption Day", fixed(time.August, 15)},
		{"Ochi Day", fixed(time.October, 28)},
		{"Christmas Day", fixed(time.December, 25)},
		{"Second Day of Christmas", fixed(time.December, 26)},
	}},
	"HR": {"Croatia", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Epiphany", fixed(time.January, 6)},
		{"Easter Sunday", easterOff(0)},
		{"Easter Monday", easterOff(1)},
		{"Labour Day", fixed(time.May, 1)},
		{"Statehood Day", fixed(time.May, 30)},
		{"Corpus Christi", easterOff(60)},
		{"Anti-Fascist Struggle Day", fixed(time.June, 22)},
		{"Victory Day", fixed(time.August, 5)},
		{"Assumption Day", fixed(time.August, 15)},
		{"All Saints' Day", fixed(time.November, 1)},
		{"Remembrance Day", fixed(time.November, 18)},
		{"Christmas Day", fixed(time.December, 25)},
		{"Saint Stephen's Day", fixed(time.December, 26)},
	}},
	"SI": {"Slovenia", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"New Year Holiday", fixed(time.January, 2)},
		{"Prešeren Day", fixed(time.February, 8)},
		{"Easter Sunday", easterOff(0)},
		{"Easter Monday", easterOff(1)},
		{"Resistance Day", fixed(time.April, 27)},
		{"Labour Day", fixed(time.May, 1)},
		{"Labour Day Holiday", fixed(time.May, 2)},
		{"Statehood Day", fixed(time.June, 25)},
		{"Assumption Day", fixed(time.August, 15)},
		{"Reformation Day", fixed(time.October, 31)},
		{"All Saints' Day", fixed(time.November, 1)},
		{"Christmas Day", fixed(time.December, 25)},
		{"Independence and Unity Day", fixed(time.December, 26)},
	}},
	"AU": {"Australia", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Australia Day", fixed(time.January, 26)},
		{"Good Friday", easterOff(-2)},
		{"Easter Saturday", easterOff(-1)},
		{"Easter Monday", easterOff(1)},
		{"Anzac Day", fixed(time.April, 25)},
		{"King's Birthday", nthWeekday(time.June, time.Monday, 2)},
		{"Christmas Day", fixed(time.December, 25)},
		{"Boxing Day", fixed(time.December, 26)},
	}},
	"NZ": {"New Zealand", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Day after New Year's Day", fixed(time.January, 2)},
		{"Waitangi Day", fixed(time.February, 6)},
		{"Good Friday", easterOff(-2)},
		{"Easter Monday", easterOff(1)},
		{"Anzac Day", fixed(time.April, 25)},
		{"King's Birthday", nthWeekday(time.June, time.Monday, 1)},
		{"Labour Day", nthWeekday(time.October, time.Monday, 4)},
		{"Christmas Day", fixed(time.December, 25)},
		{"Boxing Day", fixed(time.December, 26)},
	}},
	"JP": {"Japan", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Coming of Age Day", nthWeekday(time.January, time.Monday, 2)},
		{"National Foundation Day", fixed(time.February, 11)},
		{"Emperor's Birthday", fixed(time.February, 23)},
		{"Vernal Equinox Day", approxEquinox(20.8431)},
		{"Showa Day", fixed(time.April, 29)},
		{"Constitution Memorial Day", fixed(time.May, 3)},
		{"Greenery Day", fixed(time.May, 4)},
		{"Children's Day", fixed(time.May, 5)},
		{"Marine Day", nthWeekday(time.July, time.Monday, 3)},
		{"Mountain Day", fixed(time.August, 11)},
		{"Respect for the Aged Day", nthWeekday(time.September, time.Monday, 3)},
		{"Autumnal Equinox Day", approxEquinox(23.2488)},
		{"Sports Day", nthWeekday(time.October, time.Monday, 2)},
		{"Culture Day", fixed(time.November, 3)},
		{"Labour Thanksgiving Day", fixed(time.November, 23)},
	}},
	"IN": {"India", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Republic Day", fixed(time.January, 26)},
		{"Good Friday", easterOff(-2)},
		{"Labour Day", fixed(time.May, 1)},
		{"Independence Day", fixed(time.August, 15)},
		{"Gandhi Jayanti", fixed(time.October, 2)},
		{"Christmas Day", fixed(time.December, 25)},
	}},
	"BR": {"Brazil", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Carnival Monday", easterOff(-48)},
		{"Carnival Tuesday", easterOff(-47)},
		{"Good Friday", easterOff(-2)},
		{"Tiradentes Day", fixed(time.April, 21)},
		{"Labour Day", fixed(time.May, 1)},
		{"Corpus Christi", easterOff(60)},
		{"Independence Day", fixed(time.September, 7)},
		{"Our Lady of Aparecida", fixed(time.October, 12)},
		{"All Souls' Day", fixed(time.November, 2)},
		{"Republic Day", fixed(time.November, 15)},
		{"Christmas Day", fixed(time.December, 25)},
	}},
	"MX": {"Mexico", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Constitution Day", nthWeekday(time.February, time.Monday, 1)},
		{"Benito Juárez Day", nthWeekday(time.March, time.Monday, 3)},
		{"Labour Day", fixed(time.May, 1)},
		{"Independence Day", fixed(time.September, 16)},
		{"Revolution Day", nthWeekday(time.November, time.Monday, 3)},
		{"Day of the Virgin of Guadalupe", fixed(time.December, 12)},
		{"Christmas Day", fixed(time.December, 25)},
	}},
	"AR": {"Argentina", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Carnival Monday", easterOff(-48)},
		{"Carnival Tuesday", easterOff(-47)},
		{"Memorial Day", fixed(time.March, 24)},
		{"Malvinas Day", fixed(time.April, 2)},
		{"Good Friday", easterOff(-2)},
		{"Labour Day", fixed(time.May, 1)},
		{"May Revolution Day", fixed(time.May, 25)},
		{"Güemes Day", fixed(time.June, 17)},
		{"Belgrano Day", fixed(time.June, 20)},
		{"Independence Day", fixed(time.July, 9)},
		{"San Martín Day", nthWeekday(time.August, time.Monday, 3)},
		{"Respect for Cultural Diversity Day", nthWeekday(time.October, time.Monday, 2)},
		{"Immaculate Conception", fixed(time.December, 8)},
		{"Christmas Day", fixed(time.December, 25)},
	}},
	"CO": {"Colombia", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Epiphany", weekdayOnOrAfter(time.January, 6, time.Monday)},
		{"Saint Joseph's Day", weekdayOnOrAfter(time.March, 19, time.Monday)},
		{"Maundy Thursday", easterOff(-3)},
		{"Good Friday", easterOff(-2)},
		{"Labour Day", fixed(time.May, 1)},
		{"Ascension Day", easterOff(43)},
		{"Corpus Christi", easterOff(64)},
		{"Sacred Heart", easterOff(71)},
		{"Saints Peter and Paul", weekdayOnOrAfter(time.June, 29, time.Monday)},
		{"Independence Day", fixed(time.July, 20)},
		{"Battle of Boyacá", fixed(time.August, 7)},
		{"Assumption Day", weekdayOnOrAfter(time.August, 15, time.Monday)},
		{"Columbus Day", weekdayOnOrAfter(time.October, 12, time.Monday)},
		{"All Saints' Day", weekdayOnOrAfter(time.November, 1, time.Monday)},
		{"Cartagena Independence Day", fixed(time.November, 11)},
		{"Immaculate Conception", fixed(time.December, 8)},
		{"Christmas Day", fixed(time.December, 25)},
	}},
	"ZA": {"South Africa", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Human Rights Day", fixed(time.March, 21)},
		{"Good Friday", easterOff(-2)},
		{"Family Day", easterOff(1)},
		{"Freedom Day", fixed(time.April, 27)},
		{"Workers' Day", fixed(time.May, 1)},
		{"Youth Day", fixed(time.June, 16)},
		{"National Women's Day", fixed(time.August, 9)},
		{"Heritage Day", fixed(time.September, 24)},
		{"Day of Reconciliation", fixed(time.December, 16)},
		{"Christmas Day", fixed(time.December, 25)},
		{"Day of Goodwill", fixed(time.December, 26)},
	}},
	"UA": {"Ukraine", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Orthodox Christmas", fixed(time.January, 7)},
		{"International Women's Day", fixed(time.March, 8)},
		{"Orthodox Easter Sunday", orthodoxOff(0)},
		{"Orthodox Easter Monday", orthodoxOff(1)},
		{"Labour Day", fixed(time.May, 1)},
		{"Victory Day", fixed(time.May, 9)},
		{"Trinity Day", orthodoxOff(49)},
		{"Constitution Day", fixed(time.June, 28)},
		{"Independence Day", fixed(time.August, 24)},
		{"Defenders Day", fixed(time.October, 1)},
		{"Christmas Day", fixed(time.December, 25)},
	}},
	"RO": {"Romania", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"New Year Holiday", fixed(time.January, 2)},
		{"Unification Day", fixed(time.January, 24)},
		{"Orthodox Good Friday", orthodoxOff(-2)},
		{"Orthodox Easter Sunday", orthodoxOff(0)},
		{"Orthodox Easter Monday", orthodoxOff(1)},
		{"Labour Day", fixed(time.May, 1)},
		{"Children's Day", fixed(time.June, 1)},
		{"Orthodox Pentecost", orthodoxOff(49)},
		{"Orthodox Whit Monday", orthodoxOff(50)},
		{"Saint Mary's Day", fixed(time.August, 15)},
		{"Saint Andrew's Day", fixed(time.November, 30)},
		{"National Day", fixed(time.December, 1)},
		{"Christmas Day", fixed(time.December, 25)},
		{"Second Day of Christmas", fixed(time.December, 26)},
	}},
	"TR": {"Türkiye", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"National Sovereignty Day", fixed(time.April, 23)},
		{"Labour Day", fixed(time.May, 1)},
		{"Youth and Sports Day", fixed(time.May, 19)},
		{"Democracy Day", fixed(time.July, 15)},
		{"Victory Day", fixed(time.August, 30)},
		{"Republic Day", fixed(time.October, 29)},
	}},
	"RU": {"Russia", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Orthodox Christmas", fixed(time.January, 7)},
		{"Defender of the Fatherland Day", fixed(time.February, 23)},
		{"International Women's Day", fixed(time.March, 8)},
		{"Spring and Labour Day", fixed(time.May, 1)},
		{"Victory Day", fixed(time.May, 9)},
		{"Russia Day", fixed(time.June, 12)},
		{"Unity Day", fixed(time.November, 4)},
	}},
	"KR": {"South Korea", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Independence Movement Day", fixed(time.March, 1)},
		{"Children's Day", fixed(time.May, 5)},
		{"Memorial Day", fixed(time.June, 6)},
		{"Liberation Day", fixed(time.August, 15)},
		{"National Foundation Day", fixed(time.October, 3)},
		{"Hangul Day", fixed(time.October, 9)},
		{"Christmas Day", fixed(time.December, 25)},
	}},
	"TH": {"Thailand", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Chakri Day", fixed(time.April, 6)},
		{"Songkran", fixed(time.April, 13)},
		{"Songkran Holiday", fixed(time.April, 14)},
		{"Songkran Holiday", fixed(time.April, 15)},
		{"Labour Day", fixed(time.May, 1)},
		{"Coronation Day", fixed(time.May, 4)},
		{"King's Birthday", fixed(time.July, 28)},
		{"Queen Mother's Birthday", fixed(time.August, 12)},
		{"King Bhumibol Memorial Day", fixed(time.October, 13)},
		{"Chulalongkorn Day", fixed(time.October, 23)},
		{"King Bhumibol's Birthday", fixed(time.December, 5)},
		{"Constitution Day", fixed(time.December, 10)},
		{"New Year's Eve", fixed(time.December, 31)},
	}},
	"PH": {"Philippines", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Day of Valour", fixed(time.April, 9)},
		{"Good Friday", easterOff(-2)},
		{"Labour Day", fixed(time.May, 1)},
		{"Independence Day", fixed(time.June, 12)},
		{"National Heroes Day", lastWeekday(time.August, time.Monday)},
		{"Bonifacio Day", fixed(time.November, 30)},
		{"Christmas Day", fixed(time.December, 25)},
		{"Rizal Day", fixed(time.December, 30)},
	}},
	"ID": {"Indonesia", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Labour Day", fixed(time.May, 1)},
		{"Pancasila Day", fixed(time.June, 1)},
		{"Independence Day", fixed(time.August, 17)},
		{"Christmas Day", fixed(time.December, 25)},
	}},
	"MY": {"Malaysia", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Labour Day", fixed(time.May, 1)},
		{"National Day", fixed(time.August, 31)},
		{"Malaysia Day", fixed(time.September, 16)},
		{"Christmas Day", fixed(time.December, 25)},
	}},
	"SG": {"Singapore", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Labour Day", fixed(time.May, 1)},
		{"National Day", fixed(time.August, 9)},
		{"Christmas Day", fixed(time.December, 25)},
	}},
	"IL": {"Israel", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Christmas Day", fixed(time.December, 25)},
	}},
	"CN": {"China", []holidayDef{
		{"New Year's Day", fixed(time.January, 1)},
		{"Labour Day", fixed(time.May, 1)},
		{"National Day", fixed(time.October, 1)},
		{"National Day Holiday", fixed(time.October, 2)},
		{"National Day Holiday", fixed(time.October, 3)},
	}},
}
