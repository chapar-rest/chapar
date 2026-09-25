#include "../../common/scanner.h"

#ifdef _MSC_VER
#pragma warning(push)
#pragma warning(disable : 4100)
#elif defined(__GNUC__) || defined(__clang__)
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wunused-parameter"
#endif

/// Check if the lexer is in error recovery mode
static inline bool in_error_recovery(const bool *valid_symbols) {
    return valid_symbols[PI_TARGET] && valid_symbols[PI_CONTENT] && valid_symbols[COMMENT];
}

bool tree_sitter_dtd_external_scanner_scan(void *payload, TSLexer *lexer, const bool *valid_symbols) {
    if (in_error_recovery(valid_symbols))
        return false;

    if (valid_symbols[PI_TARGET])
        return scan_pi_target(lexer, valid_symbols);

    if (valid_symbols[PI_CONTENT])
        return scan_pi_content(lexer);

    if (valid_symbols[COMMENT]) {
        advance_if_eq(lexer, '<');
        advance_if_eq(lexer, '!');
        return scan_comment(lexer);
    }

    return false;
}

void *tree_sitter_dtd_external_scanner_create() { return NULL; }

void tree_sitter_dtd_external_scanner_destroy(void *payload) {}

unsigned tree_sitter_dtd_external_scanner_serialize(void *payload, char *buffer) { return 0; }

void tree_sitter_dtd_external_scanner_deserialize(void *payload, const char *buffer, unsigned length) {}

#ifdef _MSC_VER
#pragma warning(pop)
#elif defined(__GNUC__) || defined(__clang__)
#pragma GCC diagnostic pop
#endif
