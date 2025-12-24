package defense_middleware

import (
	"bytes"
	"context"
	// _ "embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// const defenseScript = `(function() {
//     // var ZWC = ['\u200B', '\u200C', '\u200D', '\u2060', '\u2061', '\u2062', '\u2063', '\u2064', '\uFEFF'];
// 	var ZWC = ['\u200B', '\uFEFF']; 

//     var HOMOGLYPHS = {
//         'a': ['а', 'α'], 'e': ['е', 'ε'], 'i': ['і', 'ι'], 'o': ['о', 'ο'],
//         'p': ['р', 'ρ'], 'c': ['с', 'ϲ'], 'x': ['х', 'χ'], 'y': ['у'],
//         'A': ['А', 'Α'], 'E': ['Е', 'Ε'], 'O': ['О', 'Ο'], 'P': ['Р', 'Ρ'],
//         'C': ['С', 'Ϲ'], 'B': ['В', 'Β'], 'H': ['Н', 'Η'], 'K': ['К', 'Κ'],
//         'M': ['М', 'Μ'], 'T': ['Т', 'Τ'], 'X': ['Х', 'Χ'], 'S': ['Ѕ']
//     };

//     var HIDE_STYLES = {
//         'absolute': 'position:absolute;left:-9999px;top:-9999px;width:1px;height:1px;overflow:hidden;',
//         'hidden': 'visibility:hidden;position:absolute;',
//         'opacity': 'opacity:0;position:absolute;pointer-events:none;',
//         'clip': 'clip:rect(0,0,0,0);clip-path:inset(50%);position:absolute;'
//     };

//     var BOT_ATTRS = ['mmid', 'data-testid', 'data-element-id', 'data-automation-id', 'data-qa', 'aria-roledescription'];

//     function rand(min, max) {
//         return Math.floor(Math.random() * (max - min + 1)) + min;
//     }

//     function pick(arr) {
//         return arr[rand(0, arr.length - 1)];
//     }

//     function densityRange(d) {
//         if (d === 'low') return [1, 3];
//         if (d === 'medium') return [10, 20];
//         return [150, 200];
//     }

//     function zeroWidth(text, density) {
//         if (text.length < 2) return text;
//         var r = densityRange(density);
//         var out = text[0];
//         for (var i = 1; i < text.length; i++) {
//             var n = rand(r[0], r[1]);
//             for (var j = 0; j < n; j++) out += pick(ZWC);
//             out += text[i];
//         }
//         return out;
//     }

//     function homoglyph(text) {
//         var out = '';
//         for (var i = 0; i < text.length; i++) {
//             var c = text[i];
//             if (HOMOGLYPHS[c] && Math.random() < 0.6) {
//                 out += pick(HOMOGLYPHS[c]);
//             } else {
//                 out += c;
//             }
//         }
//         return out;
//     }

//     function perturb(text, strategy, density) {
//         return strategy === 'homoglyph' ? homoglyph(text) : zeroWidth(text, density);
//     }

//     function isExcluded(node, excludes) {
//         var p = node.parentElement;
//         while (p) {
//             if (excludes.indexOf(p.tagName.toLowerCase()) !== -1) return true;
//             p = p.parentElement;
//         }
//         return false;
//     }

//     function walkText(root, fn) {
//         var w = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
//         var nodes = [];
//         while (w.nextNode()) nodes.push(w.currentNode);
//         nodes.forEach(fn);
//     }

//     function semanticInjection() {
//         var c = cfg.semanticInjection;
//         if (!c || !c.enabled || !c.prompt) return;

//         var div = document.createElement('div');
//         div.setAttribute('aria-hidden', 'true');
//         div.style.cssText = HIDE_STYLES[c.hidingMethod] || HIDE_STYLES['absolute'];
//         div.textContent = c.prompt;

//         var pos = c.position || 'body-start';
//         if (pos === 'body-end') {
//             document.body.appendChild(div);
//         } else if (pos === 'head') {
//             document.head.appendChild(div);
//         } else {
//             document.body.insertBefore(div, document.body.firstChild);
//         }
//     }

//     function textPerturbation() {
//         var c = cfg.textPerturbation;
//         if (!c || !c.enabled) return;

//         var excludes = c.excludeTags || [];
//         var freq = c.frequency || 0.8;
//         var strat = c.strategy || 'zero-width';
//         var dens = c.density || 'high';

//         function process(node) {
//             if (isExcluded(node, excludes)) return;
//             if (!node.textContent.trim()) return;
//             if (Math.random() > freq) return;

//             var txt = node.textContent;

//             if (c.targetWords && c.targetWords.length) {
//                 c.targetWords.forEach(function(w) {
//                     var re = new RegExp('\\b(' + w.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + ')\\b', 'gi');
//                     txt = txt.replace(re, function(m) { return perturb(m, strat, dens); });
//                 });
//             } else {
//                 txt = perturb(txt, strat, dens);
//             }

//             node.textContent = txt;
//         }

//         if (c.targetTags && c.targetTags.length) {
//             c.targetTags.forEach(function(tag) {
//                 document.querySelectorAll(tag).forEach(function(el) {
//                     walkText(el, process);
//                 });
//             });
//         } else if (c.targetWords && c.targetWords.length) {
//             walkText(document.body, process);
//         }
//     }

//     function elementProtection() {
//         var c = cfg.elementProtection;
//         if (!c || !c.enabled || !c.targets) return;

//         c.targets.forEach(function(t) {
//             var el = document.querySelector(t.selector);
//             if (!el) return;

//             if (t.sanitizeAttributes) {
//                 sanitizeEl(el);
//                 observeEl(el);
//             }

//             if (t.addDecoy) {
//                 createDecoy(el, t);
//             }
//         });
//     }

//     function sanitizeEl(el) {
//         BOT_ATTRS.forEach(function(attr) {
//             if (el.hasAttribute(attr)) el.removeAttribute(attr);
//         });
//     }

//     function observeEl(el) {
//         var obs = new MutationObserver(function(muts) {
//             muts.forEach(function(m) {
//                 if (m.type === 'attributes' && BOT_ATTRS.indexOf(m.attributeName) !== -1) {
//                     el.removeAttribute(m.attributeName);
//                 }
//             });
//         });
//         obs.observe(el, { attributes: true });
//     }

//     function createDecoy(el, t) {
//         var clone = el.cloneNode(true);

//         if (t.decoyHref) clone.setAttribute('href', t.decoyHref);
//         if (t.decoyAction) clone.setAttribute('action', t.decoyAction);

//         BOT_ATTRS.forEach(function(attr) {
//             var val = el.getAttribute(attr);
//             if (val) clone.setAttribute(attr, val + '_decoy');
//         });

//         clone.style.cssText = HIDE_STYLES['absolute'];
//         clone.setAttribute('aria-hidden', 'true');
//         clone.setAttribute('tabindex', '-1');

//         if (el.parentNode) {
//             el.parentNode.insertBefore(clone, el.nextSibling);
//         }
//     }

//     function init() {
//         semanticInjection();
//         textPerturbation();
//         elementProtection();
//     }

//     if (document.readyState === 'loading') {
//         document.addEventListener('DOMContentLoaded', init);
//     } else {
//         init();
//     }
// })();
// `

const defenseScript = `(function() {
    var JUNK_CHARS = 'abcdefghijklmnopqrstuvwxyz0123456789';
    var JUNK_CLASS = 'bd-' + Math.random().toString(36).slice(2, 8);

	var ZWC = ['\u200B', '\uFEFF']; 

    var HIDE_STYLES = {
        'absolute': 'position:absolute;left:-9999px;top:-9999px;width:1px;height:1px;overflow:hidden;',
        'hidden': 'visibility:hidden;position:absolute;',
        'opacity': 'opacity:0;position:absolute;pointer-events:none;',
        'clip': 'clip:rect(0,0,0,0);clip-path:inset(50%);position:absolute;'
    };

	var JUNK_STYLES = [
        'position:absolute!important;left:-9999px!important;top:0!important;',
        'display:inline-block!important;width:0!important;height:0!important;overflow:hidden!important;opacity:0!important;pointer-events:none!important;vertical-align:bottom!important;',
        'font-size:0!important;color:transparent!important;',
        'clip:rect(0,0,0,0)!important;clip-path:inset(50%)!important;position:absolute!important;',
        'font-size:1px!important;color:rgba(0,0,0,0)!important;display:inline-block!important;width:0!important;height:0!important;overflow:hidden!important;',
        'opacity:0!important;width:0!important;height:0!important;display:inline-block!important;overflow:hidden!important;',
        'display:none!important;'
    ];

    var BOT_ATTRS = ['mmid', 'data-testid', 'data-element-id', 'data-automation-id', 'data-qa', 'aria-roledescription'];

    function rand(min, max) {
        return Math.floor(Math.random() * (max - min + 1)) + min;
    }

    function pick(arr) {
        return arr[rand(0, arr.length - 1)];
    }

    function randChar() {
        return JUNK_CHARS[rand(0, JUNK_CHARS.length - 1)];
    }

    function densityRange(d) {
        if (d === 'low') return [1, 2];
        if (d === 'medium') return [2, 4];
        return [3, 6];
    }

    function injectStyle() {
        var style = document.createElement('style');
        style.textContent = '.' + JUNK_CLASS + '{position:absolute!important;left:-9999px!important;top:-9999px!important;width:1px!important;height:1px!important;overflow:hidden!important;font-size:1px!important;color:transparent!important;pointer-events:none!important;}';
        document.head.appendChild(style);
    }

	function createJunk() {
        // 33% chance of Zero Width Char, 66% chance of Span element
        var method = rand(0, 2); 
        
        if (method === 0) {
            // Method A: Zero Width Characters (Invisible by definition)
            var zwc = '';
            var count = rand(1, 3);
            for (var i = 0; i < count; i++) {
                zwc += ZWC[rand(0, ZWC.length - 1)];
            }
            return document.createTextNode(zwc);
        } else {
            // Method B: Span with randomized hiding style
            var span = document.createElement('span');
            
            // Randomize class name for every span to prevent simple rule-based blocking
            var randomClass = 'bd-' + Math.random().toString(36).slice(2, 8);
            span.className = randomClass;
            
            // Apply one of the fixed, safe styles
            span.style.cssText = JUNK_STYLES[rand(0, JUNK_STYLES.length - 1)];
            
            var junk = '';
            var count = rand(1, 4);
            for (var i = 0; i < count; i++) {
                junk += randChar();
            }
            span.textContent = junk;
            return span;
        }
    }

    function perturbTextNode(node, density) {
        var text = node.textContent;
        // Skip empty or extremely short text to avoid breaking UI icons/glyphs
        if (text.trim().length < 2) return; 

        var range = densityRange(density);
        var frag = document.createDocumentFragment();

        for (var i = 0; i < text.length; i++) {
            frag.appendChild(document.createTextNode(text[i]));
            
            // Don't inject after the very last character to be safe with spacing
            if (i < text.length - 1) {
                // Random chance to inject junk (not every character, to save DOM weight)
                // Adjust this threshold (0.5) to change density
                if (Math.random() > 0.5) { 
                    var count = rand(range[0], range[1]);
                    for (var j = 0; j < count; j++) {
                        frag.appendChild(createJunk());
                    }
                }
            }
        }

        node.parentNode.replaceChild(frag, node);
    }

    function isExcluded(node, excludes) {
        var p = node.parentElement;
        while (p) {
            if (excludes.indexOf(p.tagName.toLowerCase()) !== -1) return true;
            if (p.classList && p.classList.contains(JUNK_CLASS)) return true;
            p = p.parentElement;
        }
        return false;
    }

    function walkText(root, fn) {
        var w = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
        var nodes = [];
        while (w.nextNode()) nodes.push(w.currentNode);
        nodes.forEach(fn);
    }

    function textPerturbation() {
        var c = cfg.textPerturbation;
        if (!c || !c.enabled) return;

        // injectStyle();

        var excludes = (c.excludeTags || []).map(function(t) { return t.toLowerCase(); });
        var freq = c.frequency || 0.8;
        var dens = c.density || 'high';

        function process(node) {
            if (isExcluded(node, excludes)) return;
            if (!node.textContent.trim()) return;
            if (Math.random() > freq) return;
            perturbTextNode(node, dens);
        }

        if (c.targetTags && c.targetTags.length) {
            c.targetTags.forEach(function(tag) {
                document.querySelectorAll(tag).forEach(function(el) {
                    walkText(el, process);
                });
            });
        }
    }

    function semanticInjection() {
        var c = cfg.semanticInjection;
        if (!c || !c.enabled || !c.prompt) return;

        var div = document.createElement('div');
        div.setAttribute('aria-hidden', 'true');
        div.style.cssText = HIDE_STYLES[c.hidingMethod] || HIDE_STYLES['absolute'];
        div.textContent = c.prompt;

        var pos = c.position || 'body-start';
        if (pos === 'body-end') {
            document.body.appendChild(div);
        } else if (pos === 'head') {
            document.head.appendChild(div);
        } else {
            document.body.insertBefore(div, document.body.firstChild);
        }
    }

    function elementProtection() {
        var c = cfg.elementProtection;
        if (!c || !c.enabled || !c.targets) return;

        c.targets.forEach(function(t) {
            var el = document.querySelector(t.selector);
            if (!el) return;

            if (t.sanitizeAttributes) {
                sanitizeEl(el);
                observeEl(el);
            }

            if (t.addDecoy) {
                createDecoy(el, t);
            }
        });
    }

    function sanitizeEl(el) {
        BOT_ATTRS.forEach(function(attr) {
            if (el.hasAttribute(attr)) el.removeAttribute(attr);
        });
    }

    function observeEl(el) {
        var obs = new MutationObserver(function(muts) {
            muts.forEach(function(m) {
                if (m.type === 'attributes' && BOT_ATTRS.indexOf(m.attributeName) !== -1) {
                    el.removeAttribute(m.attributeName);
                }
            });
        });
        obs.observe(el, { attributes: true });
    }

    function createDecoy(el, t) {
        var clone = el.cloneNode(true);

        if (t.decoyHref) clone.setAttribute('href', t.decoyHref);
        if (t.decoyAction) clone.setAttribute('action', t.decoyAction);

        BOT_ATTRS.forEach(function(attr) {
            var val = el.getAttribute(attr);
            if (val) clone.setAttribute(attr, val + '_decoy');
        });

        clone.style.cssText = HIDE_STYLES['absolute'];
        clone.setAttribute('aria-hidden', 'true');
        clone.setAttribute('tabindex', '-1');

        if (el.parentNode) {
            el.parentNode.insertBefore(clone, el.nextSibling);
        }
    }

    function init() {
        semanticInjection();
        textPerturbation();
        elementProtection();
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();


`

type Config struct {
	SemanticInjection SemanticInjectionConfig `json:"semanticInjection,omitempty"`
	TextPerturbation  TextPerturbationConfig  `json:"textPerturbation,omitempty"`
	ElementProtection ElementProtectionConfig `json:"elementProtection,omitempty"`
}

type SemanticInjectionConfig struct {
	Enabled      bool   `json:"enabled,omitempty"`
	Mode         string `json:"mode,omitempty"` // server, client, both
	Prompt       string `json:"prompt,omitempty"`
	Position     string `json:"position,omitempty"`     // body-start, body-end, head
	HidingMethod string `json:"hidingMethod,omitempty"` // absolute, hidden, opacity, clip
}

type TextPerturbationConfig struct {
	Enabled     bool     `json:"enabled,omitempty"`
	Mode        string   `json:"mode,omitempty"` // server, client, both
	TargetWords []string `json:"targetWords,omitempty"`
	TargetTags  []string `json:"targetTags,omitempty"`
	ExcludeTags []string `json:"excludeTags,omitempty"`
	Strategy    string   `json:"strategy,omitempty"` // zero-width, homoglyph
	Frequency   float64  `json:"frequency,omitempty"`
	Density     string   `json:"density,omitempty"` // low, medium, high
}

type ElementProtectionConfig struct {
	Enabled bool                      `json:"enabled,omitempty"`
	Targets []ElementProtectionTarget `json:"targets,omitempty"`
}

type ElementProtectionTarget struct {
	Selector           string `json:"selector,omitempty"`
	SanitizeAttributes bool   `json:"sanitizeAttributes,omitempty"`
	AddDecoy           bool   `json:"addDecoy,omitempty"`
	DecoyHref          string `json:"decoyHref,omitempty"`
	DecoyAction        string `json:"decoyAction,omitempty"`
}

func CreateConfig() *Config {
	return &Config{
		SemanticInjection: SemanticInjectionConfig{
			Enabled:      false,
			Mode:         "both",
			Position:     "body-start",
			HidingMethod: "absolute",
		},
		TextPerturbation: TextPerturbationConfig{
			Enabled:     false,
			Mode:        "both",
			ExcludeTags: []string{"code", "pre", "script", "style"},
			Strategy:    "zero-width",
			Frequency:   0.8,
			Density:     "high",
		},
		ElementProtection: ElementProtectionConfig{
			Enabled: false,
		},
	}
}

type DefenseMiddleware struct {
	next   http.Handler
	config *Config
	rng    *rand.Rand
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config == nil {
		config = CreateConfig()
	}
	return &DefenseMiddleware{
		next:   next,
		config: config,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}, nil
}

func (d *DefenseMiddleware) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	req.Header.Del("Accept-Encoding")
	wrapper := &responseWrapper{
		ResponseWriter: rw,
		buffer:         &bytes.Buffer{},
	}

	d.next.ServeHTTP(wrapper, req)

	body := wrapper.buffer.Bytes()
	contentType := wrapper.Header().Get("Content-Type")

	if strings.Contains(strings.ToLower(contentType), "text/html") && len(body) > 0 {
		body = d.processHTML(body)
	}

	for k, v := range wrapper.Header() {
		rw.Header()[k] = v
	}
	rw.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	rw.WriteHeader(wrapper.statusCode)
	rw.Write(body)
}

func (d *DefenseMiddleware) processHTML(body []byte) []byte {
	html := string(body)
	html = d.applyServerDefenses(html)
	html = d.injectClientScript(html)
	return []byte(html)
}

func (d *DefenseMiddleware) applyServerDefenses(html string) string {
	cfg := d.config

	if cfg.SemanticInjection.Enabled && d.isServerMode(cfg.SemanticInjection.Mode) {
		html = d.serverSemanticInjection(html)
	}

	if cfg.TextPerturbation.Enabled && d.isServerMode(cfg.TextPerturbation.Mode) {
		html = d.serverTextPerturbation(html)
	}

	return html
}

func (d *DefenseMiddleware) isServerMode(mode string) bool {
	m := strings.ToLower(mode)
	return m == "server" || m == "both" || m == ""
}

func (d *DefenseMiddleware) isClientMode(mode string) bool {
	m := strings.ToLower(mode)
	return m == "client" || m == "both" || m == ""
}

func (d *DefenseMiddleware) injectClientScript(html string) string {
	clientCfg := map[string]interface{}{}

	if d.config.SemanticInjection.Enabled && d.isClientMode(d.config.SemanticInjection.Mode) {
		clientCfg["semanticInjection"] = d.config.SemanticInjection
	}

	if d.config.TextPerturbation.Enabled && d.isClientMode(d.config.TextPerturbation.Mode) {
		clientCfg["textPerturbation"] = d.config.TextPerturbation
	}

	if d.config.ElementProtection.Enabled {
		clientCfg["elementProtection"] = d.config.ElementProtection
	}

	if len(clientCfg) == 0 {
		return html
	}

	cfgJSON, _ := json.Marshal(clientCfg)
	script := fmt.Sprintf("<script>(function(){var cfg=%s;%s})()</script>", cfgJSON, defenseScript)

	if idx := strings.LastIndex(strings.ToLower(html), "</body>"); idx != -1 {
		return html[:idx] + script + html[idx:]
	}
	return html + script
}

// func (d *DefenseMiddleware) serverSemanticInjection(html string) string {
// 	cfg := d.config.SemanticInjection
// 	if cfg.Prompt == "" {
// 		return html
// 	}

// 	styles := map[string]string{
// 		"absolute": "position:absolute;left:-9999px;top:-9999px;width:1px;height:1px;overflow:hidden;",
// 		"hidden":   "visibility:hidden;position:absolute;",
// 		"opacity":  "opacity:0;position:absolute;pointer-events:none;",
// 		"clip":     "clip:rect(0,0,0,0);clip-path:inset(50%);position:absolute;",
// 	}

// 	style := styles[cfg.HidingMethod]
// 	if style == "" {
// 		style = styles["absolute"]
// 	}

// 	div := fmt.Sprintf(`<div style="%s" aria-hidden="true">%s</div>`, style, escapeHTML(cfg.Prompt))

// 	switch strings.ToLower(cfg.Position) {
// 	case "body-end":
// 		re := regexp.MustCompile(`(?i)(</body>)`)
// 		return re.ReplaceAllString(html, div+"$1")
// 	case "head":
// 		re := regexp.MustCompile(`(?i)(</head>)`)
// 		return re.ReplaceAllString(html, div+"$1")
// 	default:
// 		re := regexp.MustCompile(`(?i)(<body[^>]*>)`)
// 		return re.ReplaceAllString(html, "$1"+div)
// 	}
// }

// func (d *DefenseMiddleware) serverSemanticInjection(html string) string {
// 	cfg := d.config.SemanticInjection
// 	if cfg.Prompt == "" {
// 		return html
// 	}

// 	styles := map[string]string{
// 		// Weak: Easy to filter
// 		"absolute": "position:absolute;left:-9999px;top:-9999px;width:1px;height:1px;overflow:hidden;",
// 		"hidden":   "visibility:hidden;position:absolute;",

// 		// Strong: Bypasses simple filters (Opacity 0)
// 		"opacity": "opacity:0;position:absolute;width:1px;height:1px;overflow:hidden;pointer-events:none;z-index:-1;display:inline-block;",

// 		// Strongest: Reader Mode Bypass (Z-Index Camouflage)
// 		// Appears as valid, visible paragraph content to parsers, but hidden behind bg
// 		"z_index": "position:absolute;top:0;left:0;width:100%;height:auto;z-index:-9999;color:#000;background:#fff;opacity:1;font-size:12px;pointer-events:none;overflow:hidden;white-space:normal;",
// 	}

// 	style := styles[cfg.HidingMethod]
// 	if style == "" {
// 		style = styles["z_index"]
// 	}

// 	// Use <p> tag as Reader Mode prefers it over <div>
// 	tag := "div"
// 	if cfg.HidingMethod == "z_index" {
// 		tag = "p"
// 	}

// 	injection := fmt.Sprintf(`<%s style="%s" aria-hidden="false">%s</%s>`, tag, style, escapeHTML(cfg.Prompt), tag)

// 	switch strings.ToLower(cfg.Position) {
// 	case "body-end":
// 		re := regexp.MustCompile(`(?i)(</body>)`)
// 		return re.ReplaceAllString(html, injection+"$1")
// 	case "head":
// 		// Inject after body open if they ask for head, as visible text in head is invalid
// 		re := regexp.MustCompile(`(?i)(<body[^>]*>)`)
// 		return re.ReplaceAllString(html, "$1"+injection)
// 	default:
// 		// body-start
// 		re := regexp.MustCompile(`(?i)(<body[^>]*>)`)
// 		return re.ReplaceAllString(html, "$1"+injection)
// 	}
// }

func (d *DefenseMiddleware) serverSemanticInjection(html string) string {
	cfg := d.config.SemanticInjection
	if cfg.Prompt == "" {
		return html
	}

	// UPDATED: Inline-safe styles for injecting INSIDE a paragraph.
	// We cannot use <p> or <div> here. We must use <span>.
	// We use font-size:0 to hide it visually, but keep it in the text flow for the parser.
	styles := map[string]string{
		// 1. Font Size Zero: Classic, effective, inline-safe.
		"inline_zero": "font-size:0;width:0;height:0;opacity:0;position:absolute;z-index:-1;",
		
		// 2. Color Transparent: Keeps the space (1px) but makes it invisible.
		// Good if parsers strip 'font-size: 0'.
		"inline_transparent": "color:transparent;font-size:1px;width:1px;display:inline-block;opacity:0;overflow:hidden;",
	}

	style := styles[cfg.HidingMethod]
	if style == "" {
		style = styles["inline_zero"]
	}

	// Use <span> because we are injecting INSIDE a <p> tag.
	// aria-hidden="false" ensures bots read it.
	injection := fmt.Sprintf(`<span style="%s" aria-hidden="false">%s. </span>`, style, escapeHTML(cfg.Prompt))

	// LOGIC CHANGE: Target the content, not the container.
	switch strings.ToLower(cfg.Position) {
	case "first-paragraph":
		// Find the first <p> tag (case insensitive) and inject immediately after it opens.
		// We use Split/Join or Replace to only target the first occurrence efficiently.
		re := regexp.MustCompile(`(?i)(<p[^>]*>)`)
		// Replace only the first match (1)
		return re.ReplaceAllStringFunc(html, func(match string) string {
			return match + injection
		})
		
	default:
		// Fallback to body-start if they haven't updated config yet
		re := regexp.MustCompile(`(?i)(<body[^>]*>)`)
		return re.ReplaceAllString(html, "$1"+injection)
	}
}

func (d *DefenseMiddleware) serverTextPerturbation(html string) string {
	cfg := d.config.TextPerturbation

	if len(cfg.TargetTags) == 0 && len(cfg.TargetWords) == 0 {
		return html
	}

	freq := cfg.Frequency
	if freq <= 0 || freq > 1 {
		freq = 0.8
	}

	for _, tag := range cfg.TargetTags {
		html = d.perturbTag(html, tag, cfg, freq)
	}

	for _, word := range cfg.TargetWords {
		html = d.perturbWordInHTML(html, word, cfg, freq)
	}

	return html
}

func (d *DefenseMiddleware) perturbTag(html, tag string, cfg TextPerturbationConfig, freq float64) string {
	re := regexp.MustCompile(`(?is)(<` + regexp.QuoteMeta(tag) + `[^>]*>)(.*?)(</` + regexp.QuoteMeta(tag) + `>)`)

	return re.ReplaceAllStringFunc(html, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}

		for _, exc := range cfg.ExcludeTags {
			if strings.Contains(strings.ToLower(parts[2]), "<"+strings.ToLower(exc)) {
				return match
			}
		}

		if d.rng.Float64() > freq {
			return match
		}

		return parts[1] + d.perturbText(parts[2], cfg.Strategy, cfg.Density) + parts[3]
	})
}

func (d *DefenseMiddleware) perturbWordInHTML(html, word string, cfg TextPerturbationConfig, freq float64) string {
	re := regexp.MustCompile(`(?i)\b(` + regexp.QuoteMeta(word) + `)\b`)

	return re.ReplaceAllStringFunc(html, func(match string) string {
		if d.rng.Float64() > freq {
			return match
		}
		return d.perturbText(match, cfg.Strategy, cfg.Density)
	})
}

func (d *DefenseMiddleware) perturbText(text, strategy, density string) string {
    if strings.Contains(text, "<") || strings.Contains(text, ">") {
        return text
    }
    if strategy == "homoglyph" {
        return d.homoglyph(text)
    }
    return d.zeroWidth(text, density)
}

func (d *DefenseMiddleware) zeroWidth(text, density string) string {
	if len(text) < 2 {
		return text
	}

	// chars := []string{"\u200B", "\u200C", "\u200D", "\u2060", "\u2061", "\u2062", "\u2063", "\u2064", "\uFEFF"}
	chars := []string{"\u200B", "\uFEFF"}

	var min, max int
	switch density {
	case "low":
		min, max = 1, 3
	case "medium":
		min, max = 10, 20
	default:
		min, max = 150, 200
	}

	runes := []rune(text)
	var b strings.Builder
	b.WriteRune(runes[0])

	for i := 1; i < len(runes); i++ {
		n := min + d.rng.Intn(max-min+1)
		for j := 0; j < n; j++ {
			b.WriteString(chars[d.rng.Intn(len(chars))])
		}
		b.WriteRune(runes[i])
	}

	return b.String()
}

func (d *DefenseMiddleware) homoglyph(text string) string {
	m := map[rune][]rune{
		'a': {'а', 'α'}, 'e': {'е', 'ε'}, 'i': {'і', 'ι'}, 'o': {'о', 'ο'},
		'p': {'р', 'ρ'}, 'c': {'с', 'ϲ'}, 'x': {'х', 'χ'}, 'y': {'у'},
		'A': {'А', 'Α'}, 'E': {'Е', 'Ε'}, 'O': {'О', 'Ο'}, 'P': {'Р', 'Ρ'},
		'C': {'С', 'Ϲ'}, 'B': {'В', 'Β'}, 'H': {'Н', 'Η'}, 'K': {'К', 'Κ'},
		'M': {'М', 'Μ'}, 'T': {'Т', 'Τ'}, 'X': {'Х', 'Χ'}, 'S': {'Ѕ'},
	}

	var b strings.Builder
	for _, r := range text {
		if alts, ok := m[r]; ok && d.rng.Float64() < 0.6 {
			b.WriteRune(alts[d.rng.Intn(len(alts))])
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func escapeHTML(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;").Replace(s)
}

type responseWrapper struct {
	http.ResponseWriter
	buffer     *bytes.Buffer
	statusCode int
}

func (rw *responseWrapper) WriteHeader(code int) {
	rw.statusCode = code
}

func (rw *responseWrapper) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	return rw.buffer.Write(b)
}
