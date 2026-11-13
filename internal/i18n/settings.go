package i18n

// The settings menu (":settings").
const (
	SettingsTitle          Key = "settings.title"
	SettingAccordion       Key = "settings.accordion"
	SettingHideWidth       Key = "settings.hide_width"
	SettingLook            Key = "settings.look"
	SettingBorder          Key = "settings.border"
	SettingTransparent     Key = "settings.transparent"
	SettingHeadingNumbers  Key = "settings.heading_numbers"
	SettingLinkSuggestions Key = "settings.link_suggestions"
	SettingLanguage        Key = "settings.language"
	// SettingChanged is the bar's line after a change: the row's label and
	// its new value.
	SettingChanged        Key = "settings.changed"
	SettingOn             Key = "settings.on"
	SettingOff            Key = "settings.off"
	SettingHideWidthHelp  Key = "settings.hide_width_help"
	SettingHideWidthBad   Key = "settings.hide_width_bad"
	SettingSaveFailed     Key = "settings.save_failed"
	LookNumbers           Key = "settings.look.numbers"
	LookArrows            Key = "settings.look.arrows"
	LookPlain             Key = "settings.look.plain"
	LookCustom            Key = "settings.look.custom"
	LinkSuggestionsInline Key = "settings.link_suggestions.inline"
	LinkSuggestionsFinder Key = "settings.link_suggestions.finder"
	LanguageAuto          Key = "settings.language.auto"
)

func init() {
	register(EN, map[Key]string{
		SettingsTitle:          "Settings",
		SettingAccordion:       "Accordion folders",
		SettingHideWidth:       "Explorer hide width",
		SettingLook:            "Icons & bullets",
		SettingBorder:          "Dialog border",
		SettingTransparent:     "Transparent background",
		SettingHeadingNumbers:  "Heading numbers",
		SettingLinkSuggestions: "Link suggestions",
		SettingLanguage:        "Language",
		SettingChanged:         "%s: %s",
		SettingOn:              "on",
		SettingOff:             "off",
		SettingHideWidthHelp:   "The explorer hides below this many columns (default %d).",
		SettingHideWidthBad:    "Explorer hide width must be a whole number from 1 to %d",
		SettingSaveFailed:      "Could not save settings: %s",
		LookNumbers:            "Numbers",
		LookArrows:             "Arrows",
		LookPlain:              "Plain",
		LookCustom:             "custom",
		LinkSuggestionsInline:  "Inline",
		LinkSuggestionsFinder:  "Finder",
		LanguageAuto:           "Auto (system)",
	})
	register(ES, map[Key]string{
		SettingsTitle:          "Ajustes",
		SettingAccordion:       "Carpetas en acordeón",
		SettingHideWidth:       "Ancho para ocultar el explorador",
		SettingLook:            "Iconos y viñetas",
		SettingBorder:          "Borde de los diálogos",
		SettingTransparent:     "Fondo transparente",
		SettingHeadingNumbers:  "Numerar los títulos",
		SettingLinkSuggestions: "Sugerencias de enlaces",
		SettingLanguage:        "Idioma",
		SettingChanged:         "%s: %s",
		SettingOn:              "sí",
		SettingOff:             "no",
		SettingHideWidthHelp:   "El explorador se oculta con menos columnas que estas (por defecto %d).",
		SettingHideWidthBad:    "El ancho para ocultar el explorador debe ser un número entero de 1 a %d",
		SettingSaveFailed:      "No se pudieron guardar los ajustes: %s",
		LookNumbers:            "Números",
		LookArrows:             "Flechas",
		LookPlain:              "Simple",
		LookCustom:             "propio",
		LinkSuggestionsInline:  "En línea",
		LinkSuggestionsFinder:  "Buscador",
		LanguageAuto:           "Automático (sistema)",
	})
}
