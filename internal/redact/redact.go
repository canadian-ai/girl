// Package redact is GIRL's shared export-boundary sanitizer.
//
// It is deliberately conservative: when a value looks like a credential it is
// replaced, even at the cost of hiding harmless text. It is NOT a guarantee of
// perfect secret detection. Exports that leave the machine ("public") must
// additionally pass Scan on the serialized output, failing closed when any
// detector still fires.
package redact

import (
	"math"
	"reflect"
	"regexp"
	"strings"
)

// Placeholder replaces detected sensitive values.
const Placeholder = "<redacted>"

type rule struct {
	name string
	re   *regexp.Regexp
	repl string
}

// sensitiveKey matches assignment keys whose values are treated as secrets.
const sensitiveKey = `(?i:[a-z0-9_.\-]*(?:pass(?:word|wd|phrase)?|secret|token|api[_\-]?key|apikey|access[_\-]?key|private[_\-]?key|client[_\-]?secret|credential|auth(?:orization)?|session[_\-]?(?:key|id)|signing[_\-]?key|encryption[_\-]?key|salt|dsn|conn(?:ection)?[_\-]?str(?:ing)?|database[_\-]?url|webhook[_\-]?url)[a-z0-9_.\-]*)`

var rules = []rule{
	// Multiline PEM private keys (RSA, EC, OPENSSH, PGP, encrypted, ...).
	{"private-key-block", regexp.MustCompile(`(?s)-----BEGIN [A-Z0-9 ]*PRIVATE KEY(?: BLOCK)?-----.*?(?:-----END [A-Z0-9 ]*PRIVATE KEY(?: BLOCK)?-----|\z)`), "-----BEGIN PRIVATE KEY-----" + Placeholder + "-----END PRIVATE KEY-----"},
	// Provider token formats.
	{"anthropic-key", regexp.MustCompile(`sk-ant-[A-Za-z0-9_\-]{16,}`), "sk-ant-" + Placeholder},
	{"openai-key", regexp.MustCompile(`sk-(?:proj-|svcacct-|admin-)?[A-Za-z0-9_\-]{16,}`), "sk-" + Placeholder},
	{"stripe-key", regexp.MustCompile(`(?:sk|rk|pk)_(?:live|test)_[A-Za-z0-9]{16,}`), Placeholder},
	{"github-token", regexp.MustCompile(`(?:gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,})`), Placeholder},
	{"gitlab-token", regexp.MustCompile(`glpat-[A-Za-z0-9_\-]{20,}`), Placeholder},
	{"slack-token", regexp.MustCompile(`xox[abposr]-[A-Za-z0-9\-]{10,}`), Placeholder},
	{"slack-webhook", regexp.MustCompile(`https://hooks\.slack\.com/services/[A-Za-z0-9/_\-]+`), Placeholder},
	{"aws-access-key", regexp.MustCompile(`\b(?:AKIA|ASIA|AGPA|AIDA|AROA|ANPA|ANVA)[A-Z0-9]{16}\b`), Placeholder},
	{"google-api-key", regexp.MustCompile(`AIza[0-9A-Za-z_\-]{35}`), Placeholder},
	{"npm-token", regexp.MustCompile(`npm_[A-Za-z0-9]{30,}`), Placeholder},
	{"jwt", regexp.MustCompile(`eyJ[A-Za-z0-9_\-]{8,}\.eyJ[A-Za-z0-9_\-]{8,}\.[A-Za-z0-9_\-]{8,}`), Placeholder},
	// Authorization headers.
	{"auth-header", regexp.MustCompile(`(?i)\b(Bearer|Basic|Token)\s+[A-Za-z0-9\-._~+/=]{12,}`), "$1 " + Placeholder},
	// Credentials embedded in URLs / connection strings: scheme://user:pass@host
	{"url-credentials", regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9+.\-]*://)[^\s:/@'"]+:[^\s@'"]+@`), "${1}" + Placeholder + "@"},
	// YAML block scalars under a sensitive key: key: |\n  line\n  line
	{"yaml-block-secret", regexp.MustCompile(`(?m)^(\s*` + sensitiveKey + `\s*:\s*[|>][-+]?\s*\n)((?:[ \t]+.*(?:\n|$))+)`), "${1}  " + Placeholder + "\n"},
	// Triple-quoted values: KEY = """...""" or '''...'''
	{"triple-quoted-secret", regexp.MustCompile(`(?s)(` + sensitiveKey + `\s*[:=]\s*)("""|''')(.*?)("""|''')`), "${1}${2}" + Placeholder + "${4}"},
	// Assignments in env, shell, YAML, TOML, INI, JSON, Python, JS, Go:
	//   KEY=v  export KEY=v  key: v  "key": "v"  key = 'v'  key := "v"
	{"quoted-assignment", regexp.MustCompile(`(["']?` + sensitiveKey + `["']?\s*(?::=|=>|[:=])\s*)(["'` + "`" + `])([^"'` + "`" + `\n]{4,})(["'` + "`" + `])`), "${1}${2}" + Placeholder + "${4}"},
	{"bare-assignment", regexp.MustCompile(`(?m)((?:^|[\s;,{(])(?:export\s+)?` + sensitiveKey + `\s*(?::=|[:=])[ \t]*)([^\s"'` + "`" + `,;#}{)(\[\]][^\s,;#}{)(]{3,})`), "${1}" + Placeholder},
	// Personal email addresses.
	{"email", regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`), "<email-redacted>"},
}

// proseSecret catches "password <value>" with no assignment operator. Only
// values mixing letters and digits are replaced, so "password reset" survives.
var proseSecret = regexp.MustCompile(`(` + sensitiveKey + `[ \t]+)([A-Za-z0-9!#$%^&*_\-+.~]{8,})`)

func proseValueIsSecret(v string) bool {
	var letter, digit bool
	for _, r := range v {
		switch {
		case r >= '0' && r <= '9':
			digit = true
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			letter = true
		}
	}
	return letter && digit
}

func replaceProse(s string) string {
	return proseSecret.ReplaceAllStringFunc(s, func(m string) string {
		sub := proseSecret.FindStringSubmatch(m)
		if len(sub) == 3 && proseValueIsSecret(sub[2]) {
			return sub[1] + Placeholder
		}
		return m
	})
}

// highEntropy matches long base64/base62-like tokens. Hex-only strings (hashes,
// commit IDs) have at most 4 bits/char and are not matched by the threshold.
var highEntropy = regexp.MustCompile(`[A-Za-z0-9+/_\-]{32,}={0,2}`)

const entropyThreshold = 4.5

func shannon(s string) float64 {
	counts := map[rune]int{}
	for _, r := range s {
		counts[r]++
	}
	var h float64
	n := float64(len(s))
	for _, c := range counts {
		p := float64(c) / n
		h -= p * math.Log2(p)
	}
	return h
}

func looksLikeSecretToken(tok string) bool {
	if len(tok) < 32 || shannon(tok) < entropyThreshold {
		return false
	}
	var lower, upper, digit bool
	for _, r := range tok {
		switch {
		case r >= 'a' && r <= 'z':
			lower = true
		case r >= 'A' && r <= 'Z':
			upper = true
		case r >= '0' && r <= '9':
			digit = true
		}
	}
	return lower && upper && digit
}

// String returns s with all detected sensitive values replaced.
func String(s string) string {
	if s == "" {
		return s
	}
	for _, r := range rules {
		s = r.re.ReplaceAllString(s, r.repl)
	}
	s = replaceProse(s)
	return highEntropy.ReplaceAllStringFunc(s, func(tok string) string {
		if looksLikeSecretToken(tok) {
			return Placeholder
		}
		return tok
	})
}

// Finding names a detector that fired. It never carries the matched value, so
// it is safe to log or include in errors.
type Finding struct {
	Detector string `json:"detector"`
	Offset   int    `json:"offset"`
}

// Scan reports detectors that still fire on s. Used as an independent check
// on already-sanitized output; placeholders themselves never match.
func Scan(s string) []Finding {
	var out []Finding
	for _, r := range rules {
		for _, loc := range r.re.FindAllStringIndex(s, -1) {
			if strings.Contains(s[loc[0]:loc[1]], Placeholder) || strings.Contains(s[loc[0]:loc[1]], "<email-redacted>") {
				continue
			}
			out = append(out, Finding{Detector: r.name, Offset: loc[0]})
		}
	}
	for _, loc := range proseSecret.FindAllStringSubmatchIndex(s, -1) {
		if proseValueIsSecret(s[loc[4]:loc[5]]) {
			out = append(out, Finding{Detector: "prose-secret", Offset: loc[0]})
		}
	}
	for _, loc := range highEntropy.FindAllStringIndex(s, -1) {
		if looksLikeSecretToken(s[loc[0]:loc[1]]) {
			out = append(out, Finding{Detector: "high-entropy", Offset: loc[0]})
		}
	}
	return out
}

// Value sanitizes every string reachable from v in place: struct fields,
// slices, arrays, map values and pointers. New fields added to exported types
// are covered automatically, which is the point: privacy enforcement must not
// depend on remembering to list each field.
func Value(v any) {
	walk(reflect.ValueOf(v))
}

func walk(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			elem := v.Elem()
			if v.Kind() == reflect.Interface && elem.Kind() == reflect.String {
				if v.CanSet() {
					v.Set(reflect.ValueOf(String(elem.String())))
				}
				return
			}
			walk(elem)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if f := v.Field(i); f.CanSet() || f.Kind() == reflect.Pointer || f.Kind() == reflect.Slice || f.Kind() == reflect.Map {
				if v.Type().Field(i).IsExported() {
					walk(f)
				}
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			walk(v.Index(i))
		}
	case reflect.Map:
		if v.IsNil() {
			return
		}
		for _, k := range v.MapKeys() {
			val := v.MapIndex(k)
			if val.Kind() == reflect.String {
				v.SetMapIndex(k, reflect.ValueOf(String(val.String())).Convert(val.Type()))
				continue
			}
			copyVal := reflect.New(val.Type()).Elem()
			copyVal.Set(val)
			walk(copyVal)
			v.SetMapIndex(k, copyVal)
		}
	case reflect.String:
		if v.CanSet() {
			v.SetString(String(v.String()))
		}
	}
}

// sensitiveFileNames are files whose contents are never exported outside
// private mode, regardless of what detectors find.
var sensitiveFileNames = regexp.MustCompile(`(?i)(^|/)(\.env(\..*)?|\.envrc|\.npmrc|\.pypirc|\.netrc|\.git-credentials|id_(rsa|dsa|ecdsa|ed25519)|.*\.(pem|key|p12|pfx|keystore|jks)|credentials(\.[a-z]+)?|secrets?(\.[a-z]+)?|.*\.tfvars|.*\.tfstate)$`)

// IsSensitiveFile reports whether a path names a file whose contents must not
// leave the machine in redacted or public exports.
func IsSensitiveFile(path string) bool {
	return sensitiveFileNames.MatchString(strings.ReplaceAll(path, "\\", "/"))
}

// FieldFinding is a Finding located at a field path (e.g. "SelectedSnippets[2].Content").
type FieldFinding struct {
	Path     string `json:"path"`
	Detector string `json:"detector"`
}

// ScanValue runs Scan over every string reachable from v.
func ScanValue(v any) []FieldFinding {
	var out []FieldFinding
	scanWalk(reflect.ValueOf(v), "", &out)
	return out
}

func scanWalk(v reflect.Value, path string, out *[]FieldFinding) {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			scanWalk(v.Elem(), path, out)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if f := v.Type().Field(i); f.IsExported() {
				p := f.Name
				if path != "" {
					p = path + "." + f.Name
				}
				scanWalk(v.Field(i), p, out)
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			scanWalk(v.Index(i), path+"["+itoa(i)+"]", out)
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			scanWalk(v.MapIndex(k), path+"[key]", out)
		}
	case reflect.String:
		for _, f := range Scan(v.String()) {
			*out = append(*out, FieldFinding{Path: path, Detector: f.Detector})
		}
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}
