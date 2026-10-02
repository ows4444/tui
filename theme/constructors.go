package theme

// Preset constructors. Each XxxTheme returns a fresh copy of the preset of
// the same name; the presets themselves are unexported, so no caller can change
// the shared value. (Before v1.0 they were exported vars, Dark, Dracula and so
// on, which any package could modify.)

// DarkTheme returns a copy of the Dark preset.
func DarkTheme() Theme { return dark }

// LightTheme returns a copy of the Light preset.
func LightTheme() Theme { return light }

// DraculaTheme returns a copy of the Dracula preset.
func DraculaTheme() Theme { return dracula }

// NordTheme returns a copy of the Nord preset.
func NordTheme() Theme { return nord }

// GruvboxTheme returns a copy of the Gruvbox preset.
func GruvboxTheme() Theme { return gruvbox }

// TokyoNightTheme returns a copy of the TokyoNight preset.
func TokyoNightTheme() Theme { return tokyoNight }

// MonokaiTheme returns a copy of the Monokai preset.
func MonokaiTheme() Theme { return monokai }

// SolarizedDarkTheme returns a copy of the SolarizedDark preset.
func SolarizedDarkTheme() Theme { return solarizedDark }

// SolarizedLightTheme returns a copy of the SolarizedLight preset.
func SolarizedLightTheme() Theme { return solarizedLight }

// CatppuccinTheme returns a copy of the Catppuccin preset.
func CatppuccinTheme() Theme { return catppuccin }

// OneDarkTheme returns a copy of the OneDark preset.
func OneDarkTheme() Theme { return oneDark }

// NightOwlTheme returns a copy of the NightOwl preset.
func NightOwlTheme() Theme { return nightOwl }

// RosePineTheme returns a copy of the RosePine preset.
func RosePineTheme() Theme { return rosePine }

// EverforestDarkTheme returns a copy of the EverforestDark preset.
func EverforestDarkTheme() Theme { return everforestDark }

// GitHubDarkTheme returns a copy of the GitHubDark preset.
func GitHubDarkTheme() Theme { return githubDark }

// HighContrastTheme returns a copy of the HighContrast preset.
func HighContrastTheme() Theme { return highContrast }
