package handler

import "testing"

func TestStylometrySplitDraftText(t *testing.T) {
	stylometry_author, stylometry_text, stylometry_err := stylometrySplitDraftText("  Лев Толстой  \r\nПервая строка.\r\nВторая строка.")
	if stylometry_err != nil {
		t.Fatalf("неожиданная ошибка: %v", stylometry_err)
	}
	if stylometry_author != "Лев Толстой" {
		t.Fatalf("автор = %q, ожидался %q", stylometry_author, "Лев Толстой")
	}
	if stylometry_text != "Первая строка.\nВторая строка." {
		t.Fatalf("текст = %q", stylometry_text)
	}
}

func TestStylometrySplitDraftTextRejectsMissingPart(t *testing.T) {
	for _, stylometry_input := range []string{"Лев Толстой", "\nТекст", "Лев Толстой\n   "} {
		if _, _, stylometry_err := stylometrySplitDraftText(stylometry_input); stylometry_err == nil {
			t.Fatalf("ожидалась ошибка для %q", stylometry_input)
		}
	}
}
