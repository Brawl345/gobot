package weather

type Weathercode int

func (weathercode Weathercode) Icon() string {
	if weathercode == 0 {
		return "☀"
	} else if weathercode <= 2 {
		return "⛅"
	} else if weathercode == 3 {
		return "☁"
	} else if weathercode <= 9 {
		return "🌫"
	} else if weathercode <= 19 {
		return "⛅"
	} else if weathercode <= 21 {
		return "☂"
	} else if weathercode <= 24 {
		return "❄"
	} else if weathercode == 25 {
		return "☔"
	} else if weathercode == 26 {
		return "❄"
	} else if weathercode <= 28 {
		return "🌫"
	} else if weathercode == 29 {
		return "🌩"
	} else if weathercode <= 39 {
		return "💨"
	} else if weathercode <= 49 {
		return "🌫"
	} else if weathercode <= 69 {
		return "☔"
	} else if weathercode <= 79 {
		return "❄"
	} else if weathercode <= 90 {
		return "☔"
	} else if weathercode <= 99 {
		return "⛈"
	}
	return "🤔"
}

func (weathercode Weathercode) Description() string {
	// https://wetterkanal.kachelmannwetter.com/was-ist-der-ww-code-in-der-meteorologie/
	// https://www.meteopool.org/de/encyclopedia-wmo-ww-wx-code-id2
	switch weathercode {
	case 0:
		return "Nicht bewölkt"
	case 1:
		return "Bewölkung abnehmend"
	case 2:
		return "Bewölkung unverändert"
	case 3:
		return "Bewölkung zunehmend"
	case 4:
		return "Sicht durch Rauch oder Asche vermindert"
	case 5:
		return "trockener Dunst (relative Feuchte < 80 %)"
	case 6:
		return "verbreiteter Schwebstaub, nicht vom Wind herangeführt"
	case 7:
		return "Staub oder Sand bzw. Gischt, vom Wind herangeführt"
	case 8:
		return "gut entwickelte Staub- oder Sandwirbel"
	case 9:
		return "Staub- oder Sandsturm im Gesichtskreis, aber nicht an der Station"
	case 10:
		return "feuchter Dunst (relative Feuchte > 80 %)"
	case 11:
		return "Schwaden von Bodennebel"
	case 12:
		return "durchgehender Bodennebel"
	case 13:
		return "Wetterleuchten sichtbar, kein Donner gehört"
	case 14:
		return "Niederschlag im Gesichtskreis, nicht den Boden erreichend"
	case 15:
		return "Niederschlag in der Ferne (> 5 km), aber nicht an der Station"
	case 16:
		return "Niederschlag in der Nähe (< 5 km), aber nicht an der Station"
	case 17:
		return "Gewitter (Donner hörbar), aber kein Niederschlag an der Station"
	case 18:
		return "Markante Böen im Gesichtskreis, aber kein Niederschlag an der Station"
	case 19:
		return "Tromben (trichterförmige Wolkenschläuche) im Gesichtskreis"
	case 20:
		return "nach Sprühregen oder Schneegriesel"
	case 21:
		return "nach Regen"
	case 22:
		return "nach Schneefall"
	case 23:
		return "nach Schneeregen oder Eiskörnern"
	case 24:
		return "nach gefrierendem Regen"
	case 25:
		return "nach Regenschauer"
	case 26:
		return "nach Schneeschauer"
	case 27:
		return "nach Graupel- oder Hagelschauer"
	case 28:
		return "nach Nebel"
	case 29:
		return "nach Gewitter"
	case 30:
		return "leichter oder mäßiger Sandsturm, an Intensität abnehmend"
	case 31:
		return "leichter oder mäßiger Sandsturm, unveränderte Intensität"
	case 32:
		return "leichter oder mäßiger Sandsturm, an Intensität zunehmend"
	case 33:
		return "schwerer Sandsturm, an Intensität abnehmend"
	case 34:
		return "schwerer Sandsturm, unveränderte Intensität"
	case 35:
		return "schwerer Sandsturm, an Intensität zunehmend"
	case 36:
		return "leichtes oder mäßiges Schneefegen, unter Augenhöhe"
	case 37:
		return "starkes Schneefegen, unter Augenhöhe"
	case 38:
		return "leichtes oder mäßiges Schneetreiben, über Augenhöhe"
	case 39:
		return "starkes Schneetreiben, über Augenhöhe"
	case 40:
		return "Nebel in einiger Entfernung"
	case 41:
		return "Nebel in Schwaden oder Bänken"
	case 42:
		return "Nebel, Himmel erkennbar, dünner werdend"
	case 43:
		return "Nebel, Himmel nicht erkennbar, dünner werdend"
	case 44:
		return "Nebel, Himmel erkennbar, unverändert"
	case 45:
		return "Nebel, Himmel nicht erkennbar, unverändert"
	case 46:
		return "Nebel, Himmel erkennbar, dichter werdend"
	case 47:
		return "Nebel, Himmel nicht erkennbar, dichter werdend"
	case 48:
		return "Nebel mit Reifansatz, Himmel erkennbar"
	case 49:
		return "Nebel mit Reifansatz, Himmel nicht erkennbar"
	case 50:
		return "unterbrochener leichter Sprühregen"
	case 51:
		return "durchgehend leichter Sprühregen"
	case 52:
		return "unterbrochener mäßiger Sprühregen"
	case 53:
		return "durchgehend mäßiger Sprühregen"
	case 54:
		return "unterbrochener starker Sprühregen"
	case 55:
		return "durchgehend starker Sprühregen"
	case 56:
		return "leichter gefrierender Sprühregen"
	case 57:
		return "mäßiger oder starker gefrierender Sprühregen"
	case 58:
		return "leichter Sprühregen mit Regen"
	case 59:
		return "mäßiger oder starker Sprühregen mit Regen"
	case 60:
		return "unterbrochener leichter Regen oder einzelne Regentropfen"
	case 61:
		return "durchgehend leichter Regen"
	case 62:
		return "unterbrochener mäßiger Regen"
	case 63:
		return "durchgehend mäßiger Regen"
	case 64:
		return "unterbrochener starker Regen"
	case 65:
		return "durchgehend starker Regen"
	case 66:
		return "leichter gefrierender Regen"
	case 67:
		return "mäßiger oder starker gefrierender Regen"
	case 68:
		return "leichter Schneeregen"
	case 69:
		return "mäßiger oder starker Schneeregen"
	case 70:
		return "unterbrochener leichter Schneefall oder einzelne Schneeflocken"
	case 71:
		return "durchgehend leichter Schneefall"
	case 72:
		return "unterbrochener mäßiger Schneefall"
	case 73:
		return "durchgehend mäßiger Schneefall"
	case 74:
		return "unterbrochener starker Schneefall"
	case 75:
		return "durchgehend starker Schneefall"
	case 76:
		return "Eisnadeln (Polarschnee)"
	case 77:
		return "Schneegriesel"
	case 78:
		return "Schneekristalle"
	case 79:
		return "Eiskörner (gefrorene Regentropfen)"
	case 80:
		return "leichter Regenschauer"
	case 81:
		return "mäßiger oder starker Regenschauer"
	case 82:
		return "äußerst heftiger Regenschauer"
	case 83:
		return "leichter Schneeregenschauer"
	case 84:
		return "mäßiger oder starker Schneeregenschauer"
	case 85:
		return "leichter Schneeschauer"
	case 86:
		return "mäßiger oder starker Schneeschauer"
	case 87:
		return "leichter Graupelschauer"
	case 88:
		return "mäßiger oder starker Graupelschauer"
	case 89:
		return "leichter Hagelschauer"
	case 90:
		return "mäßiger oder starker Hagelschauer"
	case 91:
		return "Gewitter in der letzten Stunde, zurzeit leichter Regen"
	case 92:
		return "Gewitter in der letzten Stunde, zurzeit mäßiger oder starker Regen"
	case 93:
		return "Gewitter in der letzten Stunde, zurzeit leichter Schneefall/Schneeregen/Graupel/Hagel"
	case 94:
		return "Gewitter in der letzten Stunde, zurzeit mäßiger oder starker Schneefall/Schneeregen/Graupel/Hagel"
	case 95:
		return "leichtes oder mäßiges Gewitter mit Regen oder Schnee"
	case 96:
		return "leichtes oder mäßiges Gewitter mit Graupel oder Hagel"
	case 97:
		return "starkes Gewitter mit Regen oder Schnee"
	case 98:
		return "starkes Gewitter mit Sandsturm"
	case 99:
		return "starkes Gewitter mit Graupel oder Hagel"
	default:
		return "Unbekannt"
	}
}
