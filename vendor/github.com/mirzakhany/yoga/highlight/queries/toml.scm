; From github.com/tree-sitter-grammars/tree-sitter-toml@v0.7.0 queries/highlights.scm (MIT license).
; Properties
;-----------

(bare_key) @type

(quoted_key) @string

; yoga: the key itself as @property, not the whole pair, since a shorter
; capture is drawn over a longer one.
(pair
  (bare_key) @property)

(pair
  (dotted_key
    (bare_key) @property))

; Literals
;---------

(boolean) @boolean

(comment) @comment

(string) @string

[
  (integer)
  (float)
] @number

[
  (offset_date_time)
  (local_date_time)
  (local_date)
  (local_time)
] @string.special

; Punctuation
;------------

[
  "."
  ","
] @punctuation.delimiter

"=" @operator

[
  "["
  "]"
  "[["
  "]]"
  "{"
  "}"
] @punctuation.bracket
