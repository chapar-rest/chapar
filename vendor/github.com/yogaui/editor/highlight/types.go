package highlight

import "github.com/yogaui/yoga/syntax"

// The token and interface types live in the core syntax package, so themes
// and Markdown can use them without Tree-sitter. These aliases keep this
// package self-contained for callers.

type (
	ColorClass       = syntax.ColorClass
	Token            = syntax.Token
	Pt               = syntax.Pt
	Edit             = syntax.Edit
	Highlighter      = syntax.Highlighter
	RangeHighlighter = syntax.RangeHighlighter
	SizeLimited      = syntax.SizeLimited
	Noop             = syntax.Noop
	Factory          = syntax.Factory
)

const (
	ClassDefault     = syntax.ClassDefault
	ClassKeyword     = syntax.ClassKeyword
	ClassString      = syntax.ClassString
	ClassComment     = syntax.ClassComment
	ClassNumber      = syntax.ClassNumber
	ClassType        = syntax.ClassType
	ClassError       = syntax.ClassError
	ClassWarning     = syntax.ClassWarning
	ClassSuccess     = syntax.ClassSuccess
	ClassMuted       = syntax.ClassMuted
	ClassFunction    = syntax.ClassFunction
	ClassMethod      = syntax.ClassMethod
	ClassVariable    = syntax.ClassVariable
	ClassParameter   = syntax.ClassParameter
	ClassProperty    = syntax.ClassProperty
	ClassConstant    = syntax.ClassConstant
	ClassBuiltin     = syntax.ClassBuiltin
	ClassOperator    = syntax.ClassOperator
	ClassPunctuation = syntax.ClassPunctuation
	ClassTag         = syntax.ClassTag
	ClassAttribute   = syntax.ClassAttribute
	ClassNamespace   = syntax.ClassNamespace
)
