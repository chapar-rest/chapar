#include "tree_sitter/parser.h"

#if defined(__GNUC__) || defined(__clang__)
#pragma GCC diagnostic ignored "-Wmissing-field-initializers"
#endif

#define LANGUAGE_VERSION 14
#define STATE_COUNT 334
#define LARGE_STATE_COUNT 2
#define SYMBOL_COUNT 111
#define ALIAS_COUNT 0
#define TOKEN_COUNT 61
#define EXTERNAL_TOKEN_COUNT 3
#define FIELD_COUNT 1
#define MAX_ALIAS_SEQUENCE_LENGTH 10
#define PRODUCTION_ID_COUNT 2

enum ts_symbol_identifiers {
  sym_Name = 1,
  anon_sym_LT_QMARK = 2,
  anon_sym_xml = 3,
  anon_sym_QMARK_GT = 4,
  anon_sym_LT_BANG_LBRACK = 5,
  anon_sym_IGNORE = 6,
  anon_sym_INCLUDE = 7,
  anon_sym_LBRACK = 8,
  anon_sym_RBRACK_RBRACK_GT = 9,
  anon_sym_LT_BANG = 10,
  anon_sym_ELEMENT = 11,
  anon_sym_GT = 12,
  anon_sym_EMPTY = 13,
  anon_sym_ANY = 14,
  anon_sym_LPAREN = 15,
  anon_sym_POUNDPCDATA = 16,
  anon_sym_PIPE = 17,
  anon_sym_RPAREN = 18,
  anon_sym_STAR = 19,
  anon_sym_QMARK = 20,
  anon_sym_PLUS = 21,
  anon_sym_COMMA = 22,
  anon_sym_ATTLIST = 23,
  sym_StringType = 24,
  sym_TokenizedType = 25,
  anon_sym_NOTATION = 26,
  anon_sym_POUNDREQUIRED = 27,
  anon_sym_POUNDIMPLIED = 28,
  anon_sym_POUNDFIXED = 29,
  anon_sym_ENTITY = 30,
  anon_sym_PERCENT = 31,
  anon_sym_DQUOTE = 32,
  aux_sym_EntityValue_token1 = 33,
  anon_sym_SQUOTE = 34,
  aux_sym_EntityValue_token2 = 35,
  anon_sym_NDATA = 36,
  anon_sym_SEMI = 37,
  sym__S = 38,
  sym_Nmtoken = 39,
  anon_sym_AMP = 40,
  anon_sym_AMP_POUND = 41,
  aux_sym_CharRef_token1 = 42,
  anon_sym_AMP_POUNDx = 43,
  aux_sym_CharRef_token2 = 44,
  aux_sym_AttValue_token1 = 45,
  aux_sym_AttValue_token2 = 46,
  anon_sym_SYSTEM = 47,
  anon_sym_PUBLIC = 48,
  aux_sym_SystemLiteral_token1 = 49,
  aux_sym_SystemLiteral_token2 = 50,
  aux_sym_PubidLiteral_token1 = 51,
  aux_sym_PubidLiteral_token2 = 52,
  anon_sym_version = 53,
  sym_VersionNum = 54,
  anon_sym_encoding = 55,
  sym_EncName = 56,
  anon_sym_EQ = 57,
  sym_PITarget = 58,
  sym__pi_content = 59,
  sym_Comment = 60,
  sym_extSubset = 61,
  sym_TextDecl = 62,
  sym__extSubsetDecl = 63,
  sym_conditionalSect = 64,
  sym__markupdecl = 65,
  sym__DeclSep = 66,
  sym_elementdecl = 67,
  sym_contentspec = 68,
  sym_Mixed = 69,
  sym_children = 70,
  sym__cp = 71,
  sym__choice = 72,
  sym_AttlistDecl = 73,
  sym_AttDef = 74,
  sym__AttType = 75,
  sym__EnumeratedType = 76,
  sym_NotationType = 77,
  sym_Enumeration = 78,
  sym_DefaultDecl = 79,
  sym__EntityDecl = 80,
  sym_GEDecl = 81,
  sym_PEDecl = 82,
  sym_EntityValue = 83,
  sym_NDataDecl = 84,
  sym_NotationDecl = 85,
  sym_PEReference = 86,
  sym__Reference = 87,
  sym_EntityRef = 88,
  sym_CharRef = 89,
  sym_AttValue = 90,
  sym_ExternalID = 91,
  sym_PublicID = 92,
  sym_SystemLiteral = 93,
  sym_PubidLiteral = 94,
  sym__VersionInfo = 95,
  sym__EncodingDecl = 96,
  sym_PI = 97,
  sym__Eq = 98,
  aux_sym_extSubset_repeat1 = 99,
  aux_sym_Mixed_repeat1 = 100,
  aux_sym_Mixed_repeat2 = 101,
  aux_sym__choice_repeat1 = 102,
  aux_sym__choice_repeat2 = 103,
  aux_sym_AttlistDecl_repeat1 = 104,
  aux_sym_NotationType_repeat1 = 105,
  aux_sym_Enumeration_repeat1 = 106,
  aux_sym_EntityValue_repeat1 = 107,
  aux_sym_EntityValue_repeat2 = 108,
  aux_sym_AttValue_repeat1 = 109,
  aux_sym_AttValue_repeat2 = 110,
};

static const char * const ts_symbol_names[] = {
  [ts_builtin_sym_end] = "end",
  [sym_Name] = "Name",
  [anon_sym_LT_QMARK] = "<\?",
  [anon_sym_xml] = "xml",
  [anon_sym_QMARK_GT] = "\?>",
  [anon_sym_LT_BANG_LBRACK] = "<![",
  [anon_sym_IGNORE] = "IGNORE",
  [anon_sym_INCLUDE] = "INCLUDE",
  [anon_sym_LBRACK] = "[",
  [anon_sym_RBRACK_RBRACK_GT] = "]]>",
  [anon_sym_LT_BANG] = "<!",
  [anon_sym_ELEMENT] = "ELEMENT",
  [anon_sym_GT] = ">",
  [anon_sym_EMPTY] = "EMPTY",
  [anon_sym_ANY] = "ANY",
  [anon_sym_LPAREN] = "(",
  [anon_sym_POUNDPCDATA] = "#PCDATA",
  [anon_sym_PIPE] = "|",
  [anon_sym_RPAREN] = ")",
  [anon_sym_STAR] = "*",
  [anon_sym_QMARK] = "\?",
  [anon_sym_PLUS] = "+",
  [anon_sym_COMMA] = ",",
  [anon_sym_ATTLIST] = "ATTLIST",
  [sym_StringType] = "StringType",
  [sym_TokenizedType] = "TokenizedType",
  [anon_sym_NOTATION] = "NOTATION",
  [anon_sym_POUNDREQUIRED] = "#REQUIRED",
  [anon_sym_POUNDIMPLIED] = "#IMPLIED",
  [anon_sym_POUNDFIXED] = "#FIXED",
  [anon_sym_ENTITY] = "ENTITY",
  [anon_sym_PERCENT] = "%",
  [anon_sym_DQUOTE] = "\"",
  [aux_sym_EntityValue_token1] = "EntityValue_token1",
  [anon_sym_SQUOTE] = "'",
  [aux_sym_EntityValue_token2] = "EntityValue_token2",
  [anon_sym_NDATA] = "NDATA",
  [anon_sym_SEMI] = ";",
  [sym__S] = "_S",
  [sym_Nmtoken] = "Nmtoken",
  [anon_sym_AMP] = "&",
  [anon_sym_AMP_POUND] = "&#",
  [aux_sym_CharRef_token1] = "CharRef_token1",
  [anon_sym_AMP_POUNDx] = "&#x",
  [aux_sym_CharRef_token2] = "CharRef_token2",
  [aux_sym_AttValue_token1] = "AttValue_token1",
  [aux_sym_AttValue_token2] = "AttValue_token2",
  [anon_sym_SYSTEM] = "SYSTEM",
  [anon_sym_PUBLIC] = "PUBLIC",
  [aux_sym_SystemLiteral_token1] = "URI",
  [aux_sym_SystemLiteral_token2] = "URI",
  [aux_sym_PubidLiteral_token1] = "PubidLiteral_token1",
  [aux_sym_PubidLiteral_token2] = "PubidLiteral_token2",
  [anon_sym_version] = "version",
  [sym_VersionNum] = "VersionNum",
  [anon_sym_encoding] = "encoding",
  [sym_EncName] = "EncName",
  [anon_sym_EQ] = "=",
  [sym_PITarget] = "PITarget",
  [sym__pi_content] = "_pi_content",
  [sym_Comment] = "Comment",
  [sym_extSubset] = "extSubset",
  [sym_TextDecl] = "TextDecl",
  [sym__extSubsetDecl] = "_extSubsetDecl",
  [sym_conditionalSect] = "conditionalSect",
  [sym__markupdecl] = "_markupdecl",
  [sym__DeclSep] = "_DeclSep",
  [sym_elementdecl] = "elementdecl",
  [sym_contentspec] = "contentspec",
  [sym_Mixed] = "Mixed",
  [sym_children] = "children",
  [sym__cp] = "_cp",
  [sym__choice] = "_choice",
  [sym_AttlistDecl] = "AttlistDecl",
  [sym_AttDef] = "AttDef",
  [sym__AttType] = "_AttType",
  [sym__EnumeratedType] = "_EnumeratedType",
  [sym_NotationType] = "NotationType",
  [sym_Enumeration] = "Enumeration",
  [sym_DefaultDecl] = "DefaultDecl",
  [sym__EntityDecl] = "_EntityDecl",
  [sym_GEDecl] = "GEDecl",
  [sym_PEDecl] = "PEDecl",
  [sym_EntityValue] = "EntityValue",
  [sym_NDataDecl] = "NDataDecl",
  [sym_NotationDecl] = "NotationDecl",
  [sym_PEReference] = "PEReference",
  [sym__Reference] = "_Reference",
  [sym_EntityRef] = "EntityRef",
  [sym_CharRef] = "CharRef",
  [sym_AttValue] = "AttValue",
  [sym_ExternalID] = "ExternalID",
  [sym_PublicID] = "PublicID",
  [sym_SystemLiteral] = "SystemLiteral",
  [sym_PubidLiteral] = "PubidLiteral",
  [sym__VersionInfo] = "_VersionInfo",
  [sym__EncodingDecl] = "_EncodingDecl",
  [sym_PI] = "PI",
  [sym__Eq] = "_Eq",
  [aux_sym_extSubset_repeat1] = "extSubset_repeat1",
  [aux_sym_Mixed_repeat1] = "Mixed_repeat1",
  [aux_sym_Mixed_repeat2] = "Mixed_repeat2",
  [aux_sym__choice_repeat1] = "_choice_repeat1",
  [aux_sym__choice_repeat2] = "_choice_repeat2",
  [aux_sym_AttlistDecl_repeat1] = "AttlistDecl_repeat1",
  [aux_sym_NotationType_repeat1] = "NotationType_repeat1",
  [aux_sym_Enumeration_repeat1] = "Enumeration_repeat1",
  [aux_sym_EntityValue_repeat1] = "EntityValue_repeat1",
  [aux_sym_EntityValue_repeat2] = "EntityValue_repeat2",
  [aux_sym_AttValue_repeat1] = "AttValue_repeat1",
  [aux_sym_AttValue_repeat2] = "AttValue_repeat2",
};

static const TSSymbol ts_symbol_map[] = {
  [ts_builtin_sym_end] = ts_builtin_sym_end,
  [sym_Name] = sym_Name,
  [anon_sym_LT_QMARK] = anon_sym_LT_QMARK,
  [anon_sym_xml] = anon_sym_xml,
  [anon_sym_QMARK_GT] = anon_sym_QMARK_GT,
  [anon_sym_LT_BANG_LBRACK] = anon_sym_LT_BANG_LBRACK,
  [anon_sym_IGNORE] = anon_sym_IGNORE,
  [anon_sym_INCLUDE] = anon_sym_INCLUDE,
  [anon_sym_LBRACK] = anon_sym_LBRACK,
  [anon_sym_RBRACK_RBRACK_GT] = anon_sym_RBRACK_RBRACK_GT,
  [anon_sym_LT_BANG] = anon_sym_LT_BANG,
  [anon_sym_ELEMENT] = anon_sym_ELEMENT,
  [anon_sym_GT] = anon_sym_GT,
  [anon_sym_EMPTY] = anon_sym_EMPTY,
  [anon_sym_ANY] = anon_sym_ANY,
  [anon_sym_LPAREN] = anon_sym_LPAREN,
  [anon_sym_POUNDPCDATA] = anon_sym_POUNDPCDATA,
  [anon_sym_PIPE] = anon_sym_PIPE,
  [anon_sym_RPAREN] = anon_sym_RPAREN,
  [anon_sym_STAR] = anon_sym_STAR,
  [anon_sym_QMARK] = anon_sym_QMARK,
  [anon_sym_PLUS] = anon_sym_PLUS,
  [anon_sym_COMMA] = anon_sym_COMMA,
  [anon_sym_ATTLIST] = anon_sym_ATTLIST,
  [sym_StringType] = sym_StringType,
  [sym_TokenizedType] = sym_TokenizedType,
  [anon_sym_NOTATION] = anon_sym_NOTATION,
  [anon_sym_POUNDREQUIRED] = anon_sym_POUNDREQUIRED,
  [anon_sym_POUNDIMPLIED] = anon_sym_POUNDIMPLIED,
  [anon_sym_POUNDFIXED] = anon_sym_POUNDFIXED,
  [anon_sym_ENTITY] = anon_sym_ENTITY,
  [anon_sym_PERCENT] = anon_sym_PERCENT,
  [anon_sym_DQUOTE] = anon_sym_DQUOTE,
  [aux_sym_EntityValue_token1] = aux_sym_EntityValue_token1,
  [anon_sym_SQUOTE] = anon_sym_SQUOTE,
  [aux_sym_EntityValue_token2] = aux_sym_EntityValue_token2,
  [anon_sym_NDATA] = anon_sym_NDATA,
  [anon_sym_SEMI] = anon_sym_SEMI,
  [sym__S] = sym__S,
  [sym_Nmtoken] = sym_Nmtoken,
  [anon_sym_AMP] = anon_sym_AMP,
  [anon_sym_AMP_POUND] = anon_sym_AMP_POUND,
  [aux_sym_CharRef_token1] = aux_sym_CharRef_token1,
  [anon_sym_AMP_POUNDx] = anon_sym_AMP_POUNDx,
  [aux_sym_CharRef_token2] = aux_sym_CharRef_token2,
  [aux_sym_AttValue_token1] = aux_sym_AttValue_token1,
  [aux_sym_AttValue_token2] = aux_sym_AttValue_token2,
  [anon_sym_SYSTEM] = anon_sym_SYSTEM,
  [anon_sym_PUBLIC] = anon_sym_PUBLIC,
  [aux_sym_SystemLiteral_token1] = aux_sym_SystemLiteral_token1,
  [aux_sym_SystemLiteral_token2] = aux_sym_SystemLiteral_token1,
  [aux_sym_PubidLiteral_token1] = aux_sym_PubidLiteral_token1,
  [aux_sym_PubidLiteral_token2] = aux_sym_PubidLiteral_token2,
  [anon_sym_version] = anon_sym_version,
  [sym_VersionNum] = sym_VersionNum,
  [anon_sym_encoding] = anon_sym_encoding,
  [sym_EncName] = sym_EncName,
  [anon_sym_EQ] = anon_sym_EQ,
  [sym_PITarget] = sym_PITarget,
  [sym__pi_content] = sym__pi_content,
  [sym_Comment] = sym_Comment,
  [sym_extSubset] = sym_extSubset,
  [sym_TextDecl] = sym_TextDecl,
  [sym__extSubsetDecl] = sym__extSubsetDecl,
  [sym_conditionalSect] = sym_conditionalSect,
  [sym__markupdecl] = sym__markupdecl,
  [sym__DeclSep] = sym__DeclSep,
  [sym_elementdecl] = sym_elementdecl,
  [sym_contentspec] = sym_contentspec,
  [sym_Mixed] = sym_Mixed,
  [sym_children] = sym_children,
  [sym__cp] = sym__cp,
  [sym__choice] = sym__choice,
  [sym_AttlistDecl] = sym_AttlistDecl,
  [sym_AttDef] = sym_AttDef,
  [sym__AttType] = sym__AttType,
  [sym__EnumeratedType] = sym__EnumeratedType,
  [sym_NotationType] = sym_NotationType,
  [sym_Enumeration] = sym_Enumeration,
  [sym_DefaultDecl] = sym_DefaultDecl,
  [sym__EntityDecl] = sym__EntityDecl,
  [sym_GEDecl] = sym_GEDecl,
  [sym_PEDecl] = sym_PEDecl,
  [sym_EntityValue] = sym_EntityValue,
  [sym_NDataDecl] = sym_NDataDecl,
  [sym_NotationDecl] = sym_NotationDecl,
  [sym_PEReference] = sym_PEReference,
  [sym__Reference] = sym__Reference,
  [sym_EntityRef] = sym_EntityRef,
  [sym_CharRef] = sym_CharRef,
  [sym_AttValue] = sym_AttValue,
  [sym_ExternalID] = sym_ExternalID,
  [sym_PublicID] = sym_PublicID,
  [sym_SystemLiteral] = sym_SystemLiteral,
  [sym_PubidLiteral] = sym_PubidLiteral,
  [sym__VersionInfo] = sym__VersionInfo,
  [sym__EncodingDecl] = sym__EncodingDecl,
  [sym_PI] = sym_PI,
  [sym__Eq] = sym__Eq,
  [aux_sym_extSubset_repeat1] = aux_sym_extSubset_repeat1,
  [aux_sym_Mixed_repeat1] = aux_sym_Mixed_repeat1,
  [aux_sym_Mixed_repeat2] = aux_sym_Mixed_repeat2,
  [aux_sym__choice_repeat1] = aux_sym__choice_repeat1,
  [aux_sym__choice_repeat2] = aux_sym__choice_repeat2,
  [aux_sym_AttlistDecl_repeat1] = aux_sym_AttlistDecl_repeat1,
  [aux_sym_NotationType_repeat1] = aux_sym_NotationType_repeat1,
  [aux_sym_Enumeration_repeat1] = aux_sym_Enumeration_repeat1,
  [aux_sym_EntityValue_repeat1] = aux_sym_EntityValue_repeat1,
  [aux_sym_EntityValue_repeat2] = aux_sym_EntityValue_repeat2,
  [aux_sym_AttValue_repeat1] = aux_sym_AttValue_repeat1,
  [aux_sym_AttValue_repeat2] = aux_sym_AttValue_repeat2,
};

static const TSSymbolMetadata ts_symbol_metadata[] = {
  [ts_builtin_sym_end] = {
    .visible = false,
    .named = true,
  },
  [sym_Name] = {
    .visible = true,
    .named = true,
  },
  [anon_sym_LT_QMARK] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_xml] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_QMARK_GT] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_LT_BANG_LBRACK] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_IGNORE] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_INCLUDE] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_LBRACK] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_RBRACK_RBRACK_GT] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_LT_BANG] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_ELEMENT] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_GT] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_EMPTY] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_ANY] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_LPAREN] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_POUNDPCDATA] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_PIPE] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_RPAREN] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_STAR] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_QMARK] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_PLUS] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_COMMA] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_ATTLIST] = {
    .visible = true,
    .named = false,
  },
  [sym_StringType] = {
    .visible = true,
    .named = true,
  },
  [sym_TokenizedType] = {
    .visible = true,
    .named = true,
  },
  [anon_sym_NOTATION] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_POUNDREQUIRED] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_POUNDIMPLIED] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_POUNDFIXED] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_ENTITY] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_PERCENT] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_DQUOTE] = {
    .visible = true,
    .named = false,
  },
  [aux_sym_EntityValue_token1] = {
    .visible = false,
    .named = false,
  },
  [anon_sym_SQUOTE] = {
    .visible = true,
    .named = false,
  },
  [aux_sym_EntityValue_token2] = {
    .visible = false,
    .named = false,
  },
  [anon_sym_NDATA] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_SEMI] = {
    .visible = true,
    .named = false,
  },
  [sym__S] = {
    .visible = false,
    .named = true,
  },
  [sym_Nmtoken] = {
    .visible = true,
    .named = true,
  },
  [anon_sym_AMP] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_AMP_POUND] = {
    .visible = true,
    .named = false,
  },
  [aux_sym_CharRef_token1] = {
    .visible = false,
    .named = false,
  },
  [anon_sym_AMP_POUNDx] = {
    .visible = true,
    .named = false,
  },
  [aux_sym_CharRef_token2] = {
    .visible = false,
    .named = false,
  },
  [aux_sym_AttValue_token1] = {
    .visible = false,
    .named = false,
  },
  [aux_sym_AttValue_token2] = {
    .visible = false,
    .named = false,
  },
  [anon_sym_SYSTEM] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_PUBLIC] = {
    .visible = true,
    .named = false,
  },
  [aux_sym_SystemLiteral_token1] = {
    .visible = true,
    .named = true,
  },
  [aux_sym_SystemLiteral_token2] = {
    .visible = true,
    .named = true,
  },
  [aux_sym_PubidLiteral_token1] = {
    .visible = false,
    .named = false,
  },
  [aux_sym_PubidLiteral_token2] = {
    .visible = false,
    .named = false,
  },
  [anon_sym_version] = {
    .visible = true,
    .named = false,
  },
  [sym_VersionNum] = {
    .visible = true,
    .named = true,
  },
  [anon_sym_encoding] = {
    .visible = true,
    .named = false,
  },
  [sym_EncName] = {
    .visible = true,
    .named = true,
  },
  [anon_sym_EQ] = {
    .visible = true,
    .named = false,
  },
  [sym_PITarget] = {
    .visible = true,
    .named = true,
  },
  [sym__pi_content] = {
    .visible = false,
    .named = true,
  },
  [sym_Comment] = {
    .visible = true,
    .named = true,
  },
  [sym_extSubset] = {
    .visible = true,
    .named = true,
  },
  [sym_TextDecl] = {
    .visible = true,
    .named = true,
  },
  [sym__extSubsetDecl] = {
    .visible = false,
    .named = true,
  },
  [sym_conditionalSect] = {
    .visible = true,
    .named = true,
  },
  [sym__markupdecl] = {
    .visible = false,
    .named = true,
    .supertype = true,
  },
  [sym__DeclSep] = {
    .visible = false,
    .named = true,
  },
  [sym_elementdecl] = {
    .visible = true,
    .named = true,
  },
  [sym_contentspec] = {
    .visible = true,
    .named = true,
  },
  [sym_Mixed] = {
    .visible = true,
    .named = true,
  },
  [sym_children] = {
    .visible = true,
    .named = true,
  },
  [sym__cp] = {
    .visible = false,
    .named = true,
  },
  [sym__choice] = {
    .visible = false,
    .named = true,
  },
  [sym_AttlistDecl] = {
    .visible = true,
    .named = true,
  },
  [sym_AttDef] = {
    .visible = true,
    .named = true,
  },
  [sym__AttType] = {
    .visible = false,
    .named = true,
    .supertype = true,
  },
  [sym__EnumeratedType] = {
    .visible = false,
    .named = true,
    .supertype = true,
  },
  [sym_NotationType] = {
    .visible = true,
    .named = true,
  },
  [sym_Enumeration] = {
    .visible = true,
    .named = true,
  },
  [sym_DefaultDecl] = {
    .visible = true,
    .named = true,
  },
  [sym__EntityDecl] = {
    .visible = false,
    .named = true,
    .supertype = true,
  },
  [sym_GEDecl] = {
    .visible = true,
    .named = true,
  },
  [sym_PEDecl] = {
    .visible = true,
    .named = true,
  },
  [sym_EntityValue] = {
    .visible = true,
    .named = true,
  },
  [sym_NDataDecl] = {
    .visible = true,
    .named = true,
  },
  [sym_NotationDecl] = {
    .visible = true,
    .named = true,
  },
  [sym_PEReference] = {
    .visible = true,
    .named = true,
  },
  [sym__Reference] = {
    .visible = false,
    .named = true,
    .supertype = true,
  },
  [sym_EntityRef] = {
    .visible = true,
    .named = true,
  },
  [sym_CharRef] = {
    .visible = true,
    .named = true,
  },
  [sym_AttValue] = {
    .visible = true,
    .named = true,
  },
  [sym_ExternalID] = {
    .visible = true,
    .named = true,
  },
  [sym_PublicID] = {
    .visible = true,
    .named = true,
  },
  [sym_SystemLiteral] = {
    .visible = true,
    .named = true,
  },
  [sym_PubidLiteral] = {
    .visible = true,
    .named = true,
  },
  [sym__VersionInfo] = {
    .visible = false,
    .named = true,
  },
  [sym__EncodingDecl] = {
    .visible = false,
    .named = true,
  },
  [sym_PI] = {
    .visible = true,
    .named = true,
  },
  [sym__Eq] = {
    .visible = false,
    .named = true,
  },
  [aux_sym_extSubset_repeat1] = {
    .visible = false,
    .named = false,
  },
  [aux_sym_Mixed_repeat1] = {
    .visible = false,
    .named = false,
  },
  [aux_sym_Mixed_repeat2] = {
    .visible = false,
    .named = false,
  },
  [aux_sym__choice_repeat1] = {
    .visible = false,
    .named = false,
  },
  [aux_sym__choice_repeat2] = {
    .visible = false,
    .named = false,
  },
  [aux_sym_AttlistDecl_repeat1] = {
    .visible = false,
    .named = false,
  },
  [aux_sym_NotationType_repeat1] = {
    .visible = false,
    .named = false,
  },
  [aux_sym_Enumeration_repeat1] = {
    .visible = false,
    .named = false,
  },
  [aux_sym_EntityValue_repeat1] = {
    .visible = false,
    .named = false,
  },
  [aux_sym_EntityValue_repeat2] = {
    .visible = false,
    .named = false,
  },
  [aux_sym_AttValue_repeat1] = {
    .visible = false,
    .named = false,
  },
  [aux_sym_AttValue_repeat2] = {
    .visible = false,
    .named = false,
  },
};

enum ts_field_identifiers {
  field_content = 1,
};

static const char * const ts_field_names[] = {
  [0] = NULL,
  [field_content] = "content",
};

static const TSFieldMapSlice ts_field_map_slices[PRODUCTION_ID_COUNT] = {
  [1] = {.index = 0, .length = 1},
};

static const TSFieldMapEntry ts_field_map_entries[] = {
  [0] =
    {field_content, 1},
};

static const TSSymbol ts_alias_sequences[PRODUCTION_ID_COUNT][MAX_ALIAS_SEQUENCE_LENGTH] = {
  [0] = {0},
};

static const uint16_t ts_non_terminal_alias_map[] = {
  0,
};

static const TSStateId ts_primary_state_ids[STATE_COUNT] = {
  [0] = 0,
  [1] = 1,
  [2] = 2,
  [3] = 3,
  [4] = 4,
  [5] = 5,
  [6] = 6,
  [7] = 7,
  [8] = 8,
  [9] = 9,
  [10] = 10,
  [11] = 11,
  [12] = 12,
  [13] = 13,
  [14] = 14,
  [15] = 15,
  [16] = 16,
  [17] = 17,
  [18] = 18,
  [19] = 19,
  [20] = 20,
  [21] = 21,
  [22] = 22,
  [23] = 23,
  [24] = 24,
  [25] = 25,
  [26] = 26,
  [27] = 27,
  [28] = 28,
  [29] = 29,
  [30] = 30,
  [31] = 31,
  [32] = 32,
  [33] = 33,
  [34] = 34,
  [35] = 35,
  [36] = 36,
  [37] = 37,
  [38] = 38,
  [39] = 39,
  [40] = 40,
  [41] = 41,
  [42] = 42,
  [43] = 43,
  [44] = 44,
  [45] = 45,
  [46] = 17,
  [47] = 47,
  [48] = 48,
  [49] = 49,
  [50] = 50,
  [51] = 51,
  [52] = 52,
  [53] = 53,
  [54] = 54,
  [55] = 55,
  [56] = 56,
  [57] = 57,
  [58] = 58,
  [59] = 59,
  [60] = 60,
  [61] = 61,
  [62] = 62,
  [63] = 63,
  [64] = 64,
  [65] = 65,
  [66] = 66,
  [67] = 67,
  [68] = 68,
  [69] = 69,
  [70] = 70,
  [71] = 71,
  [72] = 72,
  [73] = 73,
  [74] = 17,
  [75] = 75,
  [76] = 76,
  [77] = 77,
  [78] = 78,
  [79] = 79,
  [80] = 80,
  [81] = 81,
  [82] = 82,
  [83] = 83,
  [84] = 75,
  [85] = 85,
  [86] = 86,
  [87] = 79,
  [88] = 17,
  [89] = 89,
  [90] = 82,
  [91] = 91,
  [92] = 92,
  [93] = 93,
  [94] = 94,
  [95] = 95,
  [96] = 75,
  [97] = 79,
  [98] = 75,
  [99] = 99,
  [100] = 100,
  [101] = 101,
  [102] = 82,
  [103] = 82,
  [104] = 104,
  [105] = 105,
  [106] = 79,
  [107] = 107,
  [108] = 108,
  [109] = 109,
  [110] = 110,
  [111] = 111,
  [112] = 112,
  [113] = 113,
  [114] = 114,
  [115] = 115,
  [116] = 116,
  [117] = 117,
  [118] = 118,
  [119] = 119,
  [120] = 120,
  [121] = 121,
  [122] = 122,
  [123] = 123,
  [124] = 124,
  [125] = 125,
  [126] = 126,
  [127] = 127,
  [128] = 128,
  [129] = 129,
  [130] = 130,
  [131] = 131,
  [132] = 132,
  [133] = 133,
  [134] = 134,
  [135] = 135,
  [136] = 136,
  [137] = 137,
  [138] = 138,
  [139] = 139,
  [140] = 140,
  [141] = 141,
  [142] = 142,
  [143] = 143,
  [144] = 144,
  [145] = 145,
  [146] = 146,
  [147] = 147,
  [148] = 148,
  [149] = 149,
  [150] = 150,
  [151] = 151,
  [152] = 152,
  [153] = 153,
  [154] = 154,
  [155] = 155,
  [156] = 156,
  [157] = 157,
  [158] = 158,
  [159] = 159,
  [160] = 160,
  [161] = 161,
  [162] = 162,
  [163] = 163,
  [164] = 164,
  [165] = 165,
  [166] = 166,
  [167] = 167,
  [168] = 168,
  [169] = 169,
  [170] = 170,
  [171] = 171,
  [172] = 172,
  [173] = 173,
  [174] = 174,
  [175] = 175,
  [176] = 176,
  [177] = 177,
  [178] = 178,
  [179] = 179,
  [180] = 180,
  [181] = 181,
  [182] = 182,
  [183] = 183,
  [184] = 184,
  [185] = 185,
  [186] = 186,
  [187] = 187,
  [188] = 188,
  [189] = 189,
  [190] = 190,
  [191] = 191,
  [192] = 192,
  [193] = 193,
  [194] = 194,
  [195] = 195,
  [196] = 196,
  [197] = 197,
  [198] = 198,
  [199] = 199,
  [200] = 200,
  [201] = 201,
  [202] = 202,
  [203] = 203,
  [204] = 204,
  [205] = 205,
  [206] = 206,
  [207] = 207,
  [208] = 208,
  [209] = 209,
  [210] = 210,
  [211] = 211,
  [212] = 212,
  [213] = 213,
  [214] = 214,
  [215] = 215,
  [216] = 216,
  [217] = 217,
  [218] = 218,
  [219] = 219,
  [220] = 220,
  [221] = 221,
  [222] = 222,
  [223] = 223,
  [224] = 224,
  [225] = 225,
  [226] = 226,
  [227] = 227,
  [228] = 228,
  [229] = 229,
  [230] = 230,
  [231] = 231,
  [232] = 232,
  [233] = 233,
  [234] = 234,
  [235] = 235,
  [236] = 236,
  [237] = 237,
  [238] = 238,
  [239] = 239,
  [240] = 240,
  [241] = 241,
  [242] = 242,
  [243] = 243,
  [244] = 244,
  [245] = 245,
  [246] = 246,
  [247] = 247,
  [248] = 248,
  [249] = 249,
  [250] = 250,
  [251] = 251,
  [252] = 252,
  [253] = 253,
  [254] = 254,
  [255] = 255,
  [256] = 256,
  [257] = 257,
  [258] = 258,
  [259] = 259,
  [260] = 260,
  [261] = 261,
  [262] = 262,
  [263] = 263,
  [264] = 264,
  [265] = 265,
  [266] = 266,
  [267] = 267,
  [268] = 268,
  [269] = 269,
  [270] = 270,
  [271] = 271,
  [272] = 272,
  [273] = 273,
  [274] = 274,
  [275] = 275,
  [276] = 276,
  [277] = 277,
  [278] = 278,
  [279] = 279,
  [280] = 280,
  [281] = 281,
  [282] = 282,
  [283] = 283,
  [284] = 284,
  [285] = 285,
  [286] = 286,
  [287] = 287,
  [288] = 288,
  [289] = 289,
  [290] = 290,
  [291] = 291,
  [292] = 292,
  [293] = 293,
  [294] = 294,
  [295] = 295,
  [296] = 296,
  [297] = 297,
  [298] = 298,
  [299] = 299,
  [300] = 300,
  [301] = 301,
  [302] = 302,
  [303] = 303,
  [304] = 304,
  [305] = 305,
  [306] = 306,
  [307] = 307,
  [308] = 308,
  [309] = 309,
  [310] = 310,
  [311] = 311,
  [312] = 312,
  [313] = 303,
  [314] = 250,
  [315] = 246,
  [316] = 303,
  [317] = 250,
  [318] = 246,
  [319] = 303,
  [320] = 250,
  [321] = 246,
  [322] = 312,
  [323] = 291,
  [324] = 293,
  [325] = 290,
  [326] = 312,
  [327] = 291,
  [328] = 293,
  [329] = 290,
  [330] = 312,
  [331] = 291,
  [332] = 293,
  [333] = 290,
};

static TSCharacterRange aux_sym_PubidLiteral_token1_character_set_1[] = {
  {'\n', '\n'}, {'\r', '\r'}, {' ', '!'}, {'#', '%'}, {'\'', ';'}, {'=', '='}, {'?', 'Z'}, {'_', '_'},
  {'a', 'z'},
};

static TSCharacterRange aux_sym_PubidLiteral_token2_character_set_1[] = {
  {'\n', '\n'}, {'\r', '\r'}, {' ', '!'}, {'#', '%'}, {'(', ';'}, {'=', '='}, {'?', 'Z'}, {'_', '_'},
  {'a', 'z'},
};

static bool ts_lex(TSLexer *lexer, TSStateId state) {
  START_LEXER();
  eof = lexer->eof(lexer);
  switch (state) {
    case 0:
      if (eof) ADVANCE(40);
      ADVANCE_MAP(
        '"', 66,
        '#', 70,
        '%', 65,
        '&', 120,
        '\'', 80,
        '(', 48,
        ')', 51,
        '*', 52,
        '+', 54,
        ',', 55,
        '1', 68,
        ';', 82,
        '<', 1,
        '=', 133,
        '>', 47,
        '?', 53,
        'E', 72,
        'I', 69,
        'N', 71,
        '[', 44,
        ']', 73,
        '_', 79,
        '|', 50,
        '\t', 76,
        '\n', 76,
        '\r', 76,
        ' ', 76,
        '-', 78,
        '.', 78,
        ':', 78,
        0xb7, 78,
      );
      if (('0' <= lookahead && lookahead <= '9')) ADVANCE(77);
      if (('A' <= lookahead && lookahead <= 'F') ||
          ('a' <= lookahead && lookahead <= 'f')) ADVANCE(74);
      if (('G' <= lookahead && lookahead <= 'Z') ||
          ('g' <= lookahead && lookahead <= 'z')) ADVANCE(75);
      if (lookahead != 0) ADVANCE(67);
      END_STATE();
    case 1:
      if (lookahead == '!') ADVANCE(46);
      if (lookahead == '?') ADVANCE(41);
      END_STATE();
    case 2:
      if (lookahead == '"') ADVANCE(66);
      if (lookahead == '%') ADVANCE(65);
      if (lookahead == '&') ADVANCE(120);
      if (lookahead != 0 &&
          lookahead != '<') ADVANCE(67);
      END_STATE();
    case 3:
      if (lookahead == '"') ADVANCE(66);
      if (lookahead == '&') ADVANCE(120);
      if (lookahead != 0 &&
          lookahead != '<') ADVANCE(125);
      END_STATE();
    case 4:
      if (lookahead == '%') ADVANCE(65);
      if (lookahead == '&') ADVANCE(120);
      if (lookahead == '\'') ADVANCE(80);
      if (lookahead != 0 &&
          lookahead != '<') ADVANCE(81);
      END_STATE();
    case 5:
      ADVANCE_MAP(
        '%', 65,
        '(', 48,
        '?', 9,
        'E', 101,
        'I', 84,
        'N', 99,
        '\t', 83,
        '\n', 83,
        '\r', 83,
        ' ', 83,
      );
      if (('0' <= lookahead && lookahead <= '9')) ADVANCE(122);
      if (('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(115);
      END_STATE();
    case 6:
      if (lookahead == '&') ADVANCE(120);
      if (lookahead == '\'') ADVANCE(80);
      if (lookahead != 0 &&
          lookahead != '<') ADVANCE(126);
      END_STATE();
    case 7:
      if (lookahead == '.') ADVANCE(36);
      END_STATE();
    case 8:
      if (lookahead == '>') ADVANCE(45);
      END_STATE();
    case 9:
      if (lookahead == '>') ADVANCE(42);
      END_STATE();
    case 10:
      if (lookahead == '?') ADVANCE(9);
      if (('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(115);
      END_STATE();
    case 11:
      if (lookahead == 'A') ADVANCE(31);
      END_STATE();
    case 12:
      if (lookahead == 'A') ADVANCE(49);
      END_STATE();
    case 13:
      if (lookahead == 'C') ADVANCE(14);
      END_STATE();
    case 14:
      if (lookahead == 'D') ADVANCE(11);
      END_STATE();
    case 15:
      if (lookahead == 'D') ADVANCE(64);
      END_STATE();
    case 16:
      if (lookahead == 'D') ADVANCE(63);
      END_STATE();
    case 17:
      if (lookahead == 'D') ADVANCE(62);
      END_STATE();
    case 18:
      if (lookahead == 'E') ADVANCE(29);
      END_STATE();
    case 19:
      if (lookahead == 'E') ADVANCE(15);
      END_STATE();
    case 20:
      if (lookahead == 'E') ADVANCE(16);
      END_STATE();
    case 21:
      if (lookahead == 'E') ADVANCE(17);
      END_STATE();
    case 22:
      if (lookahead == 'F') ADVANCE(23);
      if (lookahead == 'I') ADVANCE(27);
      if (lookahead == 'P') ADVANCE(13);
      if (lookahead == 'R') ADVANCE(18);
      END_STATE();
    case 23:
      if (lookahead == 'I') ADVANCE(33);
      END_STATE();
    case 24:
      if (lookahead == 'I') ADVANCE(30);
      END_STATE();
    case 25:
      if (lookahead == 'I') ADVANCE(20);
      END_STATE();
    case 26:
      if (lookahead == 'L') ADVANCE(25);
      END_STATE();
    case 27:
      if (lookahead == 'M') ADVANCE(28);
      END_STATE();
    case 28:
      if (lookahead == 'P') ADVANCE(26);
      END_STATE();
    case 29:
      if (lookahead == 'Q') ADVANCE(32);
      END_STATE();
    case 30:
      if (lookahead == 'R') ADVANCE(21);
      END_STATE();
    case 31:
      if (lookahead == 'T') ADVANCE(12);
      END_STATE();
    case 32:
      if (lookahead == 'U') ADVANCE(24);
      END_STATE();
    case 33:
      if (lookahead == 'X') ADVANCE(19);
      END_STATE();
    case 34:
      if (lookahead == ']') ADVANCE(8);
      END_STATE();
    case 35:
      if (lookahead == '\t' ||
          lookahead == '\n' ||
          lookahead == '\r' ||
          lookahead == ' ') ADVANCE(83);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(119);
      END_STATE();
    case 36:
      if (('0' <= lookahead && lookahead <= '9')) ADVANCE(131);
      END_STATE();
    case 37:
      if (('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'F') ||
          ('a' <= lookahead && lookahead <= 'f')) ADVANCE(124);
      END_STATE();
    case 38:
      if (('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(132);
      END_STATE();
    case 39:
      if (eof) ADVANCE(40);
      ADVANCE_MAP(
        '"', 66,
        '#', 22,
        '%', 65,
        '\'', 80,
        '(', 48,
        ')', 51,
        '*', 52,
        '+', 54,
        ',', 55,
        '1', 7,
        ';', 82,
        '<', 1,
        '=', 133,
        '>', 47,
        '?', 53,
        '[', 44,
        ']', 34,
        '|', 50,
        '\t', 83,
        '\n', 83,
        '\r', 83,
        ' ', 83,
      );
      if (('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(115);
      END_STATE();
    case 40:
      ACCEPT_TOKEN(ts_builtin_sym_end);
      END_STATE();
    case 41:
      ACCEPT_TOKEN(anon_sym_LT_QMARK);
      END_STATE();
    case 42:
      ACCEPT_TOKEN(anon_sym_QMARK_GT);
      END_STATE();
    case 43:
      ACCEPT_TOKEN(anon_sym_LT_BANG_LBRACK);
      END_STATE();
    case 44:
      ACCEPT_TOKEN(anon_sym_LBRACK);
      END_STATE();
    case 45:
      ACCEPT_TOKEN(anon_sym_RBRACK_RBRACK_GT);
      END_STATE();
    case 46:
      ACCEPT_TOKEN(anon_sym_LT_BANG);
      if (lookahead == '[') ADVANCE(43);
      END_STATE();
    case 47:
      ACCEPT_TOKEN(anon_sym_GT);
      END_STATE();
    case 48:
      ACCEPT_TOKEN(anon_sym_LPAREN);
      END_STATE();
    case 49:
      ACCEPT_TOKEN(anon_sym_POUNDPCDATA);
      END_STATE();
    case 50:
      ACCEPT_TOKEN(anon_sym_PIPE);
      END_STATE();
    case 51:
      ACCEPT_TOKEN(anon_sym_RPAREN);
      END_STATE();
    case 52:
      ACCEPT_TOKEN(anon_sym_STAR);
      END_STATE();
    case 53:
      ACCEPT_TOKEN(anon_sym_QMARK);
      END_STATE();
    case 54:
      ACCEPT_TOKEN(anon_sym_PLUS);
      END_STATE();
    case 55:
      ACCEPT_TOKEN(anon_sym_COMMA);
      END_STATE();
    case 56:
      ACCEPT_TOKEN(sym_TokenizedType);
      if (lookahead == 'R') ADVANCE(85);
      if (lookahead == ':' ||
          lookahead == 0xb7) ADVANCE(115);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(114);
      END_STATE();
    case 57:
      ACCEPT_TOKEN(sym_TokenizedType);
      if (lookahead == 'R') ADVANCE(88);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(115);
      END_STATE();
    case 58:
      ACCEPT_TOKEN(sym_TokenizedType);
      if (lookahead == 'S') ADVANCE(60);
      if (lookahead == ':' ||
          lookahead == 0xb7) ADVANCE(115);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(114);
      END_STATE();
    case 59:
      ACCEPT_TOKEN(sym_TokenizedType);
      if (lookahead == 'S') ADVANCE(61);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(115);
      END_STATE();
    case 60:
      ACCEPT_TOKEN(sym_TokenizedType);
      if (lookahead == ':' ||
          lookahead == 0xb7) ADVANCE(115);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(114);
      END_STATE();
    case 61:
      ACCEPT_TOKEN(sym_TokenizedType);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(115);
      END_STATE();
    case 62:
      ACCEPT_TOKEN(anon_sym_POUNDREQUIRED);
      END_STATE();
    case 63:
      ACCEPT_TOKEN(anon_sym_POUNDIMPLIED);
      END_STATE();
    case 64:
      ACCEPT_TOKEN(anon_sym_POUNDFIXED);
      END_STATE();
    case 65:
      ACCEPT_TOKEN(anon_sym_PERCENT);
      END_STATE();
    case 66:
      ACCEPT_TOKEN(anon_sym_DQUOTE);
      END_STATE();
    case 67:
      ACCEPT_TOKEN(aux_sym_EntityValue_token1);
      END_STATE();
    case 68:
      ACCEPT_TOKEN(aux_sym_EntityValue_token1);
      if (lookahead == '.') ADVANCE(116);
      if (('0' <= lookahead && lookahead <= '9')) ADVANCE(117);
      if (('A' <= lookahead && lookahead <= 'F') ||
          ('a' <= lookahead && lookahead <= 'f')) ADVANCE(118);
      if (lookahead == '-' ||
          lookahead == ':' ||
          ('G' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('g' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(119);
      END_STATE();
    case 69:
      ACCEPT_TOKEN(aux_sym_EntityValue_token1);
      if (lookahead == 'D') ADVANCE(56);
      if (lookahead == ':' ||
          lookahead == 0xb7) ADVANCE(115);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(114);
      END_STATE();
    case 70:
      ACCEPT_TOKEN(aux_sym_EntityValue_token1);
      if (lookahead == 'F') ADVANCE(23);
      if (lookahead == 'I') ADVANCE(27);
      if (lookahead == 'P') ADVANCE(13);
      if (lookahead == 'R') ADVANCE(18);
      END_STATE();
    case 71:
      ACCEPT_TOKEN(aux_sym_EntityValue_token1);
      if (lookahead == 'M') ADVANCE(108);
      if (lookahead == ':' ||
          lookahead == 0xb7) ADVANCE(115);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(114);
      END_STATE();
    case 72:
      ACCEPT_TOKEN(aux_sym_EntityValue_token1);
      if (lookahead == 'N') ADVANCE(107);
      if (lookahead == ':' ||
          lookahead == 0xb7) ADVANCE(115);
      if (('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'F') ||
          ('a' <= lookahead && lookahead <= 'f')) ADVANCE(113);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('G' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('g' <= lookahead && lookahead <= 'z')) ADVANCE(114);
      END_STATE();
    case 73:
      ACCEPT_TOKEN(aux_sym_EntityValue_token1);
      if (lookahead == ']') ADVANCE(8);
      END_STATE();
    case 74:
      ACCEPT_TOKEN(aux_sym_EntityValue_token1);
      if (lookahead == ':' ||
          lookahead == 0xb7) ADVANCE(115);
      if (('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'F') ||
          ('a' <= lookahead && lookahead <= 'f')) ADVANCE(113);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('G' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('g' <= lookahead && lookahead <= 'z')) ADVANCE(114);
      END_STATE();
    case 75:
      ACCEPT_TOKEN(aux_sym_EntityValue_token1);
      if (lookahead == ':' ||
          lookahead == 0xb7) ADVANCE(115);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(114);
      END_STATE();
    case 76:
      ACCEPT_TOKEN(aux_sym_EntityValue_token1);
      if (lookahead == '\t' ||
          lookahead == '\n' ||
          lookahead == '\r' ||
          lookahead == ' ') ADVANCE(83);
      END_STATE();
    case 77:
      ACCEPT_TOKEN(aux_sym_EntityValue_token1);
      if (('0' <= lookahead && lookahead <= '9')) ADVANCE(117);
      if (('A' <= lookahead && lookahead <= 'F') ||
          ('a' <= lookahead && lookahead <= 'f')) ADVANCE(118);
      if (lookahead == '-' ||
          lookahead == '.' ||
          lookahead == ':' ||
          ('G' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('g' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(119);
      END_STATE();
    case 78:
      ACCEPT_TOKEN(aux_sym_EntityValue_token1);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(119);
      END_STATE();
    case 79:
      ACCEPT_TOKEN(aux_sym_EntityValue_token1);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(115);
      END_STATE();
    case 80:
      ACCEPT_TOKEN(anon_sym_SQUOTE);
      END_STATE();
    case 81:
      ACCEPT_TOKEN(aux_sym_EntityValue_token2);
      END_STATE();
    case 82:
      ACCEPT_TOKEN(anon_sym_SEMI);
      END_STATE();
    case 83:
      ACCEPT_TOKEN(sym__S);
      if (lookahead == '\t' ||
          lookahead == '\n' ||
          lookahead == '\r' ||
          lookahead == ' ') ADVANCE(83);
      END_STATE();
    case 84:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'D') ADVANCE(57);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(115);
      END_STATE();
    case 85:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'E') ADVANCE(91);
      if (lookahead == ':' ||
          lookahead == 0xb7) ADVANCE(115);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(114);
      END_STATE();
    case 86:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'E') ADVANCE(100);
      if (lookahead == ':' ||
          lookahead == 0xb7) ADVANCE(115);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(114);
      END_STATE();
    case 87:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'E') ADVANCE(105);
      if (lookahead == ':' ||
          lookahead == 0xb7) ADVANCE(115);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(114);
      END_STATE();
    case 88:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'E') ADVANCE(92);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(115);
      END_STATE();
    case 89:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'E') ADVANCE(106);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(115);
      END_STATE();
    case 90:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'E') ADVANCE(102);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(115);
      END_STATE();
    case 91:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'F') ADVANCE(58);
      if (lookahead == ':' ||
          lookahead == 0xb7) ADVANCE(115);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(114);
      END_STATE();
    case 92:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'F') ADVANCE(59);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(115);
      END_STATE();
    case 93:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'I') ADVANCE(109);
      if (lookahead == ':' ||
          lookahead == 0xb7) ADVANCE(115);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(114);
      END_STATE();
    case 94:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'I') ADVANCE(87);
      if (lookahead == 'Y') ADVANCE(60);
      if (lookahead == ':' ||
          lookahead == 0xb7) ADVANCE(115);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(114);
      END_STATE();
    case 95:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'I') ADVANCE(112);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(115);
      END_STATE();
    case 96:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'I') ADVANCE(89);
      if (lookahead == 'Y') ADVANCE(61);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(115);
      END_STATE();
    case 97:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'K') ADVANCE(86);
      if (lookahead == ':' ||
          lookahead == 0xb7) ADVANCE(115);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(114);
      END_STATE();
    case 98:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'K') ADVANCE(90);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(115);
      END_STATE();
    case 99:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'M') ADVANCE(111);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(115);
      END_STATE();
    case 100:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'N') ADVANCE(58);
      if (lookahead == ':' ||
          lookahead == 0xb7) ADVANCE(115);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(114);
      END_STATE();
    case 101:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'N') ADVANCE(110);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(115);
      END_STATE();
    case 102:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'N') ADVANCE(59);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(115);
      END_STATE();
    case 103:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'O') ADVANCE(97);
      if (lookahead == ':' ||
          lookahead == 0xb7) ADVANCE(115);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(114);
      END_STATE();
    case 104:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'O') ADVANCE(98);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(115);
      END_STATE();
    case 105:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'S') ADVANCE(60);
      if (lookahead == ':' ||
          lookahead == 0xb7) ADVANCE(115);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(114);
      END_STATE();
    case 106:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'S') ADVANCE(61);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(115);
      END_STATE();
    case 107:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'T') ADVANCE(93);
      if (lookahead == ':' ||
          lookahead == 0xb7) ADVANCE(115);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(114);
      END_STATE();
    case 108:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'T') ADVANCE(103);
      if (lookahead == ':' ||
          lookahead == 0xb7) ADVANCE(115);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(114);
      END_STATE();
    case 109:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'T') ADVANCE(94);
      if (lookahead == ':' ||
          lookahead == 0xb7) ADVANCE(115);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(114);
      END_STATE();
    case 110:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'T') ADVANCE(95);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(115);
      END_STATE();
    case 111:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'T') ADVANCE(104);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(115);
      END_STATE();
    case 112:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == 'T') ADVANCE(96);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(115);
      END_STATE();
    case 113:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == ':' ||
          lookahead == 0xb7) ADVANCE(115);
      if (('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'F') ||
          ('a' <= lookahead && lookahead <= 'f')) ADVANCE(113);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('G' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('g' <= lookahead && lookahead <= 'z')) ADVANCE(114);
      END_STATE();
    case 114:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == ':' ||
          lookahead == 0xb7) ADVANCE(115);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(114);
      END_STATE();
    case 115:
      ACCEPT_TOKEN(sym_Name);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(115);
      END_STATE();
    case 116:
      ACCEPT_TOKEN(sym_Nmtoken);
      if (('0' <= lookahead && lookahead <= '9')) ADVANCE(116);
      if (lookahead == '-' ||
          lookahead == '.' ||
          lookahead == ':' ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(119);
      END_STATE();
    case 117:
      ACCEPT_TOKEN(sym_Nmtoken);
      if (('0' <= lookahead && lookahead <= '9')) ADVANCE(117);
      if (('A' <= lookahead && lookahead <= 'F') ||
          ('a' <= lookahead && lookahead <= 'f')) ADVANCE(118);
      if (lookahead == '-' ||
          lookahead == '.' ||
          lookahead == ':' ||
          ('G' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('g' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(119);
      END_STATE();
    case 118:
      ACCEPT_TOKEN(sym_Nmtoken);
      if (('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'F') ||
          ('a' <= lookahead && lookahead <= 'f')) ADVANCE(118);
      if (lookahead == '-' ||
          lookahead == '.' ||
          lookahead == ':' ||
          ('G' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('g' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(119);
      END_STATE();
    case 119:
      ACCEPT_TOKEN(sym_Nmtoken);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= ':') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z') ||
          lookahead == 0xb7) ADVANCE(119);
      END_STATE();
    case 120:
      ACCEPT_TOKEN(anon_sym_AMP);
      if (lookahead == '#') ADVANCE(121);
      END_STATE();
    case 121:
      ACCEPT_TOKEN(anon_sym_AMP_POUND);
      if (lookahead == 'x') ADVANCE(123);
      END_STATE();
    case 122:
      ACCEPT_TOKEN(aux_sym_CharRef_token1);
      if (('0' <= lookahead && lookahead <= '9')) ADVANCE(122);
      END_STATE();
    case 123:
      ACCEPT_TOKEN(anon_sym_AMP_POUNDx);
      END_STATE();
    case 124:
      ACCEPT_TOKEN(aux_sym_CharRef_token2);
      if (('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'F') ||
          ('a' <= lookahead && lookahead <= 'f')) ADVANCE(124);
      END_STATE();
    case 125:
      ACCEPT_TOKEN(aux_sym_AttValue_token1);
      END_STATE();
    case 126:
      ACCEPT_TOKEN(aux_sym_AttValue_token2);
      END_STATE();
    case 127:
      ACCEPT_TOKEN(aux_sym_SystemLiteral_token1);
      if (lookahead != 0 &&
          lookahead != '"') ADVANCE(127);
      END_STATE();
    case 128:
      ACCEPT_TOKEN(aux_sym_SystemLiteral_token2);
      if (lookahead != 0 &&
          lookahead != '\'') ADVANCE(128);
      END_STATE();
    case 129:
      ACCEPT_TOKEN(aux_sym_PubidLiteral_token1);
      if (set_contains(aux_sym_PubidLiteral_token1_character_set_1, 9, lookahead)) ADVANCE(129);
      END_STATE();
    case 130:
      ACCEPT_TOKEN(aux_sym_PubidLiteral_token2);
      if (set_contains(aux_sym_PubidLiteral_token2_character_set_1, 9, lookahead)) ADVANCE(130);
      END_STATE();
    case 131:
      ACCEPT_TOKEN(sym_VersionNum);
      if (('0' <= lookahead && lookahead <= '9')) ADVANCE(131);
      END_STATE();
    case 132:
      ACCEPT_TOKEN(sym_EncName);
      if (lookahead == '-' ||
          lookahead == '.' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(132);
      END_STATE();
    case 133:
      ACCEPT_TOKEN(anon_sym_EQ);
      END_STATE();
    default:
      return false;
  }
}

static bool ts_lex_keywords(TSLexer *lexer, TSStateId state) {
  START_LEXER();
  eof = lexer->eof(lexer);
  switch (state) {
    case 0:
      ADVANCE_MAP(
        'A', 1,
        'C', 2,
        'E', 3,
        'I', 4,
        'N', 5,
        'P', 6,
        'S', 7,
        'e', 8,
        'v', 9,
        'x', 10,
      );
      END_STATE();
    case 1:
      if (lookahead == 'N') ADVANCE(11);
      if (lookahead == 'T') ADVANCE(12);
      END_STATE();
    case 2:
      if (lookahead == 'D') ADVANCE(13);
      END_STATE();
    case 3:
      if (lookahead == 'L') ADVANCE(14);
      if (lookahead == 'M') ADVANCE(15);
      if (lookahead == 'N') ADVANCE(16);
      END_STATE();
    case 4:
      if (lookahead == 'G') ADVANCE(17);
      if (lookahead == 'N') ADVANCE(18);
      END_STATE();
    case 5:
      if (lookahead == 'D') ADVANCE(19);
      if (lookahead == 'O') ADVANCE(20);
      END_STATE();
    case 6:
      if (lookahead == 'U') ADVANCE(21);
      END_STATE();
    case 7:
      if (lookahead == 'Y') ADVANCE(22);
      END_STATE();
    case 8:
      if (lookahead == 'n') ADVANCE(23);
      END_STATE();
    case 9:
      if (lookahead == 'e') ADVANCE(24);
      END_STATE();
    case 10:
      if (lookahead == 'm') ADVANCE(25);
      END_STATE();
    case 11:
      if (lookahead == 'Y') ADVANCE(26);
      END_STATE();
    case 12:
      if (lookahead == 'T') ADVANCE(27);
      END_STATE();
    case 13:
      if (lookahead == 'A') ADVANCE(28);
      END_STATE();
    case 14:
      if (lookahead == 'E') ADVANCE(29);
      END_STATE();
    case 15:
      if (lookahead == 'P') ADVANCE(30);
      END_STATE();
    case 16:
      if (lookahead == 'T') ADVANCE(31);
      END_STATE();
    case 17:
      if (lookahead == 'N') ADVANCE(32);
      END_STATE();
    case 18:
      if (lookahead == 'C') ADVANCE(33);
      END_STATE();
    case 19:
      if (lookahead == 'A') ADVANCE(34);
      END_STATE();
    case 20:
      if (lookahead == 'T') ADVANCE(35);
      END_STATE();
    case 21:
      if (lookahead == 'B') ADVANCE(36);
      END_STATE();
    case 22:
      if (lookahead == 'S') ADVANCE(37);
      END_STATE();
    case 23:
      if (lookahead == 'c') ADVANCE(38);
      END_STATE();
    case 24:
      if (lookahead == 'r') ADVANCE(39);
      END_STATE();
    case 25:
      if (lookahead == 'l') ADVANCE(40);
      END_STATE();
    case 26:
      ACCEPT_TOKEN(anon_sym_ANY);
      END_STATE();
    case 27:
      if (lookahead == 'L') ADVANCE(41);
      END_STATE();
    case 28:
      if (lookahead == 'T') ADVANCE(42);
      END_STATE();
    case 29:
      if (lookahead == 'M') ADVANCE(43);
      END_STATE();
    case 30:
      if (lookahead == 'T') ADVANCE(44);
      END_STATE();
    case 31:
      if (lookahead == 'I') ADVANCE(45);
      END_STATE();
    case 32:
      if (lookahead == 'O') ADVANCE(46);
      END_STATE();
    case 33:
      if (lookahead == 'L') ADVANCE(47);
      END_STATE();
    case 34:
      if (lookahead == 'T') ADVANCE(48);
      END_STATE();
    case 35:
      if (lookahead == 'A') ADVANCE(49);
      END_STATE();
    case 36:
      if (lookahead == 'L') ADVANCE(50);
      END_STATE();
    case 37:
      if (lookahead == 'T') ADVANCE(51);
      END_STATE();
    case 38:
      if (lookahead == 'o') ADVANCE(52);
      END_STATE();
    case 39:
      if (lookahead == 's') ADVANCE(53);
      END_STATE();
    case 40:
      ACCEPT_TOKEN(anon_sym_xml);
      END_STATE();
    case 41:
      if (lookahead == 'I') ADVANCE(54);
      END_STATE();
    case 42:
      if (lookahead == 'A') ADVANCE(55);
      END_STATE();
    case 43:
      if (lookahead == 'E') ADVANCE(56);
      END_STATE();
    case 44:
      if (lookahead == 'Y') ADVANCE(57);
      END_STATE();
    case 45:
      if (lookahead == 'T') ADVANCE(58);
      END_STATE();
    case 46:
      if (lookahead == 'R') ADVANCE(59);
      END_STATE();
    case 47:
      if (lookahead == 'U') ADVANCE(60);
      END_STATE();
    case 48:
      if (lookahead == 'A') ADVANCE(61);
      END_STATE();
    case 49:
      if (lookahead == 'T') ADVANCE(62);
      END_STATE();
    case 50:
      if (lookahead == 'I') ADVANCE(63);
      END_STATE();
    case 51:
      if (lookahead == 'E') ADVANCE(64);
      END_STATE();
    case 52:
      if (lookahead == 'd') ADVANCE(65);
      END_STATE();
    case 53:
      if (lookahead == 'i') ADVANCE(66);
      END_STATE();
    case 54:
      if (lookahead == 'S') ADVANCE(67);
      END_STATE();
    case 55:
      ACCEPT_TOKEN(sym_StringType);
      END_STATE();
    case 56:
      if (lookahead == 'N') ADVANCE(68);
      END_STATE();
    case 57:
      ACCEPT_TOKEN(anon_sym_EMPTY);
      END_STATE();
    case 58:
      if (lookahead == 'Y') ADVANCE(69);
      END_STATE();
    case 59:
      if (lookahead == 'E') ADVANCE(70);
      END_STATE();
    case 60:
      if (lookahead == 'D') ADVANCE(71);
      END_STATE();
    case 61:
      ACCEPT_TOKEN(anon_sym_NDATA);
      END_STATE();
    case 62:
      if (lookahead == 'I') ADVANCE(72);
      END_STATE();
    case 63:
      if (lookahead == 'C') ADVANCE(73);
      END_STATE();
    case 64:
      if (lookahead == 'M') ADVANCE(74);
      END_STATE();
    case 65:
      if (lookahead == 'i') ADVANCE(75);
      END_STATE();
    case 66:
      if (lookahead == 'o') ADVANCE(76);
      END_STATE();
    case 67:
      if (lookahead == 'T') ADVANCE(77);
      END_STATE();
    case 68:
      if (lookahead == 'T') ADVANCE(78);
      END_STATE();
    case 69:
      ACCEPT_TOKEN(anon_sym_ENTITY);
      END_STATE();
    case 70:
      ACCEPT_TOKEN(anon_sym_IGNORE);
      END_STATE();
    case 71:
      if (lookahead == 'E') ADVANCE(79);
      END_STATE();
    case 72:
      if (lookahead == 'O') ADVANCE(80);
      END_STATE();
    case 73:
      ACCEPT_TOKEN(anon_sym_PUBLIC);
      END_STATE();
    case 74:
      ACCEPT_TOKEN(anon_sym_SYSTEM);
      END_STATE();
    case 75:
      if (lookahead == 'n') ADVANCE(81);
      END_STATE();
    case 76:
      if (lookahead == 'n') ADVANCE(82);
      END_STATE();
    case 77:
      ACCEPT_TOKEN(anon_sym_ATTLIST);
      END_STATE();
    case 78:
      ACCEPT_TOKEN(anon_sym_ELEMENT);
      END_STATE();
    case 79:
      ACCEPT_TOKEN(anon_sym_INCLUDE);
      END_STATE();
    case 80:
      if (lookahead == 'N') ADVANCE(83);
      END_STATE();
    case 81:
      if (lookahead == 'g') ADVANCE(84);
      END_STATE();
    case 82:
      ACCEPT_TOKEN(anon_sym_version);
      END_STATE();
    case 83:
      ACCEPT_TOKEN(anon_sym_NOTATION);
      END_STATE();
    case 84:
      ACCEPT_TOKEN(anon_sym_encoding);
      END_STATE();
    default:
      return false;
  }
}

static const TSLexMode ts_lex_modes[STATE_COUNT] = {
  [0] = {.lex_state = 0, .external_lex_state = 1},
  [1] = {.lex_state = 39, .external_lex_state = 2},
  [2] = {.lex_state = 39, .external_lex_state = 2},
  [3] = {.lex_state = 39, .external_lex_state = 2},
  [4] = {.lex_state = 39, .external_lex_state = 2},
  [5] = {.lex_state = 39, .external_lex_state = 2},
  [6] = {.lex_state = 39, .external_lex_state = 2},
  [7] = {.lex_state = 39, .external_lex_state = 2},
  [8] = {.lex_state = 39, .external_lex_state = 2},
  [9] = {.lex_state = 39, .external_lex_state = 2},
  [10] = {.lex_state = 39, .external_lex_state = 2},
  [11] = {.lex_state = 39, .external_lex_state = 2},
  [12] = {.lex_state = 4},
  [13] = {.lex_state = 39},
  [14] = {.lex_state = 4},
  [15] = {.lex_state = 2},
  [16] = {.lex_state = 39},
  [17] = {.lex_state = 39},
  [18] = {.lex_state = 2},
  [19] = {.lex_state = 4},
  [20] = {.lex_state = 2},
  [21] = {.lex_state = 5},
  [22] = {.lex_state = 39},
  [23] = {.lex_state = 6},
  [24] = {.lex_state = 3},
  [25] = {.lex_state = 6},
  [26] = {.lex_state = 39},
  [27] = {.lex_state = 39},
  [28] = {.lex_state = 39},
  [29] = {.lex_state = 3},
  [30] = {.lex_state = 6},
  [31] = {.lex_state = 3},
  [32] = {.lex_state = 39},
  [33] = {.lex_state = 39},
  [34] = {.lex_state = 39},
  [35] = {.lex_state = 39, .external_lex_state = 2},
  [36] = {.lex_state = 39, .external_lex_state = 2},
  [37] = {.lex_state = 39, .external_lex_state = 2},
  [38] = {.lex_state = 39, .external_lex_state = 2},
  [39] = {.lex_state = 39, .external_lex_state = 2},
  [40] = {.lex_state = 39},
  [41] = {.lex_state = 39, .external_lex_state = 2},
  [42] = {.lex_state = 39},
  [43] = {.lex_state = 39, .external_lex_state = 2},
  [44] = {.lex_state = 39},
  [45] = {.lex_state = 39},
  [46] = {.lex_state = 39, .external_lex_state = 2},
  [47] = {.lex_state = 39, .external_lex_state = 2},
  [48] = {.lex_state = 39, .external_lex_state = 2},
  [49] = {.lex_state = 39, .external_lex_state = 2},
  [50] = {.lex_state = 39, .external_lex_state = 2},
  [51] = {.lex_state = 39, .external_lex_state = 2},
  [52] = {.lex_state = 39, .external_lex_state = 2},
  [53] = {.lex_state = 39, .external_lex_state = 2},
  [54] = {.lex_state = 39, .external_lex_state = 2},
  [55] = {.lex_state = 39, .external_lex_state = 2},
  [56] = {.lex_state = 39},
  [57] = {.lex_state = 39},
  [58] = {.lex_state = 39, .external_lex_state = 2},
  [59] = {.lex_state = 39, .external_lex_state = 2},
  [60] = {.lex_state = 39, .external_lex_state = 2},
  [61] = {.lex_state = 39, .external_lex_state = 2},
  [62] = {.lex_state = 39},
  [63] = {.lex_state = 39},
  [64] = {.lex_state = 39},
  [65] = {.lex_state = 39},
  [66] = {.lex_state = 39},
  [67] = {.lex_state = 39},
  [68] = {.lex_state = 39},
  [69] = {.lex_state = 39},
  [70] = {.lex_state = 39},
  [71] = {.lex_state = 39},
  [72] = {.lex_state = 39},
  [73] = {.lex_state = 39, .external_lex_state = 2},
  [74] = {.lex_state = 4},
  [75] = {.lex_state = 4},
  [76] = {.lex_state = 39, .external_lex_state = 2},
  [77] = {.lex_state = 39},
  [78] = {.lex_state = 39},
  [79] = {.lex_state = 4},
  [80] = {.lex_state = 39},
  [81] = {.lex_state = 39},
  [82] = {.lex_state = 4},
  [83] = {.lex_state = 39},
  [84] = {.lex_state = 2},
  [85] = {.lex_state = 39},
  [86] = {.lex_state = 39, .external_lex_state = 2},
  [87] = {.lex_state = 2},
  [88] = {.lex_state = 2},
  [89] = {.lex_state = 39},
  [90] = {.lex_state = 2},
  [91] = {.lex_state = 39},
  [92] = {.lex_state = 39},
  [93] = {.lex_state = 39},
  [94] = {.lex_state = 39},
  [95] = {.lex_state = 39},
  [96] = {.lex_state = 3},
  [97] = {.lex_state = 6},
  [98] = {.lex_state = 6},
  [99] = {.lex_state = 39},
  [100] = {.lex_state = 39},
  [101] = {.lex_state = 39},
  [102] = {.lex_state = 3},
  [103] = {.lex_state = 6},
  [104] = {.lex_state = 39},
  [105] = {.lex_state = 39},
  [106] = {.lex_state = 3},
  [107] = {.lex_state = 39},
  [108] = {.lex_state = 39},
  [109] = {.lex_state = 39},
  [110] = {.lex_state = 39},
  [111] = {.lex_state = 39},
  [112] = {.lex_state = 39},
  [113] = {.lex_state = 39},
  [114] = {.lex_state = 39},
  [115] = {.lex_state = 39},
  [116] = {.lex_state = 39},
  [117] = {.lex_state = 39},
  [118] = {.lex_state = 39},
  [119] = {.lex_state = 39},
  [120] = {.lex_state = 39},
  [121] = {.lex_state = 39},
  [122] = {.lex_state = 39},
  [123] = {.lex_state = 39},
  [124] = {.lex_state = 39},
  [125] = {.lex_state = 39},
  [126] = {.lex_state = 39},
  [127] = {.lex_state = 39},
  [128] = {.lex_state = 39},
  [129] = {.lex_state = 39},
  [130] = {.lex_state = 39},
  [131] = {.lex_state = 39},
  [132] = {.lex_state = 39},
  [133] = {.lex_state = 39},
  [134] = {.lex_state = 39},
  [135] = {.lex_state = 39},
  [136] = {.lex_state = 39},
  [137] = {.lex_state = 39},
  [138] = {.lex_state = 39},
  [139] = {.lex_state = 39},
  [140] = {.lex_state = 39},
  [141] = {.lex_state = 39},
  [142] = {.lex_state = 39},
  [143] = {.lex_state = 39},
  [144] = {.lex_state = 39},
  [145] = {.lex_state = 39},
  [146] = {.lex_state = 39},
  [147] = {.lex_state = 39},
  [148] = {.lex_state = 39},
  [149] = {.lex_state = 5},
  [150] = {.lex_state = 39},
  [151] = {.lex_state = 39},
  [152] = {.lex_state = 39},
  [153] = {.lex_state = 39},
  [154] = {.lex_state = 39},
  [155] = {.lex_state = 39},
  [156] = {.lex_state = 39},
  [157] = {.lex_state = 39},
  [158] = {.lex_state = 39},
  [159] = {.lex_state = 39},
  [160] = {.lex_state = 39},
  [161] = {.lex_state = 39},
  [162] = {.lex_state = 39},
  [163] = {.lex_state = 39},
  [164] = {.lex_state = 39},
  [165] = {.lex_state = 39},
  [166] = {.lex_state = 39},
  [167] = {.lex_state = 39},
  [168] = {.lex_state = 39},
  [169] = {.lex_state = 39},
  [170] = {.lex_state = 39},
  [171] = {.lex_state = 39},
  [172] = {.lex_state = 39},
  [173] = {.lex_state = 39},
  [174] = {.lex_state = 39},
  [175] = {.lex_state = 39},
  [176] = {.lex_state = 39},
  [177] = {.lex_state = 39},
  [178] = {.lex_state = 39},
  [179] = {.lex_state = 39},
  [180] = {.lex_state = 39},
  [181] = {.lex_state = 5},
  [182] = {.lex_state = 39},
  [183] = {.lex_state = 5},
  [184] = {.lex_state = 39},
  [185] = {.lex_state = 39},
  [186] = {.lex_state = 5},
  [187] = {.lex_state = 39},
  [188] = {.lex_state = 39},
  [189] = {.lex_state = 39},
  [190] = {.lex_state = 39},
  [191] = {.lex_state = 39},
  [192] = {.lex_state = 39},
  [193] = {.lex_state = 39},
  [194] = {.lex_state = 39},
  [195] = {.lex_state = 39},
  [196] = {.lex_state = 39, .external_lex_state = 3},
  [197] = {.lex_state = 39},
  [198] = {.lex_state = 39},
  [199] = {.lex_state = 39},
  [200] = {.lex_state = 39},
  [201] = {.lex_state = 39},
  [202] = {.lex_state = 39},
  [203] = {.lex_state = 39},
  [204] = {.lex_state = 39},
  [205] = {.lex_state = 39},
  [206] = {.lex_state = 39},
  [207] = {.lex_state = 39},
  [208] = {.lex_state = 39},
  [209] = {.lex_state = 39},
  [210] = {.lex_state = 10},
  [211] = {.lex_state = 39},
  [212] = {.lex_state = 39},
  [213] = {.lex_state = 5},
  [214] = {.lex_state = 39},
  [215] = {.lex_state = 39},
  [216] = {.lex_state = 39},
  [217] = {.lex_state = 39},
  [218] = {.lex_state = 35},
  [219] = {.lex_state = 39},
  [220] = {.lex_state = 39},
  [221] = {.lex_state = 39},
  [222] = {.lex_state = 39},
  [223] = {.lex_state = 39},
  [224] = {.lex_state = 39},
  [225] = {.lex_state = 35},
  [226] = {.lex_state = 39},
  [227] = {.lex_state = 39},
  [228] = {.lex_state = 39},
  [229] = {.lex_state = 39},
  [230] = {.lex_state = 39},
  [231] = {.lex_state = 35},
  [232] = {.lex_state = 39},
  [233] = {.lex_state = 39},
  [234] = {.lex_state = 39},
  [235] = {.lex_state = 39},
  [236] = {.lex_state = 39},
  [237] = {.lex_state = 39},
  [238] = {.lex_state = 39},
  [239] = {.lex_state = 39},
  [240] = {.lex_state = 39},
  [241] = {.lex_state = 39},
  [242] = {.lex_state = 39},
  [243] = {.lex_state = 39},
  [244] = {.lex_state = 39},
  [245] = {.lex_state = 39},
  [246] = {.lex_state = 39},
  [247] = {.lex_state = 39},
  [248] = {.lex_state = 39},
  [249] = {.lex_state = 39},
  [250] = {.lex_state = 39},
  [251] = {.lex_state = 39},
  [252] = {.lex_state = 39},
  [253] = {.lex_state = 39},
  [254] = {.lex_state = 39},
  [255] = {.lex_state = 39},
  [256] = {.lex_state = 39},
  [257] = {.lex_state = 35},
  [258] = {.lex_state = 39},
  [259] = {.lex_state = 39},
  [260] = {.lex_state = 39},
  [261] = {.lex_state = 39},
  [262] = {.lex_state = 39},
  [263] = {.lex_state = 130},
  [264] = {.lex_state = 39},
  [265] = {.lex_state = 39},
  [266] = {.lex_state = 39},
  [267] = {.lex_state = 39},
  [268] = {.lex_state = 39},
  [269] = {.lex_state = 39},
  [270] = {.lex_state = 5},
  [271] = {.lex_state = 39},
  [272] = {.lex_state = 129},
  [273] = {.lex_state = 35},
  [274] = {.lex_state = 39},
  [275] = {.lex_state = 39},
  [276] = {.lex_state = 5},
  [277] = {.lex_state = 39},
  [278] = {.lex_state = 39},
  [279] = {.lex_state = 128},
  [280] = {.lex_state = 39},
  [281] = {.lex_state = 35},
  [282] = {.lex_state = 127},
  [283] = {.lex_state = 39},
  [284] = {.lex_state = 39},
  [285] = {.lex_state = 38},
  [286] = {.lex_state = 38},
  [287] = {.lex_state = 39},
  [288] = {.lex_state = 39},
  [289] = {.lex_state = 39},
  [290] = {.lex_state = 37},
  [291] = {.lex_state = 39},
  [292] = {.lex_state = 39},
  [293] = {.lex_state = 5},
  [294] = {.lex_state = 39},
  [295] = {.lex_state = 39},
  [296] = {.lex_state = 39},
  [297] = {.lex_state = 39},
  [298] = {.lex_state = 39},
  [299] = {.lex_state = 39},
  [300] = {.lex_state = 39},
  [301] = {.lex_state = 0, .external_lex_state = 4},
  [302] = {.lex_state = 0, .external_lex_state = 3},
  [303] = {.lex_state = 39},
  [304] = {.lex_state = 39},
  [305] = {.lex_state = 39},
  [306] = {.lex_state = 39},
  [307] = {.lex_state = 39},
  [308] = {.lex_state = 39},
  [309] = {.lex_state = 39},
  [310] = {.lex_state = 39},
  [311] = {.lex_state = 0},
  [312] = {.lex_state = 39},
  [313] = {.lex_state = 39},
  [314] = {.lex_state = 39},
  [315] = {.lex_state = 39},
  [316] = {.lex_state = 39},
  [317] = {.lex_state = 39},
  [318] = {.lex_state = 39},
  [319] = {.lex_state = 39},
  [320] = {.lex_state = 39},
  [321] = {.lex_state = 39},
  [322] = {.lex_state = 39},
  [323] = {.lex_state = 39},
  [324] = {.lex_state = 5},
  [325] = {.lex_state = 37},
  [326] = {.lex_state = 39},
  [327] = {.lex_state = 39},
  [328] = {.lex_state = 5},
  [329] = {.lex_state = 37},
  [330] = {.lex_state = 39},
  [331] = {.lex_state = 39},
  [332] = {.lex_state = 5},
  [333] = {.lex_state = 37},
};

static const uint16_t ts_parse_table[LARGE_STATE_COUNT][SYMBOL_COUNT] = {
  [0] = {
    [ts_builtin_sym_end] = ACTIONS(1),
    [sym_Name] = ACTIONS(1),
    [anon_sym_LT_QMARK] = ACTIONS(1),
    [anon_sym_xml] = ACTIONS(1),
    [anon_sym_LT_BANG_LBRACK] = ACTIONS(1),
    [anon_sym_IGNORE] = ACTIONS(1),
    [anon_sym_INCLUDE] = ACTIONS(1),
    [anon_sym_LBRACK] = ACTIONS(1),
    [anon_sym_RBRACK_RBRACK_GT] = ACTIONS(1),
    [anon_sym_LT_BANG] = ACTIONS(1),
    [anon_sym_ELEMENT] = ACTIONS(1),
    [anon_sym_GT] = ACTIONS(1),
    [anon_sym_EMPTY] = ACTIONS(1),
    [anon_sym_ANY] = ACTIONS(1),
    [anon_sym_LPAREN] = ACTIONS(1),
    [anon_sym_POUNDPCDATA] = ACTIONS(1),
    [anon_sym_PIPE] = ACTIONS(1),
    [anon_sym_RPAREN] = ACTIONS(1),
    [anon_sym_STAR] = ACTIONS(1),
    [anon_sym_QMARK] = ACTIONS(1),
    [anon_sym_PLUS] = ACTIONS(1),
    [anon_sym_COMMA] = ACTIONS(1),
    [anon_sym_ATTLIST] = ACTIONS(1),
    [sym_StringType] = ACTIONS(1),
    [sym_TokenizedType] = ACTIONS(1),
    [anon_sym_NOTATION] = ACTIONS(1),
    [anon_sym_POUNDREQUIRED] = ACTIONS(1),
    [anon_sym_POUNDIMPLIED] = ACTIONS(1),
    [anon_sym_POUNDFIXED] = ACTIONS(1),
    [anon_sym_ENTITY] = ACTIONS(1),
    [anon_sym_PERCENT] = ACTIONS(1),
    [anon_sym_DQUOTE] = ACTIONS(1),
    [aux_sym_EntityValue_token1] = ACTIONS(1),
    [anon_sym_SQUOTE] = ACTIONS(1),
    [aux_sym_EntityValue_token2] = ACTIONS(1),
    [anon_sym_NDATA] = ACTIONS(1),
    [anon_sym_SEMI] = ACTIONS(1),
    [sym__S] = ACTIONS(1),
    [sym_Nmtoken] = ACTIONS(1),
    [anon_sym_AMP] = ACTIONS(1),
    [anon_sym_AMP_POUND] = ACTIONS(1),
    [aux_sym_CharRef_token1] = ACTIONS(1),
    [anon_sym_AMP_POUNDx] = ACTIONS(1),
    [aux_sym_CharRef_token2] = ACTIONS(1),
    [aux_sym_AttValue_token1] = ACTIONS(1),
    [aux_sym_AttValue_token2] = ACTIONS(1),
    [anon_sym_SYSTEM] = ACTIONS(1),
    [anon_sym_PUBLIC] = ACTIONS(1),
    [anon_sym_version] = ACTIONS(1),
    [sym_VersionNum] = ACTIONS(1),
    [anon_sym_encoding] = ACTIONS(1),
    [sym_EncName] = ACTIONS(1),
    [anon_sym_EQ] = ACTIONS(1),
    [sym_PITarget] = ACTIONS(1),
    [sym__pi_content] = ACTIONS(1),
    [sym_Comment] = ACTIONS(1),
  },
  [1] = {
    [sym_extSubset] = STATE(311),
    [sym_TextDecl] = STATE(11),
    [sym__extSubsetDecl] = STATE(9),
    [sym_conditionalSect] = STATE(9),
    [sym__markupdecl] = STATE(9),
    [sym__DeclSep] = STATE(9),
    [sym_elementdecl] = STATE(58),
    [sym_AttlistDecl] = STATE(58),
    [sym__EntityDecl] = STATE(58),
    [sym_GEDecl] = STATE(51),
    [sym_PEDecl] = STATE(51),
    [sym_NotationDecl] = STATE(58),
    [sym_PEReference] = STATE(9),
    [sym_PI] = STATE(58),
    [aux_sym_extSubset_repeat1] = STATE(9),
    [anon_sym_LT_QMARK] = ACTIONS(3),
    [anon_sym_LT_BANG_LBRACK] = ACTIONS(5),
    [anon_sym_LT_BANG] = ACTIONS(7),
    [anon_sym_PERCENT] = ACTIONS(9),
    [sym__S] = ACTIONS(11),
    [sym_Comment] = ACTIONS(13),
  },
};

static const uint16_t ts_small_parse_table[] = {
  [0] = 10,
    ACTIONS(17), 1,
      anon_sym_LT_QMARK,
    ACTIONS(20), 1,
      anon_sym_LT_BANG_LBRACK,
    ACTIONS(23), 1,
      anon_sym_LT_BANG,
    ACTIONS(26), 1,
      anon_sym_PERCENT,
    ACTIONS(29), 1,
      sym__S,
    ACTIONS(32), 1,
      sym_Comment,
    ACTIONS(15), 2,
      ts_builtin_sym_end,
      anon_sym_RBRACK_RBRACK_GT,
    STATE(51), 2,
      sym_GEDecl,
      sym_PEDecl,
    STATE(58), 5,
      sym_elementdecl,
      sym_AttlistDecl,
      sym__EntityDecl,
      sym_NotationDecl,
      sym_PI,
    STATE(2), 6,
      sym__extSubsetDecl,
      sym_conditionalSect,
      sym__markupdecl,
      sym__DeclSep,
      sym_PEReference,
      aux_sym_extSubset_repeat1,
  [42] = 10,
    ACTIONS(5), 1,
      anon_sym_LT_BANG_LBRACK,
    ACTIONS(7), 1,
      anon_sym_LT_BANG,
    ACTIONS(9), 1,
      anon_sym_PERCENT,
    ACTIONS(13), 1,
      sym_Comment,
    ACTIONS(35), 1,
      anon_sym_LT_QMARK,
    ACTIONS(37), 1,
      anon_sym_RBRACK_RBRACK_GT,
    ACTIONS(39), 1,
      sym__S,
    STATE(51), 2,
      sym_GEDecl,
      sym_PEDecl,
    STATE(58), 5,
      sym_elementdecl,
      sym_AttlistDecl,
      sym__EntityDecl,
      sym_NotationDecl,
      sym_PI,
    STATE(7), 6,
      sym__extSubsetDecl,
      sym_conditionalSect,
      sym__markupdecl,
      sym__DeclSep,
      sym_PEReference,
      aux_sym_extSubset_repeat1,
  [83] = 10,
    ACTIONS(5), 1,
      anon_sym_LT_BANG_LBRACK,
    ACTIONS(7), 1,
      anon_sym_LT_BANG,
    ACTIONS(9), 1,
      anon_sym_PERCENT,
    ACTIONS(13), 1,
      sym_Comment,
    ACTIONS(35), 1,
      anon_sym_LT_QMARK,
    ACTIONS(41), 1,
      anon_sym_RBRACK_RBRACK_GT,
    ACTIONS(43), 1,
      sym__S,
    STATE(51), 2,
      sym_GEDecl,
      sym_PEDecl,
    STATE(58), 5,
      sym_elementdecl,
      sym_AttlistDecl,
      sym__EntityDecl,
      sym_NotationDecl,
      sym_PI,
    STATE(2), 6,
      sym__extSubsetDecl,
      sym_conditionalSect,
      sym__markupdecl,
      sym__DeclSep,
      sym_PEReference,
      aux_sym_extSubset_repeat1,
  [124] = 10,
    ACTIONS(5), 1,
      anon_sym_LT_BANG_LBRACK,
    ACTIONS(7), 1,
      anon_sym_LT_BANG,
    ACTIONS(9), 1,
      anon_sym_PERCENT,
    ACTIONS(13), 1,
      sym_Comment,
    ACTIONS(35), 1,
      anon_sym_LT_QMARK,
    ACTIONS(45), 1,
      anon_sym_RBRACK_RBRACK_GT,
    ACTIONS(47), 1,
      sym__S,
    STATE(51), 2,
      sym_GEDecl,
      sym_PEDecl,
    STATE(58), 5,
      sym_elementdecl,
      sym_AttlistDecl,
      sym__EntityDecl,
      sym_NotationDecl,
      sym_PI,
    STATE(4), 6,
      sym__extSubsetDecl,
      sym_conditionalSect,
      sym__markupdecl,
      sym__DeclSep,
      sym_PEReference,
      aux_sym_extSubset_repeat1,
  [165] = 10,
    ACTIONS(5), 1,
      anon_sym_LT_BANG_LBRACK,
    ACTIONS(7), 1,
      anon_sym_LT_BANG,
    ACTIONS(9), 1,
      anon_sym_PERCENT,
    ACTIONS(13), 1,
      sym_Comment,
    ACTIONS(35), 1,
      anon_sym_LT_QMARK,
    ACTIONS(43), 1,
      sym__S,
    ACTIONS(49), 1,
      ts_builtin_sym_end,
    STATE(51), 2,
      sym_GEDecl,
      sym_PEDecl,
    STATE(58), 5,
      sym_elementdecl,
      sym_AttlistDecl,
      sym__EntityDecl,
      sym_NotationDecl,
      sym_PI,
    STATE(2), 6,
      sym__extSubsetDecl,
      sym_conditionalSect,
      sym__markupdecl,
      sym__DeclSep,
      sym_PEReference,
      aux_sym_extSubset_repeat1,
  [206] = 10,
    ACTIONS(5), 1,
      anon_sym_LT_BANG_LBRACK,
    ACTIONS(7), 1,
      anon_sym_LT_BANG,
    ACTIONS(9), 1,
      anon_sym_PERCENT,
    ACTIONS(13), 1,
      sym_Comment,
    ACTIONS(35), 1,
      anon_sym_LT_QMARK,
    ACTIONS(43), 1,
      sym__S,
    ACTIONS(51), 1,
      anon_sym_RBRACK_RBRACK_GT,
    STATE(51), 2,
      sym_GEDecl,
      sym_PEDecl,
    STATE(58), 5,
      sym_elementdecl,
      sym_AttlistDecl,
      sym__EntityDecl,
      sym_NotationDecl,
      sym_PI,
    STATE(2), 6,
      sym__extSubsetDecl,
      sym_conditionalSect,
      sym__markupdecl,
      sym__DeclSep,
      sym_PEReference,
      aux_sym_extSubset_repeat1,
  [247] = 10,
    ACTIONS(5), 1,
      anon_sym_LT_BANG_LBRACK,
    ACTIONS(7), 1,
      anon_sym_LT_BANG,
    ACTIONS(9), 1,
      anon_sym_PERCENT,
    ACTIONS(13), 1,
      sym_Comment,
    ACTIONS(35), 1,
      anon_sym_LT_QMARK,
    ACTIONS(37), 1,
      anon_sym_RBRACK_RBRACK_GT,
    ACTIONS(43), 1,
      sym__S,
    STATE(51), 2,
      sym_GEDecl,
      sym_PEDecl,
    STATE(58), 5,
      sym_elementdecl,
      sym_AttlistDecl,
      sym__EntityDecl,
      sym_NotationDecl,
      sym_PI,
    STATE(2), 6,
      sym__extSubsetDecl,
      sym_conditionalSect,
      sym__markupdecl,
      sym__DeclSep,
      sym_PEReference,
      aux_sym_extSubset_repeat1,
  [288] = 10,
    ACTIONS(5), 1,
      anon_sym_LT_BANG_LBRACK,
    ACTIONS(7), 1,
      anon_sym_LT_BANG,
    ACTIONS(9), 1,
      anon_sym_PERCENT,
    ACTIONS(13), 1,
      sym_Comment,
    ACTIONS(35), 1,
      anon_sym_LT_QMARK,
    ACTIONS(43), 1,
      sym__S,
    ACTIONS(53), 1,
      ts_builtin_sym_end,
    STATE(51), 2,
      sym_GEDecl,
      sym_PEDecl,
    STATE(58), 5,
      sym_elementdecl,
      sym_AttlistDecl,
      sym__EntityDecl,
      sym_NotationDecl,
      sym_PI,
    STATE(2), 6,
      sym__extSubsetDecl,
      sym_conditionalSect,
      sym__markupdecl,
      sym__DeclSep,
      sym_PEReference,
      aux_sym_extSubset_repeat1,
  [329] = 10,
    ACTIONS(5), 1,
      anon_sym_LT_BANG_LBRACK,
    ACTIONS(7), 1,
      anon_sym_LT_BANG,
    ACTIONS(9), 1,
      anon_sym_PERCENT,
    ACTIONS(13), 1,
      sym_Comment,
    ACTIONS(35), 1,
      anon_sym_LT_QMARK,
    ACTIONS(41), 1,
      anon_sym_RBRACK_RBRACK_GT,
    ACTIONS(55), 1,
      sym__S,
    STATE(51), 2,
      sym_GEDecl,
      sym_PEDecl,
    STATE(58), 5,
      sym_elementdecl,
      sym_AttlistDecl,
      sym__EntityDecl,
      sym_NotationDecl,
      sym_PI,
    STATE(8), 6,
      sym__extSubsetDecl,
      sym_conditionalSect,
      sym__markupdecl,
      sym__DeclSep,
      sym_PEReference,
      aux_sym_extSubset_repeat1,
  [370] = 9,
    ACTIONS(5), 1,
      anon_sym_LT_BANG_LBRACK,
    ACTIONS(7), 1,
      anon_sym_LT_BANG,
    ACTIONS(9), 1,
      anon_sym_PERCENT,
    ACTIONS(13), 1,
      sym_Comment,
    ACTIONS(35), 1,
      anon_sym_LT_QMARK,
    ACTIONS(57), 1,
      sym__S,
    STATE(51), 2,
      sym_GEDecl,
      sym_PEDecl,
    STATE(58), 5,
      sym_elementdecl,
      sym_AttlistDecl,
      sym__EntityDecl,
      sym_NotationDecl,
      sym_PI,
    STATE(6), 6,
      sym__extSubsetDecl,
      sym_conditionalSect,
      sym__markupdecl,
      sym__DeclSep,
      sym_PEReference,
      aux_sym_extSubset_repeat1,
  [408] = 8,
    ACTIONS(59), 1,
      anon_sym_PERCENT,
    ACTIONS(62), 1,
      anon_sym_SQUOTE,
    ACTIONS(64), 1,
      aux_sym_EntityValue_token2,
    ACTIONS(67), 1,
      anon_sym_AMP,
    ACTIONS(70), 1,
      anon_sym_AMP_POUND,
    ACTIONS(73), 1,
      anon_sym_AMP_POUNDx,
    STATE(75), 2,
      sym_EntityRef,
      sym_CharRef,
    STATE(12), 3,
      sym_PEReference,
      sym__Reference,
      aux_sym_EntityValue_repeat2,
  [436] = 5,
    STATE(68), 1,
      aux_sym_Mixed_repeat1,
    STATE(135), 1,
      aux_sym_Mixed_repeat2,
    STATE(148), 1,
      sym_PEReference,
    ACTIONS(78), 3,
      anon_sym_STAR,
      anon_sym_QMARK,
      anon_sym_PLUS,
    ACTIONS(76), 5,
      anon_sym_PIPE,
      anon_sym_RPAREN,
      anon_sym_COMMA,
      anon_sym_PERCENT,
      sym__S,
  [458] = 8,
    ACTIONS(80), 1,
      anon_sym_PERCENT,
    ACTIONS(82), 1,
      anon_sym_SQUOTE,
    ACTIONS(84), 1,
      aux_sym_EntityValue_token2,
    ACTIONS(86), 1,
      anon_sym_AMP,
    ACTIONS(88), 1,
      anon_sym_AMP_POUND,
    ACTIONS(90), 1,
      anon_sym_AMP_POUNDx,
    STATE(75), 2,
      sym_EntityRef,
      sym_CharRef,
    STATE(19), 3,
      sym_PEReference,
      sym__Reference,
      aux_sym_EntityValue_repeat2,
  [486] = 8,
    ACTIONS(82), 1,
      anon_sym_DQUOTE,
    ACTIONS(92), 1,
      anon_sym_PERCENT,
    ACTIONS(94), 1,
      aux_sym_EntityValue_token1,
    ACTIONS(96), 1,
      anon_sym_AMP,
    ACTIONS(98), 1,
      anon_sym_AMP_POUND,
    ACTIONS(100), 1,
      anon_sym_AMP_POUNDx,
    STATE(84), 2,
      sym_EntityRef,
      sym_CharRef,
    STATE(18), 3,
      sym_PEReference,
      sym__Reference,
      aux_sym_EntityValue_repeat1,
  [514] = 5,
    STATE(63), 1,
      aux_sym_Mixed_repeat1,
    STATE(124), 1,
      aux_sym_Mixed_repeat2,
    STATE(148), 1,
      sym_PEReference,
    ACTIONS(78), 3,
      anon_sym_STAR,
      anon_sym_QMARK,
      anon_sym_PLUS,
    ACTIONS(76), 5,
      anon_sym_PIPE,
      anon_sym_RPAREN,
      anon_sym_COMMA,
      anon_sym_PERCENT,
      sym__S,
  [536] = 1,
    ACTIONS(102), 11,
      anon_sym_LBRACK,
      anon_sym_GT,
      anon_sym_PIPE,
      anon_sym_RPAREN,
      anon_sym_STAR,
      anon_sym_QMARK,
      anon_sym_PLUS,
      anon_sym_COMMA,
      anon_sym_PERCENT,
      sym__S,
      sym_Name,
  [550] = 8,
    ACTIONS(92), 1,
      anon_sym_PERCENT,
    ACTIONS(96), 1,
      anon_sym_AMP,
    ACTIONS(98), 1,
      anon_sym_AMP_POUND,
    ACTIONS(100), 1,
      anon_sym_AMP_POUNDx,
    ACTIONS(104), 1,
      anon_sym_DQUOTE,
    ACTIONS(106), 1,
      aux_sym_EntityValue_token1,
    STATE(84), 2,
      sym_EntityRef,
      sym_CharRef,
    STATE(20), 3,
      sym_PEReference,
      sym__Reference,
      aux_sym_EntityValue_repeat1,
  [578] = 8,
    ACTIONS(80), 1,
      anon_sym_PERCENT,
    ACTIONS(86), 1,
      anon_sym_AMP,
    ACTIONS(88), 1,
      anon_sym_AMP_POUND,
    ACTIONS(90), 1,
      anon_sym_AMP_POUNDx,
    ACTIONS(104), 1,
      anon_sym_SQUOTE,
    ACTIONS(108), 1,
      aux_sym_EntityValue_token2,
    STATE(75), 2,
      sym_EntityRef,
      sym_CharRef,
    STATE(12), 3,
      sym_PEReference,
      sym__Reference,
      aux_sym_EntityValue_repeat2,
  [606] = 8,
    ACTIONS(110), 1,
      anon_sym_PERCENT,
    ACTIONS(113), 1,
      anon_sym_DQUOTE,
    ACTIONS(115), 1,
      aux_sym_EntityValue_token1,
    ACTIONS(118), 1,
      anon_sym_AMP,
    ACTIONS(121), 1,
      anon_sym_AMP_POUND,
    ACTIONS(124), 1,
      anon_sym_AMP_POUNDx,
    STATE(84), 2,
      sym_EntityRef,
      sym_CharRef,
    STATE(20), 3,
      sym_PEReference,
      sym__Reference,
      aux_sym_EntityValue_repeat1,
  [634] = 7,
    ACTIONS(127), 1,
      anon_sym_LPAREN,
    ACTIONS(131), 1,
      anon_sym_NOTATION,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    STATE(200), 1,
      sym__AttType,
    ACTIONS(129), 2,
      sym_StringType,
      sym_TokenizedType,
    STATE(194), 2,
      sym__EnumeratedType,
      sym_PEReference,
    STATE(202), 2,
      sym_NotationType,
      sym_Enumeration,
  [659] = 1,
    ACTIONS(135), 9,
      anon_sym_GT,
      anon_sym_PIPE,
      anon_sym_RPAREN,
      anon_sym_STAR,
      anon_sym_QMARK,
      anon_sym_PLUS,
      anon_sym_COMMA,
      anon_sym_PERCENT,
      sym__S,
  [671] = 7,
    ACTIONS(137), 1,
      anon_sym_SQUOTE,
    ACTIONS(139), 1,
      anon_sym_AMP,
    ACTIONS(141), 1,
      anon_sym_AMP_POUND,
    ACTIONS(143), 1,
      anon_sym_AMP_POUNDx,
    ACTIONS(145), 1,
      aux_sym_AttValue_token2,
    STATE(30), 2,
      sym__Reference,
      aux_sym_AttValue_repeat2,
    STATE(98), 2,
      sym_EntityRef,
      sym_CharRef,
  [695] = 7,
    ACTIONS(137), 1,
      anon_sym_DQUOTE,
    ACTIONS(147), 1,
      anon_sym_AMP,
    ACTIONS(149), 1,
      anon_sym_AMP_POUND,
    ACTIONS(151), 1,
      anon_sym_AMP_POUNDx,
    ACTIONS(153), 1,
      aux_sym_AttValue_token1,
    STATE(31), 2,
      sym__Reference,
      aux_sym_AttValue_repeat1,
    STATE(96), 2,
      sym_EntityRef,
      sym_CharRef,
  [719] = 7,
    ACTIONS(139), 1,
      anon_sym_AMP,
    ACTIONS(141), 1,
      anon_sym_AMP_POUND,
    ACTIONS(143), 1,
      anon_sym_AMP_POUNDx,
    ACTIONS(155), 1,
      anon_sym_SQUOTE,
    ACTIONS(157), 1,
      aux_sym_AttValue_token2,
    STATE(23), 2,
      sym__Reference,
      aux_sym_AttValue_repeat2,
    STATE(98), 2,
      sym_EntityRef,
      sym_CharRef,
  [743] = 1,
    ACTIONS(159), 9,
      anon_sym_GT,
      anon_sym_PIPE,
      anon_sym_RPAREN,
      anon_sym_STAR,
      anon_sym_QMARK,
      anon_sym_PLUS,
      anon_sym_COMMA,
      anon_sym_PERCENT,
      sym__S,
  [755] = 1,
    ACTIONS(161), 9,
      anon_sym_GT,
      anon_sym_PIPE,
      anon_sym_RPAREN,
      anon_sym_STAR,
      anon_sym_QMARK,
      anon_sym_PLUS,
      anon_sym_COMMA,
      anon_sym_PERCENT,
      sym__S,
  [767] = 1,
    ACTIONS(163), 9,
      anon_sym_GT,
      anon_sym_PIPE,
      anon_sym_RPAREN,
      anon_sym_STAR,
      anon_sym_QMARK,
      anon_sym_PLUS,
      anon_sym_COMMA,
      anon_sym_PERCENT,
      sym__S,
  [779] = 7,
    ACTIONS(147), 1,
      anon_sym_AMP,
    ACTIONS(149), 1,
      anon_sym_AMP_POUND,
    ACTIONS(151), 1,
      anon_sym_AMP_POUNDx,
    ACTIONS(155), 1,
      anon_sym_DQUOTE,
    ACTIONS(165), 1,
      aux_sym_AttValue_token1,
    STATE(24), 2,
      sym__Reference,
      aux_sym_AttValue_repeat1,
    STATE(96), 2,
      sym_EntityRef,
      sym_CharRef,
  [803] = 7,
    ACTIONS(167), 1,
      anon_sym_SQUOTE,
    ACTIONS(169), 1,
      anon_sym_AMP,
    ACTIONS(172), 1,
      anon_sym_AMP_POUND,
    ACTIONS(175), 1,
      anon_sym_AMP_POUNDx,
    ACTIONS(178), 1,
      aux_sym_AttValue_token2,
    STATE(30), 2,
      sym__Reference,
      aux_sym_AttValue_repeat2,
    STATE(98), 2,
      sym_EntityRef,
      sym_CharRef,
  [827] = 7,
    ACTIONS(181), 1,
      anon_sym_DQUOTE,
    ACTIONS(183), 1,
      anon_sym_AMP,
    ACTIONS(186), 1,
      anon_sym_AMP_POUND,
    ACTIONS(189), 1,
      anon_sym_AMP_POUNDx,
    ACTIONS(192), 1,
      aux_sym_AttValue_token1,
    STATE(31), 2,
      sym__Reference,
      aux_sym_AttValue_repeat1,
    STATE(96), 2,
      sym_EntityRef,
      sym_CharRef,
  [851] = 6,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(197), 1,
      anon_sym_LPAREN,
    STATE(110), 1,
      sym__choice,
    STATE(224), 1,
      sym_contentspec,
    ACTIONS(195), 2,
      anon_sym_EMPTY,
      anon_sym_ANY,
    STATE(226), 3,
      sym_Mixed,
      sym_children,
      sym_PEReference,
  [873] = 7,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(201), 1,
      anon_sym_POUNDFIXED,
    ACTIONS(203), 1,
      anon_sym_DQUOTE,
    ACTIONS(205), 1,
      anon_sym_SQUOTE,
    STATE(240), 1,
      sym_DefaultDecl,
    ACTIONS(199), 2,
      anon_sym_POUNDREQUIRED,
      anon_sym_POUNDIMPLIED,
    STATE(236), 2,
      sym_PEReference,
      sym_AttValue,
  [897] = 1,
    ACTIONS(207), 9,
      anon_sym_GT,
      anon_sym_PIPE,
      anon_sym_RPAREN,
      anon_sym_STAR,
      anon_sym_QMARK,
      anon_sym_PLUS,
      anon_sym_COMMA,
      anon_sym_PERCENT,
      sym__S,
  [909] = 2,
    ACTIONS(211), 1,
      anon_sym_LT_BANG,
    ACTIONS(209), 7,
      sym_Comment,
      ts_builtin_sym_end,
      anon_sym_LT_QMARK,
      anon_sym_LT_BANG_LBRACK,
      anon_sym_RBRACK_RBRACK_GT,
      anon_sym_PERCENT,
      sym__S,
  [922] = 2,
    ACTIONS(215), 1,
      anon_sym_LT_BANG,
    ACTIONS(213), 7,
      sym_Comment,
      ts_builtin_sym_end,
      anon_sym_LT_QMARK,
      anon_sym_LT_BANG_LBRACK,
      anon_sym_RBRACK_RBRACK_GT,
      anon_sym_PERCENT,
      sym__S,
  [935] = 2,
    ACTIONS(219), 1,
      anon_sym_LT_BANG,
    ACTIONS(217), 7,
      sym_Comment,
      ts_builtin_sym_end,
      anon_sym_LT_QMARK,
      anon_sym_LT_BANG_LBRACK,
      anon_sym_RBRACK_RBRACK_GT,
      anon_sym_PERCENT,
      sym__S,
  [948] = 2,
    ACTIONS(223), 1,
      anon_sym_LT_BANG,
    ACTIONS(221), 7,
      sym_Comment,
      ts_builtin_sym_end,
      anon_sym_LT_QMARK,
      anon_sym_LT_BANG_LBRACK,
      anon_sym_RBRACK_RBRACK_GT,
      anon_sym_PERCENT,
      sym__S,
  [961] = 2,
    ACTIONS(227), 1,
      anon_sym_LT_BANG,
    ACTIONS(225), 7,
      sym_Comment,
      ts_builtin_sym_end,
      anon_sym_LT_QMARK,
      anon_sym_LT_BANG_LBRACK,
      anon_sym_RBRACK_RBRACK_GT,
      anon_sym_PERCENT,
      sym__S,
  [974] = 6,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(231), 1,
      anon_sym_RPAREN,
    ACTIONS(233), 1,
      sym__S,
    STATE(77), 1,
      aux_sym__choice_repeat1,
    ACTIONS(229), 2,
      anon_sym_PIPE,
      anon_sym_COMMA,
    STATE(111), 2,
      sym_PEReference,
      aux_sym__choice_repeat2,
  [995] = 2,
    ACTIONS(237), 1,
      anon_sym_LT_BANG,
    ACTIONS(235), 7,
      sym_Comment,
      ts_builtin_sym_end,
      anon_sym_LT_QMARK,
      anon_sym_LT_BANG_LBRACK,
      anon_sym_RBRACK_RBRACK_GT,
      anon_sym_PERCENT,
      sym__S,
  [1008] = 6,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(239), 1,
      anon_sym_RPAREN,
    ACTIONS(241), 1,
      sym__S,
    STATE(77), 1,
      aux_sym__choice_repeat1,
    ACTIONS(229), 2,
      anon_sym_PIPE,
      anon_sym_COMMA,
    STATE(105), 2,
      sym_PEReference,
      aux_sym__choice_repeat2,
  [1029] = 2,
    ACTIONS(245), 1,
      anon_sym_LT_BANG,
    ACTIONS(243), 7,
      sym_Comment,
      ts_builtin_sym_end,
      anon_sym_LT_QMARK,
      anon_sym_LT_BANG_LBRACK,
      anon_sym_RBRACK_RBRACK_GT,
      anon_sym_PERCENT,
      sym__S,
  [1042] = 6,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(247), 1,
      anon_sym_RPAREN,
    ACTIONS(249), 1,
      sym__S,
    STATE(42), 1,
      aux_sym__choice_repeat1,
    ACTIONS(229), 2,
      anon_sym_PIPE,
      anon_sym_COMMA,
    STATE(108), 2,
      sym_PEReference,
      aux_sym__choice_repeat2,
  [1063] = 2,
    ACTIONS(78), 3,
      anon_sym_STAR,
      anon_sym_QMARK,
      anon_sym_PLUS,
    ACTIONS(76), 5,
      anon_sym_PIPE,
      anon_sym_RPAREN,
      anon_sym_COMMA,
      anon_sym_PERCENT,
      sym__S,
  [1076] = 2,
    ACTIONS(251), 1,
      anon_sym_LT_BANG,
    ACTIONS(102), 7,
      sym_Comment,
      ts_builtin_sym_end,
      anon_sym_LT_QMARK,
      anon_sym_LT_BANG_LBRACK,
      anon_sym_RBRACK_RBRACK_GT,
      anon_sym_PERCENT,
      sym__S,
  [1089] = 2,
    ACTIONS(255), 1,
      anon_sym_LT_BANG,
    ACTIONS(253), 7,
      sym_Comment,
      ts_builtin_sym_end,
      anon_sym_LT_QMARK,
      anon_sym_LT_BANG_LBRACK,
      anon_sym_RBRACK_RBRACK_GT,
      anon_sym_PERCENT,
      sym__S,
  [1102] = 2,
    ACTIONS(259), 1,
      anon_sym_LT_BANG,
    ACTIONS(257), 7,
      sym_Comment,
      ts_builtin_sym_end,
      anon_sym_LT_QMARK,
      anon_sym_LT_BANG_LBRACK,
      anon_sym_RBRACK_RBRACK_GT,
      anon_sym_PERCENT,
      sym__S,
  [1115] = 2,
    ACTIONS(263), 1,
      anon_sym_LT_BANG,
    ACTIONS(261), 7,
      sym_Comment,
      ts_builtin_sym_end,
      anon_sym_LT_QMARK,
      anon_sym_LT_BANG_LBRACK,
      anon_sym_RBRACK_RBRACK_GT,
      anon_sym_PERCENT,
      sym__S,
  [1128] = 2,
    ACTIONS(267), 1,
      anon_sym_LT_BANG,
    ACTIONS(265), 7,
      sym_Comment,
      ts_builtin_sym_end,
      anon_sym_LT_QMARK,
      anon_sym_LT_BANG_LBRACK,
      anon_sym_RBRACK_RBRACK_GT,
      anon_sym_PERCENT,
      sym__S,
  [1141] = 2,
    ACTIONS(271), 1,
      anon_sym_LT_BANG,
    ACTIONS(269), 7,
      sym_Comment,
      ts_builtin_sym_end,
      anon_sym_LT_QMARK,
      anon_sym_LT_BANG_LBRACK,
      anon_sym_RBRACK_RBRACK_GT,
      anon_sym_PERCENT,
      sym__S,
  [1154] = 2,
    ACTIONS(275), 1,
      anon_sym_LT_BANG,
    ACTIONS(273), 7,
      sym_Comment,
      ts_builtin_sym_end,
      anon_sym_LT_QMARK,
      anon_sym_LT_BANG_LBRACK,
      anon_sym_RBRACK_RBRACK_GT,
      anon_sym_PERCENT,
      sym__S,
  [1167] = 2,
    ACTIONS(279), 1,
      anon_sym_LT_BANG,
    ACTIONS(277), 7,
      sym_Comment,
      ts_builtin_sym_end,
      anon_sym_LT_QMARK,
      anon_sym_LT_BANG_LBRACK,
      anon_sym_RBRACK_RBRACK_GT,
      anon_sym_PERCENT,
      sym__S,
  [1180] = 2,
    ACTIONS(283), 1,
      anon_sym_LT_BANG,
    ACTIONS(281), 7,
      sym_Comment,
      ts_builtin_sym_end,
      anon_sym_LT_QMARK,
      anon_sym_LT_BANG_LBRACK,
      anon_sym_RBRACK_RBRACK_GT,
      anon_sym_PERCENT,
      sym__S,
  [1193] = 2,
    ACTIONS(287), 1,
      anon_sym_LT_BANG,
    ACTIONS(285), 7,
      sym_Comment,
      ts_builtin_sym_end,
      anon_sym_LT_QMARK,
      anon_sym_LT_BANG_LBRACK,
      anon_sym_RBRACK_RBRACK_GT,
      anon_sym_PERCENT,
      sym__S,
  [1206] = 8,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(289), 1,
      sym_Name,
    ACTIONS(291), 1,
      anon_sym_LPAREN,
    ACTIONS(293), 1,
      anon_sym_POUNDPCDATA,
    ACTIONS(295), 1,
      sym__S,
    STATE(13), 1,
      sym_PEReference,
    STATE(44), 1,
      sym__cp,
    STATE(45), 1,
      sym__choice,
  [1231] = 6,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(239), 1,
      anon_sym_RPAREN,
    ACTIONS(241), 1,
      sym__S,
    STATE(40), 1,
      aux_sym__choice_repeat1,
    ACTIONS(229), 2,
      anon_sym_PIPE,
      anon_sym_COMMA,
    STATE(105), 2,
      sym_PEReference,
      aux_sym__choice_repeat2,
  [1252] = 2,
    ACTIONS(299), 1,
      anon_sym_LT_BANG,
    ACTIONS(297), 7,
      sym_Comment,
      ts_builtin_sym_end,
      anon_sym_LT_QMARK,
      anon_sym_LT_BANG_LBRACK,
      anon_sym_RBRACK_RBRACK_GT,
      anon_sym_PERCENT,
      sym__S,
  [1265] = 2,
    ACTIONS(303), 1,
      anon_sym_LT_BANG,
    ACTIONS(301), 7,
      sym_Comment,
      ts_builtin_sym_end,
      anon_sym_LT_QMARK,
      anon_sym_LT_BANG_LBRACK,
      anon_sym_RBRACK_RBRACK_GT,
      anon_sym_PERCENT,
      sym__S,
  [1278] = 2,
    ACTIONS(307), 1,
      anon_sym_LT_BANG,
    ACTIONS(305), 7,
      sym_Comment,
      ts_builtin_sym_end,
      anon_sym_LT_QMARK,
      anon_sym_LT_BANG_LBRACK,
      anon_sym_RBRACK_RBRACK_GT,
      anon_sym_PERCENT,
      sym__S,
  [1291] = 2,
    ACTIONS(311), 1,
      anon_sym_LT_BANG,
    ACTIONS(309), 7,
      sym_Comment,
      ts_builtin_sym_end,
      anon_sym_LT_QMARK,
      anon_sym_LT_BANG_LBRACK,
      anon_sym_RBRACK_RBRACK_GT,
      anon_sym_PERCENT,
      sym__S,
  [1304] = 6,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(289), 1,
      sym_Name,
    ACTIONS(291), 1,
      anon_sym_LPAREN,
    ACTIONS(313), 1,
      sym__S,
    STATE(44), 1,
      sym__cp,
    STATE(45), 2,
      sym__choice,
      sym_PEReference,
  [1324] = 7,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(315), 1,
      anon_sym_PIPE,
    ACTIONS(317), 1,
      anon_sym_RPAREN,
    ACTIONS(319), 1,
      sym__S,
    STATE(115), 1,
      aux_sym_Mixed_repeat1,
    STATE(125), 1,
      aux_sym_Mixed_repeat2,
    STATE(148), 1,
      sym_PEReference,
  [1346] = 6,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(289), 1,
      sym_Name,
    ACTIONS(291), 1,
      anon_sym_LPAREN,
    ACTIONS(321), 1,
      sym__S,
    STATE(114), 1,
      sym__cp,
    STATE(45), 2,
      sym__choice,
      sym_PEReference,
  [1366] = 7,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(315), 1,
      anon_sym_PIPE,
    ACTIONS(323), 1,
      anon_sym_RPAREN,
    ACTIONS(325), 1,
      sym__S,
    STATE(63), 1,
      aux_sym_Mixed_repeat1,
    STATE(124), 1,
      aux_sym_Mixed_repeat2,
    STATE(148), 1,
      sym_PEReference,
  [1388] = 6,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(289), 1,
      sym_Name,
    ACTIONS(291), 1,
      anon_sym_LPAREN,
    ACTIONS(327), 1,
      sym__S,
    STATE(95), 1,
      sym__cp,
    STATE(45), 2,
      sym__choice,
      sym_PEReference,
  [1408] = 7,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(315), 1,
      anon_sym_PIPE,
    ACTIONS(329), 1,
      anon_sym_RPAREN,
    ACTIONS(331), 1,
      sym__S,
    STATE(68), 1,
      aux_sym_Mixed_repeat1,
    STATE(135), 1,
      aux_sym_Mixed_repeat2,
    STATE(148), 1,
      sym_PEReference,
  [1430] = 7,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(315), 1,
      anon_sym_PIPE,
    ACTIONS(333), 1,
      anon_sym_RPAREN,
    ACTIONS(335), 1,
      sym__S,
    STATE(115), 1,
      aux_sym_Mixed_repeat1,
    STATE(118), 1,
      aux_sym_Mixed_repeat2,
    STATE(148), 1,
      sym_PEReference,
  [1452] = 7,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(289), 1,
      sym_Name,
    ACTIONS(291), 1,
      anon_sym_LPAREN,
    ACTIONS(337), 1,
      anon_sym_POUNDPCDATA,
    STATE(16), 1,
      sym_PEReference,
    STATE(45), 1,
      sym__choice,
    STATE(57), 1,
      sym__cp,
  [1474] = 6,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(339), 1,
      sym_Name,
    ACTIONS(341), 1,
      anon_sym_PIPE,
    ACTIONS(343), 1,
      sym__S,
    STATE(112), 1,
      aux_sym_NotationType_repeat1,
    STATE(192), 1,
      sym_PEReference,
  [1493] = 6,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(341), 1,
      anon_sym_PIPE,
    ACTIONS(343), 1,
      sym__S,
    ACTIONS(345), 1,
      sym_Name,
    STATE(70), 1,
      aux_sym_NotationType_repeat1,
    STATE(199), 1,
      sym_PEReference,
  [1512] = 5,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(289), 1,
      sym_Name,
    ACTIONS(291), 1,
      anon_sym_LPAREN,
    STATE(100), 1,
      sym__cp,
    STATE(45), 2,
      sym__choice,
      sym_PEReference,
  [1529] = 2,
    ACTIONS(349), 1,
      anon_sym_LT_BANG,
    ACTIONS(347), 5,
      sym_Comment,
      anon_sym_LT_QMARK,
      anon_sym_LT_BANG_LBRACK,
      anon_sym_PERCENT,
      sym__S,
  [1540] = 2,
    ACTIONS(251), 2,
      anon_sym_AMP,
      anon_sym_AMP_POUND,
    ACTIONS(102), 4,
      anon_sym_PERCENT,
      anon_sym_SQUOTE,
      aux_sym_EntityValue_token2,
      anon_sym_AMP_POUNDx,
  [1551] = 2,
    ACTIONS(353), 2,
      anon_sym_AMP,
      anon_sym_AMP_POUND,
    ACTIONS(351), 4,
      anon_sym_PERCENT,
      anon_sym_SQUOTE,
      aux_sym_EntityValue_token2,
      anon_sym_AMP_POUNDx,
  [1562] = 2,
    ACTIONS(357), 1,
      anon_sym_LT_BANG,
    ACTIONS(355), 5,
      sym_Comment,
      anon_sym_LT_QMARK,
      anon_sym_LT_BANG_LBRACK,
      anon_sym_PERCENT,
      sym__S,
  [1573] = 4,
    ACTIONS(364), 1,
      sym__S,
    STATE(77), 1,
      aux_sym__choice_repeat1,
    ACTIONS(359), 2,
      anon_sym_PIPE,
      anon_sym_COMMA,
    ACTIONS(362), 2,
      anon_sym_RPAREN,
      anon_sym_PERCENT,
  [1588] = 5,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(289), 1,
      sym_Name,
    ACTIONS(291), 1,
      anon_sym_LPAREN,
    STATE(114), 1,
      sym__cp,
    STATE(45), 2,
      sym__choice,
      sym_PEReference,
  [1605] = 2,
    ACTIONS(369), 2,
      anon_sym_AMP,
      anon_sym_AMP_POUND,
    ACTIONS(367), 4,
      anon_sym_PERCENT,
      anon_sym_SQUOTE,
      aux_sym_EntityValue_token2,
      anon_sym_AMP_POUNDx,
  [1616] = 6,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(341), 1,
      anon_sym_PIPE,
    ACTIONS(343), 1,
      sym__S,
    ACTIONS(345), 1,
      sym_Name,
    STATE(112), 1,
      aux_sym_NotationType_repeat1,
    STATE(199), 1,
      sym_PEReference,
  [1635] = 5,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(289), 1,
      sym_Name,
    ACTIONS(291), 1,
      anon_sym_LPAREN,
    STATE(57), 1,
      sym__cp,
    STATE(45), 2,
      sym__choice,
      sym_PEReference,
  [1652] = 2,
    ACTIONS(373), 2,
      anon_sym_AMP,
      anon_sym_AMP_POUND,
    ACTIONS(371), 4,
      anon_sym_PERCENT,
      anon_sym_SQUOTE,
      aux_sym_EntityValue_token2,
      anon_sym_AMP_POUNDx,
  [1663] = 5,
    ACTIONS(375), 1,
      anon_sym_DQUOTE,
    ACTIONS(377), 1,
      anon_sym_SQUOTE,
    ACTIONS(379), 1,
      anon_sym_SYSTEM,
    ACTIONS(381), 1,
      anon_sym_PUBLIC,
    STATE(230), 2,
      sym_EntityValue,
      sym_ExternalID,
  [1680] = 2,
    ACTIONS(353), 2,
      anon_sym_AMP,
      anon_sym_AMP_POUND,
    ACTIONS(351), 4,
      anon_sym_PERCENT,
      anon_sym_DQUOTE,
      aux_sym_EntityValue_token1,
      anon_sym_AMP_POUNDx,
  [1691] = 6,
    ACTIONS(375), 1,
      anon_sym_DQUOTE,
    ACTIONS(377), 1,
      anon_sym_SQUOTE,
    ACTIONS(379), 1,
      anon_sym_SYSTEM,
    ACTIONS(381), 1,
      anon_sym_PUBLIC,
    STATE(172), 1,
      sym_ExternalID,
    STATE(212), 1,
      sym_EntityValue,
  [1710] = 2,
    ACTIONS(385), 1,
      anon_sym_LT_BANG,
    ACTIONS(383), 5,
      sym_Comment,
      anon_sym_LT_QMARK,
      anon_sym_LT_BANG_LBRACK,
      anon_sym_PERCENT,
      sym__S,
  [1721] = 2,
    ACTIONS(369), 2,
      anon_sym_AMP,
      anon_sym_AMP_POUND,
    ACTIONS(367), 4,
      anon_sym_PERCENT,
      anon_sym_DQUOTE,
      aux_sym_EntityValue_token1,
      anon_sym_AMP_POUNDx,
  [1732] = 2,
    ACTIONS(251), 2,
      anon_sym_AMP,
      anon_sym_AMP_POUND,
    ACTIONS(102), 4,
      anon_sym_PERCENT,
      anon_sym_DQUOTE,
      aux_sym_EntityValue_token1,
      anon_sym_AMP_POUNDx,
  [1743] = 5,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(379), 1,
      anon_sym_SYSTEM,
    ACTIONS(387), 1,
      anon_sym_PUBLIC,
    STATE(259), 1,
      sym_PEReference,
    STATE(215), 2,
      sym_ExternalID,
      sym_PublicID,
  [1760] = 2,
    ACTIONS(373), 2,
      anon_sym_AMP,
      anon_sym_AMP_POUND,
    ACTIONS(371), 4,
      anon_sym_PERCENT,
      anon_sym_DQUOTE,
      aux_sym_EntityValue_token1,
      anon_sym_AMP_POUNDx,
  [1771] = 6,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(341), 1,
      anon_sym_PIPE,
    ACTIONS(343), 1,
      sym__S,
    ACTIONS(389), 1,
      sym_Name,
    STATE(80), 1,
      aux_sym_NotationType_repeat1,
    STATE(205), 1,
      sym_PEReference,
  [1790] = 4,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(393), 1,
      anon_sym_RPAREN,
    STATE(169), 1,
      sym_PEReference,
    ACTIONS(391), 2,
      anon_sym_PIPE,
      anon_sym_COMMA,
  [1804] = 4,
    ACTIONS(395), 1,
      anon_sym_RPAREN,
    ACTIONS(397), 1,
      anon_sym_PERCENT,
    ACTIONS(400), 1,
      sym__S,
    STATE(93), 2,
      sym_PEReference,
      aux_sym__choice_repeat2,
  [1818] = 5,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(323), 1,
      anon_sym_RPAREN,
    ACTIONS(403), 1,
      anon_sym_PIPE,
    STATE(124), 1,
      aux_sym_Mixed_repeat2,
    STATE(148), 1,
      sym_PEReference,
  [1834] = 1,
    ACTIONS(362), 5,
      anon_sym_PIPE,
      anon_sym_RPAREN,
      anon_sym_COMMA,
      anon_sym_PERCENT,
      sym__S,
  [1842] = 2,
    ACTIONS(353), 2,
      anon_sym_AMP,
      anon_sym_AMP_POUND,
    ACTIONS(351), 3,
      anon_sym_DQUOTE,
      anon_sym_AMP_POUNDx,
      aux_sym_AttValue_token1,
  [1852] = 2,
    ACTIONS(369), 2,
      anon_sym_AMP,
      anon_sym_AMP_POUND,
    ACTIONS(367), 3,
      anon_sym_SQUOTE,
      anon_sym_AMP_POUNDx,
      aux_sym_AttValue_token2,
  [1862] = 2,
    ACTIONS(353), 2,
      anon_sym_AMP,
      anon_sym_AMP_POUND,
    ACTIONS(351), 3,
      anon_sym_SQUOTE,
      anon_sym_AMP_POUNDx,
      aux_sym_AttValue_token2,
  [1872] = 1,
    ACTIONS(405), 5,
      anon_sym_PIPE,
      anon_sym_RPAREN,
      anon_sym_COMMA,
      anon_sym_PERCENT,
      sym__S,
  [1880] = 1,
    ACTIONS(407), 5,
      anon_sym_PIPE,
      anon_sym_RPAREN,
      anon_sym_COMMA,
      anon_sym_PERCENT,
      sym__S,
  [1888] = 4,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(239), 1,
      anon_sym_RPAREN,
    STATE(169), 1,
      sym_PEReference,
    ACTIONS(391), 2,
      anon_sym_PIPE,
      anon_sym_COMMA,
  [1902] = 2,
    ACTIONS(373), 2,
      anon_sym_AMP,
      anon_sym_AMP_POUND,
    ACTIONS(371), 3,
      anon_sym_DQUOTE,
      anon_sym_AMP_POUNDx,
      aux_sym_AttValue_token1,
  [1912] = 2,
    ACTIONS(373), 2,
      anon_sym_AMP,
      anon_sym_AMP_POUND,
    ACTIONS(371), 3,
      anon_sym_SQUOTE,
      anon_sym_AMP_POUNDx,
      aux_sym_AttValue_token2,
  [1922] = 5,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(403), 1,
      anon_sym_PIPE,
    ACTIONS(409), 1,
      anon_sym_RPAREN,
    STATE(139), 1,
      aux_sym_Mixed_repeat2,
    STATE(148), 1,
      sym_PEReference,
  [1938] = 4,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(231), 1,
      anon_sym_RPAREN,
    ACTIONS(411), 1,
      sym__S,
    STATE(93), 2,
      sym_PEReference,
      aux_sym__choice_repeat2,
  [1952] = 2,
    ACTIONS(369), 2,
      anon_sym_AMP,
      anon_sym_AMP_POUND,
    ACTIONS(367), 3,
      anon_sym_DQUOTE,
      anon_sym_AMP_POUNDx,
      aux_sym_AttValue_token1,
  [1962] = 4,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(415), 1,
      sym__S,
    STATE(184), 1,
      sym_PEReference,
    ACTIONS(413), 2,
      anon_sym_IGNORE,
      anon_sym_INCLUDE,
  [1976] = 4,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(239), 1,
      anon_sym_RPAREN,
    ACTIONS(417), 1,
      sym__S,
    STATE(93), 2,
      sym_PEReference,
      aux_sym__choice_repeat2,
  [1990] = 4,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(231), 1,
      anon_sym_RPAREN,
    STATE(169), 1,
      sym_PEReference,
    ACTIONS(391), 2,
      anon_sym_PIPE,
      anon_sym_COMMA,
  [2004] = 2,
    ACTIONS(419), 2,
      anon_sym_GT,
      sym__S,
    ACTIONS(421), 3,
      anon_sym_STAR,
      anon_sym_QMARK,
      anon_sym_PLUS,
  [2014] = 4,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(393), 1,
      anon_sym_RPAREN,
    ACTIONS(423), 1,
      sym__S,
    STATE(93), 2,
      sym_PEReference,
      aux_sym__choice_repeat2,
  [2028] = 4,
    ACTIONS(427), 1,
      anon_sym_PIPE,
    ACTIONS(430), 1,
      sym__S,
    STATE(112), 1,
      aux_sym_NotationType_repeat1,
    ACTIONS(425), 2,
      anon_sym_PERCENT,
      sym_Name,
  [2042] = 5,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(403), 1,
      anon_sym_PIPE,
    ACTIONS(433), 1,
      anon_sym_RPAREN,
    STATE(128), 1,
      aux_sym_Mixed_repeat2,
    STATE(148), 1,
      sym_PEReference,
  [2058] = 1,
    ACTIONS(435), 5,
      anon_sym_PIPE,
      anon_sym_RPAREN,
      anon_sym_COMMA,
      anon_sym_PERCENT,
      sym__S,
  [2066] = 4,
    ACTIONS(437), 1,
      anon_sym_PIPE,
    ACTIONS(442), 1,
      sym__S,
    STATE(115), 1,
      aux_sym_Mixed_repeat1,
    ACTIONS(440), 2,
      anon_sym_RPAREN,
      anon_sym_PERCENT,
  [2080] = 5,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(317), 1,
      anon_sym_RPAREN,
    ACTIONS(403), 1,
      anon_sym_PIPE,
    STATE(125), 1,
      aux_sym_Mixed_repeat2,
    STATE(148), 1,
      sym_PEReference,
  [2096] = 4,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(445), 1,
      sym_Name,
    ACTIONS(447), 1,
      anon_sym_GT,
    STATE(222), 1,
      sym_PEReference,
  [2109] = 4,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(317), 1,
      anon_sym_RPAREN,
    STATE(121), 1,
      aux_sym_Mixed_repeat2,
    STATE(148), 1,
      sym_PEReference,
  [2122] = 4,
    ACTIONS(449), 1,
      anon_sym_PIPE,
    ACTIONS(451), 1,
      anon_sym_RPAREN,
    ACTIONS(453), 1,
      sym__S,
    STATE(134), 1,
      aux_sym_Enumeration_repeat1,
  [2135] = 3,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    STATE(197), 1,
      sym_PEReference,
    ACTIONS(455), 2,
      anon_sym_IGNORE,
      anon_sym_INCLUDE,
  [2146] = 4,
    ACTIONS(457), 1,
      anon_sym_RPAREN,
    ACTIONS(459), 1,
      anon_sym_PERCENT,
    STATE(121), 1,
      aux_sym_Mixed_repeat2,
    STATE(148), 1,
      sym_PEReference,
  [2159] = 1,
    ACTIONS(462), 4,
      anon_sym_PIPE,
      anon_sym_PERCENT,
      sym__S,
      sym_Name,
  [2166] = 1,
    ACTIONS(464), 4,
      anon_sym_PIPE,
      anon_sym_RPAREN,
      anon_sym_PERCENT,
      sym__S,
  [2173] = 4,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(433), 1,
      anon_sym_RPAREN,
    STATE(121), 1,
      aux_sym_Mixed_repeat2,
    STATE(148), 1,
      sym_PEReference,
  [2186] = 4,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(409), 1,
      anon_sym_RPAREN,
    STATE(121), 1,
      aux_sym_Mixed_repeat2,
    STATE(148), 1,
      sym_PEReference,
  [2199] = 4,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(466), 1,
      sym_Name,
    ACTIONS(468), 1,
      sym__S,
    STATE(123), 1,
      sym_PEReference,
  [2212] = 1,
    ACTIONS(440), 4,
      anon_sym_PIPE,
      anon_sym_RPAREN,
      anon_sym_PERCENT,
      sym__S,
  [2219] = 4,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(470), 1,
      anon_sym_RPAREN,
    STATE(121), 1,
      aux_sym_Mixed_repeat2,
    STATE(148), 1,
      sym_PEReference,
  [2232] = 3,
    ACTIONS(472), 1,
      anon_sym_GT,
    ACTIONS(474), 1,
      sym__S,
    STATE(129), 2,
      sym_AttDef,
      aux_sym_AttlistDecl_repeat1,
  [2243] = 4,
    ACTIONS(477), 1,
      anon_sym_ELEMENT,
    ACTIONS(479), 1,
      anon_sym_ATTLIST,
    ACTIONS(481), 1,
      anon_sym_NOTATION,
    ACTIONS(483), 1,
      anon_sym_ENTITY,
  [2256] = 4,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(445), 1,
      sym_Name,
    ACTIONS(485), 1,
      anon_sym_GT,
    STATE(222), 1,
      sym_PEReference,
  [2269] = 4,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(487), 1,
      sym_Name,
    ACTIONS(489), 1,
      sym__S,
    STATE(127), 1,
      sym_PEReference,
  [2282] = 4,
    ACTIONS(449), 1,
      anon_sym_PIPE,
    ACTIONS(491), 1,
      anon_sym_RPAREN,
    ACTIONS(493), 1,
      sym__S,
    STATE(140), 1,
      aux_sym_Enumeration_repeat1,
  [2295] = 4,
    ACTIONS(449), 1,
      anon_sym_PIPE,
    ACTIONS(491), 1,
      anon_sym_RPAREN,
    ACTIONS(493), 1,
      sym__S,
    STATE(142), 1,
      aux_sym_Enumeration_repeat1,
  [2308] = 4,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(323), 1,
      anon_sym_RPAREN,
    STATE(121), 1,
      aux_sym_Mixed_repeat2,
    STATE(148), 1,
      sym_PEReference,
  [2321] = 4,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(495), 1,
      sym_Name,
    ACTIONS(497), 1,
      sym__S,
    STATE(91), 1,
      sym_PEReference,
  [2334] = 3,
    ACTIONS(447), 1,
      anon_sym_GT,
    ACTIONS(499), 1,
      sym__S,
    STATE(129), 2,
      sym_AttDef,
      aux_sym_AttlistDecl_repeat1,
  [2345] = 1,
    ACTIONS(501), 4,
      anon_sym_PIPE,
      anon_sym_RPAREN,
      anon_sym_PERCENT,
      sym__S,
  [2352] = 4,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(503), 1,
      anon_sym_RPAREN,
    STATE(121), 1,
      aux_sym_Mixed_repeat2,
    STATE(148), 1,
      sym_PEReference,
  [2365] = 4,
    ACTIONS(449), 1,
      anon_sym_PIPE,
    ACTIONS(505), 1,
      anon_sym_RPAREN,
    ACTIONS(507), 1,
      sym__S,
    STATE(142), 1,
      aux_sym_Enumeration_repeat1,
  [2378] = 3,
    ACTIONS(509), 1,
      anon_sym_GT,
    ACTIONS(511), 1,
      sym__S,
    STATE(137), 2,
      sym_AttDef,
      aux_sym_AttlistDecl_repeat1,
  [2389] = 4,
    ACTIONS(513), 1,
      anon_sym_PIPE,
    ACTIONS(516), 1,
      anon_sym_RPAREN,
    ACTIONS(518), 1,
      sym__S,
    STATE(142), 1,
      aux_sym_Enumeration_repeat1,
  [2402] = 2,
    ACTIONS(523), 1,
      sym__S,
    ACTIONS(521), 3,
      anon_sym_PIPE,
      anon_sym_PERCENT,
      sym_Name,
  [2411] = 1,
    ACTIONS(425), 4,
      anon_sym_PIPE,
      anon_sym_PERCENT,
      sym__S,
      sym_Name,
  [2418] = 2,
    ACTIONS(526), 1,
      sym__S,
    ACTIONS(425), 3,
      anon_sym_PIPE,
      anon_sym_PERCENT,
      sym_Name,
  [2427] = 2,
    ACTIONS(531), 1,
      anon_sym_STAR,
    ACTIONS(529), 2,
      anon_sym_GT,
      sym__S,
  [2435] = 3,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(533), 1,
      sym_Name,
    STATE(239), 1,
      sym_PEReference,
  [2445] = 2,
    ACTIONS(537), 1,
      sym__S,
    ACTIONS(535), 2,
      anon_sym_RPAREN,
      anon_sym_PERCENT,
  [2453] = 3,
    ACTIONS(539), 1,
      anon_sym_QMARK_GT,
    ACTIONS(541), 1,
      sym__S,
    STATE(213), 1,
      sym__EncodingDecl,
  [2463] = 3,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(543), 1,
      sym_Name,
    STATE(262), 1,
      sym_PEReference,
  [2473] = 1,
    ACTIONS(545), 3,
      anon_sym_PIPE,
      anon_sym_RPAREN,
      sym__S,
  [2479] = 3,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(547), 1,
      sym_Name,
    STATE(141), 1,
      sym_PEReference,
  [2489] = 3,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(549), 1,
      sym_Name,
    STATE(260), 1,
      sym_PEReference,
  [2499] = 3,
    ACTIONS(551), 1,
      sym_Name,
    ACTIONS(553), 1,
      anon_sym_PERCENT,
    STATE(253), 1,
      sym_PEReference,
  [2509] = 3,
    ACTIONS(555), 1,
      sym__S,
    ACTIONS(557), 1,
      anon_sym_EQ,
    STATE(234), 1,
      sym__Eq,
  [2519] = 1,
    ACTIONS(559), 3,
      anon_sym_PIPE,
      anon_sym_RPAREN,
      sym__S,
  [2525] = 3,
    ACTIONS(203), 1,
      anon_sym_DQUOTE,
    ACTIONS(205), 1,
      anon_sym_SQUOTE,
    STATE(204), 1,
      sym_AttValue,
  [2535] = 3,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(561), 1,
      sym_Name,
    STATE(71), 1,
      sym_PEReference,
  [2545] = 1,
    ACTIONS(516), 3,
      anon_sym_PIPE,
      anon_sym_RPAREN,
      sym__S,
  [2551] = 3,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(563), 1,
      anon_sym_RPAREN,
    STATE(169), 1,
      sym_PEReference,
  [2561] = 2,
    ACTIONS(567), 1,
      anon_sym_STAR,
    ACTIONS(565), 2,
      anon_sym_GT,
      sym__S,
  [2569] = 2,
    ACTIONS(571), 1,
      sym__S,
    ACTIONS(569), 2,
      anon_sym_DQUOTE,
      anon_sym_SQUOTE,
  [2577] = 3,
    ACTIONS(555), 1,
      sym__S,
    ACTIONS(557), 1,
      anon_sym_EQ,
    STATE(227), 1,
      sym__Eq,
  [2587] = 2,
    ACTIONS(575), 1,
      sym__S,
    ACTIONS(573), 2,
      anon_sym_DQUOTE,
      anon_sym_SQUOTE,
  [2595] = 3,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(393), 1,
      anon_sym_RPAREN,
    STATE(169), 1,
      sym_PEReference,
  [2605] = 3,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(577), 1,
      sym_Name,
    STATE(138), 1,
      sym_PEReference,
  [2615] = 3,
    ACTIONS(579), 1,
      anon_sym_DQUOTE,
    ACTIONS(581), 1,
      anon_sym_SQUOTE,
    STATE(309), 1,
      sym_PubidLiteral,
  [2625] = 2,
    ACTIONS(585), 1,
      anon_sym_STAR,
    ACTIONS(583), 2,
      anon_sym_GT,
      sym__S,
  [2633] = 1,
    ACTIONS(395), 3,
      anon_sym_RPAREN,
      anon_sym_PERCENT,
      sym__S,
  [2639] = 3,
    ACTIONS(587), 1,
      anon_sym_DQUOTE,
    ACTIONS(589), 1,
      anon_sym_SQUOTE,
    STATE(243), 1,
      sym_SystemLiteral,
  [2649] = 3,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(231), 1,
      anon_sym_RPAREN,
    STATE(169), 1,
      sym_PEReference,
  [2659] = 3,
    ACTIONS(591), 1,
      anon_sym_GT,
    ACTIONS(593), 1,
      sym__S,
    STATE(187), 1,
      sym_NDataDecl,
  [2669] = 3,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(466), 1,
      sym_Name,
    STATE(123), 1,
      sym_PEReference,
  [2679] = 3,
    ACTIONS(579), 1,
      anon_sym_DQUOTE,
    ACTIONS(581), 1,
      anon_sym_SQUOTE,
    STATE(219), 1,
      sym_PubidLiteral,
  [2689] = 3,
    ACTIONS(579), 1,
      anon_sym_DQUOTE,
    ACTIONS(581), 1,
      anon_sym_SQUOTE,
    STATE(217), 1,
      sym_PubidLiteral,
  [2699] = 3,
    ACTIONS(587), 1,
      anon_sym_DQUOTE,
    ACTIONS(589), 1,
      anon_sym_SQUOTE,
    STATE(207), 1,
      sym_SystemLiteral,
  [2709] = 2,
    ACTIONS(597), 1,
      anon_sym_STAR,
    ACTIONS(595), 2,
      anon_sym_GT,
      sym__S,
  [2717] = 3,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    ACTIONS(445), 1,
      sym_Name,
    STATE(222), 1,
      sym_PEReference,
  [2727] = 1,
    ACTIONS(599), 2,
      anon_sym_GT,
      sym__S,
  [2732] = 1,
    ACTIONS(601), 2,
      anon_sym_GT,
      sym__S,
  [2737] = 2,
    ACTIONS(603), 1,
      anon_sym_QMARK_GT,
    ACTIONS(605), 1,
      sym__S,
  [2744] = 2,
    ACTIONS(607), 1,
      sym__S,
    STATE(149), 1,
      sym__VersionInfo,
  [2751] = 1,
    ACTIONS(609), 2,
      anon_sym_QMARK_GT,
      sym__S,
  [2756] = 2,
    ACTIONS(611), 1,
      anon_sym_LBRACK,
    ACTIONS(613), 1,
      sym__S,
  [2763] = 1,
    ACTIONS(615), 2,
      anon_sym_GT,
      sym__S,
  [2768] = 1,
    ACTIONS(617), 2,
      anon_sym_QMARK_GT,
      sym__S,
  [2773] = 2,
    ACTIONS(619), 1,
      anon_sym_GT,
    ACTIONS(621), 1,
      sym__S,
  [2780] = 1,
    ACTIONS(595), 2,
      anon_sym_GT,
      sym__S,
  [2785] = 2,
    ACTIONS(619), 1,
      anon_sym_GT,
    ACTIONS(623), 1,
      anon_sym_NDATA,
  [2792] = 1,
    ACTIONS(625), 2,
      anon_sym_GT,
      sym__S,
  [2797] = 1,
    ACTIONS(457), 2,
      anon_sym_RPAREN,
      anon_sym_PERCENT,
  [2802] = 2,
    ACTIONS(627), 1,
      anon_sym_RPAREN,
    ACTIONS(629), 1,
      sym__S,
  [2809] = 1,
    ACTIONS(631), 2,
      anon_sym_GT,
      sym__S,
  [2814] = 1,
    ACTIONS(633), 2,
      anon_sym_GT,
      sym__S,
  [2819] = 1,
    ACTIONS(635), 2,
      anon_sym_GT,
      sym__S,
  [2824] = 2,
    ACTIONS(637), 1,
      anon_sym_xml,
    ACTIONS(639), 1,
      sym_PITarget,
  [2831] = 2,
    ACTIONS(641), 1,
      anon_sym_LBRACK,
    ACTIONS(643), 1,
      sym__S,
  [2838] = 1,
    ACTIONS(645), 2,
      anon_sym_GT,
      sym__S,
  [2843] = 2,
    ACTIONS(647), 1,
      anon_sym_RPAREN,
    ACTIONS(649), 1,
      sym__S,
  [2850] = 2,
    ACTIONS(651), 1,
      anon_sym_GT,
    ACTIONS(653), 1,
      sym__S,
  [2857] = 1,
    ACTIONS(655), 2,
      anon_sym_GT,
      sym__S,
  [2862] = 1,
    ACTIONS(657), 2,
      anon_sym_GT,
      sym__S,
  [2867] = 1,
    ACTIONS(659), 2,
      anon_sym_GT,
      sym__S,
  [2872] = 1,
    ACTIONS(661), 2,
      anon_sym_GT,
      sym__S,
  [2877] = 2,
    ACTIONS(663), 1,
      anon_sym_RPAREN,
    ACTIONS(665), 1,
      sym__S,
  [2884] = 1,
    ACTIONS(667), 2,
      anon_sym_DQUOTE,
      anon_sym_SQUOTE,
  [2889] = 1,
    ACTIONS(669), 2,
      anon_sym_GT,
      sym__S,
  [2894] = 2,
    ACTIONS(671), 1,
      anon_sym_PIPE,
    ACTIONS(673), 1,
      anon_sym_RPAREN,
  [2901] = 1,
    ACTIONS(675), 2,
      anon_sym_GT,
      sym__S,
  [2906] = 2,
    ACTIONS(677), 1,
      anon_sym_QMARK_GT,
    ACTIONS(679), 1,
      anon_sym_encoding,
  [2913] = 1,
    ACTIONS(681), 2,
      anon_sym_GT,
      sym__S,
  [2918] = 2,
    ACTIONS(591), 1,
      anon_sym_GT,
    ACTIONS(683), 1,
      sym__S,
  [2925] = 2,
    ACTIONS(677), 1,
      anon_sym_QMARK_GT,
    ACTIONS(685), 1,
      sym__S,
  [2932] = 1,
    ACTIONS(687), 2,
      anon_sym_GT,
      sym__S,
  [2937] = 2,
    ACTIONS(689), 1,
      anon_sym_GT,
    ACTIONS(691), 1,
      sym__S,
  [2944] = 1,
    ACTIONS(583), 2,
      anon_sym_GT,
      sym__S,
  [2949] = 2,
    ACTIONS(693), 1,
      anon_sym_GT,
    ACTIONS(695), 1,
      sym__S,
  [2956] = 2,
    ACTIONS(697), 1,
      sym__S,
    ACTIONS(699), 1,
      sym_Nmtoken,
  [2963] = 1,
    ACTIONS(693), 2,
      anon_sym_GT,
      sym__S,
  [2968] = 2,
    ACTIONS(701), 1,
      sym_Name,
    ACTIONS(703), 1,
      sym__S,
  [2975] = 2,
    ACTIONS(505), 1,
      anon_sym_RPAREN,
    ACTIONS(671), 1,
      anon_sym_PIPE,
  [2982] = 2,
    ACTIONS(472), 1,
      anon_sym_GT,
    ACTIONS(705), 1,
      sym__S,
  [2989] = 1,
    ACTIONS(708), 2,
      anon_sym_GT,
      sym__S,
  [2994] = 2,
    ACTIONS(710), 1,
      anon_sym_GT,
    ACTIONS(712), 1,
      sym__S,
  [3001] = 2,
    ACTIONS(714), 1,
      sym__S,
    ACTIONS(716), 1,
      sym_Nmtoken,
  [3008] = 1,
    ACTIONS(718), 2,
      anon_sym_GT,
      sym__S,
  [3013] = 2,
    ACTIONS(720), 1,
      anon_sym_DQUOTE,
    ACTIONS(722), 1,
      anon_sym_SQUOTE,
  [3020] = 1,
    ACTIONS(391), 2,
      anon_sym_PIPE,
      anon_sym_COMMA,
  [3025] = 2,
    ACTIONS(133), 1,
      anon_sym_PERCENT,
    STATE(169), 1,
      sym_PEReference,
  [3032] = 2,
    ACTIONS(724), 1,
      anon_sym_GT,
    ACTIONS(726), 1,
      sym__S,
  [3039] = 2,
    ACTIONS(728), 1,
      sym__S,
    ACTIONS(730), 1,
      sym_Nmtoken,
  [3046] = 1,
    ACTIONS(732), 2,
      anon_sym_GT,
      sym__S,
  [3051] = 2,
    ACTIONS(491), 1,
      anon_sym_RPAREN,
    ACTIONS(671), 1,
      anon_sym_PIPE,
  [3058] = 2,
    ACTIONS(734), 1,
      anon_sym_DQUOTE,
    ACTIONS(736), 1,
      anon_sym_SQUOTE,
  [3065] = 1,
    ACTIONS(573), 2,
      anon_sym_DQUOTE,
      anon_sym_SQUOTE,
  [3070] = 1,
    ACTIONS(738), 2,
      anon_sym_GT,
      sym__S,
  [3075] = 1,
    ACTIONS(529), 2,
      anon_sym_GT,
      sym__S,
  [3080] = 1,
    ACTIONS(740), 2,
      anon_sym_GT,
      sym__S,
  [3085] = 1,
    ACTIONS(742), 2,
      anon_sym_GT,
      sym__S,
  [3090] = 1,
    ACTIONS(744), 2,
      anon_sym_GT,
      sym__S,
  [3095] = 1,
    ACTIONS(746), 2,
      anon_sym_GT,
      sym__S,
  [3100] = 1,
    ACTIONS(748), 2,
      anon_sym_GT,
      sym__S,
  [3105] = 1,
    ACTIONS(750), 2,
      anon_sym_GT,
      sym__S,
  [3110] = 1,
    ACTIONS(752), 1,
      sym__S,
  [3114] = 1,
    ACTIONS(619), 1,
      anon_sym_GT,
  [3118] = 1,
    ACTIONS(754), 1,
      anon_sym_SEMI,
  [3122] = 1,
    ACTIONS(756), 1,
      sym__S,
  [3126] = 1,
    ACTIONS(531), 1,
      anon_sym_STAR,
  [3130] = 1,
    ACTIONS(758), 1,
      sym_VersionNum,
  [3134] = 1,
    ACTIONS(760), 1,
      anon_sym_SEMI,
  [3138] = 1,
    ACTIONS(762), 1,
      anon_sym_EQ,
  [3142] = 1,
    ACTIONS(764), 1,
      sym_VersionNum,
  [3146] = 1,
    ACTIONS(766), 1,
      sym__S,
  [3150] = 1,
    ACTIONS(768), 1,
      sym__S,
  [3154] = 1,
    ACTIONS(770), 1,
      sym__S,
  [3158] = 1,
    ACTIONS(772), 1,
      sym__S,
  [3162] = 1,
    ACTIONS(699), 1,
      sym_Nmtoken,
  [3166] = 1,
    ACTIONS(403), 1,
      anon_sym_PIPE,
  [3170] = 1,
    ACTIONS(774), 1,
      sym__S,
  [3174] = 1,
    ACTIONS(776), 1,
      sym__S,
  [3178] = 1,
    ACTIONS(585), 1,
      anon_sym_STAR,
  [3182] = 1,
    ACTIONS(778), 1,
      sym__S,
  [3186] = 1,
    ACTIONS(780), 1,
      aux_sym_PubidLiteral_token2,
  [3190] = 1,
    ACTIONS(782), 1,
      sym__S,
  [3194] = 1,
    ACTIONS(784), 1,
      anon_sym_LBRACK,
  [3198] = 1,
    ACTIONS(786), 1,
      sym__S,
  [3202] = 1,
    ACTIONS(788), 1,
      anon_sym_GT,
  [3206] = 1,
    ACTIONS(790), 1,
      anon_sym_STAR,
  [3210] = 1,
    ACTIONS(792), 1,
      anon_sym_SQUOTE,
  [3214] = 1,
    ACTIONS(794), 1,
      anon_sym_QMARK_GT,
  [3218] = 1,
    ACTIONS(796), 1,
      anon_sym_SQUOTE,
  [3222] = 1,
    ACTIONS(798), 1,
      aux_sym_PubidLiteral_token1,
  [3226] = 1,
    ACTIONS(800), 1,
      sym_Nmtoken,
  [3230] = 1,
    ACTIONS(671), 1,
      anon_sym_PIPE,
  [3234] = 1,
    ACTIONS(796), 1,
      anon_sym_DQUOTE,
  [3238] = 1,
    ACTIONS(802), 1,
      anon_sym_QMARK_GT,
  [3242] = 1,
    ACTIONS(804), 1,
      anon_sym_PIPE,
  [3246] = 1,
    ACTIONS(806), 1,
      anon_sym_LPAREN,
  [3250] = 1,
    ACTIONS(808), 1,
      aux_sym_SystemLiteral_token2,
  [3254] = 1,
    ACTIONS(810), 1,
      anon_sym_DQUOTE,
  [3258] = 1,
    ACTIONS(812), 1,
      sym_Nmtoken,
  [3262] = 1,
    ACTIONS(814), 1,
      aux_sym_SystemLiteral_token1,
  [3266] = 1,
    ACTIONS(810), 1,
      anon_sym_SQUOTE,
  [3270] = 1,
    ACTIONS(816), 1,
      anon_sym_SQUOTE,
  [3274] = 1,
    ACTIONS(818), 1,
      sym_EncName,
  [3278] = 1,
    ACTIONS(820), 1,
      sym_EncName,
  [3282] = 1,
    ACTIONS(822), 1,
      anon_sym_GT,
  [3286] = 1,
    ACTIONS(792), 1,
      anon_sym_DQUOTE,
  [3290] = 1,
    ACTIONS(824), 1,
      sym_Name,
  [3294] = 1,
    ACTIONS(826), 1,
      aux_sym_CharRef_token2,
  [3298] = 1,
    ACTIONS(828), 1,
      sym_Name,
  [3302] = 1,
    ACTIONS(647), 1,
      anon_sym_RPAREN,
  [3306] = 1,
    ACTIONS(826), 1,
      aux_sym_CharRef_token1,
  [3310] = 1,
    ACTIONS(641), 1,
      anon_sym_LBRACK,
  [3314] = 1,
    ACTIONS(627), 1,
      anon_sym_RPAREN,
  [3318] = 1,
    ACTIONS(597), 1,
      anon_sym_STAR,
  [3322] = 1,
    ACTIONS(830), 1,
      anon_sym_version,
  [3326] = 1,
    ACTIONS(832), 1,
      anon_sym_GT,
  [3330] = 1,
    ACTIONS(834), 1,
      anon_sym_RPAREN,
  [3334] = 1,
    ACTIONS(836), 1,
      anon_sym_GT,
  [3338] = 1,
    ACTIONS(838), 1,
      sym__pi_content,
  [3342] = 1,
    ACTIONS(639), 1,
      sym_PITarget,
  [3346] = 1,
    ACTIONS(840), 1,
      anon_sym_SEMI,
  [3350] = 1,
    ACTIONS(842), 1,
      sym__S,
  [3354] = 1,
    ACTIONS(844), 1,
      sym__S,
  [3358] = 1,
    ACTIONS(846), 1,
      sym__S,
  [3362] = 1,
    ACTIONS(848), 1,
      sym__S,
  [3366] = 1,
    ACTIONS(850), 1,
      sym__S,
  [3370] = 1,
    ACTIONS(695), 1,
      sym__S,
  [3374] = 1,
    ACTIONS(816), 1,
      anon_sym_DQUOTE,
  [3378] = 1,
    ACTIONS(852), 1,
      ts_builtin_sym_end,
  [3382] = 1,
    ACTIONS(854), 1,
      sym_Name,
  [3386] = 1,
    ACTIONS(856), 1,
      anon_sym_SEMI,
  [3390] = 1,
    ACTIONS(858), 1,
      anon_sym_SEMI,
  [3394] = 1,
    ACTIONS(860), 1,
      anon_sym_SEMI,
  [3398] = 1,
    ACTIONS(862), 1,
      anon_sym_SEMI,
  [3402] = 1,
    ACTIONS(864), 1,
      anon_sym_SEMI,
  [3406] = 1,
    ACTIONS(866), 1,
      anon_sym_SEMI,
  [3410] = 1,
    ACTIONS(868), 1,
      anon_sym_SEMI,
  [3414] = 1,
    ACTIONS(870), 1,
      anon_sym_SEMI,
  [3418] = 1,
    ACTIONS(872), 1,
      anon_sym_SEMI,
  [3422] = 1,
    ACTIONS(701), 1,
      sym_Name,
  [3426] = 1,
    ACTIONS(874), 1,
      sym_Name,
  [3430] = 1,
    ACTIONS(876), 1,
      aux_sym_CharRef_token1,
  [3434] = 1,
    ACTIONS(876), 1,
      aux_sym_CharRef_token2,
  [3438] = 1,
    ACTIONS(878), 1,
      sym_Name,
  [3442] = 1,
    ACTIONS(880), 1,
      sym_Name,
  [3446] = 1,
    ACTIONS(882), 1,
      aux_sym_CharRef_token1,
  [3450] = 1,
    ACTIONS(882), 1,
      aux_sym_CharRef_token2,
  [3454] = 1,
    ACTIONS(884), 1,
      sym_Name,
  [3458] = 1,
    ACTIONS(886), 1,
      sym_Name,
  [3462] = 1,
    ACTIONS(888), 1,
      aux_sym_CharRef_token1,
  [3466] = 1,
    ACTIONS(888), 1,
      aux_sym_CharRef_token2,
};

static const uint32_t ts_small_parse_table_map[] = {
  [SMALL_STATE(2)] = 0,
  [SMALL_STATE(3)] = 42,
  [SMALL_STATE(4)] = 83,
  [SMALL_STATE(5)] = 124,
  [SMALL_STATE(6)] = 165,
  [SMALL_STATE(7)] = 206,
  [SMALL_STATE(8)] = 247,
  [SMALL_STATE(9)] = 288,
  [SMALL_STATE(10)] = 329,
  [SMALL_STATE(11)] = 370,
  [SMALL_STATE(12)] = 408,
  [SMALL_STATE(13)] = 436,
  [SMALL_STATE(14)] = 458,
  [SMALL_STATE(15)] = 486,
  [SMALL_STATE(16)] = 514,
  [SMALL_STATE(17)] = 536,
  [SMALL_STATE(18)] = 550,
  [SMALL_STATE(19)] = 578,
  [SMALL_STATE(20)] = 606,
  [SMALL_STATE(21)] = 634,
  [SMALL_STATE(22)] = 659,
  [SMALL_STATE(23)] = 671,
  [SMALL_STATE(24)] = 695,
  [SMALL_STATE(25)] = 719,
  [SMALL_STATE(26)] = 743,
  [SMALL_STATE(27)] = 755,
  [SMALL_STATE(28)] = 767,
  [SMALL_STATE(29)] = 779,
  [SMALL_STATE(30)] = 803,
  [SMALL_STATE(31)] = 827,
  [SMALL_STATE(32)] = 851,
  [SMALL_STATE(33)] = 873,
  [SMALL_STATE(34)] = 897,
  [SMALL_STATE(35)] = 909,
  [SMALL_STATE(36)] = 922,
  [SMALL_STATE(37)] = 935,
  [SMALL_STATE(38)] = 948,
  [SMALL_STATE(39)] = 961,
  [SMALL_STATE(40)] = 974,
  [SMALL_STATE(41)] = 995,
  [SMALL_STATE(42)] = 1008,
  [SMALL_STATE(43)] = 1029,
  [SMALL_STATE(44)] = 1042,
  [SMALL_STATE(45)] = 1063,
  [SMALL_STATE(46)] = 1076,
  [SMALL_STATE(47)] = 1089,
  [SMALL_STATE(48)] = 1102,
  [SMALL_STATE(49)] = 1115,
  [SMALL_STATE(50)] = 1128,
  [SMALL_STATE(51)] = 1141,
  [SMALL_STATE(52)] = 1154,
  [SMALL_STATE(53)] = 1167,
  [SMALL_STATE(54)] = 1180,
  [SMALL_STATE(55)] = 1193,
  [SMALL_STATE(56)] = 1206,
  [SMALL_STATE(57)] = 1231,
  [SMALL_STATE(58)] = 1252,
  [SMALL_STATE(59)] = 1265,
  [SMALL_STATE(60)] = 1278,
  [SMALL_STATE(61)] = 1291,
  [SMALL_STATE(62)] = 1304,
  [SMALL_STATE(63)] = 1324,
  [SMALL_STATE(64)] = 1346,
  [SMALL_STATE(65)] = 1366,
  [SMALL_STATE(66)] = 1388,
  [SMALL_STATE(67)] = 1408,
  [SMALL_STATE(68)] = 1430,
  [SMALL_STATE(69)] = 1452,
  [SMALL_STATE(70)] = 1474,
  [SMALL_STATE(71)] = 1493,
  [SMALL_STATE(72)] = 1512,
  [SMALL_STATE(73)] = 1529,
  [SMALL_STATE(74)] = 1540,
  [SMALL_STATE(75)] = 1551,
  [SMALL_STATE(76)] = 1562,
  [SMALL_STATE(77)] = 1573,
  [SMALL_STATE(78)] = 1588,
  [SMALL_STATE(79)] = 1605,
  [SMALL_STATE(80)] = 1616,
  [SMALL_STATE(81)] = 1635,
  [SMALL_STATE(82)] = 1652,
  [SMALL_STATE(83)] = 1663,
  [SMALL_STATE(84)] = 1680,
  [SMALL_STATE(85)] = 1691,
  [SMALL_STATE(86)] = 1710,
  [SMALL_STATE(87)] = 1721,
  [SMALL_STATE(88)] = 1732,
  [SMALL_STATE(89)] = 1743,
  [SMALL_STATE(90)] = 1760,
  [SMALL_STATE(91)] = 1771,
  [SMALL_STATE(92)] = 1790,
  [SMALL_STATE(93)] = 1804,
  [SMALL_STATE(94)] = 1818,
  [SMALL_STATE(95)] = 1834,
  [SMALL_STATE(96)] = 1842,
  [SMALL_STATE(97)] = 1852,
  [SMALL_STATE(98)] = 1862,
  [SMALL_STATE(99)] = 1872,
  [SMALL_STATE(100)] = 1880,
  [SMALL_STATE(101)] = 1888,
  [SMALL_STATE(102)] = 1902,
  [SMALL_STATE(103)] = 1912,
  [SMALL_STATE(104)] = 1922,
  [SMALL_STATE(105)] = 1938,
  [SMALL_STATE(106)] = 1952,
  [SMALL_STATE(107)] = 1962,
  [SMALL_STATE(108)] = 1976,
  [SMALL_STATE(109)] = 1990,
  [SMALL_STATE(110)] = 2004,
  [SMALL_STATE(111)] = 2014,
  [SMALL_STATE(112)] = 2028,
  [SMALL_STATE(113)] = 2042,
  [SMALL_STATE(114)] = 2058,
  [SMALL_STATE(115)] = 2066,
  [SMALL_STATE(116)] = 2080,
  [SMALL_STATE(117)] = 2096,
  [SMALL_STATE(118)] = 2109,
  [SMALL_STATE(119)] = 2122,
  [SMALL_STATE(120)] = 2135,
  [SMALL_STATE(121)] = 2146,
  [SMALL_STATE(122)] = 2159,
  [SMALL_STATE(123)] = 2166,
  [SMALL_STATE(124)] = 2173,
  [SMALL_STATE(125)] = 2186,
  [SMALL_STATE(126)] = 2199,
  [SMALL_STATE(127)] = 2212,
  [SMALL_STATE(128)] = 2219,
  [SMALL_STATE(129)] = 2232,
  [SMALL_STATE(130)] = 2243,
  [SMALL_STATE(131)] = 2256,
  [SMALL_STATE(132)] = 2269,
  [SMALL_STATE(133)] = 2282,
  [SMALL_STATE(134)] = 2295,
  [SMALL_STATE(135)] = 2308,
  [SMALL_STATE(136)] = 2321,
  [SMALL_STATE(137)] = 2334,
  [SMALL_STATE(138)] = 2345,
  [SMALL_STATE(139)] = 2352,
  [SMALL_STATE(140)] = 2365,
  [SMALL_STATE(141)] = 2378,
  [SMALL_STATE(142)] = 2389,
  [SMALL_STATE(143)] = 2402,
  [SMALL_STATE(144)] = 2411,
  [SMALL_STATE(145)] = 2418,
  [SMALL_STATE(146)] = 2427,
  [SMALL_STATE(147)] = 2435,
  [SMALL_STATE(148)] = 2445,
  [SMALL_STATE(149)] = 2453,
  [SMALL_STATE(150)] = 2463,
  [SMALL_STATE(151)] = 2473,
  [SMALL_STATE(152)] = 2479,
  [SMALL_STATE(153)] = 2489,
  [SMALL_STATE(154)] = 2499,
  [SMALL_STATE(155)] = 2509,
  [SMALL_STATE(156)] = 2519,
  [SMALL_STATE(157)] = 2525,
  [SMALL_STATE(158)] = 2535,
  [SMALL_STATE(159)] = 2545,
  [SMALL_STATE(160)] = 2551,
  [SMALL_STATE(161)] = 2561,
  [SMALL_STATE(162)] = 2569,
  [SMALL_STATE(163)] = 2577,
  [SMALL_STATE(164)] = 2587,
  [SMALL_STATE(165)] = 2595,
  [SMALL_STATE(166)] = 2605,
  [SMALL_STATE(167)] = 2615,
  [SMALL_STATE(168)] = 2625,
  [SMALL_STATE(169)] = 2633,
  [SMALL_STATE(170)] = 2639,
  [SMALL_STATE(171)] = 2649,
  [SMALL_STATE(172)] = 2659,
  [SMALL_STATE(173)] = 2669,
  [SMALL_STATE(174)] = 2679,
  [SMALL_STATE(175)] = 2689,
  [SMALL_STATE(176)] = 2699,
  [SMALL_STATE(177)] = 2709,
  [SMALL_STATE(178)] = 2717,
  [SMALL_STATE(179)] = 2727,
  [SMALL_STATE(180)] = 2732,
  [SMALL_STATE(181)] = 2737,
  [SMALL_STATE(182)] = 2744,
  [SMALL_STATE(183)] = 2751,
  [SMALL_STATE(184)] = 2756,
  [SMALL_STATE(185)] = 2763,
  [SMALL_STATE(186)] = 2768,
  [SMALL_STATE(187)] = 2773,
  [SMALL_STATE(188)] = 2780,
  [SMALL_STATE(189)] = 2785,
  [SMALL_STATE(190)] = 2792,
  [SMALL_STATE(191)] = 2797,
  [SMALL_STATE(192)] = 2802,
  [SMALL_STATE(193)] = 2809,
  [SMALL_STATE(194)] = 2814,
  [SMALL_STATE(195)] = 2819,
  [SMALL_STATE(196)] = 2824,
  [SMALL_STATE(197)] = 2831,
  [SMALL_STATE(198)] = 2838,
  [SMALL_STATE(199)] = 2843,
  [SMALL_STATE(200)] = 2850,
  [SMALL_STATE(201)] = 2857,
  [SMALL_STATE(202)] = 2862,
  [SMALL_STATE(203)] = 2867,
  [SMALL_STATE(204)] = 2872,
  [SMALL_STATE(205)] = 2877,
  [SMALL_STATE(206)] = 2884,
  [SMALL_STATE(207)] = 2889,
  [SMALL_STATE(208)] = 2894,
  [SMALL_STATE(209)] = 2901,
  [SMALL_STATE(210)] = 2906,
  [SMALL_STATE(211)] = 2913,
  [SMALL_STATE(212)] = 2918,
  [SMALL_STATE(213)] = 2925,
  [SMALL_STATE(214)] = 2932,
  [SMALL_STATE(215)] = 2937,
  [SMALL_STATE(216)] = 2944,
  [SMALL_STATE(217)] = 2949,
  [SMALL_STATE(218)] = 2956,
  [SMALL_STATE(219)] = 2963,
  [SMALL_STATE(220)] = 2968,
  [SMALL_STATE(221)] = 2975,
  [SMALL_STATE(222)] = 2982,
  [SMALL_STATE(223)] = 2989,
  [SMALL_STATE(224)] = 2994,
  [SMALL_STATE(225)] = 3001,
  [SMALL_STATE(226)] = 3008,
  [SMALL_STATE(227)] = 3013,
  [SMALL_STATE(228)] = 3020,
  [SMALL_STATE(229)] = 3025,
  [SMALL_STATE(230)] = 3032,
  [SMALL_STATE(231)] = 3039,
  [SMALL_STATE(232)] = 3046,
  [SMALL_STATE(233)] = 3051,
  [SMALL_STATE(234)] = 3058,
  [SMALL_STATE(235)] = 3065,
  [SMALL_STATE(236)] = 3070,
  [SMALL_STATE(237)] = 3075,
  [SMALL_STATE(238)] = 3080,
  [SMALL_STATE(239)] = 3085,
  [SMALL_STATE(240)] = 3090,
  [SMALL_STATE(241)] = 3095,
  [SMALL_STATE(242)] = 3100,
  [SMALL_STATE(243)] = 3105,
  [SMALL_STATE(244)] = 3110,
  [SMALL_STATE(245)] = 3114,
  [SMALL_STATE(246)] = 3118,
  [SMALL_STATE(247)] = 3122,
  [SMALL_STATE(248)] = 3126,
  [SMALL_STATE(249)] = 3130,
  [SMALL_STATE(250)] = 3134,
  [SMALL_STATE(251)] = 3138,
  [SMALL_STATE(252)] = 3142,
  [SMALL_STATE(253)] = 3146,
  [SMALL_STATE(254)] = 3150,
  [SMALL_STATE(255)] = 3154,
  [SMALL_STATE(256)] = 3158,
  [SMALL_STATE(257)] = 3162,
  [SMALL_STATE(258)] = 3166,
  [SMALL_STATE(259)] = 3170,
  [SMALL_STATE(260)] = 3174,
  [SMALL_STATE(261)] = 3178,
  [SMALL_STATE(262)] = 3182,
  [SMALL_STATE(263)] = 3186,
  [SMALL_STATE(264)] = 3190,
  [SMALL_STATE(265)] = 3194,
  [SMALL_STATE(266)] = 3198,
  [SMALL_STATE(267)] = 3202,
  [SMALL_STATE(268)] = 3206,
  [SMALL_STATE(269)] = 3210,
  [SMALL_STATE(270)] = 3214,
  [SMALL_STATE(271)] = 3218,
  [SMALL_STATE(272)] = 3222,
  [SMALL_STATE(273)] = 3226,
  [SMALL_STATE(274)] = 3230,
  [SMALL_STATE(275)] = 3234,
  [SMALL_STATE(276)] = 3238,
  [SMALL_STATE(277)] = 3242,
  [SMALL_STATE(278)] = 3246,
  [SMALL_STATE(279)] = 3250,
  [SMALL_STATE(280)] = 3254,
  [SMALL_STATE(281)] = 3258,
  [SMALL_STATE(282)] = 3262,
  [SMALL_STATE(283)] = 3266,
  [SMALL_STATE(284)] = 3270,
  [SMALL_STATE(285)] = 3274,
  [SMALL_STATE(286)] = 3278,
  [SMALL_STATE(287)] = 3282,
  [SMALL_STATE(288)] = 3286,
  [SMALL_STATE(289)] = 3290,
  [SMALL_STATE(290)] = 3294,
  [SMALL_STATE(291)] = 3298,
  [SMALL_STATE(292)] = 3302,
  [SMALL_STATE(293)] = 3306,
  [SMALL_STATE(294)] = 3310,
  [SMALL_STATE(295)] = 3314,
  [SMALL_STATE(296)] = 3318,
  [SMALL_STATE(297)] = 3322,
  [SMALL_STATE(298)] = 3326,
  [SMALL_STATE(299)] = 3330,
  [SMALL_STATE(300)] = 3334,
  [SMALL_STATE(301)] = 3338,
  [SMALL_STATE(302)] = 3342,
  [SMALL_STATE(303)] = 3346,
  [SMALL_STATE(304)] = 3350,
  [SMALL_STATE(305)] = 3354,
  [SMALL_STATE(306)] = 3358,
  [SMALL_STATE(307)] = 3362,
  [SMALL_STATE(308)] = 3366,
  [SMALL_STATE(309)] = 3370,
  [SMALL_STATE(310)] = 3374,
  [SMALL_STATE(311)] = 3378,
  [SMALL_STATE(312)] = 3382,
  [SMALL_STATE(313)] = 3386,
  [SMALL_STATE(314)] = 3390,
  [SMALL_STATE(315)] = 3394,
  [SMALL_STATE(316)] = 3398,
  [SMALL_STATE(317)] = 3402,
  [SMALL_STATE(318)] = 3406,
  [SMALL_STATE(319)] = 3410,
  [SMALL_STATE(320)] = 3414,
  [SMALL_STATE(321)] = 3418,
  [SMALL_STATE(322)] = 3422,
  [SMALL_STATE(323)] = 3426,
  [SMALL_STATE(324)] = 3430,
  [SMALL_STATE(325)] = 3434,
  [SMALL_STATE(326)] = 3438,
  [SMALL_STATE(327)] = 3442,
  [SMALL_STATE(328)] = 3446,
  [SMALL_STATE(329)] = 3450,
  [SMALL_STATE(330)] = 3454,
  [SMALL_STATE(331)] = 3458,
  [SMALL_STATE(332)] = 3462,
  [SMALL_STATE(333)] = 3466,
};

static const TSParseActionEntry ts_parse_actions[] = {
  [0] = {.entry = {.count = 0, .reusable = false}},
  [1] = {.entry = {.count = 1, .reusable = false}}, RECOVER(),
  [3] = {.entry = {.count = 1, .reusable = true}}, SHIFT(196),
  [5] = {.entry = {.count = 1, .reusable = true}}, SHIFT(107),
  [7] = {.entry = {.count = 1, .reusable = false}}, SHIFT(130),
  [9] = {.entry = {.count = 1, .reusable = true}}, SHIFT(312),
  [11] = {.entry = {.count = 1, .reusable = true}}, SHIFT(9),
  [13] = {.entry = {.count = 1, .reusable = true}}, SHIFT(58),
  [15] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_extSubset_repeat1, 2, 0, 0),
  [17] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_extSubset_repeat1, 2, 0, 0), SHIFT_REPEAT(302),
  [20] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_extSubset_repeat1, 2, 0, 0), SHIFT_REPEAT(107),
  [23] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_extSubset_repeat1, 2, 0, 0), SHIFT_REPEAT(130),
  [26] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_extSubset_repeat1, 2, 0, 0), SHIFT_REPEAT(312),
  [29] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_extSubset_repeat1, 2, 0, 0), SHIFT_REPEAT(2),
  [32] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_extSubset_repeat1, 2, 0, 0), SHIFT_REPEAT(58),
  [35] = {.entry = {.count = 1, .reusable = true}}, SHIFT(302),
  [37] = {.entry = {.count = 1, .reusable = true}}, SHIFT(36),
  [39] = {.entry = {.count = 1, .reusable = true}}, SHIFT(7),
  [41] = {.entry = {.count = 1, .reusable = true}}, SHIFT(60),
  [43] = {.entry = {.count = 1, .reusable = true}}, SHIFT(2),
  [45] = {.entry = {.count = 1, .reusable = true}}, SHIFT(49),
  [47] = {.entry = {.count = 1, .reusable = true}}, SHIFT(4),
  [49] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_extSubset, 2, 0, 0),
  [51] = {.entry = {.count = 1, .reusable = true}}, SHIFT(47),
  [53] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_extSubset, 1, 0, 0),
  [55] = {.entry = {.count = 1, .reusable = true}}, SHIFT(8),
  [57] = {.entry = {.count = 1, .reusable = true}}, SHIFT(6),
  [59] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_EntityValue_repeat2, 2, 0, 0), SHIFT_REPEAT(330),
  [62] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_EntityValue_repeat2, 2, 0, 0),
  [64] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_EntityValue_repeat2, 2, 0, 0), SHIFT_REPEAT(12),
  [67] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_EntityValue_repeat2, 2, 0, 0), SHIFT_REPEAT(323),
  [70] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_EntityValue_repeat2, 2, 0, 0), SHIFT_REPEAT(324),
  [73] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_EntityValue_repeat2, 2, 0, 0), SHIFT_REPEAT(325),
  [76] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__cp, 1, 0, 0),
  [78] = {.entry = {.count = 1, .reusable = true}}, SHIFT(99),
  [80] = {.entry = {.count = 1, .reusable = true}}, SHIFT(330),
  [82] = {.entry = {.count = 1, .reusable = true}}, SHIFT(193),
  [84] = {.entry = {.count = 1, .reusable = true}}, SHIFT(19),
  [86] = {.entry = {.count = 1, .reusable = false}}, SHIFT(323),
  [88] = {.entry = {.count = 1, .reusable = false}}, SHIFT(324),
  [90] = {.entry = {.count = 1, .reusable = true}}, SHIFT(325),
  [92] = {.entry = {.count = 1, .reusable = true}}, SHIFT(326),
  [94] = {.entry = {.count = 1, .reusable = true}}, SHIFT(18),
  [96] = {.entry = {.count = 1, .reusable = false}}, SHIFT(291),
  [98] = {.entry = {.count = 1, .reusable = false}}, SHIFT(293),
  [100] = {.entry = {.count = 1, .reusable = true}}, SHIFT(290),
  [102] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_PEReference, 3, 0, 0),
  [104] = {.entry = {.count = 1, .reusable = true}}, SHIFT(238),
  [106] = {.entry = {.count = 1, .reusable = true}}, SHIFT(20),
  [108] = {.entry = {.count = 1, .reusable = true}}, SHIFT(12),
  [110] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_EntityValue_repeat1, 2, 0, 0), SHIFT_REPEAT(326),
  [113] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_EntityValue_repeat1, 2, 0, 0),
  [115] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_EntityValue_repeat1, 2, 0, 0), SHIFT_REPEAT(20),
  [118] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_EntityValue_repeat1, 2, 0, 0), SHIFT_REPEAT(291),
  [121] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_EntityValue_repeat1, 2, 0, 0), SHIFT_REPEAT(293),
  [124] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_EntityValue_repeat1, 2, 0, 0), SHIFT_REPEAT(290),
  [127] = {.entry = {.count = 1, .reusable = true}}, SHIFT(225),
  [129] = {.entry = {.count = 1, .reusable = true}}, SHIFT(194),
  [131] = {.entry = {.count = 1, .reusable = true}}, SHIFT(244),
  [133] = {.entry = {.count = 1, .reusable = true}}, SHIFT(322),
  [135] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__choice, 5, 0, 0),
  [137] = {.entry = {.count = 1, .reusable = true}}, SHIFT(203),
  [139] = {.entry = {.count = 1, .reusable = false}}, SHIFT(331),
  [141] = {.entry = {.count = 1, .reusable = false}}, SHIFT(332),
  [143] = {.entry = {.count = 1, .reusable = true}}, SHIFT(333),
  [145] = {.entry = {.count = 1, .reusable = true}}, SHIFT(30),
  [147] = {.entry = {.count = 1, .reusable = false}}, SHIFT(327),
  [149] = {.entry = {.count = 1, .reusable = false}}, SHIFT(328),
  [151] = {.entry = {.count = 1, .reusable = true}}, SHIFT(329),
  [153] = {.entry = {.count = 1, .reusable = true}}, SHIFT(31),
  [155] = {.entry = {.count = 1, .reusable = true}}, SHIFT(214),
  [157] = {.entry = {.count = 1, .reusable = true}}, SHIFT(23),
  [159] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__choice, 6, 0, 0),
  [161] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__choice, 7, 0, 0),
  [163] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__choice, 3, 0, 0),
  [165] = {.entry = {.count = 1, .reusable = true}}, SHIFT(24),
  [167] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_AttValue_repeat2, 2, 0, 0),
  [169] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_AttValue_repeat2, 2, 0, 0), SHIFT_REPEAT(331),
  [172] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_AttValue_repeat2, 2, 0, 0), SHIFT_REPEAT(332),
  [175] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_AttValue_repeat2, 2, 0, 0), SHIFT_REPEAT(333),
  [178] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_AttValue_repeat2, 2, 0, 0), SHIFT_REPEAT(30),
  [181] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_AttValue_repeat1, 2, 0, 0),
  [183] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_AttValue_repeat1, 2, 0, 0), SHIFT_REPEAT(327),
  [186] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_AttValue_repeat1, 2, 0, 0), SHIFT_REPEAT(328),
  [189] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_AttValue_repeat1, 2, 0, 0), SHIFT_REPEAT(329),
  [192] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_AttValue_repeat1, 2, 0, 0), SHIFT_REPEAT(31),
  [195] = {.entry = {.count = 1, .reusable = true}}, SHIFT(226),
  [197] = {.entry = {.count = 1, .reusable = true}}, SHIFT(56),
  [199] = {.entry = {.count = 1, .reusable = true}}, SHIFT(236),
  [201] = {.entry = {.count = 1, .reusable = true}}, SHIFT(247),
  [203] = {.entry = {.count = 1, .reusable = true}}, SHIFT(29),
  [205] = {.entry = {.count = 1, .reusable = true}}, SHIFT(25),
  [207] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__choice, 4, 0, 0),
  [209] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_AttlistDecl, 7, 0, 0),
  [211] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_AttlistDecl, 7, 0, 0),
  [213] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_conditionalSect, 6, 0, 0),
  [215] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_conditionalSect, 6, 0, 0),
  [217] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_NotationDecl, 7, 0, 0),
  [219] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_NotationDecl, 7, 0, 0),
  [221] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_PI, 3, 0, 0),
  [223] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_PI, 3, 0, 0),
  [225] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_elementdecl, 8, 0, 0),
  [227] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_elementdecl, 8, 0, 0),
  [229] = {.entry = {.count = 1, .reusable = true}}, SHIFT(66),
  [231] = {.entry = {.count = 1, .reusable = true}}, SHIFT(22),
  [233] = {.entry = {.count = 1, .reusable = true}}, SHIFT(92),
  [235] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_NotationDecl, 8, 0, 0),
  [237] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_NotationDecl, 8, 0, 0),
  [239] = {.entry = {.count = 1, .reusable = true}}, SHIFT(34),
  [241] = {.entry = {.count = 1, .reusable = true}}, SHIFT(109),
  [243] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_elementdecl, 7, 0, 0),
  [245] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_elementdecl, 7, 0, 0),
  [247] = {.entry = {.count = 1, .reusable = true}}, SHIFT(28),
  [249] = {.entry = {.count = 1, .reusable = true}}, SHIFT(101),
  [251] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_PEReference, 3, 0, 0),
  [253] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_conditionalSect, 7, 0, 0),
  [255] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_conditionalSect, 7, 0, 0),
  [257] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_PEDecl, 9, 0, 0),
  [259] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_PEDecl, 9, 0, 0),
  [261] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_conditionalSect, 4, 0, 0),
  [263] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_conditionalSect, 4, 0, 0),
  [265] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_GEDecl, 9, 0, 0),
  [267] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_GEDecl, 9, 0, 0),
  [269] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__EntityDecl, 1, 0, 0),
  [271] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym__EntityDecl, 1, 0, 0),
  [273] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_GEDecl, 7, 0, 0),
  [275] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_GEDecl, 7, 0, 0),
  [277] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_GEDecl, 8, 0, 0),
  [279] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_GEDecl, 8, 0, 0),
  [281] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_AttlistDecl, 6, 0, 0),
  [283] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_AttlistDecl, 6, 0, 0),
  [285] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_PI, 5, 0, 0),
  [287] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_PI, 5, 0, 0),
  [289] = {.entry = {.count = 1, .reusable = true}}, SHIFT(45),
  [291] = {.entry = {.count = 1, .reusable = true}}, SHIFT(62),
  [293] = {.entry = {.count = 1, .reusable = true}}, SHIFT(67),
  [295] = {.entry = {.count = 1, .reusable = true}}, SHIFT(69),
  [297] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__markupdecl, 1, 0, 0),
  [299] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym__markupdecl, 1, 0, 0),
  [301] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_PEDecl, 10, 0, 0),
  [303] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_PEDecl, 10, 0, 0),
  [305] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_conditionalSect, 5, 0, 0),
  [307] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_conditionalSect, 5, 0, 0),
  [309] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_AttlistDecl, 5, 0, 0),
  [311] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_AttlistDecl, 5, 0, 0),
  [313] = {.entry = {.count = 1, .reusable = true}}, SHIFT(81),
  [315] = {.entry = {.count = 1, .reusable = true}}, SHIFT(132),
  [317] = {.entry = {.count = 1, .reusable = true}}, SHIFT(261),
  [319] = {.entry = {.count = 1, .reusable = true}}, SHIFT(104),
  [321] = {.entry = {.count = 1, .reusable = true}}, SHIFT(72),
  [323] = {.entry = {.count = 1, .reusable = true}}, SHIFT(177),
  [325] = {.entry = {.count = 1, .reusable = true}}, SHIFT(113),
  [327] = {.entry = {.count = 1, .reusable = true}}, SHIFT(78),
  [329] = {.entry = {.count = 1, .reusable = true}}, SHIFT(161),
  [331] = {.entry = {.count = 1, .reusable = true}}, SHIFT(94),
  [333] = {.entry = {.count = 1, .reusable = true}}, SHIFT(296),
  [335] = {.entry = {.count = 1, .reusable = true}}, SHIFT(116),
  [337] = {.entry = {.count = 1, .reusable = true}}, SHIFT(65),
  [339] = {.entry = {.count = 1, .reusable = true}}, SHIFT(192),
  [341] = {.entry = {.count = 1, .reusable = true}}, SHIFT(143),
  [343] = {.entry = {.count = 1, .reusable = true}}, SHIFT(277),
  [345] = {.entry = {.count = 1, .reusable = true}}, SHIFT(199),
  [347] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_TextDecl, 6, 0, 0),
  [349] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_TextDecl, 6, 0, 0),
  [351] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__Reference, 1, 0, 0),
  [353] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym__Reference, 1, 0, 0),
  [355] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_TextDecl, 5, 0, 0),
  [357] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_TextDecl, 5, 0, 0),
  [359] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym__choice_repeat1, 2, 0, 0), SHIFT_REPEAT(66),
  [362] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym__choice_repeat1, 2, 0, 0),
  [364] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym__choice_repeat1, 2, 0, 0), SHIFT_REPEAT(228),
  [367] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_CharRef, 3, 0, 0),
  [369] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_CharRef, 3, 0, 0),
  [371] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_EntityRef, 3, 0, 0),
  [373] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_EntityRef, 3, 0, 0),
  [375] = {.entry = {.count = 1, .reusable = true}}, SHIFT(15),
  [377] = {.entry = {.count = 1, .reusable = true}}, SHIFT(14),
  [379] = {.entry = {.count = 1, .reusable = true}}, SHIFT(255),
  [381] = {.entry = {.count = 1, .reusable = true}}, SHIFT(266),
  [383] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_TextDecl, 4, 0, 0),
  [385] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_TextDecl, 4, 0, 0),
  [387] = {.entry = {.count = 1, .reusable = true}}, SHIFT(256),
  [389] = {.entry = {.count = 1, .reusable = true}}, SHIFT(205),
  [391] = {.entry = {.count = 1, .reusable = true}}, SHIFT(64),
  [393] = {.entry = {.count = 1, .reusable = true}}, SHIFT(26),
  [395] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym__choice_repeat2, 2, 0, 0),
  [397] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym__choice_repeat2, 2, 0, 0), SHIFT_REPEAT(322),
  [400] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym__choice_repeat2, 2, 0, 0), SHIFT_REPEAT(229),
  [403] = {.entry = {.count = 1, .reusable = true}}, SHIFT(126),
  [405] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__cp, 2, 0, 0),
  [407] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym__choice_repeat1, 4, 0, 0),
  [409] = {.entry = {.count = 1, .reusable = true}}, SHIFT(248),
  [411] = {.entry = {.count = 1, .reusable = true}}, SHIFT(165),
  [413] = {.entry = {.count = 1, .reusable = true}}, SHIFT(184),
  [415] = {.entry = {.count = 1, .reusable = true}}, SHIFT(120),
  [417] = {.entry = {.count = 1, .reusable = true}}, SHIFT(171),
  [419] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_children, 1, 0, 0),
  [421] = {.entry = {.count = 1, .reusable = true}}, SHIFT(198),
  [423] = {.entry = {.count = 1, .reusable = true}}, SHIFT(160),
  [425] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_NotationType_repeat1, 2, 0, 0),
  [427] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_NotationType_repeat1, 2, 0, 0), SHIFT_REPEAT(143),
  [430] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_NotationType_repeat1, 2, 0, 0), SHIFT_REPEAT(277),
  [433] = {.entry = {.count = 1, .reusable = true}}, SHIFT(168),
  [435] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym__choice_repeat1, 3, 0, 0),
  [437] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_Mixed_repeat1, 2, 0, 0), SHIFT_REPEAT(132),
  [440] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_Mixed_repeat1, 2, 0, 0),
  [442] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_Mixed_repeat1, 2, 0, 0), SHIFT_REPEAT(258),
  [445] = {.entry = {.count = 1, .reusable = true}}, SHIFT(254),
  [447] = {.entry = {.count = 1, .reusable = true}}, SHIFT(54),
  [449] = {.entry = {.count = 1, .reusable = true}}, SHIFT(231),
  [451] = {.entry = {.count = 1, .reusable = true}}, SHIFT(232),
  [453] = {.entry = {.count = 1, .reusable = true}}, SHIFT(233),
  [455] = {.entry = {.count = 1, .reusable = true}}, SHIFT(197),
  [457] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_Mixed_repeat2, 2, 0, 0),
  [459] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_Mixed_repeat2, 2, 0, 0), SHIFT_REPEAT(322),
  [462] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_NotationType_repeat1, 3, 0, 0),
  [464] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_Mixed_repeat1, 3, 0, 0),
  [466] = {.entry = {.count = 1, .reusable = true}}, SHIFT(123),
  [468] = {.entry = {.count = 1, .reusable = true}}, SHIFT(166),
  [470] = {.entry = {.count = 1, .reusable = true}}, SHIFT(146),
  [472] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_AttlistDecl_repeat1, 2, 0, 0),
  [474] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_AttlistDecl_repeat1, 2, 0, 0), SHIFT_REPEAT(178),
  [477] = {.entry = {.count = 1, .reusable = true}}, SHIFT(307),
  [479] = {.entry = {.count = 1, .reusable = true}}, SHIFT(306),
  [481] = {.entry = {.count = 1, .reusable = true}}, SHIFT(305),
  [483] = {.entry = {.count = 1, .reusable = true}}, SHIFT(304),
  [485] = {.entry = {.count = 1, .reusable = true}}, SHIFT(35),
  [487] = {.entry = {.count = 1, .reusable = true}}, SHIFT(127),
  [489] = {.entry = {.count = 1, .reusable = true}}, SHIFT(173),
  [491] = {.entry = {.count = 1, .reusable = true}}, SHIFT(223),
  [493] = {.entry = {.count = 1, .reusable = true}}, SHIFT(221),
  [495] = {.entry = {.count = 1, .reusable = true}}, SHIFT(91),
  [497] = {.entry = {.count = 1, .reusable = true}}, SHIFT(158),
  [499] = {.entry = {.count = 1, .reusable = true}}, SHIFT(131),
  [501] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_Mixed_repeat1, 4, 0, 0),
  [503] = {.entry = {.count = 1, .reusable = true}}, SHIFT(268),
  [505] = {.entry = {.count = 1, .reusable = true}}, SHIFT(209),
  [507] = {.entry = {.count = 1, .reusable = true}}, SHIFT(208),
  [509] = {.entry = {.count = 1, .reusable = true}}, SHIFT(61),
  [511] = {.entry = {.count = 1, .reusable = true}}, SHIFT(117),
  [513] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_Enumeration_repeat1, 2, 0, 0), SHIFT_REPEAT(231),
  [516] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_Enumeration_repeat1, 2, 0, 0),
  [518] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_Enumeration_repeat1, 2, 0, 0), SHIFT_REPEAT(274),
  [521] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_NotationType_repeat1, 1, 0, 0),
  [523] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_NotationType_repeat1, 1, 0, 0), SHIFT_REPEAT(144),
  [526] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_NotationType_repeat1, 2, 0, 0), SHIFT_REPEAT(122),
  [529] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_Mixed, 6, 0, 0),
  [531] = {.entry = {.count = 1, .reusable = true}}, SHIFT(211),
  [533] = {.entry = {.count = 1, .reusable = true}}, SHIFT(239),
  [535] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_Mixed_repeat2, 1, 0, 0),
  [537] = {.entry = {.count = 1, .reusable = true}}, SHIFT(191),
  [539] = {.entry = {.count = 1, .reusable = true}}, SHIFT(86),
  [541] = {.entry = {.count = 1, .reusable = true}}, SHIFT(210),
  [543] = {.entry = {.count = 1, .reusable = true}}, SHIFT(262),
  [545] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_Enumeration_repeat1, 4, 0, 0),
  [547] = {.entry = {.count = 1, .reusable = true}}, SHIFT(141),
  [549] = {.entry = {.count = 1, .reusable = true}}, SHIFT(260),
  [551] = {.entry = {.count = 1, .reusable = true}}, SHIFT(253),
  [553] = {.entry = {.count = 1, .reusable = true}}, SHIFT(220),
  [555] = {.entry = {.count = 1, .reusable = true}}, SHIFT(251),
  [557] = {.entry = {.count = 1, .reusable = true}}, SHIFT(162),
  [559] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_Enumeration_repeat1, 3, 0, 0),
  [561] = {.entry = {.count = 1, .reusable = true}}, SHIFT(71),
  [563] = {.entry = {.count = 1, .reusable = true}}, SHIFT(27),
  [565] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_Mixed, 3, 0, 0),
  [567] = {.entry = {.count = 1, .reusable = true}}, SHIFT(188),
  [569] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__Eq, 1, 0, 0),
  [571] = {.entry = {.count = 1, .reusable = true}}, SHIFT(235),
  [573] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__Eq, 2, 0, 0),
  [575] = {.entry = {.count = 1, .reusable = true}}, SHIFT(206),
  [577] = {.entry = {.count = 1, .reusable = true}}, SHIFT(138),
  [579] = {.entry = {.count = 1, .reusable = true}}, SHIFT(272),
  [581] = {.entry = {.count = 1, .reusable = true}}, SHIFT(263),
  [583] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_Mixed, 5, 0, 0),
  [585] = {.entry = {.count = 1, .reusable = true}}, SHIFT(237),
  [587] = {.entry = {.count = 1, .reusable = true}}, SHIFT(282),
  [589] = {.entry = {.count = 1, .reusable = true}}, SHIFT(279),
  [591] = {.entry = {.count = 1, .reusable = true}}, SHIFT(52),
  [593] = {.entry = {.count = 1, .reusable = true}}, SHIFT(189),
  [595] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_Mixed, 4, 0, 0),
  [597] = {.entry = {.count = 1, .reusable = true}}, SHIFT(216),
  [599] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_Mixed, 8, 0, 0),
  [601] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_NotationType, 7, 0, 0),
  [603] = {.entry = {.count = 1, .reusable = true}}, SHIFT(38),
  [605] = {.entry = {.count = 1, .reusable = true}}, SHIFT(301),
  [607] = {.entry = {.count = 1, .reusable = true}}, SHIFT(297),
  [609] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__VersionInfo, 6, 0, 0),
  [611] = {.entry = {.count = 1, .reusable = true}}, SHIFT(5),
  [613] = {.entry = {.count = 1, .reusable = true}}, SHIFT(294),
  [615] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_NotationType, 9, 0, 0),
  [617] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__EncodingDecl, 6, 0, 0),
  [619] = {.entry = {.count = 1, .reusable = true}}, SHIFT(53),
  [621] = {.entry = {.count = 1, .reusable = true}}, SHIFT(300),
  [623] = {.entry = {.count = 1, .reusable = true}}, SHIFT(308),
  [625] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_NotationType, 8, 0, 0),
  [627] = {.entry = {.count = 1, .reusable = true}}, SHIFT(190),
  [629] = {.entry = {.count = 1, .reusable = true}}, SHIFT(299),
  [631] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_EntityValue, 2, 0, 0),
  [633] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__AttType, 1, 0, 0),
  [635] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_NotationType, 6, 0, 0),
  [637] = {.entry = {.count = 1, .reusable = true}}, SHIFT(182),
  [639] = {.entry = {.count = 1, .reusable = true}}, SHIFT(181),
  [641] = {.entry = {.count = 1, .reusable = true}}, SHIFT(10),
  [643] = {.entry = {.count = 1, .reusable = true}}, SHIFT(265),
  [645] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_children, 2, 0, 0),
  [647] = {.entry = {.count = 1, .reusable = true}}, SHIFT(180),
  [649] = {.entry = {.count = 1, .reusable = true}}, SHIFT(295),
  [651] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_AttDef, 4, 0, 0),
  [653] = {.entry = {.count = 1, .reusable = true}}, SHIFT(33),
  [655] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_Enumeration, 6, 0, 0),
  [657] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__EnumeratedType, 1, 0, 0),
  [659] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_AttValue, 3, 0, 1),
  [661] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_DefaultDecl, 3, 0, 0),
  [663] = {.entry = {.count = 1, .reusable = true}}, SHIFT(195),
  [665] = {.entry = {.count = 1, .reusable = true}}, SHIFT(292),
  [667] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym__Eq, 3, 0, 0),
  [669] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_ExternalID, 3, 0, 0),
  [671] = {.entry = {.count = 1, .reusable = true}}, SHIFT(218),
  [673] = {.entry = {.count = 1, .reusable = true}}, SHIFT(201),
  [675] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_Enumeration, 5, 0, 0),
  [677] = {.entry = {.count = 1, .reusable = true}}, SHIFT(76),
  [679] = {.entry = {.count = 1, .reusable = true}}, SHIFT(163),
  [681] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_Mixed, 7, 0, 0),
  [683] = {.entry = {.count = 1, .reusable = true}}, SHIFT(245),
  [685] = {.entry = {.count = 1, .reusable = true}}, SHIFT(270),
  [687] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_AttValue, 2, 0, 0),
  [689] = {.entry = {.count = 1, .reusable = true}}, SHIFT(37),
  [691] = {.entry = {.count = 1, .reusable = true}}, SHIFT(298),
  [693] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_PublicID, 3, 0, 0),
  [695] = {.entry = {.count = 1, .reusable = true}}, SHIFT(170),
  [697] = {.entry = {.count = 1, .reusable = true}}, SHIFT(273),
  [699] = {.entry = {.count = 1, .reusable = true}}, SHIFT(156),
  [701] = {.entry = {.count = 1, .reusable = true}}, SHIFT(313),
  [703] = {.entry = {.count = 1, .reusable = true}}, SHIFT(289),
  [705] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_AttlistDecl_repeat1, 2, 0, 0), SHIFT(21),
  [708] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_Enumeration, 4, 0, 0),
  [710] = {.entry = {.count = 1, .reusable = true}}, SHIFT(43),
  [712] = {.entry = {.count = 1, .reusable = true}}, SHIFT(287),
  [714] = {.entry = {.count = 1, .reusable = true}}, SHIFT(281),
  [716] = {.entry = {.count = 1, .reusable = true}}, SHIFT(119),
  [718] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_contentspec, 1, 0, 0),
  [720] = {.entry = {.count = 1, .reusable = true}}, SHIFT(285),
  [722] = {.entry = {.count = 1, .reusable = true}}, SHIFT(286),
  [724] = {.entry = {.count = 1, .reusable = true}}, SHIFT(48),
  [726] = {.entry = {.count = 1, .reusable = true}}, SHIFT(267),
  [728] = {.entry = {.count = 1, .reusable = true}}, SHIFT(257),
  [730] = {.entry = {.count = 1, .reusable = true}}, SHIFT(159),
  [732] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_Enumeration, 3, 0, 0),
  [734] = {.entry = {.count = 1, .reusable = true}}, SHIFT(249),
  [736] = {.entry = {.count = 1, .reusable = true}}, SHIFT(252),
  [738] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_DefaultDecl, 1, 0, 0),
  [740] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_EntityValue, 3, 0, 1),
  [742] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_NDataDecl, 4, 0, 0),
  [744] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_AttDef, 6, 0, 0),
  [746] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_SystemLiteral, 3, 0, 0),
  [748] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_PubidLiteral, 3, 0, 0),
  [750] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_ExternalID, 5, 0, 0),
  [752] = {.entry = {.count = 1, .reusable = true}}, SHIFT(278),
  [754] = {.entry = {.count = 1, .reusable = true}}, SHIFT(87),
  [756] = {.entry = {.count = 1, .reusable = true}}, SHIFT(157),
  [758] = {.entry = {.count = 1, .reusable = true}}, SHIFT(280),
  [760] = {.entry = {.count = 1, .reusable = true}}, SHIFT(90),
  [762] = {.entry = {.count = 1, .reusable = true}}, SHIFT(164),
  [764] = {.entry = {.count = 1, .reusable = true}}, SHIFT(283),
  [766] = {.entry = {.count = 1, .reusable = true}}, SHIFT(85),
  [768] = {.entry = {.count = 1, .reusable = true}}, SHIFT(21),
  [770] = {.entry = {.count = 1, .reusable = true}}, SHIFT(176),
  [772] = {.entry = {.count = 1, .reusable = true}}, SHIFT(175),
  [774] = {.entry = {.count = 1, .reusable = true}}, SHIFT(174),
  [776] = {.entry = {.count = 1, .reusable = true}}, SHIFT(89),
  [778] = {.entry = {.count = 1, .reusable = true}}, SHIFT(32),
  [780] = {.entry = {.count = 1, .reusable = true}}, SHIFT(269),
  [782] = {.entry = {.count = 1, .reusable = true}}, SHIFT(83),
  [784] = {.entry = {.count = 1, .reusable = true}}, SHIFT(3),
  [786] = {.entry = {.count = 1, .reusable = true}}, SHIFT(167),
  [788] = {.entry = {.count = 1, .reusable = true}}, SHIFT(59),
  [790] = {.entry = {.count = 1, .reusable = true}}, SHIFT(179),
  [792] = {.entry = {.count = 1, .reusable = true}}, SHIFT(242),
  [794] = {.entry = {.count = 1, .reusable = true}}, SHIFT(73),
  [796] = {.entry = {.count = 1, .reusable = true}}, SHIFT(241),
  [798] = {.entry = {.count = 1, .reusable = true}}, SHIFT(288),
  [800] = {.entry = {.count = 1, .reusable = true}}, SHIFT(151),
  [802] = {.entry = {.count = 1, .reusable = true}}, SHIFT(55),
  [804] = {.entry = {.count = 1, .reusable = true}}, SHIFT(145),
  [806] = {.entry = {.count = 1, .reusable = true}}, SHIFT(136),
  [808] = {.entry = {.count = 1, .reusable = true}}, SHIFT(271),
  [810] = {.entry = {.count = 1, .reusable = true}}, SHIFT(183),
  [812] = {.entry = {.count = 1, .reusable = true}}, SHIFT(133),
  [814] = {.entry = {.count = 1, .reusable = true}}, SHIFT(275),
  [816] = {.entry = {.count = 1, .reusable = true}}, SHIFT(186),
  [818] = {.entry = {.count = 1, .reusable = true}}, SHIFT(310),
  [820] = {.entry = {.count = 1, .reusable = true}}, SHIFT(284),
  [822] = {.entry = {.count = 1, .reusable = true}}, SHIFT(39),
  [824] = {.entry = {.count = 1, .reusable = true}}, SHIFT(264),
  [826] = {.entry = {.count = 1, .reusable = true}}, SHIFT(246),
  [828] = {.entry = {.count = 1, .reusable = true}}, SHIFT(250),
  [830] = {.entry = {.count = 1, .reusable = true}}, SHIFT(155),
  [832] = {.entry = {.count = 1, .reusable = true}}, SHIFT(41),
  [834] = {.entry = {.count = 1, .reusable = true}}, SHIFT(185),
  [836] = {.entry = {.count = 1, .reusable = true}}, SHIFT(50),
  [838] = {.entry = {.count = 1, .reusable = true}}, SHIFT(276),
  [840] = {.entry = {.count = 1, .reusable = true}}, SHIFT(46),
  [842] = {.entry = {.count = 1, .reusable = true}}, SHIFT(154),
  [844] = {.entry = {.count = 1, .reusable = true}}, SHIFT(153),
  [846] = {.entry = {.count = 1, .reusable = true}}, SHIFT(152),
  [848] = {.entry = {.count = 1, .reusable = true}}, SHIFT(150),
  [850] = {.entry = {.count = 1, .reusable = true}}, SHIFT(147),
  [852] = {.entry = {.count = 1, .reusable = true}},  ACCEPT_INPUT(),
  [854] = {.entry = {.count = 1, .reusable = true}}, SHIFT(303),
  [856] = {.entry = {.count = 1, .reusable = true}}, SHIFT(17),
  [858] = {.entry = {.count = 1, .reusable = true}}, SHIFT(82),
  [860] = {.entry = {.count = 1, .reusable = true}}, SHIFT(79),
  [862] = {.entry = {.count = 1, .reusable = true}}, SHIFT(88),
  [864] = {.entry = {.count = 1, .reusable = true}}, SHIFT(102),
  [866] = {.entry = {.count = 1, .reusable = true}}, SHIFT(106),
  [868] = {.entry = {.count = 1, .reusable = true}}, SHIFT(74),
  [870] = {.entry = {.count = 1, .reusable = true}}, SHIFT(103),
  [872] = {.entry = {.count = 1, .reusable = true}}, SHIFT(97),
  [874] = {.entry = {.count = 1, .reusable = true}}, SHIFT(314),
  [876] = {.entry = {.count = 1, .reusable = true}}, SHIFT(315),
  [878] = {.entry = {.count = 1, .reusable = true}}, SHIFT(316),
  [880] = {.entry = {.count = 1, .reusable = true}}, SHIFT(317),
  [882] = {.entry = {.count = 1, .reusable = true}}, SHIFT(318),
  [884] = {.entry = {.count = 1, .reusable = true}}, SHIFT(319),
  [886] = {.entry = {.count = 1, .reusable = true}}, SHIFT(320),
  [888] = {.entry = {.count = 1, .reusable = true}}, SHIFT(321),
};

enum ts_external_scanner_symbol_identifiers {
  ts_external_token_PITarget = 0,
  ts_external_token__pi_content = 1,
  ts_external_token_Comment = 2,
};

static const TSSymbol ts_external_scanner_symbol_map[EXTERNAL_TOKEN_COUNT] = {
  [ts_external_token_PITarget] = sym_PITarget,
  [ts_external_token__pi_content] = sym__pi_content,
  [ts_external_token_Comment] = sym_Comment,
};

static const bool ts_external_scanner_states[5][EXTERNAL_TOKEN_COUNT] = {
  [1] = {
    [ts_external_token_PITarget] = true,
    [ts_external_token__pi_content] = true,
    [ts_external_token_Comment] = true,
  },
  [2] = {
    [ts_external_token_Comment] = true,
  },
  [3] = {
    [ts_external_token_PITarget] = true,
  },
  [4] = {
    [ts_external_token__pi_content] = true,
  },
};

#ifdef __cplusplus
extern "C" {
#endif
void *tree_sitter_dtd_external_scanner_create(void);
void tree_sitter_dtd_external_scanner_destroy(void *);
bool tree_sitter_dtd_external_scanner_scan(void *, TSLexer *, const bool *);
unsigned tree_sitter_dtd_external_scanner_serialize(void *, char *);
void tree_sitter_dtd_external_scanner_deserialize(void *, const char *, unsigned);

#ifdef TREE_SITTER_HIDE_SYMBOLS
#define TS_PUBLIC
#elif defined(_WIN32)
#define TS_PUBLIC __declspec(dllexport)
#else
#define TS_PUBLIC __attribute__((visibility("default")))
#endif

TS_PUBLIC const TSLanguage *tree_sitter_dtd(void) {
  static const TSLanguage language = {
    .version = LANGUAGE_VERSION,
    .symbol_count = SYMBOL_COUNT,
    .alias_count = ALIAS_COUNT,
    .token_count = TOKEN_COUNT,
    .external_token_count = EXTERNAL_TOKEN_COUNT,
    .state_count = STATE_COUNT,
    .large_state_count = LARGE_STATE_COUNT,
    .production_id_count = PRODUCTION_ID_COUNT,
    .field_count = FIELD_COUNT,
    .max_alias_sequence_length = MAX_ALIAS_SEQUENCE_LENGTH,
    .parse_table = &ts_parse_table[0][0],
    .small_parse_table = ts_small_parse_table,
    .small_parse_table_map = ts_small_parse_table_map,
    .parse_actions = ts_parse_actions,
    .symbol_names = ts_symbol_names,
    .field_names = ts_field_names,
    .field_map_slices = ts_field_map_slices,
    .field_map_entries = ts_field_map_entries,
    .symbol_metadata = ts_symbol_metadata,
    .public_symbol_map = ts_symbol_map,
    .alias_map = ts_non_terminal_alias_map,
    .alias_sequences = &ts_alias_sequences[0][0],
    .lex_modes = ts_lex_modes,
    .lex_fn = ts_lex,
    .keyword_lex_fn = ts_lex_keywords,
    .keyword_capture_token = sym_Name,
    .external_scanner = {
      &ts_external_scanner_states[0][0],
      ts_external_scanner_symbol_map,
      tree_sitter_dtd_external_scanner_create,
      tree_sitter_dtd_external_scanner_destroy,
      tree_sitter_dtd_external_scanner_scan,
      tree_sitter_dtd_external_scanner_serialize,
      tree_sitter_dtd_external_scanner_deserialize,
    },
    .primary_state_ids = ts_primary_state_ids,
  };
  return &language;
}
#ifdef __cplusplus
}
#endif
