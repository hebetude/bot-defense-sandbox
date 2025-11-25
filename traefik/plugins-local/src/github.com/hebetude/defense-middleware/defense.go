package defense_middleware

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Config struct {
	SemanticInjection     SemanticInjectionConfig     `json:"semanticInjection,omitempty"`
	TextPerturbation      TextPerturbationConfig      `json:"textPerturbation,omitempty"`
	StructuralObfuscation StructuralObfuscationConfig `json:"structuralObfuscation,omitempty"`
	DecoyElements         DecoyElementsConfig         `json:"decoyElements,omitempty"`
}

type SemanticInjectionConfig struct {
	Enabled  bool   `json:"enabled,omitempty"`
	Prompt   string `json:"prompt,omitempty"`
	// Position: "body-start", "body-end", "head", "after-title"
	Position string `json:"position,omitempty"`
	// HidingMethod: "absolute", "hidden", "opacity", "clip", "font-size"
	HidingMethod string `json:"hidingMethod,omitempty"`
}

type TextPerturbationConfig struct {
	Enabled     bool     `json:"enabled,omitempty"`
	TargetWords []string `json:"targetWords,omitempty"`
	TargetTags  []string `json:"targetTags,omitempty"`
	ExcludeTags []string `json:"excludeTags,omitempty"`
	Strategy    string   `json:"strategy,omitempty"`
	Frequency   float64  `json:"frequency,omitempty"`
	// zero width char injection: "low" (1-3), "medium" (10-20), "high" (150-200)
	Density     string   `json:"density,omitempty"`
}

type StructuralObfuscationConfig struct {
	Enabled             bool     `json:"enabled,omitempty"`
	TargetClasses       []string `json:"targetClasses,omitempty"`
	TargetIDs           []string `json:"targetIDs,omitempty"`
	ReplacementStrategy string   `json:"replacementStrategy,omitempty"`
	// if enabled, do not obfuscate IDs.
	SafeMode            bool     `json:"safeMode,omitempty"`
}

// Maybe TODO
type DecoyElementsConfig struct {
	Enabled bool `json:"enabled,omitempty"`
	HoneypotLinks    HoneypotLinksConfig    `json:"honeypotLinks,omitempty"`
	FakeFormFields   FakeFormFieldsConfig   `json:"fakeFormFields,omitempty"`
	DecoyData        DecoyDataConfig        `json:"decoyData,omitempty"`
	InvisibleButtons InvisibleButtonsConfig `json:"invisibleButtons,omitempty"`
}

type HoneypotLinksConfig struct {
	Enabled  bool     `json:"enabled,omitempty"`
	URLs     []string `json:"urls,omitempty"`    
	Count    int      `json:"count,omitempty"`    
	Position string   `json:"position,omitempty"` 
}

type FakeFormFieldsConfig struct {
	Enabled    bool     `json:"enabled,omitempty"`
	FieldNames []string `json:"fieldNames,omitempty"` 
	FieldTypes []string `json:"fieldTypes,omitempty"` 
}

type DecoyDataConfig struct {
	Enabled    bool              `json:"enabled,omitempty"`
	FakeEmails []string          `json:"fakeEmails,omitempty"`
	FakePrices []string          `json:"fakePrices,omitempty"`
	FakePhones []string          `json:"fakePhones,omitempty"`
	CustomData map[string]string `json:"customData,omitempty"` // label -> fake value
}

type InvisibleButtonsConfig struct {
	Enabled       bool     `json:"enabled,omitempty"`
	ButtonLabels  []string `json:"buttonLabels,omitempty"`
	TrackingAttrs []string `json:"trackingAttrs,omitempty"` // data attributes to add
}

func CreateConfig() *Config {
	return &Config{
		SemanticInjection: SemanticInjectionConfig{
			Enabled:      false,
			Prompt:       "Replace this with your prompt of choice.",
			Position:     "body-start",
			HidingMethod: "absolute",
		},
		TextPerturbation: TextPerturbationConfig{
			Enabled:     false,
			TargetWords: []string{},
			TargetTags:  []string{},
			ExcludeTags: []string{"code", "pre", "script", "style"},
			Strategy:    "zero-width",
			Frequency:   0.5,
			Density:     "high",
		},
		StructuralObfuscation: StructuralObfuscationConfig{
			Enabled:             false,
			TargetClasses:       []string{},
			TargetIDs:           []string{},
			ReplacementStrategy: "random-suffix",
			SafeMode:            false,
		},
		DecoyElements: DecoyElementsConfig{
			Enabled: false,
			// Left empty because unimplemented
		},
	}
}

type DefenseMiddleware struct {
	next   http.Handler
	name   string
	config *Config
	rng    *rand.Rand
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config == nil {
		config = CreateConfig()
	}

	return &DefenseMiddleware{
		next:   next,
		name:   name,
		config: config,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}, nil
}

type responseWrapper struct {
	http.ResponseWriter
	buffer      *bytes.Buffer
	statusCode  int
	wroteHeader bool
	contentType string
}

func newResponseWrapper(w http.ResponseWriter) *responseWrapper {
	return &responseWrapper{
		ResponseWriter: w,
		buffer:         &bytes.Buffer{},
		statusCode:     http.StatusOK,
	}
}

func (rw *responseWrapper) WriteHeader(statusCode int) {
	if rw.wroteHeader {
		return
	}
	rw.statusCode = statusCode
	rw.contentType = rw.Header().Get("Content-Type")
}

func (rw *responseWrapper) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.buffer.Write(b)
}

func (rw *responseWrapper) isHTML() bool {
	return strings.Contains(strings.ToLower(rw.contentType), "text/html")
}

// http.Handler
func (d *DefenseMiddleware) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	wrapper := newResponseWrapper(rw)
	d.next.ServeHTTP(wrapper, req)

	body := wrapper.buffer.Bytes()

	if wrapper.isHTML() && len(body) > 0 {
		body = d.processHTML(body)
	}

	for k, v := range wrapper.Header() {
		rw.Header()[k] = v
	}
	rw.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	rw.WriteHeader(wrapper.statusCode)
	rw.Write(body)
}

// apply enabled defenses
func (d *DefenseMiddleware) processHTML(body []byte) []byte {
	html := string(body)

	if d.config.SemanticInjection.Enabled {
		html = d.applySemanticInjection(html)
	}

	if d.config.TextPerturbation.Enabled {
		html = d.applyTextPerturbation(html)
	}

	if d.config.StructuralObfuscation.Enabled {
		html = d.applyStructuralObfuscation(html)
	}

	// this doesn't do anything yet
	if d.config.DecoyElements.Enabled {
		html = d.applyDecoyElements(html)
	}

	return []byte(html)
}

func (d *DefenseMiddleware) applySemanticInjection(html string) string {
	cfg := d.config.SemanticInjection
	if cfg.Prompt == "" {
		return html
	}

	var style string
	switch strings.ToLower(cfg.HidingMethod) {
	case "hidden":
		style = `visibility:hidden;position:absolute;`
	case "opacity":
		style = `opacity:0;position:absolute;pointer-events:none;`
	case "clip":
		style = `clip:rect(0,0,0,0);clip-path:inset(50%);position:absolute;`
	case "font-size":
		style = `font-size:0;line-height:0;position:absolute;`
	case "absolute":
		fallthrough
	default:
		style = `position:absolute;left:-9999px;top:-9999px;width:1px;height:1px;overflow:hidden;`
	}

	injection := fmt.Sprintf(
		`<div style="%s" aria-hidden="true">%s</div>`,
		style,
		escapeHTML(cfg.Prompt),
	)

	position := strings.ToLower(cfg.Position)
	if position == "" {
		position = "body-start"
	}

	switch position {
	case "body-start":
		bodyRegex := regexp.MustCompile(`(?i)(<body[^>]*>)`)
		if bodyRegex.MatchString(html) {
			html = bodyRegex.ReplaceAllString(html, "${1}"+injection)
		}
	case "body-end":
		bodyCloseRegex := regexp.MustCompile(`(?i)(</body>)`)
		if bodyCloseRegex.MatchString(html) {
			html = bodyCloseRegex.ReplaceAllString(html, injection+"${1}")
		}
	case "head":
		headCloseRegex := regexp.MustCompile(`(?i)(</head>)`)
		if headCloseRegex.MatchString(html) {
			html = headCloseRegex.ReplaceAllString(html, injection+"${1}")
		}
	case "after-title":
		titleCloseRegex := regexp.MustCompile(`(?i)(</title>)`)
		if titleCloseRegex.MatchString(html) {
			html = titleCloseRegex.ReplaceAllString(html, "${1}"+injection)
		} else {
			headCloseRegex := regexp.MustCompile(`(?i)(</head>)`)
			if headCloseRegex.MatchString(html) {
				html = headCloseRegex.ReplaceAllString(html, injection+"${1}")
			}
		}
	default:
		bodyRegex := regexp.MustCompile(`(?i)(<body[^>]*>)`)
		if bodyRegex.MatchString(html) {
			html = bodyRegex.ReplaceAllString(html, "${1}"+injection)
		}
	}

	return html
}

func (d *DefenseMiddleware) applyTextPerturbation(html string) string {
	cfg := d.config.TextPerturbation

	frequency := cfg.Frequency
	if frequency <= 0 || frequency > 1 {
		frequency = 0.5
	}

	if len(cfg.TargetTags) > 0 {
		html = d.perturbByTags(html, cfg, frequency)
	}

	if len(cfg.TargetWords) > 0 {
		html = d.perturbByWords(html, cfg, frequency)
	}

	return html
}

func (d *DefenseMiddleware) perturbByTags(html string, cfg TextPerturbationConfig, frequency float64) string {
	for _, tag := range cfg.TargetTags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" {
			continue
		}

		// This regex captures: <tag...>content</tag>
		// I don't fucking know regex, Claude came up with this
		pattern := regexp.MustCompile(`(?i)(<` + regexp.QuoteMeta(tag) + `[^>]*>)(.*?)(</` + regexp.QuoteMeta(tag) + `>)`)

		html = pattern.ReplaceAllStringFunc(html, func(match string) string {
			submatches := pattern.FindStringSubmatch(match)
			if len(submatches) != 4 {
				return match
			}

			openTag := submatches[1]
			content := submatches[2]
			closeTag := submatches[3]

			// Skip if content contains excluded tags
			for _, excludeTag := range cfg.ExcludeTags {
				excludePattern := regexp.MustCompile(`(?i)<` + regexp.QuoteMeta(excludeTag) + `[^>]*>`)
				if excludePattern.MatchString(content) {
					return match
				}
			}

			perturbedContent := d.perturbTextContent(content, cfg.Strategy, cfg.Density, frequency)
			return openTag + perturbedContent + closeTag
		})
	}

	return html
}

// Perturb text content (not HTML tags within)
func (d *DefenseMiddleware) perturbTextContent(content string, strategy string, density string, frequency float64) string {
	tagPattern := regexp.MustCompile(`(<[^>]+>)`)
	parts := tagPattern.Split(content, -1)
	tags := tagPattern.FindAllString(content, -1)

	var result strings.Builder
	tagIndex := 0

	for i, part := range parts {
		if part != "" && d.rng.Float64() < frequency {
			words := strings.Fields(part)
			var perturbedWords []string
			for _, word := range words {
				perturbedWords = append(perturbedWords, d.perturbWord(word, strategy, density))
			}
			result.WriteString(strings.Join(perturbedWords, " "))
			if strings.HasSuffix(part, " ") {
				result.WriteString(" ")
			}
		} else {
			result.WriteString(part)
		}

		// Re-insert the HTML tag that followed this text
		if i < len(parts)-1 && tagIndex < len(tags) {
			result.WriteString(tags[tagIndex])
			tagIndex++
		}
	}

	return result.String()
}

// perturbByWords perturbs specific words throughout the HTML.
func (d *DefenseMiddleware) perturbByWords(html string, cfg TextPerturbationConfig, frequency float64) string {
	for _, word := range cfg.TargetWords {
		if word == "" {
			continue
		}

		pattern := regexp.MustCompile(`(?i)\b(` + regexp.QuoteMeta(word) + `)\b`)
		html = pattern.ReplaceAllStringFunc(html, func(match string) string {
			if d.rng.Float64() > frequency {
				return match
			}

			if isInsideTag(html, strings.Index(html, match)) {
				return match
			}

			return d.perturbWord(match, cfg.Strategy, cfg.Density)
		})
	}

	return html
}

// perturbWord applies the perturbation strategy to a single word.
func (d *DefenseMiddleware) perturbWord(word string, strategy string, density string) string {
	switch strings.ToLower(strategy) {
	case "homoglyph":
		return d.applyHomoglyph(word)
	case "zero-width":
		fallthrough
	default:
		return d.applyZeroWidth(word, density)
	}
}

// applyZeroWidth inserts zero-width characters into the word.
func (d *DefenseMiddleware) applyZeroWidth(word string, density string) string {
	if len(word) < 2 {
		return word
	}

	// Taken from gibberifier
	zeroWidthChars := []string{
		"\u200B", // ZERO WIDTH SPACE
		"\u200C", // ZERO WIDTH NON-JOINER
		"\u200D", // ZERO WIDTH JOINER
		"\u2060", // WORD JOINER
		"\u2061", // FUNCTION APPLICATION
		"\u2062", // INVISIBLE TIMES
		"\u2063", // INVISIBLE SEPARATOR
		"\u2064", // INVISIBLE PLUS
		"\uFEFF", // ZERO WIDTH NO-BREAK SPACE (BOM)
	}

	// Density: "low" (1-3), "medium" (10-20), "high" (150-200)
	var minCount, maxCount int
	switch strings.ToLower(density) {
	case "low":
		minCount, maxCount = 1, 3
	case "medium":
		minCount, maxCount = 10, 20
	case "high":
		fallthrough
	default:
		minCount, maxCount = 150, 200
	}

	runes := []rune(word)
	var result strings.Builder
	result.WriteRune(runes[0])

	for i := 1; i < len(runes); i++ {
		// Insert random zero-width characters between each character
		count := minCount + d.rng.Intn(maxCount-minCount+1)
		for j := 0; j < count; j++ {
			zwc := zeroWidthChars[d.rng.Intn(len(zeroWidthChars))]
			result.WriteString(zwc)
		}
		result.WriteRune(runes[i])
	}

	return result.String()
}

// replace chars with similar looking unicode ones
func (d *DefenseMiddleware) applyHomoglyph(word string) string {
	homoglyphs := map[rune][]rune{
		'a': {'а', 'ɑ', 'α'},           // Cyrillic, Latin, Greek
		'A': {'Α', 'А'},                 // Greek, Cyrillic
		'e': {'е', 'ε', 'ė'},            // Cyrillic, Greek, Latin
		'E': {'Ε', 'Е'},                 // Greek, Cyrillic
		'i': {'і', 'ι', 'ί'},            // Cyrillic, Greek
		'I': {'Ι', 'І', 'Ί'},            // Greek, Cyrillic
		'o': {'о', 'ο', 'ö'},            // Cyrillic, Greek, Latin
		'O': {'О', 'Ο', 'Ö'},            // Cyrillic, Greek, Latin
		'p': {'р', 'ρ'},                 // Cyrillic, Greek
		'P': {'Р', 'Ρ'},                 // Cyrillic, Greek
		'c': {'с', 'ϲ'},                 // Cyrillic, Greek
		'C': {'С', 'Ϲ'},                 // Cyrillic, Greek
		'x': {'х', 'χ'},                 // Cyrillic, Greek
		'X': {'Χ', 'Х'},                 // Greek, Cyrillic
		'y': {'у', 'γ'},                 // Cyrillic, Greek
		'Y': {'Υ', 'У'},                 // Greek, Cyrillic
		'B': {'В', 'Β'},                 // Cyrillic, Greek
		'H': {'Η', 'Н'},                 // Greek, Cyrillic
		'K': {'Κ', 'К'},                 // Greek, Cyrillic
		'M': {'Μ', 'М'},                 // Greek, Cyrillic
		'N': {'Ν', 'Н'},                 // Greek, Cyrillic
		'T': {'Τ', 'Т'},                 // Greek, Cyrillic
		's': {'ѕ', 'ş'},                 // Cyrillic, Latin
		'S': {'Ѕ', 'Ş'},                 // Cyrillic, Latin
		'j': {'ј'},                      // Cyrillic
		'J': {'Ј'},                      // Cyrillic
		'l': {'ӏ', 'ł', '|'},            // Cyrillic, Latin, Pipe
		'1': {'ӏ', 'l', '|'},            // Cyrillic, Latin, Pipe
		'0': {'О', 'о', 'Ο', 'ο'},       // Cyrillic, Greek
	}

	runes := []rune(word)
	var result strings.Builder

	for _, r := range runes {
		if replacements, ok := homoglyphs[r]; ok && d.rng.Float64() < 0.6 {
			result.WriteRune(replacements[d.rng.Intn(len(replacements))])
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

func (d *DefenseMiddleware) applyStructuralObfuscation(html string) string {
	cfg := d.config.StructuralObfuscation

	for _, class := range cfg.TargetClasses {
		if class == "" {
			continue
		}
		html = d.obfuscateClass(html, class, cfg.ReplacementStrategy)
	}

	// Skip IDs in safe mode, maybe preserves JS
	if !cfg.SafeMode {
		for _, id := range cfg.TargetIDs {
			if id == "" {
				continue
			}
			html = d.obfuscateID(html, id, cfg.ReplacementStrategy)
		}
	}

	return html
}

func (d *DefenseMiddleware) obfuscateClass(html, class, strategy string) string {
	newClass := d.generateReplacement(class, strategy)

	classAttrRegex := regexp.MustCompile(`class\s*=\s*"([^"]*)"`)
	html = classAttrRegex.ReplaceAllStringFunc(html, func(match string) string {
		wordBoundary := regexp.MustCompile(`\b` + regexp.QuoteMeta(class) + `\b`)
		return wordBoundary.ReplaceAllString(match, newClass)
	})

	classAttrSingleRegex := regexp.MustCompile(`class\s*=\s*'([^']*)'`)
	html = classAttrSingleRegex.ReplaceAllStringFunc(html, func(match string) string {
		wordBoundary := regexp.MustCompile(`\b` + regexp.QuoteMeta(class) + `\b`)
		return wordBoundary.ReplaceAllString(match, newClass)
	})

	styleRegex := regexp.MustCompile(`(?s)<style[^>]*>(.*?)</style>`)
	html = styleRegex.ReplaceAllStringFunc(html, func(match string) string {
		cssSelector := regexp.MustCompile(`\.` + regexp.QuoteMeta(class) + `\b`)
		return cssSelector.ReplaceAllString(match, "."+newClass)
	})

	return html
}

func (d *DefenseMiddleware) obfuscateID(html, id, strategy string) string {
	newID := d.generateReplacement(id, strategy)

	idAttrRegex := regexp.MustCompile(`id\s*=\s*"` + regexp.QuoteMeta(id) + `"`)
	html = idAttrRegex.ReplaceAllString(html, `id="`+newID+`"`)

	idAttrSingleRegex := regexp.MustCompile(`id\s*=\s*'` + regexp.QuoteMeta(id) + `'`)
	html = idAttrSingleRegex.ReplaceAllString(html, `id='`+newID+`'`)

	fragmentRegex := regexp.MustCompile(`(href\s*=\s*["'])#` + regexp.QuoteMeta(id) + `(["'])`)
	html = fragmentRegex.ReplaceAllString(html, "${1}#"+newID+"${2}")

	styleRegex := regexp.MustCompile(`(?s)<style[^>]*>(.*?)</style>`)
	html = styleRegex.ReplaceAllStringFunc(html, func(match string) string {
		cssSelector := regexp.MustCompile(`#` + regexp.QuoteMeta(id) + `\b`)
		return cssSelector.ReplaceAllString(match, "#"+newID)
	})

	jsGetElemRegex := regexp.MustCompile(`getElementById\s*\(\s*["']` + regexp.QuoteMeta(id) + `["']\s*\)`)
	html = jsGetElemRegex.ReplaceAllString(html, `getElementById("`+newID+`")`)

	return html
}

func (d *DefenseMiddleware) generateReplacement(original, strategy string) string {
	suffix := d.randomString(8)

	switch strings.ToLower(strategy) {
	case "total-replace":
		return d.randomString(12)
	case "random-suffix":
		fallthrough
	default:
		return original + "_" + suffix
	}
}

func (d *DefenseMiddleware) randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[d.rng.Intn(len(charset))]
	}
	return string(result)
}

func isInsideTag(html string, pos int) bool {
	if pos < 0 || pos >= len(html) {
		return false
	}

	lastOpen := strings.LastIndex(html[:pos], "<")
	lastClose := strings.LastIndex(html[:pos], ">")

	return lastOpen > lastClose
}

// this is absolutely not the best way to escape HTML, but I don't really see how this might be a problem unless you're
// trying to XSS yourself or something
func escapeHTML(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(s)
}

// TODO
func (d *DefenseMiddleware) applyDecoyElements(html string) string {
	cfg := d.config.DecoyElements

	if cfg.HoneypotLinks.Enabled {
		html = d.injectHoneypotLinks(html)
	}

	if cfg.FakeFormFields.Enabled {
		html = d.injectFakeFormFields(html)
	}

	if cfg.DecoyData.Enabled {
		html = d.injectDecoyData(html)
	}

	if cfg.InvisibleButtons.Enabled {
		html = d.injectInvisibleButtons(html)
	}

	return html
}

func (d *DefenseMiddleware) injectHoneypotLinks(html string) string {
	// TODO: Implement
	// - Insert hidden <a> tags with trap URLs
	return html
}

func (d *DefenseMiddleware) injectFakeFormFields(html string) string {
	// TODO: Implement
	// - Find <form> elements
	// - Insert hidden input fields with sexy names
	return html
}

func (d *DefenseMiddleware) injectDecoyData(html string) string {
	// TODO: Implement
	// - Insert hidden divs with fake prices, emails, phone numbers
	return html
}

func (d *DefenseMiddleware) injectInvisibleButtons(html string) string {
	// TODO: Implement
	// - Insert hidden buttons/links with tracking data attributes
	return html
}
