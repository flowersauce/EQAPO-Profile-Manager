package i18n

import "testing"

func TestWindowsLanguageMapping(t *testing.T) {
	for _, locale := range []string{"zh-CN", "zh-TW", "zh-HK", "zh-Hans", "ZH_cn"} {
		if !ForLocale(locale).Chinese {
			t.Errorf("not Chinese: %s", locale)
		}
	}
	for _, locale := range []string{"en-US", "de-DE", "", "ja-JP"} {
		if ForLocale(locale).Chinese {
			t.Errorf("did not fall back to English: %s", locale)
		}
	}
	for key, pair := range messages {
		if pair[0] == "" || pair[1] == "" {
			t.Errorf("missing translation: %s", key)
		}
	}
}
