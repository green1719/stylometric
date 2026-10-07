package analysis

import (
	"math"
	"regexp"
	"strings"
)

// StylometrySampleMetrics хранит рассчитанные стилометрические параметры.
type StylometrySampleMetrics struct {
	StylometrySampleAverageSentenceLength float64
	StylometrySampleConjunctionFrequency  float64
}

var stylometrySampleSeedTexts = map[string]string{
	"tolstoy":    "Всё смешалось в доме Облонских. Жена узнала, что муж был в связи с бывшею в их доме гувернанткой, и объявила мужу, что не может жить с ним в одном доме.",
	"chekhov":    "В приёмной у земского доктора сидело много народу. День был душный, больные говорили вполголоса, и за окнами лениво пылила дорога.",
	"dostoevsky": "В начале июля, в чрезвычайно жаркое время, под вечер, один молодой человек вышел из своей каморки и медленно, как бы в нерешимости, отправился к мосту.",
	"turgenev":   "Вечер был тихий и ясный; солнце садилось, и последние лучи его мягко золотили верхушки деревьев, между которыми тянулась широкая дорога.",
	"pushkin":    "Мой дядя самых честных правил, когда не в шутку занемог, он уважать себя заставил и лучше выдумать не мог.",
	"gogol":      "Я пригласил вас, господа, с тем чтобы сообщить вам пренеприятное известие: к нам едет ревизор.",
	"lermontov":  "Белеет парус одинокий в тумане моря голубом. Что ищет он в стране далёкой, что кинул он в краю родном?",
	"kuprin":     "Гранатовый браслет был золотой, низкопробный, с плохо отшлифованными гранатами, но Вера сразу узнала его.",
	"bunyn":      "Тёмные аллеи стояли передо мной, и в прохладной тени старых лип вспоминалось всё, что давно минуло.",
}

var stylometrySampleWordRegexp = regexp.MustCompile(`[\p{L}\p{N}]+`)

var stylometrySampleSentenceRegexp = regexp.MustCompile(`[.!?…]+`)

var stylometrySampleConjunctions = map[string]struct{}{
	"и": {}, "а": {}, "но": {}, "да": {}, "или": {}, "либо": {}, "что": {}, "чтобы": {},
	"как": {}, "когда": {}, "если": {}, "хотя": {}, "пока": {}, "будто": {}, "словно": {},
	"потому": {}, "поэтому": {}, "однако": {}, "зато": {}, "причём": {}, "притом": {},
}

// SeedTextByKey возвращает стартовый текст для начального наполнения базы.
func SeedTextByKey(stylometry_key string) (string, bool) {
	stylometry_normalized_key := strings.TrimSpace(strings.ToLower(stylometry_key))
	stylometry_text, stylometry_ok := stylometrySampleSeedTexts[stylometry_normalized_key]
	return stylometry_text, stylometry_ok
}

// AnalyzeText считает среднюю длину предложения и частоту союзов.
func AnalyzeText(stylometry_text string) StylometrySampleMetrics {
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
		StylometrySampleAverageSentenceLength: stylometrySampleRoundOne(stylometry_avg_len),
		StylometrySampleConjunctionFrequency:  stylometrySampleRoundOne(stylometry_conj_freq),
	}
}

// stylometrySampleRoundOne округляет число до одного знака после запятой.
func stylometrySampleRoundOne(stylometry_value float64) float64 {
	return math.Round(stylometry_value*10) / 10
}
