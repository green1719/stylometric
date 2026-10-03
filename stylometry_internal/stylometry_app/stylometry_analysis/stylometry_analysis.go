package analysis

import (
	"math"
	"regexp"
	"strings"
)

// StylometrySampleFragment описывает один фрагмент текста классика.
type StylometrySampleFragment struct {
	StylometrySampleKey        string
	StylometrySampleLabel      string
	StylometrySampleAuthorName string
	StylometrySampleText       string
}

// StylometrySampleMetrics хранит рассчитанные стилометрические параметры.
type StylometrySampleMetrics struct {
	StylometrySampleAvgSentLen float64
	StylometrySampleConjFreq   float64
}

var stylometrySampleClassicFragments = [...]StylometrySampleFragment{
	{
		StylometrySampleKey:        "tolstoy",
		StylometrySampleLabel:      "Л. Н. Толстой",
		StylometrySampleAuthorName: "Л. Н. Толстой",
		StylometrySampleText:       "Всё смешалось в доме Облонских. Жена узнала, что муж был в связи с бывшею в их доме гувернанткой, и объявила мужу, что не может жить с ним в одном доме.",
	},
	{
		StylometrySampleKey:        "chekhov",
		StylometrySampleLabel:      "А. П. Чехов",
		StylometrySampleAuthorName: "А. П. Чехов",
		StylometrySampleText:       "В приёмной у земского доктора сидело много народу. День был душный, больные говорили вполголоса, и за окнами лениво пылила дорога.",
	},
	{
		StylometrySampleKey:        "dostoevsky",
		StylometrySampleLabel:      "Ф. М. Достоевский",
		StylometrySampleAuthorName: "Ф. М. Достоевский",
		StylometrySampleText:       "В начале июля, в чрезвычайно жаркое время, под вечер, один молодой человек вышел из своей каморки и медленно, как бы в нерешимости, отправился к мосту.",
	},
	{
		StylometrySampleKey:        "turgenev",
		StylometrySampleLabel:      "И. С. Тургенев",
		StylometrySampleAuthorName: "И. С. Тургенев",
		StylometrySampleText:       "Вечер был тихий и ясный; солнце садилось, и последние лучи его мягко золотили верхушки деревьев, между которыми тянулась широкая дорога.",
	},
	{
		StylometrySampleKey:        "pushkin",
		StylometrySampleLabel:      "А. С. Пушкин",
		StylometrySampleAuthorName: "А. С. Пушкин",
		StylometrySampleText:       "Мой дядя самых честных правил, когда не в шутку занемог, он уважать себя заставил и лучше выдумать не мог.",
	},
}

var stylometrySampleWordRegexp = regexp.MustCompile(`[\p{L}\p{N}]+`)

var stylometrySampleSentenceRegexp = regexp.MustCompile(`[.!?…]+`)

var stylometrySampleConjunctions = map[string]struct{}{
	"и": {}, "а": {}, "но": {}, "да": {}, "или": {}, "либо": {}, "что": {}, "чтобы": {},
	"как": {}, "когда": {}, "если": {}, "хотя": {}, "пока": {}, "будто": {}, "словно": {},
	"потому": {}, "поэтому": {}, "однако": {}, "зато": {}, "причём": {}, "притом": {},
}

// ClassicFragments возвращает список фрагментов для HTML-формы.
func ClassicFragments() []StylometrySampleFragment {
	return stylometrySampleClassicFragments[:]
}

// ResolveFragment выбирает пользовательский фрагмент или один из сохранённых фрагментов классиков.
func ResolveFragment(stylometry_title string, stylometry_key string, stylometry_custom_text string) StylometrySampleFragment {
	stylometry_text := strings.TrimSpace(stylometry_custom_text)
	if stylometry_text != "" {
		return StylometrySampleFragment{
			StylometrySampleKey:        "custom",
			StylometrySampleLabel:      "Свой фрагмент",
			StylometrySampleAuthorName: "Не указан",
			StylometrySampleText:       stylometry_text,
		}
	}

	if stylometry_fragment, stylometry_ok := FragmentByKey(stylometry_key); stylometry_ok {
		return stylometry_fragment
	}

	stylometry_title_lower := strings.ToLower(stylometry_title)
	for _, stylometry_fragment := range stylometrySampleClassicFragments {
		stylometry_author_lower := strings.ToLower(stylometry_fragment.StylometrySampleAuthorName)
		if strings.Contains(stylometry_title_lower, stylometry_fragment.StylometrySampleKey) || strings.Contains(stylometry_title_lower, stylometry_author_lower) {
			return stylometry_fragment
		}
	}

	return stylometrySampleClassicFragments[0]
}

// FragmentByKey ищет сохранённый фрагмент по ключу.
func FragmentByKey(stylometry_key string) (StylometrySampleFragment, bool) {
	stylometry_normalized_key := strings.TrimSpace(strings.ToLower(stylometry_key))
	for _, stylometry_fragment := range stylometrySampleClassicFragments {
		if stylometry_fragment.StylometrySampleKey == stylometry_normalized_key {
			return stylometry_fragment, true
		}
	}
	return StylometrySampleFragment{}, false
}

// AnalyzeFragment считает среднюю длину предложения и частоту союзов.
func AnalyzeFragment(stylometry_text string) StylometrySampleMetrics {
	stylometry_words := stylometrySampleWordRegexp.FindAllString(strings.ToLower(stylometry_text), -1)
	stylometry_word_count := len(stylometry_words)
	if stylometry_word_count == 0 {
		return StylometrySampleMetrics{}
	}

	stylometry_sentence_count := 0
	for _, stylometry_sentence := range stylometrySampleSentenceRegexp.Split(stylometry_text, -1) {
		if strings.TrimSpace(stylometry_sentence) != "" {
			stylometry_sentence_count++
		}
	}
	if stylometry_sentence_count == 0 {
		stylometry_sentence_count = 1
	}

	stylometry_conjunction_count := 0
	for _, stylometry_word := range stylometry_words {
		if _, stylometry_ok := stylometrySampleConjunctions[stylometry_word]; stylometry_ok {
			stylometry_conjunction_count++
		}
	}

	stylometry_avg_len := float64(stylometry_word_count) / float64(stylometry_sentence_count)
	stylometry_conj_freq := float64(stylometry_conjunction_count) / float64(stylometry_word_count) * 100

	return StylometrySampleMetrics{
		StylometrySampleAvgSentLen: stylometrySampleRoundOne(stylometry_avg_len),
		StylometrySampleConjFreq:   stylometrySampleRoundOne(stylometry_conj_freq),
	}
}

// stylometrySampleRoundOne округляет число до одного знака после запятой.
func stylometrySampleRoundOne(stylometry_value float64) float64 {
	return math.Round(stylometry_value*10) / 10
}
