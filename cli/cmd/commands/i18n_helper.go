package commands

import "github.com/icosmos-space/ipen/cli/cmd/i18n"

// T i18n.T缩写
func T(m map[i18n.Language]string) string { return i18n.T(m) }

// TR i18n.Translations缩写
var TR = i18n.Translations
