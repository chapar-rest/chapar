package codeeditor

import "github.com/tidwall/pretty"

func BeautifyCode(lang, code string) string {
	switch lang {
	case CodeLanguageJSON:
		return beautifyJSON(code)
	default:
		return code
	}
}

func beautifyJSON(inputJSON string) string {
	return string(pretty.PrettyOptions([]byte(inputJSON), &pretty.Options{
		Indent:   "    ",
		SortKeys: false,
	}))
}
