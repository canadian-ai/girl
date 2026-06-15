package lang

const (
	Go              = "go"
	TypeScript      = "typescript"
	TypeScriptReact = "typescriptreact"
	JavaScript      = "javascript"
	JavaScriptReact = "javascriptreact"
)

func IsGo(lang string) bool { return lang == Go }

func IsTypeScript(lang string) bool { return lang == TypeScript || lang == TypeScriptReact }

func Resolve(lang string) string {
	switch lang {
	case "go":
		return Go
	case "ts", "typescript":
		return TypeScript
	case "tsx", "typescriptreact":
		return TypeScriptReact
	case "js", "javascript":
		return JavaScript
	case "jsx", "javascriptreact":
		return JavaScriptReact
	}
	return lang
}
