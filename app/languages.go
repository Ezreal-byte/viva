package app

import "strings"

// PreferredLanguages lists the languages the user asked their system to speak,
// most preferred first, as BCP 47 tags such as "zh-Hans-CN" or "en-US". It is
// empty where the system does not say. An application uses it to pick the language
// of its own interface when the user has not chosen one: it follows the person's
// setting, which a time zone or a region does not (a Chinese time zone is common
// on an English system).
//
// macOS and Windows report the user's UI language list, the browser reports
// navigator.languages, and Linux reads LANGUAGE, LC_ALL, LC_MESSAGES and LANG.
func PreferredLanguages() []string { return normalizeLanguages(nativePreferredLanguages()) }

// normalizeLanguages turns whatever a platform reported into BCP 47 tags: POSIX
// names lose their encoding and modifier ("zh_CN.UTF-8" is "zh-CN"), the C and
// POSIX locales say nothing, and a tag appears once, where it first did.
func normalizeLanguages(reported []string) []string {
	var tags []string
	seen := make(map[string]bool, len(reported))
	for _, name := range reported {
		name = strings.TrimSpace(name)
		if cut := strings.IndexAny(name, ".@"); cut >= 0 {
			name = name[:cut]
		}
		name = strings.ReplaceAll(name, "_", "-")
		if name == "" || name == "C" || name == "POSIX" || seen[strings.ToLower(name)] {
			continue
		}
		seen[strings.ToLower(name)] = true
		tags = append(tags, name)
	}
	return tags
}

// languagesFromEnvironment is the POSIX answer: the LANGUAGE list when there is
// one, else the first locale variable that is set, in the order libc consults them.
func languagesFromEnvironment(getenv func(string) string) []string {
	var reported []string
	if list := getenv("LANGUAGE"); list != "" {
		reported = strings.Split(list, ":")
	}
	for _, variable := range [...]string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if value := getenv(variable); value != "" {
			reported = append(reported, value)
			break
		}
	}
	return reported
}
