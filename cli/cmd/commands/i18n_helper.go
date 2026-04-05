package commands

import "github.com/icosmos-space/ipen/cli/cmd/i18n"

// T is shorthand for i18n.T
func T(m map[i18n.Language]string) string { return i18n.T(m) }

// TR is shorthand for i18n.Translations
var TR = i18n.Translations
